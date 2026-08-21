package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/gcloud"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	newProjectFlag     string
	newSAFlag          string
	newImpersonateFlag bool
	newNoImpFlag       bool
	newNonInteractFlag bool
)

var newCmd = &cobra.Command{
	Use:   "new <environment> [project_id] [sa_name_or_email]",
	Short: "Create a new isolated Google Cloud environment",
	Long: `Create and configure a new isolated Google Cloud environment tree end-to-end.

This will:
  1. Create an isolated configuration directory (~/.gcloud-envs/<env>)
  2. Authenticate your user account via browserless login
  3. Configure the active GCP project and grant token creator permissions
  4. Optionally set up service account creation, IAM role bindings, and ADC impersonation`,
	Example: `  # Interactive wizard
  gcx new my-env

  # Fully scripted creation
  gcx new prod-env --project my-company-prod --sa deployer-sa --impersonate

  # User-only authentication without service account impersonation
  gcx new dev-env --project my-dev-project --no-impersonate`,
	Args: cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := gcloud.CheckInstalled(); err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)

		envName := args[0]
		if envName == "" {
			return fmt.Errorf("environment name cannot be empty")
		}

		// Resolve Project ID
		projectID := newProjectFlag
		if len(args) >= 2 && args[1] != "" {
			projectID = args[1]
		}
		if projectID == "" {
			if newNonInteractFlag {
				return fmt.Errorf("project ID is required (specify via argument or --project)")
			}
			fmt.Print("Enter Google Cloud Project ID: ")
			input, _ := reader.ReadString('\n')
			projectID = strings.TrimSpace(input)
			if projectID == "" {
				return fmt.Errorf("project ID is required")
			}
		}

		// Resolve Service Account
		sa := newSAFlag
		if len(args) >= 3 && args[2] != "" {
			sa = args[2]
		}
		if sa == "" && !newNoImpFlag && !newNonInteractFlag {
			fmt.Print("Service account name or email (leave blank for user credentials): ")
			input, _ := reader.ReadString('\n')
			sa = strings.TrimSpace(input)
		}

		if sa != "" && !strings.Contains(sa, "@") {
			sa = fmt.Sprintf("%s@%s.iam.gserviceaccount.com", sa, projectID)
		}

		// Determine Impersonation preference
		cliImpersonate := newImpersonateFlag
		if sa != "" && !newImpersonateFlag && !newNoImpFlag && !newNonInteractFlag {
			fmt.Printf("Configure gcloud CLI to use service account impersonation ('%s')? [y/N]: ", sa)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			cliImpersonate = (input == "y" || input == "yes")
		}

		// 1. Setup isolated directory
		envsDir := config.GetEnvsDir()
		envDir := filepath.Join(envsDir, envName)
		if err := os.MkdirAll(envDir, 0700); err != nil {
			return fmt.Errorf("failed to create environment directory %s: %w", envDir, err)
		}

		fmt.Println()
		fmt.Printf("%s\n", ui.TitleStyle.Render("Creating isolated environment: "+envName))
		fmt.Printf("%s : %s\n\n", ui.KeyStyle.Render("Target Dir"), envDir)

		// Step 1: Named configuration
		fmt.Println(ui.StepHeader(1, 4, fmt.Sprintf("Creating/Activating gcloud configuration '%s'", envName)))
		if err := gcloud.SetupConfigDir(envName, envDir); err != nil {
			return fmt.Errorf("failed to initialize gcloud config: %w", err)
		}
		_ = gcloud.UnsetImpersonation(envDir)

		// Step 2: User authentication
		fmt.Println()
		fmt.Println(ui.StepHeader(2, 4, "Authenticating user account (copy generated URL into browser)"))
		if err := gcloud.AuthLogin(envDir); err != nil {
			return fmt.Errorf("user authentication failed: %w", err)
		}

		// Step 3: Project setup & user token creator permissions
		fmt.Println()
		fmt.Println(ui.StepHeader(3, 4, fmt.Sprintf("Setting project defaults (%s)", projectID)))
		if err := gcloud.SetProject(projectID, envDir); err != nil {
			return fmt.Errorf("failed to set project: %w", err)
		}

		userAcct, err := gcloud.GetAccount(envDir)
		if err != nil {
			return fmt.Errorf("could not retrieve authenticated account: %w", err)
		}
		fmt.Printf("   Authenticated as: %s\n", ui.BoldStyle.Render(userAcct))
		fmt.Println("   Granting IAM Token Creator & Service Account User roles to user...")
		if err := gcloud.GrantUserTokenCreatorRoles(projectID, userAcct, envDir); err != nil {
			return err
		}

		// Step 4: SA and ADC setup
		fmt.Println()
		if sa != "" {
			fmt.Println(ui.StepHeader(4, 4, "Setting up Service Account & Impersonated ADC"))
			fmt.Printf("   Checking service account '%s'...\n", sa)
			exists, _ := gcloud.ServiceAccountExists(projectID, sa, envDir)
			if !exists {
				create := true
				if !newNonInteractFlag {
					fmt.Printf("   Service account '%s' does not exist in project '%s'. Create it now? [Y/n]: ", sa, projectID)
					input, _ := reader.ReadString('\n')
					input = strings.TrimSpace(strings.ToLower(input))
					if input == "n" || input == "no" {
						create = false
					}
				}
				if !create {
					return fmt.Errorf("cannot proceed without a valid service account")
				}
				saName := strings.Split(sa, "@")[0]
				fmt.Printf("   Creating service account '%s'...\n", saName)
				if err := gcloud.CreateServiceAccount(projectID, saName, envDir); err != nil {
					return fmt.Errorf("failed to create service account: %w", err)
				}
			}

			// Grant SA project roles with retry backoff
			fmt.Println("   Ensuring Service Account has project permissions...")
			err = gcloud.GrantSARolesWithRetry(projectID, sa, envDir, func(msg string) {
				fmt.Println("   " + msg)
			})
			if err != nil {
				return err
			}

			// CLI impersonation
			if cliImpersonate {
				fmt.Println("   Configuring gcloud CLI service account impersonation...")
				_ = gcloud.SetImpersonation(sa, envDir)
			} else {
				_ = gcloud.UnsetImpersonation(envDir)
			}

			// ADC impersonation
			fmt.Println("   Logging into Application Default Credentials (ADC) with impersonation...")
			if err := gcloud.AuthApplicationDefaultLogin(envDir, sa); err != nil {
				return fmt.Errorf("failed to setup ADC: %w", err)
			}
		} else {
			fmt.Println(ui.StepHeader(4, 4, "Setting up Application Default Credentials (User Account)"))
			_ = gcloud.UnsetImpersonation(envDir)
			if err := gcloud.AuthApplicationDefaultLogin(envDir, ""); err != nil {
				return fmt.Errorf("failed to setup ADC: %w", err)
			}
		}

		fmt.Println()
		fmt.Println(ui.SuccessBox(fmt.Sprintf("Environment '%s' created successfully!", envName)))
		fmt.Println()
		fmt.Println("To switch to this environment in your current shell:")
		fmt.Printf("  eval \"$(gcx env %s)\"\n", envName)
		fmt.Printf("  # or simply 'gcx %s' if shell integration is enabled\n\n", envName)

		return nil
	},
}

func init() {
	newCmd.Flags().StringVarP(&newProjectFlag, "project", "p", "", "Google Cloud Project ID")
	newCmd.Flags().StringVar(&newSAFlag, "sa", "", "Service account name or email")
	newCmd.Flags().BoolVar(&newImpersonateFlag, "impersonate", false, "Configure gcloud CLI to use service account impersonation")
	newCmd.Flags().BoolVar(&newNoImpFlag, "no-impersonate", false, "Do not configure service account impersonation")
	newCmd.Flags().BoolVar(&newNonInteractFlag, "non-interactive", false, "Run non-interactively, failing if required inputs are missing")
}

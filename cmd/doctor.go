package cmd

import (
	"fmt"
	"os"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/gcloud"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Check system health and environment configurations",
	Long:    "Run sanity checks on gcloud installation, environments directory, credentials, and permissions.",
	Example: "  gcx doctor",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.TitleStyle.Render("gcx Doctor Diagnostics"))
		fmt.Println()

		// 1. gcloud CLI check
		fmt.Print("Checking gcloud CLI installation... ")
		if err := gcloud.CheckInstalled(); err != nil {
			fmt.Println(ui.ErrorBox("FAILED"))
			fmt.Printf("   %s\n", err.Error())
		} else {
			out, _ := gcloud.RunCapture(nil, "version")
			firstLine := "installed"
			if out != "" {
				for _, line := range []string{out} {
					if len(line) > 40 {
						line = line[:40] + "..."
					}
					firstLine = line
					break
				}
			}
			fmt.Println(ui.SuccessBox(firstLine))
		}

		// 2. Environments Directory check
		envsDir := config.GetEnvsDir()
		fmt.Printf("Checking environments directory (%s)... ", envsDir)
		dirInfo, err := os.Stat(envsDir)
		if os.IsNotExist(err) {
			fmt.Println(ui.WarningBox("Not created yet"))
			fmt.Println("   Create your first environment with 'gcx new <name>'")
			return nil
		} else if err != nil {
			fmt.Println(ui.ErrorBox("Error accessing directory: " + err.Error()))
			return nil
		} else {
			fmt.Println(ui.SuccessBox("Found"))
			if dirInfo.Mode().Perm() != 0700 {
				fmt.Printf("   %s Directory permissions are %o (recommended: 700)\n", ui.WarningStyle.Render("▲"), dirInfo.Mode().Perm())
			}
		}

		// 3. Host Context Aware Access posture
		host := config.DetectHostContextAware()
		fmt.Print("Checking Context Aware Access (client certificate) on this host... ")
		if host == nil {
			fmt.Println(ui.SuccessBox("not required"))
		} else {
			fmt.Println(ui.SuccessBox("enabled via " + host.Source))
			for _, kv := range host.Settings.Properties() {
				fmt.Printf("   %s = %s\n", kv[0], kv[1])
			}
			if host.DroppedCertPath != "" {
				fmt.Printf("   %s Host advertises certificate config %s but it does not exist; it will not be copied into trees.\n", ui.WarningStyle.Render("▲"), host.DroppedCertPath)
			}
			if os.Getenv(config.EnvContextAwareUseClientCertificate) == "" {
				fmt.Printf("   %s %s is not set in this shell; trees must carry the setting themselves.\n", ui.WarningStyle.Render("▲"), config.EnvContextAwareUseClientCertificate)
			}
		}

		// 4. Scan Environments
		envs, err := config.ListEnvironments()
		if err != nil {
			fmt.Println(ui.ErrorBox("Failed to list environments: " + err.Error()))
			return nil
		}

		fmt.Printf("\nInspecting %d environment(s):\n\n", len(envs))
		fixesNeeded := 0
		fixesApplied := 0
		for _, e := range envs {
			fmt.Printf("  • %s\n", ui.BoldStyle.Render(e.Name))
			if e.Project == "" {
				fmt.Printf("    %s No project configured in active config '%s'\n", ui.WarningStyle.Render("▲"), e.ActiveConfig)
			} else {
				fmt.Printf("    %s Project: %s\n", ui.SuccessStyle.Render("✔"), e.Project)
			}

			if e.Account == "" {
				fmt.Printf("    %s No account logged in\n", ui.WarningStyle.Render("▲"))
			} else {
				fmt.Printf("    %s Account: %s\n", ui.SuccessStyle.Render("✔"), e.Account)
			}

			if e.ADCExists {
				if e.ADC != nil && e.ADC.ImpersonatedEmail != "" {
					fmt.Printf("    %s ADC: Impersonating %s\n", ui.SuccessStyle.Render("✔"), e.ADC.ImpersonatedEmail)
				} else if e.ADC != nil && e.ADC.Type != "" {
					fmt.Printf("    %s ADC: %s\n", ui.SuccessStyle.Render("✔"), e.ADC.Type)
				} else {
					fmt.Printf("    %s ADC file present\n", ui.SuccessStyle.Render("✔"))
				}
			} else {
				fmt.Printf("    %s No ADC file (application_default_credentials.json missing)\n", ui.WarningStyle.Render("▲"))
			}

			// Context Aware Access drift
			issues := config.DiagnoseTreeContextAware(host, e.ContextAware, fileExistsForDoctor)
			if len(issues) == 0 {
				if host != nil {
					fmt.Printf("    %s CAA: client certificate configured\n", ui.SuccessStyle.Render("✔"))
				}
			} else {
				fixesNeeded++
				for _, issue := range issues {
					marker := ui.WarningStyle.Render("▲")
					if issue.Fatal {
						marker = ui.ErrorStyle.Render("✖")
					}
					fmt.Printf("    %s CAA: %s\n", marker, issue.Message)
				}
				if doctorFixFlag && host != nil {
					if err := gcloud.SetProperties(e.Path, host.Settings.Properties()); err != nil {
						fmt.Printf("    %s CAA fix failed: %v\n", ui.ErrorStyle.Render("✖"), err)
					} else {
						fixesApplied++
						fmt.Printf("    %s CAA: applied host settings to tree\n", ui.SuccessStyle.Render("✔"))
					}
				}
			}
			fmt.Println()
		}

		if fixesNeeded > 0 && !doctorFixFlag {
			fmt.Printf("%s %d environment(s) have Context Aware Access drift. Run 'gcx doctor --fix' to repair.\n\n", ui.WarningStyle.Render("▲"), fixesNeeded)
		}
		if doctorFixFlag {
			fmt.Printf("Applied fixes to %d of %d environment(s) needing them.\n\n", fixesApplied, fixesNeeded)
		}

		fmt.Println(ui.SuccessBox("Diagnostics completed."))
		return nil
	},
}

var doctorFixFlag bool

func fileExistsForDoctor(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorFixFlag, "fix", false, "Repair environments whose Context Aware Access settings drift from the host")
}

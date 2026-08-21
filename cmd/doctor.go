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
	Use:   "doctor",
	Short: "Check system health and environment configurations",
	Long:  "Run sanity checks on gcloud installation, environments directory, credentials, and permissions.",
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

		// 3. Scan Environments
		envs, err := config.ListEnvironments()
		if err != nil {
			fmt.Println(ui.ErrorBox("Failed to list environments: " + err.Error()))
			return nil
		}

		fmt.Printf("\nInspecting %d environment(s):\n\n", len(envs))
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
			fmt.Println()
		}

		fmt.Println(ui.SuccessBox("Diagnostics completed."))
		return nil
	},
}

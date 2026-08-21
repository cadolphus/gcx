package cmd

import (
	"fmt"
	"strings"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var whoCmd = &cobra.Command{
	Use:     "who",
	Aliases: []string{"current", "status"},
	Short:   "Show details of the currently loaded environment and authentication",
	Long:    "Show details of the currently active gcloud environment, configuration, project, and ADC credentials.",
	Example: "  gcx who\n  gcx status",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := config.GetCurrentState()
		if err != nil {
			return err
		}

		var sb strings.Builder

		// Environment line
		if state.CloudSDKConfig == "" {
			sb.WriteString(ui.PrintMutedKV("env", "<none - using global ~/.config/gcloud>"))
			sb.WriteString("\n")
			sb.WriteString(ui.PrintKV("tree", config.GetGlobalGCloudDir()))
			sb.WriteString("\n")
		} else {
			if state.ActiveEnvName != "" {
				if state.DerivedFromPath {
					sb.WriteString(ui.PrintKV("env", fmt.Sprintf("%s  (derived from path - run 'gcx %s' to set env vars)", state.ActiveEnvName, state.ActiveEnvName)))
				} else {
					sb.WriteString(ui.PrintKV("env", state.ActiveEnvName))
				}
			} else if state.IsCustomConfig {
				sb.WriteString(ui.PrintMutedKV("env", "<custom CLOUDSDK_CONFIG, not managed by gcx>"))
			} else {
				sb.WriteString(ui.PrintMutedKV("env", "<unknown>"))
			}
			sb.WriteString("\n")
			sb.WriteString(ui.PrintKV("tree", state.CloudSDKConfig))
			sb.WriteString("\n")
		}

		// Config
		configVal := state.ActiveConfig
		if configVal == "" {
			configVal = "<default>"
		}
		sb.WriteString(ui.PrintKV("config", configVal))
		sb.WriteString("\n")

		// Project
		projVal := state.Project
		if projVal == "" {
			projVal = "<unset>"
		}
		sb.WriteString(ui.PrintKV("project", projVal))
		sb.WriteString("\n")

		// Quota project
		quotaVal := state.QuotaProject
		if quotaVal == "" {
			quotaVal = "<unset - run 'gcx <env>' to set it>"
		}
		sb.WriteString(ui.PrintKV("quota", quotaVal))
		sb.WriteString("\n")

		// Account
		acctVal := state.Account
		if acctVal == "" {
			acctVal = "<unset>"
		}
		sb.WriteString(ui.PrintKV("gcloud as", acctVal))
		sb.WriteString("\n")

		// Impersonate
		impVal := state.ImpersonateSA
		if impVal == "" {
			sb.WriteString(ui.PrintMutedKV("impersona", "<unset - gcloud CLI will act as your user>"))
		} else {
			sb.WriteString(ui.PrintKV("impersona", impVal))
		}
		sb.WriteString("\n")

		// ADC
		adcVal := "<no ADC file>"
		if state.ADC != nil {
			if state.ADC.ImpersonatedEmail != "" {
				adcVal = state.ADC.ImpersonatedEmail
			} else if state.ADC.Type == "authorized_user" {
				adcVal = "authorized_user (not impersonating)"
			} else if state.ADC.Type != "" {
				adcVal = fmt.Sprintf("%s (not impersonating)", state.ADC.Type)
			}
		}
		sb.WriteString(ui.PrintKV("ADC", adcVal))
		sb.WriteString("\n")

		fmt.Print(sb.String())
		return nil
	},
}

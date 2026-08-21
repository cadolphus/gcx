package ui

import (
	"fmt"
	"strings"

	"github.com/cadolphus/gcx/internal/config"
)

// RenderEnvironmentsTable renders a formatted table of environments.
func RenderEnvironmentsTable(envs []*config.Environment) string {
	if len(envs) == 0 {
		return MutedStyle.Render("No environments found in " + config.GetEnvsDir() + "\nCreate one with 'gcx new <name> <project_id>'")
	}

	// Calculate column widths
	envWidth := 14
	projWidth := 24
	acctWidth := 28
	saWidth := 28

	for _, e := range envs {
		if len(e.Name) > envWidth {
			envWidth = len(e.Name)
		}
		if len(e.Project) > projWidth {
			projWidth = len(e.Project)
		}
		if len(e.Account) > acctWidth {
			acctWidth = len(e.Account)
		}
		if len(e.ImpersonateSA) > saWidth {
			saWidth = len(e.ImpersonateSA)
		}
	}

	// Cap max column widths so it stays terminal-friendly
	if envWidth > 20 {
		envWidth = 20
	}
	if projWidth > 32 {
		projWidth = 32
	}
	if acctWidth > 34 {
		acctWidth = 34
	}
	if saWidth > 34 {
		saWidth = 34
	}

	var sb strings.Builder

	// Header
	header := fmt.Sprintf(
		"  %-*s  %-*s  %-*s  %-*s  %s",
		envWidth, "ENV",
		projWidth, "PROJECT",
		acctWidth, "ACCOUNT",
		saWidth, "IMPERSONATION",
		"ADC",
	)
	sb.WriteString(HeaderColStyle.Render(header))
	sb.WriteString("\n")

	divider := fmt.Sprintf(
		"  %s  %s  %s  %s  %s",
		strings.Repeat("─", envWidth),
		strings.Repeat("─", projWidth),
		strings.Repeat("─", acctWidth),
		strings.Repeat("─", saWidth),
		strings.Repeat("─", 16),
	)
	sb.WriteString(MutedStyle.Render(divider))
	sb.WriteString("\n")

	for _, e := range envs {
		prefix := "  "
		envCol := e.Name
		if e.IsActive {
			prefix = SuccessStyle.Render("* ")
			envCol = SuccessStyle.Render(PadRight(Truncate(e.Name, envWidth), envWidth))
		} else {
			envCol = PadRight(Truncate(e.Name, envWidth), envWidth)
		}

		projVal := e.Project
		if projVal == "" {
			projVal = "<unset>"
		}
		projCol := PadRight(Truncate(projVal, projWidth), projWidth)

		acctVal := e.Account
		if acctVal == "" {
			acctVal = "<unset>"
		}
		acctCol := PadRight(Truncate(acctVal, acctWidth), acctWidth)

		saVal := e.ImpersonateSA
		if saVal == "" {
			saVal = "-"
		}
		saCol := PadRight(Truncate(saVal, saWidth), saWidth)

		adcVal := "-"
		if e.ADCExists {
			if e.ADC != nil {
				if e.ADC.ImpersonatedEmail != "" {
					adcVal = "SA (" + Truncate(e.ADC.ImpersonatedEmail, 14) + ")"
				} else if e.ADC.Type == "authorized_user" {
					adcVal = "User"
				} else if e.ADC.Type != "" {
					adcVal = e.ADC.Type
				} else {
					adcVal = "Present"
				}
			} else {
				adcVal = "Present"
			}
		} else {
			adcVal = WarningStyle.Render("Missing")
		}

		activeIndicator := ""
		if e.IsActive {
			activeIndicator = ActiveBadgeStyle.Render("  <-- active")
		}

		row := fmt.Sprintf("%s%s  %s  %s  %s  %s%s\n",
			prefix,
			envCol,
			projCol,
			acctCol,
			saCol,
			adcVal,
			activeIndicator,
		)
		sb.WriteString(row)
	}

	return sb.String()
}

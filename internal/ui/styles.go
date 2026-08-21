package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Brand / Header styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#4285F4")) // Google Blue

	BoldStyle = lipgloss.NewStyle().Bold(true)

	// Status styles
	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#34A853")) // Google Green

	ActiveBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#34A853"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBC05")) // Google Yellow

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EA4335")) // Google Red

	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	KeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#4285F4")).
			Width(16)

	ValueStyle = lipgloss.NewStyle()

	HeaderColStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#E0E0E0")).
			PaddingRight(2)
)

// PrintKV formats a key-value row with alignment.
func PrintKV(key, value string) string {
	return fmt.Sprintf("%s : %s", KeyStyle.Render(key), ValueStyle.Render(value))
}

// PrintMutedKV formats a key-value row with muted value.
func PrintMutedKV(key, value string) string {
	return fmt.Sprintf("%s : %s", KeyStyle.Render(key), MutedStyle.Render(value))
}

// StepHeader formats a wizard step.
func StepHeader(step int, total int, title string) string {
	badge := TitleStyle.Render(fmt.Sprintf("==> [%d/%d]", step, total))
	return fmt.Sprintf("%s %s", badge, BoldStyle.Render(title))
}

// SuccessBox formats a success message.
func SuccessBox(msg string) string {
	return SuccessStyle.Render("✔ " + msg)
}

// WarningBox formats a warning message.
func WarningBox(msg string) string {
	return WarningStyle.Render("▲ " + msg)
}

// ErrorBox formats an error message.
func ErrorBox(msg string) string {
	return ErrorStyle.Render("✖ " + msg)
}

// Truncate ensures text doesn't overflow a target width
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// PadRight pads string with spaces to min width
func PadRight(s string, minWidth int) string {
	if len(s) >= minWidth {
		return s
	}
	return s + strings.Repeat(" ", minWidth-len(s))
}

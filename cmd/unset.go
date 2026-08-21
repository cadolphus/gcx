package cmd

import (
	"fmt"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var unsetCmd = &cobra.Command{
	Use:     "unset",
	Aliases: []string{"clear", "reset"},
	Short:   "Unset environment variables and return to default global gcloud configuration",
	Long:    "Clear all isolated gcloud environment variables and restore default global gcloud behavior (~/.config/gcloud).",
	Example: "  gcx unset\n  gcx clear\n  gcx reset",
	RunE: func(cmd *cobra.Command, args []string) error {
		evalMode, _ := cmd.Flags().GetBool("eval")
		if evalMode {
			fmt.Print(config.GenerateShellUnset())
			return nil
		}

		fmt.Printf("%s Restoring global gcloud default (~/.config/gcloud).\n\n", ui.SuccessBox("Default config:"))
		fmt.Println("To apply this to your current shell session, run:")
		fmt.Println("  eval \"$(gcx unset --eval)\"")
		return nil
	},
}

func init() {
	unsetCmd.Flags().Bool("eval", false, "Output shell unset statements directly")
}

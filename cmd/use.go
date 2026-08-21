package cmd

import (
	"fmt"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:     "use <environment>",
	Aliases: []string{"switch"},
	Short:   "Switch the active shell environment",
	Long:    "Switch the active shell environment to the specified isolated gcloud configuration tree.",
	Args:    cobra.ExactArgs(1),
	Example: "  gcx use code\n  gcx use omni",
	RunE: func(cmd *cobra.Command, args []string) error {
		envName := args[0]
		env, err := config.GetEnvironment(envName)
		if err != nil {
			return err
		}

		fmt.Printf("%s Environment '%s' selected.\n\n", ui.SuccessBox("Ready:"), env.Name)
		fmt.Println("To activate in your current shell session, run:")
		fmt.Printf("  eval \"$(gcx env %s)\"\n\n", env.Name)
		fmt.Println("Tip: Add shell integration to your .zshrc or .bashrc to switch directly with 'gcx <env>':")
		fmt.Println("  eval \"$(gcx init zsh)\"   # in ~/.zshrc")
		fmt.Println("  eval \"$(gcx init bash)\"  # in ~/.bashrc")
		return nil
	},
}

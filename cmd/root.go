package cmd

import (
	"fmt"
	"os"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gcx [command|environment]",
	Short: "gcx - High-performance Google Cloud environment & configuration manager",
	Long: ui.TitleStyle.Render("gcx") + ` - High-performance Google Cloud environment & configuration manager.

Seamlessly manage isolated gcloud configuration trees, Application Default
Credentials (ADC), and service account impersonation for fast project switching
and direnv integration.`,
	Example: `  # List all available GCP environments
  gcx ls

  # Switch active shell environment (requires shell integration: eval "$(gcx init zsh)")
  gcx use code
  gcx code

  # View active environment and ADC details
  gcx who

  # Create a new isolated environment end-to-end
  gcx new my-env --project my-project-id --sa deployer

  # Return to default global gcloud configuration
  gcx unset

  # Use with direnv inside any project's .envrc
  use gcx my-env`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		firstArg := args[0]
		if firstArg == "help" {
			return cmd.Help()
		}

		// Check if argument corresponds to an existing environment
		env, err := config.GetEnvironment(firstArg)
		if err == nil && env != nil {
			// If run directly without shell wrapper, explain how to switch
			fmt.Printf("%s Environment '%s' found at %s\n\n", ui.SuccessBox("Environment found:"), env.Name, env.Path)
			fmt.Println("To switch active shell environment, run:")
			fmt.Printf("  eval \"$(gcx env %s)\"\n\n", env.Name)
			fmt.Println("Tip: Add shell integration to your .zshrc or .bashrc to switch directly with 'gcx <env>':")
			fmt.Println("  eval \"$(gcx init zsh)\"   # in ~/.zshrc")
			fmt.Println("  eval \"$(gcx init bash)\"  # in ~/.bashrc")
			return nil
		}

		return fmt.Errorf("unknown command or environment '%s'. Run 'gcx help' for usage", firstArg)
	},
}

// Execute runs the root CLI command.
func Execute() {
	// Intercept subcommands with trailing 'help' argument, e.g. 'gcx new help' -> 'gcx help new'
	preprocessHelpArgs(os.Args)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.ErrorBox(err.Error()))
		os.Exit(1)
	}
}

// preprocessHelpArgs rewrites "gcx <cmd> help" into "gcx <cmd> --help"
func preprocessHelpArgs(args []string) {
	if len(args) >= 3 && args[len(args)-1] == "help" {
		args[len(args)-1] = "--help"
	}
}

func init() {
	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(whoCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(unsetCmd)
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(direnvCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(initCmd)
}

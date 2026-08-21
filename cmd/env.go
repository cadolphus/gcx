package cmd

import (
	"fmt"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env <environment>",
	Short: "Output shell export commands for a specific environment",
	Long:  "Generate and output shell export and unset statements for evaluation by a shell wrapper or script.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		envName := args[0]
		exports, err := config.GenerateShellExports(envName)
		if err != nil {
			return err
		}

		fmt.Print(exports)
		return nil
	},
}

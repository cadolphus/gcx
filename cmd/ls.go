package cmd

import (
	"fmt"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all available Google Cloud environments",
	Long:    "List all isolated Google Cloud environments discovered in the environments directory.",
	Example: "  gcx ls\n  gcx list",
	RunE: func(cmd *cobra.Command, args []string) error {
		envs, err := config.ListEnvironments()
		if err != nil {
			return err
		}

		fmt.Print(ui.RenderEnvironmentsTable(envs))
		return nil
	},
}

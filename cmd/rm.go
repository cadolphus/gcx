package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var rmForceFlag bool

var rmCmd = &cobra.Command{
	Use:     "rm <environment>",
	Aliases: []string{"delete"},
	Short:   "Delete an isolated Google Cloud environment",
	Long:    "Delete the specified environment directory and its associated credentials.",
	Args:    cobra.ExactArgs(1),
	Example: "  gcx rm my-env\n  gcx rm my-env -f",
	RunE: func(cmd *cobra.Command, args []string) error {
		envName := args[0]
		env, err := config.GetEnvironment(envName)
		if err != nil {
			return err
		}

		if !rmForceFlag {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Are you sure you want to delete environment '%s' (%s)? [y/N]: ", env.Name, env.Path)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "y" && input != "yes" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		if err := os.RemoveAll(env.Path); err != nil {
			return fmt.Errorf("failed to remove directory %s: %w", env.Path, err)
		}

		fmt.Println(ui.SuccessBox(fmt.Sprintf("Environment '%s' deleted successfully.", envName)))
		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForceFlag, "force", "f", false, "Force delete without confirmation prompt")
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cadolphus/gcx/internal/config"
	"github.com/spf13/cobra"
)

var direnvCmd = &cobra.Command{
	Use:   "direnv [environment|hook]",
	Short: "Direnv integration helper for .envrc files",
	Long: `Generate instant environment exports for direnv (.envrc).

To configure direnv globally, add this helper to ~/.config/direnv/direnvrc:

  use_gcx() {
    eval "$(gcx direnv "$1")"
  }

Then in any project directory's .envrc file:

  use gcx <environment_name>`,
	Example: `  # Inside ~/.config/direnv/direnvrc
  use_gcx() { eval "$(gcx direnv "$1")"; }

  # Inside any repository .envrc
  use gcx code`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := args[0]
		if arg == "hook" {
			fmt.Println(`use_gcx() {
  eval "$(gcx direnv "$1")"
}`)
			return nil
		}

		env, err := config.GetEnvironment(arg)
		if err != nil {
			return err
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("export CLOUDSDK_CONFIG=%q\n", env.Path))
		sb.WriteString(fmt.Sprintf("export GCP_ENV=%q\n", env.Name))

		adcPath := filepath.Join(env.Path, "application_default_credentials.json")
		if _, err := os.Stat(adcPath); err == nil {
			sb.WriteString(fmt.Sprintf("export GOOGLE_APPLICATION_CREDENTIALS=%q\n", adcPath))
		}

		if env.Project != "" {
			sb.WriteString(fmt.Sprintf("export GOOGLE_PROJECT=%q\n", env.Project))
			sb.WriteString(fmt.Sprintf("export CLOUDSDK_CORE_PROJECT=%q\n", env.Project))
			sb.WriteString(fmt.Sprintf("export GOOGLE_CLOUD_QUOTA_PROJECT=%q\n", env.Project))
		}

		fmt.Print(sb.String())
		return nil
	},
}

package cmd

import (
	"fmt"
	"runtime"

	"github.com/cadolphus/gcx/internal/ui"
	"github.com/spf13/cobra"
)

var (
	Version   = "1.0.0"
	GitCommit = "HEAD"
	BuildDate = "2026-08-21"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of gcx",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s v%s (%s/%s)\n", ui.TitleStyle.Render("gcx"), Version, runtime.GOOS, runtime.GOARCH)
		fmt.Printf("commit: %s, built: %s\n", GitCommit, BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

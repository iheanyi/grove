package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time
	Commit = "unknown"
	// Date is set at build time
	Date = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("grove version %s\n", Version)
		fmt.Printf("  commit: %s\n", Commit)
		fmt.Printf("  built:  %s\n", Date)
		fmt.Printf("  go:     %s\n", runtime.Version())
		fmt.Printf("  os:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

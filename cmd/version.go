package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These values are replaced by release builds through -ldflags. Keeping
// meaningful development defaults makes source builds easy to identify.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show AEM build version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("aem %s (commit %s, built %s)\n", Version, Commit, BuildDate)
		},
	}
}

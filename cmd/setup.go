package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Run setup command",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("setup initiated")
	},
}

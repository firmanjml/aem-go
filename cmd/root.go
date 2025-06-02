package cmd

import (
	"fmt"
	"os"
	"runtime"

	"aem/assets"
	"aem/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "aem",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if config.IsDebug {
			fmt.Println("[DEBUG] AEM verbose mode is enabled.")
			fmt.Println("[DEBUG] Operating System:", runtime.GOOS)
			fmt.Println("[DEBUG] Architecture:", runtime.GOARCH)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(assets.Banner)
		cmd.Help()
	},
}

func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&config.IsDebug, "debug", "d", false, "enable verbose mode")
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(javaCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

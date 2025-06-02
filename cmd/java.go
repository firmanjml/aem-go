package cmd

import (
	"aem/internal/java"
	"aem/internal/node"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var javaCmd = &cobra.Command{
	Use:   "java",
	Short: "Manage JDK versions",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	useCmd := &cobra.Command{
		Use:   "use [major version]",
		Short: "Use a specific major version of JDK",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := args[0]

			versionPath := filepath.Join("sys_installed", "java", "v"+input)

			if _, err := os.Stat(versionPath); err == nil {
				fmt.Println("[INFO] Version", input, "already exists. Skipping download.")
			} else {
				extractedPath, err := java.DownloadAndExtractJDK(input)
				if err != nil {
					log.Fatalf("Failed: %v", err)
				}
				versionPath = extractedPath
			}

			link := os.Getenv("AEM_JAVA_SYMLINK")
			target, err := filepath.Abs(versionPath)
			if err != nil {
				log.Fatalf("Failed to get absolute path: %v", err)
			}

			if err := node.CreateDirSymlink(link, target); err != nil {
				log.Fatalf("Symlink creation failed: %v", err)
			}

			fmt.Println("Symlink created:", link, "->", target)
		},
	}

	javaCmd.AddCommand(useCmd)
}

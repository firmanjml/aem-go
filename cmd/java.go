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
	installCmd := &cobra.Command{
		Use:   "install [major version]",
		Short: "Use a specific major version of JDK",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := args[0]

			versionPath := filepath.Join("sys_installed", "java", "v"+input)

			if _, err := os.Stat(versionPath); err == nil {
				fmt.Println("[INFO] Version", input, "already exists. Skipping download.")
			}

			_, err := java.DownloadAndExtractJDK(input)
			if err != nil {
				log.Fatalf("Failed: %v", err)
			}

			fmt.Println("[DEBUG] Version", input, "installed.")
		},
	}

	useCmd := &cobra.Command{
		Use:   "use [major version]",
		Short: "Use a specific major version of JDK",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := args[0]
			versionPath := filepath.Join("sys_installed", "java", input)

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

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed java",
		Run: func(cmd *cobra.Command, args []string) {
			versionPath := filepath.Join("sys_installed", "java")

			if err := os.MkdirAll(versionPath, os.ModePerm); err != nil {
				log.Fatal(err)
			}

			entries, err := os.ReadDir(versionPath)
			if err != nil {
				log.Fatal(err)
			}

			if len(entries) == 0 {
				fmt.Println("No jdk installed")
			}

			for _, e := range entries {
				fmt.Println(e.Name())
			}
		},
	}

	javaCmd.AddCommand(installCmd)
	javaCmd.AddCommand(useCmd)
	javaCmd.AddCommand(listCmd)
}

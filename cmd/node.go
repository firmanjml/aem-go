package cmd

import (
	"aem/internal/node"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage Node.js versions",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	useCmd := &cobra.Command{
		Use:   "use [major version]",
		Short: "Use a specific major version of Node.js",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := args[0]
			if !strings.HasPrefix(input, "v") {
				input = "v" + input
			}

			versionPath := filepath.Join("sys_installed", "node", input)

			link := os.Getenv("AEM_NODE_SYMLINK")
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

	installCmd := &cobra.Command{
		Use:   "install [major version]",
		Short: "Use a specific major version of Node.js",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			input := args[0]
			if !strings.HasPrefix(input, "v") {
				input = "v" + input
			}

			versions := node.GetVersions()

			var matched []string
			for _, v := range versions {
				if strings.HasPrefix(v, input) {
					matched = append(matched, v)
				}
			}

			if len(matched) == 0 {
				fmt.Printf("No versions found for Node.js major version %s\n", input)
				return
			}

			latest := matched[len(matched)-1]

			downloadUrl := node.DownloadURL(latest)
			_, err := node.DownloadAndExtractZip(downloadUrl, latest)
			if err != nil {
				log.Fatalf("Failed: %v", err)
			}
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed node.js",
		Run: func(cmd *cobra.Command, args []string) {
			versionPath := filepath.Join("sys_installed", "node")

			if err := os.MkdirAll(versionPath, os.ModePerm); err != nil {
				log.Fatal(err)
			}

			entries, err := os.ReadDir(versionPath)
			if err != nil {
				log.Fatal(err)
			}

			if len(entries) == 0 {
				fmt.Println("No node.js installed")
			}

			for _, e := range entries {
				fmt.Println(e.Name())
			}
		},
	}

	nodeCmd.AddCommand(installCmd)
	nodeCmd.AddCommand(useCmd)
	nodeCmd.AddCommand(listCmd)
}

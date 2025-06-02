package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"golang.org/x/net/html"
)

const banner = `
   $$$$$$\  $$$$$$$$\ $$\      $$\ 
  $$  __$$\ $$  _____|$$$\    $$$ |
  $$ /  $$ |$$ |      $$$$\  $$$$ |
  $$$$$$$$ |$$$$$\    $$\$$\$$ $$ |
  $$  __$$ |$$  __|   $$ \$$$  $$ |
  $$ |  $$ |$$ |      $$ |\$  /$$ |
  $$ |  $$ |$$$$$$$$\ $$ | \_/ $$ |
  \__|  \__|\________|\__|     \__|
    
Less time to setup, more time to code

`

func checkDebug() bool {
	debugStr := os.Getenv("AEM_DEBUG")

	if debugStr == "" {
		debugStr = "false"
	}

	isDebug, err := strconv.ParseBool(debugStr)
	if err != nil {
		isDebug = false
	}

	return isDebug
}

func getNodeVersions() {
	nodeDistribution := "https://nodejs.org/dist/"

	resp, err := http.Get(nodeDistribution)
	if err != nil {
		log.Fatalf("Error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Error parsing HTML: %v", err)
	}

	var versions []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" &&
					strings.HasPrefix(attr.Val, "v") &&
					strings.HasSuffix(attr.Val, "/") &&
					semver.IsValid(attr.Val[:len(attr.Val)-1]) {
					version := strings.TrimSuffix(attr.Val, "/")
					versions = append(versions, version)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})

	for _, version := range versions {
		fmt.Println(version)
	}
}

func main() {
	// Check if AEM is set to debug mode
	isDebug := checkDebug()

	var rootCmd = &cobra.Command{
		Use: "aem",
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if isDebug {
				fmt.Println()
				fmt.Println("[DEBUG] AEM verbose mode is enabled.")
				fmt.Println("[DEBUG] Operating System:", runtime.GOOS)
				fmt.Println("[DEBUG] Architecture:", runtime.GOARCH)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(banner)
			cmd.Help()
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&isDebug, "debug", "d", false, "enable verbose mode")

	var setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Run setup command",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("setup initiated")
		},
	}

	var nodeCmd = &cobra.Command{
		Use:   "node",
		Short: "Run node command",
		Run: func(cmd *cobra.Command, args []string) {
			getNodeVersions()
		},
	}

	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(nodeCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

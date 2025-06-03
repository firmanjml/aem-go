package cmd

import (
	"aem/assets"
	"aem/internal/java"
	"aem/internal/node"
	"aem/pkg/logger"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	debug      bool
	log        *logger.Logger
	installDir = "sys_installed"
)

var rootCmd = &cobra.Command{
	Use: "aem",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log = logger.New(debug)
		if debug {
			log.Debug("AEM verbose mode enabled")
			log.Debug("Operating System: %s", runtime.GOOS)
			log.Debug("Architecture: %s", runtime.GOARCH)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(assets.Banner)
		cmd.Help()
	},
}

func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable verbose mode")

	// rootCmd.AddCommand(newSetupCmd())
	rootCmd.AddCommand(newNodeCmd())
	rootCmd.AddCommand(newJavaCmd())

	if err := rootCmd.Execute(); err != nil {
		if log != nil {
			log.Fatal("Command execution failed: %v", err)
		} else {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func newJavaCmd() *cobra.Command {
	javaCmd := &cobra.Command{
		Use:   "java",
		Short: "Manage JDK versions",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	installCmd := &cobra.Command{
		Use:   "install [major version]",
		Short: "Install a specific major version of JDK",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			javaService := java.NewService(log, installDir)
			return javaService.Install(args[0])
		},
	}

	useCmd := &cobra.Command{
		Use:     "use [version]",
		Short:   "Use a specific version of JDK",
		Aliases: []string{"set"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			javaService := java.NewService(log, installDir)
			symlinkPath := os.Getenv("AEM_JAVA_SYMLINK")
			return javaService.Use(args[0], symlinkPath)
		},
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List installed JDK versions",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			javaService := java.NewService(log, installDir)
			versions, err := javaService.List()
			if err != nil {
				return err
			}

			if len(versions) == 0 {
				fmt.Println("No JDK versions installed")
				return nil
			}

			for _, version := range versions {
				fmt.Println(version)
			}
			return nil
		},
	}

	currentCmd := &cobra.Command{
		Use:   "current",
		Short: "List the current JDK versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			javaService := java.NewService(log, installDir)
			version, err := javaService.GetCurrentJDKVersion()
			if err != nil {
				return err
			}
			fmt.Println("*  " + version)
			return nil
		},
	}

	javaCmd.AddCommand(installCmd)
	javaCmd.AddCommand(useCmd)
	javaCmd.AddCommand(listCmd)
	javaCmd.AddCommand(currentCmd)

	return javaCmd
}

func newNodeCmd() *cobra.Command {

	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "Manage Node.js versions",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	installCmd := &cobra.Command{
		Use:   "install [major version]",
		Short: "Install a specific major version of Node.js",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeService := node.NewService(log, installDir)
			return nodeService.Install(args[0])
		},
	}

	useCmd := &cobra.Command{
		Use:     "use [version]",
		Short:   "Use a specific version of Node.js",
		Aliases: []string{"set"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeService := node.NewService(log, installDir)
			symlinkPath := os.Getenv("AEM_NODE_SYMLINK")
			return nodeService.Use(args[0], symlinkPath)
		},
	}

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List installed NodeJS versions",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeService := node.NewService(log, installDir)
			versions, err := nodeService.List()
			if err != nil {
				return err
			}

			if len(versions) == 0 {
				fmt.Println("No NodeJS versions installed")
				return nil
			}

			for _, version := range versions {
				fmt.Println(version)
			}
			return nil
		},
	}

	currentCmd := &cobra.Command{
		Use:   "current",
		Short: "List the current NodeJS versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			nodeService := node.NewService(log, installDir)
			version, err := nodeService.GetCurrentNodeVersion()
			if err != nil {
				return err
			}
			fmt.Println("*  " + version)
			return nil
		},
	}

	nodeCmd.AddCommand(installCmd)
	nodeCmd.AddCommand(useCmd)
	nodeCmd.AddCommand(listCmd)
	nodeCmd.AddCommand(currentCmd)

	return nodeCmd

}

package cmd

import (
	"aem/assets"
	"aem/extensions/java"
	"aem/extensions/node"
	"aem/internal/manager"
	"aem/pkg/filesystem"
	"aem/pkg/logger"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	debug bool
	log   *logger.Logger
	fs    *filesystem.FileSystem
)

var extensionMgr = manager.NewExtensionManager()

var rootCmd = &cobra.Command{
	Use: "aem",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log = logger.New(debug)
		fs = filesystem.New(log)

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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list all module versions",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		module := args[0]
		var version *string

		if len(args) == 2 {
			version = &args[1]
		}

		extension, exists := extensionMgr.GetExtension(module)
		if !exists {
			return fmt.Errorf("%s module does not exist", module)
		}

		versions, err := extension.ListVersions(version)
		if err != nil {
			return err
		}

		if len(versions) == 0 {
			fmt.Println("This version is not found.")
		}

		if len(versions) > 10 {
			for _, v := range versions[:10] {
				fmt.Println(v)
			}
		} else {
			for _, v := range versions {
				fmt.Println(v)
			}
		}

		return nil
	},
}

func Execute() {
	nodeExtension := node.NewNodeExtension()
	javaExtension := java.NewJavaExtension()
	extensionMgr.RegisterExtension("node", nodeExtension)
	extensionMgr.RegisterExtension("java", javaExtension)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable verbose mode")

	// rootCmd.AddCommand(newSetupCmd())
	// rootCmd.AddCommand(newNodeCmd())
	rootCmd.AddCommand(listCmd)

	if err := rootCmd.Execute(); err != nil {
		if log != nil {
			log.Fatal("Command execution failed: %v", err)
		} else {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// func newJavaCmd() *cobra.Command {
// 	javaCmd := &cobra.Command{
// 		Use:   "java",
// 		Short: "Manage JDK versions",
// 		Run: func(cmd *cobra.Command, args []string) {
// 			cmd.Help()
// 		},
// 	}

// 	installCmd := &cobra.Command{
// 		Use:   "install [major version]",
// 		Short: "Install a specific major version of JDK",
// 		Args:  cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			javaService := java.NewService(log, installDir)
// 			_, err := javaService.Install(args[0])
// 			return err
// 		},
// 	}

// 	useCmd := &cobra.Command{
// 		Use:     "use [version]",
// 		Short:   "Use a specific version of JDK",
// 		Aliases: []string{"set"},
// 		Args:    cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			javaService := java.NewService(log, installDir)
// 			symlinkPath := os.Getenv("AEM_JAVA_SYMLINK")
// 			return javaService.Use(args[0], symlinkPath)
// 		},
// 	}

// 	listCmd := &cobra.Command{
// 		Use:     "list",
// 		Short:   "List installed JDK versions",
// 		Aliases: []string{"ls"},
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			javaService := java.NewService(log, installDir)
// 			versions, err := javaService.List()
// 			if err != nil {
// 				return err
// 			}

// 			if len(versions) == 0 {
// 				fmt.Println("No JDK versions installed")
// 				return nil
// 			}

// 			for _, version := range versions {
// 				fmt.Println(version)
// 			}
// 			return nil
// 		},
// 	}

// 	currentCmd := &cobra.Command{
// 		Use:   "current",
// 		Short: "List the current JDK versions",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			javaService := java.NewService(log, installDir)
// 			version, err := javaService.GetCurrentJDKVersion()
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Printf("*  %s", version)
// 			return nil
// 		},
// 	}

// 	removeCmd := &cobra.Command{
// 		Use:     "remove",
// 		Short:   "Remove the installed JDK version",
// 		Aliases: []string{"rm"},
// 		Args:    cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			javaService := java.NewService(log, installDir)
// 			err := javaService.Uninstall(args[0])
// 			if err != nil {
// 				return err
// 			}

// 			return nil
// 		},
// 	}

// 	javaCmd.AddCommand(installCmd)
// 	javaCmd.AddCommand(useCmd)
// 	javaCmd.AddCommand(listCmd)
// 	javaCmd.AddCommand(currentCmd)
// 	javaCmd.AddCommand(removeCmd)

// 	return javaCmd
// }

// func newNodeCmd() *cobra.Command {

// 	nodeCmd := &cobra.Command{
// 		Use:   "node",
// 		Short: "Manage Node.js versions",
// 		Run: func(cmd *cobra.Command, args []string) {
// 			cmd.Help()
// 		},
// 	}

// 	installCmd := &cobra.Command{
// 		Use:   "install [major version]",
// 		Short: "Install a specific major version of Node.js",
// 		Args:  cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			nodeService := node.NewService(log, installDir)
// 			_, err := nodeService.Install(args[0])
// 			return err
// 		},
// 	}

// 	useCmd := &cobra.Command{
// 		Use:     "use [version]",
// 		Short:   "Use a specific version of Node.js",
// 		Aliases: []string{"set"},
// 		Args:    cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			nodeService := node.NewService(log, installDir)
// 			symlinkPath := os.Getenv("AEM_NODE_SYMLINK")
// 			return nodeService.Use(args[0], symlinkPath)
// 		},
// 	}

// 	listCmd := &cobra.Command{
// 		Use:     "list",
// 		Short:   "List installed NodeJS versions",
// 		Aliases: []string{"ls"},
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			nodeService := node.NewService(log, installDir)
// 			versions, err := nodeService.List()
// 			if err != nil {
// 				return err
// 			}

// 			if len(versions) == 0 {
// 				fmt.Println("No NodeJS versions installed")
// 				return nil
// 			}

// 			for _, version := range versions {
// 				fmt.Println(version)
// 			}
// 			return nil
// 		},
// 	}

// 	currentCmd := &cobra.Command{
// 		Use:   "current",
// 		Short: "List the current NodeJS versions",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			nodeService := node.NewService(log, installDir)
// 			version, err := nodeService.GetCurrentNodeVersion()
// 			if err != nil {
// 				return err
// 			}
// 			fmt.Printf("*  %s", version)
// 			return nil
// 		},
// 	}

// 	removeCmd := &cobra.Command{
// 		Use:     "remove",
// 		Short:   "Remove the installed NodeJS version",
// 		Aliases: []string{"rm"},
// 		Args:    cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			nodeService := node.NewService(log, installDir)
// 			err := nodeService.Uninstall(args[0])
// 			if err != nil {
// 				return err
// 			}

// 			return nil
// 		},
// 	}

// 	nodeCmd.AddCommand(installCmd)
// 	nodeCmd.AddCommand(useCmd)
// 	nodeCmd.AddCommand(listCmd)
// 	nodeCmd.AddCommand(currentCmd)
// 	nodeCmd.AddCommand(removeCmd)

// 	return nodeCmd

// }

// func newSetupCmd() *cobra.Command {

// 	setupCmd := &cobra.Command{
// 		Use:   "setup",
// 		Short: "Setup development environment from aem.json",
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			installDir, dirErr := getInstallDir()
// 			if dirErr != nil {
// 				return dirErr
// 			}
// 			setupService := setup.NewService(log, installDir)
// 			return setupService.Setup()
// 		},
// 	}

// 	return setupCmd
// }

// func getInstallDir() (string, error) {
// 	return fs.GetInstallDir()
// }

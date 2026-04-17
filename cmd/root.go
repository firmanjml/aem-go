package cmd

import (
	"aem/assets"
	javaext "aem/extensions/java"
	nodeext "aem/extensions/node"
	javasvc "aem/internal/java"
	nodesvc "aem/internal/node"
	"aem/internal/setup"
	"aem/internal/manager"
	"aem/pkg/filesystem"
	"aem/pkg/logger"
	"aem/pkg/process"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

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

var installCmd = &cobra.Command{
	Use:   "install [module] [version]",
	Short: "Install a Java or Node version",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		module := args[0]
		version := args[1]

		installDir, err := fs.GetInstallDir()
		if err != nil {
			return err
		}

		switch module {
		case "node":
			service := nodesvc.NewService(log, installDir)
			installedVersion, err := service.Install(version)
			if err != nil {
				return err
			}
			fmt.Printf("Installed node %s\n", installedVersion)
			return nil
		case "java":
			service := javasvc.NewService(log, installDir)
			installedVersion, err := service.Install(version)
			if err != nil {
				return err
			}
			fmt.Printf("Installed java %s\n", strings.TrimPrefix(installedVersion, "v"))
			return nil
		default:
			return fmt.Errorf("%s module does not exist", module)
		}
	},
}

var useCmd = &cobra.Command{
	Use:     "use [module] [version]",
	Aliases: []string{"set"},
	Short:   "Switch the active Java or Node version",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		module := args[0]
		version := args[1]

		installDir, err := fs.GetInstallDir()
		if err != nil {
			return err
		}

		switch module {
		case "node":
			service := nodesvc.NewService(log, installDir)
			symlinkPath, err := resolveRuntimeSymlinkPath("AEM_NODE_SYMLINK", "node")
			if err != nil {
				return err
			}
			if err := service.Use(version, symlinkPath); err != nil {
				return err
			}
			fmt.Printf("Using node %s\n", version)
			return nil
		case "java":
			service := javasvc.NewService(log, installDir)
			symlinkPath, err := resolveRuntimeSymlinkPath("AEM_JAVA_SYMLINK", "java")
			if err != nil {
				return err
			}
			if err := service.Use(normalizeJavaVersion(version), symlinkPath); err != nil {
				return err
			}
			fmt.Printf("Using java %s\n", strings.TrimPrefix(version, "v"))
			return nil
		default:
			return fmt.Errorf("%s module does not exist", module)
		}
	},
}

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active runtimes",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := fs.GetState()
		if err != nil {
			return err
		}

		nodeVersion, err := st.CurrentNodeVersion()
		if err != nil {
			return err
		}

		javaVersion, err := st.CurrentJavaVersion()
		if err != nil {
			return err
		}

		androidPath, err := st.CurrentAndroidPath()
		if err != nil {
			return err
		}

		printCurrent("node", nodeVersion)
		printCurrent("java", javaVersion)
		if androidPath == "" {
			fmt.Println("android: none")
		} else {
			fmt.Printf("android: %s\n", androidPath)
		}

		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Show local AEM state and health checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		aemHome, err := fs.GetAEMHome()
		if err != nil {
			return err
		}
		installDir, err := fs.GetInstallDir()
		if err != nil {
			return err
		}
		currentRoot, err := fs.GetCurrentRoot()
		if err != nil {
			return err
		}
		st, err := fs.GetState()
		if err != nil {
			return err
		}

		nodeVersion, err := st.CurrentNodeVersion()
		if err != nil {
			return err
		}
		javaVersion, err := st.CurrentJavaVersion()
		if err != nil {
			return err
		}
		androidPath, err := st.CurrentAndroidPath()
		if err != nil {
			return err
		}

		fmt.Printf("AEM home: %s\n", aemHome)
		fmt.Printf("Install dir: %s\n", installDir)
		fmt.Printf("Current links: %s\n", currentRoot)
		fmt.Printf("Node installed: %d\n", countInstalledDirs(filepath.Join(installDir, "node")))
		fmt.Printf("Java installed: %d\n", countInstalledDirs(filepath.Join(installDir, "java")))
		printDoctorRuntime("node", nodeVersion, filepath.Join(currentRoot, "node"))
		printDoctorRuntime("java", javaVersion, filepath.Join(currentRoot, "java"))
		if androidPath == "" {
			fmt.Printf("android current: missing (%s)\n", filepath.Join(currentRoot, "android"))
		} else {
			fmt.Printf("android current: %s\n", androidPath)
		}
		if legacyVersionsExists(aemHome) {
			fmt.Printf("versions.json: present (%s)\n", filepath.Join(aemHome, "versions.json"))
		} else {
			fmt.Println("versions.json: absent")
		}

		return nil
	},
}

func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	process.SetContext(ctx)

	nodeExtension := nodeext.NewNodeExtension()
	javaExtension := javaext.NewJavaExtension()
	extensionMgr.RegisterExtension("node", nodeExtension)
	extensionMgr.RegisterExtension("java", javaExtension)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable verbose mode")

	rootCmd.AddCommand(newSetupCmd())
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(doctorCmd)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		if log != nil {
			log.Fatal("Command execution failed: %v", err)
		} else {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func printCurrent(name, version string) {
	if version == "" {
		fmt.Printf("%s: none\n", name)
		return
	}
	fmt.Printf("%s: %s\n", name, version)
}

func printDoctorRuntime(name, version, linkPath string) {
	if version == "" {
		fmt.Printf("%s current: missing (%s)\n", name, linkPath)
		return
	}
	fmt.Printf("%s current: %s\n", name, version)
}

func countInstalledDirs(path string) int {
	entries, err := fs.ListDir(path)
	if err != nil {
		return 0
	}

	total := 0
	for _, entry := range entries {
		if entry.IsDir() {
			total++
		}
	}
	return total
}

func legacyVersionsExists(aemHome string) bool {
	return fs.Exists(filepath.Join(aemHome, "versions.json"))
}

func resolveRuntimeSymlinkPath(envName, module string) (string, error) {
	if value := os.Getenv(envName); value != "" {
		return value, nil
	}

	currentRoot, err := fs.GetCurrentRoot()
	if err != nil {
		return "", err
	}

	return filepath.Join(currentRoot, module), nil
}

func normalizeJavaVersion(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "v" + version
}

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Setup development environment from the nearest aem.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			installDir, err := fs.GetInstallDir()
			if err != nil {
				return err
			}

			setupService := setup.NewService(log, installDir)
			return setupService.Setup()
		},
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

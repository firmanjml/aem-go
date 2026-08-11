package setup

import (
	"aem/internal/android"
	"aem/internal/config"
	"aem/internal/java"
	"aem/internal/node"
	"aem/pkg/logger"
	"aem/pkg/process"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type Service struct {
	logger  *logger.Logger
	node    *node.Service
	java    *java.Service
	android *android.Service
}

func NewService(logger *logger.Logger, installDir string) *Service {
	return &Service{
		logger:  logger,
		node:    node.NewService(logger, installDir),
		java:    java.NewService(logger, installDir),
		android: android.NewService(logger, installDir),
	}
}

func (s *Service) Setup() error {
	s.logger.Info("Starting environment setup")

	configPath, err := config.FindProjectConfig("")
	if err != nil {
		return err
	}
	s.logger.Debug("Using project config: %s", configPath)

	projectConfig, err := config.LoadProjectConfig(configPath)
	if err != nil {
		return err
	}
	for _, warning := range projectConfig.MigrationWarnings() {
		s.logger.Info("Configuration migration: %s", warning)
	}

	projectDir := filepath.Dir(configPath)
	if err := runHooks(projectConfig.Hooks.PreSetup, projectDir, "preSetup"); err != nil {
		return err
	}

	javaHome, err := s.setupCoreRuntimes(projectConfig)
	if err != nil {
		return err
	}

	if err := s.setupAndroid(projectConfig.Android, javaHome); err != nil {
		return err
	}
	if err := runHooks(projectConfig.Hooks.PostSetup, projectDir, "postSetup"); err != nil {
		return err
	}

	s.logger.Info("Environment setup completed successfully")
	return nil
}

func (s *Service) setupCoreRuntimes(projectConfig *config.ProjectConfig) (string, error) {
	var wg sync.WaitGroup

	nodeErrCh := make(chan error, 1)
	javaErrCh := make(chan error, 1)
	javaHomeCh := make(chan string, 1)

	if projectConfig.Runtime.Node != "" {
		wg.Add(1)
		go func(version string) {
			defer wg.Done()
			nodeErrCh <- s.setupNode(version)
		}(projectConfig.Runtime.Node)
	} else {
		s.logger.Debug("No Node.js version specified in config")
	}

	if projectConfig.Runtime.Java != "" {
		wg.Add(1)
		go func(version string) {
			defer wg.Done()
			javaHome, err := s.setupJava(version)
			if err != nil {
				javaErrCh <- err
				return
			}
			javaHomeCh <- javaHome
			javaErrCh <- nil
		}(projectConfig.Runtime.Java)
	} else {
		s.logger.Debug("No JDK version specified in config")
	}

	wg.Wait()
	close(nodeErrCh)
	close(javaErrCh)
	close(javaHomeCh)

	for err := range nodeErrCh {
		if err != nil {
			return "", err
		}
	}

	for err := range javaErrCh {
		if err != nil {
			return "", err
		}
	}

	for javaHome := range javaHomeCh {
		return javaHome, nil
	}

	return "", nil
}

func (s *Service) setupNode(version string) error {
	s.logger.Debug("Setting up Node.js version: %s", version)

	// Normalize version format
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Install Node.js
	lastestNodeVersion, err := s.node.Install(version)
	if err != nil {
		return fmt.Errorf("failed to install Node.js: %w", err)
	}

	if lastestNodeVersion == "" {
		return fmt.Errorf("failed to find installed version for %s", version)
	}

	symlinkPath, err := resolveSymlinkPath("AEM_NODE_SYMLINK", "current", "node")
	if err != nil {
		return err
	}

	if err := s.node.Use(lastestNodeVersion, symlinkPath); err != nil {
		return fmt.Errorf("failed to set Node.js version: %w", err)
	}

	return nil
}

func (s *Service) setupJava(version string) (string, error) {
	s.logger.Debug("Setting up JDK version: %s", version)

	lastestJdkVersion, err := s.java.Install(version)
	if err != nil {
		return "", fmt.Errorf("failed to install JDK: %w", err)
	}

	if lastestJdkVersion == "" {
		return "", fmt.Errorf("failed to find installed version for %s", version)
	}

	symlinkPath, err := resolveSymlinkPath("AEM_JAVA_SYMLINK", "current", "java")
	if err != nil {
		return "", err
	}

	if err := s.java.Use(lastestJdkVersion, symlinkPath); err != nil {
		return "", fmt.Errorf("failed to set JDK version: %w", err)
	}

	javaHome, err := resolveJavaHome(symlinkPath)
	if err != nil {
		return "", err
	}
	return javaHome, nil
}

// resolveJavaHome finds the directory expected by Java tooling. macOS JDK
// bundles place it below Contents/Home, while Linux and Windows JDK archives
// normally expose bin directly from the installation root.
func resolveJavaHome(runtimePath string) (string, error) {
	runtimePath = filepath.Clean(runtimePath)
	candidates := []string{
		filepath.Join(runtimePath, "Contents", "Home"),
		runtimePath,
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "bin", javaExecutableName())); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("active JDK at %s does not contain %s", runtimePath, filepath.Join("bin", javaExecutableName()))
}

func javaExecutableName() string {
	if runtime.GOOS == "windows" {
		return "java.exe"
	}
	return "java"
}

func (s *Service) setupAndroid(cfg config.AndroidConfig, javaHome string) error {
	if len(cfg.Platforms) == 0 && len(cfg.NDK) == 0 && len(cfg.BuildTools) == 0 && len(cfg.CMake) == 0 && len(cfg.SystemImages) == 0 {
		s.logger.Debug("No Android SDK configuration specified in config")
		return nil
	}

	s.logger.Debug("Setting up Android SDK packages")
	if err := s.android.Setup(cfg, javaHome); err != nil {
		return err
	}

	return nil
}

func resolveSymlinkPath(envName string, defaults ...string) (string, error) {
	if value := strings.TrimSpace(getEnv(envName)); value != "" {
		return value, nil
	}

	aemHome := strings.TrimSpace(getEnv("AEM_HOME"))
	if aemHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("%s and AEM_HOME are not configured", envName)
		}
		aemHome = filepath.Join(homeDir, ".aem")
	}

	return filepath.Join(append([]string{aemHome}, defaults...)...), nil
}

var getEnv = func(key string) string {
	return os.Getenv(key)
}

func runHooks(hooks config.StringList, dir, phase string) error {
	for _, hook := range hooks {
		cmd := hookCommand(hook)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s hook %q failed: %w", phase, hook, err)
		}
	}
	return nil
}

func hookCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(process.Context(), "cmd", "/C", command)
	}
	return exec.CommandContext(process.Context(), "sh", "-c", command)
}

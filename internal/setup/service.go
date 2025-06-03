package setup

import (
	"aem/internal/java"
	"aem/internal/node"
	"aem/pkg/logger"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Service struct {
	logger *logger.Logger
	node   *node.Service
	java   *java.Service
}

type AEMConfig struct {
	Node    string        `json:"node"`
	JDK     string        `json:"jdk"`
	Android AndroidConfig `json:"android"`
}

type AndroidConfig struct {
	SDK       []string `json:"sdk"`
	NDK       []string `json:"ndk"`
	BuildTool []string `json:"build-tool"`
}

func NewService(logger *logger.Logger, installDir string) *Service {
	return &Service{
		logger: logger,
		node:   node.NewService(logger, installDir),
		java:   java.NewService(logger, installDir),
	}
}

func (s *Service) Setup() error {
	s.logger.Info("Starting environment setup")

	// Read aem.json file
	config, err := s.readConfig()
	if err != nil {
		return err
	}

	// Process Node.js installation if specified
	if config.Node != "" {
		if err := s.setupNode(config.Node); err != nil {
			return err
		}
	} else {
		s.logger.Info("No Node.js version specified in config")
	}

	// Process JDK installation if specified
	if config.JDK != "" {
		if err := s.setupJava(config.JDK); err != nil {
			return err
		}
	} else {
		s.logger.Info("No JDK version specified in config")
	}

	s.logger.Info("Environment setup completed successfully")
	return nil
}

func (s *Service) readConfig() (*AEMConfig, error) {
	s.logger.Debug("Reading aem.json configuration file")
	data, err := os.ReadFile("aem.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read aem.json: %w", err)
	}

	var config AEMConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse aem.json: %w", err)
	}

	return &config, nil
}

func (s *Service) setupNode(version string) error {
	s.logger.Info("Setting up Node.js version: %s", version)

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

	// Set Node.js version
	symlinkPath := os.Getenv("AEM_NODE_SYMLINK")
	if err := s.node.Use(lastestNodeVersion, symlinkPath); err != nil {
		return fmt.Errorf("failed to set Node.js version: %w", err)
	}

	return nil
}

func (s *Service) setupJava(version string) error {
	s.logger.Info("Setting up JDK version: %s", version)

	// Install JDK
	lastestJdkVersion, err := s.java.Install(version)
	if err != nil {
		return fmt.Errorf("failed to install JDK: %w", err)
	}

	if lastestJdkVersion == "" {
		return fmt.Errorf("failed to find installed version for %s", version)
	}

	// Set JDK version
	symlinkPath := os.Getenv("AEM_JAVA_SYMLINK")
	if err := s.java.Use(lastestJdkVersion, symlinkPath); err != nil {
		return fmt.Errorf("failed to set JDK version: %w", err)
	}

	return nil
}

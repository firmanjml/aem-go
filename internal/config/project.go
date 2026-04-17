package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ProjectConfigFileName = "aem.json"

type ProjectConfig struct {
	Node    string        `json:"node"`
	JDK     string        `json:"jdk"`
	Android AndroidConfig `json:"android"`
}

type AndroidConfig struct {
	SDK       StringList `json:"sdk"`
	NDK       StringList `json:"ndk"`
	BuildTool StringList `json:"build-tool"`
}

type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		if single == "" {
			*s = nil
			return nil
		}
		*s = []string{single}
		return nil
	}

	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}

	*s = many
	return nil
}

func FindProjectConfig(startDir string) (string, error) {
	if startDir == "" {
		var err error
		startDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to determine working directory: %w", err)
		}
	}

	current := startDir
	for {
		candidate := filepath.Join(current, ProjectConfigFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("%s not found in %s or any parent directory", ProjectConfigFileName, startDir)
		}
		current = parent
	}
}

func LoadProjectConfig(configPath string) (*ProjectConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	return &cfg, nil
}

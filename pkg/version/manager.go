// pkg/version/manager.go
package version

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
	"encoding/json"
	"os"
	"sync"
)

const VersionsFileName = "versions.json"

type Manager struct {
	logger     *logger.Logger
	configPath string
	mu         sync.RWMutex
}

type VersionConfig struct {
	Node string `json:"node,omitempty"`
	Java string `json:"java,omitempty"`
}

func NewManager(logger *logger.Logger, configPath string) *Manager {
	if configPath == "" {
		configPath = VersionsFileName
	}
	return &Manager{
		logger:     logger,
		configPath: configPath,
	}
}

func (m *Manager) GetConfig() (*VersionConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.fileExists() {
		// Return empty config if file doesn't exist
		return &VersionConfig{}, nil
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, errors.NewFileSystemError("failed to read versions config", err)
	}

	var config VersionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.NewFileSystemError("failed to parse versions config", err)
	}

	return &config, nil
}

func (m *Manager) SetNodeVersion(version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, err := m.getConfigUnsafe()
	if err != nil {
		return err
	}

	config.Node = version
	return m.saveConfigUnsafe(config)
}

func (m *Manager) SetJavaVersion(version string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, err := m.getConfigUnsafe()
	if err != nil {
		return err
	}

	config.Java = version
	return m.saveConfigUnsafe(config)
}

func (m *Manager) GetNodeVersion() (string, error) {
	config, err := m.GetConfig()
	if err != nil {
		return "", err
	}

	if config.Node == "" {
		return "no current version", nil
	}

	return config.Node, nil
}

func (m *Manager) GetJavaVersion() (string, error) {
	config, err := m.GetConfig()
	if err != nil {
		return "", err
	}

	if config.Java == "" {
		return "no current version", nil
	}

	return config.Java, nil
}

func (m *Manager) ClearNodeVersion() error {
	return m.SetNodeVersion("")
}

func (m *Manager) ClearJavaVersion() error {
	return m.SetJavaVersion("")
}

// Internal methods (not thread-safe, require external locking)
func (m *Manager) getConfigUnsafe() (*VersionConfig, error) {
	if !m.fileExists() {
		return &VersionConfig{}, nil
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, errors.NewFileSystemError("failed to read versions config", err)
	}

	var config VersionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.NewFileSystemError("failed to parse versions config", err)
	}

	return &config, nil
}

func (m *Manager) saveConfigUnsafe(config *VersionConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.NewFileSystemError("failed to marshal versions config", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return errors.NewFileSystemError("failed to write versions config", err)
	}

	m.logger.Debug("Saved versions config to %s", m.configPath)
	return nil
}

func (m *Manager) fileExists() bool {
	_, err := os.Stat(m.configPath)
	return err == nil
}

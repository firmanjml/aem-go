package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProjectConfigFileName = "aem.json"
	SchemaURL             = "https://raw.githubusercontent.com/firmanjml/aem-go/main/aem.schema.json"
)

type ProjectConfig struct {
	Schema  string        `json:"$schema,omitempty"`
	Runtime RuntimeConfig `json:"runtime,omitempty"`
	Android AndroidConfig `json:"android,omitempty"`
	Hooks   HooksConfig   `json:"hooks,omitempty"`

	legacyFields []string
}

type RuntimeConfig struct {
	Node string `json:"node,omitempty"`
	Java string `json:"java,omitempty"`
}

type AndroidConfig struct {
	Platforms    StringList    `json:"platforms,omitempty"`
	BuildTools   StringList    `json:"buildTools,omitempty"`
	NDK          StringList    `json:"ndk,omitempty"`
	CMake        StringList    `json:"cmake,omitempty"`
	SystemImages []SystemImage `json:"systemImages,omitempty"`

	legacyFields []string
}

type SystemImage struct {
	APILevel     int    `json:"apiLevel"`
	Variant      string `json:"variant"`
	Architecture string `json:"architecture"`
	Package      string `json:"package,omitempty"`
}

type HooksConfig struct {
	PreSetup  StringList `json:"preSetup,omitempty"`
	PostSetup StringList `json:"postSetup,omitempty"`
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

func (c *ProjectConfig) UnmarshalJSON(data []byte) error {
	type projectWire struct {
		Schema  string        `json:"$schema"`
		Runtime RuntimeConfig `json:"runtime"`
		Node    string        `json:"node"`
		JDK     string        `json:"jdk"`
		Android AndroidConfig `json:"android"`
		Hooks   HooksConfig   `json:"hooks"`
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var wire projectWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	_, hasRuntime := raw["runtime"]
	_, hasNode := raw["node"]
	_, hasJDK := raw["jdk"]
	if hasRuntime && (hasNode || hasJDK) {
		return fmt.Errorf("runtime cannot be combined with legacy node or jdk fields")
	}

	c.Schema = wire.Schema
	c.Runtime = wire.Runtime
	c.Android = wire.Android
	c.Hooks = wire.Hooks
	c.legacyFields = nil
	if hasNode {
		c.Runtime.Node = wire.Node
		c.legacyFields = append(c.legacyFields, "node")
	}
	if hasJDK {
		c.Runtime.Java = wire.JDK
		c.legacyFields = append(c.legacyFields, "jdk")
	}
	c.legacyFields = append(c.legacyFields, c.Android.legacyFields...)
	return nil
}

func (c *AndroidConfig) UnmarshalJSON(data []byte) error {
	type androidWire struct {
		Platforms    StringList    `json:"platforms"`
		BuildTools   StringList    `json:"buildTools"`
		NDK          StringList    `json:"ndk"`
		CMake        StringList    `json:"cmake"`
		SystemImages []SystemImage `json:"systemImages"`
		SDK          StringList    `json:"sdk"`
		BuildTool    StringList    `json:"build-tool"`
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var wire androidWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if err := requireStringArrays(raw, "platforms", "buildTools", "ndk", "cmake"); err != nil {
		return err
	}

	_, hasPlatforms := raw["platforms"]
	_, hasSDK := raw["sdk"]
	_, hasBuildTools := raw["buildTools"]
	_, hasBuildTool := raw["build-tool"]
	if hasPlatforms && hasSDK {
		return fmt.Errorf("android.platforms cannot be combined with legacy android.sdk")
	}
	if hasBuildTools && hasBuildTool {
		return fmt.Errorf("android.buildTools cannot be combined with legacy android.build-tool")
	}

	c.Platforms = wire.Platforms
	c.BuildTools = wire.BuildTools
	c.NDK = wire.NDK
	c.CMake = wire.CMake
	c.SystemImages = wire.SystemImages
	c.legacyFields = nil
	if hasSDK {
		c.Platforms = wire.SDK
		c.legacyFields = append(c.legacyFields, "android.sdk")
	}
	if hasBuildTool {
		c.BuildTools = wire.BuildTool
		c.legacyFields = append(c.legacyFields, "android.build-tool")
	}
	return nil
}

func (c *HooksConfig) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := requireStringArrays(raw, "preSetup", "postSetup"); err != nil {
		return err
	}
	type hooksWire HooksConfig
	var wire hooksWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*c = HooksConfig(wire)
	return nil
}

func requireStringArrays(raw map[string]json.RawMessage, fields ...string) error {
	for _, field := range fields {
		data, exists := raw[field]
		if !exists {
			continue
		}
		var values []string
		if err := json.Unmarshal(data, &values); err != nil {
			return fmt.Errorf("%s must be an array of strings", field)
		}
	}
	return nil
}

func (c *ProjectConfig) Validate() error {
	if err := validateVersion("runtime.node", c.Runtime.Node); err != nil {
		return err
	}
	if err := validateVersion("runtime.java", c.Runtime.Java); err != nil {
		return err
	}
	for _, entry := range []struct {
		name   string
		values StringList
	}{
		{"android.platforms", c.Android.Platforms},
		{"android.buildTools", c.Android.BuildTools},
		{"android.ndk", c.Android.NDK},
		{"android.cmake", c.Android.CMake},
		{"hooks.preSetup", c.Hooks.PreSetup},
		{"hooks.postSetup", c.Hooks.PostSetup},
	} {
		for _, value := range entry.values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s cannot contain an empty value", entry.name)
			}
		}
	}
	for _, image := range c.Android.SystemImages {
		if image.APILevel <= 0 || strings.TrimSpace(image.Variant) == "" || strings.TrimSpace(image.Architecture) == "" {
			return fmt.Errorf("android.systemImages entries require apiLevel, variant, and architecture")
		}
		if image.Package != "" && strings.TrimSpace(image.Package) != image.Package {
			return fmt.Errorf("android.systemImages package cannot begin or end with whitespace")
		}
	}
	return nil
}

func validateVersion(name, value string) error {
	if strings.TrimSpace(value) != value {
		return fmt.Errorf("%s cannot begin or end with whitespace", name)
	}
	return nil
}

func (c *ProjectConfig) MigrationWarnings() []string {
	if len(c.legacyFields) == 0 {
		return nil
	}
	fields := append([]string(nil), c.legacyFields...)
	return []string{fmt.Sprintf("legacy fields (%s) are supported temporarily; migrate to runtime.node/runtime.java and android.platforms/android.buildTools", strings.Join(fields, ", "))}
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
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", configPath, err)
	}

	return &cfg, nil
}

func NewProjectConfig() *ProjectConfig {
	return &ProjectConfig{
		Schema: SchemaURL,
		Hooks: HooksConfig{
			PreSetup:  StringList{},
			PostSetup: StringList{},
		},
	}
}

func WriteProjectConfig(path string, cfg *ProjectConfig, overwrite bool) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid project configuration: %w", err)
	}
	if _, err := os.Stat(path); err == nil && !overwrite {
		return fmt.Errorf("%s already exists; use --force to overwrite it", path)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect %s: %w", path, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode project configuration: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

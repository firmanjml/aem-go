package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindProjectConfigFindsNearestAncestor(t *testing.T) {
	root := t.TempDir()
	parentConfig := filepath.Join(root, ProjectConfigFileName)
	if err := os.WriteFile(parentConfig, []byte(`{"node":"20"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "one", "two")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	nearestConfig := filepath.Join(root, "one", ProjectConfigFileName)
	if err := os.WriteFile(nearestConfig, []byte(`{"node":"22"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := FindProjectConfig(nested)
	if err != nil {
		t.Fatalf("FindProjectConfig() error = %v", err)
	}
	if got != nearestConfig {
		t.Fatalf("FindProjectConfig() = %q, want %q", got, nearestConfig)
	}
}

func TestFindProjectConfigReturnsHelpfulErrorWhenMissing(t *testing.T) {
	start := filepath.Join(t.TempDir(), "nested")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := FindProjectConfig(start)
	if err == nil || !strings.Contains(err.Error(), ProjectConfigFileName+" not found") {
		t.Fatalf("FindProjectConfig() error = %v, want missing-config error", err)
	}
}

func TestLoadProjectConfigParsesCanonicalRuntimeAndAndroidLists(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectConfigFileName)
	contents := `{
  "$schema": "` + SchemaURL + `",
  "runtime": {"node": "20.19.4", "java": "17"},
  "android": {
    "platforms": ["35"],
    "buildTools": ["35.0.0", "34.0.0"],
    "ndk": ["27.0.12077973"],
    "cmake": ["3.22.1"],
    "systemImages": [{"apiLevel": 35, "variant": "google_apis", "architecture": "arm64-v8a"}]
  },
  "hooks": {"preSetup": ["echo before"], "postSetup": ["echo after"]}
}`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(path)
	if err != nil {
		t.Fatalf("LoadProjectConfig() error = %v", err)
	}
	if cfg.Runtime.Node != "20.19.4" || cfg.Runtime.Java != "17" {
		t.Fatalf("runtime = node %q, java %q", cfg.Runtime.Node, cfg.Runtime.Java)
	}
	if got, want := strings.Join(cfg.Android.Platforms, ","), "35"; got != want {
		t.Fatalf("android.platforms = %q, want %q", got, want)
	}
	if got, want := strings.Join(cfg.Android.BuildTools, ","), "35.0.0,34.0.0"; got != want {
		t.Fatalf("android.buildTools = %q, want %q", got, want)
	}
}

func TestLoadProjectConfigMigratesLegacyFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectConfigFileName)
	if err := os.WriteFile(path, []byte(`{"node":"20", "jdk":"17", "android":{"sdk":"35", "build-tool":"35.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(path)
	if err != nil {
		t.Fatalf("LoadProjectConfig() error = %v", err)
	}
	if cfg.Runtime.Node != "20" || cfg.Runtime.Java != "17" {
		t.Fatalf("legacy runtime = %+v", cfg.Runtime)
	}
	if got := strings.Join(cfg.Android.Platforms, ","); got != "35" {
		t.Fatalf("legacy android.sdk = %q", got)
	}
	if len(cfg.MigrationWarnings()) != 1 {
		t.Fatalf("MigrationWarnings() = %v, want one warning", cfg.MigrationWarnings())
	}
}

func TestLoadProjectConfigRejectsMixedLegacyAndCanonicalFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectConfigFileName)
	if err := os.WriteFile(path, []byte(`{"node":"20", "runtime":{"node":"22"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadProjectConfig(path); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("LoadProjectConfig() error = %v, want mixed-format error", err)
	}
}

func TestLoadProjectConfigRequiresCanonicalListsToBeArrays(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectConfigFileName)
	if err := os.WriteFile(path, []byte(`{"android":{"platforms":"35"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadProjectConfig(path); err == nil || !strings.Contains(err.Error(), "must be an array") {
		t.Fatalf("LoadProjectConfig() error = %v, want array validation error", err)
	}
}

func TestWriteProjectConfigCreatesValidCanonicalFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectConfigFileName)
	if err := WriteProjectConfig(path, NewProjectConfig(), false); err != nil {
		t.Fatalf("WriteProjectConfig() error = %v", err)
	}
	cfg, err := LoadProjectConfig(path)
	if err != nil {
		t.Fatalf("LoadProjectConfig() error = %v", err)
	}
	if cfg.Schema != SchemaURL {
		t.Fatalf("$schema = %q, want %q", cfg.Schema, SchemaURL)
	}
	if err := WriteProjectConfig(path, NewProjectConfig(), false); err == nil {
		t.Fatal("WriteProjectConfig() overwrote an existing file without force")
	}
}

func TestLoadProjectConfigRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectConfigFileName)
	if err := os.WriteFile(path, []byte(`{"node":`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadProjectConfig(path)
	if err == nil || !strings.Contains(err.Error(), "failed to parse") {
		t.Fatalf("LoadProjectConfig() error = %v, want parse error", err)
	}
}

func TestStringListRejectsNonStringValues(t *testing.T) {
	var values StringList
	if err := values.UnmarshalJSON([]byte(`[1]`)); err == nil {
		t.Fatal("UnmarshalJSON() succeeded for non-string value")
	}
}

package cmd

import (
	"aem/internal/config"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitCreatesValidCanonicalConfiguration(t *testing.T) {
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	cmd := newInitCmd()
	cmd.SetArgs([]string{"--non-interactive"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("aem init error = %v", err)
	}

	path := filepath.Join(dir, config.ProjectConfigFileName)
	cfg, err := config.LoadProjectConfig(path)
	if err != nil {
		t.Fatalf("generated aem.json did not load: %v", err)
	}
	if cfg.Schema != config.SchemaURL {
		t.Fatalf("$schema = %q, want %q", cfg.Schema, config.SchemaURL)
	}
	if err := cmd.Execute(); err == nil {
		t.Fatal("aem init overwrote an existing configuration")
	}
}

func TestInitInteractivelySelectsDetectedComponents(t *testing.T) {
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	cmd := newInitCmdWithDiscovery(
		func() (initCandidates, error) {
			return initCandidates{
				Node:    []runtimeCandidate{{Version: "22.14.0", Source: "AEM"}, {Version: "20.19.4", Source: "PATH"}},
				Java:    []runtimeCandidate{{Version: "21.0.6", Source: "AEM"}},
				Android: []string{"platforms;android-35", "build-tools;35.0.0"},
			}, nil
		},
		func(_ *cobra.Command, candidates initCandidates) (config.RuntimeConfig, config.AndroidConfig, error) {
			if len(candidates.Android) != 2 {
				t.Fatalf("Android candidates = %q, want two values", candidates.Android)
			}
			return config.RuntimeConfig{Node: "20.19.4", Java: "21.0.6"}, config.AndroidConfig{
				Platforms:  config.StringList{"35"},
				BuildTools: config.StringList{"35.0.0"},
			}, nil
		},
	)
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("interactive aem init error = %v", err)
	}

	cfg, err := config.LoadProjectConfig(filepath.Join(dir, config.ProjectConfigFileName))
	if err != nil {
		t.Fatalf("generated aem.json did not load: %v", err)
	}
	if cfg.Runtime.Node != "20.19.4" || cfg.Runtime.Java != "21.0.6" {
		t.Fatalf("runtime = %+v, want node 20.19.4 and java 21.0.6", cfg.Runtime)
	}
	if got := []string(cfg.Android.Platforms); !reflect.DeepEqual(got, []string{"35"}) {
		t.Fatalf("android.platforms = %q, want [35]", got)
	}
}

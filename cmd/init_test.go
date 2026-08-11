package cmd

import (
	"aem/internal/config"
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestManagedRuntimeCandidatesReadsAEMInstallations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AEM_HOME", home)
	for _, path := range []string{
		filepath.Join(home, "sys_installed", "node", "v20.19.4"),
		filepath.Join(home, "sys_installed", "node", "v22.14.0"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got := mergeAndSortCandidates(managedRuntimeCandidates("node"))
	want := []runtimeCandidate{
		{Version: "22.14.0", Source: "AEM"},
		{Version: "20.19.4", Source: "AEM"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("managedRuntimeCandidates() = %#v, want %#v", got, want)
	}
}

func TestMergeAndSortCandidatesDeduplicatesSources(t *testing.T) {
	got := mergeAndSortCandidates([]runtimeCandidate{
		{Version: "20.19.4", Source: "AEM"},
		{Version: "20.19.4", Source: "PATH"},
		{Version: "22.14.0", Source: "PATH"},
	})
	want := []runtimeCandidate{
		{Version: "22.14.0", Source: "PATH"},
		{Version: "20.19.4", Source: "AEM, PATH"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeAndSortCandidates() = %#v, want %#v", got, want)
	}
}

func TestAndroidConfigFromPackages(t *testing.T) {
	android, err := androidConfigFromPackages([]string{
		"platforms;android-35",
		"build-tools;35.0.0",
		"ndk;27.0.12077973",
		"cmake;3.22.1",
		"system-images;android-35;google_apis;arm64-v8a",
	})
	if err != nil {
		t.Fatalf("androidConfigFromPackages() error = %v", err)
	}
	if got := []string(android.Platforms); !reflect.DeepEqual(got, []string{"35"}) {
		t.Fatalf("platforms = %q, want [35]", got)
	}
	if got := []string(android.BuildTools); !reflect.DeepEqual(got, []string{"35.0.0"}) {
		t.Fatalf("build tools = %q, want [35.0.0]", got)
	}
	if !reflect.DeepEqual(android.SystemImages, []config.SystemImage{{APILevel: 35, Variant: "google_apis", Architecture: "arm64-v8a"}}) {
		t.Fatalf("system images = %#v", android.SystemImages)
	}
}

func TestAndroidConfigFromPackagesPreservesPreviewSystemImagePackage(t *testing.T) {
	const previewPackage = "system-images;android-37.0;google_apis_playstore_ps16k;arm64-v8a"
	android, err := androidConfigFromPackages([]string{previewPackage})
	if err != nil {
		t.Fatalf("androidConfigFromPackages() error = %v", err)
	}
	want := []config.SystemImage{{
		APILevel: 37, Variant: "google_apis_playstore_ps16k", Architecture: "arm64-v8a", Package: previewPackage,
	}}
	if !reflect.DeepEqual(android.SystemImages, want) {
		t.Fatalf("system images = %#v, want %#v", android.SystemImages, want)
	}
}

func TestReadTUIKeyRecognizesNavigationAndSelection(t *testing.T) {
	tests := []struct {
		input string
		want  tuiKey
	}{
		{"\x1b[A", tuiUp},
		{"\x1b[B", tuiDown},
		{" ", tuiSpace},
		{"\r", tuiEnter},
		{"q", tuiCancel},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := readTUIKey(bufio.NewReader(strings.NewReader(test.input)))
			if err != nil {
				t.Fatalf("readTUIKey() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("readTUIKey(%q) = %v, want %v", test.input, got, test.want)
			}
		})
	}
}

func TestRenderTUIUsesCarriageReturns(t *testing.T) {
	output := &bytes.Buffer{}
	renderTUI(output, "Node.js runtime", "Use arrows", func() {
		writeTUILine(output, "› Do not add a runtime requirement")
		writeTUILine(output, "  24.15.0 (AEM)")
	})

	want := "AEM init — Node.js runtime\r\n\r\nUse arrows\r\n\r\n› Do not add a runtime requirement\r\n  24.15.0 (AEM)\r\n"
	if !strings.Contains(output.String(), want) {
		t.Fatalf("rendered TUI = %q, want CRLF-aligned lines", output.String())
	}
}

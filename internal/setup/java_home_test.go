package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveJavaHomeUsesMacBundleHome(t *testing.T) {
	runtimePath := filepath.Join(t.TempDir(), "current", "java")
	javaPath := filepath.Join(runtimePath, "Contents", "Home", "bin", javaExecutableName())
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(javaPath, nil, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveJavaHome(runtimePath)
	if err != nil {
		t.Fatalf("resolveJavaHome() error = %v", err)
	}
	want := filepath.Join(runtimePath, "Contents", "Home")
	if got != want {
		t.Fatalf("resolveJavaHome() = %q, want %q", got, want)
	}
}

func TestResolveJavaHomeUsesStandardJDKHome(t *testing.T) {
	runtimePath := filepath.Join(t.TempDir(), "current", "java")
	javaPath := filepath.Join(runtimePath, "bin", javaExecutableName())
	if err := os.MkdirAll(filepath.Dir(javaPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(javaPath, nil, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveJavaHome(runtimePath)
	if err != nil {
		t.Fatalf("resolveJavaHome() error = %v", err)
	}
	if got != runtimePath {
		t.Fatalf("resolveJavaHome() = %q, want %q", got, runtimePath)
	}
}

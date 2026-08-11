package java

import (
	"aem/pkg/logger"
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallRefusesRuntimeActiveThroughCustomSymlink(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "installs")
	target := filepath.Join(installDir, "java", "v17.0.12")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "links", "java")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AEM_JAVA_SYMLINK", link)

	service := NewService(logger.New(false), installDir)
	if err := service.Uninstall("v17.0.12"); err == nil {
		t.Fatal("Uninstall() removed runtime selected through AEM_JAVA_SYMLINK")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("active runtime was removed: %v", err)
	}
}

func TestUseAcceptsVersionWithOrWithoutVPrefix(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "installs")
	target := filepath.Join(installDir, "java", "v17.0.19")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "current", "java")
	service := NewService(logger.New(false), installDir)

	for _, version := range []string{"17.0.19", "v17.0.19"} {
		if err := service.Use(version, link); err != nil {
			t.Fatalf("Use(%q) error = %v", version, err)
		}
		got, err := os.Readlink(link)
		if err != nil {
			t.Fatal(err)
		}
		if got != target {
			t.Fatalf("Use(%q) target = %q, want %q", version, got, target)
		}
	}
}

func TestNormalizeRequestedVersion(t *testing.T) {
	for input, want := range map[string]string{
		"17.0.19":  "17.0.19",
		"v17.0.19": "17.0.19",
		" v17 ":    "17",
	} {
		if got := normalizeRequestedVersion(input); got != want {
			t.Fatalf("normalizeRequestedVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

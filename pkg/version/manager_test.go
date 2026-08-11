package version

import (
	"aem/pkg/logger"
	"path/filepath"
	"testing"
)

func TestManagerPersistsAndClearsActiveVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), VersionsFileName)
	manager := NewManager(logger.New(false), path)

	if got, err := manager.GetNodeVersion(); err != nil || got != "no current version" {
		t.Fatalf("GetNodeVersion() before set = %q, %v", got, err)
	}
	if err := manager.SetNodeVersion("20.19.4"); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetJavaVersion("17.0.12"); err != nil {
		t.Fatal(err)
	}

	reloaded := NewManager(logger.New(false), path)
	if got, err := reloaded.GetNodeVersion(); err != nil || got != "20.19.4" {
		t.Fatalf("GetNodeVersion() = %q, %v", got, err)
	}
	if got, err := reloaded.GetJavaVersion(); err != nil || got != "17.0.12" {
		t.Fatalf("GetJavaVersion() = %q, %v", got, err)
	}
	if err := reloaded.ClearNodeVersion(); err != nil {
		t.Fatal(err)
	}
	if got, err := reloaded.GetNodeVersion(); err != nil || got != "no current version" {
		t.Fatalf("GetNodeVersion() after clear = %q, %v", got, err)
	}
}

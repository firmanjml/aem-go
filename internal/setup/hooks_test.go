package setup

import (
	"aem/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHooksExecutesCommandsInProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := runHooks(config.StringList{"echo complete > hook-result"}, dir, "preSetup"); err != nil {
		t.Fatalf("runHooks() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "hook-result"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != "complete" {
		t.Fatalf("hook result = %q, want %q", data, "complete")
	}
}

func TestRunHooksReportsFailedCommand(t *testing.T) {
	if err := runHooks(config.StringList{"exit 7"}, t.TempDir(), "postSetup"); err == nil {
		t.Fatal("runHooks() succeeded for failed hook")
	}
}

package node

import (
	"aem/pkg/logger"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDownloadURL(t *testing.T) {
	if got, want := resolveDownloadURL("https://nodejs.org/dist/v20.19.4", "/dist/v20.19.4/node.tar.gz"), "https://nodejs.org/dist/v20.19.4/node.tar.gz"; got != want {
		t.Fatalf("resolveDownloadURL(root relative) = %q, want %q", got, want)
	}
	if got, want := resolveDownloadURL("https://nodejs.org/dist/v20.19.4", "node.tar.gz"), "https://nodejs.org/dist/v20.19.4/node.tar.gz"; got != want {
		t.Fatalf("resolveDownloadURL(relative) = %q, want %q", got, want)
	}
}

func TestUninstallRefusesRuntimeActiveThroughCustomSymlink(t *testing.T) {
	root := t.TempDir()
	installDir := filepath.Join(root, "installs")
	target := filepath.Join(installDir, "node", "v20.19.4")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "links", "node")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AEM_NODE_SYMLINK", link)

	service := NewService(logger.New(false), installDir)
	if err := service.Uninstall("20.19.4"); err == nil {
		t.Fatal("Uninstall() removed runtime selected through AEM_NODE_SYMLINK")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("active runtime was removed: %v", err)
	}
}

//go:build windows

package filesystem

import (
	"aem/pkg/logger"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateSymlinkReplacesWindowsJunction(t *testing.T) {
	root := t.TempDir()
	firstTarget := filepath.Join(root, "first")
	secondTarget := filepath.Join(root, "second")
	for _, target := range []string{firstTarget, secondTarget} {
		if err := os.Mkdir(target, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "current", "node")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := createWindowsJunction(link, firstTarget); err != nil {
		t.Fatalf("createWindowsJunction() error = %v", err)
	}

	fs := New(logger.New(false))
	if err := fs.CreateSymlink(link, secondTarget); err != nil {
		t.Fatalf("CreateSymlink() error = %v", err)
	}

	info, err := os.Stat(link)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.Stat(secondTarget)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(info, want) {
		t.Fatalf("link %q does not target %q", link, secondTarget)
	}
}

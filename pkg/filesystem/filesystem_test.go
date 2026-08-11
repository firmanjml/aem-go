package filesystem

import (
	"aem/pkg/logger"
	"os"
	"path/filepath"
	"testing"
)

func TestAEMDirectoriesAreCreatedUnderConfiguredHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("AEM_HOME", home)
	fs := New(logger.New(false))

	for name, getPath := range map[string]func() (string, error){
		"home":      fs.GetAEMHome,
		"temporary": fs.GetTempDir,
		"install":   fs.GetInstallDir,
		"current":   fs.GetCurrentRoot,
	} {
		path, err := getPath()
		if err != nil {
			t.Fatalf("Get %s directory: %v", name, err)
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Fatalf("%s directory %q was not created", name, path)
		}
		if !filepath.IsAbs(path) || filepath.Dir(path) != home && path != home {
			t.Fatalf("%s directory %q is not under AEM_HOME %q", name, path, home)
		}
	}
}

func TestCreateSymlinkReplacesOnlyExistingSymlink(t *testing.T) {
	root := t.TempDir()
	fs := New(logger.New(false))
	firstTarget := filepath.Join(root, "first")
	secondTarget := filepath.Join(root, "second")
	for _, target := range []string{firstTarget, secondTarget} {
		if err := os.Mkdir(target, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "current", "node")

	if err := fs.CreateSymlink(link, firstTarget); err != nil {
		t.Fatalf("CreateSymlink(first) error = %v", err)
	}
	if err := fs.CreateSymlink(link, secondTarget); err != nil {
		t.Fatalf("CreateSymlink(second) error = %v", err)
	}

	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(secondTarget)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("link target = %q, want %q", got, want)
	}
}

func TestCreateSymlinkDoesNotReplaceRegularFile(t *testing.T) {
	root := t.TempDir()
	fs := New(logger.New(false))
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "current", "node")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(link, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := fs.CreateSymlink(link, target); err == nil {
		t.Fatal("CreateSymlink() succeeded when a regular file already exists")
	}
	contents, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "keep me" {
		t.Fatalf("regular file contents = %q, want preserved contents", contents)
	}
}

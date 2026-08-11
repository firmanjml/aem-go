package state

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeLinkReader struct {
	targets map[string]string
	errs    map[string]error
}

func (r fakeLinkReader) Readlink(name string) (string, error) {
	if err := r.errs[name]; err != nil {
		return "", err
	}
	target, ok := r.targets[name]
	if !ok {
		return "", os.ErrNotExist
	}
	return target, nil
}

func TestStateReadsCurrentRuntimeVersions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "current")
	reader := fakeLinkReader{targets: map[string]string{
		filepath.Join(root, "node"): filepath.Join("installs", "node", "v20.19.4"),
		filepath.Join(root, "java"): filepath.Join("installs", "java", "v17.0.12"),
	}}
	state := New(reader, root)

	if got, err := state.CurrentNodeVersion(); err != nil || got != "20.19.4" {
		t.Fatalf("CurrentNodeVersion() = %q, %v", got, err)
	}
	if got, err := state.CurrentJavaVersion(); err != nil || got != "17.0.12" {
		t.Fatalf("CurrentJavaVersion() = %q, %v", got, err)
	}
}

func TestStateTreatsMissingLinksAsNoActiveRuntime(t *testing.T) {
	state := New(fakeLinkReader{}, t.TempDir())

	if got, err := state.CurrentNodeVersion(); err != nil || got != "" {
		t.Fatalf("CurrentNodeVersion() = %q, %v", got, err)
	}
	if got, err := state.CurrentJavaVersion(); err != nil || got != "" {
		t.Fatalf("CurrentJavaVersion() = %q, %v", got, err)
	}
}

func TestStateWrapsUnexpectedReadlinkErrors(t *testing.T) {
	root := t.TempDir()
	state := New(fakeLinkReader{errs: map[string]error{
		filepath.Join(root, "node"): errors.New("permission denied"),
	}}, root)

	if _, err := state.CurrentNodeVersion(); err == nil {
		t.Fatal("CurrentNodeVersion() succeeded for reader failure")
	}
}

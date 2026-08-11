package state

import (
	"aem/pkg/errors"
	"path/filepath"
	"strings"
)

type LinkReader interface {
	Readlink(name string) (string, error)
}

type State struct {
	reader      LinkReader
	currentRoot string
}

func New(reader LinkReader, currentRoot string) *State {
	return &State{
		reader:      reader,
		currentRoot: currentRoot,
	}
}

func (s *State) CurrentNodeVersion() (string, error) {
	return s.CurrentVersionAt(filepath.Join(s.currentRoot, "node"), "node")
}

func (s *State) CurrentJavaVersion() (string, error) {
	return s.CurrentVersionAt(filepath.Join(s.currentRoot, "java"), "java")
}

// CurrentVersionAt returns the version selected by a runtime link, including
// links configured outside AEM_HOME through AEM_NODE_SYMLINK or AEM_JAVA_SYMLINK.
func (s *State) CurrentVersionAt(linkPath, module string) (string, error) {
	target, err := s.reader.Readlink(linkPath)
	if err != nil {
		if isNotExist(err) {
			return "", nil
		}
		return "", errors.NewFileSystemError("failed to read current "+module+" symlink", err)
	}

	version := filepath.Base(filepath.Clean(target))
	version = strings.TrimPrefix(version, "v")
	return version, nil
}

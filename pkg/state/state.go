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
	return s.currentVersion("node")
}

func (s *State) CurrentJavaVersion() (string, error) {
	return s.currentVersion("java")
}

func (s *State) CurrentAndroidPath() (string, error) {
	linkPath := filepath.Join(s.currentRoot, "android")
	target, err := s.reader.Readlink(linkPath)
	if err != nil {
		if isNotExist(err) {
			return "", nil
		}
		return "", errors.NewFileSystemError("failed to read android symlink", err)
	}
	return filepath.Clean(target), nil
}

func (s *State) currentVersion(module string) (string, error) {
	linkPath := filepath.Join(s.currentRoot, module)
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


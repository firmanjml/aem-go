package filesystem

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
	"os"
	"path/filepath"
	"runtime"
)

type FileSystem struct {
	logger *logger.Logger
}

func New(logger *logger.Logger) *FileSystem {
	return &FileSystem{logger: logger}
}

func (fs *FileSystem) EnsureDir(path string) error {
	fs.logger.Debug("Creating directory: %s", path)
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.NewFileSystemError("failed to create directory", err)
	}
	return nil
}

func (fs *FileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (fs *FileSystem) RemoveAll(path string) error {
	fs.logger.Debug("Removing directory: %s", path)
	if err := os.RemoveAll(path); err != nil {
		return errors.NewFileSystemError("failed to remove directory", err)
	}
	return nil
}

func (fs *FileSystem) Move(src, dst string) error {
	fs.logger.Debug("Moving %s to %s", src, dst)
	if err := os.Rename(src, dst); err != nil {
		return errors.NewFileSystemError("failed to move file/directory", err)
	}
	return nil
}

func (fs *FileSystem) CreateSymlink(link, target string) error {
	fs.logger.Debug("Creating symlink: %s -> %s", link, target)

	// Remove existing symlink if it exists
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return errors.NewFileSystemError("failed to remove existing symlink", err)
		}
	}

	// Get absolute path for target
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return errors.NewFileSystemError("failed to get absolute path", err)
	}

	// Create symlink
	if err := os.Symlink(absTarget, link); err != nil {
		if runtime.GOOS == "windows" {
			return errors.NewFileSystemError("failed to create symlink (may need administrator privileges on Windows)", err)
		}
		return errors.NewFileSystemError("failed to create symlink", err)
	}

	return nil
}

func (fs *FileSystem) ListDir(path string) ([]os.DirEntry, error) {
	fs.logger.Debug("Listing directory: %s", path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, errors.NewFileSystemError("failed to read directory", err)
	}
	return entries, nil
}

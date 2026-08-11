package filesystem

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
	"aem/pkg/state"
	"aem/pkg/version"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type FileSystem struct {
	logger     *logger.Logger
	versionMgr *version.Manager
}

func New(logger *logger.Logger) *FileSystem {
	aemHome := resolveAEMHome()
	var configPath string
	if aemHome != "" {
		configPath = filepath.Join(aemHome, "versions.json")
	}

	return &FileSystem{
		logger:     logger,
		versionMgr: version.NewManager(logger, configPath),
	}
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

	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		return errors.NewFileSystemError("failed to create symlink parent directory", err)
	}

	// Replace an existing link, but never remove a regular file or directory at
	// the configured link path. A custom AEM_*_SYMLINK path may point anywhere
	// on disk, so treating every existing path as replaceable would be unsafe.
	if info, err := os.Lstat(link); err == nil {
		if !isReplaceableLink(link, info) {
			return errors.NewFileSystemError("refusing to replace non-symlink path", nil)
		}
		if err := os.Remove(link); err != nil {
			return errors.NewFileSystemError("failed to remove existing symlink", err)
		}
	} else if !os.IsNotExist(err) {
		return errors.NewFileSystemError("failed to inspect existing symlink path", err)
	}

	// Get absolute path for target
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return errors.NewFileSystemError("failed to get absolute path", err)
	}

	// Create symlink
	if err := os.Symlink(absTarget, link); err != nil {
		if runtime.GOOS == "windows" {
			// Directory junctions do not require Developer Mode or an elevated
			// shell. They provide the same activation semantics for AEM's managed
			// runtime directories when Windows refuses a symbolic link.
			if junctionErr := createWindowsJunction(link, absTarget); junctionErr == nil {
				return nil
			} else {
				return errors.NewFileSystemError("failed to create symlink or directory junction on Windows", fmt.Errorf("symlink: %w; junction: %v", err, junctionErr))
			}
		}
		return errors.NewFileSystemError("failed to create symlink", err)
	}

	return nil
}

func createWindowsJunction(link, target string) error {
	command := exec.Command("cmd", "/C", "mklink", "/J", link, target)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("mklink /J failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (fs *FileSystem) extractModuleVersion(target string) (string, string) {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(target)
	pathParts := strings.Split(normalizedPath, "/")

	// Look for module and version in path
	for i, part := range pathParts {
		if (part == "node" || part == "java") && i+1 < len(pathParts) {
			module := part
			version := pathParts[i+1]
			// Clean version string (remove 'v' prefix if present)
			if strings.HasPrefix(version, "v") {
				version = strings.TrimPrefix(version, "v")
			}
			return module, version
		}
	}
	return "", ""
}

func (fs *FileSystem) updateVersionManager(module, version string) error {
	switch module {
	case "node":
		return fs.versionMgr.SetNodeVersion(version)
	case "java":
		return fs.versionMgr.SetJavaVersion(version)
	default:
		fs.logger.Debug("Unknown module type: %s", module)
		return nil
	}
}

func (fs *FileSystem) ListDir(path string) ([]os.DirEntry, error) {
	fs.logger.Debug("Listing directory: %s", path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, errors.NewFileSystemError("failed to read directory", err)
	}
	return entries, nil
}

// GetAEMHome returns the AEM_HOME directory, creating it if necessary
func (fs *FileSystem) GetAEMHome() (string, error) {
	aemHome := resolveAEMHome()
	if aemHome == "" {
		return "", errors.NewValidationError("unable to determine AEM home directory")
	}

	if err := fs.EnsureDir(aemHome); err != nil {
		return "", err
	}

	return aemHome, nil
}

// GetTempDir returns the temporary directory within AEM_HOME
func (fs *FileSystem) GetTempDir() (string, error) {
	aemHome, err := fs.GetAEMHome()
	if err != nil {
		return "", err
	}

	tmpDir := filepath.Join(aemHome, "tmp")
	if err := fs.EnsureDir(tmpDir); err != nil {
		return "", err
	}

	return tmpDir, nil
}

// GetInstallDir returns the installation directory within AEM_HOME
func (fs *FileSystem) GetInstallDir() (string, error) {
	aemHome, err := fs.GetAEMHome()
	if err != nil {
		return "", err
	}

	installDir := filepath.Join(aemHome, "sys_installed")
	if err := fs.EnsureDir(installDir); err != nil {
		return "", err
	}

	return installDir, nil
}

func (fs *FileSystem) GetCurrentRoot() (string, error) {
	aemHome, err := fs.GetAEMHome()
	if err != nil {
		return "", err
	}

	currentRoot := filepath.Join(aemHome, "current")
	if err := fs.EnsureDir(currentRoot); err != nil {
		return "", err
	}

	return currentRoot, nil
}

func (fs *FileSystem) GetState() (*state.State, error) {
	currentRoot, err := fs.GetCurrentRoot()
	if err != nil {
		return nil, err
	}

	return state.New(state.NewOSReader(), currentRoot), nil
}

// GetVersionManager returns the version manager instance
func (fs *FileSystem) GetVersionManager() *version.Manager {
	return fs.versionMgr
}

func resolveAEMHome() string {
	if value := strings.TrimSpace(os.Getenv("AEM_HOME")); value != "" {
		return value
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".aem")
}

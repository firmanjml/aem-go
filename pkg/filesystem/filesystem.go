package filesystem

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
	"aem/pkg/state"
	"aem/pkg/version"
	"os"
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

package java

import (
	"aem/internal/platform"
	"aem/pkg/archiver"
	"aem/pkg/downloader"
	"aem/pkg/errors"
	"aem/pkg/filesystem"
	"aem/pkg/logger"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Service struct {
	logger     *logger.Logger
	downloader *downloader.Downloader
	fs         *filesystem.FileSystem
	extractor  *archiver.ZipExtractor
	installDir string
}

type AzulPackage struct {
	DownloadURL string `json:"download_url"`
	JavaVersion []int  `json:"java_version"`
	Name        string `json:"name"`
}

func NewService(logger *logger.Logger, installDir string) *Service {
	return &Service{
		logger:     logger,
		downloader: downloader.New(logger),
		fs:         filesystem.New(logger),
		extractor:  archiver.NewZipExtractor(logger),
		installDir: installDir,
	}
}

func (s *Service) Install(majorVersion string) (string, error) {
	s.logger.Info("Installing JDK version: %s", majorVersion)

	// Check if already installed
	versionPath := filepath.Join(s.installDir, "java", "v"+majorVersion)
	if s.fs.Exists(versionPath) {
		s.logger.Info("JDK version %s already installed", majorVersion)
		return "", nil
	}

	// Get platform info
	platform := platform.GetInfo()

	// Fetch available packages
	packages, err := s.fetchPackages(majorVersion, platform)
	if err != nil {
		return "", err
	}

	if len(packages) == 0 {
		return "", errors.NewValidationError("no JDK packages found for version " + majorVersion)
	}

	pkg := packages[0]

	// Create version string
	versionStr := s.createVersionString(pkg.JavaVersion)
	finalPath := filepath.Join(s.installDir, "java", versionStr)

	// Download and install
	if err := s.downloadAndInstall(pkg, finalPath); err != nil {
		return "", err
	}

	s.logger.Info("Successfully installed JDK version: %s", versionStr)
	return versionStr, nil
}

func (s *Service) Use(version string, symlinkPath string) error {
	s.logger.Info("Setting JDK version: %s", version)

	versionPath := filepath.Join(s.installDir, "java", version)
	if !s.fs.Exists(versionPath) {
		return errors.NewValidationError("JDK version not installed: " + version)
	}

	if symlinkPath == "" {
		return errors.NewValidationError("symlink path not configured")
	}

	if err := s.fs.CreateSymlink(symlinkPath, versionPath); err != nil {
		return err
	}

	s.logger.Info("Successfully set JDK version: %s", version)
	return nil
}

func (s *Service) List() ([]string, error) {
	javaPath := filepath.Join(s.installDir, "java")

	if err := s.fs.EnsureDir(javaPath); err != nil {
		return nil, err
	}

	entries, err := s.fs.ListDir(javaPath)
	if err != nil {
		return nil, err
	}

	jdkVersion := ""
	data, err := os.ReadFile("java.txt")
	if err == nil {
		jdkVersion = strings.TrimSpace(string(data))
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			prefix := "   "
			if version == jdkVersion {
				prefix = "*  "
			}
			versions = append(versions, prefix+version)
		}
	}

	return versions, nil
}

func (s *Service) fetchPackages(javaVersion string, platform platform.Info) ([]AzulPackage, error) {
	apiURL := fmt.Sprintf(
		"https://api.azul.com/metadata/v1/zulu/packages/?java_version=%s&arch=%s&os=%s&archive_type=zip&java_package_type=jdk",
		javaVersion, platform.MapArchitecture(), platform.OS,
	)

	s.logger.Debug("Fetching JDK packages from: %s", apiURL)

	resp, err := s.downloader.GetHTML(apiURL)
	if err != nil {
		return nil, errors.NewAPIError("failed to fetch JDK packages", err)
	}
	defer resp.Close()

	var packages []AzulPackage
	if err := json.NewDecoder(resp).Decode(&packages); err != nil {
		return nil, errors.NewAPIError("failed to parse API response", err)
	}

	return packages, nil
}

func (s *Service) downloadAndInstall(pkg AzulPackage, finalPath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return errors.NewExtractionError("failed to get executable path: %w", err)
	}

	baseDir := filepath.Dir(execPath)
	tmpDir := filepath.Join(baseDir, "tmp")

	zipPath := filepath.Join(tmpDir, pkg.Name)
	extractDir := filepath.Join(tmpDir, "jdk_extract")

	// Ensure cleanup
	defer func() {
		s.fs.RemoveAll(zipPath)
		s.fs.RemoveAll(extractDir)
	}()

	// Download
	if err := s.downloader.Download(pkg.DownloadURL, zipPath); err != nil {
		return err
	}

	// Extract
	if err := s.extractor.Extract(zipPath, extractDir); err != nil {
		return err
	}

	// Find extracted root directory
	entries, err := s.fs.ListDir(extractDir)
	if err != nil {
		return err
	}

	if len(entries) != 1 || !entries[0].IsDir() {
		return errors.NewExtractionError("expected single root directory in JDK archive", nil)
	}

	extractedRoot := filepath.Join(extractDir, entries[0].Name())

	// Ensure destination directory exists
	if err := s.fs.EnsureDir(filepath.Dir(finalPath)); err != nil {
		return err
	}

	// Move to final location
	s.fs.RemoveAll(finalPath) // Remove if exists
	return s.fs.Move(extractedRoot, finalPath)
}

func (s *Service) createVersionString(javaVersion []int) string {
	parts := make([]string, len(javaVersion))
	for i, v := range javaVersion {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ".")
}

func (s *Service) GetCurrentJDKVersion() (string, error) {
	s.logger.Debug("Fetching JDK current environment versions")

	data, err := os.ReadFile("java.txt")
	if err != nil {
		if os.IsNotExist(err) {
			return "no current version", nil
		}
		return "", errors.NewFileSystemError("failed to read JDK setting", err)
	}

	return string(data), nil
}

func (s *Service) Uninstall(majorVersion string) error {
	s.logger.Info("Un-installing JDK version: %s", majorVersion)

	// Check if the environment is being set
	currentVersion, err := s.GetCurrentJDKVersion()
	if err != nil {
		return err
	}

	if currentVersion == majorVersion {
		return errors.UninstallError(fmt.Sprintf("cannot uninstall version %s as it's the currently active version", majorVersion), nil)
	}

	// Check if already installed
	versionPath := filepath.Join(s.installDir, "java", majorVersion)

	if !s.fs.Exists(versionPath) {
		s.logger.Info("JDK version %s not found at %s", majorVersion, versionPath)
		return nil
	}

	// Remove version
	s.logger.Info("Removing JDK version %s from %s", majorVersion, versionPath)
	if err := s.fs.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove JDK version %s: %w", majorVersion, err)
	}

	s.logger.Info("Successfully removed JDK version %s", majorVersion)
	return nil
}

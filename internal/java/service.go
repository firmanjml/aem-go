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
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
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
	s.logger.Debug("Installing JDK version: %s", majorVersion)

	// Check if already installed
	versionPath := filepath.Join(s.installDir, "java", "v"+majorVersion)
	if s.fs.Exists(versionPath) {
		s.logger.Debug("JDK version %s already installed", majorVersion)
		return "v" + majorVersion, nil
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

	s.logger.Debug("Successfully installed JDK version: %s", versionStr)
	return versionStr, nil
}

func (s *Service) Use(version string, symlinkPath string) error {
	s.logger.Debug("Setting JDK version: %s", version)

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

	s.logger.Debug("Successfully set JDK version: %s", version)
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

	state, err := s.fs.GetState()
	if err != nil {
		s.logger.Error("Failed to create state reader for JDK: %v", err)
	}

	currentVersion := ""
	if state != nil {
		currentVersion, err = state.CurrentJavaVersion()
		if err != nil {
			s.logger.Error("Failed to get current JDK version: %v", err)
			currentVersion = ""
		}
	}

	var installed []string
	for _, entry := range entries {
		if entry.IsDir() {
			installed = append(installed, entry.Name())
		}
	}

	sort.Slice(installed, func(i, j int) bool {
		left := installed[i]
		right := installed[j]
		if semver.IsValid(left) && semver.IsValid(right) {
			return semver.Compare(left, right) < 0
		}
		return left < right
	})

	var versions []string
	for _, version := range installed {
		cleanVersion := strings.TrimPrefix(version, "v")
		prefix := "   "
		if cleanVersion == currentVersion || version == currentVersion {
			prefix = "*  "
		}
		versions = append(versions, prefix+version)
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
	// Get temp directory from AEM_HOME
	tmpDir, err := s.fs.GetTempDir()
	if err != nil {
		return err
	}

	zipPath := filepath.Join(tmpDir, pkg.Name)
	extractDir := filepath.Join(tmpDir, "jdk_extract")

	// Ensure cleanup
	defer func() {
		s.fs.RemoveAll(zipPath)
		s.fs.RemoveAll(extractDir)
	}()

	s.fs.RemoveAll(zipPath)
	s.fs.RemoveAll(extractDir)

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
	return "v" + strings.Join(parts, ".")
}

func (s *Service) GetCurrentJDKVersion() (string, error) {
	state, err := s.fs.GetState()
	if err != nil {
		return "", err
	}
	return state.CurrentJavaVersion()
}

func (s *Service) Uninstall(majorVersion string) error {
	s.logger.Debug("Un-installing JDK version: %s", majorVersion)

	// Check if the environment is being set
	currentVersion, err := s.GetCurrentJDKVersion()
	if err != nil {
		return err
	}

	if currentVersion == majorVersion || currentVersion == "v"+majorVersion {
		return errors.UninstallError(fmt.Sprintf("cannot uninstall version %s as it's the currently active version", majorVersion), nil)
	}

	// Check if already installed
	versionPath := filepath.Join(s.installDir, "java", majorVersion)
	if !s.fs.Exists(versionPath) {
		vVersionPath := filepath.Join(s.installDir, "java", "v"+majorVersion)
		if s.fs.Exists(vVersionPath) {
			versionPath = vVersionPath
		} else {
			s.logger.Debug("JDK version %s not found", majorVersion)
			return nil
		}
	}

	// Remove version
	s.logger.Debug("Removing JDK version %s from %s", majorVersion, versionPath)
	if err := s.fs.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove JDK version %s: %w", majorVersion, err)
	}

	s.logger.Debug("Successfully removed JDK version %s", majorVersion)
	return nil
}

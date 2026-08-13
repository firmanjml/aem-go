package java

import (
	"aem/internal/platform"
	"aem/pkg/archiver"
	"aem/pkg/downloader"
	"aem/pkg/errors"
	"aem/pkg/filesystem"
	"aem/pkg/logger"
	"aem/pkg/state"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
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
	tarGz      *archiver.TarGzExtractor
	installDir string
	apiURL     string
	platform   platform.Info
}

type AzulPackage struct {
	DownloadURL string `json:"download_url"`
	JavaVersion []int  `json:"java_version"`
	Name        string `json:"name"`
}

func NewService(logger *logger.Logger, installDir string) *Service {
	return newService(logger, installDir, "https://api.azul.com/metadata/v1/zulu/packages/", platform.GetInfo())
}

// newService permits package-level integration tests to use a local provider
// endpoint and explicit cross-platform target without real installations.
func newService(log *logger.Logger, installDir, apiURL string, target platform.Info) *Service {
	return &Service{
		logger:     log,
		downloader: downloader.New(log),
		fs:         filesystem.New(log),
		extractor:  archiver.NewZipExtractor(log),
		tarGz:      archiver.NewTarGzExtractor(log),
		installDir: installDir,
		apiURL:     apiURL,
		platform:   target,
	}
}

func (s *Service) Install(majorVersion string) (string, error) {
	majorVersion = normalizeRequestedVersion(majorVersion)
	if majorVersion == "" {
		return "", errors.NewValidationError("JDK version is required")
	}
	s.logger.Debug("Installing JDK version: %s", majorVersion)

	installedVersion, err := s.findInstalledVersion(majorVersion)
	if err != nil {
		return "", err
	}
	if installedVersion != "" {
		s.logger.Debug("JDK version %s already installed", installedVersion)
		return installedVersion, nil
	}

	// Fetch available packages
	packages, err := s.fetchPackages(majorVersion, s.platform)
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

// findInstalledVersion treats a requested major or minor version as a stable
// constraint. Once a matching JDK is installed, setup reuses it instead of
// silently downloading a newer patch release on every invocation.
func (s *Service) findInstalledVersion(requested string) (string, error) {
	javaDir := filepath.Join(s.installDir, "java")
	if !s.fs.Exists(javaDir) {
		return "", nil
	}
	entries, err := s.fs.ListDir(javaDir)
	if err != nil {
		return "", err
	}
	prefix := "v" + requested
	candidates := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		if version == prefix || strings.HasPrefix(version, prefix+".") {
			candidates = append(candidates, version)
		}
	}
	if len(candidates) == 0 {
		return "", nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		return semver.Compare(candidates[i], candidates[j]) > 0
	})
	return candidates[0], nil
}

func (s *Service) Use(version string, symlinkPath string) error {
	version = normalizeInstalledVersion(version)
	if version == "v" {
		return errors.NewValidationError("JDK version is required")
	}
	s.logger.Debug("Setting JDK version: %s", version)

	versionPath := filepath.Join(s.installDir, "java", version)
	if !s.fs.Exists(versionPath) {
		return errors.NewValidationError("JDK version not installed: " + version)
	}

	if symlinkPath == "" {
		return errors.NewValidationError("symlink path not configured")
	}

	if err := s.ensureMacOSJavaHome(versionPath); err != nil {
		return err
	}

	if err := s.fs.CreateSymlink(symlinkPath, versionPath); err != nil {
		return err
	}

	s.logger.Debug("Successfully set JDK version: %s", version)
	return nil
}

// ensureMacOSJavaHome normalizes Azul archives that wrap the JDK bundle in a
// compatibility directory. Some Zulu 8 packages expose bin/java at the
// archive root through symlinks while keeping the real home below
// <name>.jdk/Contents/Home. AEM's shell integration uses Contents/Home as the
// stable macOS JAVA_HOME, so make that path available before activating the
// runtime.
func (s *Service) ensureMacOSJavaHome(versionPath string) error {
	if s.platform.OS != "darwin" {
		return nil
	}

	javaHome := filepath.Join(versionPath, "Contents", "Home")
	if isJavaHome(javaHome) {
		return nil
	}

	entries, err := s.fs.ListDir(versionPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jdk") {
			continue
		}
		bundleHome := filepath.Join(versionPath, entry.Name(), "Contents", "Home")
		if !isJavaHome(bundleHome) {
			continue
		}
		if err := s.fs.CreateSymlink(javaHome, bundleHome); err != nil {
			return fmt.Errorf("failed to normalize macOS JDK home: %w", err)
		}
		return nil
	}

	return errors.NewValidationError("installed macOS JDK does not contain Contents/Home")
}

func isJavaHome(path string) bool {
	info, err := os.Stat(filepath.Join(path, "bin", "java"))
	return err == nil && !info.IsDir()
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

func (s *Service) fetchPackages(javaVersion string, target platform.Info) ([]AzulPackage, error) {
	osName, arch, err := target.AzulTarget()
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(s.apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Azul metadata URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("java_version", javaVersion)
	query.Set("arch", arch)
	query.Set("os", osName)
	query.Set("archive_type", "zip")
	query.Set("java_package_type", "jdk")
	query.Set("availability_types", "CA")
	query.Set("release_status", "ga")
	query.Set("javafx_bundled", "false")
	endpoint.RawQuery = query.Encode()
	apiURL := endpoint.String()

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

	// Azul responds with a direct package URL. Extract the actual archive type
	// instead of assuming the Windows ZIP convention on every host.
	archiveName := strings.ToLower(pkg.Name)
	if archiveName == "" {
		archiveName = strings.ToLower(pkg.DownloadURL)
	}
	switch {
	case strings.HasSuffix(archiveName, ".zip"):
		if err := s.extractor.Extract(zipPath, extractDir); err != nil {
			return err
		}
	case strings.HasSuffix(archiveName, ".tar.gz"), strings.HasSuffix(archiveName, ".tgz"):
		if err := s.tarGz.Extract(zipPath, extractDir); err != nil {
			return err
		}
	default:
		return errors.NewExtractionError("unsupported Zulu JDK archive format", nil)
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
	if symlinkPath := strings.TrimSpace(os.Getenv("AEM_JAVA_SYMLINK")); symlinkPath != "" {
		return state.New(state.NewOSReader(), "").CurrentVersionAt(symlinkPath, "java")
	}
	state, err := s.fs.GetState()
	if err != nil {
		return "", err
	}
	return state.CurrentJavaVersion()
}

func (s *Service) Uninstall(majorVersion string) error {
	majorVersion = normalizeRequestedVersion(majorVersion)
	if majorVersion == "" {
		return errors.NewValidationError("JDK version is required")
	}
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
	versionPath := filepath.Join(s.installDir, "java", normalizeInstalledVersion(majorVersion))
	if !s.fs.Exists(versionPath) {
		s.logger.Debug("JDK version %s not found", majorVersion)
		return nil
	}

	// Remove version
	s.logger.Debug("Removing JDK version %s from %s", majorVersion, versionPath)
	if err := s.fs.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove JDK version %s: %w", majorVersion, err)
	}

	s.logger.Debug("Successfully removed JDK version %s", majorVersion)
	return nil
}

func normalizeRequestedVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func normalizeInstalledVersion(version string) string {
	version = normalizeRequestedVersion(version)
	if version == "" {
		return "v"
	}
	return "v" + version
}

package node

import (
	"aem/internal/platform"
	"aem/pkg/archiver"
	"aem/pkg/downloader"
	"aem/pkg/errors"
	"aem/pkg/filesystem"
	"aem/pkg/logger"
	"aem/pkg/state"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
	"golang.org/x/net/html"
)

type Service struct {
	logger     *logger.Logger
	downloader *downloader.Downloader
	fs         *filesystem.FileSystem
	zipper     *archiver.ZipExtractor
	tarGz      *archiver.TarGzExtractor
	installDir string
	distURL    string
	platform   platform.Info
}

func NewService(logger *logger.Logger, installDir string) *Service {
	return newService(logger, installDir, "https://nodejs.org/dist", platform.GetInfo())
}

// newService permits package-level integration tests to supply a local
// distribution server and an explicit target without changing production API.
func newService(log *logger.Logger, installDir, distURL string, target platform.Info) *Service {
	return &Service{
		logger:     log,
		downloader: downloader.New(log),
		fs:         filesystem.New(log),
		zipper:     archiver.NewZipExtractor(log),
		tarGz:      archiver.NewTarGzExtractor(log),
		installDir: installDir,
		distURL:    strings.TrimSuffix(distURL, "/"),
		platform:   target,
	}
}

func (s *Service) Install(majorVersion string) (string, error) {
	s.logger.Debug("Installing Node.js version: %s", majorVersion)

	// Normalize version format
	if !strings.HasPrefix(majorVersion, "v") {
		majorVersion = "v" + majorVersion
	}

	// Get available versions
	versions, err := s.GetVersions()
	if err != nil {
		return "", err
	}

	// Find matching versions
	var matched []string
	for _, v := range versions {
		if strings.HasPrefix(v, majorVersion) {
			matched = append(matched, v)
		}
	}

	if len(matched) == 0 {
		return "", errors.NewValidationError("no Node.js versions found for major version: " + majorVersion)
	}

	// Use latest version
	latest := matched[len(matched)-1]
	s.logger.Debug("Installing latest version: %s", latest)

	// Check if already installed
	versionPath := filepath.Join(s.installDir, "node", latest)
	if s.fs.Exists(versionPath) {
		s.logger.Debug("Node.js version %s already installed", latest)
		return strings.TrimPrefix(latest, "v"), nil
	}

	// Download and install
	downloadURL, err := s.getDownloadURL(latest)
	if err != nil {
		return "", err
	}

	if err := s.downloadAndInstall(downloadURL, strings.TrimPrefix(latest, "v")); err != nil {
		return "", err
	}

	s.logger.Debug("Successfully installed Node.js version: %s", latest)
	return strings.TrimPrefix(latest, "v"), nil
}

func (s *Service) Use(version string, symlinkPath string) error {
	s.logger.Debug("Setting Node.js version: %s", version)

	// Handle both with and without 'v' prefix
	versionPath := filepath.Join(s.installDir, "node", version)
	if !s.fs.Exists(versionPath) {
		// Try with 'v' prefix
		vVersionPath := filepath.Join(s.installDir, "node", "v"+version)
		if s.fs.Exists(vVersionPath) {
			versionPath = vVersionPath
			version = "v" + version
		} else {
			return errors.NewValidationError("Node.js version not installed: " + version)
		}
	}

	if symlinkPath == "" {
		return errors.NewValidationError("symlink path not configured")
	}

	if err := s.fs.CreateSymlink(symlinkPath, versionPath); err != nil {
		return err
	}

	s.logger.Debug("Successfully set Node.js version: %s", version)
	return nil
}

func (s *Service) List() ([]string, error) {
	nodePath := filepath.Join(s.installDir, "node")
	if err := s.fs.EnsureDir(nodePath); err != nil {
		return nil, err
	}

	entries, err := s.fs.ListDir(nodePath)
	if err != nil {
		return nil, err
	}

	state, err := s.fs.GetState()
	if err != nil {
		s.logger.Error("Failed to create state reader for Node.js: %v", err)
	}

	currentVersion := ""
	if state != nil {
		currentVersion, err = state.CurrentNodeVersion()
		if err != nil {
			s.logger.Error("Failed to get current Node.js version: %v", err)
			currentVersion = ""
		}
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			version := entry.Name()
			cleanVersion := strings.TrimPrefix(version, "v")
			prefix := "   "
			if cleanVersion == currentVersion || version == currentVersion {
				prefix = "*  "
			}
			versions = append(versions, prefix+version)
		}
	}

	return versions, nil
}

func (s *Service) GetVersions() ([]string, error) {
	s.logger.Debug("Fetching Node.js versions")

	resp, err := s.downloader.GetHTML(s.distURL + "/")
	if err != nil {
		return nil, errors.NewAPIError("failed to fetch Node.js versions", err)
	}
	defer resp.Close()

	doc, err := html.Parse(resp)
	if err != nil {
		return nil, errors.NewAPIError("failed to parse HTML response", err)
	}

	var versions []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" &&
					strings.HasPrefix(attr.Val, "v") &&
					strings.HasSuffix(attr.Val, "/") &&
					semver.IsValid(attr.Val[:len(attr.Val)-1]) {
					version := strings.TrimSuffix(attr.Val, "/")
					versions = append(versions, version)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) < 0
	})

	return versions, nil
}

func (s *Service) getDownloadURL(version string) (string, error) {
	target, err := s.platform.NodeTarget()
	if err != nil {
		return "", err
	}
	archiveSuffix := ".tar.gz"
	if s.platform.OS == "windows" {
		archiveSuffix = ".zip"
	}

	pageURL := s.distURL + "/" + version
	s.logger.Debug("Searching for Node.js binary at: %s", pageURL)

	resp, err := s.downloader.GetHTML(pageURL)
	if err != nil {
		return "", errors.NewAPIError("failed to fetch Node.js download page", err)
	}
	defer resp.Close()

	doc, err := html.Parse(resp)
	if err != nil {
		return "", errors.NewAPIError("failed to parse download page", err)
	}

	var downloadURL string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" &&
					strings.Contains(attr.Val, target) &&
					strings.HasSuffix(attr.Val, archiveSuffix) {
					downloadURL = resolveDownloadURL(pageURL, attr.Val)
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if downloadURL == "" {
		return "", errors.NewValidationError("no suitable Node.js binary found for " + target)
	}

	return downloadURL, nil
}

func resolveDownloadURL(pageURL, reference string) string {
	page, err := url.Parse(pageURL + "/")
	if err != nil {
		return reference
	}
	ref, err := url.Parse(reference)
	if err != nil {
		return reference
	}
	return page.ResolveReference(ref).String()
}

func (s *Service) downloadAndInstall(url, version string) error {
	// Get temp directory from AEM_HOME
	tmpDir, err := s.fs.GetTempDir()
	if err != nil {
		return err
	}

	fileName := filepath.Base(url)
	zipPath := filepath.Join(tmpDir, fileName)
	extractDir := filepath.Join(tmpDir, "node_extract")
	finalPath := filepath.Join(s.installDir, "node", "v"+version)

	// Ensure cleanup
	defer func() {
		s.fs.RemoveAll(zipPath)
		s.fs.RemoveAll(extractDir)
	}()

	s.fs.RemoveAll(zipPath)
	s.fs.RemoveAll(extractDir)

	// Download
	if err := s.downloader.Download(url, zipPath); err != nil {
		return err
	}

	// Extract
	switch {
	case strings.HasSuffix(zipPath, ".zip"):
		if err := s.zipper.Extract(zipPath, extractDir); err != nil {
			return err
		}
	case strings.HasSuffix(zipPath, ".tar.gz"):
		if err := s.tarGz.Extract(zipPath, extractDir); err != nil {
			return err
		}
	default:
		return errors.NewExtractionError("unsupported Node.js archive format", nil)
	}

	// Find extracted root directory
	entries, err := s.fs.ListDir(extractDir)
	if err != nil {
		return err
	}

	if len(entries) != 1 || !entries[0].IsDir() {
		return errors.NewExtractionError("expected single root directory in Node.js archive", nil)
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

func (s *Service) GetCurrentNodeVersion() (string, error) {
	if symlinkPath := strings.TrimSpace(os.Getenv("AEM_NODE_SYMLINK")); symlinkPath != "" {
		return state.New(state.NewOSReader(), "").CurrentVersionAt(symlinkPath, "node")
	}
	state, err := s.fs.GetState()
	if err != nil {
		return "", err
	}
	return state.CurrentNodeVersion()
}

func (s *Service) Uninstall(majorVersion string) error {
	s.logger.Debug("Un-installing Node version: %s", majorVersion)

	// Check if the environment is being set
	currentVersion, err := s.GetCurrentNodeVersion()
	if err != nil {
		return err
	}

	if currentVersion == majorVersion || currentVersion == "v"+majorVersion {
		return errors.UninstallError(fmt.Sprintf("cannot uninstall version %s as it's the currently active version", majorVersion), nil)
	}

	// Try both with and without 'v' prefix
	versionPath := filepath.Join(s.installDir, "node", majorVersion)
	if !s.fs.Exists(versionPath) {
		vVersionPath := filepath.Join(s.installDir, "node", "v"+majorVersion)
		if s.fs.Exists(vVersionPath) {
			versionPath = vVersionPath
		} else {
			s.logger.Debug("Node version %s not found", majorVersion)
			return nil
		}
	}

	// Remove version
	s.logger.Debug("Removing Node version %s from %s", majorVersion, versionPath)
	if err := s.fs.RemoveAll(versionPath); err != nil {
		return fmt.Errorf("failed to remove Node version %s: %w", majorVersion, err)
	}

	s.logger.Debug("Successfully removed Node version %s", majorVersion)
	return nil
}

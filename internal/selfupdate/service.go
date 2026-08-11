// Package selfupdate finds release binaries on source control and replaces the
// running aem executable with a verified release archive.
package selfupdate

import (
	"aem/pkg/archiver"
	"aem/pkg/downloader"
	"aem/pkg/errors"
	"aem/pkg/logger"
	"aem/pkg/process"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

// DefaultRepository is the source-control project whose tagged releases
// provide the aem binaries that `aem update` installs.
const DefaultRepository = "Adaptive-Cloud/aem-go"

const defaultAPIBaseURL = "https://api.github.com"

// Release describes one tagged release and its downloadable assets.
type Release struct {
	Tag     string            // e.g. v1.2.3
	Version string            // Tag without the leading v, e.g. 1.2.3
	Assets  map[string]string // asset name to its download URL
}

// CheckResult reports how the current build compares to a resolved release.
type CheckResult struct {
	Release         *Release
	Current         string
	Target          string
	UpdateAvailable bool
	UpToDate        bool
	Downgrade       bool
	CurrentIsDev    bool
}

// UpdateResult describes the outcome of self-updating the executable.
type UpdateResult struct {
	From             string
	To               string
	Path             string
	AlreadyInstalled bool
}

// Service resolves releases, verifies their checksums, and swaps the running
// executable. The API base URL and repository can be overridden so tests can
// work against local servers instead of the real source-control host.
type Service struct {
	logger     *logger.Logger
	downloader *downloader.Downloader
	apiBaseURL string
	repository string
	tempDir    string
	client     *http.Client
}

type Option func(*Service)

// WithAPIBaseURL overrides the release API endpoint (used by tests).
func WithAPIBaseURL(url string) Option {
	return func(s *Service) {
		s.apiBaseURL = strings.TrimSuffix(url, "/")
	}
}

// WithRepository overrides the "owner/name" repository releases are read from.
func WithRepository(repository string) Option {
	return func(s *Service) {
		s.repository = repository
	}
}

// NewService creates a self-update service that stages downloads in tempDir.
func NewService(log *logger.Logger, tempDir string, opts ...Option) *Service {
	service := &Service{
		logger:     log,
		downloader: downloader.New(log),
		apiBaseURL: defaultAPIBaseURL,
		repository: DefaultRepository,
		tempDir:    tempDir,
		client:     &http.Client{},
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

// IsReleaseVersion reports whether a bundled version string describes a real
// tagged release rather than a development build such as "dev".
func IsReleaseVersion(version string) bool {
	return semver.IsValid(normalizeVersion(version))
}

// Check compares the installed version with the latest release, or with a
// pinned version when targetVersion is not empty.
func (s *Service) Check(current, targetVersion string) (*CheckResult, error) {
	release, err := s.fetchRelease(targetVersion)
	if err != nil {
		return nil, err
	}

	result := &CheckResult{
		Release: release,
		Current: current,
		Target:  release.Tag,
	}

	if !IsReleaseVersion(current) {
		result.CurrentIsDev = true
		result.UpdateAvailable = true
		return result, nil
	}

	switch semver.Compare(normalizeVersion(current), normalizeTag(release.Tag)) {
	case -1:
		result.UpdateAvailable = true
	case 0:
		result.UpToDate = true
	case 1:
		result.Downgrade = true
	}
	return result, nil
}

// Update downloads the resolved release, verifies and extracts it, and swaps
// the executable at exePath. The calling command supplies the executable path
// so tests can update a stand-in binary in a temporary directory.
func (s *Service) Update(current, targetVersion, exePath string, force bool) (*UpdateResult, error) {
	check, err := s.Check(current, targetVersion)
	if err != nil {
		return nil, err
	}

	switch {
	case check.CurrentIsDev && !force:
		return nil, errors.NewValidationError(
			"this aem build has no release version bundled (dev build); rerun with --force (and optionally --version X.Y.Z) to replace it with a release binary")
	case check.Downgrade && !force:
		return nil, errors.NewValidationError(
			fmt.Sprintf("release %s is not newer than the installed version %s; rerun with --force to downgrade", check.Target, check.Current))
	case check.UpToDate && !force:
		return &UpdateResult{From: current, To: check.Target, Path: exePath, AlreadyInstalled: true}, nil
	}

	if err := s.Apply(check.Release, exePath); err != nil {
		return nil, err
	}
	return &UpdateResult{From: current, To: check.Target, Path: exePath}, nil
}

// Apply downloads, verifies, and installs the given release over exePath.
func (s *Service) Apply(release *Release, exePath string) error {
	assetName := archiveAssetName(release.Version)
	archiveURL, ok := release.Assets[assetName]
	if !ok {
		return errors.NewDownloadError(
			fmt.Sprintf("release %s does not contain the required archive %s", release.Tag, assetName), nil)
	}
	checksumsURL, ok := release.Assets["checksums.txt"]
	if !ok {
		return errors.NewDownloadError(
			fmt.Sprintf("release %s does not contain checksums.txt", release.Tag), nil)
	}

	staging := filepath.Join(s.tempDir, "selfupdate", release.Version)
	if err := os.RemoveAll(staging); err != nil {
		return errors.NewFileSystemError("failed to clear the update staging directory", err)
	}
	if err := os.MkdirAll(staging, 0755); err != nil {
		return errors.NewFileSystemError("failed to create the update staging directory", err)
	}
	defer os.RemoveAll(staging)

	archivePath := filepath.Join(staging, assetName)
	checksumsPath := filepath.Join(staging, "checksums.txt")

	s.logger.Info("Downloading aem %s for %s/%s...", release.Tag, runtime.GOOS, runtime.GOARCH)
	if err := s.downloader.Download(archiveURL, archivePath); err != nil {
		return err
	}
	if err := s.downloader.Download(checksumsURL, checksumsPath); err != nil {
		return err
	}
	if err := verifyChecksum(archivePath, checksumsPath, assetName); err != nil {
		return err
	}

	unpacked := filepath.Join(staging, "unpacked")
	var extractorErr error
	if runtime.GOOS == "windows" {
		extractorErr = archiver.NewZipExtractor(s.logger).Extract(archivePath, unpacked)
	} else {
		extractorErr = archiver.NewTarGzExtractor(s.logger).Extract(archivePath, unpacked)
	}
	if extractorErr != nil {
		return extractorErr
	}

	newBinary, err := findBinary(unpacked)
	if err != nil {
		return err
	}

	if err := replaceExecutable(exePath, newBinary); err != nil {
		return err
	}

	s.logger.Info("Installed aem %s to %s", release.Tag, exePath)
	return nil
}

func (s *Service) fetchRelease(targetVersion string) (*Release, error) {
	endpoint := s.apiBaseURL + "/repos/" + s.repository + "/releases/latest"
	description := "latest"
	if targetVersion != "" {
		endpoint = s.apiBaseURL + "/repos/" + s.repository + "/releases/tags/" + normalizeTag(targetVersion)
		description = targetVersion
	}

	s.logger.Debug("Fetching release information from: %s", endpoint)

	req, err := http.NewRequestWithContext(process.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.NewAPIError("failed to create the release request", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "aem")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.NewAPIError("failed to fetch release information", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.NewAPIError(
			fmt.Sprintf("release request for %s failed with status: %s", description, resp.Status), nil)
	}

	var payload struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, errors.NewAPIError("failed to parse the release response", err)
	}
	if payload.TagName == "" {
		return nil, errors.NewAPIError("the release response did not include a tag name", nil)
	}

	release := &Release{
		Tag:     payload.TagName,
		Version: strings.TrimPrefix(payload.TagName, "v"),
		Assets:  make(map[string]string, len(payload.Assets)),
	}
	for _, asset := range payload.Assets {
		release.Assets[asset.Name] = asset.BrowserDownloadURL
	}
	return release, nil
}

// verifyChecksum confirms the downloaded archive matches the sha256 published
// in the release's checksums.txt next to the archive file name.
func verifyChecksum(archivePath, checksumsPath, assetName string) error {
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return errors.NewDownloadError("failed to read checksums.txt", err)
	}

	expected := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if strings.TrimPrefix(fields[1], "*") == assetName {
			expected = fields[0]
			break
		}
	}
	if expected == "" {
		return errors.NewDownloadError("no checksum found for "+assetName, nil)
	}

	archive, err := os.Open(archivePath)
	if err != nil {
		return errors.NewDownloadError("failed to open the downloaded archive for verification", err)
	}
	defer archive.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, archive); err != nil {
		return errors.NewDownloadError("failed to hash the downloaded archive", err)
	}

	if actual := hex.EncodeToString(hash.Sum(nil)); !strings.EqualFold(actual, expected) {
		return errors.NewDownloadError(
			fmt.Sprintf("checksum verification failed for %s (expected %s, got %s)", assetName, expected, actual), nil)
	}
	return nil
}

// findBinary locates the aem executable inside an extracted release archive.
func findBinary(root string) (string, error) {
	name := binaryName()
	candidate := filepath.Join(root, name)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate, nil
	}

	found := ""
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == name {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil {
		return "", errors.NewExtractionError("failed to search the release archive for the aem executable", walkErr)
	}
	if found == "" {
		return "", errors.NewExtractionError("the release archive did not contain "+name, nil)
	}
	return found, nil
}

// replaceExecutable atomically (as far as the platform allows) swaps the
// executable at targetPath for newBinaryPath. The current binary is first
// moved aside because Windows will not overwrite a running executable, and it
// is restored if installing the new binary fails.
func replaceExecutable(targetPath, newBinaryPath string) error {
	resolved, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return errors.NewFileSystemError("failed to resolve the executable path", err)
	}

	info, err := os.Stat(newBinaryPath)
	if err != nil {
		return errors.NewFileSystemError("failed to inspect the updated executable", err)
	}

	staged := resolved + ".update-new"
	backup := resolved + ".update-old"

	if err := copyFile(newBinaryPath, staged, info.Mode()); err != nil {
		return err
	}
	defer os.Remove(staged)

	_ = os.Remove(backup) // clear any leftover from a previous update

	if err := os.Rename(resolved, backup); err != nil {
		return errors.NewFileSystemError(
			"failed to move the current executable aside (check write permissions on its directory)", err)
	}
	if err := os.Rename(staged, resolved); err != nil {
		if rollbackErr := os.Rename(backup, resolved); rollbackErr != nil {
			return errors.NewFileSystemError(
				"failed to install the updated executable and failed to restore the previous one", rollbackErr)
		}
		return errors.NewFileSystemError(
			"failed to install the updated executable; the previous version was restored", err)
	}

	_ = os.Remove(backup) // best effort; Windows may keep it until the process exits
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return errors.NewFileSystemError("failed to open the updated executable", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode|0755)
	if err != nil {
		return errors.NewFileSystemError("failed to stage the updated executable next to the current one", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return errors.NewFileSystemError("failed to write the staged executable", err)
	}
	if err := out.Close(); err != nil {
		return errors.NewFileSystemError("failed to close the staged executable", err)
	}
	return nil
}

func archiveAssetName(version string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("aem_%s_%s_%s.zip", version, runtime.GOOS, runtime.GOARCH)
	}
	return fmt.Sprintf("aem_%s_%s_%s.tar.gz", version, runtime.GOOS, runtime.GOARCH)
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "aem.exe"
	}
	return "aem"
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return version
}

func normalizeTag(tag string) string {
	return normalizeVersion(tag)
}

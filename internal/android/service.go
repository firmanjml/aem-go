package android

import (
	"aem/internal/config"
	"aem/internal/platform"
	"aem/pkg/archiver"
	"aem/pkg/downloader"
	"aem/pkg/errors"
	"aem/pkg/filesystem"
	"aem/pkg/logger"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const androidRepositoryURL = "https://dl.google.com/android/repository/repository2-1.xml"

type Service struct {
	logger     *logger.Logger
	downloader *downloader.Downloader
	fs         *filesystem.FileSystem
	zipper     *archiver.ZipExtractor
	installDir string
}

type repositoryXML struct {
	Packages []remotePackage `xml:"remotePackage"`
}

type remotePackage struct {
	Path         string           `xml:"path,attr"`
	Archives     archiveContainer `xml:"archives"`
	Revision     revision         `xml:"revision"`
	DisplayName  string           `xml:"display-name"`
	ChannelRef   channelRef       `xml:"channelRef"`
	UsesLicense  usesLicense      `xml:"uses-license"`
	BaseRevision baseRevision     `xml:"base-revision"`
}

type archiveContainer struct {
	Archive []remoteArchive `xml:"archive"`
}

type remoteArchive struct {
	HostOS struct {
		Value string `xml:",chardata"`
	} `xml:"host-os"`
	Complete struct {
		URL string `xml:"url"`
	} `xml:"complete"`
}

type revision struct {
	Major int `xml:"major"`
	Minor int `xml:"minor"`
	Micro int `xml:"micro"`
}

type channelRef struct {
	ID string `xml:"ref,attr"`
}

type usesLicense struct {
	Ref string `xml:"ref,attr"`
}

type baseRevision struct {
	Major int `xml:"major"`
}

func NewService(logger *logger.Logger, installDir string) *Service {
	return &Service{
		logger:     logger,
		downloader: downloader.New(logger),
		fs:         filesystem.New(logger),
		zipper:     archiver.NewZipExtractor(logger),
		installDir: installDir,
	}
}

func (s *Service) Setup(cfg config.AndroidConfig, javaHome string) error {
	requestedPackages := requestedAndroidPackages(cfg)
	if len(requestedPackages) == 0 {
		s.logger.Debug("No Android SDK packages requested in aem.json")
		return nil
	}

	sdkRoot := s.sdkRoot()
	if err := s.fs.EnsureDir(sdkRoot); err != nil {
		return err
	}

	if err := s.ensureCommandLineTools(sdkRoot); err != nil {
		return err
	}

	if err := s.acceptLicenses(sdkRoot, javaHome); err != nil {
		return err
	}

	if err := s.installPackages(sdkRoot, javaHome, requestedPackages); err != nil {
		return err
	}

	s.logger.Debug("Android SDK packages are ready in %s", sdkRoot)
	return nil
}

func (s *Service) Use(symlinkPath string) error {
	if symlinkPath == "" {
		return errors.NewValidationError("android symlink path not configured")
	}

	return s.fs.CreateSymlink(symlinkPath, s.sdkRoot())
}

func (s *Service) sdkRoot() string {
	return filepath.Join(s.installDir, "android", "sdk")
}

func (s *Service) ensureCommandLineTools(sdkRoot string) error {
	binPath := s.sdkManagerPath(sdkRoot)
	if s.fs.Exists(binPath) {
		s.logger.Debug("Android command-line tools already installed")
		return nil
	}

	archiveURL, err := s.resolveCommandLineToolsURL()
	if err != nil {
		return err
	}

	tmpDir, err := s.fs.GetTempDir()
	if err != nil {
		return err
	}

	zipPath := filepath.Join(tmpDir, filepath.Base(archiveURL))
	extractDir := filepath.Join(tmpDir, "android_cmdline_tools_extract")

	defer func() {
		_ = s.fs.RemoveAll(zipPath)
		_ = s.fs.RemoveAll(extractDir)
	}()

	_ = s.fs.RemoveAll(zipPath)
	_ = s.fs.RemoveAll(extractDir)

	if err := s.downloader.Download(archiveURL, zipPath); err != nil {
		return err
	}

	if err := s.zipper.Extract(zipPath, extractDir); err != nil {
		return err
	}

	sourceDir, err := findCmdlineToolsRoot(extractDir)
	if err != nil {
		return err
	}

	targetDir := filepath.Join(sdkRoot, "cmdline-tools", "latest")
	if err := s.fs.EnsureDir(filepath.Dir(targetDir)); err != nil {
		return err
	}

	_ = s.fs.RemoveAll(targetDir)
	if err := s.fs.Move(sourceDir, targetDir); err != nil {
		return err
	}

	return ensureExecutable(filepath.Join(targetDir, "bin"))
}

func (s *Service) resolveCommandLineToolsURL() (string, error) {
	body, err := s.downloader.GetHTML(androidRepositoryURL)
	if err != nil {
		return "", errors.NewAPIError("failed to fetch Android repository metadata", err)
	}
	defer body.Close()

	var repository repositoryXML
	if err := xml.NewDecoder(body).Decode(&repository); err != nil {
		return "", errors.NewAPIError("failed to parse Android repository metadata", err)
	}

	hostOS := mapAndroidHostOS(platform.GetInfo().OS)
	type candidate struct {
		url      string
		revision revision
	}

	var candidates []candidate
	for _, pkg := range repository.Packages {
		if pkg.Path != "cmdline-tools;latest" {
			continue
		}
		for _, archive := range pkg.Archives.Archive {
			if archive.HostOS.Value != hostOS {
				continue
			}
			if archive.Complete.URL == "" {
				continue
			}
			candidates = append(candidates, candidate{
				url:      "https://dl.google.com/android/repository/" + archive.Complete.URL,
				revision: pkg.Revision,
			})
		}
	}

	if len(candidates) == 0 {
		return "", errors.NewValidationError("no Android command-line tools archive found for " + hostOS)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return compareRevision(candidates[i].revision, candidates[j].revision) > 0
	})

	return candidates[0].url, nil
}

func (s *Service) acceptLicenses(sdkRoot, javaHome string) error {
	cmd := s.newSDKManagerCommand(sdkRoot, "--sdk_root="+sdkRoot, "--licenses")
	cmd.Env = s.commandEnv(sdkRoot, javaHome)
	cmd.Stdin = strings.NewReader(strings.Repeat("y\n", 32))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to accept Android SDK licenses: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *Service) installPackages(sdkRoot, javaHome string, packages []string) error {
	args := []string{"--sdk_root=" + sdkRoot}
	args = append(args, packages...)

	cmd := s.newSDKManagerCommand(sdkRoot, args...)
	cmd.Env = s.commandEnv(sdkRoot, javaHome)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install Android SDK packages: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (s *Service) sdkManagerPath(sdkRoot string) string {
	binary := "sdkmanager"
	if platform.GetInfo().OS == "windows" {
		binary = "sdkmanager.bat"
	}
	return filepath.Join(sdkRoot, "cmdline-tools", "latest", "bin", binary)
}

func (s *Service) newSDKManagerCommand(sdkRoot string, args ...string) *exec.Cmd {
	sdkManager := s.sdkManagerPath(sdkRoot)
	if platform.GetInfo().OS == "windows" {
		cmdArgs := append([]string{"/c", sdkManager}, args...)
		return exec.Command("cmd", cmdArgs...)
	}
	return exec.Command(sdkManager, args...)
}

func (s *Service) commandEnv(sdkRoot, javaHome string) []string {
	env := os.Environ()
	env = append(env, "ANDROID_SDK_ROOT="+sdkRoot)
	env = append(env, "ANDROID_HOME="+sdkRoot)
	if javaHome != "" {
		env = append(env, "JAVA_HOME="+javaHome)
	}
	return env
}

func requestedAndroidPackages(cfg config.AndroidConfig) []string {
	seen := make(map[string]struct{})
	var packages []string

	for _, value := range cfg.SDK {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, ";") {
			packages = appendUnique(packages, seen, value)
			continue
		}
		packages = appendUnique(packages, seen, "platforms;android-"+value)
	}

	for _, value := range cfg.BuildTool {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, ";") {
			packages = appendUnique(packages, seen, value)
			continue
		}
		packages = appendUnique(packages, seen, "build-tools;"+value)
	}

	for _, value := range cfg.NDK {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if strings.Contains(value, ";") {
			packages = appendUnique(packages, seen, value)
			continue
		}
		packages = appendUnique(packages, seen, "ndk;"+value)
	}

	packages = appendUnique(packages, seen, "platform-tools")
	return packages
}

func appendUnique(values []string, seen map[string]struct{}, value string) []string {
	if _, exists := seen[value]; exists {
		return values
	}
	seen[value] = struct{}{}
	return append(values, value)
}

func compareRevision(a, b revision) int {
	if a.Major != b.Major {
		return a.Major - b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor - b.Minor
	}
	return a.Micro - b.Micro
}

func mapAndroidHostOS(goos string) string {
	switch goos {
	case "darwin":
		return "macosx"
	case "windows":
		return "windows"
	default:
		return "linux"
	}
}

func findCmdlineToolsRoot(base string) (string, error) {
	candidates := []string{
		filepath.Join(base, "cmdline-tools"),
		filepath.Join(base, "cmdline-tools", "cmdline-tools"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "bin")); err == nil {
			return candidate, nil
		}
	}

	return "", errors.NewExtractionError("android command-line tools archive layout was not recognized", nil)
}

func ensureExecutable(binDir string) error {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bat") {
			continue
		}
		if err := os.Chmod(filepath.Join(binDir, entry.Name()), 0755); err != nil {
			return err
		}
	}

	return nil
}

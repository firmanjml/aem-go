package cmd

import (
	androidsvc "aem/internal/android"
	"aem/internal/config"
	"aem/pkg/logger"
	"aem/pkg/process"
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"golang.org/x/term"
)

type runtimeCandidate struct {
	Version string
	Source  string
}

type initCandidates struct {
	Node           []runtimeCandidate
	Java           []runtimeCandidate
	Android        []string
	AndroidSDKRoot string
}

type initDiscovery func() (initCandidates, error)
type initPrompter func(*cobra.Command, initCandidates) (config.RuntimeConfig, config.AndroidConfig, error)

func newInitCmd() *cobra.Command {
	return newInitCmdWithDiscovery(discoverInitCandidates, promptInitTUI)
}

func newInitCmdWithDiscovery(discover initDiscovery, prompt initPrompter) *cobra.Command {
	var force, nonInteractive bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactively create and validate a canonical aem.json",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workingDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to determine working directory: %w", err)
			}
			path := filepath.Join(workingDir, config.ProjectConfigFileName)

			projectConfig := config.NewProjectConfig()
			if !nonInteractive {
				candidates, err := discover()
				if err != nil {
					return err
				}
				projectConfig.Runtime, projectConfig.Android, err = prompt(cmd, candidates)
				if err != nil {
					return err
				}
			}

			if err := config.WriteProjectConfig(path, projectConfig, force); err != nil {
				return err
			}
			if _, err := config.LoadProjectConfig(path); err != nil {
				return fmt.Errorf("generated configuration did not validate: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created and validated %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite an existing aem.json")
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "create a minimal configuration without prompts")
	return cmd
}

// discoverInitCandidates finds AEM-managed and PATH runtimes, then scans the
// local Android SDK. All discovery is read-only.
func discoverInitCandidates() (initCandidates, error) {
	nodeVersions, javaVersions := discoverRuntimeCandidates()
	installDir, err := aemInstallDir()
	if err != nil {
		return initCandidates{Node: nodeVersions, Java: javaVersions}, nil
	}

	android := androidsvc.NewService(logger.New(false), installDir)
	packages, err := android.List()
	if err != nil {
		return initCandidates{}, fmt.Errorf("failed to scan installed Android SDK components: %w", err)
	}
	return initCandidates{
		Node:           nodeVersions,
		Java:           javaVersions,
		Android:        configurableAndroidPackages(packages),
		AndroidSDKRoot: android.SDKRoot(),
	}, nil
}

// discoverRuntimeCandidates finds versions AEM already manages and the runtime
// versions available on PATH. Discovery is read-only so init never creates an
// AEM directory just to populate a project configuration.
func discoverRuntimeCandidates() (nodeVersions, javaVersions []runtimeCandidate) {
	nodeVersions = append(nodeVersions, managedRuntimeCandidates("node")...)
	javaVersions = append(javaVersions, managedRuntimeCandidates("java")...)

	if version := commandVersion("node", "--version"); version != "" {
		nodeVersions = append(nodeVersions, runtimeCandidate{Version: version, Source: "PATH"})
	}
	if version := javaCommandVersion(); version != "" {
		javaVersions = append(javaVersions, runtimeCandidate{Version: version, Source: "PATH"})
	}

	return mergeAndSortCandidates(nodeVersions), mergeAndSortCandidates(javaVersions)
}

func aemInstallDir() (string, error) {
	home := strings.TrimSpace(os.Getenv("AEM_HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(userHome, ".aem")
	}
	return filepath.Join(home, "sys_installed"), nil
}

func managedRuntimeCandidates(module string) []runtimeCandidate {
	installDir, err := aemInstallDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(installDir, module))
	if err != nil {
		return nil
	}

	versions := make([]runtimeCandidate, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := strings.TrimPrefix(entry.Name(), "v")
		if version != "" {
			versions = append(versions, runtimeCandidate{Version: version, Source: "AEM"})
		}
	}
	return versions
}

func commandVersion(name string, args ...string) string {
	output, err := exec.CommandContext(process.Context(), name, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(output)), "v")
}

func javaCommandVersion() string {
	output, err := exec.CommandContext(process.Context(), "java", "-version").CombinedOutput()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && strings.HasSuffix(fields[0], "version") {
			return strings.Trim(fields[2], "\"")
		}
	}
	return ""
}

func mergeAndSortCandidates(candidates []runtimeCandidate) []runtimeCandidate {
	byVersion := make(map[string]runtimeCandidate, len(candidates))
	for _, candidate := range candidates {
		if existing, found := byVersion[candidate.Version]; found {
			existing.Source += ", " + candidate.Source
			byVersion[candidate.Version] = existing
			continue
		}
		byVersion[candidate.Version] = candidate
	}

	merged := make([]runtimeCandidate, 0, len(byVersion))
	for _, candidate := range byVersion {
		merged = append(merged, candidate)
	}
	sort.Slice(merged, func(i, j int) bool {
		left, right := "v"+merged[i].Version, "v"+merged[j].Version
		if semver.IsValid(left) && semver.IsValid(right) {
			return semver.Compare(left, right) > 0
		}
		return merged[i].Version > merged[j].Version
	})
	return merged
}

func configurableAndroidPackages(packages []string) []string {
	configurable := make([]string, 0, len(packages))
	for _, pkg := range packages {
		parts := strings.Split(pkg, ";")
		switch {
		case len(parts) == 2 && (parts[0] == "platforms" || parts[0] == "build-tools" || parts[0] == "ndk" || parts[0] == "cmake"):
			configurable = append(configurable, pkg)
		case len(parts) == 4 && parts[0] == "system-images":
			configurable = append(configurable, pkg)
		}
	}
	sort.Strings(configurable)
	return configurable
}

func androidConfigFromPackages(packages []string) (config.AndroidConfig, error) {
	var android config.AndroidConfig
	for _, pkg := range packages {
		parts := strings.Split(pkg, ";")
		switch {
		case len(parts) == 2 && parts[0] == "platforms":
			version := strings.TrimPrefix(parts[1], "android-")
			if version == "" || version == parts[1] {
				return config.AndroidConfig{}, fmt.Errorf("invalid Android platform package %q", pkg)
			}
			android.Platforms = append(android.Platforms, version)
		case len(parts) == 2 && parts[0] == "build-tools":
			android.BuildTools = append(android.BuildTools, parts[1])
		case len(parts) == 2 && parts[0] == "ndk":
			android.NDK = append(android.NDK, parts[1])
		case len(parts) == 2 && parts[0] == "cmake":
			android.CMake = append(android.CMake, parts[1])
		case len(parts) == 4 && parts[0] == "system-images":
			apiLevel, exact, err := parseSystemImageAPILevel(parts[1])
			if err != nil {
				return config.AndroidConfig{}, fmt.Errorf("invalid Android system image package %q", pkg)
			}
			image := config.SystemImage{
				APILevel: apiLevel, Variant: parts[2], Architecture: parts[3],
			}
			if exact != fmt.Sprintf("android-%d", apiLevel) {
				image.Package = pkg
			}
			android.SystemImages = append(android.SystemImages, image)
		default:
			return config.AndroidConfig{}, fmt.Errorf("unsupported Android package %q", pkg)
		}
	}
	return android, nil
}

// parseSystemImageAPILevel preserves preview package identifiers such as
// android-37.0 while retaining the integer apiLevel used by existing configs.
func parseSystemImageAPILevel(identifier string) (apiLevel int, exact string, err error) {
	if !strings.HasPrefix(identifier, "android-") {
		return 0, "", fmt.Errorf("missing android prefix")
	}
	exact = identifier
	major := strings.TrimPrefix(identifier, "android-")
	if dot := strings.IndexByte(major, '.'); dot >= 0 {
		major = major[:dot]
	}
	apiLevel, err = strconv.Atoi(major)
	if err != nil || apiLevel <= 0 {
		return 0, "", fmt.Errorf("invalid API level")
	}
	return apiLevel, exact, nil
}

func promptInitTUI(cmd *cobra.Command, candidates initCandidates) (config.RuntimeConfig, config.AndroidConfig, error) {
	input, inputOK := cmd.InOrStdin().(*os.File)
	output, outputOK := cmd.OutOrStdout().(*os.File)
	if !inputOK || !outputOK || !term.IsTerminal(int(input.Fd())) || !term.IsTerminal(int(output.Fd())) {
		return config.RuntimeConfig{}, config.AndroidConfig{}, fmt.Errorf("aem init needs an interactive terminal; use --non-interactive for scripts")
	}

	oldState, err := term.MakeRaw(int(input.Fd()))
	if err != nil {
		return config.RuntimeConfig{}, config.AndroidConfig{}, fmt.Errorf("failed to start interactive init: %w", err)
	}
	defer term.Restore(int(input.Fd()), oldState)
	defer fmt.Fprint(output, "\x1b[?25h\x1b[?1049l")
	fmt.Fprint(output, "\x1b[?1049h\x1b[?25l")
	reader := bufio.NewReader(input)

	node, err := selectTUIOption(reader, output, "Node.js runtime", runtimeOptions(candidates.Node))
	if err != nil {
		return config.RuntimeConfig{}, config.AndroidConfig{}, err
	}
	java, err := selectTUIOption(reader, output, "Java runtime", runtimeOptions(candidates.Java))
	if err != nil {
		return config.RuntimeConfig{}, config.AndroidConfig{}, err
	}
	selectedAndroid, err := selectTUIMultiple(reader, output, "Android SDK components", candidates.Android, candidates.AndroidSDKRoot)
	if err != nil {
		return config.RuntimeConfig{}, config.AndroidConfig{}, err
	}
	android, err := androidConfigFromPackages(selectedAndroid)
	if err != nil {
		return config.RuntimeConfig{}, config.AndroidConfig{}, err
	}
	return config.RuntimeConfig{Node: node, Java: java}, android, nil
}

type tuiOption struct {
	Label string
	Value string
}

func runtimeOptions(candidates []runtimeCandidate) []tuiOption {
	options := []tuiOption{{Label: "Do not add a runtime requirement"}}
	for _, candidate := range candidates {
		options = append(options, tuiOption{Label: fmt.Sprintf("%s  (%s)", candidate.Version, candidate.Source), Value: candidate.Version})
	}
	return options
}

func selectTUIOption(input *bufio.Reader, output *os.File, title string, options []tuiOption) (string, error) {
	selected := 0
	for {
		renderTUI(output, title, "Use ↑/↓ (or j/k) to move, Enter to confirm, q to cancel.", func() {
			for index, option := range options {
				cursor := "  "
				if index == selected {
					cursor = "› "
				}
				writeTUILine(output, "%s%s", cursor, option.Label)
			}
		})

		key, err := readTUIKey(input)
		if err != nil {
			return "", err
		}
		switch key {
		case tuiUp:
			selected = (selected + len(options) - 1) % len(options)
		case tuiDown:
			selected = (selected + 1) % len(options)
		case tuiEnter:
			return options[selected].Value, nil
		case tuiCancel:
			return "", fmt.Errorf("interactive init cancelled")
		}
	}
}

func selectTUIMultiple(input *bufio.Reader, output *os.File, title string, options []string, sdkRoot string) ([]string, error) {
	if len(options) == 0 {
		renderTUI(output, title, "No configurable installed components were found. Press Enter to continue, or q to cancel.", func() {
			writeTUILine(output, "Android SDK: %s", sdkRoot)
		})
		for {
			key, err := readTUIKey(input)
			if err != nil {
				return nil, err
			}
			if key == tuiEnter {
				return nil, nil
			}
			if key == tuiCancel {
				return nil, fmt.Errorf("interactive init cancelled")
			}
		}
	}

	selected := make([]bool, len(options))
	cursor := 0
	for {
		renderTUI(output, title, "Space toggles components. Use ↑/↓ (or j/k), Enter to confirm, q to cancel.", func() {
			writeTUILine(output, "Android SDK: %s", sdkRoot)
			writeTUILine(output, "")
			for index, option := range options {
				marker, current := "[ ]", "  "
				if selected[index] {
					marker = "[x]"
				}
				if index == cursor {
					current = "› "
				}
				writeTUILine(output, "%s%s %s", current, marker, option)
			}
		})

		key, err := readTUIKey(input)
		if err != nil {
			return nil, err
		}
		switch key {
		case tuiUp:
			cursor = (cursor + len(options) - 1) % len(options)
		case tuiDown:
			cursor = (cursor + 1) % len(options)
		case tuiSpace:
			selected[cursor] = !selected[cursor]
		case tuiEnter:
			packages := make([]string, 0, len(options))
			for index, option := range options {
				if selected[index] {
					packages = append(packages, option)
				}
			}
			return packages, nil
		case tuiCancel:
			return nil, fmt.Errorf("interactive init cancelled")
		}
	}
}

func renderTUI(output io.Writer, title, instructions string, body func()) {
	fmt.Fprint(output, "\x1b[2J\x1b[H")
	writeTUILine(output, "AEM init — %s", title)
	writeTUILine(output, "")
	writeTUILine(output, "%s", instructions)
	writeTUILine(output, "")
	body()
}

// writeTUILine emits CRLF explicitly. Terminal raw mode does not guarantee
// that a lone line feed returns the cursor to column zero.
func writeTUILine(output io.Writer, format string, args ...any) {
	fmt.Fprintf(output, format, args...)
	fmt.Fprint(output, "\r\n")
}

type tuiKey int

const (
	tuiUnknown tuiKey = iota
	tuiUp
	tuiDown
	tuiEnter
	tuiSpace
	tuiCancel
)

func readTUIKey(input *bufio.Reader) (tuiKey, error) {
	key, err := input.ReadByte()
	if err != nil {
		return tuiUnknown, fmt.Errorf("failed to read interactive input: %w", err)
	}
	switch key {
	case '\r', '\n':
		return tuiEnter, nil
	case ' ':
		return tuiSpace, nil
	case 'j':
		return tuiDown, nil
	case 'k':
		return tuiUp, nil
	case 'q', 3:
		return tuiCancel, nil
	case 27:
		next, err := input.ReadByte()
		if err != nil || next != '[' {
			return tuiCancel, nil
		}
		direction, err := input.ReadByte()
		if err != nil {
			return tuiUnknown, fmt.Errorf("failed to read terminal escape sequence: %w", err)
		}
		switch direction {
		case 'A':
			return tuiUp, nil
		case 'B':
			return tuiDown, nil
		}
	}
	return tuiUnknown, nil
}

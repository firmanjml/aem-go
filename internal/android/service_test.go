package android

import (
	"aem/internal/config"
	"aem/pkg/logger"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestRequestedAndroidPackagesUsesCanonicalConfiguration(t *testing.T) {
	cfg := config.AndroidConfig{
		Platforms:  config.StringList{"35", "platforms;android-34"},
		BuildTools: config.StringList{"35.0.0"},
		NDK:        config.StringList{"27.0.12077973"},
		CMake:      config.StringList{"3.22.1"},
		SystemImages: []config.SystemImage{{
			APILevel:     35,
			Variant:      "google_apis",
			Architecture: "arm64-v8a",
		}},
	}

	got := requestedAndroidPackages(cfg)
	want := []string{
		"platforms;android-35",
		"platforms;android-34",
		"build-tools;35.0.0",
		"ndk;27.0.12077973",
		"cmake;3.22.1",
		"system-images;android-35;google_apis;arm64-v8a",
		"platform-tools",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requestedAndroidPackages() = %v, want %v", got, want)
	}
}

func TestRequestedAndroidPackagesPreservesPreviewSystemImagePackage(t *testing.T) {
	const previewPackage = "system-images;android-37.0;google_apis_playstore_ps16k;arm64-v8a"
	got := requestedAndroidPackages(config.AndroidConfig{SystemImages: []config.SystemImage{{
		APILevel: 37, Variant: "google_apis_playstore_ps16k", Architecture: "arm64-v8a", Package: previewPackage,
	}}})
	want := []string{previewPackage, "platform-tools"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requestedAndroidPackages() = %v, want %v", got, want)
	}
}

func TestAndroidPackagePathUsesExpectedSDKLayout(t *testing.T) {
	sdkRoot := filepath.Join(t.TempDir(), "sdk")
	tests := map[string]string{
		"platform-tools":       filepath.Join("platform-tools"),
		"platforms;android-35": filepath.Join("platforms", "android-35"),
		"build-tools;35.0.0":   filepath.Join("build-tools", "35.0.0"),
		"ndk;27.0.12077973":    filepath.Join("ndk", "27.0.12077973"),
		"cmake;3.22.1":         filepath.Join("cmake", "3.22.1"),
		"system-images;android-35;google_apis;arm64-v8a": filepath.Join("system-images", "android-35", "google_apis", "arm64-v8a"),
	}
	for pkg, wantSuffix := range tests {
		path, ok := androidPackagePath(sdkRoot, pkg)
		if !ok || path != filepath.Join(sdkRoot, wantSuffix) {
			t.Errorf("androidPackagePath(%q) = %q, %v; want %q, true", pkg, path, ok, filepath.Join(sdkRoot, wantSuffix))
		}
	}

	for _, pkg := range []string{"platforms;..", "system-images;android-35;google_apis", "unknown;package"} {
		if _, ok := androidPackagePath(sdkRoot, pkg); ok {
			t.Errorf("androidPackagePath(%q) accepted an invalid package", pkg)
		}
	}
}

func TestSetupInstallsOnlyMissingPackagesWithoutNetwork(t *testing.T) {
	service := NewService(logger.New(false), t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	sdkRoot := service.SDKRoot()
	if err := os.MkdirAll(filepath.Dir(service.sdkManagerPath(sdkRoot)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(service.sdkManagerPath(sdkRoot), []byte("placeholder"), 0o755); err != nil {
		t.Fatal(err)
	}

	var licenseCalls, installCalls int
	service.runCommand = func(root, javaHome string, args ...string) ([]byte, error) {
		if root != sdkRoot {
			t.Fatalf("SDK root = %q, want %q", root, sdkRoot)
		}
		if len(args) == 2 && args[1] == "--licenses" {
			licenseCalls++
			return nil, nil
		}
		installCalls++
		for _, pkg := range args[1:] {
			path, ok := androidPackagePath(sdkRoot, pkg)
			if !ok {
				t.Fatalf("unexpected package %q", pkg)
			}
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		return nil, nil
	}

	cfg := config.AndroidConfig{
		Platforms:  config.StringList{"35"},
		BuildTools: config.StringList{"35.0.0"},
		NDK:        config.StringList{"27.0.12077973"},
		CMake:      config.StringList{"3.22.1"},
		SystemImages: []config.SystemImage{{
			APILevel: 35, Variant: "google_apis", Architecture: "arm64-v8a",
		}},
	}
	if err := service.Setup(cfg, ""); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if licenseCalls != 1 || installCalls != 1 {
		t.Fatalf("first setup calls = licenses %d, installs %d; want 1, 1", licenseCalls, installCalls)
	}
	if err := service.Setup(cfg, ""); err != nil {
		t.Fatalf("second Setup() error = %v", err)
	}
	if licenseCalls != 1 || installCalls != 1 {
		t.Fatalf("idempotent setup calls = licenses %d, installs %d; want 1, 1", licenseCalls, installCalls)
	}
}

func TestInstallInstallsRawPackageIDAndIsIdempotent(t *testing.T) {
	service := NewService(logger.New(false), t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	sdkRoot := service.SDKRoot()
	if err := os.MkdirAll(filepath.Dir(service.sdkManagerPath(sdkRoot)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(service.sdkManagerPath(sdkRoot), []byte("placeholder"), 0o755); err != nil {
		t.Fatal(err)
	}

	var licenseCalls, installCalls int
	service.runCommand = func(root, javaHome string, args ...string) ([]byte, error) {
		if root != sdkRoot {
			t.Fatalf("SDK root = %q, want %q", root, sdkRoot)
		}
		if len(args) == 2 && args[1] == "--licenses" {
			licenseCalls++
			return nil, nil
		}
		installCalls++
		if got, want := args, []string{"--sdk_root=" + sdkRoot, "platforms;android-37.1"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("sdkmanager arguments = %q, want %q", got, want)
		}
		path, ok := androidPackagePath(sdkRoot, args[1])
		if !ok {
			t.Fatalf("package %q has no SDK path", args[1])
		}
		return nil, os.MkdirAll(path, 0o755)
	}

	if err := service.Install("platforms;android-37.1", ""); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if err := service.Install("platforms;android-37.1", ""); err != nil {
		t.Fatalf("second Install() error = %v", err)
	}
	if licenseCalls != 1 || installCalls != 1 {
		t.Fatalf("calls = licenses %d, installs %d; want 1, 1", licenseCalls, installCalls)
	}
}

func TestInstallRejectsInvalidPackageID(t *testing.T) {
	service := NewService(logger.New(false), t.TempDir())
	for _, packageID := range []string{"", "--licenses", "platforms;android-35 extra"} {
		if err := service.Install(packageID, ""); err == nil {
			t.Errorf("Install(%q) succeeded, want validation error", packageID)
		}
	}
}

func TestUninstallRemovesInstalledPackageAndIsIdempotent(t *testing.T) {
	service := NewService(logger.New(false), t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	sdkRoot := service.SDKRoot()
	packageID := "system-images;android-37.1;google_apis_playstore_ps16k;arm64-v8a"
	packagePath, ok := androidPackagePath(sdkRoot, packageID)
	if !ok {
		t.Fatalf("androidPackagePath(%q) did not resolve", packageID)
	}
	if err := os.MkdirAll(packagePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(service.sdkManagerPath(sdkRoot)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(service.sdkManagerPath(sdkRoot), []byte("placeholder"), 0o755); err != nil {
		t.Fatal(err)
	}

	var calls int
	service.runCommand = func(root, javaHome string, args ...string) ([]byte, error) {
		calls++
		want := []string{"--sdk_root=" + sdkRoot, "--uninstall", packageID}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("sdkmanager arguments = %q, want %q", args, want)
		}
		return nil, os.RemoveAll(packagePath)
	}

	if err := service.Uninstall(packageID, ""); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if err := service.Uninstall(packageID, ""); err != nil {
		t.Fatalf("second Uninstall() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("sdkmanager calls = %d, want 1", calls)
	}
	if service.fs.Exists(packagePath) {
		t.Fatalf("package remains at %s", packagePath)
	}
}

func TestUninstallRetainsCommandLineTools(t *testing.T) {
	service := NewService(logger.New(false), t.TempDir())
	if err := service.Uninstall("cmdline-tools;latest", ""); err == nil {
		t.Fatal("Uninstall() allowed removal of Android command-line tools")
	}
}

func TestSDKManagerCapturesPackageInstallationOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX shell script")
	}

	service := NewService(logger.New(false), t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	sdkRoot := service.SDKRoot()
	sdkManager := service.sdkManagerPath(sdkRoot)
	if err := os.MkdirAll(filepath.Dir(sdkManager), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sdkManager, []byte("#!/bin/sh\necho downloading\necho extracting >&2\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	output, err := service.runSDKManagerCommand(sdkRoot, "", "--sdk_root="+sdkRoot, "platform-tools")
	if err != nil {
		t.Fatalf("runSDKManagerCommand() error = %v", err)
	}
	for _, message := range []string{"downloading", "extracting"} {
		if !strings.Contains(string(output), message) {
			t.Errorf("captured output = %q, want %q", output, message)
		}
	}
}

func TestSDKManagerErrorDetailExcludesTransientProgress(t *testing.T) {
	output := "Loading package information...\r[===                                    ] 10% Computing updates...\rWarning: Failed to find package 'system-images;android-37.1;google_apis_playstore;arm64-v8a'\n"
	const want = "Warning: Failed to find package 'system-images;android-37.1;google_apis_playstore;arm64-v8a'"
	if got := sdkManagerErrorDetail(output); got != want {
		t.Fatalf("sdkManagerErrorDetail() = %q, want %q", got, want)
	}
}

func TestListAtReportsInstalledAndroidComponents(t *testing.T) {
	service := NewService(logger.New(false), t.TempDir())
	sdkRoot := filepath.Join(t.TempDir(), "sdk")
	t.Setenv("ANDROID_HOME", sdkRoot)
	packages := []string{
		"platform-tools",
		"cmdline-tools;latest",
		"platforms;android-35",
		"build-tools;35.0.0",
		"ndk;27.0.12077973",
		"cmake;3.22.1",
		"system-images;android-35;google_apis;arm64-v8a",
	}
	for _, pkg := range packages {
		var path string
		if pkg == "cmdline-tools;latest" {
			path = filepath.Join(sdkRoot, "cmdline-tools", "latest")
		} else {
			var ok bool
			path, ok = androidPackagePath(sdkRoot, pkg)
			if !ok {
				t.Fatalf("no local path for %q", pkg)
			}
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(sdkRoot, "extras", "google", "m2repository"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sdkRoot, "extras", "google", "m2repository", "package.xml"), []byte(`<localPackage path="extras;google;m2repository"/>`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := service.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	want := []string{
		"build-tools;35.0.0",
		"cmake;3.22.1",
		"cmdline-tools;latest",
		"extras;google;m2repository",
		"ndk;27.0.12077973",
		"platform-tools",
		"platforms;android-35",
		"system-images;android-35;google_apis;arm64-v8a",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %v, want %v", got, want)
	}
}

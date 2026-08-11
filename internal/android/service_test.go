package android

import (
	"aem/internal/config"
	"aem/pkg/logger"
	"os"
	"path/filepath"
	"reflect"
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

package node

import (
	"aem/internal/platform"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNodeExtensionResolvesAndFiltersVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.json" {
			t.Fatalf("request path = %q, want /index.json", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
  {"version":"v22.12.0", "files":["osx-arm64-tar"]},
  {"version":"v22.11.0", "files":["osx-arm64-tar"]},
  {"version":"v20.19.4", "files":["linux-x64"]}
]`))
	}))
	defer server.Close()

	extension := newNodeExtension(server.URL, platform.Info{OS: "darwin", Arch: "arm64"})
	versions, err := extension.ListVersions(stringPtr("22"))
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if want := []string{"22.12.0", "22.11.0"}; !reflect.DeepEqual(versions, want) {
		t.Fatalf("ListVersions() = %v, want %v", versions, want)
	}

	found, err := extension.CheckVersion("22.12.0")
	if err != nil || !found {
		t.Fatalf("CheckVersion() = %v, %v; want true, nil", found, err)
	}
	found, err = extension.CheckVersion("20.0.0")
	if err != nil || found {
		t.Fatalf("CheckVersion() = %v, %v; want false, nil", found, err)
	}
}

func TestNodeExtensionSelectsHostSpecificArchive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"version":"v22.12.0", "files":["win-arm64-zip", "linux-x64"]}]`))
	}))
	defer server.Close()

	windows := newNodeExtension(server.URL, platform.Info{OS: "windows", Arch: "arm64"})
	got, err := windows.GetDownloadURL("22.12.0")
	want := server.URL + "/v22.12.0/node-v22.12.0-win-arm64.zip"
	if err != nil || got != want {
		t.Fatalf("Windows GetDownloadURL() = %q, %v; want %q, nil", got, err, want)
	}

	linux := newNodeExtension(server.URL, platform.Info{OS: "linux", Arch: "arm64"})
	if _, err := linux.GetDownloadURL("22.12.0"); err == nil {
		t.Fatal("GetDownloadURL() succeeded without a Linux ARM64 binary")
	}
}

func stringPtr(value string) *string { return &value }

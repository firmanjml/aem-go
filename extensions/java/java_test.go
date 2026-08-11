package java

import (
	"aem/internal/platform"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestJavaExtensionResolvesAndFiltersVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("java_version") {
		case "17":
			_, _ = w.Write([]byte(`[
  {"java_version":[17,0,12]},
  {"java_version":[17,0,11]},
  {"java_version":[21,0,4]},
  {"java_version":[17,0,12]}
]`))
		case "17.0.12":
			_, _ = w.Write([]byte(`[{"java_version":[17,0,12]}]`))
		default:
			t.Errorf("unexpected java_version query %q", r.URL.Query().Get("java_version"))
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	extension := newJavaExtension(server.URL, platform.Info{OS: "darwin", Arch: "arm64"})
	versions, err := extension.ListVersions(stringPtr("17"))
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if want := []string{"17.0.12", "17.0.11"}; !reflect.DeepEqual(versions, want) {
		t.Fatalf("ListVersions() = %v, want %v", versions, want)
	}

	found, err := extension.CheckVersion("17.0.12")
	if err != nil || !found {
		t.Fatalf("CheckVersion(17.0.12) = %v, %v; want true, nil", found, err)
	}
	found, err = extension.CheckVersion("v17.0.12")
	if err != nil || !found {
		t.Fatalf("CheckVersion(v17.0.12) = %v, %v; want true, nil", found, err)
	}
}

func TestJavaExtensionUsesAzulHostTargetAndReturnsURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("os"), "windows"; got != want {
			t.Errorf("os = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("arch"), "x64"; got != want {
			t.Errorf("arch = %q, want %q", got, want)
		}
		_, _ = w.Write([]byte(`[{"java_version":[17,0,12],"download_url":"https://example.test/zulu.zip"}]`))
	}))
	defer server.Close()

	extension := newJavaExtension(server.URL, platform.Info{OS: "windows", Arch: "amd64"})
	got, err := extension.GetDownloadURL("v17.0.12")
	if err != nil || got != "https://example.test/zulu.zip" {
		t.Fatalf("GetDownloadURL() = %q, %v", got, err)
	}
}

func stringPtr(value string) *string { return &value }

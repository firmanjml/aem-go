package java

import (
	"aem/internal/platform"
	"aem/pkg/filesystem"
	"aem/pkg/logger"
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWithTemporaryAEMHomeAndMockedZuluDownload(t *testing.T) {
	t.Setenv("AEM_HOME", t.TempDir())
	archive := jdkTarGz(t, "zulu17/bin/java", []byte("java"))
	downloads := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/packages/":
			if got := r.URL.Query().Get("os"); got != "linux" {
				t.Errorf("os = %q, want linux", got)
			}
			if got := r.URL.Query().Get("arch"); got != "x64" {
				t.Errorf("arch = %q, want x64", got)
			}
			_, _ = w.Write([]byte(`[{"name":"zulu17.tar.gz","download_url":"` + server.URL + `/zulu17.tar.gz","java_version":[17,0,12]}]`))
		case "/zulu17.tar.gz":
			downloads++
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	fs := filesystem.New(logger.New(false))
	installDir, err := fs.GetInstallDir()
	if err != nil {
		t.Fatal(err)
	}
	service := newService(logger.New(false), installDir, server.URL+"/packages/", platform.Info{OS: "linux", Arch: "amd64"})
	installed, err := service.Install("17")
	if err != nil || installed != "v17.0.12" {
		t.Fatalf("Install() = %q, %v; want v17.0.12, nil", installed, err)
	}
	link := filepath.Join(t.TempDir(), "current", "java")
	if err := service.Use(installed, link); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "bin", "java")); err != nil {
		t.Fatalf("installed Java executable missing: %v", err)
	}
	if _, err := service.Install("17"); err != nil {
		t.Fatalf("idempotent Install() error = %v", err)
	}
	if downloads != 1 {
		t.Fatalf("downloads = %d, want one", downloads)
	}
}

func jdkTarGz(t *testing.T, name string, contents []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gzipWriter)
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(contents))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

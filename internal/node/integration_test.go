package node

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

func TestInstallWithTemporaryAEMHomeAndMockedNodeDownload(t *testing.T) {
	t.Setenv("AEM_HOME", t.TempDir())
	archive := tarGz(t, "node-v20.19.4/bin/node", []byte("node"))
	downloads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dist/":
			_, _ = w.Write([]byte(`<a href="v20.19.4/">v20.19.4/</a>`))
		case "/dist/v20.19.4":
			_, _ = w.Write([]byte(`<a href="/dist/v20.19.4/node-v20.19.4-linux-x64.tar.gz">archive</a>`))
		case "/dist/v20.19.4/node-v20.19.4-linux-x64.tar.gz":
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
	service := newService(logger.New(false), installDir, server.URL+"/dist", platform.Info{OS: "linux", Arch: "amd64"})
	installed, err := service.Install("20")
	if err != nil || installed != "20.19.4" {
		t.Fatalf("Install() = %q, %v; want 20.19.4, nil", installed, err)
	}
	link := filepath.Join(t.TempDir(), "current", "node")
	if err := service.Use(installed, link); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if target != filepath.Join(installDir, "node", "v20.19.4") {
		t.Fatalf("active link = %q", target)
	}
	if _, err := os.Stat(filepath.Join(target, "bin", "node")); err != nil {
		t.Fatalf("installed executable missing: %v", err)
	}
	if _, err := service.Install("20"); err != nil {
		t.Fatalf("idempotent Install() error = %v", err)
	}
	if downloads != 1 {
		t.Fatalf("downloads = %d, want one", downloads)
	}
}

func tarGz(t *testing.T, name string, contents []byte) []byte {
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

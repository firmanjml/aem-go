package archiver

import (
	"aem/pkg/logger"
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestZipExtractorPreservesRuntimeExecutable(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "node.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: "node-v20/bin/node", Method: zip.Deflate}
	header.SetMode(0o755)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("node")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(root, "extract")
	if err := NewZipExtractor(logger.New(false)).Extract(archivePath, dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	assertExecutableWhenSupported(t, filepath.Join(dest, "node-v20", "bin", "node"))
}

func TestTarGzExtractorPreservesRuntimeExecutable(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "node.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	writer := tar.NewWriter(gzipWriter)
	if err := writer.WriteHeader(&tar.Header{Name: "node-v20/bin/node", Mode: 0o755, Size: 4, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("node")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(root, "extract")
	if err := NewTarGzExtractor(logger.New(false)).Extract(archivePath, dest); err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	assertExecutableWhenSupported(t, filepath.Join(dest, "node-v20", "bin", "node"))
}

func assertExecutableWhenSupported(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		t.Fatalf("%s mode = %v; executable bit was not preserved", path, info.Mode())
	}
}

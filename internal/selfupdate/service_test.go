package selfupdate

import (
	"aem/pkg/logger"
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const testBinaryContent = "new aem release binary"

func testAssetName(version string) string {
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("aem_%s_%s_%s.%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

func testBinaryName() string {
	if runtime.GOOS == "windows" {
		return "aem.exe"
	}
	return "aem"
}

func buildTestArchive(t *testing.T) []byte {
	t.Helper()

	buf := &bytes.Buffer{}
	if runtime.GOOS == "windows" {
		zw := zip.NewWriter(buf)
		header := &zip.FileHeader{Name: testBinaryName(), Method: zip.Deflate}
		header.SetMode(0755)
		writer, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(testBinaryContent)); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	gzw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gzw)
	if err := tw.WriteHeader(&tar.Header{
		Name:     testBinaryName(),
		Typeflag: tar.TypeReg,
		Size:     int64(len(testBinaryContent)),
		Mode:     0755,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(testBinaryContent)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// testArchiveServer serves a latest release (tag), a pinned release
// (pinnedTag), the release archive, and checksums.txt from in-memory data.
// Asset URLs are built from the request host so the server can be pinned to a
// tampered archive between requests.
type testArchiveServer struct {
	t         *testing.T
	tag       string
	pinnedTag string
	archive   []byte // mutable so a test can tamper with it
	checksum  string // checksum computed once from the original archive
}

func (s *testArchiveServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	base := "http://" + r.Host
	assetName := testAssetName(strings.TrimPrefix(s.tag, "v"))
	pinnedAssetName := testAssetName(strings.TrimPrefix(s.pinnedTag, "v"))

	releaseJSON := func(tag string) string {
		name := testAssetName(strings.TrimPrefix(tag, "v"))
		return fmt.Sprintf(`{
  "tag_name": %q,
  "assets": [
    {"name": %q, "browser_download_url": %q},
    {"name": "checksums.txt", "browser_download_url": %q}
  ]
}`, tag, name, base+"/dl/"+name, base+"/dl/checksums.txt")
	}

	switch r.URL.Path {
	case "/repos/aem/aem/releases/latest":
		if r.Header.Get("User-Agent") == "" {
			s.t.Error("release request did not set a user agent")
		}
		io.WriteString(w, releaseJSON(s.tag))
	case "/repos/aem/aem/releases/tags/" + s.pinnedTag:
		io.WriteString(w, releaseJSON(s.pinnedTag))
	case "/dl/checksums.txt":
		fmt.Fprintf(w, "%s  %s\n%s  %s\n", s.checksum, assetName, s.checksum, pinnedAssetName)
	default:
		if strings.HasPrefix(r.URL.Path, "/dl/") {
			w.Write(s.archive)
			return
		}
		http.NotFound(w, r)
	}
}

func newTestServer(t *testing.T, tag, pinnedTag string, archive []byte) *httptest.Server {
	t.Helper()
	sum := sha256.Sum256(archive)
	server := httptest.NewServer(&testArchiveServer{
		t:         t,
		tag:       tag,
		pinnedTag: pinnedTag,
		archive:   archive,
		checksum:  hex.EncodeToString(sum[:]),
	})
	t.Cleanup(server.Close)
	return server
}

func newTestService(t *testing.T, server *httptest.Server) *Service {
	t.Helper()
	return NewService(
		logger.New(false),
		t.TempDir(),
		WithAPIBaseURL(server.URL),
		WithRepository("aem/aem"),
	)
}

func writeFakeExecutable(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := filepath.Join(dir, testBinaryName())
	if err := os.WriteFile(exe, []byte("old aem build"), 0755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestIsReleaseVersion(t *testing.T) {
	if IsReleaseVersion("dev") {
		t.Error("dev builds must not be treated as release versions")
	}
	if !IsReleaseVersion("1.2.3") || !IsReleaseVersion("v1.2.3") {
		t.Error("tagged versions must be treated as release versions")
	}
}

func TestCheckUpdateAvailableAndUpToDate(t *testing.T) {
	archive := buildTestArchive(t)
	server := newTestServer(t, "v1.2.3", "v1.1.0", archive)
	service := newTestService(t, server)

	available, err := service.Check("1.1.0", "")
	if err != nil {
		t.Fatalf("Check error = %v", err)
	}
	if !available.UpdateAvailable || available.UpToDate || available.Downgrade {
		t.Fatalf("expected update status for 1.1.0 -> v1.2.3, got %+v", available)
	}

	uptodate, err := service.Check("v1.2.3", "")
	if err != nil {
		t.Fatalf("Check error = %v", err)
	}
	if !uptodate.UpToDate || uptodate.UpdateAvailable {
		t.Fatalf("expected up-to-date status for v1.2.3, got %+v", uptodate)
	}

	downgrade, err := service.Check("1.2.3", "1.1.0")
	if err != nil {
		t.Fatalf("Check error = %v", err)
	}
	if !downgrade.Downgrade || downgrade.UpdateAvailable {
		t.Fatalf("expected downgrade status for 1.2.3 -> 1.1.0, got %+v", downgrade)
	}

	dev, err := service.Check("dev", "")
	if err != nil {
		t.Fatalf("Check error = %v", err)
	}
	if !dev.CurrentIsDev || !dev.UpdateAvailable {
		t.Fatalf("expected dev-build status, got %+v", dev)
	}
}

func TestUpdateInstallsLatestRelease(t *testing.T) {
	archive := buildTestArchive(t)
	server := newTestServer(t, "v1.2.3", "v1.1.0", archive)
	service := newTestService(t, server)
	exe := writeFakeExecutable(t)

	result, err := service.Update("1.1.0", "", exe, false)
	if err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if result.To != "v1.2.3" || result.AlreadyInstalled {
		t.Fatalf("unexpected update result %+v", result)
	}

	installed, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != testBinaryContent {
		t.Fatalf("executable content = %q, want the new release binary", installed)
	}

	info, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		t.Fatalf("updated executable is not executable: mode %v", info.Mode())
	}
	if _, err := os.Stat(exe + ".update-old"); !os.IsNotExist(err) {
		t.Error("backup executable should be removed after a successful update")
	}
}

func TestUpdateSkipsWhenAlreadyUpToDate(t *testing.T) {
	archive := buildTestArchive(t)
	server := newTestServer(t, "v1.2.3", "v1.1.0", archive)
	service := newTestService(t, server)
	exe := writeFakeExecutable(t)

	result, err := service.Update("1.2.3", "", exe, false)
	if err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if !result.AlreadyInstalled {
		t.Fatalf("expected already-installed result, got %+v", result)
	}

	installed, _ := os.ReadFile(exe)
	if string(installed) != "old aem build" {
		t.Fatal("an up-to-date executable must not be replaced")
	}
}

func TestUpdateRefusesDowngradeWithoutForce(t *testing.T) {
	archive := buildTestArchive(t)
	server := newTestServer(t, "v1.2.3", "v1.1.0", archive)
	service := newTestService(t, server)
	exe := writeFakeExecutable(t)

	if _, err := service.Update("1.2.3", "1.1.0", exe, false); err == nil {
		t.Fatal("a pinned downgrade without --force must fail")
	}

	installed, _ := os.ReadFile(exe)
	if string(installed) != "old aem build" {
		t.Fatal("a refused downgrade must not modify the executable")
	}

	if _, err := service.Update("1.2.3", "1.1.0", exe, true); err != nil {
		t.Fatalf("a forced downgrade must succeed: %v", err)
	}
	installed, _ = os.ReadFile(exe)
	if string(installed) != testBinaryContent {
		t.Fatal("a forced downgrade must replace the executable")
	}
}

func TestUpdateRefusesDevBuildsWithoutForce(t *testing.T) {
	archive := buildTestArchive(t)
	server := newTestServer(t, "v1.2.3", "v1.1.0", archive)
	service := newTestService(t, server)
	exe := writeFakeExecutable(t)

	if _, err := service.Update("dev", "", exe, false); err == nil {
		t.Fatal("updating a dev build without --force must fail")
	}

	result, err := service.Update("dev", "", exe, true)
	if err != nil {
		t.Fatalf("a forced dev-build update must succeed: %v", err)
	}
	if result.To != "v1.2.3" {
		t.Fatalf("forced dev-build update installed %+v", result)
	}
}

func TestUpdateRejectsChecksumMismatch(t *testing.T) {
	archive := buildTestArchive(t)
	server := newTestServer(t, "v1.2.3", "v1.1.0", archive)
	service := newTestService(t, server)
	exe := writeFakeExecutable(t)

	// Corrupt the archive the server hands out so its checksum no longer
	// matches the published value.
	tampered := append([]byte("tampered"), archive...)
	server.Config.Handler.(*testArchiveServer).archive = tampered

	if _, err := service.Update("1.1.0", "", exe, false); err == nil {
		t.Fatal("a checksum mismatch must abort the update")
	} else if !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected a checksum error, got %v", err)
	}

	installed, _ := os.ReadFile(exe)
	if string(installed) != "old aem build" {
		t.Fatal("a failed update must leave the current executable intact")
	}
}

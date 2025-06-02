package node

import (
	"aem/internal/config"
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/net/html"
)

func DownloadURL(version string) string {
	url := "https://nodejs.org/dist/" + version
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var target string

	switch goos {
	case "darwin":
		target = "darwin-x64"
		if goarch == "arm64" {
			target = "darwin-arm64"
		}
	case "linux":
		switch goarch {
		case "amd64":
			target = "linux-x64"
		case "arm64":
			target = "linux-arm64"
		case "arm":
			target = "linux-armv7l"
		default:
			log.Fatalf("Unsupported architecture: %s", goarch)
		}
	case "windows":
		if goarch == "amd64" {
			target = "win-x64"
		} else {
			target = "win-x86"
		}
	default:
		log.Fatalf("Unsupported OS: %s", goos)
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error fetching URL: %v", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatalf("Error parsing HTML: %v", err)
	}

	var downloadURL string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" &&
					strings.Contains(attr.Val, target) && strings.HasSuffix(attr.Val, ".zip") {
					downloadURL = "https://nodejs.org/" + attr.Val
					if config.IsDebug {
						fmt.Println("[DEBUG] Downloading Node.js from:", downloadURL)
					}
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if downloadURL == "" {
		log.Fatalf("No suitable Node.js binary found for %s-%s", goos, goarch)
	}

	return downloadURL
}

func DownloadAndExtractZip(url string, version string) (string, error) {
	tmpDir := "tmp"

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp dir: %w", err)
	}

	fileName := filepath.Base(url)
	zipPath := filepath.Join(tmpDir, fileName)

	out, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		out.Close()
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		out.Close()
		return "", fmt.Errorf("bad response status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	tempExtractDir := filepath.Join(tmpDir, "unzip_temp")
	os.RemoveAll(tempExtractDir)
	if err := unzip(zipPath, tempExtractDir); err != nil {
		return "", fmt.Errorf("failed to unzip: %w", err)
	}

	files, err := os.ReadDir(tempExtractDir)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted dir: %w", err)
	}

	if len(files) != 1 || !files[0].IsDir() {
		return "", fmt.Errorf("expected exactly one root folder in zip, found %d entries", len(files))
	}

	extractedRoot := filepath.Join(tempExtractDir, files[0].Name())

	finalDir := filepath.Join(tmpDir, version)

	os.RemoveAll(finalDir)

	if err := os.Rename(extractedRoot, finalDir); err != nil {
		return "", fmt.Errorf("failed to rename extracted folder: %w", err)
	}

	if err := os.Remove(zipPath); err != nil {
		return "", fmt.Errorf("failed to delete zip file: %w", err)
	}

	sysInstalledDir := "sys_installed"
	if err := os.MkdirAll(filepath.Join(sysInstalledDir, "node"), 0755); err != nil {
		return "", fmt.Errorf("failed to create sys_installed dir: %w", err)
	}

	finalInstalledPath := filepath.Join(sysInstalledDir, "node", version)

	os.RemoveAll(finalInstalledPath)

	if err := os.Rename(finalDir, finalInstalledPath); err != nil {
		return "", fmt.Errorf("failed to move folder to sys_installed: %w", err)
	}

	os.RemoveAll(tempExtractDir)

	return finalInstalledPath, nil
}

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func CreateDirSymlink(link string, target string) error {

	if _, err := os.Lstat(link); err == nil {
		err = os.Remove(link)
		if err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	if runtime.GOOS != "windows" {
		// On non-Windows, just create a normal symlink
		return os.Symlink(target, link)
	}

	// On Windows, use os.Symlink but need elevated privileges to create directory symlinks
	// The second argument to os.Symlink for Windows is the target.
	// The third argument is whether it is a directory (os.Symlink does not take a third argument,
	// so we rely on os.Symlink inferring target type or use syscall)
	// Go 1.16+ os.Symlink supports directory symlink creation if privileges allow.

	err := os.Symlink(target, link)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

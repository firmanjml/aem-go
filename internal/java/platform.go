package java

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type AzulPackage struct {
	DownloadURL string `json:"download_url"`
	JavaVersion []int  `json:"java_version"`
	Name        string `json:"name"`
}

func DownloadAndExtractJDK(javaVersion string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var target string

	switch goarch {
	case "386":
		target = "x86"
	case "amd64":
		target = "x64"
	case "amd64p32":
		target = "x64"
	case "arm":
		target = "arm"
	case "arm64":
		target = "aarch64"
	case "arm64be":
		target = "aarch64"
	case "armbe":
		target = "aarch32"
	case "loong64":
		target = "loong64"
	case "mips":
		target = "mips"
	case "mips64":
		target = "mips64"
	case "mips64le":
		target = "mips64le"
	case "mips64p32":
		target = "mips64p32"
	case "mips64p32le":
		target = "mips64p32le"
	case "mipsle":
		target = "mipsle"
	case "ppc":
		target = "ppc32"
	case "ppc64":
		target = "ppc64"
	case "ppc64le":
		target = "ppc64le"
	case "riscv":
		target = "riscv"
	case "riscv64":
		target = "riscv64"
	case "s390":
		target = "s390"
	case "s390x":
		target = "s390x"
	case "sparc":
		target = "sparc32"
	case "sparc64":
		target = "sparc64"
	case "wasm":
		target = "wasm"
	}

	api := fmt.Sprintf("https://api.azul.com/metadata/v1/zulu/packages/?java_version=%s&arch=%s&os=%s&archive_type=zip&java_package_type=jdk", javaVersion, target, goos)

	resp, err := http.Get(api)
	if err != nil {
		return "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API response error: %s", resp.Status)
	}

	var packages []AzulPackage
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(packages) == 0 {
		return "", fmt.Errorf("no packages found for the given parameters")
	}

	pkg := packages[0]
	url := pkg.DownloadURL

	parts := make([]string, len(pkg.JavaVersion))
	for i, v := range pkg.JavaVersion {
		parts[i] = strconv.Itoa(v)
	}
	versionStr := strings.Join(parts, ".")

	tmpDir := "tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create tmp dir: %w", err)
	}

	zipPath := filepath.Join(tmpDir, pkg.Name)
	out, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer out.Close()

	downloadResp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download JDK zip: %w", err)
	}
	defer downloadResp.Body.Close()

	if _, err := io.Copy(out, downloadResp.Body); err != nil {
		return "", fmt.Errorf("failed to write JDK zip: %w", err)
	}

	tempExtractDir := filepath.Join(tmpDir, "unzip_temp")
	os.RemoveAll(tempExtractDir)
	if err := unzip(zipPath, tempExtractDir); err != nil {
		return "", fmt.Errorf("failed to unzip: %w", err)
	}

	files, err := os.ReadDir(tempExtractDir)
	if err != nil {
		return "", fmt.Errorf("failed to read unzip dir: %w", err)
	}
	if len(files) != 1 || !files[0].IsDir() {
		return "", fmt.Errorf("expected one directory in zip, found %d", len(files))
	}

	extractedRoot := filepath.Join(tempExtractDir, files[0].Name())
	finalDir := filepath.Join("sys_installed", "java", versionStr)

	os.RemoveAll(finalDir)
	if err := os.MkdirAll(filepath.Dir(finalDir), 0755); err != nil {
		return "", fmt.Errorf("failed to create sys_installed/java dir: %w", err)
	}

	if err := os.Rename(extractedRoot, finalDir); err != nil {
		return "", fmt.Errorf("failed to move extracted folder: %w", err)
	}

	os.Remove(zipPath)
	os.RemoveAll(tempExtractDir)

	return finalDir, nil
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

package archiver

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
	"aem/pkg/process"
	"aem/pkg/progress"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ZipExtractor struct {
	logger *logger.Logger
}

func NewZipExtractor(logger *logger.Logger) *ZipExtractor {
	return &ZipExtractor{logger: logger}
}

func (ze *ZipExtractor) Extract(src, dest string) error {
	ze.logger.Debug("Extracting zip file: %s to %s", src, dest)

	r, err := zip.OpenReader(src)
	if err != nil {
		return errors.NewExtractionError("failed to open zip file", err)
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return errors.NewExtractionError("failed to create destination directory", err)
	}

	tracker := progress.New("Extracting "+filepath.Base(src), int64(len(r.File)))
	defer tracker.Finish()

	for _, f := range r.File {
		if err := process.Context().Err(); err != nil {
			return errors.NewExtractionError("zip extraction canceled", err)
		}
		if err := ze.extractFile(f, dest); err != nil {
			return err
		}
		tracker.Add(1)
	}

	ze.logger.Debug("Successfully extracted zip file")
	return nil
}

func (ze *ZipExtractor) extractFile(f *zip.File, destBase string) error {
	fpath := filepath.Join(destBase, f.Name)

	// Security check: prevent zip slip
	if !strings.HasPrefix(fpath, filepath.Clean(destBase)+string(os.PathSeparator)) {
		return errors.NewExtractionError("illegal file path detected", fmt.Errorf("path: %s", fpath))
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(fpath, f.Mode())
	}

	if f.Mode()&os.ModeSymlink != 0 {
		return ze.extractSymlink(f, fpath)
	}

	if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
		return errors.NewExtractionError("failed to create file directory", err)
	}

	if err := os.RemoveAll(fpath); err != nil {
		return errors.NewExtractionError("failed to remove existing extracted file", err)
	}

	outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return errors.NewExtractionError("failed to create extracted file", err)
	}
	defer outFile.Close()

	rc, err := f.Open()
	if err != nil {
		return errors.NewExtractionError("failed to open file in zip", err)
	}
	defer rc.Close()

	_, err = io.Copy(outFile, rc)
	if err != nil {
		return errors.NewExtractionError("failed to write extracted file", err)
	}

	if err := os.Chmod(fpath, f.Mode()); err != nil {
		return errors.NewExtractionError("failed to apply extracted file permissions", err)
	}

	return nil
}

func (ze *ZipExtractor) extractSymlink(f *zip.File, linkPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return errors.NewExtractionError("failed to create symlink directory", err)
	}

	rc, err := f.Open()
	if err != nil {
		return errors.NewExtractionError("failed to open symlink in zip", err)
	}
	defer rc.Close()

	targetBytes, err := io.ReadAll(rc)
	if err != nil {
		return errors.NewExtractionError("failed to read symlink target from zip", err)
	}

	target := strings.TrimSpace(string(targetBytes))
	if target == "" {
		return errors.NewExtractionError("empty symlink target in zip", nil)
	}

	if err := os.RemoveAll(linkPath); err != nil {
		return errors.NewExtractionError("failed to remove existing symlink target path", err)
	}

	if err := os.Symlink(target, linkPath); err != nil {
		return errors.NewExtractionError("failed to create symlink from zip", err)
	}

	return nil
}

package archiver

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
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

	for _, f := range r.File {
		if err := ze.extractFile(f, dest); err != nil {
			return err
		}
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

	if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
		return errors.NewExtractionError("failed to create file directory", err)
	}

	outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
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

	return nil
}

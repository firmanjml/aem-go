package archiver

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
	"aem/pkg/process"
	"aem/pkg/progress"
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TarGzExtractor struct {
	logger *logger.Logger
}

func NewTarGzExtractor(logger *logger.Logger) *TarGzExtractor {
	return &TarGzExtractor{logger: logger}
}

func (te *TarGzExtractor) Extract(src, dest string) error {
	te.logger.Debug("Extracting tar.gz file: %s to %s", src, dest)

	totalEntries, err := te.countEntries(src)
	if err != nil {
		return err
	}

	file, err := os.Open(src)
	if err != nil {
		return errors.NewExtractionError("failed to open tar.gz file", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return errors.NewExtractionError("failed to create gzip reader", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return errors.NewExtractionError("failed to create destination directory", err)
	}

	tracker := progress.New("Extracting "+filepath.Base(src), totalEntries)
	defer tracker.Finish()

	for {
		if err := process.Context().Err(); err != nil {
			return errors.NewExtractionError("tar.gz extraction canceled", err)
		}

		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.NewExtractionError("failed reading tar entry", err)
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return errors.NewExtractionError("illegal file path detected in tar.gz archive", nil)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return errors.NewExtractionError("failed to create directory from tar.gz archive", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return errors.NewExtractionError("failed to create file directory from tar.gz archive", err)
			}
			if err := os.RemoveAll(target); err != nil {
				return errors.NewExtractionError("failed to remove existing tar.gz path", err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return errors.NewExtractionError("failed to create file from tar.gz archive", err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return errors.NewExtractionError("failed to write tar.gz file", err)
			}
			if err := out.Close(); err != nil {
				return errors.NewExtractionError("failed to close extracted tar.gz file", err)
			}
			if err := os.Chmod(target, os.FileMode(header.Mode)); err != nil {
				return errors.NewExtractionError("failed to apply tar.gz file permissions", err)
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return errors.NewExtractionError("failed to create symlink directory from tar.gz archive", err)
			}
			if err := os.RemoveAll(target); err != nil {
				return errors.NewExtractionError("failed to remove existing tar.gz symlink path", err)
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return errors.NewExtractionError("failed to create symlink from tar.gz archive", err)
			}
		}

		tracker.Add(1)
	}

	return nil
}

func (te *TarGzExtractor) countEntries(src string) (int64, error) {
	file, err := os.Open(src)
	if err != nil {
		return 0, errors.NewExtractionError("failed to open tar.gz file for counting", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return 0, errors.NewExtractionError("failed to create gzip reader for counting", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var count int64
	for {
		if err := process.Context().Err(); err != nil {
			return 0, errors.NewExtractionError("tar.gz entry count canceled", err)
		}
		_, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, errors.NewExtractionError("failed reading tar entry count", err)
		}
		count++
	}

	return count, nil
}

package downloader

import (
	"aem/pkg/errors"
	"aem/pkg/logger"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type Downloader struct {
	logger *logger.Logger
	client *http.Client
}

func New(logger *logger.Logger) *Downloader {
	return &Downloader{
		logger: logger,
		client: &http.Client{},
	}
}

func (d *Downloader) Download(url, destPath string) error {
	d.logger.Info("Downloading from: %s", url)

	resp, err := d.client.Get(url)
	if err != nil {
		return errors.NewDownloadError("failed to make HTTP request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.NewDownloadError("HTTP request failed with status: "+resp.Status, nil)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.NewDownloadError("failed to create destination directory", err)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return errors.NewDownloadError("failed to create destination file", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.NewDownloadError("failed to write downloaded content", err)
	}

	d.logger.Debug("Successfully downloaded to: %s", destPath)
	return nil
}

func (d *Downloader) GetHTML(url string) (io.ReadCloser, error) {
	d.logger.Debug("Fetching HTML from: %s", url)

	resp, err := d.client.Get(url)
	if err != nil {
		return nil, errors.NewDownloadError("failed to fetch HTML", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, errors.NewDownloadError("HTTP request failed with status: "+resp.Status, nil)
	}

	return resp.Body, nil
}

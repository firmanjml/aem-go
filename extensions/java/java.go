package java

import (
	"aem/internal/manager"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type JavaExtension struct {
	manager.BaseExtension
}

type JavaRelease struct {
	JavaVersion []int  `json:"java_version"`
	Name        string `json:"name"`
}

func NewJavaExtension() *JavaExtension {
	return &JavaExtension{
		BaseExtension: manager.BaseExtension{BaseUrl: "https://api.azul.com/metadata/v1/zulu/packages/"},
	}
}

func (n *JavaExtension) CheckVersion(version string) (bool, error) {
	jsonURL := fmt.Sprintf(strings.TrimSuffix(n.BaseUrl, "/")+"?archive_type=zip&arch=%s&os=%s&java_package_type=jdk&page_size=1000&availability_type=CA&java_version=%s&javafx_bundled=false", "x64", "win", version)
	resp, err := http.Get(jsonURL)

	if err != nil {
		return false, err
	}

	if resp.StatusCode == 200 {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, fmt.Errorf("failed to read JSON response: %w", err)
		}

		var releases []JavaRelease
		if err := json.Unmarshal(body, &releases); err != nil {
			return false, fmt.Errorf("failed to parse JSON: %w", err)
		}

		for _, release := range releases {
			parts := make([]string, len(release.JavaVersion))
			for i, num := range release.JavaVersion {
				parts[i] = strconv.Itoa(num)
			}
			releaseVersion := strings.Join(parts, ".")
			if releaseVersion == version {
				return true, nil
			}
		}
	}

	return false, nil
}

func (n *JavaExtension) ListVersions(version *string) ([]string, error) {
	base := strings.TrimSuffix(n.BaseUrl, "/")
	jsonURL := fmt.Sprintf("%s?archive_type=zip&arch=%s&os=%s&java_package_type=jdk&page_size=1000&availability_type=CA&javafx_bundled=false", base, "x64", "win")

	if version != nil {
		jsonURL += fmt.Sprintf("&java_version=%s", *version)
	}

	resp, err := http.Get(jsonURL)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return []string{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, fmt.Errorf("failed to read JSON response: %w", err)
	}

	var releases []JavaRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return []string{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	versions := make(map[string]struct{})
	for _, release := range releases {
		parts := make([]string, len(release.JavaVersion))
		for i, num := range release.JavaVersion {
			parts[i] = strconv.Itoa(num)
		}
		releaseVersion := strings.Join(parts, ".")
		if version == nil || strings.HasPrefix(releaseVersion, *version) {
			if _, exists := versions[releaseVersion]; !exists {
				versions[releaseVersion] = struct{}{}
			}
		}
		if len(versions) == 10 {
			break
		}
	}

	var result []string
	for v := range versions {
		result = append(result, v)
	}
	return result, nil
}

func (n *JavaExtension) GetDownloadURL(version string) (string, error) {
	return "", fmt.Errorf("not found for version %s", version)
}

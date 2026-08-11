package java

import (
	"aem/internal/manager"
	"aem/internal/platform"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type JavaExtension struct {
	manager.BaseExtension
	platform platform.Info
}

type JavaRelease struct {
	JavaVersion []int  `json:"java_version"`
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

func NewJavaExtension() *JavaExtension {
	return newJavaExtension("https://api.azul.com/metadata/v1/zulu/packages/", platform.GetInfo())
}

func newJavaExtension(baseURL string, target platform.Info) *JavaExtension {
	return &JavaExtension{
		BaseExtension: manager.BaseExtension{BaseUrl: baseURL},
		platform:      target,
	}
}

func (n *JavaExtension) CheckVersion(version string) (bool, error) {
	version = normalizedVersion(version)
	releases, err := n.fetchReleases(version)
	if err != nil {
		return false, err
	}
	for _, release := range releases {
		if releaseVersion(release) == version {
			return true, nil
		}
	}
	return false, nil
}

func (n *JavaExtension) ListVersions(version *string) ([]string, error) {
	requested := ""
	if version != nil {
		requested = normalizedVersion(*version)
	}
	releases, err := n.fetchReleases(requested)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	versions := make([]string, 0, 10)
	for _, release := range releases {
		current := releaseVersion(release)
		if requested != "" && !strings.HasPrefix(current, requested) {
			continue
		}
		if _, exists := seen[current]; exists {
			continue
		}
		seen[current] = struct{}{}
		versions = append(versions, current)
		if len(versions) == 10 {
			break
		}
	}
	return versions, nil
}

func (n *JavaExtension) GetDownloadURL(version string) (string, error) {
	version = normalizedVersion(version)
	releases, err := n.fetchReleases(version)
	if err != nil {
		return "", err
	}
	for _, release := range releases {
		if releaseVersion(release) == version && release.DownloadURL != "" {
			return release.DownloadURL, nil
		}
	}
	return "", fmt.Errorf("Zulu JDK version %s not found for this platform", version)
}

func (n *JavaExtension) fetchReleases(javaVersion string) ([]JavaRelease, error) {
	osName, arch, err := n.platform.AzulTarget()
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(n.BaseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid Azul metadata URL: %w", err)
	}
	query := endpoint.Query()
	query.Set("archive_type", "zip")
	query.Set("arch", arch)
	query.Set("os", osName)
	query.Set("java_package_type", "jdk")
	query.Set("page_size", "1000")
	query.Set("availability_types", "CA")
	query.Set("release_status", "ga")
	query.Set("javafx_bundled", "false")
	if javaVersion != "" {
		query.Set("java_version", javaVersion)
	}
	endpoint.RawQuery = query.Encode()

	resp, err := http.Get(endpoint.String()) // #nosec G107 -- provider endpoint is configured by the extension.
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON response: %w", err)
	}
	var releases []JavaRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return releases, nil
}

func normalizedVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

func releaseVersion(release JavaRelease) string {
	parts := make([]string, len(release.JavaVersion))
	for i, value := range release.JavaVersion {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ".")
}

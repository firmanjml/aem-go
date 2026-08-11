package node

import (
	"aem/internal/manager"
	"aem/internal/platform"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type NodeExtension struct {
	manager.BaseExtension
	platform platform.Info
}

type NodeJSRelease struct {
	Version string   `json:"version"`
	Date    string   `json:"date"`
	Files   []string `json:"files"`
}

func NewNodeExtension() *NodeExtension {
	return newNodeExtension("https://nodejs.org/dist", platform.GetInfo())
}

func newNodeExtension(baseURL string, target platform.Info) *NodeExtension {
	return &NodeExtension{
		BaseExtension: manager.BaseExtension{BaseUrl: baseURL},
		platform:      target,
	}
}

func (n *NodeExtension) CheckVersion(version string) (bool, error) {
	version = normalizeVersion(version)
	releases, err := n.fetchReleases()
	if err != nil {
		return false, err
	}
	for _, release := range releases {
		if release.Version == version {
			return n.supportsTarget(release), nil
		}
	}
	return false, nil
}

func (n *NodeExtension) ListVersions(version *string) ([]string, error) {
	if version != nil {
		v := normalizeVersion(*version)
		version = &v
	}

	releases, err := n.fetchReleases()
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, 10)
	for _, release := range releases {
		if !n.supportsTarget(release) {
			continue
		}
		if version == nil || strings.HasPrefix(release.Version, *version) {
			versions = append(versions, strings.TrimPrefix(release.Version, "v"))
		}
		if len(versions) == 10 {
			break
		}
	}
	return versions, nil
}

func (n *NodeExtension) GetDownloadURL(version string) (string, error) {
	version = normalizeVersion(version)
	target, err := n.platform.NodeTarget()
	if err != nil {
		return "", err
	}
	metadataKey, err := n.platform.NodeMetadataKey()
	if err != nil {
		return "", err
	}
	releases, err := n.fetchReleases()
	if err != nil {
		return "", err
	}
	for _, release := range releases {
		if release.Version != version {
			continue
		}
		if !contains(release.Files, metadataKey) {
			return "", fmt.Errorf("Node.js version %s has no binary for %s", version, target)
		}
		extension := ".tar.gz"
		if n.platform.OS == "windows" {
			extension = ".zip"
		}
		return fmt.Sprintf("%s/%s/node-%s-%s%s", strings.TrimSuffix(n.BaseUrl, "/"), version, version, target, extension), nil
	}
	return "", fmt.Errorf("Node.js version %s not found", version)
}

func (n *NodeExtension) fetchReleases() ([]NodeJSRelease, error) {
	jsonURL := strings.TrimSuffix(n.BaseUrl, "/") + "/index.json"
	resp, err := http.Get(jsonURL) // #nosec G107 -- provider endpoint is configured by the extension.
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
	var releases []NodeJSRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return releases, nil
}

func (n *NodeExtension) supportsTarget(release NodeJSRelease) bool {
	target, err := n.platform.NodeMetadataKey()
	if err != nil {
		return false
	}
	// Node's old index records did not include files. Keep historical versions
	// discoverable; GetDownloadURL remains the definitive availability check.
	return len(release.Files) == 0 || contains(release.Files, target)
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

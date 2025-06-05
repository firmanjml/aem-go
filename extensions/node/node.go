package node

import (
	"aem/internal/manager"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type NodeExtension struct {
	manager.BaseExtension
}

type NodeJSRelease struct {
	Version string   `json:"version"`
	Date    string   `json:"date"`
	Files   []string `json:"files"`
}

func NewNodeExtension() *NodeExtension {
	return &NodeExtension{
		BaseExtension: manager.BaseExtension{BaseUrl: "https://www.nodejs.org/dist"},
	}
}

func (n *NodeExtension) CheckVersion(version string) (bool, error) {
	// Ensure version starts with 'v'
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	jsonURL := strings.TrimSuffix(n.BaseUrl, "/") + "/index.json"
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

		var releases []NodeJSRelease
		if err := json.Unmarshal(body, &releases); err != nil {
			return false, fmt.Errorf("failed to parse JSON: %w", err)
		}

		for _, release := range releases {
			if release.Version == version {
				return true, nil
			}
		}
	}

	return false, nil
}

func (n *NodeExtension) ListVersions(version *string) ([]string, error) {
	if version != nil {
		if !strings.HasPrefix(*version, "v") {
			v := "v" + *version
			version = &v
		}
	}

	jsonURL := strings.TrimSuffix(n.BaseUrl, "/") + "/index.json"
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

	var releases []NodeJSRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return []string{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	var versions []string
	for _, release := range releases {

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

	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	exists, err := n.CheckVersion(version)
	if err != nil {
		return "", fmt.Errorf("failed to check version: %w", err)
	}
	if !exists {
		return "", fmt.Errorf("version %s not found", version)
	}

	jsonURL := strings.TrimSuffix(n.BaseUrl, "/")
	resp, err := http.Get(jsonURL)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			var releases []NodeJSRelease
			if json.Unmarshal(body, &releases) == nil {
				for _, release := range releases {
					if release.Version == version {
						// Check if target file exists in the files list
						v := release.Version
						a := "x64"
						return n.BaseUrl + "/" + v + "/node-" + v + "-win-" + a + ".zip", nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("not found for version %s", version)
}

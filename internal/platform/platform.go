package platform

import (
	"fmt"
	"runtime"
	"strings"
)

type Info struct {
	OS   string
	Arch string
}

func GetInfo() Info {
	return Info{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
}

func (p Info) MapArchitecture() string {
	switch p.Arch {
	case "386":
		return "x86"
	case "amd64", "amd64p32":
		return "x64"
	case "arm":
		return "arm"
	case "arm64", "arm64be":
		return "aarch64"
	case "armbe":
		return "aarch32"
	default:
		return p.Arch
	}
}

// NodeTarget returns the platform token used in Node.js distribution names.
// It intentionally rejects unknown targets so callers do not accidentally
// download a binary for a different architecture.
func (p Info) NodeTarget() (string, error) {
	switch p.OS {
	case "darwin":
		switch p.Arch {
		case "amd64":
			return "darwin-x64", nil
		case "arm64":
			return "darwin-arm64", nil
		}
	case "linux":
		switch p.Arch {
		case "amd64":
			return "linux-x64", nil
		case "arm64":
			return "linux-arm64", nil
		case "arm":
			return "linux-armv7l", nil
		case "ppc64le", "s390x", "riscv64":
			return "linux-" + p.Arch, nil
		}
	case "windows":
		switch p.Arch {
		case "amd64":
			return "win-x64", nil
		case "386":
			return "win-x86", nil
		case "arm64":
			return "win-arm64", nil
		}
	}
	return "", fmt.Errorf("unsupported Node.js platform %s/%s", p.OS, p.Arch)
}

// GetNodeTarget is kept for compatibility with callers that previously used
// the permissive helper. New download code should use NodeTarget so an
// unsupported target returns a useful error instead of an invalid URL.
func (p Info) GetNodeTarget() string {
	target, _ := p.NodeTarget()
	return target
}

// NodeMetadataKey returns the availability token used by Node's index.json.
// Those tokens differ from distribution filenames on macOS (osx vs darwin)
// and Windows (the archive format is part of the key).
func (p Info) NodeMetadataKey() (string, error) {
	target, err := p.NodeTarget()
	if err != nil {
		return "", err
	}
	switch p.OS {
	case "darwin":
		return "osx-" + strings.TrimPrefix(target, "darwin-") + "-tar", nil
	case "windows":
		return target + "-zip", nil
	default:
		return target, nil
	}
}

// AzulTarget returns the operating-system and architecture values accepted by
// the Azul Zulu Metadata API. Azul calls Apple Silicon "arm" rather than the
// Go architecture name "arm64" and calls macOS "macos".
func (p Info) AzulTarget() (osName, arch string, err error) {
	switch p.OS {
	case "darwin":
		osName = "macos"
	case "linux":
		osName = "linux"
	case "windows":
		osName = "windows"
	default:
		return "", "", fmt.Errorf("unsupported Azul Zulu operating system %s", p.OS)
	}

	switch p.Arch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x86"
	case "arm", "arm64":
		arch = "arm"
	default:
		return "", "", fmt.Errorf("unsupported Azul Zulu architecture %s", p.Arch)
	}
	return osName, arch, nil
}

package platform

import (
	"runtime"
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

func (p Info) GetNodeTarget() string {
	switch p.OS {
	case "darwin":
		if p.Arch == "arm64" {
			return "darwin-arm64"
		}
		return "darwin-x64"
	case "linux":
		switch p.Arch {
		case "amd64":
			return "linux-x64"
		case "arm64":
			return "linux-arm64"
		case "arm":
			return "linux-armv7l"
		default:
			return "linux-" + p.Arch
		}
	case "windows":
		if p.Arch == "amd64" {
			return "win-x64"
		}
		return "win-x86"
	default:
		return p.OS + "-" + p.Arch
	}
}

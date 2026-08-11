package platform

import "testing"

func TestNodeTarget(t *testing.T) {
	tests := []struct {
		info Info
		want string
	}{
		{Info{OS: "darwin", Arch: "amd64"}, "darwin-x64"},
		{Info{OS: "darwin", Arch: "arm64"}, "darwin-arm64"},
		{Info{OS: "linux", Arch: "amd64"}, "linux-x64"},
		{Info{OS: "linux", Arch: "arm64"}, "linux-arm64"},
		{Info{OS: "linux", Arch: "arm"}, "linux-armv7l"},
		{Info{OS: "windows", Arch: "386"}, "win-x86"},
		{Info{OS: "windows", Arch: "amd64"}, "win-x64"},
		{Info{OS: "windows", Arch: "arm64"}, "win-arm64"},
	}
	for _, test := range tests {
		got, err := test.info.NodeTarget()
		if err != nil || got != test.want {
			t.Errorf("NodeTarget(%+v) = %q, %v; want %q, nil", test.info, got, err, test.want)
		}
	}
	if _, err := (Info{OS: "linux", Arch: "mips64"}).NodeTarget(); err == nil {
		t.Error("NodeTarget() succeeded for an unsupported target")
	}
}

func TestNodeMetadataKey(t *testing.T) {
	tests := []struct {
		info Info
		want string
	}{
		{Info{OS: "darwin", Arch: "amd64"}, "osx-x64-tar"},
		{Info{OS: "darwin", Arch: "arm64"}, "osx-arm64-tar"},
		{Info{OS: "linux", Arch: "arm64"}, "linux-arm64"},
		{Info{OS: "windows", Arch: "amd64"}, "win-x64-zip"},
	}
	for _, test := range tests {
		got, err := test.info.NodeMetadataKey()
		if err != nil || got != test.want {
			t.Errorf("NodeMetadataKey(%+v) = %q, %v; want %q, nil", test.info, got, err, test.want)
		}
	}
}

func TestAzulTarget(t *testing.T) {
	tests := []struct {
		info     Info
		wantOS   string
		wantArch string
	}{
		{Info{OS: "darwin", Arch: "arm64"}, "macos", "arm"},
		{Info{OS: "darwin", Arch: "amd64"}, "macos", "x64"},
		{Info{OS: "linux", Arch: "amd64"}, "linux", "x64"},
		{Info{OS: "windows", Arch: "386"}, "windows", "x86"},
	}
	for _, test := range tests {
		gotOS, gotArch, err := test.info.AzulTarget()
		if err != nil || gotOS != test.wantOS || gotArch != test.wantArch {
			t.Errorf("AzulTarget(%+v) = %q, %q, %v; want %q, %q, nil", test.info, gotOS, gotArch, err, test.wantOS, test.wantArch)
		}
	}
}

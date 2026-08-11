//go:build !windows

package filesystem

import "os"

func isReplaceableLink(_ string, info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

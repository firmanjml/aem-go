//go:build windows

package filesystem

import (
	"os"

	"golang.org/x/sys/windows"
)

func isReplaceableLink(path string, info os.FileInfo) bool {
	if info.Mode()&os.ModeSymlink != 0 {
		return true
	}

	// Go reports a directory junction as a directory rather than a symlink.
	// Its reparse-point attribute lets AEM replace the junction it creates when
	// symlink creation is unavailable, while ordinary files and directories
	// remain protected.
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attributes, err := windows.GetFileAttributes(pathPointer)
	return err == nil && attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

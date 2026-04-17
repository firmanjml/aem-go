package state

import (
	"os"
)

type OSReader struct{}

func NewOSReader() *OSReader {
	return &OSReader{}
}

func (r *OSReader) Readlink(name string) (string, error) {
	return os.Readlink(name)
}

func isNotExist(err error) bool {
	return os.IsNotExist(err)
}

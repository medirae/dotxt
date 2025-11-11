package utils

import (
	"fmt"
	"path/filepath"
	"syscall"
)

func MkPtr[T any](val T) *T {
	return &val
}

// tries to use device:inode key; falls back to absolute path.
func GetFileCanonicalKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	var st syscall.Stat_t
	if err := syscall.Stat(abs, &st); err == nil {
		return fmt.Sprintf("%d:%d", st.Dev, st.Ino)
	}
	return abs
}

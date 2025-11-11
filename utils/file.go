package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FileExists(path string) bool {
	path, err := NormalizePath(path)
	if err != nil {
		panic(err)
	}
	_, err = os.Stat(path)
	if err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		panic(fmt.Errorf("file existence check error: %w", err))
	}
}

func NormalizePath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(homeDir, RuneSlice(path, 1))
	}
	return path, nil
}

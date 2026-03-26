package utils

import (
	"errors"
	"os"
)

func CheckFileExists(path string) (exists bool, isDir bool) {
	info, err := os.Stat(path)
	if err == nil {
		return true, info.IsDir()
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, false
	}

	return false, false
}

package utils

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
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

func OpenInExternalApp(path string) error {
	switch runtime.GOOS {
	case "darwin", "ios":
		return runCmd("open", path)
	case "windows":
		return runCmd("explorer", path)
	default:
		// linux, unix flavors.
		return runCmd("xdg-open", path)
	}
}

func runCmd(cmdName string, arg ...string) error {
	cmd := exec.Command(cmdName, arg...)
	return cmd.Run()
}

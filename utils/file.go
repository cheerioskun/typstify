package utils

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func CheckExists(file string) bool {
	_, err := os.Stat(file)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	log.Println("cannot check if executable exist", err)
	return true
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

package utils

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// LookupExecutable looks up the executable dir and add it the the PATH of the
// current process.
func LookupExecutable(exeName string, externalDir string) bool {
	binDir := ""
	if externalDir != "" {
		binDir = externalDir
	} else {
		exePath, err := os.Executable()
		if err == nil {
			binDir = filepath.Dir(exePath)
		}
	}

	// Fallback to the process root dir.
	if binDir == "" || !checkExists(binDir, exeName) {
		binDir, _ = filepath.Abs(".") // all 3 main OSes are supported.
	}

	if checkExists(binDir, exeName) {
		// update permission to ensure it can be picked up by os.LookPath.
		os.Chmod(filepath.Join(binDir, exeName), 0755)

		pathEnv := os.Getenv("PATH")
		if runtime.GOOS == "windows" {
			os.Setenv("PATH", binDir+";"+pathEnv)
		} else {
			// linux or macos or any other OS have the same format.
			os.Setenv("PATH", binDir+":"+pathEnv)
		}
	}

	if _, err := exec.LookPath(exeName); err != nil {
		log.Printf("No %s found after searching PATH: %s", exeName, os.Getenv("PATH"))
		return false
	}

	return true
}

func checkExists(binDir string, exeName string) bool {
	_, err := os.Stat(filepath.Join(binDir, exeName))
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	log.Println("cannot check if executable exist", err)
	return true
}

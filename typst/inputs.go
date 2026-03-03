package typst

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const sysInputsFile = "sys-inputs.json"

func LoadInputs(dir string, createIfNotExists bool) (map[string]string, error) {
	inputsFile := filepath.Join(dir, sysInputsFile)
	file, err := os.Open(inputsFile)
	if err != nil {
		// Check if the file exists
		if _, err := os.Stat(inputsFile); errors.Is(err, os.ErrNotExist) && createIfNotExists {
			writeErr := os.WriteFile(inputsFile, []byte("{\n}"), 0644)
			if writeErr != nil {
				log.Printf("create file %s failed: %v", inputsFile, writeErr)
				return map[string]string{}, writeErr
			}
			// try to open the newly created file.
			file, err = os.Open(inputsFile)
		} else {
			log.Println("read sys-inputs.json failed: ", err)
			return map[string]string{}, nil
		}
	}

	var inputs map[string]string
	err = json.NewDecoder(file).Decode(&inputs)
	if err != nil {
		errors.Join(err, fmt.Errorf("invalid sys.inputs file: %s, content must be a valid json: ", inputsFile))
	}

	return inputs, nil
}

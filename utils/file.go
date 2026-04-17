package utils

import (
	"errors"
	"io"
	"os"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/lexers"
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

// ported from https://cs.opensource.google/go/x/tools/+/refs/tags/v0.26.0:godoc/util/util.go;l=69
func IsTextFile(filePath string) bool {
	if lexer := lexers.Match(filePath); lexer != nil {
		return true
	}

	// the extension is not known; read an initial chunk
	// of the file and check if it looks like text
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	var buf [1024]byte
	n, err := f.Read(buf[0:])
	if err != nil {
		if err == io.EOF && n == 0 {
			return true
		}
		return false
	}

	// return IsText(buf[0:n])

	//  reports whether a significant prefix of buf looks like correct UTF-8;
	// that is, if it is likely that s is human-readable text.
	for i, c := range string(buf[0:n]) {
		if i+utf8.UTFMax > len(buf) {
			// last char may be incomplete - ignore
			break
		}
		if c == 0xFFFD || c < ' ' && c != '\n' && c != '\t' && c != '\f' {
			// decoding error or control character - not a text file
			return false
		}
	}
	return true
}

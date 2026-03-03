package typst

import (
	"strings"
	"testing"
)

func TestGetTypstVersion(t *testing.T) {
	ver := VersionCmd()
	if ver == "" || strings.HasPrefix(ver, "typst") {
		t.Fail()
	}
}

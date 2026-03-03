package settings

import (
	"os"
	"testing"
)

func TestModelSave(t *testing.T) {
	db := newSettings("./", nil)
	general := db.General()

	t.Log("general: ", general)

	general.Language = "cn/zh"

	err := general.Save()
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	t.Cleanup(func() {
		db.db.Close()
		// os.Remove("./settings.db")
	})

}

func TestModelGetDefault(t *testing.T) {
	db := newSettings("./", nil)
	general := db.General()
	typst := db.Typst()

	t.Log("general: ", general)
	t.Log("typst: ", typst)
	t.Log("editor: ", db.Editor())

	t.Cleanup(func() {
		db.db.Close()
		os.Remove("./settings.db")
	})

}

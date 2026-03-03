package fonts

import (
	_ "embed"
)

//go:embed Hack-Regular.ttf
var HackRegular []byte

//go:embed NotoSansMath-Regular.ttf
var NotoSansMath []byte

//go:embed NotoEmoji-VariableFont_wght.ttf
var NotoEmojiVarWght []byte

//go:embed RobotoMono-VariableFont_wght.ttf
var RobotoMonoVar []byte

var Embedded = [][]byte{
	HackRegular,
	NotoSansMath,
	NotoEmojiVarWght,
	RobotoMonoVar,
}

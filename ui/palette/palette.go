package palette

import (
	"fmt"
	"slices"

	"github.com/oligo/gioview/misc"
	th "github.com/oligo/gioview/theme"
)

type UIPalette struct {
	th.Palette
	// chroma style name, see https://xyproto.github.io/splash/docs/ for the full list
	CodeColorScheme string
}

var themeMap = map[string]UIPalette{
	"Default Light": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x383A42),
			Bg:            misc.HexColor(0xFCFCFC),
			ContrastFg:    misc.HexColor(0xFFFFFF),
			ContrastBg:    misc.HexColor(0x007AFF),
			Bg2:           misc.HexColor(0xF2F2F2),
			HoverAlpha:    36,
			SelectedAlpha: 64,
		},
		CodeColorScheme: "monokailight",
	},

	"Default Dark": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0xABB2BF),
			Bg:            misc.HexColor(0x282C34),
			ContrastFg:    misc.HexColor(0x1F1B24),
			ContrastBg:    misc.HexColor(0xABB2BF),
			Bg2:           misc.HexColor(0x21252B),
			HoverAlpha:    36,
			SelectedAlpha: 48,
		},
		CodeColorScheme: "doom-one",
	},

	"Solarized Light": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0x657b83),
			Bg:            misc.HexColor(0xfdf6e3),
			ContrastFg:    misc.HexColor(0xfdf6e3),
			ContrastBg:    misc.HexColor(0xb58900),
			Bg2:           misc.HexColor(0xeee8d5),
			HoverAlpha:    36,
			SelectedAlpha: 48,
		},
		CodeColorScheme: "solarized-light",
	},

	"Solarized Dark": {
		Palette: th.Palette{
			Fg:            misc.HexColor(0xeee8d5),
			Bg:            misc.HexColor(0x002b36),
			ContrastFg:    misc.HexColor(0x002b36),
			ContrastBg:    misc.HexColor(0x839496),
			Bg2:           misc.HexColor(0x002b36),
			HoverAlpha:    24,
			SelectedAlpha: 36,
		},
		CodeColorScheme: "solarized-dark",
	},
}

func ThemeNames() []string {
	var names []string
	for k := range themeMap {
		names = append(names, k)
	}

	slices.SortFunc[[]string, string](names, func(a, b string) int {
		if a == "Default Light" {
			return -1
		}
		if b == "Default Light" {
			return 1
		}

		if a >= b {
			return 1
		} else {
			return -1
		}
	})

	return names
}

func ThemeConfig(themeName string) (UIPalette, error) {
	p, ok := themeMap[themeName]
	if !ok {
		return UIPalette{}, fmt.Errorf("no theme found for name: %s", themeName)
	}

	return p, nil
}

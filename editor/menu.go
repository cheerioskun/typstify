package editor

import (
	"fmt"
	"io"
	"strings"

	"gioui.org/io/clipboard"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/menu"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gvcode"
	"looz.ws/typstify/i18n"
)

func EditorMenuOptions(gtx C, editor *TextEditor) [][]menu.MenuOption {
	return [][]menu.MenuOption{
		{
			// copy
			{
				OnClicked: func() error {
					selectedTxt := editor.state.SelectedText()
					if selectedTxt != "" {
						gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(selectedTxt))})
					}
					return nil
				},

				Layout: func(gtx C, th *theme.Theme) D {
					return layoutOption(gtx, th, i18n.Translate("Copy"), "C")
				},
			},
			// cut
			{
				OnClicked: func() error {
					selectedTxt := editor.state.SelectedText()
					if selectedTxt != "" {
						gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(selectedTxt))})
						editor.state.Delete(1)
					}
					return nil
				},

				Layout: func(gtx C, th *theme.Theme) D {
					return layoutOption(gtx, th, i18n.Translate("Cut"), "X")
				},
			},
			// paste
			{
				OnClicked: func() error {
					if editor.state.Mode() != gvcode.ModeReadOnly {
						// the other part is in editor processKey method.
						gtx.Execute(clipboard.ReadCmd{Tag: editor.state})
						gtx.Execute(op.InvalidateCmd{})
					}
					return nil
				},

				Layout: func(gtx C, th *theme.Theme) D {
					return layoutOption(gtx, th, i18n.Translate("Paste"), "V")
				},
			},
			// search & replace
			{
				OnClicked: func() error {
					editor.searchbar.Show(gtx)
					return nil
				},

				Layout: func(gtx C, th *theme.Theme) D {
					return layoutOption(gtx, th, i18n.Translate("Find & Replace"), "F")
				},
			},
			// Lock & Unlock
			{
				OnClicked: func() error {
					editor.state.WithOptions(gvcode.ReadOnlyMode(editor.state.Mode() != gvcode.ModeReadOnly))
					return nil
				},

				Layout: func(gtx C, th *theme.Theme) D {
					label := i18n.Translate("Lock")
					if editor.state.Mode() == gvcode.ModeReadOnly {
						label = i18n.Translate("Unlock")
					}
					return layoutOption(gtx, th, label, "L")
				},
			},
			// wrap line
			{
				OnClicked: func() error {
					editor.state.WithOptions(gvcode.WrapLine(!editor.wrapLine))
					editor.wrapLine = !editor.wrapLine
					return nil
				},

				Layout: func(gtx C, th *theme.Theme) D {
					label := i18n.Translate("Unwrap Lines")
					if !editor.wrapLine {
						label = i18n.Translate("Wrap Lines")
					}
					return layoutOption(gtx, th, label, "W")
				},
			},
		},
	}
}

func layoutOption(gtx C, th *theme.Theme, name string, shortcutKey string) D {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceBetween,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return material.Label(th.Theme, th.TextSize, name).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(20)}.Layout),
		layout.Rigid(func(gtx C) D {
			var modKey key.Name
			if key.ModShortcut == key.ModCommand {
				modKey = key.NameCommand
			} else {
				modKey = key.NameCtrl
			}

			return material.Label(th.Theme, th.TextSize, fmt.Sprintf("%s+%s", modKey, shortcutKey)).Layout(gtx)
		}),
	)
}

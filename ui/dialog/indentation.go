package dialog

import (
	"errors"
	"fmt"
	"strconv"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	gw "github.com/oligo/gioview/widget"
	"github.com/oligo/gvcode"
	"looz.ws/typstify/i18n"
)

var ChangeIndentationDialogViewID = view.NewViewID("ChangeIndentationDialogViewID")
var _ Dialog = (*DeleteFileDialog)(nil)

var indentKindMap = map[gvcode.TabStyle]string{
	gvcode.Spaces: "Spaces",
	gvcode.Tabs:   "Tab",
}

type ChangeIndentationDialog struct {
	indentKind        widget.Enum
	tabWidthInput     gw.TextField
	convertContent    widget.Bool
	onConfirm         func(tabKind gvcode.TabStyle, tabWidth int, convertText bool) error
	indentKindChoices []layout.FlexChild
}

func NewChangeIndentationDialog() view.View {
	dialog := NewDialogModal(ChangeIndentationDialogViewID, i18n.Translate("Change File Indentation"), i18n.Translate("Confirm"))
	dialog.Dialog = &ChangeIndentationDialog{}
	return dialog
}

func (d *ChangeIndentationDialog) OnInit(intent view.Intent) error {
	indent, ok := intent.Params["indentation"]
	if !ok {
		return errors.New("missing mandatory params")
	}
	tabWidth, ok := intent.Params["tabWidth"]
	if !ok {
		return errors.New("missing mandatory params")
	}

	onConfirm, ok := intent.Params["onConfirm"]
	if !ok || onConfirm == nil {
		return errors.New("no callback provided")
	}

	d.indentKind.Value = fmt.Sprintf("%d", indent.(gvcode.TabStyle))
	d.tabWidthInput.SetText(fmt.Sprintf("%d", tabWidth.(int)))
	d.onConfirm = onConfirm.(func(tabKind gvcode.TabStyle, tabWidth int, convertText bool) error)
	return nil
}

func (d *ChangeIndentationDialog) OnConfirm() error {
	indentVal, err := strconv.Atoi(d.indentKind.Value)
	if err != nil {
		return err
	}

	indent := gvcode.TabStyle(indentVal)

	tabWidth, err := strconv.Atoi(d.tabWidthInput.Text())
	if err != nil {
		return err
	}

	if d.onConfirm != nil {
		err := d.onConfirm(indent, tabWidth, d.convertContent.Value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *ChangeIndentationDialog) LayoutBody(gtx C, th *theme.Theme) D {
	if d.indentKindChoices == nil {
		for key, name := range indentKindMap {
			name := name
			key := key
			d.indentKindChoices = append(d.indentKindChoices, layout.Rigid(func(gtx C) D {
				return material.RadioButton(th.Theme, &d.indentKind, fmt.Sprintf("%d", key), name).Layout(gtx)
			}))
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return formItem{Axis: layout.Vertical}.Layout(gtx, th, i18n.Translate("Indent with spaces or tabs"), i18n.Translate("Choose the indentation style for the current file."),
				func(gtx C) D {
					return layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx, d.indentKindChoices...)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return formItem{Axis: layout.Vertical}.Layout(gtx, th, i18n.Translate("Tab Width"), i18n.Translate("Change tab display size."),
				func(gtx C) D {
					d.tabWidthInput.Alignment = text.Start
					return d.tabWidthInput.Layout(gtx, th, "")
				})
		}),

		layout.Rigid(func(gtx C) D {
			return formItem{Axis: layout.Vertical}.Layout(gtx, th, i18n.Translate("Convert the indentation"),
				i18n.Translate("Convert the indentation from spaces to tabs or from tabs to spaces, depending on what you choosed."),
				func(gtx C) D {
					checkbox := material.CheckBox(th.Theme, &d.convertContent, i18n.Translate("Convert the indentation"))
					return checkbox.Layout(gtx)
				})
		}),
	)

}

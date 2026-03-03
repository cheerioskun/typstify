package dialog

import (
	"errors"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"looz.ws/typstify/i18n"
)

var DeleteFileDialogViewID = view.NewViewID("DeleteFileDialogView")
var _ Dialog = (*DeleteFileDialog)(nil)

type DeleteFileDialog struct {
	dest       string
	resultChan chan DialogResult[bool]
}

func NewDeleteFileDialog() view.View {
	dialog := NewDialogModal(DeleteFileDialogViewID, i18n.Translate("Delete File/Folder"), i18n.Translate("Confirm"))
	dialog.Dialog = &DeleteFileDialog{}
	return dialog
}

func (d *DeleteFileDialog) OnInit(intent view.Intent) error {
	rc, ok := intent.Params["resultChan"]
	if !ok {
		return errors.New("missing mandatory params")
	}

	dest := intent.Params["destination"]
	if dest == nil {
		return errors.New("no file/folder path provided")
	}

	d.dest = dest.(string)
	d.resultChan = rc.(chan DialogResult[bool])
	return nil
}

func (d *DeleteFileDialog) OnConfirm() error {
	d.resultChan <- DialogResult[bool]{
		Params: true,
	}
	return nil
}

func (d *DeleteFileDialog) LayoutBody(gtx C, th *theme.Theme) D {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			lb := material.Subtitle1(th.Theme, i18n.Translate("Are you sure you want to delete '%s'?", d.dest))
			lb.Alignment = text.Middle
			return lb.Layout(gtx)

		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx C) D {
			lb := material.Label(th.Theme, th.TextSize*0.8, i18n.Translate("You can restore this file from the Trash."))
			lb.Alignment = text.Middle
			return lb.Layout(gtx)
		}),
	)

}

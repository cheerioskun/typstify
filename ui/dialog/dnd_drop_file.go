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

var DndDropFileDialogViewID = view.NewViewID("DndDropFileDialogView")
var _ Dialog = (*DndDropFileDialog)(nil)

type DndDropFileDialog struct {
	src        string
	dest       string
	resultChan chan DialogResult[bool]
}

func NewDndDropFileDialog() view.View {
	dialog := NewDialogModal(DndDropFileDialogViewID, i18n.Translate("Move File/Folder"), i18n.Translate("Confirm"))
	dialog.Dialog = &DndDropFileDialog{}
	return dialog
}

func (d *DndDropFileDialog) OnInit(intent view.Intent) error {
	rc, ok := intent.Params["resultChan"]
	if !ok {
		return errors.New("missing mandatory params")
	}

	dest := intent.Params["destination"]
	if dest == nil {
		return errors.New("no file/folder path provided")
	}

	src := intent.Params["source"]
	if src == nil {
		return errors.New("no file/folder path provided")
	}

	d.dest = dest.(string)
	d.src = src.(string)
	d.resultChan = rc.(chan DialogResult[bool])
	return nil
}

func (d *DndDropFileDialog) OnConfirm() error {
	d.resultChan <- DialogResult[bool]{
		Params: true,
	}
	return nil
}

func (d *DndDropFileDialog) LayoutBody(gtx C, th *theme.Theme) D {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			lb := material.Label(th.Theme, th.TextSize, i18n.Translate("Are you sure you want to move"))
			lb.Alignment = text.Middle
			return lb.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx C) D {
			lb := material.Label(th.Theme, th.TextSize, i18n.Translate("'%s' into '%s'", d.src, d.dest))
			lb.Alignment = text.Middle
			return lb.Layout(gtx)
		}),
	)

}

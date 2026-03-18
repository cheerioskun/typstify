package dialog

import (
	"errors"
	"path/filepath"

	"gioui.org/text"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/utils"
)

var OpenWithExternalAppDialogViewID = view.NewViewID("OpenWithExternalAppDialogViewID")
var _ Dialog = (*OpenWithExternalAppDialog)(nil)

type OpenWithExternalAppDialog struct {
	targetFile string
	resultChan chan DialogResult[bool]
}

func NewOpenWithExternalAppDialog() view.View {
	dialog := NewDialogModal(OpenWithExternalAppDialogViewID, i18n.Translate("Open File With..."), i18n.Translate("Confirm"))
	dialog.Dialog = &OpenWithExternalAppDialog{}
	return dialog
}

func (d *OpenWithExternalAppDialog) OnInit(intent view.Intent) error {
	targetFile, ok := intent.Params["path"]
	if !ok {
		return errors.New("missing target file")
	}

	d.targetFile = targetFile.(string)
	return nil
}

func (d *OpenWithExternalAppDialog) OnConfirm() error {
	return utils.OpenInExternalApp(d.targetFile)
}

func (d *OpenWithExternalAppDialog) LayoutBody(gtx C, th *theme.Theme) D {
	lb := material.Subtitle1(th.Theme, i18n.Translate("Are you sure you want to open file '%s'?", filepath.Base(d.targetFile)))
	lb.Alignment = text.Middle
	return lb.Layout(gtx)
}

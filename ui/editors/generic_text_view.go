package editors

import (
	"errors"
	"path/filepath"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"looz.ws/typstify/editor"
	"looz.ws/typstify/service"

	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/page"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	//"image"
)

var (
	GenericTextEditorViewID = view.NewViewID("GenericTextEditor")
)

type GenericTextEditor struct {
	*view.BaseView
	page.PageStyle
	srv *service.ServiceFacade

	srcEditor   *editor.TextEditor
	currentFile string
	relPath     string
}

func (te *GenericTextEditor) ID() view.ViewID {
	return GenericTextEditorViewID
}

func (te *GenericTextEditor) Title() string {
	if te.currentFile == "" {
		return "Text Editor"
	} else {
		return te.currentFile
	}
}

func (te *GenericTextEditor) OnNavTo(intent view.Intent) error {
	te.BaseView.OnNavTo(intent)
	path, ok := intent.Params["path"].(string)
	if !ok {
		return errors.New("missing parameters")
	}

	rootDir := te.srv.CurrentProjectDir()
	isChildEntry := false
	if rootDir == "" {
		te.relPath = path
	} else {
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			relPath = path
		}
		te.relPath = relPath
		isChildEntry = strings.HasPrefix(path, rootDir)
	}

	te.currentFile = filepath.Base(path)
	srcEditor, err := editor.NewTextEditor(path, false, !isChildEntry, te.srv.Settings().Editor())
	if err != nil {
		return err
	}

	te.srcEditor = srcEditor
	return nil
}

func (te *GenericTextEditor) Actions() []view.ViewAction {
	return []view.ViewAction{}
}

func (te *GenericTextEditor) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	return te.layoutEditor(gtx, th)
}

func (te *GenericTextEditor) layoutEditor(gtx C, th *theme.Theme) D {

	return layout.Inset{
		Left:  unit.Dp(1),
		Right: unit.Dp(0),
		Top:   unit.Dp(1),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return te.layoutFileInfo(gtx, th)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx C) D {
				return te.srcEditor.Layout(gtx, th, te.srv.Settings().Editor())
			}),
		)
	})
}

func (te *GenericTextEditor) layoutFileInfo(gtx C, th *theme.Theme) D {
	return layout.Inset{
		Top:    unit.Dp(1),
		Bottom: unit.Dp(1),
	}.Layout(gtx, func(gtx C) D {
		lb := material.Label(th.Theme, th.TextSize, te.relPath)
		lb.Color = misc.WithAlpha(th.Fg, 0xb6)
		return lb.Layout(gtx)
	})
}

// Implements StatusIndicator to let statusbar render it.
func (te *GenericTextEditor) LayoutStatus(gtx C, th *theme.Theme) D {
	return te.srcEditor.LayoutStatus(gtx, th, te.srv)
}

func (va *GenericTextEditor) OnFinish() {
	va.BaseView.OnFinish()
	// Put your cleanup code here.
	if va.srcEditor != nil {
		va.srcEditor.Close()
	}
}

func NewGenericTextEditor(srv *service.ServiceFacade) view.View {
	return &GenericTextEditor{
		BaseView: &view.BaseView{},
		srv:      srv,
	}
}

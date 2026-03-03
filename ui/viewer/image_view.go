package viewer

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"looz.ws/typstify/i18n"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var ImgViewerViewID = view.NewViewID("ImageViewer")

type ImageViewerView struct {
	*view.BaseView
	currentFile string
	imgViewer   *ImageViewer
}

func (vw *ImageViewerView) ID() view.ViewID {
	return ImgViewerViewID
}

func (vw *ImageViewerView) Title() string {
	if vw.currentFile == "" {
		return "Image Viewer"
	} else {
		return vw.currentFile
	}
}

func (vw *ImageViewerView) OnNavTo(intent view.Intent) error {
	vw.BaseView.OnNavTo(intent)
	path, ok := intent.Params["path"].(string)
	if !ok {
		return errors.New("missing parameters")
	}

	vw.currentFile = filepath.Base(path)
	if vw.imgViewer == nil {
		vw.imgViewer = NewImageViewer()
	}

	vw.imgViewer.Set(path)

	return nil
}

func (vw *ImageViewerView) Layout(gtx C, th *theme.Theme) D {
	return vw.imgViewer.Layout(gtx, th)
}

func (vw *ImageViewerView) LayoutStatus(gtx C, th *theme.Theme) D {
	if vw.imgViewer.src == nil {
		return D{}
	}

	src := vw.imgViewer.src

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
		Spacing:   layout.SpaceBetween,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return material.Label(th.Theme, th.TextSize*0.9, strings.ToUpper(src.Format())).Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		layout.Rigid(func(gtx C) D {
			return layout.E.Layout(gtx, material.Label(th.Theme, th.TextSize*0.9, i18n.Translate("Width: %d, Height: %d", src.Size().X, src.Size().Y)).Layout)
		}),

		layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
		layout.Rigid(func(gtx C) D {
			return material.Label(th.Theme, th.TextSize*0.9, fmt.Sprintf("%.0f%%", 100*vw.imgViewer.ScaleRatio())).Layout(gtx)
		}),
	)
}

func NewImgViewerView() view.View {
	return &ImageViewerView{
		BaseView: &view.BaseView{},
	}
}

package viewer

import (
	goimg "image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/image"
	"github.com/oligo/gioview/theme"
)

const (
	maxScaleRatio = 3
	minScaleRatio = 0.1
	scaleStep     = 0.1
)

type ImageViewer struct {
	src        *image.ImageSource
	scaleRatio float32

	xScrollbar widget.Scrollbar
	yScrollbar widget.Scrollbar
	xScrollOff int
	yScrollOff int
}

func NewImageViewer() *ImageViewer {
	return &ImageViewer{}
}

func (iv *ImageViewer) Set(imgPath string) {
	iv.src = image.ImageFromFile(imgPath)
}

func (iv *ImageViewer) update(gtx C) {
	if iv.scaleRatio <= 0 {
		iv.scaleRatio = 1
	}

	for {
		evt, ok := gtx.Event(pointer.Filter{
			Target:  iv,
			Kinds:   pointer.Scroll,
			ScrollX: pointer.ScrollRange{Min: -1e6, Max: 1e6},
			ScrollY: pointer.ScrollRange{Min: -1e6, Max: 1e6}})
		if !ok {
			break
		}

		e, ok := evt.(pointer.Event)
		if !ok {
			break
		}

		if e.Kind == pointer.Scroll && e.Modifiers.Contain(key.ModShortcut) {
			if e.Scroll.Y > 0 {
				iv.scaleRatio += scaleStep
			} else if e.Scroll.Y < 0 {
				iv.scaleRatio -= scaleStep
			}
		}
	}

	iv.scaleRatio = min(iv.scaleRatio, maxScaleRatio)
	iv.scaleRatio = max(iv.scaleRatio, minScaleRatio)
}

func (iv *ImageViewer) Layout(gtx C, th *theme.Theme) D {
	iv.update(gtx)

	macro := op.Record(gtx.Ops)
	dims := iv.layoutImage(gtx, th)
	callOp := macro.Stop()

	defer clip.Rect(goimg.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, iv)
	callOp.Add(gtx.Ops)

	return dims

}

func (iv *ImageViewer) layoutImage(gtx C, th *theme.Theme) D {
	if iv.src == nil {
		return D{}
	}

	imgSize := iv.fullSize(gtx, iv.src)

	if delta := iv.xScrollbar.ScrollDistance(); delta > 0 {
		iv.xScrollOff += int(float32(imgSize.X) * delta)
	}
	if delta := iv.yScrollbar.ScrollDistance(); delta > 0 {
		iv.yScrollOff += int(float32(imgSize.Y) * delta)
	}

	scrollRatioX, scollRatioY := iv.viewPortRatio(gtx, iv.src)

	dims := layout.Center.Layout(gtx, func(gtx C) D {
		return image.ImageStyle{
			Src:    iv.src,
			Radius: 0,
			Scale:  iv.scaleRatio / gtx.Metric.PxPerDp,
			Fit:    widget.Unscaled,
		}.Layout(gtx)
	})
	//calculate start, end from  me.editor.
	layout.S.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Scrollbar(th.Theme, &iv.xScrollbar).Layout(gtx, layout.Horizontal, scrollRatioX[0], scrollRatioX[1])
	})

	layout.E.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Scrollbar(th.Theme, &iv.yScrollbar).Layout(gtx, layout.Vertical, scollRatioY[0], scollRatioY[1])
	})

	return dims
}

func (iv *ImageViewer) fullSize(gtx C, src *image.ImageSource) goimg.Point {
	return goimg.Point{
		X: int(iv.scaleRatio * float32(src.Size().X) * src.ScaleRatio()),
		Y: int(iv.scaleRatio * float32(src.Size().Y) * src.ScaleRatio()),
	}
}

func (iv *ImageViewer) ScaleRatio() float32 {
	return iv.scaleRatio
}

func (iv *ImageViewer) viewPortRatio(gtx C, src *image.ImageSource) ([]float32, []float32) {
	viewport := gtx.Constraints.Max
	imgSize := iv.fullSize(gtx, src)

	//log.Printf("full size: %v, viewport: %v", imgSize, viewport)

	xViewportRatio := []float32{
		float32(iv.xScrollOff) / float32(imgSize.X),
		float32(iv.xScrollOff+viewport.X) / float32(imgSize.X),
	}
	if xViewportRatio[1] > 1.0 {
		xViewportRatio[1] = 1.0
	}

	yViewportRatio := []float32{
		float32(iv.yScrollOff) / float32(imgSize.Y),
		float32(iv.yScrollOff+viewport.Y) / float32(imgSize.Y),
	}
	if yViewportRatio[1] > 1.0 {
		yViewportRatio[1] = 1.0
	}

	return xViewportRatio, yViewportRatio
}

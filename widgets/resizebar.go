package widgets

import (
	"time"

	"gioui.org/gesture"
	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
)

var (
	elapseDuration = time.Second * 1
)

type ResizeBar struct {
	misc.DividerStyle
	hover     gesture.Hover
	lastHover time.Time
}

func (bar *ResizeBar) update(gtx layout.Context, th *theme.Theme) {
	bar.DividerStyle.Fill = misc.WithAlpha(th.Fg, 0x20)

	if bar.hover.Update(gtx.Source) {
		bar.Thickness = unit.Dp(4)
		bar.lastHover = gtx.Now
		gtx.Execute(op.InvalidateCmd{At: bar.lastHover.Add(elapseDuration)})
	}

	if !bar.lastHover.IsZero() && gtx.Now.Sub(bar.lastHover) > elapseDuration {
		bar.Thickness = unit.Dp(1)
	}
}

func (bar *ResizeBar) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	bar.update(gtx, th)

	if bar.Axis == layout.Horizontal {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
	} else {
		gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
	}

	macro := op.Record(gtx.Ops)
	dims := bar.layout(gtx, th)
	callOp := macro.Stop()

	defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: th.Bg}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bar.hover.Add(gtx.Ops)
	event.Op(gtx.Ops, bar)
	callOp.Add(gtx.Ops)
	return dims
}

func (bar *ResizeBar) layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	// Invisible but interactive area around the divider
	// div := bar.DividerStyle
	// div.Thickness = unit.Dp(6)
	// div.Fill = color.NRGBA{A: 1}
	// dims := div.Layout(gtx, th)

	// draw a visible divider.
	return bar.DividerStyle.Layout(gtx, th)

	// return dims
}

func NewResizeBar(axis layout.Axis) *ResizeBar {
	return &ResizeBar{
		DividerStyle: *misc.Divider(axis, unit.Dp(1)),
	}
}

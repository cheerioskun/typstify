package viewer

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/i18n"
)

type OpKind uint8

const (
	NOOP OpKind = iota
	RefreshKind
	PopupKind
	RestoreKind
)

type OpEvent struct {
	Kind OpKind
}

type previewerOp struct {
	refreshBtn  widget.Clickable
	popupBtn    widget.Clickable
	hovering    bool
	isPopup     bool
	currentPage int
	pageNumber  int
	events      []OpEvent
}

var (
	openInNewIcon, _ = widget.NewIcon(icons.ActionOpenInNew)
	// restoreIcon, _   = widget.NewIcon(icons.ActionRestore)
	refreshIcon, _ = widget.NewIcon(icons.NavigationRefresh)
)

func (bar *previewerOp) update(gtx C) []OpEvent {
	for {
		event, ok := gtx.Event(
			pointer.Filter{Target: bar, Kinds: pointer.Enter | pointer.Leave | pointer.Cancel},
		)
		if !ok {
			break
		}

		switch event := event.(type) {
		case pointer.Event:
			switch event.Kind {
			case pointer.Enter:
				bar.hovering = true
			case pointer.Leave:
				bar.hovering = false
			case pointer.Cancel:
				bar.hovering = false
			}
		}
	}

	bar.events = bar.events[:0]

	if bar.refreshBtn.Clicked(gtx) {
		bar.events = append(bar.events, OpEvent{Kind: RefreshKind})
	}

	if bar.popupBtn.Clicked(gtx) {
		bar.isPopup = true
		bar.events = append(bar.events, OpEvent{Kind: PopupKind})
	}

	return bar.events
}

func (bar *previewerOp) Layout(gtx C, th *theme.Theme) D {
	bar.update(gtx)
	//gtx.Constraints.Max.Y = gtx.Dp(unit.Dp(36))
	gtx.Constraints.Min.Y = 0

	if bar.pageNumber <= 0 {
		return D{}
	}

	macro := op.Record(gtx.Ops)
	dims := layout.Center.Layout(gtx, func(gtx C) D {
		macro2 := op.Record(gtx.Ops)
		dims2 := layout.Inset{
			Top:    unit.Dp(6),
			Bottom: unit.Dp(6),
			Left:   unit.Dp(16),
			Right:  unit.Dp(16),
		}.Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
				Spacing:   layout.SpaceEvenly,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					// refresh
					btn := misc.IconButton(th, refreshIcon, &bar.refreshBtn, "refresh the preview")
					btn.Size = unit.Dp(24)
					btn.Background = color.NRGBA{}
					btn.Color = th.Fg
					if bar.hovering {
						btn.Color = th.ContrastFg
					}
					return btn.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),

				layout.Rigid(func(gtx C) D {
					lb := material.Label(th.Theme, th.TextSize, fmt.Sprintf("%d/%d", bar.currentPage+1, bar.pageNumber))
					if bar.hovering {
						lb.Color = th.ContrastFg
					}
					return lb.Layout(gtx)
				}),

				layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
				layout.Rigid(func(gtx C) D {
					if bar.isPopup {
						// msg = i18n.Translate("restore from the popup window to a embedded previewer.")
						// icon = restoreIcon
						return D{}
					}

					msg := i18n.Translate("popup and display the preview in a dedicated window")
					icon := openInNewIcon
					btn := misc.IconButton(th, icon, &bar.popupBtn, msg)
					btn.Background = color.NRGBA{}
					btn.Color = th.Fg
					if bar.hovering {
						btn.Color = th.ContrastFg
					}

					btn.Size = unit.Dp(24)

					return btn.Layout(gtx)
				}),
			)
		})
		callOp2 := macro2.Stop()
		fill := misc.WithAlpha(th.ContrastBg, th.HoverAlpha)
		if bar.hovering {
			fill = misc.WithAlpha(th.ContrastBg, 0xb6)
		}

		rect := clip.UniformRRect(image.Rectangle{Max: dims2.Size}, gtx.Dp(unit.Dp(8)))
		paint.FillShape(gtx.Ops, fill, rect.Op(gtx.Ops))
		callOp2.Add(gtx.Ops)
		return dims2
	})

	callOp := macro.Stop()

	defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(unit.Dp(8))).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, bar)
	callOp.Add(gtx.Ops)

	return dims
}

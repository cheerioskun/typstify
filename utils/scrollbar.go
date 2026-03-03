package utils

import (
	"image/color"

	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func MakeScrollbar(th *material.Theme, scroll *widget.Scrollbar, color color.NRGBA) material.ScrollbarStyle {
	bar := material.Scrollbar(th, scroll)
	bar.Indicator.Color = color
	bar.Indicator.CornerRadius = unit.Dp(0)
	bar.Indicator.MinorWidth = unit.Dp(10)
	bar.Track.MajorPadding = unit.Dp(0)
	bar.Track.MinorPadding = unit.Dp(1)
	return bar
}

package viewer

import (
	"slices"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
)

type MultiImagesViewer struct {
	// for muliple images.
	listState  widget.List
	srcPaths   []string
	srcMap     map[string]*ImageSource
	changedIdx int
}

func (iv *MultiImagesViewer) Update(images []string) {
	iv.srcPaths = append(iv.srcPaths, images...)
	slices.Sort(iv.srcPaths)
	iv.srcPaths = slices.Compact(iv.srcPaths)

	if iv.srcMap == nil {
		iv.srcMap = make(map[string]*ImageSource)
	}

	// FIXME: how to remove deleted images.
	// unless the issue of https://github.com/typst/typst/issues/4711 is resolved,
	// we don't have any ways to figure out what image should be removed.
	firstChanged := -1
	for idx, p := range iv.srcPaths {
		if old, ok := iv.srcMap[p]; !ok {
			src := ImageFromFile(p)

			iv.srcMap[p] = src
			if firstChanged < 0 {
				firstChanged = idx
			}
		} else {
			// reduce memory allocation:
			if slices.Contains(images, p) {
				// file changed, need to refresh. do not replace the original img, as that causes flicker.
				old.Reresh()

				if firstChanged < 0 {
					firstChanged = idx
				}
			}
		}
	}

	if firstChanged >= 0 {
		iv.changedIdx = firstChanged
	}
}

func (iv *MultiImagesViewer) Page() (int, int) {
	return len(iv.srcPaths), iv.listState.Position.First
}

func (iv *MultiImagesViewer) Clear() {
	iv.srcPaths = iv.srcPaths[:0]
	iv.changedIdx = 0
	clear(iv.srcMap)
}

func (iv *MultiImagesViewer) Layout(gtx C, th *theme.Theme) D {
	iv.listState.Axis = layout.Vertical

	if len(iv.srcPaths) <= 0 {
		return D{}
	}

	if iv.changedIdx >= 0 {
		if iv.changedIdx == len(iv.srcPaths)-1 {
			iv.listState.ScrollToEnd = true
			iv.listState.Position.BeforeEnd = false
			defer func() {
				iv.listState.ScrollToEnd = false
				//iv.listState.Position.BeforeEnd = false
			}()
		} else {
			iv.listState.Position.First = iv.changedIdx
			iv.listState.Position.Offset = 0
		}
		iv.changedIdx = -1
	}

	return material.List(th.Theme, &iv.listState).Layout(gtx, len(iv.srcPaths), func(gtx C, index int) D {
		inset := layout.Inset{
			Left:   unit.Dp(4),
			Right:  unit.Dp(4),
			Bottom: unit.Dp(8),
		}

		if index == 0 {
			inset.Top = 20
		}
		if index == len(iv.srcPaths)-1 {
			inset.Bottom = 29
		}

		return layout.Center.Layout(gtx, func(gtx C) D {
			return inset.Layout(gtx, func(gtx C) D {
				src := iv.srcMap[iv.srcPaths[index]]
				if int(gtx.Metric.PxPerDp) == 1 {
					// low pixel density needs higher scale quality.
					src.ScaleQuality = High
				} else {
					src.ScaleQuality = Medium
				}

				return ImageStyle{
					Src:      src,
					Radius:   0,
					Scale:    1 / gtx.Metric.PxPerDp,
					Fit:      widget.Unscaled, // Do not scale in gio Image, as scaling of Gio Image causes blurry output.
					Position: layout.Center,
				}.Layout(gtx)

			})
		})
	})
}

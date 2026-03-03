package editor

import (
	"image"
	"image/color"
	"maps"
	"math"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/oligo/gioview/theme"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/gutter/providers"
)

type MarkerKind int

const (
	MarkerKindDiagnostic MarkerKind = iota // LSP diagnostics
	MarkerKindDiff                         // Git diff changes
)

var (
	rulerMarkerWidth = unit.Dp(10)
)

// OverviewRuler is a vertical bar that shows global position in a document
// using colored boxes. It is located immediately under the main vertical
// scroll bar.
type OverviewRuler struct {
	// total lines rendered of the document.
	lineCount       int
	lineMarkers     map[MarkerKind][]lineMarker
	diagnosticColor gvcolor.Color
	diffColor       gvcolor.Color

	// Width configuration
	markerWidth unit.Dp // total width per marker (default: 10dp)
}

type lineMarker struct {
	line int // zero based line index
}

func (r *OverviewRuler) SetLines(lines int) {
	r.lineCount = lines
}

// Clear markers by category
func (r *OverviewRuler) clearDiagnosticMarkers() {
	if r.lineMarkers == nil {
		r.lineMarkers = map[MarkerKind][]lineMarker{}
	}
	r.lineMarkers[MarkerKindDiagnostic] = r.lineMarkers[MarkerKindDiagnostic][:0]
}

func (r *OverviewRuler) clearDiffMarkers() {
	if r.lineMarkers == nil {
		r.lineMarkers = map[MarkerKind][]lineMarker{}
	}
	r.lineMarkers[MarkerKindDiff] = r.lineMarkers[MarkerKindDiff][:0]
}

// Add markers
func (r *OverviewRuler) UpdateDiagnosticMarkers(lines ...int) {
	r.clearDiagnosticMarkers()
	for _, line := range lines {
		r.lineMarkers[MarkerKindDiagnostic] = append(r.lineMarkers[MarkerKindDiagnostic], lineMarker{line: line})
	}
}

func (r *OverviewRuler) UpdateDiffMarkers(hunks []*providers.DiffHunk) {
	r.clearDiffMarkers()

	for _, hunk := range hunks {
		if hunk.StartLine > hunk.EndLine {
			continue // invalid hunk
		}
		for line := hunk.StartLine; line <= hunk.EndLine; line++ {
			r.lineMarkers[MarkerKindDiff] = append(r.lineMarkers[MarkerKindDiff], lineMarker{line: line})
		}
	}
}

// Color configuration
func (r *OverviewRuler) SetDiagnosticColor(color gvcolor.Color) {
	r.diagnosticColor = color
}

func (r *OverviewRuler) SetDiffColor(color gvcolor.Color) {
	r.diffColor = color
}

func (r *OverviewRuler) UseDefaultColors() {
	r.diagnosticColor, _ = gvcolor.Hex2Color("#e74c3c")
	r.diffColor, _ = gvcolor.Hex2Color("#007AFF99")
	// Default dimensions
	if r.markerWidth == 0 {
		r.markerWidth = rulerMarkerWidth
	}
}

func (r *OverviewRuler) colorForCategory(cat MarkerKind) gvcolor.Color {
	switch cat {
	case MarkerKindDiagnostic:
		return r.diagnosticColor
	case MarkerKindDiff:
		return r.diffColor
	default:
		return gvcolor.MakeColor(color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}) // gray fallback
	}
}

// allCategories returns all categories in fixed drawing order
func allCategories() []MarkerKind {
	return []MarkerKind{MarkerKindDiff, MarkerKindDiagnostic}
}

func (r *OverviewRuler) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	if len(r.lineMarkers) == 0 {
		return layout.Dimensions{}
	}

	macro := op.Record(gtx.Ops)
	r.layoutMarker(gtx)
	callOp := macro.Stop()

	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	callOp.Add(gtx.Ops)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (r *OverviewRuler) layoutMarker(gtx layout.Context) layout.Dimensions {
	if r.lineCount == 0 {
		// No lines, nothing to draw
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
	maxY := gtx.Constraints.Max.Y
	markerHeight := (float64(maxY) / float64(r.lineCount))

	// Use default dimensions if not set
	if r.markerWidth == 0 {
		r.markerWidth = rulerMarkerWidth
	}

	// Calculate segment position
	numSegments := len(allCategories())
	segmentWidth := gtx.Dp(r.markerWidth) / numSegments
	if segmentWidth < 1 {
		segmentWidth = 1
	}

	// Draw consecutive diff-only lines as continuous bars
	r.layoutDiffMarker(gtx, markerHeight, segmentWidth, maxY)

	// Draw by kind in fixed order for other markers.
	r.layoutOtherMarkers(gtx, segmentWidth, markerHeight)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

func (r *OverviewRuler) layoutOtherMarkers(gtx layout.Context, segmentWidth int, markerHeight float64) {
	otherMarkers := maps.Clone(r.lineMarkers)
	delete(otherMarkers, MarkerKindDiff)

	for kindIdx, kind := range allCategories() {
		if kind == MarkerKindDiff {
			continue
		}

		color := r.colorForCategory(kind)

		lines := otherMarkers[kind]
		// Draw all lines of this kind
		for _, line := range lines {
			offsetX := segmentWidth * kindIdx

			rectHeight := min(int(math.Round(markerHeight)), gtx.Dp(unit.Dp(3)))
			if rectHeight < 1 {
				rectHeight = 1
			}

			offsetTrans := op.Offset(image.Point{
				X: offsetX,
				Y: int(math.Round(markerHeight * float64(line.line))),
			}).Push(gtx.Ops)
			markerArea := clip.Rect(image.Rectangle{
				Max: image.Point{
					X: segmentWidth,
					Y: rectHeight,
				},
			}).Push(gtx.Ops)

			paint.ColorOp{Color: color.NRGBA()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			markerArea.Pop()
			offsetTrans.Pop()
		}
	}
}

func (r *OverviewRuler) layoutDiffMarker(gtx layout.Context, markerHeight float64, segmentWidth int, maxY int) {
	diffLines := r.lineMarkers[MarkerKindDiff]
	if len(diffLines) <= 0 {
		return
	}

	start := diffLines[0].line
	end := start
	offsetX := 0

	for _, marker := range diffLines[1:] {
		if marker.line == end+1 {
			end = marker.line
		} else {
			// Draw bar from start to end inclusive
			startY := int(math.Round(markerHeight * float64(start)))
			endY := int(math.Round(markerHeight * float64(end+1))) // +1 because end is inclusive
			// Clamp to available height
			if endY > maxY {
				endY = maxY
			}
			if startY > maxY {
				startY = maxY
			}
			barHeight := endY - startY
			if barHeight < 1 {
				barHeight = 1
			}

			color := r.diffColor
			offsetTrans := op.Offset(image.Point{
				X: offsetX,
				Y: startY,
			}).Push(gtx.Ops)
			barArea := clip.Rect(image.Rectangle{
				Max: image.Point{
					X: segmentWidth,
					Y: barHeight,
				},
			}).Push(gtx.Ops)
			paint.ColorOp{Color: color.NRGBA()}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			barArea.Pop()
			offsetTrans.Pop()

			start = marker.line
			end = start
		}
	}

	// Draw the last bar
	startY := int(math.Round(markerHeight * float64(start)))
	endY := int(math.Round(markerHeight * float64(end+1)))
	if endY > maxY {
		endY = maxY
	}
	if startY > maxY {
		startY = maxY
	}
	if startY == endY {
		startY = endY - int(math.Round(markerHeight))
	}

	barHeight := endY - startY
	if barHeight < 1 {
		barHeight = 1
	}

	color := r.diffColor
	offsetTrans := op.Offset(image.Point{
		X: offsetX,
		Y: startY,
	}).Push(gtx.Ops)
	barArea := clip.Rect(image.Rectangle{
		Max: image.Point{
			X: segmentWidth,
			Y: barHeight,
		},
	}).Push(gtx.Ops)
	paint.ColorOp{Color: color.NRGBA()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	barArea.Pop()
	offsetTrans.Pop()
}

package console

import (
	"io"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/syntax"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/utils"
	"looz.ws/typstify/widgets/icons"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var (
	closeIcon = icons.NewSvgIcon(icons.X)
	clearIcon = icons.NewSvgIcon(icons.BrushCleaning)
)

var _ io.Writer = (*ConsoleState)(nil)

type ConsoleState struct {
	lines       []string
	partialLine string
	state       *gvcode.Editor
	colorScheme syntax.ColorScheme
	yScroll     widget.Scrollbar
	maxLines    int
	err         error
	textUpated  atomic.Bool
	mu          sync.Mutex

	ShowConsole bool
}

// Create a console.
func NewConsoleState(maxLines int) *ConsoleState {
	state := &gvcode.Editor{}
	state.WithOptions(
		gvcode.WithLineHeight(0, 1.5),
		gvcode.ReadOnlyMode(true),
		gvcode.WrapLine(true),
	)
	c := &ConsoleState{
		maxLines: maxLines,
		state:    state,
	}

	return c
}

func (c *ConsoleState) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	msg := Strip(string(data))
	c.appendText(msg)
	c.textUpated.Store(true)

	return len(data), nil
}

func (c *ConsoleState) HasMore() bool {
	return c.textUpated.Load()
}

func (c *ConsoleState) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = c.lines[:0]
	c.partialLine = ""
	c.state.SetText("")
}

func (c *ConsoleState) readBuffered() {
	c.mu.Lock()
	text := c.visibleText()
	c.state.SetText(text)
	textLen := c.state.Len()
	c.state.SetCaret(textLen, textLen)
	c.mu.Unlock()
}

func (c *ConsoleState) Layout(gtx C, th *theme.Theme) D {
	c.update(gtx, th)

	if c.err != nil {
		lb := material.Label(th.Theme, th.TextSize*0.9, i18n.Translate("Console has error:", c.err.Error()))
		lb.Font.Typeface = th.Face
		lb.Font.Weight = font.SemiBold

		return lb.Layout(gtx)
	}

	if c.state.Len() <= 0 {
		lb := material.Label(th.Theme, th.TextSize*0.9, i18n.Translate("No messages."))
		lb.Font.Typeface = th.Face

		return lb.Layout(gtx)
	}

	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Flexed(1.0, func(gtx layout.Context) layout.Dimensions {
			return c.state.Layout(gtx, th.Shaper)
		}),

		layout.Rigid(func(gtx C) D {
			_, _, minY, maxY := c.state.ScrollRatio()
			scrollIndicatorColor := misc.WithAlpha(th.Fg, 0x30)

			bar := utils.MakeScrollbar(th.Theme, &c.yScroll, scrollIndicatorColor)
			return bar.Layout(gtx, layout.Vertical, minY, maxY)
		}),
	)

}

func (c *ConsoleState) update(gtx C, th *theme.Theme) {
	if c.colorScheme.Foreground.NRGBA() != th.Fg || c.colorScheme.Background.NRGBA() != th.Bg {
		c.colorScheme.Foreground = color.MakeColor(th.Fg)
		c.colorScheme.Background = color.MakeColor(th.Bg) // overwrite with global palette color.
		c.colorScheme.SelectColor = color.MakeColor(th.ContrastBg).MulAlpha(0x60)
		c.colorScheme.LineColor = color.MakeColor(th.ContrastBg).MulAlpha(0x30)
		c.colorScheme.LineNumberColor = c.colorScheme.Foreground.MulAlpha(0xb6)
		c.state.WithOptions(gvcode.WithColorScheme(c.colorScheme))
	}

	c.state.WithOptions(
		gvcode.WithTextSize(th.TextSize),
		gvcode.WithFont(font.Font{Typeface: th.Face, Weight: font.Medium}),
		gvcode.WithDefaultGutters(),
		gvcode.WithGutterGap(unit.Dp(8)),
	)

	if c.textUpated.CompareAndSwap(true, false) {
		c.readBuffered()
		c.truncate()
	}

	yScrollDist := c.yScroll.ScrollDistance()
	if yScrollDist != 0.0 {
		c.state.Scroll(gtx, 0, yScrollDist)
	}

}

// truncate keeps only the newest maxLines of completed/visible console output.
// It runs after new text is appended, so older lines are dropped as soon as the
// buffered line store grows past the configured cap.
func (c *ConsoleState) truncate() {
	if c.maxLines <= 0 {
		c.lines = c.lines[:0]
		c.partialLine = ""
		return
	}

	lineCount := len(c.lines)
	if c.partialLine != "" {
		lineCount++
	}
	if lineCount <= c.maxLines {
		return
	}

	overflow := lineCount - c.maxLines
	if overflow >= len(c.lines) {
		c.lines = c.lines[:0]
		return
	}

	c.lines = append(c.lines[:0], c.lines[overflow:]...)
}

func (c *ConsoleState) appendText(msg string) {
	if msg == "" {
		return
	}

	combined := c.partialLine + msg
	c.partialLine = ""

	parts := strings.SplitAfter(combined, "\n")
	if !strings.HasSuffix(combined, "\n") {
		c.partialLine = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}

	c.lines = append(c.lines, parts...)
	c.truncate()
}

func (c *ConsoleState) visibleText() string {
	var b strings.Builder
	total := 0
	for _, line := range c.lines {
		total += len(line)
	}
	total += len(c.partialLine)
	b.Grow(total)

	for _, line := range c.lines {
		b.WriteString(line)
	}
	b.WriteString(c.partialLine)

	return b.String()
}

// ansi matches terminal escape/control sequences such as colors, cursor moves,
// and other CSI/OSC-style commands so GUI console output stays plain text.
const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

// Strip removes ANSI terminal escape sequences from streamed console output.
// The pattern is not empty: it starts with ESC/C1 control bytes and then matches
// the parameter bytes and final opcode used by ANSI control sequences.
func Strip(str string) string {
	return re.ReplaceAllString(str, "")
}

type Console struct {
	state           *ConsoleState
	closeConsoleBtn widget.Clickable
	clearConsoleBtn widget.Clickable
}

func NewConsolePanel(cs *ConsoleState) *Console {
	c := &Console{
		state: cs,
	}

	return c
}

func (c *Console) Layout(gtx C, th *theme.Theme) D {
	if c.closeConsoleBtn.Clicked(gtx) {
		c.state.ShowConsole = false
	}
	if c.clearConsoleBtn.Clicked(gtx) {
		c.state.Clear()
	}

	if !c.state.ShowConsole {
		return D{}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return c.layoutBorder(gtx, th)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{
				Top:   unit.Dp(4),
				Left:  unit.Dp(24),
				Right: unit.Dp(0),
			}.Layout(gtx, func(gtx C) D {
				return c.state.Layout(gtx, th)
			})
		}),
	)
}

func (c *Console) layoutBorder(gtx C, th *theme.Theme) D {
	macro := op.Record(gtx.Ops)
	dims := layout.Inset{
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
		Left:   unit.Dp(12),
		Right:  unit.Dp(12),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
			Spacing:   layout.SpaceBetween,
		}.Layout(gtx,
			layout.Flexed(1, func(gtx C) D {
				cap := material.Caption(th.Theme, "Output")
				cap.Color = th.Fg
				return cap.Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				return material.Clickable(gtx, &c.clearConsoleBtn, func(gtx C) D {
					return clearIcon.Layout(gtx, th.Fg, th.TextSize)
				})
			}),
			layout.Rigid(layout.Spacer{Width: 4}.Layout),
			layout.Rigid(func(gtx C) D {
				return material.Clickable(gtx, &c.closeConsoleBtn, func(gtx C) D {
					return closeIcon.Layout(gtx, th.Fg, th.TextSize)
				})
			}),
		)
	})
	callOp := macro.Stop()

	defer clip.Rect{Max: dims.Size}.Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, misc.WithAlpha(th.Bg2, 0xb6))
	callOp.Add(gtx.Ops)

	return dims
}

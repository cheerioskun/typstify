package statusbar

import (
	"image"
	"log"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/ui/console"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

var defaultIdleDuration = time.Second * 5
var (
	notifyIcon, _   = widget.NewIcon(icons.ActionInfoOutline)
	warningIcon, _  = widget.NewIcon(icons.AlertErrorOutline)
	warningIcon2, _ = widget.NewIcon(icons.AlertError)
)

// Views can implement StatusLine interface to let StatusBar render their
// status indicator.
type StatusIndicator interface {
	LayoutStatus(gtx C, th *theme.Theme) D
}

type NotificationBar struct {
	message string
	// reset notification after the specified duration.
	idleDuration time.Duration
	// last notification update time
	lastUpdateTime time.Time
}

type Notification struct {
	Content  string
	Duration time.Duration
}

type StatusBar struct {
	vm             view.ViewManager
	notification   *NotificationBar
	consoleState   *console.ConsoleState
	showConsoleBtn widget.Clickable
}

func (n *NotificationBar) Layout(gtx C, th *theme.Theme) D {
	if n.message == "" {
		return D{}
	}

	// If idleDuration has zero value, the message will not expire.
	if n.lastUpdateTime.IsZero() {
		n.lastUpdateTime = gtx.Now
		if n.idleDuration > 0 {
			gtx.Execute(op.InvalidateCmd{At: n.lastUpdateTime.Add(n.idleDuration)})
		}

	} else if n.idleDuration > 0 && gtx.Now.Sub(n.lastUpdateTime) > n.idleDuration {
		defer func() {
			n.lastUpdateTime = gtx.Now
			n.message = ""
		}()
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return misc.Icon{Icon: notifyIcon, Size: unit.Dp(16)}.Layout(gtx, th)
		}),

		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx C) D {
			return material.Label(th.Theme, th.TextSize*0.9, n.message).Layout(gtx)
		}),
	)
}

func (s *StatusBar) Update(gtx C) bool {
	if s.showConsoleBtn.Clicked(gtx) {
		return true
	}

	return false
}

func (s *StatusBar) Layout(gtx C, th *theme.Theme) D {
	s.Update(gtx)

	if s.notification.message == "" && s.vm.CurrentView() == nil {
		return D{}
	}

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
			Spacing:   layout.SpaceStart,
		}.Layout(gtx,
			layout.Flexed(1, func(gtx C) D {

				return s.notification.Layout(gtx, th)
			}),

			layout.Rigid(func(gtx C) D {
				vw := s.vm.CurrentView()
				if vw == nil {
					return D{}
				}

				status, ok := vw.(StatusIndicator)
				if !ok {
					return D{}
				}

				return status.LayoutStatus(gtx, th)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx C) D {
				return material.Clickable(gtx, &s.showConsoleBtn, func(gtx C) D {
					icon := misc.Icon{Icon: warningIcon, Color: th.Fg, Size: unit.Dp(th.TextSize * 1.2)}
					if s.consoleState.HasMore() {
						icon.Icon = warningIcon2
						// icon.Color = misc.WithAlpha(icon.Color, 0xb6)
					}
					return icon.Layout(gtx, th)
				})
			}),
		)
	})

	statusOps := macro.Stop()

	rect := clip.Rect(image.Rectangle{Max: dims.Size})
	paint.FillShape(gtx.Ops, misc.WithAlpha(th.Bg2, 250), rect.Op())
	statusOps.Add(gtx.Ops)

	return dims
}

func NewStatusBar(srv *service.ServiceFacade, vm view.ViewManager) *StatusBar {
	sb := &StatusBar{
		vm:           vm,
		notification: &NotificationBar{},
		consoleState: srv.Console(),
	}
	eventbus := srv.EventBus()
	eventbus.Subscribe(sb, "statusbar.event", `statusbar\.*`, func(topic string, data interface{}) {
		switch topic {
		case bus.TopicStatusbarNotifyEvent:
			msg := data.(Notification)
			sb.notification.message = msg.Content
			sb.notification.idleDuration = msg.Duration
			if sb.notification.idleDuration <= 0 {
				sb.notification.idleDuration = defaultIdleDuration
			}
			vm.Invalidate()
			log.Println("notification: ", msg)
		}
	})

	return sb
}

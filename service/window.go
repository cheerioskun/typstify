package service

import (
	"context"
	"log"
	"sync"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/oligo/gioview/theme"
	"looz.ws/typstify/fonts"
	"looz.ws/typstify/service/settings"
	"looz.ws/typstify/ui/palette"
)

// Application keeps track of all the windows and global state.
type WindowService struct {
	settings *settings.Settings
	// Context is used to broadcast application shutdown.
	Context context.Context
	// Shutdown shuts down all windows.
	Shutdown func()
	// active keeps track the open windows, such that application
	// can shut down, when all of them are closed.
	active sync.WaitGroup
}

func NewWindowService(ctx context.Context, settings *settings.Settings) *WindowService {
	ctx, cancel := context.WithCancel(ctx)
	return &WindowService{
		Context:  ctx,
		Shutdown: cancel,
		settings: settings,
	}
}

// Wait waits for all windows to close.
func (w *WindowService) Wait() {
	w.active.Wait()
}

// NewWindow creates a new tracked window.
func (w *WindowService) NewWindow(ctx context.Context, title string, view WindowView, opts ...app.Option) {
	opts = append(opts, app.Title(title))
	w.active.Add(1)
	go func() {
		defer w.active.Done()

		w := &Window{
			Service: w,
			Window:  new(app.Window),
		}
		w.Window.Option(opts...)
		view.Run(ctx, w)
	}()
}

func (w *WindowService) LoadTheme() *theme.Theme {
	th := theme.NewTheme("", fonts.Embedded, false)

	themeName := w.settings.General().Theme
	if themeName == "" {
		themeName = "Default Light"
	}

	cfg, err := palette.ThemeConfig(themeName)
	if err != nil {
		log.Println("Theme query failed: ", err)
		return th
	}

	th.TextSize = unit.Sp(w.settings.General().TextSize)
	th = th.WithPalette(cfg.Palette)
	return th
}

// Window holds window state.
type Window struct {
	Service *WindowService
	*app.Window
}

type WindowView interface {
	// Run handles the window event loop.
	Run(ctx context.Context, w *Window) error
}

// WidgetView allows to use gioview Widget as a view.
type WidgetView func(gtx layout.Context, th *theme.Theme) layout.Dimensions

// Run displays the widget with default handling.
func (view WidgetView) Run(ctx context.Context, w *Window) error {
	var ops op.Ops
	th := w.Service.LoadTheme()

	go func() {
		select {
		case <-w.Service.Context.Done():
			w.Perform(system.ActionClose)
			log.Println("window is closed")
		case <-ctx.Done():
			w.Perform(system.ActionClose)
			log.Println("window is closed")
		}
	}()

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			view(gtx, th)
			e.Frame(gtx.Ops)
		}
	}
}

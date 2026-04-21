package preview

import (
	"context"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/unit"
	"github.com/oligo/gioview/theme"

	"looz.ws/typstify/lsp"
	"looz.ws/typstify/service"
)

type Previewer struct {
	srv            *service.ServiceFacade
	err            error
	webview        *WebView
	previewMode    lsp.PreviewMode
	destroyPending bool // true when webview should be destroyed on next layout
}

func NewPreviewer(srv *service.ServiceFacade) *Previewer {
	return &Previewer{
		srv:         srv,
		webview:     NewWebView(),
		previewMode: lsp.PreviewMode(srv.Workspace().LoadWorkspaceSettings().PreviewMode),
	}
}

func (p *Previewer) Navigate(url string) {
	p.webview.Navigate(url)
}

func (p *Previewer) Mode() lsp.PreviewMode {
	return p.previewMode
}

func (p *Previewer) ToggleMode() {
	p.previewMode = lsp.PreviewMode(p.srv.Workspace().LoadWorkspaceSettings().PreviewMode)

	switch p.previewMode {
	case lsp.DocumentPreviewMode:
		p.previewMode = lsp.SlidePreviewMode
	case lsp.SlidePreviewMode:
		p.previewMode = lsp.DocumentPreviewMode
	default:
		p.previewMode = lsp.DocumentPreviewMode
	}

	p.srv.Workspace().SetPreviewMode(string(p.previewMode))
	p.srv.RestartPreview(context.Background(), func() {
		serverAddr := p.srv.PreviewService().Address()
		if serverAddr != "" {
			p.Navigate(serverAddr)
		}
	})

}

func (p *Previewer) Restart() {
	p.srv.RestartPreview(context.Background(), func() {
		serverAddr := p.srv.PreviewService().Address()

		if serverAddr != "" {
			p.Navigate(serverAddr)
		}
	})
}

func (p *Previewer) Layout(gtx layout.Context, th *theme.Theme) layout.Dimensions {
	// Handle pending destroy request
	if p.destroyPending {
		p.webview.Destroy(gtx)
		p.webview = nil
		p.destroyPending = false
		return layout.Dimensions{}
	}

	// If webview was destroyed and not recreated, return empty dimensions
	if p.webview == nil {
		return layout.Dimensions{}
	}

	// Left inset so the native webview doesn't cover the resize drag handle.
	return layout.Inset{Left: unit.Dp(6)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		absOffset := f32.Point{
			X: float32(p.srv.WindowContentWidth - gtx.Constraints.Max.X),
			Y: float32(p.srv.ViewAreaTopOffset),
		}
		return p.webview.Layout(gtx, th, absOffset)
	})
}

// Destroy permanently destroys the native webview.
// Called when switching to OpenInBrowser mode to release resources.
func (p *Previewer) Destroy() {
	// We need a valid context to execute the destroy command.
	// Since this is called outside of layout, we defer it to the next layout.
	// The webview will be destroyed on the next frame.
	// Actually, we need to handle this differently - the Destroy needs to be
	// called during a frame. We'll use a flag that gets processed in Layout.
	p.destroyPending = true
}

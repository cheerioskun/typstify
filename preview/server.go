package preview

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	webview "github.com/webview/webview_go"
	"golang.org/x/exp/jsonrpc2"
	"looz.ws/typstify/i18n"
)

const (
	previewRPCServer       = "127.0.0.1:15422"
	rpcMethodClose         = "preview/close"
	rpcMethodNew           = "preview/new"
	rpcMethodNotifyClosing = "preview/notifyClosing"
)

var (
	// RequestCancelledError should be used when a request is cancelled early.
	RequestCancelledError = jsonrpc2.NewError(-32800, "JSON RPC cancelled")
)

type PreviewServer struct {
	*dispatcher
	running      atomic.Bool
	serverCancel context.CancelFunc
	mu           sync.Mutex
	lastWebview  webview.WebView
	wvClosed     bool
	lastFile     string

	connMu sync.Mutex
	// Assume there is always one connection be make.
	clientConns []*jsonrpc2.Connection
}

func NewPreviewServer() *PreviewServer {
	return &PreviewServer{
		dispatcher: newDispathcer(),
	}
}

// Run start the previewer rpc server. Should only be called when spawning the previewer process.
func (p *PreviewServer) startServer() {
	if !p.running.CompareAndSwap(false, true) {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.serverCancel = cancel
	defer cancel()
	listener, err := jsonrpc2.NetListener(ctx, "tcp", previewRPCServer, jsonrpc2.NetListenOptions{})
	if err != nil {
		log.Println("Creating net listener failed, preview server quited!", err)
		os.Exit(1)
	}

	server, err := jsonrpc2.Serve(ctx, listener, p)
	server.Wait()
	log.Println("Preview server quited")
}

func (p *PreviewServer) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	p.connMu.Lock()
	defer p.connMu.Unlock()
	p.clientConns = append(p.clientConns, conn)
	return jsonrpc2.ConnectionOptions{
		Handler: jsonrpc2.HandlerFunc(p.rpcHandle),
	}, nil
}

// should be called after detecting Update returned OpKind.
func (p *PreviewServer) showup(req PreviewReq) {
	wv := webview.New(false)
	p.lastWebview = wv
	p.wvClosed = false

	wv.SetTitle(i18n.Translate("%s preview", filepath.Base(req.TargetFile)))
	wv.SetSize(640, 480, webview.HintNone)
	wv.Navigate(req.Server)
	defer func() {
		wv.Destroy()
		p.notifyClosingPreview()
		p.wvClosed = true
		p.lastFile = ""
		p.lastWebview = nil
		slog.Info("webview destroyed!!!")
	}()

	wv.Run()
}

// Run must be executed in the main thread.
func (p *PreviewServer) Run() {
	defer func() {
		if x := recover(); x != nil {
			log.Panicln("preview server panic", x)
		}
	}()

	go func() {
		p.startServer()
	}()

	// start dispatcher loop.
	p.StartLoop()
}

func (p *PreviewServer) Close() {
	if p.running.CompareAndSwap(true, false) {
		p.serverCancel()
		if p.lastWebview != nil && !p.wvClosed {
			p.lastWebview.Terminate()
		}
	}
}

func (p *PreviewServer) closePreview(targetFile string) {
	if p.lastWebview != nil && !p.wvClosed && targetFile == p.lastFile {
		p.lastWebview.Terminate()
	}
}

func (p *PreviewServer) startWebview(req PreviewReq) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.lastWebview == nil {
		p.lastFile = req.TargetFile
		p.Add(func() {
			p.showup(req)
		})
		return
	}

	if p.lastFile == req.TargetFile {
		return
	}

	// new preview requested.
	// close other webviews to make room for the requested one.
	p.notifyClosingPreview()

	p.lastFile = req.TargetFile
	p.lastWebview.Dispatch(func() {
		p.lastWebview.SetTitle(i18n.Translate("%s preview", filepath.Base(req.TargetFile)))
		p.lastWebview.Navigate(req.Server)
	})

}

func (p *PreviewServer) notifyClosingPreview() {
	p.connMu.Lock()
	defer p.connMu.Unlock()
	params := WebviewClosedNotifyReq{
		TargetFile: p.lastFile,
	}

	i := 0
	for _, conn := range p.clientConns {
		err := conn.Notify(context.Background(), rpcMethodNotifyClosing, &params)
		if err == nil || !errors.Is(err, nil) {
			p.clientConns[i] = conn
			i++
		}

		if err != nil {
			log.Println("notify webview closing failed", err)
		}
	}

	// nil the left over conn to prevent memory leaks.
	for j := i; j < len(p.clientConns); j++ {
		p.clientConns[j] = nil
	}

	p.clientConns = p.clientConns[:i]
}

// If the request is a call, it must return a value or an error for the reply
func (p *PreviewServer) rpcHandle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	if ctx.Err() != nil {
		return nil, RequestCancelledError
	}

	defer func() {
		if x := recover(); x != nil {
			log.Printf("panic in request, method: %s", req.Method)
			panic(x)
		}
	}()

	switch req.Method {
	case rpcMethodClose:
		var closeReq PreviewCloseReq
		err := UnmarshalJSON(req.Params, &closeReq)
		if err != nil {
			return nil, err
		}
		p.closePreview(closeReq.TargetFile)
		return "ok", nil
	case rpcMethodNew:
		var previewReq PreviewReq
		err := UnmarshalJSON(req.Params, &previewReq)
		if err != nil {
			return nil, err
		}
		p.startWebview(previewReq)
		return "ok", nil
	}

	return nil, jsonrpc2.ErrMethodNotFound

}

type PreviewReq struct {
	Server     string
	TargetFile string
}

type PreviewCloseReq struct {
	TargetFile string
}

type WebviewClosedNotifyReq struct {
	TargetFile string
}

// UnmarshalJSON unmarshals msg into the variable pointed to by
// params. In JSONRPC, optional messages may be
// "null", in which case it is a no-op.
func UnmarshalJSON(msg json.RawMessage, v any) error {
	if len(msg) == 0 || bytes.Equal(msg, []byte("null")) {
		return nil
	}
	return json.Unmarshal(msg, v)
}

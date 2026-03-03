package preview

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/oligo/gvcode"
	"golang.org/x/exp/jsonrpc2"

	"looz.ws/typstify/lsp"
)

type PreviewOptions struct {
	PreviewMode      string
	ProjectRoot      string
	FontPath         string
	PackagePath      string
	PackageCachePath string
	InvertColor      string
	PartialRender    bool
	OpenInBrowser    bool
}

// Previewer is the previewer client.
type Previewer struct {
	client       *lsp.Client
	conn         *jsonrpc2.Connection
	targetFile   string
	lastTask     string
	isPreviewing bool
}

func NewPreviewer(client *lsp.Client) (*Previewer, error) {
	p := &Previewer{client: client}

	dialer := jsonrpc2.NetDialer("tcp", previewRPCServer, net.Dialer{
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	conn, err := jsonrpc2.Dial(ctx, dialer, jsonrpc2.ConnectionOptions{
		Handler: clientHandler(p),
	})
	if err != nil {
		return nil, err
	}

	p.conn = conn
	return p, nil
}

// func (p *Previewer) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
// 	return jsonrpc2.ConnectionOptions{
// 		Handler: jsonrpc2.HandlerFunc(func(ctx context.Context, req *jsonrpc2.Request) (interface{}, error) {
// 			return p.clientRPCHandle(ctx, req)
// 		}),
// 	}, nil
// }

func (p *Previewer) New(ctx context.Context, targetFile string, opts PreviewOptions) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	if p.isPreviewing {
		// kill the last preview first here as we can not get callback when browser tab is closing.
		err := p.killLspPreview(ctx)
		if err != nil {
			return err
		}
	}

	previewServerPort, err := p.requestLspPreview(ctx, targetFile, opts)
	if err != nil {
		return err
	}

	p.targetFile = targetFile

	if opts.OpenInBrowser {
		return nil
	}

	params := PreviewReq{
		Server:     fmt.Sprintf("http://127.0.0.1:%d", previewServerPort),
		TargetFile: targetFile,
	}

	//log.Println("requesting preview with params: ", params)
	var result interface{}
	err = p.conn.Call(ctx, rpcMethodNew, &params).Await(ctx, &result)
	if err != nil {
		p.killLspPreview(ctx)
		return err
	}

	p.isPreviewing = true
	return nil
}

func (p *Previewer) Close(ctx context.Context, targetFile string) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := p.killLspPreview(ctx)
	if err != nil {
		return err
	}

	p.isPreviewing = false

	params := PreviewCloseReq{
		TargetFile: targetFile,
	}
	var result interface{}
	return p.conn.Call(ctx, rpcMethodClose, &params).Await(ctx, &result)

}

func (p *Previewer) Destroy(ctx context.Context) {
	p.Close(ctx, p.targetFile)
	p.conn.Close()
}

func (p *Previewer) requestLspPreview(ctx context.Context, targetFile string, opts PreviewOptions) (int, error) {
	taskID := rand.Text()[:8]
	p.lastTask = taskID
	args := []any{
		"--task-id", taskID,
		"--data-plane-host", "127.0.0.1:0",
		"--preview-mode", opts.PreviewMode,
		"--root", opts.ProjectRoot,
		"--invert-colors", opts.InvertColor,
	}

	// if opts.PackagePath != "" {
	// 	args = append(args, "--package-path", opts.PackagePath)
	// }
	// if opts.PackageCachePath != "" {
	// 	args = append(args, "--package-cache-path", opts.PackageCachePath)
	// }

	// if len(opts.SysInputs) != 0 {
	// 	var inputsBuilder strings.Builder
	// 	for k, v := range opts.SysInputs {
	// 		inputsBuilder.WriteString(fmt.Sprintf("%s=%s", k, v))
	// 	}
	// 	args = append(args, "--input", inputsBuilder.String())
	// }

	if opts.PartialRender {
		args = append(args, "--partial-rendering")
	}
	if opts.OpenInBrowser {
		args = append(args, "--open", "--not-primary")
	} else {
		args = append(args, "--no-open")
	}

	args = append(args, targetFile)
	cmd := "tinymist.doStartPreview"
	if opts.OpenInBrowser {
		cmd = "tinymist.doStartBrowsingPreview"
	}

	result, err := p.client.ExecuteCommand(ctx, cmd, []any{args})
	if err != nil {
		log.Println("start previewer failed: ", err)
		return 0, err
	}

	// Try to open in built-in webview
	cmdResp, ok := result.(map[string]any)
	if !ok {
		panic("invalid cmd response type")
	}

	previewServerPort := cmdResp["staticServerPort"].(float64)

	return int(previewServerPort), nil
}

func (p *Previewer) killLspPreview(ctx context.Context) error {
	if p.lastTask == "" {
		return nil
	}

	_, err := p.client.ExecuteCommand(ctx, "tinymist.doKillPreview", []any{p.lastTask})
	if err != nil {
		log.Printf("kill preview task %s failed: %v", p.lastTask, err)
		return err
	}

	return nil
}

func (p *Previewer) scollLspPreview(ctx context.Context, req map[string]any) error {
	_, err := p.client.ExecuteCommand(ctx, "tinymist.scrollPreview", []any{p.lastTask, req})
	if err != nil {
		log.Println("scroll previewer failed: ", err)
		return err
	}

	return nil
}

func (p *Previewer) ScrollOnSelectionChange(ctx context.Context, pos gvcode.Position) {
	if !p.isPreviewing {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	req := map[string]any{
		"event":     "panelScrollTo",
		"filepath":  p.targetFile,
		"line":      pos.Line,
		"character": pos.Column,
	}
	p.scollLspPreview(ctx, req)

	// req2 := map[string]any{
	// 	"event":     "changeCursorPosition",
	// 	"filepath":  p.targetFile,
	// 	"line":      pos.Line,
	// 	"character": pos.Column,
	// }
	// p.scollLspPreview(ctx, req2)
}

func (p *Previewer) onWebviewClosed(ctx context.Context) error {
	p.isPreviewing = false
	err := p.killLspPreview(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *Previewer) clientRPCHandle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	if ctx.Err() != nil {
		return nil, RequestCancelledError
	}

	defer func() {
		if x := recover(); x != nil {
			log.Printf("panic in request, method: %s", req.Method)
		}
	}()

	switch req.Method {
	case rpcMethodNotifyClosing:
		var closeReq WebviewClosedNotifyReq
		err := UnmarshalJSON(req.Params, &closeReq)
		if err != nil {
			return nil, err
		}

		if closeReq.TargetFile == p.targetFile {
			p.onWebviewClosed(ctx)
			log.Println("received webview closing notify for file ", p.targetFile)
		}
		return nil, nil
	}

	return nil, jsonrpc2.ErrMethodNotFound
}

func (p *Previewer) IsPreviewing() bool {
	return p.isPreviewing
}

func clientHandler(previewer *Previewer) jsonrpc2.Handler {
	return jsonrpc2.HandlerFunc(func(ctx context.Context, req *jsonrpc2.Request) (any, error) {
		if ctx.Err() != nil {
			return nil, RequestCancelledError
		}
		return previewer.clientRPCHandle(ctx, req)
	})
}

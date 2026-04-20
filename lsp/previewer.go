package lsp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/oligo/gvcode"
)

type PreviewMode string

const (
	DocumentPreviewMode PreviewMode = "document"
	SlidePreviewMode    PreviewMode = "slide"
)

type PreviewOptions struct {
	Mode          PreviewMode
	ProjectRoot   string
	InvertColor   string
	PartialRender bool
}

type previewTask struct {
	opts       PreviewOptions
	serverAddr string
}

// PreviewService starts a global preview server for a project root. Tinymist then
// handles the currently 'focused' file based on LSP didOpen/didChange events. When
// the previewed files changes, the preview server refreshes based on its content,
// so there is not need to start a seperate preview server for each of the file in
// the root dir. This also prevents the issue 'skip compilation for ProjectInsId("primary")
// (or the taskID) due to harmless vfs changes'.
type PreviewService struct {
	client *Client
	task   *previewTask
	mu     sync.Mutex
}

func NewPreviwService(client *Client) *PreviewService {
	return &PreviewService{
		client: client,
	}
}

// Start starts a new preview server by calling a Tinymist LSP command.
// The server use a randomly selected port. If there is already one active
// task, it stops it first.
func (p *PreviewService) Start(ctx context.Context, opts PreviewOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	retry := 10
	for retry > 0 {
		if p.client.IsReady() {
			break
		}

		time.Sleep(1 * time.Second)
		retry--
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Kill existing preveiw first.
	p.killLspPreview(ctx)
	// Update tinymist.startDefaultPreview configs via workspace/didChangeConfiguration notification.
	err := p.updatePreview(ctx, opts)
	if err != nil {
		return err
	}

	// Lastly start a new preview server.
	previewServerPort, err := p.startPreview(ctx)
	if err != nil {
		return err
	}

	task := &previewTask{
		opts:       opts,
		serverAddr: fmt.Sprintf("http://127.0.0.1:%d", previewServerPort),
	}
	p.task = task
	return nil
}

func (p *PreviewService) Address() string {
	if p.task == nil {
		return ""
	}

	return p.task.serverAddr
}

func (p *PreviewService) Destroy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	err := p.killLspPreview(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *PreviewService) updatePreview(ctx context.Context, opts PreviewOptions) error {
	args := []any{
		"--data-plane-host=127.0.0.1:0",
		fmt.Sprintf("--preview-mode=%s", string(opts.Mode)),
		fmt.Sprintf("--invert-colors=%s", opts.InvertColor),
	}

	if opts.PartialRender {
		args = append(args, "--partial-rendering=true")
	}

	args = append(args, "--no-open")

	settings := map[string]any{
		"preview": map[string]any{
			"browsing": map[string]any{
				"args": args,
			},
		},
	}

	return p.client.NotifyWorkspaceConfigChanges(ctx, settings)

}

func (p *PreviewService) startPreview(ctx context.Context) (int, error) {
	result, err := p.client.ExecuteCommand(ctx, "tinymist.startDefaultPreview", nil)
	if err != nil {
		log.Println("start previewer failed: ", err)
		return 0, err
	}

	// Try to open in built-in webview
	cmdResp, ok := result.(map[string]any)
	if !ok {
		panic("invalid cmd response type")
	}

	log.Println("preview response: ", cmdResp)

	previewServerPort := cmdResp["staticServerPort"].(float64)

	return int(previewServerPort), nil
}

func (p *PreviewService) killLspPreview(ctx context.Context) error {
	if p.task == nil {
		return nil
	}

	_, err := p.client.ExecuteCommand(ctx, "tinymist.doKillPreview", []any{}) // kill all previews.
	if err != nil {
		log.Printf("kill preview failed: %v", err)
		return err
	}
	p.task = nil

	return nil
}

func (p *PreviewService) scollLspPreview(ctx context.Context) error {
	// With no arguments, this scroll all preview instances to the server’s inferred
	// current cursor/focus position.
	_, err := p.client.ExecuteCommand(ctx, "tinymist.scrollPreview", []any{})
	if err != nil {
		log.Println("scroll previewer failed: ", err)
		return err
	}

	return nil
}

func (p *PreviewService) ScrollOnSelectionChange(ctx context.Context, pos gvcode.Position) {
	if p.task == nil {
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p.scollLspPreview(ctx)
}

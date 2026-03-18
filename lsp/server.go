package lsp

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"golang.org/x/exp/jsonrpc2"
)

type Server struct {
	// project root directory with source code.
	workspace string
	// command that execute the LSP server.
	serverCmd *lspServer
	mu        sync.Mutex
	running   atomic.Bool
	logger    *slog.Logger
}

func newServer(workspace string) *Server {
	return &Server{
		workspace: workspace,
		logger:    slog.Default(),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running.CompareAndSwap(false, true) {
		s.logger.Info("server is already running!")
		return errors.New("server is already running!")
	}

	var err error
	s.serverCmd, err = newLspServer(ctx)
	if err != nil {
		s.logger.Error("start server failed", "error", err)
		return err
	}

	s.serverCmd.Dir = s.workspace

	s.logger.Info("Running LSP server")
	err = s.serverCmd.Start()
	if err != nil {
		err = errors.Wrapf(err, "failed to start server %s", s.serverCmd)
		s.running.CompareAndSwap(true, false)
		return err
	}

	go func() {
		if err := s.serverCmd.Wait(); err != nil {
			s.Stop()
			s.logger.Error("LSP server failed and exited!", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() {
	if s.running.CompareAndSwap(true, false) {
		if s.serverCmd.Cancel != nil {
			s.serverCmd.Cancel()
		}
	}
}

func (s *Server) IsRunning() bool {
	return s.running.Load()
}

func (s *Server) Workspace() string {
	return s.workspace
}

func (s *Server) Connect(client *Client) (*jsonrpc2.Connection, error) {
	return jsonrpc2.Dial(context.Background(), s.serverCmd, &lspClientBinder{client: client})
}

func (s *Server) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

func (s *Server) LogReader() io.Reader {
	if s.serverCmd == nil {
		return nil
	}
	return s.serverCmd.logReader()
}

type lspServer struct {
	*exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func newLspServer(ctx context.Context) (*lspServer, error) {
	c := &lspServer{Cmd: cmdBuilder.Build(ctx, "lsp")}

	// Get pipes for stdin and stdout of the lsp server process
	stdinPipe, err := c.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderrPipe, err := c.StderrPipe()
	if err != nil {
		return nil, err
	}

	c.stdin = stdinPipe
	c.stdout = stdoutPipe
	c.stderr = stderrPipe
	return c, nil
}

func (c *lspServer) Read(p []byte) (n int, err error) {
	return c.stdout.Read(p)
}

func (c *lspServer) Write(p []byte) (n int, err error) {
	return c.stdin.Write(p)
}

// Tinymist writes all server logs to stderr, so we have to
// read logs via stderr.
func (c *lspServer) logReader() io.Reader {
	return c.stderr
}

func (c *lspServer) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	if c.stderr != nil {
		c.stderr.Close()
	}
	return nil
}

func (c *lspServer) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return c, nil
}

type lspClientBinder struct {
	client *Client
}

// Bind is invoked when creating a new connection.
// The connection is not ready to use when Bind is called.
func (b *lspClientBinder) Bind(ctx context.Context, conn *jsonrpc2.Connection) (jsonrpc2.ConnectionOptions, error) {
	connOpt := jsonrpc2.ConnectionOptions{
		Framer:    jsonrpc2.HeaderFramer(),
		Preempter: nil,
		Handler:   jsonrpc2.HandlerFunc(b.client.Handle),
	}

	return connOpt, nil
}

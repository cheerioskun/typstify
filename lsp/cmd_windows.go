package lsp

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

const lspServerName = "tinymist.exe"

func newCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, lspServerName, args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

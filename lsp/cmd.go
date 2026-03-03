//go:build !windows
// +build !windows

package lsp

import (
	"context"
	"os"
	"os/exec"
)

const lspServerName = "tinymist"

func newCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, lspServerName, args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	return cmd
}

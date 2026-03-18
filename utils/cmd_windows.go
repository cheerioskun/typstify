package utils

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

func buildCmd(ctx context.Context, path string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

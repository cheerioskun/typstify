//go:build !windows
// +build !windows

package utils

import (
	"context"
	"os"
	"os/exec"
)

func buildCmd(ctx context.Context, path string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	return cmd
}

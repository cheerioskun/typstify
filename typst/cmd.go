//go:build !windows
// +build !windows

package typst

import (
	"context"
	"os"
	"os/exec"
)

const executableName = "typst"

func newCmd(ctx context.Context, args ...string) *exec.Cmd {
	args = append([]string{"--color=never"}, args...)
	cmd := exec.CommandContext(ctx, executableName, args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	return cmd
}

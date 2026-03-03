package typst

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

const executableName = "typst.exe"

func newCmd(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executableName, args...)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

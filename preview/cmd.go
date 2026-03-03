package preview

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
)

func SpawnPreview(ctx context.Context) (*exec.Cmd, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, errors.Join(err, errors.New("Spawning previewer failed"))
	}

	cmd := exec.CommandContext(ctx, exe, "preview")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Stderr = os.Stdout
	cmd.Stdout = os.Stdout

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		err := cmd.Wait()
		slog.Info("preview server stopped!", "exitErr", err)
	}()

	return cmd, nil
}

package update

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
)

// StartStagedAndWait reuses final hash/size verification and waits for the
// installer rather than exiting its launcher. Cancellation never kills an
// installer midway through a transaction; once started we wait for its exit.
func StartStagedAndWait(ctx context.Context, result ApplyResult) error {
	if runtime.GOOS != "windows" || !CanStartStaged(result) {
		return errors.New("managed installer requires Windows")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := startStaged(result, func(path string) error {
		cmd := exec.Command(path)
		if err := cmd.Start(); err != nil {
			return err
		}
		// Keep maintenance admission until the installer actually exits.
		return cmd.Wait()
	})
	return err
}

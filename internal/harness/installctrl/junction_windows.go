//go:build windows

package installctrl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func createWindowsJunction(link, target string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	_ = os.Remove(link)
	// mklink /J creates a directory junction without elevated privileges.
	cmd := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

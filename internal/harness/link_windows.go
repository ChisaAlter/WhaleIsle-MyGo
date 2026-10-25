package harness

import (
	"fmt"
	"os/exec"

	"whaleisle/internal/processutil"
)

func symlinkDir(source, target string) error {
	// NTFS junctions do not require SeCreateSymbolicLinkPrivilege (unlike
	// directory symlinks). Match Electron's fs.symlinkSync(..., 'junction').
	cmd := exec.Command("cmd", "/c", "mklink", "/J", target, source)
	processutil.Background(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J %s -> %s: %w (%s)", target, source, err, string(out))
	}
	return nil
}

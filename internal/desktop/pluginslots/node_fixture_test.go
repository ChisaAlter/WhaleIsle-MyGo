package pluginslots

import (
	"os/exec"
	"whaleisle/internal/processutil"
)

func runNodeFixture(node string, args ...string) ([]byte, error) {
	cmd := exec.Command(node, args...)
	processutil.Background(cmd)
	return cmd.CombinedOutput()
}

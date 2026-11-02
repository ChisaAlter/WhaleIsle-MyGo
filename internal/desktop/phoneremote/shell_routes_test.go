package phoneremote

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/processutil"
)

// The composed bridge script must carry the shell route module; otherwise the
// daemon would start without them and the phone's git/list calls would 404/405.
func TestComposedBridgeScriptCarriesShellRoutes(t *testing.T) {
	script := composedBridgeScript()
	for _, needle := range []string{
		"const __mygoShellRoutes=",
		"/__remote__/shell/",
		"createShellHandler",
		"timingSafeEqual",
		"const originalGit=loadOriginalGit(",
	} {
		if !strings.Contains(script, needle) {
			t.Fatalf("composed bridge script missing %q", needle)
		}
	}
	if strings.Count(script, "const __mygoShellRoutes=") != 1 {
		t.Fatal("shell routes module embedded more than once")
	}
}

// TestShellRoutesNodeTable runs the request table with the bundled Node when it
// is available; it is skipped on machines without Node rather than silently
// passing.
func TestShellRoutesNodeTable(t *testing.T) {
	node := nodeBinary(t)
	if node == "" {
		t.Skip("node not available")
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "--test", filepath.Join(dir, "shell_routes.test.cjs"))
	processutil.Background(cmd)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node --test failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "fail 0") {
		t.Fatalf("node test table reported failures:\n%s", out)
	}
}

func nodeBinary(t *testing.T) string {
	t.Helper()
	if path, err := exec.LookPath("node"); err == nil {
		return path
	}
	candidates := []string{
		filepath.Join("..", "..", "..", "runtime", "node", "node.exe"),
		`C:\软件\Whale Isle\resources\runtime\primary-runtime\dependencies\node\bin\node.exe`,
	}
	for _, candidate := range candidates {
		if abs, err := filepath.Abs(candidate); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return ""
}

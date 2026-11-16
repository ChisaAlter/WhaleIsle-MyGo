package harness

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/paths"
)

func TestManagedWhaleUIValidatorEnvironment(t *testing.T) {
	home := t.TempDir()
	path, err := prepareWhaleUIHost(home)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) != string(openui.WhaleHostValidator()) {
		t.Fatal("validator differs", err)
	}
	t.Setenv("DSHD_WHALE_UI_HOST", "foreign-validator")
	managed := strings.Join(childEnv(LaunchOptions{DesktopDefaults: true, WhaleUIHost: path}), "\n")
	for _, required := range []string{"DSHD_WHALE_UI_ENABLED=1", "DSHD_WHALE_UI_HOST=" + path} {
		if !strings.Contains(managed, required) {
			t.Fatal(required)
		}
	}
	if strings.Contains(managed, "=foreign") {
		t.Fatal("inherited desktop env overrode owned composition")
	}
	external := strings.Join(childEnv(LaunchOptions{}), "\n")
	if !strings.Contains(external, "DSHD_WHALE_UI_HOST=foreign-validator") {
		t.Fatal("changed configured launcher contract")
	}
}

func TestPlanStartSingleResolveAppliesPreparedOverlays(t *testing.T) {
	base := t.TempDir()
	t.Setenv(paths.EnvUserData, filepath.Join(base, "user-data"))
	vendor := filepath.Join(base, "vendor")
	root := filepath.Join(vendor, "deepseek-harness")
	writeSessionRuntimeRegistry(t, root, desktopSessionEventTypes)
	for _, rel := range []string{"apps/cli/lib/bin.js", "apps/web/dist/index.html"} {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("// fixture"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"dsh-task-control", "dsh-platform-session", "dsh-usage-panel", "dsh-im"} {
		dir := filepath.Join(vendor, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"`+name+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	node := filepath.Join(base, "node", nodeExecutableName())
	if err := os.MkdirAll(filepath.Dir(node), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(node, []byte("node"), 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := PlanStart(LaunchOptions{Host: "127.0.0.1", VendorDir: vendor, HarnessRoot: root, NodeBin: node}, PrepareOptions{RemoteSet: true, Remote: false}, 3080)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Command != node {
		t.Fatalf("Command = %q, want %q", plan.Command, node)
	}
	if !slices.Contains(plan.Args, "--import") {
		t.Fatalf("missing proactive preload --import in Args: %v", plan.Args)
	}
	joined := strings.Join(plan.PatchFiles, "\n")
	if !strings.Contains(joined, routeAOverlayFile) {
		t.Fatalf("missing route-a overlay in PlanStart PatchFiles: %v", plan.PatchFiles)
	}
	for _, p := range plan.PatchFiles {
		if !slices.Contains(plan.Args, p) {
			t.Fatalf("Plan.Args missing patch %s: %v", p, plan.Args)
		}
	}
	if !strings.Contains(strings.Join(plan.Env, "\n"), "DSHD_WHALE_UI_ENABLED=1") {
		t.Fatalf("missing DSHD_WHALE_UI_ENABLED in Env: %v", plan.Env)
	}
}

func TestResolveNodePrefersSiblingAndPairedRC4(t *testing.T) {
	base := t.TempDir()
	t.Chdir(base)
	t.Setenv(paths.EnvVendor, "")
	t.Setenv(paths.EnvNodeDir, "")

	rcNode := filepath.Join(base, "runtime", "node", nodeExecutableName())
	rc4Harness := filepath.Join(base, "runtime-rc4", "vendor", "deepseek-harness")
	rc4Node := filepath.Join(base, "runtime-rc4", "node", nodeExecutableName())
	for _, dir := range []string{filepath.Dir(rcNode), rc4Harness, filepath.Dir(rc4Node)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(rcNode, []byte("rc2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rc4Node, []byte("rc4"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveNode("")
	if err != nil || got != rc4Node {
		t.Fatalf("resolveNode(\"\") = %q, %v; want %q", got, err, rc4Node)
	}
	// Sibling vendor lookup beats working-directory rc4 when VendorDir differs.
	gotSibling, err := resolveNodeForVendor("", filepath.Join(base, "runtime", "vendor"))
	if err != nil || gotSibling != rcNode {
		t.Fatalf("resolveNodeForVendor sibling = %q, %v; want %q", gotSibling, err, rcNode)
	}
}

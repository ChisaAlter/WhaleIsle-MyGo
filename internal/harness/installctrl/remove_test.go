package installctrl

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestOfflineRemoveCommandFixture(t *testing.T) {
	root := t.TempDir()
	node := filepath.Join(root, "node")
	bin := filepath.Join(root, "apps", "cli", "lib", "bin.js")
	if err := os.MkdirAll(filepath.Dir(bin), 0700); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{node, bin} {
		if err := os.WriteFile(file, []byte("fixture, never executed"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	cfg := InstallConfig{NodeBin: node, HarnessRoot: root, ProfileDir: root, Runner: func(ctx context.Context, dir, cmd string, args ...string) (int, string, error) {
		calls++
		if dir != root || cmd != node || !slices.Equal(args, []string{bin, "plugin", "--profile", "web", "remove", "demo"}) {
			t.Fatalf("wrong offline CLI: %s %s %v", dir, cmd, args)
		}
		return 0, "removed demo", nil
	}}
	result := RemovePluginContext(context.Background(), cfg, "demo")
	if !result.OK || calls != 1 {
		t.Fatalf("%+v calls=%d", result, calls)
	}
	result = RemovePluginContext(context.Background(), cfg, "../escape")
	if result.OK || calls != 1 {
		t.Fatal("invalid name invoked CLI")
	}
}

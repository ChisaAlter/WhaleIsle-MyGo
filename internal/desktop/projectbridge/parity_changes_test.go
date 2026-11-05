package projectbridge

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/processutil"
)

func TestPublishedRequirementChangesDispatch(t *testing.T) {
	// Only this test's disposable repository is changed. No configured workspace,
	// user Git profile, remote, Host, model or credential is used.
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node unavailable")
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_CONFIG_COUNT"} {
		value, present := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if present {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
	root, home := t.TempDir(), t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(git, args...)
		cmd.Dir = root
		processutil.BackgroundTree(cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return string(out)
	}
	run("init")
	file := filepath.Join(root, "result.txt")
	write := func(text string) {
		t.Helper()
		if err := os.WriteFile(file, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("base\n")
	run("add", "--all")
	run("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgSign=false", "-c", "core.hooksPath="+t.TempDir(), "commit", "-m", "fixture")
	b := New(Options{NodeBin: node, Home: home, Workspace: root})
	call := func(action string, more map[string]any) map[string]any {
		t.Helper()
		payload := map[string]any{"action": action, "workingDirectory": root}
		for k, v := range more {
			payload[k] = v
		}
		result := b.Dispatch(context.Background(), payload)
		if result["ok"] != true {
			t.Fatalf("%s: %v", action, result)
		}
		return result
	}
	if check := call("check-directory", nil); check["allowed"] != true {
		t.Fatal(check)
	}
	base := call("baseline", nil)["tree"]
	if base == nil || base == "" {
		t.Fatal("baseline did not capture a tree")
	}
	index := run("diff", "--cached")
	write("changed\n")
	changes := call("changes", map[string]any{"base": base, "diff": true})
	if !strings.Contains(changes["diff"].(string), "changed") {
		t.Fatal("change content missing", changes)
	}
	end := changes["end"]
	if run("diff", "--cached") != index {
		t.Fatal("baseline/change capture altered the user's index")
	}
	preview := call("revert", map[string]any{"base": base, "end": end, "checkOnly": true})
	if preview["applied"] != false {
		t.Fatal("preflight applied changes", preview)
	}
	write("foreign\n")
	conflict := call("revert", map[string]any{"base": base, "end": end})
	if conflict["conflict"] != true || conflict["applied"] != false {
		t.Fatal("conflict was not retained", conflict)
	}
	if body, _ := os.ReadFile(file); string(body) != "foreign\n" {
		t.Fatal("conflicting file changed")
	}
	write("changed\n")
	if reverted := call("revert", map[string]any{"base": base, "end": end}); reverted["applied"] != true {
		t.Fatal(reverted)
	}
	if body, _ := os.ReadFile(file); string(body) != "base\n" {
		t.Fatal("recorded change did not revert")
	}
}

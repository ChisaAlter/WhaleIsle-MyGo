package projectbridge

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"whaleisle/internal/processutil"
)

func TestWindowsLongWorktreeCheckoutAndInterruptedContent(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git for Windows checkout child contract")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	if err = os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	projectID := "project-01234567-0123-0123-0123-012345678901"
	workID := "work-01234567-0123-0123-0123-012345678901"
	// Keep the full, previously failing directory shape. Do not shorten fixture
	// paths to avoid the inherited-GIT_DIR failure reproduced by real Host work.
	for len(filepath.Join(home, "project-worktrees", projectID, workID)) < 225 {
		home = filepath.Join(home, "padding")
	}
	if err = os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "no-global-config"))
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_CONFIG_COUNT"} {
		value, present := os.LookupEnv(key)
		if err = os.Unsetenv(key); err != nil {
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
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(git, append([]string{"-c", "core.longpaths=true", "-C", source}, args...)...)
		processutil.BackgroundTree(cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture Git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "--quiet")
	file := filepath.Join(source, "result.txt")
	if err = os.WriteFile(file, []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "--", "result.txt")
	run("-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgSign=false", "-c", "core.hooksPath="+home, "commit", "--quiet", "-m", "owned fixture")
	if err = os.WriteFile(file, []byte("source dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bridge := New(Options{NodeBin: node, Home: home, Workspace: source})
	payload := map[string]any{"action": "prepare-workspace", "projectId": projectID, "workstreamId": workID, "workingDirectory": source, "mode": "worktree"}
	result := bridge.Dispatch(context.Background(), payload)
	if result["ok"] != true {
		t.Fatalf("long-path worktree: %v", result)
	}
	target, _ := result["canonicalPath"].(string)
	if len(target) < 225 || target != filepath.Join(home, "project-worktrees", projectID, workID) || result["ownership"] != "project" {
		t.Fatalf("actual full directory ownership: %v", result)
	}
	targetFile := filepath.Join(target, "result.txt")
	if data, err := os.ReadFile(targetFile); err != nil || string(data) != "base\n" {
		t.Fatalf("actual clean checkout: %q %v", data, err)
	}
	receipt := result["receipt"]
	if err = os.WriteFile(targetFile, []byte("worker edit\n"), 0600); err != nil {
		t.Fatal(err)
	}
	again := bridge.Dispatch(context.Background(), payload)
	if again["ok"] != true || again["receipt"] != receipt {
		t.Fatalf("ready reuse: %v", again)
	}
	if data, _ := os.ReadFile(targetFile); string(data) != "worker edit\n" {
		t.Fatal("ready reuse reset worktree content")
	}
	// Simulate a crash before marking the newly checked-out workspace ready.
	// A retry must preserve an intervening edit rather than repeat reset --hard.
	bindingFile := filepath.Join(home, "projects", projectID, "internal", "desktop-workspaces", workID+".json")
	data, err := os.ReadFile(bindingFile)
	if err != nil {
		t.Fatal(err)
	}
	var binding map[string]any
	if err = json.Unmarshal(data, &binding); err != nil {
		t.Fatal(err)
	}
	binding["state"] = "preparing"
	binding["checkoutPending"] = true
	data, err = json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(bindingFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	refused := bridge.Dispatch(context.Background(), payload)
	if refused["ok"] != false {
		t.Fatalf("interrupted dirty checkout was adopted: %v", refused)
	}
	if data, _ := os.ReadFile(targetFile); string(data) != "worker edit\n" {
		t.Fatal("interrupted checkout discarded actual content")
	}
	// A fully completed, clean initial checkout still recovers its original receipt.
	if err = os.WriteFile(targetFile, []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	recovered := bridge.Dispatch(context.Background(), payload)
	if recovered["ok"] != true || recovered["receipt"] != receipt {
		t.Fatalf("clean interrupted checkout recovery: %v", recovered)
	}
	if data, _ := os.ReadFile(file); string(data) != "source dirty\n" {
		t.Fatal("source dirty file changed")
	}
}

func TestWindowsLegacyPreparingWorktreeNeedsCompleteCheckout(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Git for Windows legacy preparing receipt")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	source, home := filepath.Join(root, "source"), filepath.Join(root, "home")
	if err = os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "no-global-config"))
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_CONFIG_COUNT"} {
		value, present := os.LookupEnv(key)
		if err = os.Unsetenv(key); err != nil {
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
	run := func(cwd string, args ...string) string {
		t.Helper()
		cmd := exec.Command(git, append([]string{"-c", "core.longpaths=true", "-C", cwd}, args...)...)
		processutil.BackgroundTree(cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("legacy fixture Git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run(source, "init", "--quiet")
	if err = os.WriteFile(filepath.Join(source, "result.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run(source, "add", "--", "result.txt")
	run(source, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgSign=false", "-c", "core.hooksPath="+home, "commit", "--quiet", "-m", "owned legacy fixture")
	head := run(source, "rev-parse", "HEAD")
	projectID, workID := "project-legacy-preparing", "work-legacy-preparing"
	branch := "project/" + projectID + "/" + workID
	target := filepath.Join(home, "project-worktrees", projectID, workID)
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	// Actual Git registration and HEAD, but deliberately no checkout: the exact
	// state an older worktree-add may leave before its checkout child succeeds.
	run(source, "worktree", "add", "--no-checkout", "-b", branch, target, head)
	bridge := New(Options{NodeBin: node, Home: home, Workspace: source})
	repository := bridge.Dispatch(context.Background(), map[string]any{"action": "repository", "workingDirectory": source})
	if repository["ok"] != true {
		t.Fatal(repository)
	}
	binding := map[string]any{"projectId": projectID, "workstreamId": workID, "mode": "worktree", "sourceIdentity": strings.ToLower(source), "canonicalPath": target, "requestedBase": nil, "directoryIdentity": strings.ToLower(target), "ownership": "project", "branch": branch, "baseCommit": head, "repoIdentity": repository["repoIdentity"], "repositoryRoot": source, "state": "preparing", "receipt": "owned-legacy-receipt"}
	bindingFile := filepath.Join(home, "projects", projectID, "internal", "desktop-workspaces", workID+".json")
	if err = os.MkdirAll(filepath.Dir(bindingFile), 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(bindingFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(target, "keep-owned.txt")
	if err = os.WriteFile(keep, []byte("keep original content\n"), 0600); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"action": "prepare-workspace", "projectId": projectID, "workstreamId": workID, "workingDirectory": source, "mode": "worktree"}
	refused := bridge.Dispatch(context.Background(), payload)
	if refused["ok"] != false {
		t.Fatalf("legacy unmaterialized worktree marked ready: %v", refused)
	}
	if _, err = os.Stat(filepath.Join(target, "result.txt")); !os.IsNotExist(err) {
		t.Fatal("legacy retry checked out or replaced missing file")
	}
	if content, _ := os.ReadFile(keep); string(content) != "keep original content\n" {
		t.Fatal("legacy retry changed existing file")
	}
	raw, err = os.ReadFile(bindingFile)
	if err != nil {
		t.Fatal(err)
	}
	var retained map[string]any
	if err = json.Unmarshal(raw, &retained); err != nil {
		t.Fatal(err)
	}
	if retained["state"] != "preparing" || retained["receipt"] != binding["receipt"] || retained["branch"] != branch || retained["checkoutPending"] != nil {
		t.Fatalf("legacy receipt or branch changed: %v", retained)
	}
	if got := run(target, "symbolic-ref", "--short", "HEAD"); got != branch {
		t.Fatal("legacy branch replaced", got)
	}
	// An explicitly completed clean checkout can then recover the same legacy
	// receipt. This setup repair affects only this test's disposable files.
	if err = os.Remove(keep); err != nil {
		t.Fatal(err)
	}
	run(target, "reset", "--hard", head)
	recovered := bridge.Dispatch(context.Background(), payload)
	if recovered["ok"] != true || recovered["receipt"] != binding["receipt"] || recovered["branch"] != branch {
		t.Fatalf("clean legacy receipt could not recover: %v", recovered)
	}
}

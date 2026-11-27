package installctrl

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestPNPMInstall_success(t *testing.T) {
	var gotDir, gotBin string
	var gotArgs []string
	profile := t.TempDir()
	cfg := InstallConfig{
		ProfileDir:  profile,
		DshHome:     t.TempDir(),
		HarnessRoot: fakeHarnessRoot(t),
		NodeBin:     fakeNode(t),
		Runner: func(ctx context.Context, dir, name string, args ...string) (int, string, error) {
			gotDir, gotBin = dir, name
			gotArgs = append([]string(nil), args...)
			plantLoadablePackage(t, profile, "widget")
			return 0, "added ok\n", nil
		},
	}
	result := PNPMInstall(cfg)("github:acme/widget", InstallOpts{})
	if !result.OK || result.Spec != "github:acme/widget" || result.Error != "" {
		t.Fatalf("result = %+v", result)
	}
	if gotBin != cfg.NodeBin {
		t.Fatalf("node = %q", gotBin)
	}
	if len(gotArgs) < 5 || gotArgs[1] != "plugin" || gotArgs[len(gotArgs)-2] != "add" {
		t.Fatalf("args = %v", gotArgs)
	}
	if gotDir == "" {
		t.Fatal("empty cwd")
	}
}

func TestPNPMInstall_needsAllowBuilds(t *testing.T) {
	cfg := InstallConfig{
		ProfileDir:  t.TempDir(),
		DshHome:     t.TempDir(),
		HarnessRoot: fakeHarnessRoot(t),
		NodeBin:     fakeNode(t),
		Runner: func(context.Context, string, string, ...string) (int, string, error) {
			return 1, "ERR_PNPM ignored build scripts: esbuild\n", errExitCode
		},
	}
	result := PNPMInstall(cfg)("github:acme/widget", InstallOpts{})
	if result.OK || !result.NeedsAllowBuilds {
		t.Fatalf("result = %+v", result)
	}
	if len(result.AllowBuilds) != 1 || result.AllowBuilds[0] != "esbuild" {
		t.Fatalf("allowBuilds = %v", result.AllowBuilds)
	}
}

func TestPNPMInstall_mergesAllowBuilds(t *testing.T) {
	profile := t.TempDir()
	cfg := InstallConfig{
		ProfileDir:  profile,
		DshHome:     t.TempDir(),
		HarnessRoot: fakeHarnessRoot(t),
		NodeBin:     fakeNode(t),
		Runner: func(context.Context, string, string, ...string) (int, string, error) {
			plantLoadablePackage(t, profile, "widget")
			return 0, "", nil
		},
	}
	result := PNPMInstall(cfg)("github:acme/widget", InstallOpts{AllowBuilds: []string{"esbuild"}})
	if !result.OK {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(filepath.Join(profile, "pnpm-workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"esbuild": true`) {
		t.Fatalf("workspace yaml = %q", data)
	}
}

func TestPNPMInstall_rollbackUnloadable(t *testing.T) {
	profile := t.TempDir()
	if err := os.WriteFile(filepath.Join(profile, "package.json"), []byte(`{"name":"web"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := InstallConfig{
		ProfileDir:  profile,
		DshHome:     t.TempDir(),
		HarnessRoot: fakeHarnessRoot(t),
		NodeBin:     fakeNode(t),
		Runner: func(context.Context, string, string, ...string) (int, string, error) {
			dir := filepath.Join(profile, "node_modules", "junk")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"junk","version":"1.0.0"}`), 0o644); err != nil {
				t.Fatal(err)
			}
			return 0, "", nil
		},
	}
	result := PNPMInstall(cfg)("github:acme/junk", InstallOpts{})
	if result.OK || !result.RolledBack {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(profile, "node_modules", "junk")); !os.IsNotExist(err) {
		t.Fatalf("junk should be rolled back: %v", err)
	}
	if _, err := os.Stat(filepath.Join(profile, "package.json")); err != nil {
		t.Fatal(err)
	}
}

func TestPNPMInstall_notReady(t *testing.T) {
	result := PNPMInstall(InstallConfig{})("github:a/b", InstallOpts{})
	if result.OK || result.Error == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestInstallReady(t *testing.T) {
	root := fakeHarnessRoot(t)
	node := fakeNode(t)
	if !InstallReady(InstallConfig{NodeBin: node, HarnessRoot: root}) {
		t.Fatal("expected ready")
	}
	if InstallReady(InstallConfig{NodeBin: node, HarnessRoot: t.TempDir()}) {
		t.Fatal("missing bin.js should not be ready")
	}
}

func fakeHarnessRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "apps", "cli", "lib", "bin.js")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("// stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func fakeNode(t *testing.T) string {
	t.Helper()
	name := "node"
	if runtime.GOOS == "windows" {
		name = "node.exe"
	}
	p := filepath.Join(t.TempDir(), name)
	mode := os.FileMode(0o644)
	if runtime.GOOS != "windows" {
		mode = 0o755
	}
	if err := os.WriteFile(p, []byte(""), mode); err != nil {
		t.Fatal(err)
	}
	return p
}

func plantLoadablePackage(t *testing.T, profile, name string) {
	t.Helper()
	dir := filepath.Join(profile, "node_modules", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := `{"name":"` + name + `","version":"1.0.0","main":"index.js"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte("module.exports = {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPNPMInstall_timeout(t *testing.T) {
	cfg := InstallConfig{
		ProfileDir:  t.TempDir(),
		DshHome:     t.TempDir(),
		HarnessRoot: fakeHarnessRoot(t),
		NodeBin:     fakeNode(t),
		Timeout:     20 * time.Millisecond,
		Runner: func(ctx context.Context, dir, name string, args ...string) (int, string, error) {
			<-ctx.Done()
			return 1, "slow", ctx.Err()
		},
	}
	result := PNPMInstall(cfg)("github:acme/widget", InstallOpts{})
	if result.OK || !strings.Contains(result.Error, "超时") {
		t.Fatalf("result = %+v", result)
	}
}

func TestHelperPluginLockTree(t *testing.T) {
	mode := os.Getenv("WHALEISLE_TEST_PLUGIN_TREE")
	if mode == "" {
		return
	}
	lockFile := os.Getenv("WHALEISLE_TEST_LOCK_FILE")
	readyFile := os.Getenv("WHALEISLE_TEST_READY_FILE")
	if mode == "grandchild" {
		if err := os.MkdirAll(filepath.Dir(lockFile), 0o755); err != nil {
			os.Exit(2)
		}
		f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
		if err != nil {
			os.Exit(3)
		}
		_, _ = f.WriteString("locked by grandchild\n")
		_ = os.WriteFile(readyFile, []byte("ready"), 0o644)
		time.Sleep(time.Minute)
		_ = f.Close()
		os.Exit(0)
	}
	if mode == "parent" {
		exe, err := os.Executable()
		if err != nil {
			os.Exit(4)
		}
		child := exec.Command(exe, "-test.run=^TestHelperPluginLockTree$")
		child.Env = append(os.Environ(), "WHALEISLE_TEST_PLUGIN_TREE=grandchild")
		if err := child.Start(); err != nil {
			os.Exit(5)
		}
		time.Sleep(time.Minute)
		os.Exit(0)
	}
}

func TestExecPluginRunnerCancelKillsGrandchildTreeBeforeRollback(t *testing.T) {
	profile := t.TempDir()
	if err := os.WriteFile(filepath.Join(profile, "package.json"), []byte(`{"name":"web"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	lockFile := filepath.Join(profile, "node_modules", "locked-pkg", "lock.txt")
	readyFile := filepath.Join(t.TempDir(), "ready.flag")

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WHALEISLE_TEST_PLUGIN_TREE", "parent")
	t.Setenv("WHALEISLE_TEST_LOCK_FILE", lockFile)
	t.Setenv("WHALEISLE_TEST_READY_FILE", readyFile)

	cfg := InstallConfig{
		ProfileDir:  profile,
		DshHome:     t.TempDir(),
		HarnessRoot: fakeHarnessRoot(t),
		NodeBin:     exe,
	}
	realRunner := execPluginRunner(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg.Runner = func(runCtx context.Context, dir, _ string, _ ...string) (int, string, error) {
		return realRunner(runCtx, dir, exe, "-test.run=^TestHelperPluginLockTree$")
	}

	done := make(chan Result, 1)
	go func() {
		done <- PNPMInstallContext(ctx, cfg)("github:acme/locked-pkg", InstallOpts{})
	}()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(readyFile); err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if _, err := os.Stat(readyFile); err != nil {
		cancel()
		t.Fatalf("grandchild did not acquire lock file in time: %v", err)
	}

	cancel()

	select {
	case res := <-done:
		if res.OK {
			t.Fatalf("expected cancelled install to fail, got %+v", res)
		}
		if !res.RolledBack || res.RollbackError != "" {
			t.Fatalf("expected clean rollback after killing grandchild tree, got %+v", res)
		}
		if _, err := os.Stat(filepath.Join(profile, "node_modules", "locked-pkg")); !os.IsNotExist(err) {
			t.Fatalf("locked-pkg remained after rollback: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("cancelled install did not finish after killing grandchild tree")
	}
}

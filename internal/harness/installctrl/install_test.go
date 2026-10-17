package installctrl

import (
	"context"
	"os"
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

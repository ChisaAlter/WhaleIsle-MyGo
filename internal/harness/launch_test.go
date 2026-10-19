package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePrefersVendoredHarness(t *testing.T) {
	vendor := t.TempDir()
	root := filepath.Join(vendor, "deepseek-harness")
	writeSessionRuntimeRegistry(t, root, desktopSessionEventTypes)
	bin := filepath.Join(root, "apps", "cli", "lib", "bin.js")
	web := filepath.Join(root, "apps", "web", "dist", "index.html")
	for _, p := range []string{bin, web} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("//"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	node := filepath.Join(t.TempDir(), "node.exe")
	if err := os.WriteFile(node, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(t.TempDir(), "a.patch.yml")
	_ = os.WriteFile(patch, []byte("- insert:\n"), 0o644)

	p, err := Resolve(LaunchOptions{
		Host: "127.0.0.1", NodeBin: node, VendorDir: vendor,
		PatchFiles: []string{patch}, SkipUserPlugins: true,
	}, 3080)
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != "source" || p.Command != node {
		t.Fatalf("plan %+v", p)
	}
	args := strings.Join(p.Args, " ")
	if !strings.Contains(args, bin) || !strings.Contains(args, "--patch "+patch) {
		t.Fatalf("args %v", p.Args)
	}
	// Electron order: web, skip, patches, host, port, no-open
	wantPrefix := []string{bin, "web", "--skip-user-plugins", "--patch", patch, "--host", "127.0.0.1", "--port", "3080", "--no-open"}
	if strings.Join(p.Args, "\x00") != strings.Join(wantPrefix, "\x00") {
		t.Fatalf("args order:\n got %v\nwant %v", p.Args, wantPrefix)
	}
	env := strings.Join(p.Env, "\n")
	if !strings.Contains(env, "DSH_HARNESS_ROOT="+root) {
		t.Fatalf("missing DSH_HARNESS_ROOT in\n%s", env)
	}
}

func TestResolvePATHWhenNoVendor(t *testing.T) {
	node := filepath.Join(t.TempDir(), "node")
	_ = os.WriteFile(node, nil, 0o755)
	defer func(orig func(string) (string, error)) { lookPath = orig }(lookPath)
	lookPath = func(name string) (string, error) {
		if name == "dsh" {
			return `/usr/bin/dsh`, nil
		}
		return "", os.ErrNotExist
	}
	p, err := Resolve(LaunchOptions{Host: "127.0.0.1", NodeBin: node, VendorDir: t.TempDir()}, 3080)
	if err != nil || p.Kind != "PATH" || p.Command != `/usr/bin/dsh` {
		t.Fatalf("got %+v %v", p, err)
	}
}

func TestResolveNpxRequiresAllow(t *testing.T) {
	nodeDir := t.TempDir()
	node := filepath.Join(nodeDir, "node.exe")
	npx := filepath.Join(nodeDir, "npx.cmd")
	_ = os.WriteFile(node, nil, 0o755)
	_ = os.WriteFile(npx, nil, 0o755)
	defer func(orig func(string) (string, error)) { lookPath = orig }(lookPath)
	lookPath = func(string) (string, error) { return "", os.ErrNotExist }

	if _, err := Resolve(LaunchOptions{Host: "127.0.0.1", NodeBin: node, VendorDir: t.TempDir()}, 3080); err == nil {
		t.Fatal("expected error without AllowNpx")
	}
	p, err := Resolve(LaunchOptions{
		Host: "127.0.0.1", NodeBin: node, VendorDir: t.TempDir(),
		AllowNpx: true, Version: "0.2.0-rc.2", SkipUserPlugins: true,
	}, 3081)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--yes", "@deepseek-ai/dsh@0.2.0-rc.2", "web", "--skip-user-plugins", "--host", "127.0.0.1", "--port", "3081", "--no-open"}
	if p.Command != npx || strings.Join(p.Args, " ") != strings.Join(want, " ") {
		t.Fatalf("npx plan cmd=%q args=%v", p.Command, p.Args)
	}
}

func TestChildEnvSetsHome(t *testing.T) {
	t.Setenv("ELECTRON_RUN_AS_NODE", "1")
	t.Setenv("DSH_HOME", "/elsewhere")
	env := strings.Join(childEnv(LaunchOptions{Home: "/data/dsh-home"}), "\n")
	if strings.Contains(env, "ELECTRON_RUN_AS_NODE") || strings.Contains(env, "/elsewhere") || !strings.Contains(env, "DSH_HOME=/data/dsh-home") {
		t.Fatalf("env:\n%s", env)
	}
}

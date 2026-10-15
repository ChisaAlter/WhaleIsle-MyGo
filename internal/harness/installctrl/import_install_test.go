package installctrl

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryPluginSpecAndExistingPackageResolution(t *testing.T) {
	for _, spec := range []string{"fixture@1.2.3", "fixture@^1.2.3", "@fixture/plugin@~1.2.3-beta.1+build"} {
		name := RegistryPluginName(spec)
		if name == "" { t.Fatalf("valid spec rejected: %q", spec) }
		if got := ResolveAddedPackageNames(spec, []string{name}, []string{name}); !reflect.DeepEqual(got, []string{name}) { t.Fatalf("existing package not validated: %v", got) }
	}
	for _, spec := range []string{"fixture", "fixture@latest", "fixture@*", "fixture@npm:other@1.0.0", "fixture@file:../x", "fixture@1.0.0 --flag", "@fixture/plugin@>=1.0.0"} {
		if RegistryPluginName(spec) != "" { t.Fatalf("unsafe spec accepted: %q", spec) }
	}
}

func TestPNPMInstallContextCancelledBeforeWrites(t *testing.T) {
	root := t.TempDir()
	profile := filepath.Join(root, "profile")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	result := PNPMInstallContext(ctx, InstallConfig{
		ProfileDir: profile,
		Runner: func(context.Context, string, string, ...string) (int, string, error) { called = true; return 0, "", nil },
	})("fixture@1.0.0", InstallOpts{})
	if result.OK || called { t.Fatalf("cancelled install ran: %+v", result) }
	if _, err := os.Stat(profile); !os.IsNotExist(err) { t.Fatalf("cancelled install wrote destination: %v", err) }
}

func TestPNPMInstallContextCancellationRollsBackApproval(t *testing.T) {
	profile := t.TempDir()
	workspace := filepath.Join(profile, "pnpm-workspace.yaml")
	if err := os.WriteFile(workspace, []byte("packages: []\n"), 0600); err != nil { t.Fatal(err) }
	ctx, cancel := context.WithCancel(context.Background())
	cfg := InstallConfig{
		ProfileDir: profile, DshHome: t.TempDir(), NodeBin: fakeNode(t), HarnessRoot: fakeHarnessRoot(t),
		Runner: func(ctx context.Context, dir, name string, args ...string) (int, string, error) {
			data, err := os.ReadFile(workspace)
			if err != nil || !strings.Contains(string(data), "esbuild") { t.Fatalf("approved build missing: %s %v", data, err) }
			plantLoadablePackage(t, profile, "fixture")
			cancel()
			return 1, "fixture cancellation", ctx.Err()
		},
	}
	result := PNPMInstallContext(ctx, cfg)("fixture@1.0.0", InstallOpts{AllowBuilds: []string{"esbuild"}})
	if result.OK || !result.RolledBack || !strings.Contains(result.Error, "取消") { t.Fatalf("cancellation: %+v", result) }
	data, err := os.ReadFile(workspace)
	if err != nil || string(data) != "packages: []\n" { t.Fatalf("approval mutation survived cancellation: %s %v", data, err) }
}

func TestPNPMInstallContextReinstallsExistingExpectedPackage(t *testing.T) {
	for _, spec := range []string{"fixture@^1.2.3", "github:fixture/different-repo#v1"} {
		t.Run(spec, func(t *testing.T) {
			profile := t.TempDir()
			plantLoadablePackage(t, profile, "fixture")
			cfg := InstallConfig{
				ProfileDir: profile, DshHome: t.TempDir(), NodeBin: fakeNode(t), HarnessRoot: fakeHarnessRoot(t),
				ExpectedPackageName: "fixture",
				Runner: func(ctx context.Context, dir, name string, args ...string) (int, string, error) {
					if args[len(args)-1] != spec { t.Fatalf("source spec lost: %v", args) }
					return 0, "fixture installed", nil
				},
			}
			result := PNPMInstallContext(context.Background(), cfg)(spec, InstallOpts{})
			if !result.OK { t.Fatalf("existing expected package failed: %+v", result) }
		})
	}
}

package installctrl

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/paths"
)

func TestEnsurePnpmShimDirEscapesPercentAndAtomic(t *testing.T) {
	userData := t.TempDir()
	t.Setenv(paths.EnvUserData, userData)

	nodeBin := filepath.Join(userData, "100%node", "node.exe")
	pnpmCJS := filepath.Join(userData, "50%pnpm", "pnpm.cjs")
	dir, err := ensurePnpmShimDir(nodeBin, pnpmCJS)
	if err != nil {
		t.Fatalf("ensurePnpmShimDir: %v", err)
	}
	if runtime.GOOS == "windows" {
		raw, err := os.ReadFile(filepath.Join(dir, "pnpm.cmd"))
		if err != nil {
			t.Fatal(err)
		}
		got := string(raw)
		if !strings.Contains(got, "100%%node") || !strings.Contains(got, "50%%pnpm") {
			t.Fatalf("pnpm.cmd missing escaped %% paths:\n%s", got)
		}
	} else {
		raw, err := os.ReadFile(filepath.Join(dir, "pnpm"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), nodeBin) || !strings.Contains(string(raw), pnpmCJS) {
			t.Fatalf("pnpm shim missing paths:\n%s", string(raw))
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Fatalf("unexpected temp file leftover: %s", entry.Name())
		}
	}
}

func TestMergeAllowBuildsLeavesNoFixedTmp(t *testing.T) {
	profile := t.TempDir()
	file, err := mergeAllowBuildsIntoWorkspace(profile, []string{"esbuild", "sharp"})
	if err != nil {
		t.Fatalf("mergeAllowBuildsIntoWorkspace: %v", err)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"esbuild": true`) || !strings.Contains(string(raw), `"sharp": true`) {
		t.Fatalf("unexpected workspace content:\n%s", string(raw))
	}
	entries, err := os.ReadDir(profile)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Fatalf("unexpected temp file leftover: %s", entry.Name())
		}
	}
}

func TestConcurrentInstallAndRemoveSerializePerProfileDir(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "web-profile")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	var inFlight atomic.Int32
	var maxInFlight atomic.Int32

	recordEnter := func() {
		cur := inFlight.Add(1)
		for {
			prev := maxInFlight.Load()
			if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
		inFlight.Add(-1)
	}

	makeCfg := func(prof string) InstallConfig {
		return InstallConfig{
			ProfileDir:  prof,
			DshHome:     t.TempDir(),
			HarnessRoot: fakeHarnessRoot(t),
			NodeBin:     fakeNode(t),
			Runner: func(_ context.Context, _ string, _ string, args ...string) (int, string, error) {
				recordEnter()
				if len(args) >= 4 && args[len(args)-2] == "add" {
					plantLoadablePackage(t, prof, "widget")
				}
				return 0, "ok\n", nil
			},
		}
	}

	altProfile := profile + string(filepath.Separator) + "."
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		res := PNPMInstallContext(context.Background(), makeCfg(profile))("github:acme/widget", InstallOpts{})
		if !res.OK {
			t.Errorf("install 1 failed: %+v", res)
		}
	}()
	go func() {
		defer wg.Done()
		res := PNPMInstallContext(context.Background(), makeCfg(altProfile))("github:acme/widget", InstallOpts{})
		if !res.OK {
			t.Errorf("install 2 failed: %+v", res)
		}
	}()
	go func() {
		defer wg.Done()
		res := RemovePluginContext(context.Background(), makeCfg(profile), "widget")
		if !res.OK {
			t.Errorf("remove failed: %+v", res)
		}
	}()
	wg.Wait()

	if max := maxInFlight.Load(); max > 1 {
		t.Fatalf("same profile operations overlapped: maxInFlight=%d", max)
	}
}

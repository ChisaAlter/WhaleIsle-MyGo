package components

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Only synthetic files and fake process/command operations are used here.
func TestBridgeManagementFixture(t *testing.T) {
	setup := func(t *testing.T) (string, *BridgeUninstall, *bridgeRemovalOps, *int, *int) {
		t.Helper()
		ud := t.TempDir()
		root := filepath.Join(ud, "components", "whalebridge")
		payload := []byte("synthetic non-executable payload")
		for _, v := range []string{"1.2.3", "1.2.2"} {
			dir := filepath.Join(root, "versions", v)
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			m := bridgeReleaseManifest{ID: "whalebridge", Version: v, License: "MIT License", Platforms: map[string]bridgeSpec{"win32-x64": {Asset: bridgeAsset, SHA256: fmt.Sprintf("%x", sha256.Sum256(payload)), Size: int64(len(payload))}}}
			b, _ := json.Marshal(m)
			if err := os.WriteFile(filepath.Join(dir, bridgeManifest), b, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, bridgeAsset), payload, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := replaceBridgeRecord(root, installedFile{Version: "1.2.3", Previous: "1.2.2"}); err != nil {
			t.Fatal(err)
		}
		for _, dir := range []string{filepath.Join(root, "data"), filepath.Join(ud, "dsh-home")} {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
		}
		for _, path := range []string{filepath.Join(root, "data", "config.json"), filepath.Join(ud, "dsh-home", "chat.json")} {
			if err := os.WriteFile(path, []byte("synthetic-user-data"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		stopped, disconnected := new(int), new(int)
		ops := &bridgeRemovalOps{
			inspect: func(context.Context, string, string, string) (bool, error) { return true, nil },
			stop:    func(context.Context, string, string) (bool, error) { *stopped++; return false, nil },
			start:   func(string, string) error { return errors.New("unexpected service start") },
			disconnect: func(_ context.Context, exe, data, dsh string) error {
				*disconnected++
				if !strings.Contains(exe, ".uninstall-") || !strings.HasSuffix(exe, bridgeAsset) || data != filepath.Join(root, "data") || dsh != filepath.Join(ud, "dsh-home") {
					t.Fatalf("wrong disconnect target: %s %s %s", exe, data, dsh)
				}
				if _, err := os.Stat(filepath.Join(root, "installed.json")); !os.IsNotExist(err) {
					t.Fatal("record active during disconnect")
				}
				return nil
			},
			rename: os.Rename, remove: os.RemoveAll,
		}
		plan, err := prepareBridgeUninstall(context.Background(), ud, *ops)
		if err != nil {
			t.Fatal(err)
		}
		if plan.Version() != "1.2.3" || !plan.DefaultModel() || *stopped != 0 || *disconnected != 0 {
			t.Fatal("incorrect or mutating preview")
		}
		return ud, plan, ops, stopped, disconnected
	}
	preserved := func(t *testing.T, ud string) {
		t.Helper()
		for _, p := range []string{"components/whalebridge/data/config.json", "dsh-home/chat.json"} {
			b, e := os.ReadFile(filepath.Join(ud, p))
			if e != nil || string(b) != "synthetic-user-data" {
				t.Fatalf("data changed: %s %v", p, e)
			}
		}
	}
	t.Run("uninstall-keeps-data", func(t *testing.T) {
		ud, p, ops, stopped, disconnected := setup(t)
		msg, err := p.apply(context.Background(), *ops)
		if err != nil || msg != "" || *stopped != 1 || *disconnected != 1 {
			t.Fatalf("result: %q %v", msg, err)
		}
		if row, _ := whalebridgeRow(p.root); row.InstalledVersion != "" || row.State != "available" {
			t.Fatalf("not uninstalled: %+v", row)
		}
		if _, e := os.Stat(filepath.Join(p.root, "versions")); !os.IsNotExist(e) {
			t.Fatal("payload installed")
		}
		preserved(t, ud)
	})
	t.Run("cancel-before-stop", func(t *testing.T) {
		ud, p, ops, stopped, disconnected := setup(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, e := p.apply(ctx, *ops); e == nil || *stopped != 0 || *disconnected != 0 {
			t.Fatal("cancel ignored")
		}
		preserved(t, ud)
	})
	t.Run("stale-version", func(t *testing.T) {
		_, p, ops, stopped, disconnected := setup(t)
		if e := replaceBridgeRecord(p.root, installedFile{Version: "1.2.2"}); e != nil {
			t.Fatal(e)
		}
		if _, e := p.apply(context.Background(), *ops); e == nil || *stopped != 0 || *disconnected != 0 {
			t.Fatal("stale confirmation applied")
		}
	})
	t.Run("changed-default-model", func(t *testing.T) {
		_, p, ops, stopped, disconnected := setup(t)
		ops.inspect = func(context.Context, string, string, string) (bool, error) { return false, nil }
		if _, e := p.apply(context.Background(), *ops); e == nil || *stopped != 0 || *disconnected != 0 {
			t.Fatal("changed impact accepted")
		}
	})
	t.Run("changed-binary", func(t *testing.T) {
		_, p, ops, stopped, disconnected := setup(t)
		if e := os.WriteFile(filepath.Join(p.root, "versions", p.Version(), bridgeAsset), []byte("changed"), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := p.apply(context.Background(), *ops); e == nil || *stopped != 0 || *disconnected != 0 {
			t.Fatal("changed binary accepted")
		}
	})
	for _, running := range []bool{false, true} {
		t.Run(fmt.Sprintf("disconnect-failure-running-%t", running), func(t *testing.T) {
			ud, p, ops, _, _ := setup(t)
			started := 0
			ops.stop = func(context.Context, string, string) (bool, error) { return running, nil }
			ops.start = func(root, dsh string) error {
				started++
				if root != p.root || dsh != p.dshHome {
					t.Fatal("wrong restore target")
				}
				return nil
			}
			ops.disconnect = func(context.Context, string, string, string) error { return errors.New("synthetic failure") }
			if _, e := p.apply(context.Background(), *ops); e == nil || !strings.Contains(e.Error(), "恢复") {
				t.Fatalf("restore: %v", e)
			}
			if (started == 1) != running {
				t.Fatalf("wrong running state: %d", started)
			}
			if readInstalled(filepath.Join(p.root, "installed.json")) != p.from {
				t.Fatal("record lost")
			}
			if _, e := verifyBridgeVersion(p.root, p.Version()); e != nil {
				t.Fatal(e)
			}
			preserved(t, ud)
		})
	}
	t.Run("cleanup-warning", func(t *testing.T) {
		ud, p, ops, _, _ := setup(t)
		ops.remove = func(string) error { return errors.New("synthetic locked file") }
		msg, e := p.apply(context.Background(), *ops)
		if e != nil || !strings.Contains(msg, "清理") {
			t.Fatalf("cleanup: %q %v", msg, e)
		}
		if readInstalled(filepath.Join(p.root, "installed.json")).Version != "" {
			t.Fatal("false installed state")
		}
		preserved(t, ud)
	})
	t.Run("stop-refused", func(t *testing.T) {
		ud, p, ops, _, disconnected := setup(t)
		ops.stop = func(context.Context, string, string) (bool, error) { return false, errors.New("busy tasks") }
		if _, e := p.apply(context.Background(), *ops); e == nil || *disconnected != 0 {
			t.Fatal("stop refusal ignored")
		}
		if readInstalled(filepath.Join(p.root, "installed.json")) != p.from {
			t.Fatal("record lost")
		}
		preserved(t, ud)
	})
	t.Run("existing-rollback-keeps-stopped-state", func(t *testing.T) {
		ud, _, _, _, _ := setup(t)
		rollback, err := PrepareWhaleBridgeRollback(ud)
		if err != nil || rollback.Version() != "1.2.2" {
			t.Fatalf("rollback preview: %v", err)
		}
		if err := rollback.apply(context.Background(), bridgeLifecycle{}); err != nil {
			t.Fatal(err)
		}
		if got := readInstalled(filepath.Join(rollback.root, "installed.json")); got.Version != "1.2.2" || got.Previous != "1.2.3" {
			t.Fatalf("wrong rollback: %+v", got)
		}
		preserved(t, ud)
	})
	t.Run("command-output-bounded", func(t *testing.T) {
		var out bridgeCommandOutput
		payload := make([]byte, 128<<10)
		if n, err := out.Write(payload); err != nil || n != len(payload) || !out.overflow || len(out.bytes) != 64<<10 {
			t.Fatal("output bound failed")
		}
		if n, err := out.Write(payload); err != nil || n != len(payload) || len(out.bytes) != 64<<10 {
			t.Fatal("output grew past bound")
		}
	})
	t.Run("replacement-instance-blocks-disconnect", func(t *testing.T) {
		ud, p, ops, _, disconnected := setup(t)
		ops.stop = func(context.Context, string, string) (bool, error) {
			return false, os.WriteFile(filepath.Join(p.root, "data", "state.json"), []byte(`{"pid":9999}`), 0600)
		}
		if _, err := p.apply(context.Background(), *ops); err == nil || *disconnected != 0 {
			t.Fatal("replacement instance was disconnected")
		}
		preserved(t, ud)
	})
	t.Run("record-move-failure-restores-payload", func(t *testing.T) {
		ud, p, ops, _, disconnected := setup(t)
		ops.rename = func(from, to string) error {
			if from == filepath.Join(p.root, "installed.json") {
				return errors.New("fixture record locked")
			}
			return os.Rename(from, to)
		}
		if _, err := p.apply(context.Background(), *ops); err == nil || *disconnected != 0 {
			t.Fatal("record failure ignored")
		}
		if readInstalled(filepath.Join(p.root, "installed.json")) != p.from {
			t.Fatal("record lost")
		}
		if _, err := verifyBridgeVersion(p.root, p.Version()); err != nil {
			t.Fatal(err)
		}
		preserved(t, ud)
	})
	t.Run("symlink-versions", func(t *testing.T) {
		_, p, ops, stopped, _ := setup(t)
		vers := filepath.Join(p.root, "versions")
		if e := os.Rename(vers, vers+"-real"); e != nil {
			t.Fatal(e)
		}
		if e := os.Symlink(vers+"-real", vers); e != nil {
			t.Skip(e)
		}
		if _, e := p.apply(context.Background(), *ops); e == nil || *stopped != 0 {
			t.Fatal("symlink followed")
		}
	})
}

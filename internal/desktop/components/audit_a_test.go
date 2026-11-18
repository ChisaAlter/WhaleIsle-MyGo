package components

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeGenericRegistry plants one registry record per entry under root.
func writeGenericRegistry(t *testing.T, root string, records map[string]any) {
	t.Helper()
	dir := filepath.Join(root, "components")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"components": records})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "registry.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

// blockGenericRegistryWrites replaces the registry file with a non-empty
// directory so every save fails, the same mechanism
// TestWriteJSONAtomicPreservesExistingFileOnRenameFailure uses.
func blockGenericRegistryWrites(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "components", "registry.json")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func loadedGenericService(t *testing.T, root string) *genericService {
	t.Helper()
	s := newGenericService(root, GenericOptions{}, defaultGenericRuntimeDeps())
	s.mu.Lock()
	s.ensureLoadedLocked()
	s.mu.Unlock()
	return s
}

func TestGenericUninstallReportsRecordSaveFailure(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "components", "notes", "versions", "1.0.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeGenericRegistry(t, root, map[string]any{"notes": map[string]any{"version": "1.0.0", "state": "stopped"}})
	s := loadedGenericService(t, root)
	blockGenericRegistryWrites(t, root)

	result := s.uninstall("notes")
	if result.OK {
		t.Fatalf("uninstall reported success after the record write failed: %#v", result)
	}
	if result.Error != "record-save-failed" {
		t.Fatalf("error = %q, want record-save-failed", result.Error)
	}
	if result.Message == "" || result.Message == "record-save-failed" {
		t.Fatalf("message is not localized: %q", result.Message)
	}
}

func TestGenericStopReportsRecordSaveFailure(t *testing.T) {
	root := t.TempDir()
	writeGenericRegistry(t, root, map[string]any{"notes": map[string]any{"version": "1.0.0", "state": "running"}})
	s := loadedGenericService(t, root)
	blockGenericRegistryWrites(t, root)

	result := s.stop("notes")
	if result.OK {
		t.Fatalf("stop reported success after the record write failed: %#v", result)
	}
	if result.Error != "record-save-failed" || result.State != "stopped" {
		t.Fatalf("result = %#v, want record-save-failed with state stopped", result)
	}
	if result.Message == "" || result.Message == "record-save-failed" {
		t.Fatalf("message is not localized: %q", result.Message)
	}
}

func TestGenericRegistryQuarantineFailureRefusesWrites(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "components")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := filepath.Join(dir, "registry.json")
	original := []byte("{not a registry")
	if err := os.WriteFile(registry, original, 0o600); err != nil {
		t.Fatal(err)
	}
	// The quarantine destination is a non-empty directory, so the rename cannot
	// succeed and the corrupt file must stay where it is.
	if err := os.MkdirAll(filepath.Join(registry+".broken", "child"), 0o700); err != nil {
		t.Fatal(err)
	}

	reg := loadGenericRegistry(registry)
	if reg.LoadFailure == "" {
		t.Fatal("expected LoadFailure to be set when the corrupt registry could not be set aside")
	}
	s := newGenericService(root, GenericOptions{}, defaultGenericRuntimeDeps())
	s.mu.Lock()
	s.reg = reg
	s.loaded = true
	err := s.saveRegistryLocked()
	s.mu.Unlock()
	if err == nil {
		t.Fatal("expected saves to be refused for an unreadable registry")
	}
	onDisk, readErr := os.ReadFile(registry)
	if readErr != nil || string(onDisk) != string(original) {
		t.Fatalf("corrupt registry was modified: %q err=%v", onDisk, readErr)
	}
}

func TestWhaleBridgeUninstallKeepsPayloadWhenRollbackRenameFails(t *testing.T) {
	root := t.TempDir()
	payload := filepath.Join(root, "versions", "1.2.3", "WhaleBridge-win32-x64.exe")
	if err := os.MkdirAll(filepath.Dir(payload), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(payload, []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "installed.json"), []byte(`{"version":"1.2.3","previous":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewWhaleBridgeService(ServiceOptions{
		Root:    root,
		DshHome: filepath.Join(t.TempDir(), "dsh"),
		Disconnect: func(context.Context, string, string) error {
			// Occupy the restore destination with a non-empty directory so the
			// rollback rename has to fail.
			if err := os.MkdirAll(filepath.Join(root, "versions", "blocker"), 0o700); err != nil {
				return err
			}
			return errors.New("disconnect failed")
		},
	})

	_, err := svc.Uninstall(context.Background(), false, nil)
	if err == nil {
		t.Fatal("expected the uninstall to fail")
	}
	staged, globErr := filepath.Glob(filepath.Join(root, ".uninstall-*"))
	if globErr != nil || len(staged) != 1 {
		t.Fatalf("staged dirs = %v err=%v", staged, globErr)
	}
	kept := filepath.Join(staged[0], "versions", "1.2.3", "WhaleBridge-win32-x64.exe")
	if _, statErr := os.Stat(kept); statErr != nil {
		t.Fatalf("payload was deleted although the rollback rename failed: %v", statErr)
	}
	if !strings.Contains(err.Error(), staged[0]) {
		t.Fatalf("error does not name the retained path: %v", err)
	}
}

func TestWhaleBridgeSwitchRollbackFailureDoesNotClaimRestore(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "versions", "2.0.0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "installed.json"), []byte(`{"version":"1.0.0","previous":""}`), 0o600); err != nil {
		t.Fatal(err)
	}
	starts := 0
	svc := NewWhaleBridgeService(ServiceOptions{
		Root:    root,
		DshHome: filepath.Join(t.TempDir(), "dsh"),
		StartRuntime: func(context.Context, string, string) error {
			starts++
			if starts == 1 {
				return errors.New("start failed")
			}
			return nil
		},
		StopRuntime: func(context.Context, string) error {
			// Replace installed.json with a non-empty directory so the rollback
			// record write fails.
			path := filepath.Join(root, "installed.json")
			if err := os.Remove(path); err != nil {
				return err
			}
			return os.MkdirAll(filepath.Join(path, "child"), 0o700)
		},
	})

	_, err := svc.switchVersion(context.Background(), "1.0.0", "2.0.0", "", nil)
	if err == nil {
		t.Fatal("expected the switch to fail")
	}
	if strings.Contains(err.Error(), "restored previous version") {
		t.Fatalf("error still claims the previous version was restored: %v", err)
	}
	if !strings.Contains(err.Error(), "1.0.0") {
		t.Fatalf("error does not name the version whose record was not restored: %v", err)
	}
}

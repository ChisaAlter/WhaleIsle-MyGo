package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestReconcileHostVersionFollowsTheInstalledRuntime is the stale field the
// operator's config carried (0.2.0-rc.2 beside a 0.2.1-rc.4 runtime). Nobody
// should have to know the field exists: a launch that ever falls back to npm
// must ask for the Host this app ships.
func TestReconcileHostVersionFollowsTheInstalledRuntime(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(c *Config) { c.DshVersion = "0.2.0-rc.2" }); err != nil {
		t.Fatal(err)
	}
	if err := store.ReconcileHostVersion("0.2.1-rc.4"); err != nil {
		t.Fatal(err)
	}
	if got := store.Get().DshVersion; got != "0.2.1-rc.4" {
		t.Fatalf("stored version = %q, want the installed runtime", got)
	}
	// The correction reaches the file, so the next launch starts from it.
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Get().DshVersion; got != "0.2.1-rc.4" {
		t.Fatalf("persisted version = %q", got)
	}
}

// TestReconcileHostVersionLeavesSettledValuesAlone keeps a start from writing the
// config on every launch, and keeps an unknown runtime from inventing a value.
func TestReconcileHostVersionLeavesSettledValuesAlone(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(c *Config) { c.DshVersion = " 0.2.1-rc.4 " }); err != nil {
		t.Fatal(err)
	}
	before := configModified(t, dir)
	if err := store.ReconcileHostVersion("0.2.1-rc.4"); err != nil {
		t.Fatal(err)
	}
	if after := configModified(t, dir); !after.Equal(before) {
		t.Fatal("a matching version rewrote the config")
	}
	if err := store.ReconcileHostVersion(""); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(store.Get().DshVersion); got != "0.2.1-rc.4" {
		t.Fatalf("an unreadable runtime invented a version: %q", got)
	}
}

func configModified(t *testing.T, dir string) time.Time {
	t.Helper()
	info, err := os.Stat(filepath.Join(dir, FileName))
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime()
}

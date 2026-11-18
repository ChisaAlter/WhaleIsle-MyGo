package main

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/config"
)

// breakConfigWrites replaces the config file with a non-empty directory so the
// atomic write cannot land.
func breakConfigWrites(t *testing.T, st *config.Store) {
	t.Helper()
	path := st.Path()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestDropDisabledPluginReportsWriteFailure(t *testing.T) {
	st, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	breakConfigWrites(t, st)
	if err := dropDisabledPlugin(st, "dsh-im"); err == nil {
		t.Fatal("config write failure was not returned")
	}
}

func TestDropDisabledPluginPersistsList(t *testing.T) {
	st, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Update(func(c *config.Config) { c.DisabledPlugins = []string{"dsh-im", "other"} }); err != nil {
		t.Fatal(err)
	}
	if err := dropDisabledPlugin(st, "dsh-im"); err != nil {
		t.Fatal(err)
	}
	got := st.Get().DisabledPlugins
	if len(got) != 1 || got[0] != "other" {
		t.Fatalf("disabled plugins = %v, want [other]", got)
	}
}

func TestMigrateViewTabsSettingReportsWriteFailure(t *testing.T) {
	st, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	breakConfigWrites(t, st)
	if err := migrateViewTabsSetting(st, true); err == nil {
		t.Fatal("config write failure was not returned")
	}
}

func TestMigrateViewTabsSettingPersistsValue(t *testing.T) {
	st, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateViewTabsSetting(st, true); err != nil {
		t.Fatal(err)
	}
	if !st.Get().HideChatViewTabs {
		t.Fatal("HideChatViewTabs was not persisted")
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"

	"whaleisle/internal/config"
	"whaleisle/internal/paths"
	"whaleisle/internal/product"
)

// TestEnsureLaunchWorkspaceAdoptsTheDefault covers a first launch: no saved
// folder becomes the default workspace under the managed Documents root, and
// the setting is persisted (config.js defaultWorkspace + app-ready.js
// saveConfig).
func TestEnsureLaunchWorkspaceAdoptsTheDefault(t *testing.T) {
	userData := t.TempDir()
	t.Setenv(paths.EnvUserData, userData)
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store}
	want := filepath.Join(userData, "Documents", product.EnglishName)
	got := s.ensureLaunchWorkspace(config.Config{})
	if got.Workspace != want {
		t.Fatalf("workspace = %q, want %q", got.Workspace, want)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("default folder was not created: %v", err)
	}
	if saved := store.Get().Workspace; saved != want {
		t.Fatalf("saved workspace = %q, want %q", saved, want)
	}
}

// TestEnsureLaunchWorkspaceRecreatesAVanishedFolder keeps a saved folder that
// only vanished: the reference recreates it rather than moving the setting.
func TestEnsureLaunchWorkspaceRecreatesAVanishedFolder(t *testing.T) {
	userData := t.TempDir()
	t.Setenv(paths.EnvUserData, userData)
	saved := filepath.Join(t.TempDir(), "vanished")
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store}
	got := s.ensureLaunchWorkspace(config.Config{Workspace: saved})
	if got.Workspace != saved {
		t.Fatalf("workspace = %q, want the recreated %q", got.Workspace, saved)
	}
	if info, err := os.Stat(saved); err != nil || !info.IsDir() {
		t.Fatalf("saved folder was not recreated: %v", err)
	}
	if cfg := store.Get().Workspace; cfg != "" && cfg != saved {
		t.Fatalf("store rewrote a recreated folder: %q", cfg)
	}
}

// TestEnsureLaunchWorkspaceFallsBackToTheDefault covers a saved folder the
// file system refuses to recreate: only then does the default replace it.
func TestEnsureLaunchWorkspaceFallsBackToTheDefault(t *testing.T) {
	userData := t.TempDir()
	t.Setenv(paths.EnvUserData, userData)
	blocker := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(blocker, "workspace")
	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store}
	want := filepath.Join(userData, "Documents", product.EnglishName)
	got := s.ensureLaunchWorkspace(config.Config{Workspace: saved})
	if got.Workspace != want {
		t.Fatalf("workspace = %q, want fallback %q", got.Workspace, want)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Fatalf("fallback folder was not created: %v", err)
	}
}

// TestEnsureLaunchWorkspaceKeepsAnExistingFolder leaves a usable saved folder
// untouched.
func TestEnsureLaunchWorkspaceKeepsAnExistingFolder(t *testing.T) {
	userData := t.TempDir()
	t.Setenv(paths.EnvUserData, userData)
	saved := t.TempDir()
	s := &mainState{}
	got := s.ensureLaunchWorkspace(config.Config{Workspace: saved})
	if got.Workspace != saved {
		t.Fatalf("workspace = %q, want %q", got.Workspace, saved)
	}
}

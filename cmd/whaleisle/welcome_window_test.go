package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWelcomeKeyShape(t *testing.T) {
	for _, good := range []string{"sk-abc123", "sk-!weird~printable", "\"sk-unbalanced'"} {
		if !welcomeKeyShape(good) {
			t.Fatalf("rejected printable key %q", good)
		}
	}
	// The Electron renderer refuses a value still wrapped in quotes or
	// backticks and any KEY=value environment assignment
	// (reference/electron/src/renderer/welcome/welcome.js:40).
	for _, bad := range []string{
		"", "has space", "KEY=\"quoted\"", "DEEPSEEK_API_KEY=sk-abc", "sk-中文", "line\nbreak",
		"\"sk-abc123\"", "'sk-abc123'", "\x60sk-abc123\x60",
	} {
		if welcomeKeyShape(bad) {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestWelcomeRequiredGate(t *testing.T) {
	s := &mainState{credsMissing: true}
	if !s.welcomeRequired() {
		t.Fatal("a profile without a stored API key must see the welcome window")
	}
	s.welcomeDone = true
	if s.welcomeRequired() {
		t.Fatal("a completed or skipped welcome must not gate again")
	}
	s.welcomeDone = false
	s.credsMissing = false
	if s.welcomeRequired() {
		t.Fatal("a stored API key must skip the welcome window")
	}
}

func TestWriteWelcomeMarkerIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	writeWelcomeMarker(dir)
	writeWelcomeMarker(dir)
	path := filepath.Join(dir, welcomeMarker)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("marker must be a file")
	}
}

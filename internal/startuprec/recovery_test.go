package startuprec

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestColdGateFailureAndRecoveryHold(t *testing.T) {
	for _, tt := range []struct {
		name                        string
		failed, skip, imports, auto bool
		want                        bool
	}{
		{"healthy", false, false, true, true, true}, {"failed", true, false, true, true, false},
		{"recovery", false, true, true, true, false}, {"import blocked", false, false, false, true, false},
		{"manual preference", false, false, true, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldAutoStart("/synthetic", tt.auto, tt.imports, tt.failed, tt.skip); got != tt.want {
				t.Fatalf("got %v", got)
			}
		})
	}
	if ShouldAutoStart(" ", true, true, false, false) {
		t.Fatal("empty workspace auto-started")
	}
	if ShouldCloseLauncher(true, true, true, false) || ShouldCloseLauncher(true, true, false, true) || ShouldCloseLauncher(true, false, false, false) {
		t.Fatal("unsafe launcher close")
	}
	if !ShouldCloseLauncher(true, true, false, false) {
		t.Fatal("healthy full start did not close")
	}
}

func TestUpdatePromptWaitsForVisibleIdleLauncher(t *testing.T) {
	if CanPromptUpdate(true, true, false, false, false, false, false) {
		t.Fatal("hidden window prompted")
	}
	if CanPromptUpdate(true, true, true, false, false, false, false) {
		t.Fatal("desktop stole focus")
	}
	if CanPromptUpdate(true, true, true, true, true, false, false) {
		t.Fatal("starting prompted")
	}
	if CanPromptUpdate(false, true, true, true, false, false, false) {
		t.Fatal("disabled setting prompted")
	}
	if CanPromptUpdate(true, true, true, true, false, true, false) || CanPromptUpdate(true, true, true, true, false, false, true) {
		t.Fatal("busy/quitting prompted")
	}
	if !CanPromptUpdate(true, true, true, true, false, false, false) {
		t.Fatal("visible idle launcher not prompted")
	}
}

func TestOutcomePersistsFailureThenFullRecovery(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Last().Failed() {
		t.Fatal("missing state is failure")
	}
	if err = s.Record(false, "synthetic crash", []string{"failed to apply loader entry fixture (@fixture/plugin)"}, "v1"); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.Last().Failed() || reopened.Last().AppVersion != "v1" {
		t.Fatal("failure not durable")
	}
	snap := reopened.Last()
	snap.LogTail[0] = "mutated"
	if reopened.Last().LogTail[0] == "mutated" {
		t.Fatal("aliased snapshot")
	}
	if err = reopened.Record(true, "", nil, "v1"); err != nil {
		t.Fatal(err)
	}
	again, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again.Last().Failed() || len(again.Last().LogTail) != 0 {
		t.Fatal("healthy recovery retained failure")
	}
}

func TestOutcomeConflictAndMalformedStateFailClosed(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	p := filepath.Join(dir, FileName)
	if err := os.WriteFile(p, []byte(`{"ok":false,"error":"external"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(true, "", nil, "v1"); !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict got %v", err)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "external") {
		t.Fatal("overwrote external state")
	}
	if err := os.WriteFile(p, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("corrupt state accepted")
	}
}

func TestOutcomeBoundsAndRedactsSyntheticEvidence(t *testing.T) {
	dir := t.TempDir()
	s, _ := Open(dir)
	synthetic := `Authorization: Bearer syntheticSecret token=syntheticToken password="synthetic password" api_key=syntheticKey https://person:syntheticPass@example.invalid/a?token=syntheticQuery&other=1 Cookie: session=syntheticCookie`
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = synthetic + strings.Repeat("x", 1000)
	}
	if err := s.Record(false, synthetic, lines, "v1"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, FileName))
	for _, secret := range []string{"syntheticSecret", "syntheticToken", "synthetic password", "syntheticKey", "syntheticPass", "syntheticQuery", "syntheticCookie"} {
		if strings.Contains(string(b), secret) {
			t.Fatalf("leaked %s", secret)
		}
	}
	if len(s.Last().LogTail) != 80 {
		t.Fatal("tail unbounded")
	}
	for _, line := range s.Last().LogTail {
		if len([]rune(line)) > 240 {
			t.Fatal("line unbounded")
		}
	}
}

func TestRedactIndependentCredentialForms(t *testing.T) {
	for _, line := range []string{
		`token=syntheticToken`, `password="synthetic password"`, `api_key=syntheticKey`,
		`https://person:syntheticPass@example.invalid/a?token=syntheticQuery&other=1`,
		`Cookie: session=syntheticCookie`, `Authorization: Bearer syntheticSecret`,
	} {
		if got := RedactLine(line); strings.Contains(got, "synthetic") {
			t.Fatalf("secret survived: %s", got)
		}
	}
}

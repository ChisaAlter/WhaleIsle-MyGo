package assistant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	got := Dir(`C:\data`)
	want := filepath.Join(`C:\data`, "assistant")
	if got != want {
		t.Fatalf("Dir = %q, want %q", got, want)
	}
}

func TestOpenPersistModeAndHistory(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Snapshot().Mode != ModeWhale {
		t.Fatalf("default mode = %q", s.Snapshot().Mode)
	}
	s.SetMode(ModeDshbot)
	s.AppendTurn(Turn{
		Mode:      ModeDshbot,
		Prompt:    "hello bot",
		Response:  "hi",
		SessionID: "sess-dsh",
		Source:    "session",
	})
	snap := s.Snapshot()
	if snap.Mode != ModeDshbot {
		t.Fatalf("mode = %q", snap.Mode)
	}
	if len(snap.History) != 1 || snap.History[0].Prompt != "hello bot" {
		t.Fatalf("history = %+v", snap.History)
	}
	if snap.SessionID != "sess-dsh" {
		t.Fatalf("sessionID = %q", snap.SessionID)
	}

	raw, err := os.ReadFile(filepath.Join(Dir(dir), historyFile))
	if err != nil {
		t.Fatal(err)
	}
	var turns []Turn
	if err := json.Unmarshal(raw, &turns); err != nil {
		t.Fatal(err)
	}
	if len(turns) != 1 {
		t.Fatalf("history.json len = %d", len(turns))
	}

	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	snap2 := s2.Snapshot()
	if snap2.Mode != ModeDshbot || len(snap2.History) != 1 {
		t.Fatalf("reopen snap = %+v", snap2)
	}
}

func TestMaxHistoryTrim(t *testing.T) {
	s := New()
	for i := 0; i < MaxHistory+5; i++ {
		s.AppendTurn(Turn{Prompt: "p", Response: "r", Mode: ModeWhale})
	}
	if n := len(s.Snapshot().History); n != MaxHistory {
		t.Fatalf("history len = %d, want %d", n, MaxHistory)
	}
}

func TestPrefixForMode(t *testing.T) {
	if PrefixForMode(ModeWhale) != "[whale] " {
		t.Fatalf("whale prefix = %q", PrefixForMode(ModeWhale))
	}
	if PrefixForMode(ModeDshbot) != "[dshbot] " {
		t.Fatalf("dshbot prefix = %q", PrefixForMode(ModeDshbot))
	}
}

func TestHistoryFromSessionMessages(t *testing.T) {
	msgs := []struct{ Role, Body string }{
		{"user", "q1"},
		{"assistant", "a1"},
		{"user", "q2"},
		{"assistant", "a2"},
		{"assistant", "orphan"},
	}
	got := HistoryFromSessionMessages(msgs, 10)
	if len(got) != 2 {
		t.Fatalf("pairs = %d, want 2; %#v", len(got), got)
	}
	if got[0].Prompt != "q2" || got[0].Response != "a2" {
		t.Fatalf("newest first = %+v", got[0])
	}
	if got[1].Prompt != "q1" {
		t.Fatalf("older = %+v", got[1])
	}
}

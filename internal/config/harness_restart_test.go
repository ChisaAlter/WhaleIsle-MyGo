package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSP06RestartPreferenceValidation(t *testing.T) {
	for _, tc := range []struct {
		enabled         bool
		attempts, delay string
		want            RestartPreferences
	}{
		{true, "1", "500", RestartPreferences{true, 1, 500}},
		{false, "10", "30000", RestartPreferences{false, 10, 30000}},
		{true, " 3 ", " 1000 ", RestartPreferences{true, 3, 1000}},
	} {
		got, err := ParseRestartPreferences(tc.enabled, tc.attempts, tc.delay)
		if err != nil || got != tc.want {
			t.Fatalf("parse %+v: got %+v, %v", tc, got, err)
		}
	}
	for _, tc := range []struct{ attempts, delay string }{
		{"0", "1000"}, {"11", "1000"}, {"", "1000"}, {"1.5", "1000"}, {"999999999999999999999999", "1000"},
		{"3", "499"}, {"3", "30001"}, {"3", ""}, {"3", "5e2"}, {"3", "-500"},
	} {
		if _, err := ParseRestartPreferences(false, tc.attempts, tc.delay); err == nil {
			t.Fatalf("accepted invalid disabled draft: %+v", tc)
		}
	}
}

func TestSP06RestartPreferencesCommitPreservesUnknown(t *testing.T) {
	dir := t.TempDir()
	original := []byte(`{"harnessAutoRestart":true,"harnessRestartMaxAttempts":3,"harnessRestartBaseDelayMs":1000,"workspace":"/synthetic/project","remoteEnabled":false,"futureSetting":{"nested":[1,true,"keep"]}}`)
	if err := os.WriteFile(filepath.Join(dir, FileName), original, 0600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	next := RestartPreferences{false, 10, 30000}
	if err = store.Update(next.Apply); err != nil {
		t.Fatal(err)
	}
	if got := RestartPreferencesFrom(store.Get()); got != next {
		t.Fatalf("commit receipt %+v", got)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := RestartPreferencesFrom(reopened.Get()); got != next {
		t.Fatalf("reopen %+v", got)
	}
	if reopened.Get().RemoteEnabled || reopened.Get().Workspace != "/synthetic/project" {
		t.Fatal("unrelated config changed")
	}
	raw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err = json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	var future any
	json.Unmarshal(body["futureSetting"], &future)
	encoded, _ := json.Marshal(future)
	if string(encoded) != `{"nested":[1,true,"keep"]}` {
		t.Fatalf("unknown value changed: %s", encoded)
	}
}

func TestSP06RestartPreferencesAtomicFailureKeepsCommittedPolicy(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := RestartPreferencesFrom(store.Get())
	if err = store.Update(before.Apply); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(store.Path())
	draft := RestartPreferences{false, 8, 2500}
	denied := errors.New("synthetic rename failure")
	store.rename = func(string, string) error { return denied }
	if err = store.Update(draft.Apply); !errors.Is(err, denied) {
		t.Fatalf("save result: %v", err)
	}
	if got := RestartPreferencesFrom(store.Get()); got != before {
		t.Fatalf("failed save published %+v", got)
	}
	after, _ := os.ReadFile(store.Path())
	if !bytes.Equal(original, after) {
		t.Fatal("failed atomic save changed bytes")
	}
	if draft != (RestartPreferences{false, 8, 2500}) {
		t.Fatal("failed save destroyed draft")
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".config-*.json"))
	if len(leftovers) != 0 {
		t.Fatalf("temporary files leaked: %v", leftovers)
	}
}

func TestSP06RestartPreferencesExternalConflictKeepsPolicy(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	before := RestartPreferencesFrom(store.Get())
	external := []byte(`{"futureSetting":"external edit"}`)
	if err = os.WriteFile(store.Path(), external, 0600); err != nil {
		t.Fatal(err)
	}
	if err = store.Update((RestartPreferences{false, 2, 500}).Apply); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if RestartPreferencesFrom(store.Get()) != before {
		t.Fatal("conflict changed committed policy")
	}
	got, _ := os.ReadFile(store.Path())
	if !bytes.Equal(got, external) {
		t.Fatal("external bytes overwritten")
	}
}

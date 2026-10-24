package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func readJournalLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	var lines []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if raw == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(raw), &record); err != nil {
			t.Fatalf("journal line %q: %v", raw, err)
		}
		lines = append(lines, record)
	}
	return lines
}

// TestNewJournalWritesTheStartedMilestone pins the first line: a fresh journal
// always records that this process began, with the schema and identity fields
// the cross-restart correlation needs.
func TestNewJournalWritesTheStartedMilestone(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "update-journal")
	j, err := NewJournal(dir, "1.2.3")
	if err != nil {
		t.Fatalf("NewJournal: %v", err)
	}
	if !filepath.IsAbs(j.Path()) || filepath.Dir(j.Path()) != dir {
		t.Fatalf("journal path = %q", j.Path())
	}
	info, err := os.Stat(j.Path())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Windows carries no POSIX mode bits; the documented 0600 owner-only mode is
	// only observable on the platforms that implement it.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}
	lines := readJournalLines(t, j.Path())
	if len(lines) != 1 {
		t.Fatalf("lines = %d", len(lines))
	}
	first := lines[0]
	if first["schemaVersion"] != float64(JournalSchemaVersion) || first["event"] != JournalActionStarted || first["version"] != "1.2.3" {
		t.Fatalf("first line = %v", first)
	}
	if first["sequence"] != float64(0) || first["pid"] == nil || first["time"] == "" {
		t.Fatalf("identity fields = %v", first)
	}
}

// TestJournalRequiresAnAbsoluteAndKnownAction keeps the evidence off a relative
// working directory and out of free-text territory.
func TestJournalRequiresAnAbsoluteAndKnownAction(t *testing.T) {
	if _, err := NewJournal(filepath.Join("relative", "dir"), "dev"); err == nil {
		t.Fatal("relative directory accepted")
	}
	j, err := NewJournal(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("NewJournal: %v", err)
	}
	if err := j.Action("free text"); err == nil {
		t.Fatal("unknown action accepted")
	}
	if err := j.Action(JournalActionCheckRequested); err != nil {
		t.Fatalf("known action: %v", err)
	}
	lines := readJournalLines(t, j.Path())
	if len(lines) != 2 || lines[1]["event"] != JournalActionCheckRequested || lines[1]["sequence"] != float64(1) {
		t.Fatalf("lines = %v", lines)
	}
}

// TestJournalStateWhitelistsFields records integer download progress only while
// downloading and the error class only while failing.
func TestJournalStateWhitelistsFields(t *testing.T) {
	j, err := NewJournal(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("NewJournal: %v", err)
	}
	percent := 42.9
	if err := j.State(JournalState{Phase: JournalPhaseAvailable, Version: "2.0.0", Message: "https://example.com/secret"}); err != nil {
		t.Fatalf("state: %v", err)
	}
	if err := j.State(JournalState{Phase: JournalPhaseDownloading, Version: "2.0.0", Percent: &percent}); err != nil {
		t.Fatalf("state: %v", err)
	}
	if err := j.State(JournalState{Phase: JournalPhaseError, FailedOperation: "download", Message: "request failed: ETIMEDOUT after 45s"}); err != nil {
		t.Fatalf("state: %v", err)
	}
	if err := j.State(JournalState{Phase: JournalPhaseError, FailedOperation: "install", Message: "signature rejected"}); err != nil {
		t.Fatalf("state: %v", err)
	}
	lines := readJournalLines(t, j.Path())
	if len(lines) != 5 {
		t.Fatalf("lines = %d", len(lines))
	}
	available := lines[1]
	if available["phase"] != JournalPhaseAvailable || available["targetVersion"] != "2.0.0" {
		t.Fatalf("available = %v", available)
	}
	if _, ok := available["message"]; ok {
		t.Fatalf("free text persisted: %v", available)
	}
	downloading := lines[2]
	if downloading["percent"] != float64(42) {
		t.Fatalf("percent = %v", downloading["percent"])
	}
	if _, ok := downloading["errorCode"]; ok {
		t.Fatalf("error fields outside error phase: %v", downloading)
	}
	classified := lines[3]
	if classified["errorCode"] != "ETIMEDOUT" || classified["failedOperation"] != "download" {
		t.Fatalf("error line = %v", classified)
	}
	if unclassified := lines[4]; unclassified["errorCode"] != JournalErrorUnclassified {
		t.Fatalf("unclassified line = %v", unclassified)
	}
}

// TestJournalSkipsRepeatedStates keeps a repeated identical state to one line,
// so a download reporting the same integer percent does not grow the file.
func TestJournalSkipsRepeatedStates(t *testing.T) {
	j, err := NewJournal(t.TempDir(), "dev")
	if err != nil {
		t.Fatalf("NewJournal: %v", err)
	}
	percent := 10.0
	state := JournalState{Phase: JournalPhaseDownloading, Version: "2.0.0", Percent: &percent}
	for i := 0; i < 3; i++ {
		if err := j.State(state); err != nil {
			t.Fatalf("state: %v", err)
		}
	}
	next := 11.0
	if err := j.State(JournalState{Phase: JournalPhaseDownloading, Version: "2.0.0", Percent: &next}); err != nil {
		t.Fatalf("state: %v", err)
	}
	lines := readJournalLines(t, j.Path())
	if len(lines) != 3 {
		t.Fatalf("lines = %d (%v)", len(lines), lines)
	}
	if lines[1]["percent"] != float64(10) || lines[2]["percent"] != float64(11) {
		t.Fatalf("percent lines = %v", lines[1:])
	}
}

// TestJournalToleratesNil keeps the degraded path (journal creation failed) a
// no-op instead of a crash inside the update flow.
func TestJournalToleratesNil(t *testing.T) {
	var j *Journal
	if err := j.Action(JournalActionStarted); err == nil {
		t.Fatal("nil journal accepted an action")
	}
	if err := j.State(JournalState{Phase: JournalPhaseChecking}); err == nil {
		t.Fatal("nil journal accepted a state")
	}
	if j.Path() != "" {
		t.Fatalf("nil path = %q", j.Path())
	}
}

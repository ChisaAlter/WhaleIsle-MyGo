package importer

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestSessionEncodingClassifiesWithoutCopy pins what a row knows about a
// session it cannot import: a category, and no sentence. The scanner's own
// words travel in the error, which the caller puts in the log.
func TestSessionEncodingClassifiesWithoutCopy(t *testing.T) {
	plain := t.TempDir()
	if err := os.WriteFile(filepath.Join(plain, "session.v3.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if blocker, err := sessionEncoding(plain); !blocker.Importable() || err != nil {
		t.Fatalf("a plaintext session is blocked: %v %v", blocker, err)
	}

	mixed := t.TempDir()
	for _, name := range []string{"session.v3.jsonl", "session.v3.jsonl.zstd"} {
		if err := os.WriteFile(filepath.Join(mixed, name), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if blocker, err := sessionEncoding(mixed); blocker != BlockedMixedEncoding || err != nil {
		t.Fatalf("mixed logs classified as %v %v", blocker, err)
	}

	legacy := t.TempDir()
	if err := os.WriteFile(filepath.Join(legacy, "session.db"), []byte("inert database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if blocker, err := sessionEncoding(legacy); blocker != BlockedLegacyDatabase || err != nil {
		t.Fatalf("legacy database classified as %v %v", blocker, err)
	}
}

// TestUnreadableSessionKeepsItsWordsInTheError covers the failure the review
// found on screen: the directory cannot be read, and the OS message used to be
// the row's reason. It is now the category's error, for the log.
func TestUnreadableSessionKeepsItsWordsInTheError(t *testing.T) {
	gone := filepath.Join(t.TempDir(), "gone")
	blocker, err := sessionEncoding(gone)
	if blocker != BlockedUnreadable {
		t.Fatalf("missing directory classified as %v", blocker)
	}
	if err == nil || !strings.Contains(err.Error(), "gone") {
		t.Fatalf("the OS message did not survive for the log: %v", err)
	}
	if strings.Contains(blocker.String(), "gone") {
		t.Fatalf("the category carries the scanner's words: %q", blocker.String())
	}
}

// TestUnreadableSessionDirectoryIsClassifiedNotQuoted runs the same failure
// against a directory that exists but denies access, so the classification is
// proven on a real permission error rather than only on a missing path.
func TestUnreadableSessionDirectoryIsClassifiedNotQuoted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a 0000 directory still reads on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root reads a 0000 directory; the missing-path case above covers this on such a host")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "session.v3.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	blocker, err := sessionEncoding(dir)
	if blocker != BlockedUnreadable || err == nil {
		t.Fatalf("denied directory = %v %v", blocker, err)
	}
	if strings.Contains(blocker.String(), dir) {
		t.Fatalf("the category carries the path: %q", blocker.String())
	}
}

// sessionClassifierResults returns the declared result list of the first
// sessionEncoding in source, or "" when there is none.
func sessionClassifierResults(source string) (signature, results string) {
	match := regexp.MustCompile(`(?m)^func sessionEncoding\([^)]*\)\s*\(([^)]*)\)`).FindStringSubmatch(source)
	if match == nil {
		return "", ""
	}
	return match[0], match[1]
}

// TestSessionClassifierReturnsNoFreeText is the structural half of the copy
// rule for this package: a string in sessionEncoding's results is a sentence
// the launcher will paint, and that is how the import dialog grew an English
// OS message as a checkbox's second half. Returns the category; put the
// scanner's words in the error for the log.
func TestSessionClassifierReturnsNoFreeText(t *testing.T) {
	// The shape that shipped on main, kept here as the guard's own check: if
	// this reads as fine, the guard below has stopped seeing the defect.
	const shipped = "func sessionEncoding(dir string) (unsupported, mixed bool, reason string) {"
	if _, results := sessionClassifierResults(shipped); !strings.Contains(results, "string") {
		t.Fatalf("guard no longer recognises the classifier that leaked: %q", results)
	}

	source, err := os.ReadFile("catalog.go")
	if err != nil {
		t.Fatal(err)
	}
	signature, results := sessionClassifierResults(string(source))
	if signature == "" {
		t.Fatal("sessionEncoding is gone or no longer returns a result list; the launcher's category comes from it")
	}
	if strings.Contains(results, "string") {
		t.Fatalf("sessionEncoding returns text a caller can paint (%s); return the Blocker category and keep the scanner's words in the error", signature)
	}
}

// TestApplyRefusalKeepsTheCauseInTheJournal covers the race the launcher can
// really hit: the session was readable at scan time and is not when the copy
// starts. The status line gets a sentence; the cause goes to the journal.
func TestApplyRefusalKeepsTheCauseInTheJournal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a 0000 directory still reads on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root reads a 0000 directory")
	}
	source, dest := t.TempDir(), t.TempDir()
	locked := filepath.Join(source, "sessions", "locked")
	if err := os.MkdirAll(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "session.v3.jsonl"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cat, err := BuildCatalog(source, dest)
	if err != nil {
		t.Fatal(err)
	}
	sel := SelectionFromCatalog(cat, false, false)
	if len(sel.SelectedRels) != 1 {
		t.Fatalf("scan did not select the session: %+v", sel)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	result, err := ApplyCatalog(cat, sel)
	if err == nil {
		t.Fatal("an unreadable session was copied")
	}
	if strings.Contains(result.Message, "permission denied") || strings.Contains(result.Message, locked) {
		t.Fatalf("status message carries the scanner's words: %q", result.Message)
	}
	if !strings.Contains(result.Message, "原因见日志") {
		t.Fatalf("status message does not say where the cause is: %q", result.Message)
	}
	if journal := strings.Join(result.Journal, "\n"); !strings.Contains(journal, "permission denied") {
		t.Fatalf("journal lost the cause: %q", result.Journal)
	}
}

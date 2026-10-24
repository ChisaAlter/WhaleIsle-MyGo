package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestCrashReportFileNameMatchesTheNativePattern pins the native contract
// (reference/electron/src/main/crash-report.js:18-19,81-83): an ISO instant with
// ":" and "." replaced by "-", then the source.
func TestCrashReportFileNameMatchesTheNativePattern(t *testing.T) {
	at := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	if got, want := CrashReportFileName(at, "renderer"), "crash-2026-09-26T12-00-00-000Z-renderer.log"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	// Milliseconds are part of the name, and the name always matches the
	// retention regex.
	millis := time.Date(2026, 9, 26, 12, 0, 0, 123_000_000, time.UTC)
	if got, want := CrashReportFileName(millis, "host"), "crash-2026-09-26T12-00-00-123Z-host.log"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	for _, source := range []string{"host", "web-boot", "renderer", "main", "unknown"} {
		name := CrashReportFileName(millis, source)
		if !crashReportName.MatchString(name) {
			t.Fatalf("%s gave %q, which the retention pattern rejects", source, name)
		}
	}
	// An unknown source cannot escape the pattern or the directory.
	if got := CrashReportFileName(millis, "../../evil"); got != "crash-2026-09-26T12-00-00-123Z-main.log" {
		t.Fatalf("unknown source = %q, want the main fallback", got)
	}
	// The zone is UTC whatever the machine's local zone is.
	local := time.Date(2026, 9, 26, 20, 0, 0, 0, time.FixedZone("CST", 8*3600))
	if got, want := CrashReportFileName(local, "main"), "crash-2026-09-26T12-00-00-000Z-main.log"; got != want {
		t.Fatalf("local time = %q, want the UTC instant %q", got, want)
	}
}

// TestCrashReportRendersEverySection checks the native section order
// (crash-report.js:93-101).
func TestCrashReportRendersEverySection(t *testing.T) {
	body := RenderCrashReport(CrashReportInput{
		Time:           time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
		Source:         "host",
		Error:          "host exited before ready",
		HostDiagnostic: "port 32383 busy",
		RendererConsole: []string{
			"TypeError: x",
		},
	})
	for _, want := range []string{
		"source: host",
		"phase: error",
		"app: Whale Isle ",
		"platform: ",
		"locale: ",
		"shell pid: ",
		"--- error ---",
		"host exited before ready",
		"--- host diagnostic (as reported by the Host process) ---",
		"port 32383 busy",
		"--- renderer console (error level, oldest first) ---",
		"TypeError: x",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("report is missing %q:\n%s", want, body)
		}
	}
	if strings.Index(body, "--- error ---") > strings.Index(body, "--- host diagnostic") {
		t.Fatal("the host diagnostic must follow the error section")
	}
	// No console captured: the native placeholder, not an empty section.
	empty := RenderCrashReport(CrashReportInput{Time: time.Now(), Source: "main", Error: "boom"})
	if !strings.Contains(empty, "(no error-level renderer console output was captured)") {
		t.Fatalf("missing console placeholder:\n%s", empty)
	}
}

// TestCrashReportBoundsTheErrorSection covers ERROR_SECTION_MAX_CHARS
// (crash-report.js:22,85-91).
func TestCrashReportBoundsTheErrorSection(t *testing.T) {
	huge := strings.Repeat("x", CrashErrorSectionMaxChars+4096)
	section := boundedCrashError(huge)
	if !strings.HasPrefix(section, strings.Repeat("x", 1024)) {
		t.Fatal("the kept part must be the start of the error")
	}
	if want := CrashErrorSectionMaxChars + len("\n(error section cut at 262144 characters)"); len(section) != want {
		t.Fatalf("bounded section = %d chars, want %d", len(section), want)
	}
	if !strings.Contains(section, "(error section cut at 262144 characters)") {
		t.Fatalf("missing the cut notice:\n%.120s…", section)
	}
	if got := boundedCrashError("  "); got != "(no error details were captured)" {
		t.Fatalf("empty error = %q", got)
	}
}

// TestCrashConsoleTailKeepsTheNewestBytes covers RENDERER_CONSOLE_MAX_BYTES and
// the UTF-8-safe suffix of an oversized line (crash-report.js:29-73).
func TestCrashConsoleTailKeepsTheNewestBytes(t *testing.T) {
	tail := newCrashConsoleTail(32)
	tail.Push("first line")  // 10 bytes
	tail.Push("second line") // 11 bytes
	if got := tail.Snapshot(); len(got) != 2 {
		t.Fatalf("tail = %q, want both lines while under the 32 byte budget", got)
	}
	// Oldest lines leave first, and the newest line always stays.
	tail.Push("third line") // 10 more: 31, still inside
	tail.Push("fourth")     // 6 more: 37, so "first line" leaves
	got := tail.Snapshot()
	if len(got) == 0 || got[len(got)-1] != "fourth" {
		t.Fatalf("tail = %q, want the newest line kept", got)
	}
	if strings.Contains(strings.Join(got, "\n"), "first line") {
		t.Fatalf("tail = %q, want the oldest line evicted", got)
	}
	for _, kept := range got[1:] {
		if kept == "" {
			t.Fatalf("tail = %q kept an empty line", got)
		}
	}
	if total := len(strings.Join(got, "")); total > 32 && len(got) != 1 {
		t.Fatalf("tail = %q kept %d bytes over the 32 byte budget", got, total)
	}
	// An empty line is dropped rather than counted.
	empties := newCrashConsoleTail(8)
	empties.Push("a")
	empties.Push("")
	if got := empties.Snapshot(); len(got) != 1 || got[0] != "a" {
		t.Fatalf("tail = %q, want the empty line dropped", got)
	}
	// One line larger than the whole budget keeps only its suffix, starting on
	// a rune boundary so no half glyph is decoded.
	oversized := strings.Repeat("é", 40)
	single := newCrashConsoleTail(8)
	single.Push(oversized)
	suffix := single.Snapshot()
	if len(suffix) != 1 || len(suffix[0]) > 8 || !strings.HasSuffix(oversized, suffix[0]) {
		t.Fatalf("oversized line kept %q, want its UTF-8 suffix within 8 bytes", suffix)
	}
	for _, r := range suffix[0] {
		if r == '\uFFFD' {
			t.Fatalf("suffix %q decoded a replacement glyph", suffix[0])
		}
	}
}

// TestCrashReportsRetainTheNewestTen covers CRASH_REPORTS_RETAINED and the
// name-ordered pruning (crash-report.js:119-135).
func TestCrashReportsRetainTheNewestTen(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var names []string
	for i := 0; i < 13; i++ {
		name := CrashReportFileName(base.Add(time.Duration(i)*time.Hour), "main")
		names = append(names, name)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	keep := filepath.Join(dir, "keep-me.log")
	if err := os.WriteFile(keep, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	PruneCrashReports(dir, CrashReportsRetained)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != CrashReportsRetained+1 {
		t.Fatalf("retained %d files, want %d reports and the unrelated file", len(entries), CrashReportsRetained)
	}
	for _, dropped := range names[:3] {
		if _, err := os.Stat(filepath.Join(dir, dropped)); !os.IsNotExist(err) {
			t.Fatalf("oldest report %s survived pruning", dropped)
		}
	}
	for _, kept := range names[len(names)-CrashReportsRetained:] {
		if _, err := os.Stat(filepath.Join(dir, kept)); err != nil {
			t.Fatalf("newest report %s was pruned: %v", kept, err)
		}
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("pruning removed an unrelated file: %v", err)
	}
	// A missing directory is not an error, as the native ENOENT branch.
	PruneCrashReports(filepath.Join(dir, "absent"), CrashReportsRetained)
}

// TestWriteCrashReportIsBestEffort keeps a diagnostic aid from masking the fatal
// error: a failed write reports no file and never panics.
func TestWriteCrashReportIsBestEffort(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "logs")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	// MkdirAll below a regular file must fail, and that failure is swallowed.
	if got := WriteCrashReport(filepath.Join(blocker, "crash"), CrashReportInput{Error: "boom", Source: "main"}); got != "" {
		t.Fatalf("write into a file path returned %q, want no report", got)
	}
	dir := filepath.Join(root, "crash")
	path := WriteCrashReport(dir, CrashReportInput{
		Time:   time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
		Source: "host",
		Error:  "boom",
	})
	if path == "" {
		t.Fatal("a writable directory must produce a report")
	}
	if want := filepath.Join(dir, "crash-2026-09-26T12-00-00-000Z-host.log"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "source: host") || !regexp.MustCompile(`shell pid: \d+`).MatchString(string(body)) {
		t.Fatalf("report body:\n%s", body)
	}
	// An existing file is never overwritten: one failure, one file ("wx").
	again := WriteCrashReport(dir, CrashReportInput{
		Time:   time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
		Source: "host",
		Error:  "second",
	})
	if again != "" {
		t.Fatalf("a second write to the same name returned %q, want no report", again)
	}
	kept, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(kept), "boom") || strings.Contains(string(kept), "second") {
		t.Fatal("the existing report was overwritten")
	}
}

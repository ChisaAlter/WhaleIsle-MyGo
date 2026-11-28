package paths

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"whaleisle/internal/config"
)

// useTempConfigRoot points the data-location record at a throwaway config
// root and clears the cached record for the duration of the test.
func useTempConfigRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	prev := userConfigDir
	userConfigDir = func() (string, error) { return root, nil }
	InvalidateDataLocation()
	t.Cleanup(func() {
		userConfigDir = prev
		InvalidateDataLocation()
	})
	return root
}

func defaultUnder(root string) string {
	return filepath.Join(root, config.AppDirName, config.DefaultDataDirName)
}

func TestStoredDataDirOverridesDefault(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	recorded := filepath.Join(t.TempDir(), "mygo-data")
	if err := os.MkdirAll(recorded, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetStoredDataDir(recorded); err != nil {
		t.Fatalf("SetStoredDataDir: %v", err)
	}
	if got, want := UserDataDir(), recorded; got != want {
		t.Fatalf("UserDataDir() = %q, want %q", got, want)
	}
	if got, want := LocationFile(), filepath.Join(root, config.AppDirName, locationFileName); got != want {
		t.Fatalf("LocationFile() = %q, want %q", got, want)
	}
	// The record must not live inside the directory it names.
	if filepath.Dir(LocationFile()) == recorded {
		t.Fatalf("location record is inside the data directory")
	}
}

func TestEnvBeatsStoredDataDir(t *testing.T) {
	useTempConfigRoot(t)

	recorded := filepath.Join(t.TempDir(), "recorded")
	if err := os.MkdirAll(recorded, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetStoredDataDir(recorded); err != nil {
		t.Fatalf("SetStoredDataDir: %v", err)
	}
	isolated := filepath.Join(t.TempDir(), "isolated")
	t.Setenv(EnvUserData, isolated)
	if got := UserDataDir(); got != isolated {
		t.Fatalf("UserDataDir() = %q, want %q", got, isolated)
	}
}

func TestDefaultRecordLivesOutsideDefaultDataDir(t *testing.T) {
	root := useTempConfigRoot(t)
	if got, want := DefaultUserDataDir(), defaultUnder(root); got != want {
		t.Fatalf("DefaultUserDataDir() = %q, want %q", got, want)
	}
	// The record names a directory; it must not be one of its children.
	if insideOrSame(DefaultUserDataDir(), filepath.Dir(LocationFile())) {
		t.Fatalf("record %q is inside the default data directory %q", LocationFile(), DefaultUserDataDir())
	}
}

func TestCorruptRecordFallsBackToDefault(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	if err := os.MkdirAll(filepath.Join(root, config.AppDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LocationFile(), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateDataLocation()

	if got, want := UserDataDir(), defaultUnder(root); got != want {
		t.Fatalf("UserDataDir() = %q, want default %q", got, want)
	}
}

func TestRecordWithNoDirectoryFallsBackToDefault(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	if err := os.MkdirAll(filepath.Join(root, config.AppDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{}`, `{"dataDir":""}`, `{"dataDir":"relative/path"}`} {
		if err := os.WriteFile(LocationFile(), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		InvalidateDataLocation()
		if got, want := UserDataDir(), defaultUnder(root); got != want {
			t.Fatalf("record %s: UserDataDir() = %q, want default %q", body, got, want)
		}
	}
}

func TestMissingRecordedDirFallsBackToDefault(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	// A directory that was there when it was recorded, and is gone now: an
	// unplugged volume or a folder the user deleted.
	gone := filepath.Join(t.TempDir(), "removable")
	if err := SetStoredDataDir(gone); err != nil {
		t.Fatalf("SetStoredDataDir: %v", err)
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	InvalidateDataLocation()
	// The fallback is logged; the record itself stays, so restoring the
	// directory restores the location on the next launch.
	if got, want := UserDataDir(), defaultUnder(root); got != want {
		t.Fatalf("UserDataDir() = %q, want default %q", got, want)
	}
	if got := RecordedDataDir(); got != gone {
		t.Fatalf("RecordedDataDir() = %q, want the recorded %q", got, gone)
	}
	issue, ok := CachedDataLocationIssue()
	if !ok || issue.Path != gone || issue.Unreadable {
		t.Fatalf("issue = %+v %v, want the missing recorded %q", issue, ok, gone)
	}
}

func TestRecordDataDirKeepsThisProcessOnItsDirectory(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")
	current := filepath.Join(root, "current")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetStoredDataDir(current); err != nil {
		t.Fatal(err)
	}
	if got := UserDataDir(); got != current {
		t.Fatalf("UserDataDir() = %q, want %q", got, current)
	}

	// Changing the location must not move the running process: the next
	// launch opens the new directory, this one keeps writing where it opened.
	next := filepath.Join(root, "next")
	if err := RecordDataDir(next, current); err != nil {
		t.Fatalf("RecordDataDir: %v", err)
	}
	if got := UserDataDir(); got != current {
		t.Fatalf("UserDataDir() moved to %q before a relaunch", got)
	}
	// The cache keeps describing the location in use, so a frame painted
	// after the write still reports the truth for this process.
	if got := RecordedDataDir(); got != current {
		t.Fatalf("RecordedDataDir() = %q, want the cached %q", got, current)
	}
	raw, err := os.ReadFile(LocationFile())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), filepath.Base(next)) {
		t.Fatalf("record does not name the next location: %s", raw)
	}
	if info, err := os.Stat(next); err != nil || !info.IsDir() {
		t.Fatalf("RecordDataDir did not create its target: %v", err)
	}

	// The test-facing setter does move this process, which is how tests point
	// the package elsewhere mid-run.
	if err := SetStoredDataDir(next); err != nil {
		t.Fatal(err)
	}
	if got := UserDataDir(); got != next {
		t.Fatalf("UserDataDir() = %q after SetStoredDataDir, want %q", got, next)
	}
}

func TestCorruptRecordReportsAnIssue(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")
	if err := os.MkdirAll(filepath.Join(root, config.AppDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LocationFile(), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateDataLocation()
	issue, ok := RecordedDataLocationIssue()
	if !ok || !issue.Unreadable || issue.Path != LocationFile() {
		t.Fatalf("issue = %+v %v, want the unreadable record file", issue, ok)
	}
}

func TestClearStoredDataDirRestoresDefault(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	recorded := filepath.Join(t.TempDir(), "recorded")
	if err := os.MkdirAll(recorded, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetStoredDataDir(recorded); err != nil {
		t.Fatalf("SetStoredDataDir: %v", err)
	}
	if err := ClearStoredDataDir(); err != nil {
		t.Fatalf("ClearStoredDataDir: %v", err)
	}
	if got, want := UserDataDir(), defaultUnder(root); got != want {
		t.Fatalf("UserDataDir() = %q, want default %q", got, want)
	}
}

func TestSetStoredDataDirLeavesNoTemporaryFile(t *testing.T) {
	useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	dir := filepath.Join(t.TempDir(), "recorded")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetStoredDataDir(dir); err != nil {
		t.Fatalf("SetStoredDataDir: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(LocationFile()))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != locationFileName {
		t.Fatalf("record directory holds %v, want only %s", namesOf(entries), locationFileName)
	}
}

func namesOf(entries []os.DirEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name())
	}
	return out
}

func TestRecordedLocationThatIsAFileReportsAnIssue(t *testing.T) {
	root := useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A record left by an older run can name a path that has since become a
	// file; it is written directly because RecordDataDir refuses to create it.
	if err := os.MkdirAll(filepath.Dir(LocationFile()), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{"dataDir": file})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LocationFile(), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	InvalidateDataLocation()
	issue, ok := RecordedDataLocationIssue()
	if !ok || issue.Path != file || issue.Unreadable || issue.NotWritable {
		t.Fatalf("issue = %+v %v, want the recorded file %q", issue, ok, file)
	}
	if got, want := UserDataDir(), defaultUnder(root); got != want {
		t.Fatalf("UserDataDir() = %q, want default %q", got, want)
	}
}

func TestRecordedLocationThatRejectsWritesReportsAnIssue(t *testing.T) {
	useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	recorded := filepath.Join(t.TempDir(), "read-only")
	if err := SetStoredDataDir(recorded); err != nil {
		t.Fatal(err)
	}
	InvalidateDataLocation()
	previous := probeDir
	probeDir = func(string) error { return errProbeRefused }
	t.Cleanup(func() { probeDir = previous })

	issue, ok := RecordedDataLocationIssue()
	if !ok || issue.Path != recorded || !issue.NotWritable {
		t.Fatalf("issue = %+v %v, want a not-writable recorded directory", issue, ok)
	}
	if _, ok := CachedDataLocationIssue(); !ok {
		t.Fatal("the cached issue forgot a directory that rejects writes")
	}
	if got := UserDataDir(); strings.EqualFold(got, recorded) {
		t.Fatalf("UserDataDir() = %q, the unwritable recorded directory; continuing with the default must not write there", got)
	}
	if got := StoredDataDir(); strings.EqualFold(got, recorded) {
		t.Fatalf("StoredDataDir() = %q, the unwritable recorded directory", got)
	}
}

func TestDefaultIsNotTheElectronDirectory(t *testing.T) {
	root := useTempConfigRoot(t)
	electron := filepath.Join(root, config.ElectronDataDirName)
	if got := DefaultUserDataDir(); strings.EqualFold(got, electron) {
		t.Fatalf("DefaultUserDataDir() = %q, the Electron directory", got)
	}
}

func TestRecordedPreviousLocationIsOfferedForImport(t *testing.T) {
	useTempConfigRoot(t)
	t.Setenv(EnvUserData, "")

	old := filepath.Join(t.TempDir(), "old-profile")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	next := filepath.Join(t.TempDir(), "new-profile")
	if err := RecordDataDir(next, old); err != nil {
		t.Fatalf("RecordDataDir: %v", err)
	}
	if got := PreviousDataDir(); got != old {
		t.Fatalf("PreviousDataDir() = %q, want the replaced %q", got, old)
	}
	// A record that names itself as previous is not a previous location.
	InvalidateDataLocation()
	if err := RecordDataDir(next, next); err != nil {
		t.Fatalf("RecordDataDir: %v", err)
	}
	if got := PreviousDataDir(); got != "" {
		t.Fatalf("PreviousDataDir() = %q, want none", got)
	}
}

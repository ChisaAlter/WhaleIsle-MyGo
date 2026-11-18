package importer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"whaleisle/internal/config"
)

// firstRunFixture builds a legacy profile with the three artifacts the
// migration treats differently: config.json published as bytes, credentials
// through the credentials package, and dsh-home through the import catalog.
func firstRunFixture(t *testing.T) (source, dest string) {
	t.Helper()
	root := t.TempDir()
	source = filepath.Join(root, "Deepseek-Harness-Desktop")
	dest = filepath.Join(root, "WhaleIsle-MyGo", "data")
	for _, dir := range []string{source, filepath.Join(source, "dsh-home")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeFileAll(t, filepath.Join(source, config.FileName), `{"locale":"zh","workspace":"C:\\work"}`)
	writeFileAll(t, filepath.Join(source, "credentials.json"), `{"apiKey":"fixture-key","baseUrl":"https://fixture.invalid"}`)
	sessionDir := filepath.Join(source, "dsh-home", "sessions", "fixture-cwd", "fixture-session")
	writeFileAll(t, filepath.Join(sessionDir, "session.jsonl"), `{"type":"session","title":"Fixture session"}`+"\n")
	writeFileAll(t, filepath.Join(source, "dsh-home", "settings.yaml"), "ui-theme:\n  mode: dark\n")
	return source, dest
}

func writeFileAll(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, body)
}

func TestDetectFirstRunOffersOnlyForAnEmptyDestination(t *testing.T) {
	source, dest := firstRunFixture(t)

	plan, offered := DetectFirstRun(dest, source)
	if !offered {
		t.Fatal("an empty destination next to a populated legacy profile was not offered")
	}
	if plan.Source != filepath.Clean(source) || plan.Dest != filepath.Clean(dest) {
		t.Fatalf("plan paths: %q -> %q", plan.Source, plan.Dest)
	}

	if _, offered := DetectFirstRun(source, source); offered {
		t.Fatal("the legacy directory was offered as its own destination")
	}
	if _, offered := DetectFirstRun(dest, filepath.Join(t.TempDir(), "absent")); offered {
		t.Fatal("a missing legacy directory was offered")
	}

	// Once MyGo has written its own profile the offer is over.
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dest, config.FileName), `{}`)
	if _, offered := DetectFirstRun(dest, source); offered {
		t.Fatal("a populated destination was offered the legacy import again")
	}
}

func TestDetectFirstRunHonoursSkipAndDone(t *testing.T) {
	source, dest := firstRunFixture(t)
	if err := SkipFirstRun(dest); err != nil {
		t.Fatalf("SkipFirstRun: %v", err)
	}
	if got := FirstRunStatus(dest); got != FirstRunSkipped {
		t.Fatalf("status = %q, want %q", got, FirstRunSkipped)
	}
	if _, offered := DetectFirstRun(dest, source); offered {
		t.Fatal("a skipped offer came back on the next launch")
	}
	// A skipped marker is not one of the artifacts that make a profile used.
	if !isEmptyProfile(dest) {
		t.Fatal("the skip marker made the destination count as populated")
	}
}

func TestPartialImportIsOfferedAgain(t *testing.T) {
	source, dest := firstRunFixture(t)
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	// A run that stopped after copying the profile: the destination is no
	// longer empty, and the marker says the migration is unfinished.
	writeFile(t, filepath.Join(dest, config.FileName), `{}`)
	if err := writeFirstRunMarker(dest, firstRunMarker{Status: FirstRunPartial}); err != nil {
		t.Fatal(err)
	}
	if got := FirstRunStatus(dest); got != FirstRunPartial {
		t.Fatalf("status = %q, want %q", got, FirstRunPartial)
	}
	if _, offered := DetectFirstRun(dest, source); !offered {
		t.Fatal("a partial import was not offered again")
	}
	// A finished or skipped one never comes back.
	for _, status := range []string{FirstRunDone, FirstRunSkipped} {
		if err := writeFirstRunMarker(dest, firstRunMarker{Status: status}); err != nil {
			t.Fatal(err)
		}
		if _, offered := DetectFirstRun(dest, source); offered {
			t.Fatalf("%s import was offered again", status)
		}
	}
}

func TestRunFirstRunCopiesWithoutTouchingTheSource(t *testing.T) {
	source, dest := firstRunFixture(t)
	originalConfig, err := os.ReadFile(filepath.Join(source, config.FileName))
	if err != nil {
		t.Fatal(err)
	}
	plan, offered := DetectFirstRun(dest, source)
	if !offered {
		t.Fatal("offer missing")
	}
	result, err := RunFirstRun(context.Background(), plan)
	if err != nil {
		t.Fatalf("RunFirstRun: %v (%+v)", err, result.Journal)
	}
	copiedConfig, err := os.ReadFile(filepath.Join(dest, config.FileName))
	if err != nil {
		t.Fatalf("config.json not imported: %v", err)
	}
	if string(copiedConfig) != string(originalConfig) {
		t.Fatalf("config.json changed in transit: %q", copiedConfig)
	}
	if _, err := os.Stat(filepath.Join(dest, "dsh-home", "sessions", "fixture-cwd", "fixture-session", "session.jsonl")); err != nil {
		t.Fatalf("session not imported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "dsh-home", "settings.yaml")); err != nil {
		t.Fatalf("settings not imported: %v", err)
	}
	// Credentials go through the credentials package: a platform that
	// refuses the write reports it instead of failing the migration.
	_, credErr := os.Stat(filepath.Join(dest, "credentials.json"))
	journal := strings.Join(result.Journal, "\n")
	if credErr != nil && !strings.Contains(journal, "credentials.json: skipped") {
		t.Fatalf("credentials were neither written nor reported: %v\n%s", credErr, journal)
	}
	if got := FirstRunStatus(dest); got != FirstRunDone {
		t.Fatalf("marker = %q, want done", got)
	}
	// The legacy profile is untouched: copy, never move.
	for _, name := range []string{config.FileName, "credentials.json"} {
		if _, err := os.Stat(filepath.Join(source, name)); err != nil {
			t.Fatalf("source %s disappeared: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(source, "dsh-home", "sessions", "fixture-cwd", "fixture-session")); err != nil {
		t.Fatalf("source session disappeared: %v", err)
	}
}

func TestRunFirstRunIsReentrant(t *testing.T) {
	source, dest := firstRunFixture(t)
	plan, offered := DetectFirstRun(dest, source)
	if !offered {
		t.Fatal("offer missing")
	}
	if _, err := RunFirstRun(context.Background(), plan); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// A second run over the same plan keeps what exists and reports it as
	// skipped rather than replacing or duplicating anything.
	second, err := RunFirstRun(context.Background(), plan)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if second.Copied != 0 || second.Skipped == 0 {
		t.Fatalf("second run copied %d, skipped %d", second.Copied, second.Skipped)
	}
	if _, err := os.Stat(filepath.Join(dest, "first-run-import.json.tmp")); !os.IsNotExist(err) {
		t.Fatalf("a temporary marker file was left behind: %v", err)
	}
}

func TestRunFirstRunRejectsRecursiveRoots(t *testing.T) {
	source, dest := firstRunFixture(t)
	// A destination inside the source would copy the source into itself.
	plan := FirstRunPlan{Source: source, Dest: filepath.Join(source, "nested", "data")}
	if _, err := RunFirstRun(context.Background(), plan); err == nil {
		t.Fatal("accepted a destination inside the source")
	}
	if _, err := os.Stat(plan.Dest); !os.IsNotExist(err) {
		t.Fatalf("refused plan still created %q", plan.Dest)
	}
	if dest == "" {
		t.Fatal("fixture destination missing")
	}
}

func TestFirstRunMarkerTreatsIncompleteRunsAsPartial(t *testing.T) {
	cases := []struct {
		name   string
		result FirstRunResult
		err    error
		want   string
	}{
		{"clean run", FirstRunResult{Copied: 3}, nil, FirstRunDone},
		{"profile file failed", FirstRunResult{Copied: 2, Failed: 1}, nil, FirstRunPartial},
		{"stopped run", FirstRunResult{Copied: 1}, context.Canceled, FirstRunPartial},
	}
	for _, tc := range cases {
		marker := firstRunMarkerFor(tc.result, tc.err)
		if marker.Status != tc.want {
			t.Fatalf("%s: status = %q, want %q", tc.name, marker.Status, tc.want)
		}
		if marker.Copied != tc.result.Copied || marker.Failed != tc.result.Failed {
			t.Fatalf("%s: counts = %+v for %+v", tc.name, marker, tc.result)
		}
	}
}

func TestRunFirstRunWritesPartialWhenItStops(t *testing.T) {
	source, dest := firstRunFixture(t)
	plan, offered := DetectFirstRun(dest, source)
	if !offered {
		t.Fatal("offer missing")
	}
	// The copy stops part way: a file holds the directory the session needs.
	// Recording that as partial is what makes the next launch finish the copy
	// instead of treating the destination as a profile the user chose.
	blocked := filepath.Join(dest, "dsh-home", "sessions", "fixture-cwd")
	if err := os.MkdirAll(filepath.Dir(blocked), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, blocked, "in the way")

	if result, err := RunFirstRun(context.Background(), plan); err == nil {
		t.Fatalf("an import whose session could not be published reported success: %+v", result)
	}
	if got := FirstRunStatus(dest); got != FirstRunPartial {
		t.Fatalf("marker = %q, want %q", got, FirstRunPartial)
	}
	if _, offered := DetectFirstRun(dest, source); !offered {
		t.Fatal("a stopped import was not offered again")
	}
}

// Planning a copy must not read a session log: the offer is drawn from the
// profile's shape, and the catalog that decides what to copy belongs to the
// run. Reading a log used to mean one decoder process per session, which put a
// whole profile's worth of spawns inside the frame that offered the copy.
func TestFirstRunPlanningReadsNoSessionLog(t *testing.T) {
	source, dest := firstRunFixture(t)
	frame := func(s string) []byte {
		n := len(s)
		h := uint32(n<<3 | 1)
		return append([]byte{0x28, 0xb5, 0x2f, 0xfd, 0x20, byte(n), byte(h), byte(h >> 8), byte(h >> 16)}, []byte(s)...)
	}
	writeFileAll(t, filepath.Join(source, "dsh-home", "sessions", "packed-cwd", "packed-session", "session.v4.jsonl.zstd"),
		string(frame(`{"type":"session","title":"Packed","cwd":"/packed"}`+"\n")))

	var decoders atomic.Int32
	previous := readCompressedSessionMeta
	readCompressedSessionMeta = func(context.Context, []string) map[string]sessionMeta {
		decoders.Add(1)
		return nil
	}
	t.Cleanup(func() { readCompressedSessionMeta = previous })

	plan, offered := FirstRunOffer(dest, source)
	if !offered {
		t.Fatal("the legacy profile was not offered as an import source")
	}
	if _, offered := DetectFirstRun(dest, source); !offered {
		t.Fatal("the first-run offer was not planned")
	}
	if got := decoders.Load(); got != 0 {
		t.Fatalf("planning a copy opened %d session decoders on the caller", got)
	}

	if _, err := RunFirstRun(context.Background(), plan); err != nil {
		t.Fatalf("RunFirstRun: %v", err)
	}
	if got := decoders.Load(); got != 1 {
		t.Fatalf("the copy opened %d session decoders, want one for the whole profile", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "dsh-home", "sessions", "packed-cwd", "packed-session")); err != nil {
		t.Fatalf("the packed session was not copied: %v", err)
	}
}

package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

func updateCenterFixture() desktopupdate.NativeRelease {
	return desktopupdate.NativeRelease{
		Feed: desktopupdate.Feed{
			Version: "1.2.3",
			URL:     "https://github.com/ChisaAlter/WhaleIsle-MyGo/releases/download/v1.2.3/fixture.exe",
			SHA256:  strings.Repeat("a", 64),
			Size:    16,
		},
		Tag:           "v1.2.3",
		Platform:      runtime.GOOS,
		Arch:          runtime.GOARCH,
		Format:        "nsis",
		DownloadRoute: "github",
	}
}

func TestNativeUpdateCenterRestoresPinnedPauseAndPreference(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("installed native installer recovery baseline is Windows")
	}
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	release := updateCenterFixture()
	c := NewCenterForTest(nil, "1.2.2", "github")
	c.RestoreReleaseState(ReleaseState{
		CenterLoaded:   true,
		AutoDownload:   true,
		Paused:         true,
		Phase:          "paused",
		Active:         release,
		Route:          "cnb",
		RequireRuntime: true,
		Received:       8,
		Total:          16,
		Staging:        nativeReleaseStagingDir(dir, release),
	})
	c.Persist()

	reopened := NewCenterForTest(nil, "1.2.2", "github")
	reopened.EnsureLoaded()
	r := reopened.ReleaseSnapshot()
	if !r.CenterLoaded || !r.Paused || r.Busy || !r.AutoDownload || !r.RequireRuntime || r.Active.URL != release.URL || r.Active.DownloadRoute != "github" || r.Received != 8 {
		t.Fatalf("recovery=%+v", r)
	}
	if r.Staged.Path != "" {
		t.Fatal("partial file became an installer")
	}
	reopened.Cancel(nil)

	reopened2 := NewCenterForTest(nil, "1.2.2", "github")
	reopened2.EnsureLoaded()
	r2 := reopened2.ReleaseSnapshot()
	if r2.Active.Tag != "" || r2.Paused || !r2.AutoDownload {
		t.Fatal("cancel or preference did not persist")
	}
}

func TestNativeUpdateCenterPreservesMalformedStateAndBoundsDiscard(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	updatesDir := filepath.Join(dir, "updates")
	if err := os.MkdirAll(updatesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(updatesDir, "update-center.json")
	if err := os.WriteFile(name, []byte("invalid fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := NewCenter(nil)
	c.EnsureLoaded()
	if c.ReleaseSnapshot().Err == "" {
		t.Fatal("corrupt recovery was hidden")
	}
	preserved, err := filepath.Glob(name + ".invalid-*")
	if err != nil || len(preserved) != 1 {
		t.Fatalf("original not preserved: %v %v", preserved, err)
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "keep"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if DiscardNativePartial(dir, outside) == nil {
		t.Fatal("outside directory was accepted for recursive removal")
	}
	if b, err := os.ReadFile(filepath.Join(outside, "keep")); err != nil || string(b) != "keep" {
		t.Fatal("unrelated recovery material changed")
	}
}

func TestNativeUpdateCancelDoesNotMutateInstallAdmission(t *testing.T) {
	c := NewCenter(nil)
	c.RestoreReleaseState(ReleaseState{Busy: true, Installing: true, Phase: "installing", StagedTag: "v1.2.3"})
	c.Cancel(nil)
	r := c.ReleaseSnapshot()
	if r.CancelRequested || !r.Installing || r.Phase != "installing" || r.StagedTag != "v1.2.3" {
		t.Fatalf("download cancellation changed installer admission: %+v", r)
	}
}

func TestNativeAutomaticDownloadRetiresAfterIndependentUpgrade(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native installer fixture is Windows NSIS")
	}
	previousTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previousTransport })
	t.Setenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE", "")
	t.Setenv("WHALEISLE_UPDATE_PUBLISHER", "")
	t.Setenv(paths.EnvVendor, "")

	payload := []byte("inert native installer fixture; never executed")
	digest := sha256.Sum256(payload)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	http.DefaultTransport = server.Client().Transport

	for _, explicit := range []bool{false, true} {
		name := "automatic"
		if explicit {
			name = "explicit"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv(paths.EnvUserData, t.TempDir())
			release := updateCenterFixture()
			release.URL = server.URL + "/fixture.exe"
			release.Size = int64(len(payload))
			release.SHA256 = hex.EncodeToString(digest[:])

			c := NewCenterForTest(nil, "1.2.2", "github")
			c.SetVendorReady(true)
			c.RestoreReleaseState(ReleaseState{
				CenterLoaded: true,
				Initialized:  true,
				AutoDownload: true,
				Rows:         []desktopupdate.NativeRelease{release},
				Route:        "github",
				CatalogRoute: "github",
			})
			if explicit {
				c.Download(nil, release.PackageKey(), false)
			} else {
				c.MaybeAutoDownload(nil)
			}

			deadline := time.Now().Add(20 * time.Second)
			var record NativeUpdateRecord
			for time.Now().Before(deadline) {
				rec, err := c.ReadRecord()
				snap := c.ReleaseSnapshot()
				if !snap.Busy && err == nil && rec.Phase == "ready" {
					if rec.Staged.Path == "" || rec.StagedTag != release.Tag || rec.Staged.SHA256 != release.SHA256 || rec.Staged.Size != release.Size {
						t.Fatalf("ready published without a complete durable installer identity: %+v", rec)
					}
					record = rec
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if record.Phase != "ready" || record.ExplicitVersion != explicit {
				t.Fatalf("download intent was not persisted: record=%+v state=%+v", record, c.ReleaseSnapshot())
			}

			reopened := NewCenterForTest(nil, release.Version, "github")
			reopened.EnsureLoaded()
			snap2 := reopened.ReleaseSnapshot()
			if explicit && snap2.Staged.Path == "" {
				t.Fatal("explicit reinstall was retired")
			}
			if !explicit && snap2.Active.Tag != "" {
				t.Fatal("automatic download survived an independent upgrade to the same version")
			}
		})
	}
}

func TestNativeDownloadCompletionPublishesAfterPersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	release := updateCenterFixture()
	c := NewCenterForTest(nil, "1.2.2", "github")
	c.RestoreReleaseState(ReleaseState{Busy: true, Phase: "verifying", Active: release, ExplicitVersion: true})
	staged := desktopupdate.ApplyResult{Path: filepath.Join(nativeReleaseStagingDir(dir, release), "fixture.exe"), SHA256: release.SHA256, Size: release.Size}
	c.diskMu.Lock()
	done := make(chan struct{})
	go func() {
		c.completeDownload(dir, "1.2.2", "github", nil, release, staged, nil)
		close(done)
	}()
	// While persistence is unavailable, polling every UI surface must still
	// see an in-flight update rather than an installable in-memory result.
	snap := c.ReleaseSnapshot()
	tray := c.TraySnapshot()
	if !snap.Busy || snap.Phase == "ready" || snap.Staged.Path != "" || tray.UpdateReady {
		c.diskMu.Unlock()
		t.Fatal("download became ready before its record could be committed")
	}
	c.diskMu.Unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("download completion did not finish")
	}
	snap = c.ReleaseSnapshot()
	record, err := c.ReadRecord()
	if err != nil || snap.Busy || snap.Phase != "ready" || record.Phase != "ready" || record.Staged.Path != staged.Path || record.StagedTag != release.Tag || !record.ExplicitVersion {
		t.Fatalf("published completion and record disagree: state=%+v record=%+v err=%v", snap, record, err)
	}
}

func TestNativeDownloadCompletionPersistenceFailureRetainsBytesWithoutInstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	release := updateCenterFixture()
	staging := nativeReleaseStagingDir(dir, release)
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	installer := filepath.Join(staging, "fixture.exe")
	if err := os.WriteFile(installer, []byte("verified fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A directory in place of the journal deterministically refuses atomic
	// promotion on Windows and Unix without changing user filesystem rights.
	if err := os.Mkdir(filepath.Join(dir, "updates", "update-center.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	c := NewCenterForTest(nil, "1.2.2", "github")
	c.RestoreReleaseState(ReleaseState{Busy: true, Phase: "verifying", Active: release, Staging: staging})
	journal := c.completeDownload(dir, "1.2.2", "github", nil, release, desktopupdate.ApplyResult{Path: installer, Size: release.Size, SHA256: release.SHA256}, nil)
	snap := c.ReleaseSnapshot()
	if snap.Busy || snap.Phase != "error" || snap.Err == "" || !snap.Paused || snap.Staged.Path != "" || c.TraySnapshot().UpdateReady || journal.Phase != desktopupdate.JournalPhaseError {
		t.Fatalf("uncommitted installer was offered: %+v journal=%+v", snap, journal)
	}
	if _, err := os.Stat(installer); err != nil {
		t.Fatalf("verified download was lost instead of being kept for resume: %v", err)
	}
}

func TestNativeDownloadCompletionHonorsAcceptedCancel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	release := updateCenterFixture()
	staging := nativeReleaseStagingDir(dir, release)
	if err := os.MkdirAll(staging, 0o700); err != nil {
		t.Fatal(err)
	}
	installer := filepath.Join(staging, "fixture.exe")
	if err := os.WriteFile(installer, []byte("inert fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := NewCenterForTest(nil, "1.2.2", "github")
	c.RestoreReleaseState(ReleaseState{Busy: true, Phase: "verifying", Active: release, Staging: staging})
	c.Cancel(nil)
	c.completeDownload(dir, "1.2.2", "github", nil, release, desktopupdate.ApplyResult{Path: installer}, nil)
	snap := c.ReleaseSnapshot()
	record, err := c.ReadRecord()
	if err != nil || snap.Busy || snap.Active.Tag != "" || snap.Staged.Path != "" || record.Active.Tag != "" || record.Staged.Path != "" {
		t.Fatalf("accepted cancellation became a ready installer: state=%+v record=%+v err=%v", snap, record, err)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Fatalf("cancelled owned partial was retained: %v", err)
	}
}

func TestAboutOffersFinalReleaseToSameCorePrerelease(t *testing.T) {
	release := updateCenterFixture()
	c := NewCenterForTest(nil, release.Version+"-rc.1", "github")
	c.RestoreReleaseState(ReleaseState{CenterLoaded: true, Rows: []desktopupdate.NativeRelease{release}})
	snap := c.AboutSnapshot(nil)
	if snap.UpdateVersion != release.Version || snap.UpdatePhase != "available" {
		t.Fatalf("final release was hidden from prerelease installation: %+v", snap)
	}
}

func TestAboutSnapshotWithoutHostRestoresPausedStateAndErrors(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("persisted native installer fixture uses the installed Windows baseline")
	}
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })

	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	release := updateCenterFixture()
	seed := NewCenterForTest(nil, "1.2.2", "github")
	seed.RestoreReleaseState(ReleaseState{
		CenterLoaded:   true,
		AutoDownload:   true,
		RequireRuntime: true,
		Paused:         true,
		Phase:          "paused",
		Active:         release,
		Received:       8,
		Total:          16,
	})
	seed.Persist()

	reopened := NewCenterForTest(nil, "1.2.2", "github")
	about := reopened.AboutSnapshot(nil)
	if !about.CenterLoaded || !about.AutoDownload || !about.RequireRuntime || about.UpdatePhase != "paused" || about.UpdatePercent != 50 {
		t.Fatalf("About took an unrestored update snapshot: %+v", about)
	}

	// Corrupt journal surface
	t.Setenv(paths.EnvUserData, t.TempDir())
	updatesDir := filepath.Join(paths.UserDataDir(), "updates")
	if err := os.MkdirAll(updatesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(updatesDir, "update-center.json"), []byte("synthetic malformed"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := NewCenterForTest(nil, "1.2.2", "github")
	brokenAbout := broken.AboutSnapshot(nil)
	if brokenAbout.UpdatePhase != "error" || !strings.Contains(brokenAbout.UpdateStatus, "Could not read update recovery state") {
		t.Fatalf("recovery error was hidden: %+v", brokenAbout)
	}
}

func TestUpdateDownloadProgressAndDetailFormatting(t *testing.T) {
	if got := DownloadDetail(0, 0); got != "" {
		t.Fatalf("empty detail = %q", got)
	}
	if got := DownloadDetail(1536, 0); got != "2 KB" {
		t.Fatalf("received-only detail = %q", got)
	}
	if got := DownloadDetail(1500000, 29000000); got != "1.4 MB / 27.7 MB" {
		t.Fatalf("detail = %q", got)
	}
	if got := DownloadPercent(0, 0); got >= 0 {
		t.Fatalf("unknown total percent = %v", got)
	}
	if got := DownloadPercent(5, 10); got != 50 {
		t.Fatalf("percent = %v", got)
	}
	if got := DownloadPercent(20, 10); got != 100 {
		t.Fatalf("clamped percent = %v", got)
	}
	if got := FormatBytes(3 * 1024 * 1024 * 1024); got != "3.00 GB" {
		t.Fatalf("gigabytes = %q", got)
	}
	if got := FormatBytes(12); got != "12 B" {
		t.Fatalf("bytes = %q", got)
	}
}

func TestUpdateAvailableDetailKeepsEvidenceBounded(t *testing.T) {
	res := desktopupdate.Result{
		Status:              desktopupdate.StatusUpdateAvailable,
		Version:             "2.0.0",
		ReleaseURL:          "https://example.com/secret-installer.exe",
		SHA256:              "abc123",
		Size:                2048,
		IncrementalPossible: true,
	}
	got := AvailableDetail(res, " CNB ")
	for _, want := range []string{"2.0.0", "cnb", "2 KB", "abc123"} {
		if !strings.Contains(got, want) {
			t.Fatalf("detail %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "secret-installer") || strings.Contains(got, "example.com") {
		t.Fatalf("detail leaked download URL: %q", got)
	}
	if got := AvailableDetail(desktopupdate.Result{}, ""); got != "" {
		t.Fatalf("empty detail = %q", got)
	}
}

func TestUpdateJournalRecordsAndStaysBestEffort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	c := NewCenterForTest(nil, "1.0.0", "github")
	c.RecordAction(desktopupdate.JournalActionCheckRequested)
	c.RecordCheckResult(desktopupdate.Result{Status: desktopupdate.StatusUpdateAvailable, Version: "2.0.0"}, nil)
	percent := 42.5
	c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseDownloading, Version: "2.0.0", Percent: &percent})
	c.RecordState(desktopupdate.JournalState{Phase: desktopupdate.JournalPhaseError, FailedOperation: "download", Message: "dial tcp: ETIMEDOUT"})

	files, err := filepath.Glob(filepath.Join(dir, JournalDirectory, "*.jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("journal files = %v (%v)", files, err)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{`"event":"started"`, `"event":"check-requested"`, `"phase":"available"`, `"targetVersion":"2.0.0"`, `"percent":42`, `"errorCode":"ETIMEDOUT"`, `"failedOperation":"download"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("journal missing %s: %s", want, text)
		}
	}
	if strings.Contains(text, "dial tcp") {
		t.Fatalf("journal kept raw diagnostics: %s", text)
	}

	blockedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(blockedDir, JournalDirectory), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(paths.EnvUserData, blockedDir)
	c2 := NewCenterForTest(nil, "1.0.0", "github")
	c2.RecordAction(desktopupdate.JournalActionCheckRequested)
	c2.RecordAction(desktopupdate.JournalActionDownloadRequested)
	if !c2.JournalFailed() {
		t.Fatal("journal failure was not remembered")
	}
}

func TestCheckMessageUsesProductCopy(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	cases := []struct {
		name string
		res  desktopupdate.Result
		err  error
		want string
		deny string
	}{
		{"unconfigured", desktopupdate.Result{Status: desktopupdate.StatusNotConfigured, Message: "MyGo updater feed not configured"}, nil, "no update feed configured", "not_configured"},
		{"latest", desktopupdate.Result{Status: desktopupdate.StatusUpToDate, Version: "1.2.3"}, nil, "latest version", "up_to_date"},
		{"available", desktopupdate.Result{Status: desktopupdate.StatusUpdateAvailable, Version: "2.0.0", ReleaseURL: "https://example.com/r"}, nil, "2.0.0", "update_available"},
		{"failure", desktopupdate.Result{}, errors.New("network down"), "network down", ""},
	}
	for _, tc := range cases {
		got := CheckMessage(tc.res, tc.err)
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s: message %q lacks %q", tc.name, got, tc.want)
		}
		if tc.deny != "" && strings.Contains(got, tc.deny) {
			t.Errorf("%s: raw status %q leaked: %q", tc.name, tc.deny, got)
		}
	}
}

func TestCenterConcurrentSnapshotAndPersist(t *testing.T) {
	t.Setenv(paths.EnvUserData, t.TempDir())
	c := NewCenterForTest(nil, "1.0.0", "github")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c.SetHold(id%2 == 0)
			_ = c.Hold()
			_ = c.AboutSnapshot(nil)
			_ = c.TraySnapshot()
			_ = c.FrameSnapshot()
			c.Persist()
		}(i)
	}
	wg.Wait()
}

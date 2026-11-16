package main

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/app/updater"
	"whaleisle/internal/desktop/taskcontrol"
	"whaleisle/internal/desktop/update"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

func TestTaskInspectionSummaryDistinguishesKnownAndUnknownWork(t *testing.T) {
	previous := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(previous) })
	for _, locale := range []string{"zh", "en"} {
		i18n.SetLocale(locale)
		inspection := taskcontrol.Inspection{LocalRunning: 2, ActiveWork: make([]taskcontrol.Work, 3), ScheduledWork: make([]taskcontrol.Work, 4)}
		known := taskInspectionSummary(inspection)
		for _, count := range []string{"2", "3", "4"} {
			if !strings.Contains(known, count) {
				t.Fatalf("%s known work lost count %s: %s", locale, count, known)
			}
		}
		if strings.Contains(known, "%!") || strings.Contains(known, "unknown") || strings.Contains(known, "未知") {
			t.Fatalf("%s known work was presented as unknown or malformed: %s", locale, known)
		}
		inspection.Unknown = "inspection unavailable"
		unknown := taskInspectionSummary(inspection)
		if strings.Contains(unknown, "%!") || (!strings.Contains(unknown, "unknown") && !strings.Contains(unknown, "未知")) {
			t.Fatalf("%s missing unknown-work warning: %s", locale, unknown)
		}
	}
}

func TestNativeUpdateCancelDoesNotMutateInstallAdmission(t *testing.T) {
	s := &mainState{}
	u := s.updaterCenter()
	u.RestoreReleaseState(updater.ReleaseState{Busy: true, Installing: true, Phase: "installing", StagedTag: "v1.2.3"})
	u.Cancel(nil)
	r := u.ReleaseSnapshot()
	if r.CancelRequested || !r.Installing || r.Phase != "installing" || r.StagedTag != "v1.2.3" {
		t.Fatalf("download cancellation changed installer admission: %+v", r)
	}
}

func TestNativeAutomaticDownloadRetiresAfterIndependentUpgrade(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native installer fixture is Windows NSIS")
	}
	previousVersion, previousTransport := buildVersion, http.DefaultTransport
	t.Cleanup(func() { buildVersion, http.DefaultTransport = previousVersion, previousTransport })
	t.Setenv("WHALEISLE_UPDATE_REQUIRE_SIGNATURE", "")
	t.Setenv("WHALEISLE_UPDATE_PUBLISHER", "")
	t.Setenv(paths.EnvVendor, "")
	payload := []byte("inert native installer fixture; never executed")
	digest := sha256.Sum256(payload)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(payload) }))
	defer server.Close()
	http.DefaultTransport = server.Client().Transport
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "automatic", true: "explicit"}[explicit], func(t *testing.T) {
			t.Setenv(paths.EnvUserData, t.TempDir())
			buildVersion = "1.2.2"
			release := updateCenterFixture()
			release.URL, release.Size, release.SHA256 = server.URL+"/fixture.exe", int64(len(payload)), hex.EncodeToString(digest[:])
			s := &mainState{}
			u := s.updaterCenter()
			u.SetVendorReady(true)
			u.SetFallbackRoute("github")
			u.RestoreReleaseState(updater.ReleaseState{
				CenterLoaded: true, Initialized: true, AutoDownload: true,
				Rows: []update.NativeRelease{release}, Route: "github", CatalogRoute: "github",
			})
			if explicit {
				u.Download(nil, release.PackageKey(), false)
			} else {
				u.MaybeAutoDownload(nil)
			}
			deadline := time.Now().Add(20 * time.Second)
			var record updater.NativeUpdateRecord
			for time.Now().Before(deadline) {
				rec, err := u.ReadRecord()
				snap := u.ReleaseSnapshot()
				if !snap.Busy && err == nil && rec.Phase == "ready" {
					record = rec
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if record.Phase != "ready" || record.ExplicitVersion != explicit {
				t.Fatalf("download intent was not persisted: record=%+v state=%+v", record, u.ReleaseSnapshot())
			}
			buildVersion = release.Version
			reopened := &mainState{}
			reopened.ensureNativeUpdateCenter()
			snap := reopened.updaterCenter().ReleaseSnapshot()
			if explicit && snap.Staged.Path == "" {
				t.Fatal("explicit reinstall was retired")
			}
			if !explicit && snap.Active.Tag != "" {
				t.Fatal("automatic download survived an independent upgrade to the same version")
			}
		})
	}
}

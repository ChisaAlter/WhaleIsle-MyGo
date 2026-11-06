package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	s.nativeReleases = nativeReleaseState{busy: true, installing: true, phase: "installing", stagedTag: "v1.2.3"}
	s.cancelNativeUpdate(nil)
	if s.nativeReleases.cancelRequested || !s.nativeReleases.installing || s.nativeReleases.phase != "installing" || s.nativeReleases.stagedTag != "v1.2.3" {
		t.Fatalf("download cancellation changed installer admission: %+v", s.nativeReleases)
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
			s.startupRecovery.updateVendorReady = true
			s.launcherDownloadRoute = "github"
			s.nativeReleases = nativeReleaseState{centerLoaded: true, initialized: true, autoDownload: true, rows: []update.NativeRelease{release}, route: "github", catalogRoute: "github"}
			if explicit {
				s.downloadNativeRelease(nil, release.PackageKey(), false)
			} else {
				s.maybeAutoDownloadNative(nil)
			}
			name := filepath.Join(paths.UserDataDir(), "updates", "update-center.json")
			deadline := time.Now().Add(20 * time.Second)
			var record nativeUpdateRecord
			for time.Now().Before(deadline) {
				// Use the same publication lock as the recovery reader. A bare
				// Windows read handle can otherwise deny the atomic rename.
				nativeUpdateDiskMu.Lock()
				raw, err := os.ReadFile(name)
				nativeUpdateDiskMu.Unlock()
				s.mu.Lock()
				busy := s.nativeReleases.busy
				s.mu.Unlock()
				if !busy && err == nil && json.Unmarshal(raw, &record) == nil && record.Phase == "ready" {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if record.Phase != "ready" || record.ExplicitVersion != explicit {
				s.mu.Lock()
				state := s.nativeReleases
				s.mu.Unlock()
				t.Fatalf("download intent was not persisted: record=%+v state=%+v", record, state)
			}
			buildVersion = release.Version
			reopened := &mainState{}
			reopened.ensureNativeUpdateCenter()
			if explicit && reopened.nativeReleases.staged.Path == "" {
				t.Fatal("explicit reinstall was retired")
			}
			if !explicit && reopened.nativeReleases.active.Tag != "" {
				t.Fatal("automatic download survived an independent upgrade to the same version")
			}
		})
	}
}

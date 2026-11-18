package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/launcher"
)

// This exercises real HTTP and the launcher event simulation. It does not
// certify Windows input or the native window's update queue.
func TestLauncherUpdateCheckDoesNotHoldInput(t *testing.T) {
	checkLabel := "检查更新"
	if i18n.Locale() == i18n.LocaleEN {
		checkLabel = "Check for updates"
	}
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var requests atomic.Int32
			requested := make(chan struct{}, 4)
			release := make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				requested <- struct{}{}
				<-release
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(desktopupdate.Feed{Version: buildVersion, URL: "https://example.invalid/desktop.zip"})
			}))
			defer server.Close()
			defer unblock()
			t.Setenv(desktopupdate.EnvFeedURL, server.URL)
			s := &mainState{}
			s.launcherController().SetUpdateAvailableForTest(true)
			dispatched := make(chan error, 1)
			go func() {
				left := false
				tt := ui.NewTester(func(c *ui.Context) {
					if ui.Button(c, "Back to workbench").Clicked() {
						left = true
					}
					launcher.UpdateCenter(c, launcher.Model{VendorReady: true,
						OnRefreshReleases: func() { s.checkLauncherUpdate(nil) },
					})
				}, 1200, 800)
				for _, label := range []string{checkLabel, checkLabel, "Back to workbench"} {
					if err := tt.Click(label); err != nil {
						dispatched <- err
						return
					}
				}
				if !left {
					dispatched <- fmt.Errorf("pending check prevented navigation")
					return
				}
				dispatched <- nil
			}()
			select {
			case <-requested:
			case <-time.After(3 * time.Second):
				t.Fatal("update feed was not requested")
			}
			select {
			case err := <-dispatched:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("launcher dispatch waited for the gated HTTP response")
			}
			if !s.launcherController().UpdateProbeState().CheckBusy {
				t.Fatal("check lost its busy state before the response")
			}
			select {
			case <-requested:
				t.Fatal("duplicate click started another request while one was pending")
			case <-time.After(100 * time.Millisecond):
			}
			unblock()
			deadline := time.Now().Add(3 * time.Second)
			for {
				st := s.launcherController().UpdateProbeState()
				busy, message, available := st.CheckBusy, st.Message, st.Available
				if !busy {
					want := string(desktopupdate.StatusUpToDate)
					if status != http.StatusOK {
						want = "HTTP 503"
					}
					if !strings.Contains(message, want) || available {
						t.Fatalf("incorrect result: message=%q available=%v", message, available)
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("completed check did not clear busy")
				}
				time.Sleep(5 * time.Millisecond)
			}
			if got := requests.Load(); got != 1 {
				t.Fatalf("requests=%d, want 1", got)
			}
		})
	}
}

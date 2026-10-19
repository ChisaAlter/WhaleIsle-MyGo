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
			s := &mainState{launcherUpdateAvailable: true}
			dispatched := make(chan error, 1)
			go func() {
				tab := ""
				tt := ui.NewTester(func(c *ui.Context) {
					launcher.View(c, launcher.Model{VendorReady: true, ActiveTab: launcher.TabHome,
						OnCheckUpdate: func() { s.checkLauncherUpdate(nil) },
						OnTab:         func(id string) { tab = id },
					})
				}, 1200, 800)
				for _, label := range []string{i18n.T(i18n.KeyLauncherCheckUpdate), i18n.T(i18n.KeyLauncherCheckUpdate), i18n.T(i18n.KeyLauncherTabComponents)} {
					if err := tt.Click(label); err != nil {
						dispatched <- err
						return
					}
				}
				if tab != launcher.TabComponents {
					dispatched <- fmt.Errorf("pending check prevented rail callback: %q", tab)
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
			s.mu.Lock()
			busy := s.launcherUpdateCheckBusy
			s.mu.Unlock()
			if !busy {
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
				s.mu.Lock()
				busy, message, available := s.launcherUpdateCheckBusy, s.launcherUpdate, s.launcherUpdateAvailable
				s.mu.Unlock()
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

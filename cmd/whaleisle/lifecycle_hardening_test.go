package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	assistantctl "whaleisle/internal/app/assistant"
	launcherctl "whaleisle/internal/app/launcher"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestSkillHubTimeoutAndLifecycleCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := &skillHubTestRPC{resultName: "owned-skill", holdMethod: "skillInventory/searchHub", release: make(chan struct{})}
		defer f.unblock()
		s := newSkillHubTestState(t, f)
		s.searchSettingsSkillHub("synthetic")
		synctest.Wait()
		if len(f.calls) != 1 {
			t.Fatalf("expected 1 search call, got %d", len(f.calls))
		}
		call := f.calls[0]
		if _, ok := call.ctx.Deadline(); !ok {
			t.Fatal("skill hub search RPC lacked bounded timeout")
		}
		// Trigger lifecycle cancellation directly as quit does without nil window dependencies.
		s.mu.Lock()
		s.quitting = true
		cancel := s.lifecycleCancel
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		synctest.Wait()
		if call.ctx.Err() != context.Canceled {
			t.Fatalf("expected lifecycle quit to cancel in-flight search RPC, got %v", call.ctx.Err())
		}
		// Allow held transport to drain so goroutine exits cleanly inside synctest bubble.
		f.unblock()
		synctest.Wait()
	})
}

func TestIsolatedDesktopRejectsJunctionToInstalledDataDir(t *testing.T) {
	root := t.TempDir()
	userConfigBase := filepath.Join(root, "AppData")
	installed := filepath.Join(userConfigBase, config.DataDirName)
	if err := os.MkdirAll(installed, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"",
		"relative-dir",
		installed,
		strings.ToLower(installed),
	} {
		if _, err := launcherctl.ValidateIsolatedUserDataDir(bad, userConfigBase); err == nil {
			t.Fatalf("expected rejection for %q", bad)
		}
	}
	good := filepath.Join(root, "isolated-profile", "child")
	if canonical, err := launcherctl.ValidateIsolatedUserDataDir(good, userConfigBase); err != nil || canonical == "" {
		t.Fatalf("expected unrelated isolated path to pass, got canonical=%q err=%v", canonical, err)
	}

	linkPath := filepath.Join(root, "junction-data")
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", linkPath, installed).CombinedOutput(); err != nil {
			t.Skipf("mklink /J unavailable in test environment: %v (%s)", err, string(out))
		}
	} else {
		if err := os.Symlink(installed, linkPath); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	}
	t.Cleanup(func() { _ = os.Remove(linkPath) })

	if _, err := launcherctl.ValidateIsolatedUserDataDir(linkPath, userConfigBase); err == nil {
		t.Fatalf("expected link pointing to installed data dir %q to be rejected", linkPath)
	}
	// Prefix inside a non-existent child still resolves via its link parent, never equal to installed unless target is itself; verify canonical target escapes check when pointing straight at installed.
	subInstalled := filepath.Join(installed, "sub")
	linkParent := filepath.Join(root, "junction-parent")
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", linkParent, userConfigBase).CombinedOutput(); err != nil {
			t.Skipf("mklink /J parent unavailable: %v (%s)", err, string(out))
		}
	} else {
		if err := os.Symlink(userConfigBase, linkParent); err != nil {
			t.Skipf("symlink parent unavailable: %v", err)
		}
	}
	t.Cleanup(func() { _ = os.Remove(linkParent) })

	aliasedInstalledChild := filepath.Join(linkParent, config.DataDirName, "sub")
	resolvedChild, err := launcherctl.ResolveCanonicalPath(aliasedInstalledChild)
	if err != nil {
		t.Fatalf("ResolveCanonicalPath failed: %v", err)
	}
	if !strings.EqualFold(resolvedChild, subInstalled) {
		t.Fatalf("expected canonical child %q, got %q", subInstalled, resolvedChild)
	}
	aliasedInstalled := filepath.Join(linkParent, config.DataDirName)
	if _, err := launcherctl.ValidateIsolatedUserDataDir(aliasedInstalled, userConfigBase); err == nil {
		t.Fatalf("expected alias via parent junction %q to be rejected", aliasedInstalled)
	}
}

func TestCachedWhaleDigestLocked(t *testing.T) {
	var s mainState
	svc := s.assistantService()
	c1 := &dsh.Client{}
	ep := harness.Endpoint{Origin: "http://127.0.0.1:1234", Cookie: "super-secret-cookie"}
	d1 := svc.WhaleDigest(c1, ep, 1)
	if d1 == "" || strings.Contains(d1, "super-secret") {
		t.Fatalf("unexpected digest %q", d1)
	}
	d2 := svc.WhaleDigest(c1, ep, 1)
	if d1 != d2 {
		t.Fatalf("digest did not hit cache: %q vs %q", d1, d2)
	}
	d3 := svc.WhaleDigest(c1, ep, 2)
	if d3 == d1 {
		t.Fatal("digest did not change across host generations")
	}
	if svc.WhaleDigest(nil, ep, 2) != "" {
		t.Fatal("nil client produced non-empty digest")
	}
}

func TestObserveWhaleNavLabelRetainsNameAndBackoffOnError(t *testing.T) {
	var failMode atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if failMode.Load() {
			http.Error(w, "transient down", http.StatusInternalServerError)
			return
		}
		var req struct {
			RPCID string `json:"rpcId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": req.RPCID,
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"name":    "小鲸",
					"memory":  "m",
					"homeDir": "/tmp/whale",
				},
			},
		})
	}))
	defer srv.Close()

	cfgStore, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ep := harness.Endpoint{Origin: srv.URL, Cookie: "cookie-a"}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}}
	s := &mainState{
		cfgStore:   cfgStore,
		dshClient:  client,
		bootStatus: harness.Status{State: harness.Ready, Generation: 1, Endpoint: ep},
	}
	t.Cleanup(func() {
		s.mu.Lock()
		s.quitting = true
		cancel := s.lifecycleCancel
		s.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	})

	svc := s.assistantService()
	s.observeWhaleNavLabel(ep)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		st := svc.WhaleNavStatus()
		if !st.Busy && st.Name == "小鲸" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	st := svc.WhaleNavStatus()
	if st.Name != "小鲸" {
		t.Fatalf("expected whaleNavName=小鲸, got %q", st.Name)
	}
	// Expire cache to force refetch under transient RPC error.
	svc.ExpireWhaleNavForTest()

	failMode.Store(true)
	s.observeWhaleNavLabel(ep)
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		st = svc.WhaleNavStatus()
		if !st.Busy && !st.FetchedAt.IsZero() && st.Name == "小鲸" {
			// Verify backoff is ~5s remaining before next retry (< WhaleNavRefetch ago, > WhaleNavRefetch-10s ago).
			if age := time.Since(st.FetchedAt); age < assistantctl.WhaleNavRefetch-10*time.Second {
				t.Fatalf("expected backoff timestamp near retry window, got age=%v", age)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	st = svc.WhaleNavStatus()
	t.Fatalf("timed out waiting for error retention/backoff: busy=%v name=%q fetchedAt=%v", st.Busy, st.Name, st.FetchedAt)
}

func TestNativeReleaseNotifyProgressThrottles(t *testing.T) {
	var s mainState
	u := s.updaterCenter()
	u.NotifyProgress(nil)
	first := u.LastProgressNotifyAt()
	if first == 0 {
		t.Fatal("expected initial progress timestamp to be recorded")
	}
	u.NotifyProgress(nil)
	if second := u.LastProgressNotifyAt(); second != first {
		t.Fatalf("expected immediate second call within 120ms to throttle: %d vs %d", first, second)
	}
}

func TestLauncherNoWorkspaceLocalizationAndQuitLifecycleCtx(t *testing.T) {
	if got := i18n.T(i18n.KeyLauncherNoWorkspace); strings.TrimSpace(got) == "" || got == "workspace not configured" {
		t.Fatalf("expected localized missing workspace text, got %q", got)
	}
	var s mainState
	ctx := s.lifecycleContext()
	if ctx.Err() != nil {
		t.Fatalf("expected live lifecycle context, got %v", ctx.Err())
	}
	s.mu.Lock()
	s.quitting = true
	cancel := s.lifecycleCancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if ctx.Err() != context.Canceled {
		t.Fatalf("expected canceled lifecycle context on quit, got %v", ctx.Err())
	}
	if postQuit := s.lifecycleContext(); postQuit.Err() != context.Canceled {
		t.Fatalf("expected post-quit lifecycleContext to return already-canceled ctx, got %v", postQuit.Err())
	}
}

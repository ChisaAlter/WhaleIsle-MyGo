package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/interaction"
)

// A restarted Host retains the interaction Store and its old synchronous notify.
// That callback re-enters mainState through cancelComposerVoiceIfStale. Initial
// startup has no old notify, so testing only a new Store misses the deadlock.
func TestDialogueRebindAutoAllowNotificationCanReenterMainState(t *testing.T) {
	root := t.TempDir()
	t.Setenv(paths.EnvUserData, filepath.Join(root, "profile"))
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("USERPROFILE", filepath.Join(root, "home"))
	store, err := config.Open(paths.UserDataDir())
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Update(func(c *config.Config) { c.AutoAllowTools = true; c.Workspace = filepath.Join(root, "workspace") }); err != nil {
		t.Fatal(err)
	}
	// No Host is required: every attempted read has a bounded local response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "owned rebind fixture", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	inter := interaction.New()
	s := &mainState{cfgStore: store, interactions: inter}
	var entered, underLock atomic.Bool
	inter.Attach(func() {
		if !inter.Snapshot().AutoAllow {
			return
		}
		entered.Store(true)
		// Fail the original lock contract without leaving a deadlocked goroutine in
		// the test process. The real old callback uses Lock, as proven by run3 stacks.
		if !s.mu.TryLock() {
			underLock.Store(true)
			return
		}
		s.mu.Unlock()
		s.cancelComposerVoiceIfStale()
	})
	done := make(chan struct{})
	go func() {
		s.syncDialogue(harness.Status{State: harness.Ready, Endpoint: harness.Endpoint{Origin: server.URL}})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Fatal("Host rebind blocked while publishing retained interaction state")
	}
	defer s.stopDSH()
	if !entered.Load() {
		t.Fatal("rebind did not publish accepted auto-allow to the retained Store")
	}
	if underLock.Load() {
		t.Fatal("retained interaction notify ran while mainState.mu was held")
	}
	if !inter.Snapshot().AutoAllow {
		t.Fatal("rebind lost accepted auto-allow")
	}
}

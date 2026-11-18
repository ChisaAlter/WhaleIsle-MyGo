package main

import (
	"context"
	"sync/atomic"
	"testing"

	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/dsh"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	imstore "whaleisle/internal/store/im"
)

func cleanupDeferredStoreTestState(s *mainState) {
	s.mu.Lock()
	s.quitting = true
	cancel := s.lifecycleCancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.assistantService().SuspendAll()
	s.stopDSH()
}

func TestDeferredColdStoresRaceSafeAgainstEarlyHostReady(t *testing.T) {
	dir := t.TempDir()
	earlyClient := &dsh.Client{}
	earlyCtx, cancelEarly := context.WithCancel(context.Background())
	defer cancelEarly()

	placeholderIM := imstore.New()
	placeholderIM.Attach(earlyClient, func() {})

	s := &mainState{
		dshClient:  earlyClient,
		dshCtx:     earlyCtx,
		dshRefresh: func() {},
		im:         placeholderIM,
	}
	t.Cleanup(func() { cleanupDeferredStoreTestState(s) })

	var calls atomic.Int32
	ops := &deferredColdStoreOps{
		openIM: func(userData string) (*imstore.Store, error) {
			calls.Add(1)
			return imstore.Open(userData)
		},
		openAssistant: func(userData string) (*assiststore.Store, error) {
			calls.Add(1)
			return assiststore.Open(userData)
		},
		openProactive: func(userData string) {
			calls.Add(1)
			s.openProactive(userData)
		},
		openCustomPanels: func(userData string) {
			calls.Add(1)
			s.openCustomPanels(userData)
		},
		openDshbotRoster: func(userData string) (*dshbotstore.Store, error) {
			calls.Add(1)
			return dshbotstore.Open(userData)
		},
		loadSSHTunnel: func(string) (sshremote.TunnelConfig, error) {
			calls.Add(1)
			return sshremote.TunnelConfig{Host: "deferred.example", Port: 2222, User: "alice"}, nil
		},
	}

	done1 := s.triggerDeferredColdStores(dir, ops)
	done2 := s.triggerDeferredColdStores(dir, ops)
	if done1 == nil || done2 != done1 {
		t.Fatalf("expected shared completion channel, got %v and %v", done1, done2)
	}
	<-done1

	if calls.Load() != 6 {
		t.Fatalf("deferred openers ran %d times, want 6 (sync.Once deduplication)", calls.Load())
	}
	s.mu.Lock()
	gotIM := s.im
	gotCustom := s.customPanels.Store
	gotHost, gotPort, gotUser, gotTunnel := s.sshHost, s.sshPort, s.sshUser, s.sshTunnel
	s.mu.Unlock()

	if gotIM == nil || gotIM == placeholderIM || gotIM.Root() != imstore.Dir(dir) {
		t.Fatalf("expected persisted IM store replacing placeholder, got %v (root=%q)", gotIM, func() string {
			if gotIM == nil {
				return ""
			}
			return gotIM.Root()
		}())
	}
	if s.assistantService().AssistantStore() == nil {
		t.Fatal("expected assistant store attached after deferred load")
	}
	if s.assistantService().ProactiveService() == nil {
		t.Fatal("expected proactive service attached after deferred load")
	}
	if s.assistantService().DshbotRoster() == nil {
		t.Fatal("expected dshbot roster attached after deferred load")
	}
	if gotCustom == nil {
		t.Fatal("expected custom panels store attached after deferred load")
	}
	if gotTunnel == nil || gotHost != "deferred.example" || gotPort != "2222" || gotUser != "alice" {
		t.Fatalf("unexpected SSH tunnel config applied: tunnel=%v host=%q port=%q user=%q", gotTunnel != nil, gotHost, gotPort, gotUser)
	}
}

func TestDeferredColdStoresPreserveEarlySSHTunnelEdits(t *testing.T) {
	dir := t.TempDir()
	existing := sshremote.NewTunnelSession()
	s := &mainState{
		sshTunnel: existing,
		sshHost:   "user-edited.example",
		sshPort:   "2200",
	}
	t.Cleanup(func() { cleanupDeferredStoreTestState(s) })

	ops := &deferredColdStoreOps{
		loadSSHTunnel: func(string) (sshremote.TunnelConfig, error) {
			return sshremote.TunnelConfig{Host: "disk.example", Port: 22}, nil
		},
	}
	<-s.triggerDeferredColdStores(dir, ops)

	s.mu.Lock()
	gotTunnel, gotHost, gotPort := s.sshTunnel, s.sshHost, s.sshPort
	s.mu.Unlock()
	if gotTunnel != existing || gotHost != "user-edited.example" || gotPort != "2200" {
		t.Fatalf("deferred load overwrote early SSH edits: host=%q port=%q", gotHost, gotPort)
	}
}

func TestDeferredColdStoresRespectQuitCancellation(t *testing.T) {
	s := &mainState{quitting: true}
	var calls atomic.Int32
	ops := &deferredColdStoreOps{
		openIM:           func(string) (*imstore.Store, error) { calls.Add(1); return nil, nil },
		openAssistant:    func(string) (*assiststore.Store, error) { calls.Add(1); return nil, nil },
		openProactive:    func(string) { calls.Add(1) },
		openCustomPanels: func(string) { calls.Add(1) },
		openDshbotRoster: func(string) (*dshbotstore.Store, error) { calls.Add(1); return nil, nil },
		loadSSHTunnel:    func(string) (sshremote.TunnelConfig, error) { calls.Add(1); return sshremote.TunnelConfig{}, nil },
	}
	<-s.triggerDeferredColdStores(t.TempDir(), ops)
	if calls.Load() != 0 {
		t.Fatalf("deferred openers ran %d times after quitting started", calls.Load())
	}
}

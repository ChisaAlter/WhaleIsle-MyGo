package main

import (
	"context"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/paths"
)

func phoneResumeState(t *testing.T, enabled bool) *mainState {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(paths.EnvUserData, dir)
	// Missing resources make a start attempt observable without starting a
	// real daemon or depending on the developer's installed runtime.
	t.Setenv(paths.EnvVendor, t.TempDir())
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(c *config.Config) { c.RemoteEnabled = enabled }); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store, phoneActionMsg: "waiting"}
	s.lifecycleContext()
	t.Cleanup(s.lifecycleCancel)
	return s
}

func waitPhoneResume(t *testing.T, finished <-chan struct{}) {
	t.Helper()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("phone stop continuation did not settle")
	}
}

func TestPhoneStopContinuationUsesLatestEnabledState(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			s := phoneResumeState(t, !enabled)
			stopped, finished := make(chan struct{}), make(chan struct{})
			go func() {
				s.resumePhoneRemoteAfterStop(s.lifecycleContext(), stopped, 0)
				close(finished)
			}()
			if err := s.cfgStore.Update(func(c *config.Config) { c.RemoteEnabled = enabled }); err != nil {
				t.Fatal(err)
			}
			select {
			case <-finished:
				t.Fatal("saved settings applied before the stopped child exited")
			default:
			}
			close(stopped)
			waitPhoneResume(t, finished)
			got := s.phoneRuntimeManager().Snapshot().Status
			if enabled && got != "missing" {
				t.Fatalf("latest enabled state did not attempt a start: %s", got)
			}
			if !enabled && (got != "offline" || s.phoneActionMsg != "disabled") {
				t.Fatalf("latest disabled state restarted the gateway: %s / %s", got, s.phoneActionMsg)
			}
		})
	}
}

func TestPhoneStopContinuationAlreadyExited(t *testing.T) {
	s := phoneResumeState(t, true)
	stopped := make(chan struct{})
	close(stopped)
	s.resumePhoneRemoteAfterStop(s.lifecycleContext(), stopped, 0)
	if got := s.phoneRuntimeManager().Snapshot().Status; got != "missing" {
		t.Fatalf("exit before subscription lost the continuation: %s", got)
	}
}

func TestPhoneStopContinuationCanceled(t *testing.T) {
	for _, kind := range []string{"application quit", "explicit Host stop", "superseded after exit"} {
		t.Run(kind, func(t *testing.T) {
			s := phoneResumeState(t, true)
			ctx, cancel := context.WithCancel(s.lifecycleContext())
			defer cancel()
			stopped, finished := make(chan struct{}), make(chan struct{})
			s.phoneOp.Lock()
			s.phoneStopResume = cancel
			go func() {
				s.resumePhoneRemoteAfterStop(ctx, stopped, 0)
				close(finished)
			}()
			switch kind {
			case "application quit":
				s.lifecycleCancel()
			case "superseded after exit":
				close(stopped)
				s.cancelPhoneStopResumeLocked()
			}
			s.phoneOp.Unlock()
			if kind == "explicit Host stop" {
				s.stopPhoneRuntime()
			}
			waitPhoneResume(t, finished)
			if s.phoneActionMsg != "waiting" || s.phoneRuntimeManager().Snapshot().Status != "offline" {
				t.Fatalf("canceled continuation resumed the gateway: %s", s.phoneActionMsg)
			}
		})
	}
}

func TestPhoneApplyIntentBeforeExplicitStopCannotRestart(t *testing.T) {
	s := phoneResumeState(t, true)
	// A save has reserved its intent, but its worker has not been scheduled.
	s.phoneIntent = 1
	s.stopPhoneRuntime()
	s.applyPhoneGatewayIntent(1)
	if got := s.phoneRuntimeManager().Snapshot().Status; got != "offline" || s.phoneActionMsg != "waiting" {
		t.Fatalf("queued old save restarted after explicit Host stop: %s / %s", got, s.phoneActionMsg)
	}
	// A later user save is a new intent and may start normally.
	s.mu.Lock()
	s.phoneIntent++
	intent := s.phoneIntent
	s.mu.Unlock()
	s.applyPhoneGatewayIntent(intent)
	if got := s.phoneRuntimeManager().Snapshot().Status; got != "missing" {
		t.Fatalf("new save was suppressed by the previous stop: %s", got)
	}
}

package main

import (
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

func TestStopDSHClosesAccountPlatformOutsideDSHLock(t *testing.T) {
	uiDone := make(chan struct{})
	closed := make(chan struct{})
	s := &mainState{}
	s.dshClient = &dsh.Client{}
	s.dshMux = &dsh.Mux{}
	s.dshMuxReady = true
	s.accountReturn.platform = &accountPlatformLease{
		cancel: func() {
			go func() {
				// Simulate a UI-thread RunOnMain callback acquiring dshMu
				// synchronously during platform lease teardown (e.g. host.Close).
				s.dshMu.Lock()
				_ = s.dshClient
				s.dshMu.Unlock()
				close(uiDone)
			}()
			select {
			case <-uiDone:
			case <-time.After(2 * time.Second):
			}
			close(closed)
		},
	}

	stopDone := make(chan struct{})
	go func() {
		s.stopDSH()
		close(stopDone)
	}()

	select {
	case <-stopDone:
	case <-time.After(3 * time.Second):
		t.Fatal("stopDSH deadlocked with dshMu acquisition inside platform lease teardown")
	}
	select {
	case <-closed:
	default:
		t.Fatal("platform lease teardown did not run")
	}
	select {
	case <-uiDone:
	default:
		t.Fatal("simulated RunOnMain acquiring dshMu did not complete during teardown")
	}

	s.dshMu.Lock()
	s.mu.Lock()
	if s.dshClient != nil || s.dshMux != nil || s.dshMuxReady {
		s.mu.Unlock()
		s.dshMu.Unlock()
		t.Fatalf("expected cleared dsh fields, got client=%v mux=%v ready=%v", s.dshClient, s.dshMux, s.dshMuxReady)
	}
	s.mu.Unlock()
	s.dshMu.Unlock()
}

func TestDSHStateWriteReadConsistencyUnderBothLocks(t *testing.T) {
	s := &mainState{}
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Writer cycling attach -> ready -> stopDSH using strict dshMu -> mu order.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			client := &dsh.Client{}
			mux := &dsh.Mux{}
			s.dshMu.Lock()
			s.mu.Lock()
			s.dshClient = client
			s.dshMux = nil
			s.dshMuxReady = false
			s.mu.Unlock()
			s.dshMu.Unlock()

			s.dshMu.Lock()
			s.mu.Lock()
			s.dshMux = mux
			s.dshMuxReady = true
			s.mu.Unlock()
			s.dshMu.Unlock()

			s.stopDSH()
		}
		close(done)
	}()

	// Reader under mu only (models project_* controllers).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				s.mu.Lock()
				c, m, r := s.dshClient, s.dshMux, s.dshMuxReady
				s.mu.Unlock()
				if r && (c == nil || m == nil) {
					t.Errorf("inconsistent ready state under mu: client=%v mux=%v", c, m)
					return
				}
				if m != nil && c == nil {
					t.Errorf("inconsistent mux without client under mu")
					return
				}
			}
		}
	}()

	// Reader under dshMu only (models connectionAttemptState.ownsHost).
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				s.dshMu.Lock()
				c, m := s.dshClient, s.dshMux
				s.dshMu.Unlock()
				if m != nil && c == nil {
					t.Errorf("inconsistent mux without client under dshMu")
					return
				}
			}
		}
	}()

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
	case <-time.After(5 * time.Second):
		t.Fatal("lock consistency stress test timed out")
	}
}

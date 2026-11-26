package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"whaleisle/internal/desktop/sshremote"
	"whaleisle/internal/ui/i18n"
)

func newSSHOperation(t *testing.T) (*mainState, *sshConnectOperation) {
	t.Helper()
	s := &mainState{sshTunnelMsg: "current status"}
	s.mu.Lock()
	op := s.beginSSHConnectLocked(sshremote.TunnelConfig{Host: "owned.invalid"})
	s.mu.Unlock()
	t.Cleanup(op.cancel)
	return s, op
}

func invalidateSSHOperation(s *mainState, action string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidateSSHConnectLocked()
	switch action {
	case "save":
		s.sshTunnel.SetConfig(sshremote.TunnelConfig{Host: "replacement.invalid"})
		s.sshTunnelMsg = "saved"
	case "disconnect":
		s.sshTunnel.Disconnect()
		s.sshTunnelMsg = "disconnected"
	case "replace":
		op := s.beginSSHConnectLocked(sshremote.TunnelConfig{Host: "new.invalid"})
		s.sshTunnelMsg = "new attempt"
		// This test does not launch the replacement operation.
		op.cancel()
	}
}

func waitSSHOperation(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("SSH operation did not finish")
	}
}

func TestSSHConnectOperationStalePromptAccept(t *testing.T) {
	for _, action := range []string{"save", "disconnect", "replace"} {
		t.Run(action, func(t *testing.T) {
			s, op := newSSHOperation(t)
			promptEntered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			knownHosts := filepath.Join(t.TempDir(), "known_hosts")
			connects, accepts, redraws := 0, 0, 0
			go func() {
				defer close(done)
				op.run(sshConnectEffects{
					connect: func(ctx context.Context) error {
						connects++
						return &sshremote.UnknownHostKey{Address: "owned.invalid:22"}
					},
					prompt: func(_ *sshremote.UnknownHostKey, beforeShow func() bool) bool {
						close(promptEntered)
						<-release
						if beforeShow() {
							t.Error("stale queued prompt retained ownership")
						}
						// Also model Accept from a dialog already visible at invalidation.
						return true
					},
					accept: func(*sshremote.UnknownHostKey) error {
						accepts++
						return os.WriteFile(knownHosts, []byte("must not be persisted"), 0600)
					},
					redraw: func() { redraws++ },
				})
			}()
			waitSSHOperation(t, promptEntered)
			invalidateSSHOperation(s, action)
			s.mu.Lock()
			want := s.sshTunnelMsg
			s.mu.Unlock()
			if !errors.Is(op.ctx.Err(), context.Canceled) {
				t.Fatal("invalidation did not cancel the operation context")
			}
			close(release)
			waitSSHOperation(t, done)
			if connects != 1 || accepts != 0 || redraws != 0 || s.sshTunnelMsg != want {
				t.Fatalf("late acceptance mutated state: connects=%d accepts=%d redraws=%d status=%q", connects, accepts, redraws, s.sshTunnelMsg)
			}
			if _, err := os.Stat(knownHosts); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("late acceptance wrote known_hosts: %v", err)
			}
		})
	}
}

func TestSSHConnectOperationLateResult(t *testing.T) {
	for _, result := range []error{nil, errors.New("old failure"), &sshremote.UnknownHostKey{}, &sshremote.HostKeyChanged{}} {
		t.Run(func() string {
			if result == nil {
				return "success"
			}
			return result.Error()
		}(), func(t *testing.T) {
			s, op := newSSHOperation(t)
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			go func() {
				defer close(done)
				op.run(sshConnectEffects{
					connect: func(ctx context.Context) error { close(entered); <-release; return result },
					prompt:  func(*sshremote.UnknownHostKey, func() bool) bool { t.Error("late result opened prompt"); return true },
					changed: func(*sshremote.HostKeyChanged, func() bool) { t.Error("late result opened warning") },
					accept:  func(*sshremote.UnknownHostKey) error { t.Error("late result wrote host key"); return nil },
					redraw:  func() { t.Error("late result published") },
				})
			}()
			waitSSHOperation(t, entered)
			invalidateSSHOperation(s, "disconnect")
			close(release)
			waitSSHOperation(t, done)
			if s.sshTunnelMsg != "disconnected" {
				t.Fatalf("late result overwrote status: %q", s.sshTunnelMsg)
			}
		})
	}
}

func TestSSHConnectOperationCurrentAcceptRetry(t *testing.T) {
	s, op := newSSHOperation(t)
	connects, accepts, redraws := 0, 0, 0
	op.run(sshConnectEffects{
		connect: func(ctx context.Context) error {
			if ctx != op.ctx || ctx.Err() != nil {
				t.Error("connect did not receive current cancelable context")
			}
			connects++
			if connects == 1 {
				return &sshremote.UnknownHostKey{Address: "owned.invalid:22"}
			}
			return nil
		},
		prompt: func(_ *sshremote.UnknownHostKey, beforeShow func() bool) bool { return beforeShow() },
		accept: func(*sshremote.UnknownHostKey) error { accepts++; return nil },
		redraw: func() { redraws++ },
	})
	if connects != 2 || accepts != 1 || redraws != 1 || s.sshTunnelMsg != i18n.T(i18n.KeyRemoteSSHConnected) {
		t.Fatalf("current acceptance failed: connects=%d accepts=%d redraws=%d status=%q", connects, accepts, redraws, s.sshTunnelMsg)
	}
}

func TestSSHConnectOperationCanceledAfterPersistence(t *testing.T) {
	s, op := newSSHOperation(t)
	connects, accepts := 0, 0
	op.run(sshConnectEffects{
		connect: func(context.Context) error {
			connects++
			return &sshremote.UnknownHostKey{Address: "owned.invalid:22"}
		},
		prompt: func(_ *sshremote.UnknownHostKey, beforeShow func() bool) bool { return beforeShow() },
		accept: func(*sshremote.UnknownHostKey) error {
			accepts++
			// Cancellation can arrive after the serialized write returns and
			// before retry. It must also fence that boundary, not only prompts.
			op.cancel()
			return nil
		},
		redraw: func() { t.Error("canceled operation published") },
	})
	if connects != 1 || accepts != 1 || s.sshTunnelMsg != "current status" {
		t.Fatalf("canceled operation retried/published: connects=%d accepts=%d status=%q", connects, accepts, s.sshTunnelMsg)
	}
}

func TestSSHConnectOperationOwnerReplacementAndQuit(t *testing.T) {
	for _, replaceTunnel := range []bool{false, true} {
		s, op := newSSHOperation(t)
		s.mu.Lock()
		if replaceTunnel {
			s.sshTunnel = sshremote.NewTunnelSession()
		} else {
			s.quitting = true
		}
		s.mu.Unlock()
		if op.current() || op.publish("stale") {
			t.Fatal("lost owner still current")
		}
		op.run(sshConnectEffects{connect: func(context.Context) error { t.Fatal("stale operation connected"); return nil }})
	}
}

func startLoopbackSSHServerForTest(t *testing.T) (host string, port int, password string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	const wantPass = "test-secret"
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if string(pass) == wantPass {
				return &ssh.Permissions{}, nil
			}
			return nil, errors.New("denied")
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			raw, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				sc, chans, reqs, err := ssh.NewServerConn(c, cfg)
				if err != nil {
					return
				}
				defer sc.Close()
				go ssh.DiscardRequests(reqs)
				for ch := range chans {
					_ = ch.Reject(ssh.UnknownChannelType, "unused")
				}
			}(raw)
		}
	}()
	h, pStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(pStr)
	if err != nil {
		t.Fatal(err)
	}
	return h, p, wantPass
}

func TestFinalizeQuitCancelsSSHConnectAndDisconnectsTunnel(t *testing.T) {
	host, port, pass := startLoopbackSSHServerForTest(t)
	tunnel := sshremote.NewTunnelSession()
	tunnel.SetKnownHostsFile(filepath.Join(t.TempDir(), "known_hosts"))
	tunnel.SetHostKeyMode(sshremote.HostKeyOff)
	tunnel.SetConfig(sshremote.TunnelConfig{
		Host:     host,
		Port:     port,
		User:     "tester",
		Password: pass,
	})
	if err := tunnel.Connect(context.Background()); err != nil {
		t.Fatalf("connect test SSH tunnel: %v", err)
	}
	t.Cleanup(tunnel.Disconnect)

	if _, st, _ := tunnel.Snapshot(); st != sshremote.TunnelConnected {
		t.Fatalf("tunnel state before quit = %q, want %q", st, sshremote.TunnelConnected)
	}

	opCtx, opCancel := context.WithCancel(context.Background())
	defer opCancel()
	s := &mainState{
		sshTunnel:        tunnel,
		sshConnectEpoch:  1,
		sshConnectCancel: opCancel,
	}

	s.finalizeQuit(nil)

	if !errors.Is(opCtx.Err(), context.Canceled) {
		t.Fatalf("finalizeQuit did not cancel in-flight SSH connect context: %v", opCtx.Err())
	}
	s.mu.Lock()
	epoch := s.sshConnectEpoch
	cancelFn := s.sshConnectCancel
	s.mu.Unlock()
	if epoch != 2 || cancelFn != nil {
		t.Fatalf("finalizeQuit did not invalidate SSH connect state: epoch=%d cancel=%v", epoch, cancelFn != nil)
	}
	if _, st, _ := tunnel.Snapshot(); st != sshremote.TunnelIdle {
		t.Fatalf("tunnel state after finalizeQuit = %q, want %q", st, sshremote.TunnelIdle)
	}
}

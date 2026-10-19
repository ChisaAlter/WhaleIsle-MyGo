package sshremote

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// The SSH server implements only TCP forwarding to this test's echo server.
// It cannot execute commands, open sessions, or dial any other destination.
type loopbackSSH struct {
	listener    net.Listener
	mu          sync.Mutex
	signer      ssh.Signer
	clientKey   ssh.PublicKey
	target      string
	connections map[net.Conn]bool
	done        chan struct{}
}

func newLoopbackSSH(t *testing.T, target string, clientKey ssh.PublicKey) *loopbackSSH {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	signer, _ := testHostKey(t)
	s := &loopbackSSH{listener: l, signer: signer, clientKey: clientKey, target: target, connections: make(map[net.Conn]bool), done: make(chan struct{})}
	go func() {
		defer close(s.done)
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.connections[c] = true
			signer := s.signer
			s.mu.Unlock()
			go s.serve(c, signer)
		}
	}()
	t.Cleanup(func() {
		_ = l.Close()
		<-s.done
		s.mu.Lock()
		defer s.mu.Unlock()
		for c := range s.connections {
			_ = c.Close()
		}
	})
	return s
}

func (s *loopbackSSH) serve(raw net.Conn, signer ssh.Signer) {
	defer raw.Close()
	defer func() { s.mu.Lock(); delete(s.connections, raw); s.mu.Unlock() }()
	cfg := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if !bytes.Equal(key.Marshal(), s.clientKey.Marshal()) {
			return nil, fmt.Errorf("test client key required")
		}
		return nil, nil
	}}
	cfg.AddHostKey(signer)
	c, channels, requests, err := ssh.NewServerConn(raw, cfg)
	if err != nil {
		return
	}
	defer c.Close()
	var listeners sync.Map
	defer listeners.Range(func(_, v any) bool { _ = v.(net.Listener).Close(); return true })
	go func() {
		for req := range requests {
			var p struct {
				Host string
				Port uint32
			}
			if ssh.Unmarshal(req.Payload, &p) != nil || p.Host != "127.0.0.1" {
				_ = req.Reply(false, nil)
				continue
			}
			addr := net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port)))
			switch req.Type {
			case "tcpip-forward":
				l, e := net.Listen("tcp", addr)
				if e != nil {
					_ = req.Reply(false, nil)
					continue
				}
				listeners.Store(addr, l)
				_ = req.Reply(true, nil)
				go func() {
					for {
						local, e := l.Accept()
						if e != nil {
							return
						}
						go func() {
							payload := struct {
								Host       string
								Port       uint32
								Origin     string
								OriginPort uint32
							}{p.Host, p.Port, "127.0.0.1", 1}
							ch, rs, e := c.OpenChannel("forwarded-tcpip", ssh.Marshal(payload))
							if e != nil {
								local.Close()
								return
							}
							go ssh.DiscardRequests(rs)
							copyLoopback(local, ch)
						}()
					}
				}()
			case "cancel-tcpip-forward":
				if l, ok := listeners.LoadAndDelete(addr); ok {
					_ = l.(net.Listener).Close()
				}
				_ = req.Reply(true, nil)
			default:
				_ = req.Reply(false, nil)
			}
		}
	}()
	for next := range channels {
		var p struct {
			Host       string
			Port       uint32
			Origin     string
			OriginPort uint32
		}
		if next.ChannelType() != "direct-tcpip" || ssh.Unmarshal(next.ExtraData(), &p) != nil || net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port))) != s.target {
			_ = next.Reject(ssh.Prohibited, "only owned echo destination allowed")
			continue
		}
		local, e := net.DialTimeout("tcp", s.target, time.Second)
		if e != nil {
			_ = next.Reject(ssh.ConnectionFailed, "echo unavailable")
			continue
		}
		ch, rs, e := next.Accept()
		if e != nil {
			local.Close()
			continue
		}
		go ssh.DiscardRequests(rs)
		go copyLoopback(local, ch)
	}
}

func copyLoopback(a io.ReadWriteCloser, b io.ReadWriteCloser) {
	defer a.Close()
	defer b.Close()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
	_ = a.Close()
	_ = b.Close()
	<-done
}

func loopbackProfile(t *testing.T) (string, ssh.PublicKey) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("SSH_AUTH_SOCK", "")
	_, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	block, e := ssh.MarshalPrivateKey(key, "owned-loopback-only")
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(root, "synthetic-key")
	if e = os.WriteFile(file, pem.EncodeToMemory(block), 0600); e != nil {
		t.Fatal(e)
	}
	signer, e := ssh.NewSignerFromKey(key)
	if e != nil {
		t.Fatal(e)
	}
	return file, signer.PublicKey()
}

func loopbackPort(t *testing.T) int {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func awaitTunnel(t *testing.T, s *TunnelSession, want func(TunnelState) bool) {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for time.Now().Before(until) {
		_, state, _ := s.Snapshot()
		if want(state) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	_, state, message := s.Snapshot()
	t.Fatalf("tunnel state=%s message=%s", state, message)
}

func TestFunctionalOwnedSSHHostKeyForwardAndPeerClose(t *testing.T) {
	identity, key := loopbackProfile(t)
	echo, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer echo.Close()
	go func() {
		for {
			c, e := echo.Accept()
			if e != nil {
				return
			}
			go func() { defer c.Close(); _, _ = io.Copy(c, c) }()
		}
	}()
	server := newLoopbackSSH(t, echo.Addr().String(), key)
	port := server.listener.Addr().(*net.TCPAddr).Port
	s := NewTunnelSession()
	defer s.Disconnect()
	kh := filepath.Join(filepath.Dir(identity), "known_hosts")
	s.SetKnownHostsFile(kh)
	s.SetTrustOnFirstUse(false)
	autoPort := loopbackPort(t)
	autoAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(autoPort))
	s.SetConfig(TunnelConfig{Host: "127.0.0.1", Port: port, User: "owned", IdentityFile: identity,
		LocalForward: fmt.Sprintf("%d:127.0.0.1:%d", autoPort, echo.Addr().(*net.TCPAddr).Port)})
	var unknown *UnknownHostKey
	if err := s.Connect(context.Background()); !errors.As(err, &unknown) {
		t.Fatalf("unknown key not rejected: %v", err)
	}
	if data, e := os.ReadFile(kh); e != nil || len(data) != 0 {
		t.Fatal("unknown key auto-accepted", e)
	}
	if e = AcceptHostKey(kh, server.listener.Addr().String(), unknown.Key); e != nil {
		t.Fatal(e)
	}
	if e = s.Connect(context.Background()); e != nil {
		t.Fatal(e)
	}
	awaitTunnel(t, s, func(state TunnelState) bool { return state == TunnelConnected })
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	for _, direction := range []ForwardDirection{ForwardLocal, ForwardReverse} {
		t.Run(string(direction), func(t *testing.T) {
			listenPort := loopbackPort(t)
			stop, e := StartForward(context.Background(), client, ForwardSpec{Direction: direction, ListenHost: "127.0.0.1", ListenPort: listenPort, TargetHost: "127.0.0.1", TargetPort: echo.Addr().(*net.TCPAddr).Port})
			if e != nil {
				t.Fatal(e)
			}
			defer stop()
			addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
			c, e := net.DialTimeout("tcp", addr, time.Second)
			if e != nil {
				t.Fatal(e)
			}
			defer c.Close()
			_ = c.SetDeadline(time.Now().Add(2 * time.Second))
			payload := []byte("owned SSH echo / 合成数据")
			if _, e = c.Write(payload); e != nil {
				t.Fatal(e)
			}
			got := make([]byte, len(payload))
			if _, e = io.ReadFull(c, got); e != nil || !bytes.Equal(got, payload) {
				t.Fatal("forward echo", e)
			}
			stop()
			stop()
			if _, e = c.Read(make([]byte, 1)); e == nil {
				t.Fatal("active forward survived stop")
			} else {
				var timeout net.Error
				if errors.As(e, &timeout) && timeout.Timeout() {
					t.Fatal("active forward timed out instead of closing")
				}
			}
			if next, e := net.DialTimeout("tcp", addr, 100*time.Millisecond); e == nil {
				next.Close()
				t.Fatal("forward listener survived stop")
			}
		})
	}
	var automatic net.Conn
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		automatic, e = net.DialTimeout("tcp", autoAddr, 50*time.Millisecond)
		if e == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if e != nil {
		t.Fatal("configured automatic forward did not listen", e)
	}
	defer automatic.Close()
	_ = automatic.SetDeadline(time.Now().Add(time.Second))
	if _, e = automatic.Write([]byte("auto")); e != nil {
		t.Fatal(e)
	}
	if _, e = io.ReadFull(automatic, make([]byte, 4)); e != nil {
		t.Fatal("automatic echo", e)
	}
	server.mu.Lock()
	for c := range server.connections {
		_ = c.Close()
	}
	server.mu.Unlock()
	awaitTunnel(t, s, func(state TunnelState) bool { return state != TunnelConnected && state != TunnelConnecting })
	if _, e = automatic.Read(make([]byte, 1)); e == nil {
		t.Fatal("peer exit retained active forward")
	} else {
		var timeout net.Error
		if errors.As(e, &timeout) && timeout.Timeout() {
			t.Fatal("peer exit timed out instead of closing active forward")
		}
	}
	until = time.Now().Add(time.Second)
	var reclaimed net.Listener
	for time.Now().Before(until) {
		reclaimed, e = net.Listen("tcp", autoAddr)
		if e == nil {
			_ = reclaimed.Close()
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if e != nil {
		t.Fatal("peer exit retained automatic listener", e)
	}
	s.Disconnect()
	s.Disconnect()
	// Same endpoint with another synthetic host key must never replace the pin.
	before, e := os.ReadFile(kh)
	if e != nil {
		t.Fatal(e)
	}
	replacement, _ := testHostKey(t)
	server.mu.Lock()
	server.signer = replacement
	server.mu.Unlock()
	var changed *HostKeyChanged
	if e = s.Connect(context.Background()); !errors.As(e, &changed) {
		t.Fatalf("changed key not rejected: %v", e)
	}
	after, e := os.ReadFile(kh)
	if e != nil || !bytes.Equal(before, after) {
		t.Fatal("changed key replaced pin", e)
	}
	t.Log("real SSH handshake: unknown rejected, own key explicitly accepted, changed rejected; local+reverse forwarding byte echo and active/listener close; peer exit reflected")
}

func TestFunctionalOwnedSSHHandshakeCancellation(t *testing.T) {
	identity, _ := loopbackProfile(t)
	for _, disconnect := range []bool{false, true} {
		t.Run(fmt.Sprintf("disconnect=%v", disconnect), func(t *testing.T) {
			l, e := net.Listen("tcp", "127.0.0.1:0")
			if e != nil {
				t.Fatal(e)
			}
			defer l.Close()
			accepted := make(chan net.Conn, 1)
			go func() {
				c, e := l.Accept()
				if e == nil {
					accepted <- c
				}
			}()
			s := NewTunnelSession()
			defer s.Disconnect()
			s.SetKnownHostsFile(filepath.Join(filepath.Dir(identity), "known_hosts-cancel"))
			s.SetConfig(TunnelConfig{Host: "127.0.0.1", Port: l.Addr().(*net.TCPAddr).Port, User: "owned", IdentityFile: identity})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- s.Connect(ctx) }()
			var peer net.Conn
			select {
			case peer = <-accepted:
			case <-time.After(time.Second):
				t.Fatal("owned peer not accepted")
			}
			defer peer.Close()
			if e = s.Connect(context.Background()); e == nil {
				t.Fatal("parallel connect admitted")
			}
			if disconnect {
				s.Disconnect()
			} else {
				cancel()
			}
			select {
			case e = <-done:
				if !errors.Is(e, context.Canceled) {
					t.Fatalf("cancel=%v", e)
				}
			case <-time.After(time.Second):
				t.Fatal("SSH handshake cancellation stuck")
			}
			_, state, _ := s.Snapshot()
			if disconnect && state != TunnelIdle {
				t.Fatalf("late failure overwrote disconnect: %s", state)
			}
		})
	}
}

func TestTunnelPreCanceledRetryPreservesDisconnectedState(t *testing.T) {
	root := t.TempDir()
	s := NewTunnelSession()
	s.SetKnownHostsFile(filepath.Join(root, "known_hosts"))
	s.SetConfig(TunnelConfig{Host: "127.0.0.1", User: "owned", IdentityFile: filepath.Join(root, "not-created-key")})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.Disconnect()
	if err := s.Connect(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled retry returned %v", err)
	}
	_, state, message := s.Snapshot()
	if state != TunnelIdle || message != "" {
		t.Fatalf("canceled retry overwrote disconnect: %s %q", state, message)
	}
	if _, err := os.Stat(filepath.Join(root, "known_hosts")); !os.IsNotExist(err) {
		t.Fatal("canceled retry touched known_hosts")
	}
}

func TestFunctionalOwnedSSHSessionHostKeyPolicy(t *testing.T) {
	identity, key := loopbackProfile(t)
	server := newLoopbackSSH(t, "", key)
	s := NewTunnelSession()
	defer s.Disconnect()
	kh := filepath.Join(filepath.Dir(identity), "policy-known_hosts")
	s.SetKnownHostsFile(kh)
	cfg := TunnelConfig{Host: "127.0.0.1", Port: server.listener.Addr().(*net.TCPAddr).Port, User: "owned", IdentityFile: identity}
	s.SetConfig(cfg)
	s.SetHostKeyMode(HostKeyVerify)
	var unknown *UnknownHostKey
	if err := s.Connect(context.Background()); !errors.As(err, &unknown) {
		t.Fatalf("verify must reject unknown own key: %v", err)
	}
	// This is an explicitly selected policy only for the owned synthetic server.
	s.SetHostKeyMode(HostKeyOff)
	if err := s.Connect(context.Background()); err != nil {
		t.Fatal("explicit session policy not honored", err)
	}
	s.Disconnect()
	if b, err := os.ReadFile(kh); err != nil || len(b) != 0 {
		t.Fatal("off policy changed trust store", err)
	}
	cfg.HostKeyMode = string(HostKeyVerify)
	s.SetConfig(cfg)
	if err := s.Connect(context.Background()); !errors.As(err, &unknown) {
		t.Fatalf("explicit config must override session policy: %v", err)
	}
	cfg.HostKeyMode = ""
	s.SetConfig(cfg)
	s.SetHostKeyMode(HostKeyAcceptNew)
	if err := s.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(kh); err != nil || len(b) == 0 {
		t.Fatal("accept-new did not pin own key", err)
	}
	t.Log("explicit own-session verify/off/accept-new policies honored; config override wins; default policy unchanged")
}

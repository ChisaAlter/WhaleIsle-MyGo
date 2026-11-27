package sshremote

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestLoadSaveTunnelConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := TunnelConfig{Host: "example.com", User: "alice", Port: 2222, IdentityFile: "id_ed25519"}
	if err := SaveTunnelConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := LoadTunnelConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != cfg.Host || got.User != cfg.User || got.Port != cfg.Port {
		t.Fatalf("%+v", got)
	}
	path := ConfigPath(dir)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	_ = filepath.Base(path)
}

func TestLoadTunnelConfigMissing(t *testing.T) {
	got, err := LoadTunnelConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got.Port != 22 {
		t.Fatalf("port=%d", got.Port)
	}
}

func TestTunnelKeepaliveTimeoutTransitionsToError(t *testing.T) {
	signer, _ := testHostKey(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	release := make(chan struct{})
	defer close(release)
	go func() {
		raw, err := ln.Accept()
		if err != nil {
			return
		}
		defer raw.Close()
		cfg := &ssh.ServerConfig{
			PasswordCallback: func(_ ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
				if string(pass) == "pw" {
					return nil, nil
				}
				return nil, errors.New("bad password")
			},
		}
		cfg.AddHostKey(signer)
		c, chans, reqs, err := ssh.NewServerConn(raw, cfg)
		if err != nil {
			return
		}
		defer c.Close()
		go func() {
			for ch := range chans {
				_ = ch.Reject(ssh.Prohibited, "no")
			}
		}()
		// Intentionally never reply to keepalive@openssh.com requests.
		for range reqs {
			<-release
			return
		}
	}()

	s := NewTunnelSession()
	defer s.Disconnect()
	s.SetHostKeyMode(HostKeyOff)
	s.mu.Lock()
	s.keepaliveInterval = 30 * time.Millisecond
	s.keepaliveTimeout = 80 * time.Millisecond
	s.mu.Unlock()
	s.SetConfig(TunnelConfig{
		Host:     "127.0.0.1",
		Port:     ln.Addr().(*net.TCPAddr).Port,
		User:     "alice",
		Password: "pw",
	})
	if err := s.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	awaitTunnel(t, s, func(state TunnelState) bool { return state == TunnelError })
	_, state, msg := s.Snapshot()
	if state != TunnelError || msg == "" {
		t.Fatalf("expected TunnelError with keepalive message, got state=%s msg=%q", state, msg)
	}
}

func TestTunnelKeyboardInteractiveAuthAndForwardHalfClose(t *testing.T) {
	// Target server reads until client EOF (half-close), then sends its full reply.
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer targetLn.Close()
	go func() {
		for {
			c, err := targetLn.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				req, err := io.ReadAll(conn)
				if err != nil {
					return
				}
				// Delay slightly to ensure a premature bidirectional close would truncate it.
				time.Sleep(40 * time.Millisecond)
				_, _ = conn.Write(append([]byte("REPLY:"), req...))
			}(c)
		}
	}()

	signer, _ := testHostKey(t)
	sshLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer sshLn.Close()
	targetAddr := targetLn.Addr().String()
	go func() {
		for {
			raw, err := sshLn.Accept()
			if err != nil {
				return
			}
			go func(raw net.Conn) {
				defer raw.Close()
				cfg := &ssh.ServerConfig{
					KeyboardInteractiveCallback: func(_ ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
						answers, err := client("alice", "", []string{"Password: "}, []bool{false})
						if err != nil || len(answers) != 1 || answers[0] != "kbd-secret" {
							return nil, errors.New("keyboard-interactive rejected")
						}
						return nil, nil
					},
				}
				cfg.AddHostKey(signer)
				c, chans, reqs, err := ssh.NewServerConn(raw, cfg)
				if err != nil {
					return
				}
				defer c.Close()
				go ssh.DiscardRequests(reqs)
				for next := range chans {
					if next.ChannelType() != "direct-tcpip" {
						_ = next.Reject(ssh.Prohibited, "no")
						continue
					}
					remote, err := net.DialTimeout("tcp", targetAddr, time.Second)
					if err != nil {
						_ = next.Reject(ssh.ConnectionFailed, err.Error())
						continue
					}
					ch, chReqs, err := next.Accept()
					if err != nil {
						_ = remote.Close()
						continue
					}
					go ssh.DiscardRequests(chReqs)
					go func() {
						defer remote.Close()
						defer ch.Close()
						done := make(chan struct{}, 2)
						go func() {
							_, _ = io.Copy(remote, ch)
							if tc, ok := remote.(*net.TCPConn); ok {
								_ = tc.CloseWrite()
							}
							done <- struct{}{}
						}()
						go func() {
							_, _ = io.Copy(ch, remote)
							_ = ch.CloseWrite()
							done <- struct{}{}
						}()
						<-done
						<-done
					}()
				}
			}(raw)
		}
	}()

	s := NewTunnelSession()
	defer s.Disconnect()
	s.SetHostKeyMode(HostKeyOff)
	s.SetConfig(TunnelConfig{
		Host:                "127.0.0.1",
		Port:                sshLn.Addr().(*net.TCPAddr).Port,
		User:                "alice",
		Password:            "kbd-secret",
		KeyboardInteractive: true,
	})
	if err := s.Connect(context.Background()); err != nil {
		t.Fatalf("keyboard-interactive Connect failed: %v", err)
	}

	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	fwdPort := loopbackPort(t)
	stopFwd, err := StartForward(context.Background(), client, ForwardSpec{
		Direction:  ForwardLocal,
		ListenHost: "127.0.0.1",
		ListenPort: fwdPort,
		TargetHost: "127.0.0.1",
		TargetPort: targetLn.Addr().(*net.TCPAddr).Port,
	})
	if err != nil {
		t.Fatalf("StartForward: %v", err)
	}
	defer stopFwd()

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(fwdPort)), time.Second)
	if err != nil {
		t.Fatalf("dial forward: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte("half-close-payload")); err != nil {
		t.Fatal(err)
	}
	if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("ReadAll after CloseWrite: %v", err)
	}
	want := []byte("REPLY:half-close-payload")
	if !bytes.Equal(got, want) {
		t.Fatalf("half-close response = %q, want %q", got, want)
	}
}

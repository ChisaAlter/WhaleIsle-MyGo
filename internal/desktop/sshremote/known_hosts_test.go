package sshremote

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testHostKey(t *testing.T) (ssh.Signer, ssh.PublicKey) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signer, signer.PublicKey()
}

func TestKnownHostsPath(t *testing.T) {
	got := KnownHostsPath("userdata")
	want := filepath.Join("userdata", "ssh", "known_hosts")
	if got != want {
		t.Fatalf("path=%q want=%q", got, want)
	}
}

func TestHostKeyCallbackTOFU(t *testing.T) {
	dir := t.TempDir()
	path := KnownHostsPath(dir)
	_, pub := testHostKey(t)

	cb, err := NewHostKeyCallback(HostKeyOptions{Path: path, TrustOnFirstUse: true})
	if err != nil {
		t.Fatal(err)
	}
	addr := "example.test:2222"
	remote := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 2222}
	if err := cb(addr, remote, pub); err != nil {
		t.Fatalf("first connect tofu: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || !strings.Contains(string(data), pub.Type()) {
		t.Fatalf("known_hosts not updated: %q", data)
	}
	// Second connect with same key succeeds.
	if err := cb(addr, remote, pub); err != nil {
		t.Fatalf("pinned reconnect: %v", err)
	}
}

func TestHostKeyCallbackUnknownHostKey(t *testing.T) {
	dir := t.TempDir()
	path := KnownHostsPath(dir)
	_, pub := testHostKey(t)

	cb, err := NewHostKeyCallback(HostKeyOptions{Path: path, TrustOnFirstUse: false})
	if err != nil {
		t.Fatal(err)
	}
	addr := "prompt.test:22"
	remote := &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 22}
	err = cb(addr, remote, pub)
	var unk *UnknownHostKey
	if !errors.As(err, &unk) {
		t.Fatalf("got %v (%T), want *UnknownHostKey", err, err)
	}
	if unk.Address != addr || unk.KeyType != pub.Type() || unk.Fingerprint == "" {
		t.Fatalf("%+v", unk)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if len(data) != 0 {
		t.Fatalf("should not auto-add without TOFU: %q", data)
	}

	if err := AcceptHostKey(path, addr, pub); err != nil {
		t.Fatal(err)
	}
	if err := cb(addr, remote, pub); err != nil {
		t.Fatalf("after AcceptHostKey: %v", err)
	}
}

func TestHostKeyCallbackMismatch(t *testing.T) {
	dir := t.TempDir()
	path := KnownHostsPath(dir)
	_, pub1 := testHostKey(t)
	_, pub2 := testHostKey(t)

	if err := AcceptHostKey(path, "mitm.test:22", pub1); err != nil {
		t.Fatal(err)
	}
	cb, err := NewHostKeyCallback(HostKeyOptions{Path: path, TrustOnFirstUse: true})
	if err != nil {
		t.Fatal(err)
	}
	remote := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}
	err = cb("mitm.test:22", remote, pub2)
	var changed *HostKeyChanged
	if !errors.As(err, &changed) {
		t.Fatalf("got %v (%T), want *HostKeyChanged", err, err)
	}
	if !strings.Contains(changed.Error(), "host key changed") {
		t.Fatalf("message=%q", changed.Error())
	}
	// TOFU must not overwrite on mismatch.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	blob2 := strings.TrimSpace(strings.SplitN(string(ssh.MarshalAuthorizedKey(pub2)), " ", 3)[1])
	if strings.Contains(string(data), blob2) {
		t.Fatalf("mismatch must not store new key: %q", data)
	}
}

func TestAcceptHostKeyAndEnsure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ssh", "known_hosts")
	_, pub := testHostKey(t)
	if err := AcceptHostKey(path, "host.example:2222", pub); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("empty known_hosts")
	}
}

func TestNewHostKeyCallbackRequiresPath(t *testing.T) {
	_, err := NewHostKeyCallback(HostKeyOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTunnelSessionUsesKnownHostsFile(t *testing.T) {
	s := NewTunnelSession()
	if !s.tofu {
		t.Fatal("default TOFU should be enabled")
	}
	dir := t.TempDir()
	path := KnownHostsPath(dir)
	s.SetKnownHostsFile(path)
	s.SetTrustOnFirstUse(false)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.knownHostsFile != path {
		t.Fatalf("path=%q", s.knownHostsFile)
	}
	if s.tofu {
		t.Fatal("TOFU should be disabled")
	}
}

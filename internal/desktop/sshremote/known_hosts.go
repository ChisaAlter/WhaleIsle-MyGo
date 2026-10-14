package sshremote

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// KnownHostsPath is userData/ssh/known_hosts (OpenSSH format).
func KnownHostsPath(userData string) string {
	return filepath.Join(userData, "ssh", "known_hosts")
}

// UnknownHostKey is returned when the remote host is not in known_hosts and
// TrustOnFirstUse is false. UI may prompt the user, then call AcceptHostKey.
type UnknownHostKey struct {
	Address     string // host:port as presented to the SSH callback
	KeyType     string
	Fingerprint string // SHA256:...
	Key         ssh.PublicKey
}

func (e *UnknownHostKey) Error() string {
	return fmt.Sprintf("ssh: unknown host key for %s (%s %s)", e.Address, e.KeyType, e.Fingerprint)
}

// HostKeyChanged is returned when the remote host key does not match
// known_hosts. Connection must not proceed.
type HostKeyChanged struct {
	Address     string
	KeyType     string
	Fingerprint string
}

func (e *HostKeyChanged) Error() string {
	return fmt.Sprintf("ssh: WARNING: host key changed for %s (remote presented %s %s); refusing connection",
		e.Address, e.KeyType, e.Fingerprint)
}

// HostKeyOptions configures host-key verification against a known_hosts file.
type HostKeyOptions struct {
	// Path is the OpenSSH known_hosts file (typically KnownHostsPath(userData)).
	Path string
	// TrustOnFirstUse, when true, auto-adds unknown host keys (TOFU).
	// When false, unknown hosts return *UnknownHostKey so UI can prompt.
	// TunnelSession.Connect defaults to TOFU until a prompt UI lands.
	TrustOnFirstUse bool
}

// EnsureKnownHostsFile creates the parent directory and an empty known_hosts
// file when missing.
func EnsureKnownHostsFile(path string) error {
	if path == "" {
		return errors.New("known_hosts path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

// AcceptHostKey appends an OpenSSH known_hosts line for address (host:port)
// and key. Safe for UI use after the user confirms an *UnknownHostKey.
func AcceptHostKey(path string, address string, key ssh.PublicKey) error {
	if key == nil {
		return errors.New("host key required")
	}
	if err := EnsureKnownHostsFile(path); err != nil {
		return err
	}
	line := knownhosts.Line([]string{address}, key)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, line)
	return err
}

// NewHostKeyCallback returns an ssh.HostKeyCallback backed by opts.Path.
//
// Behavior:
//   - known key match → accept
//   - known key mismatch → *HostKeyChanged (never connect)
//   - unknown host + TrustOnFirstUse → append and accept (TOFU)
//   - unknown host + !TrustOnFirstUse → *UnknownHostKey
func NewHostKeyCallback(opts HostKeyOptions) (ssh.HostKeyCallback, error) {
	if opts.Path == "" {
		return nil, errors.New("known_hosts path required")
	}
	if err := EnsureKnownHostsFile(opts.Path); err != nil {
		return nil, err
	}
	var mu sync.Mutex
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		mu.Lock()
		defer mu.Unlock()

		cb, err := knownhosts.New(opts.Path)
		if err != nil {
			return err
		}
		err = cb(hostname, remote, key)
		if err == nil {
			return nil
		}
		var ke *knownhosts.KeyError
		if !errors.As(err, &ke) {
			return err
		}
		fp := ssh.FingerprintSHA256(key)
		if len(ke.Want) > 0 {
			return &HostKeyChanged{
				Address:     hostname,
				KeyType:     key.Type(),
				Fingerprint: fp,
			}
		}
		if opts.TrustOnFirstUse {
			if err := AcceptHostKey(opts.Path, hostname, key); err != nil {
				return fmt.Errorf("ssh: tofu store host key: %w", err)
			}
			return nil
		}
		return &UnknownHostKey{
			Address:     hostname,
			KeyType:     key.Type(),
			Fingerprint: fp,
			Key:         key,
		}
	}, nil
}

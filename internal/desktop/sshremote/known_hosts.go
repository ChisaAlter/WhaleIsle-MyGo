package sshremote

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyMode mirrors dsh-remote's SSH host-key policy.
//
// AcceptNew records an unknown key once and rejects later changes. Verify
// rejects unknown keys until the caller explicitly accepts one. Off disables
// host-key verification (intended only for an explicit user choice).
type HostKeyMode string

const (
	HostKeyAcceptNew HostKeyMode = "accept-new"
	HostKeyVerify    HostKeyMode = "verify"
	HostKeyOff       HostKeyMode = "off"
)

// NormalizeHostKeyMode applies the product default and rejects unknown
// values. dsh-remote defaults to accept-new for a new connection.
func NormalizeHostKeyMode(mode string) HostKeyMode {
	switch HostKeyMode(strings.ToLower(strings.TrimSpace(mode))) {
	case HostKeyVerify:
		return HostKeyVerify
	case HostKeyOff:
		return HostKeyOff
	default:
		return HostKeyAcceptNew
	}
}

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
	// Mode is the explicit host-key policy. Empty keeps the legacy
	// TrustOnFirstUse behavior, with true mapping to accept-new and false to
	// verify. Mode takes precedence when supplied.
	Mode HostKeyMode
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
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("known_hosts path must not be a symlink")
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
	// UI retries after a transient connection failure are common. Avoid
	// appending duplicate lines while retaining every existing host entry.
	knownHostsMu.Lock()
	defer knownHostsMu.Unlock()
	var needsSeparator bool
	if data, err := os.ReadFile(path); err == nil {
		needsSeparator = len(data) > 0 && data[len(data)-1] != '\n'
		for _, existing := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(existing) == strings.TrimSpace(line) {
				return nil
			}
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if needsSeparator {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(f, line)
	return err
}

var knownHostsMu sync.Mutex

// ForgetHostKey removes entries whose host marker contains address. It is the
// native equivalent of dsh-remote's /remote-forget-key action and leaves
// unrelated hosts untouched.
func ForgetHostKey(path, address string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("known_hosts path required")
	}
	address = strings.TrimSpace(address)
	if address == "" {
		return errors.New("host address required")
	}
	knownHostsMu.Lock()
	defer knownHostsMu.Unlock()
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return errors.New("known_hosts path must not be a symlink")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	kept := lines[:0]
	changed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			kept = append(kept, line)
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 3 {
			matched := false
			for _, host := range strings.Split(fields[0], ",") {
				if host == address {
					matched = true
					break
				}
			}
			if matched {
				changed = true
				continue
			}
		}
		kept = append(kept, line)
	}
	if !changed {
		return nil
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)
	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.WriteString(strings.Join(kept, "\n")); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

// NewHostKeyCallback returns an ssh.HostKeyCallback backed by opts.Path.
//
// Behavior:
//   - known key match → accept
//   - known key mismatch → *HostKeyChanged (never connect)
//   - unknown host + TrustOnFirstUse → append and accept (TOFU)
//   - unknown host + !TrustOnFirstUse → *UnknownHostKey
func NewHostKeyCallback(opts HostKeyOptions) (ssh.HostKeyCallback, error) {
	mode := opts.Mode
	if mode == "" {
		if opts.TrustOnFirstUse {
			mode = HostKeyAcceptNew
		} else {
			mode = HostKeyVerify
		}
	}
	mode = NormalizeHostKeyMode(string(mode))
	if opts.Path == "" && mode != HostKeyOff {
		return nil, errors.New("known_hosts path required")
	}
	if mode != HostKeyOff {
		if err := EnsureKnownHostsFile(opts.Path); err != nil {
			return nil, err
		}
	}
	if mode == HostKeyOff {
		return func(string, net.Addr, ssh.PublicKey) error { return nil }, nil
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
		if mode == HostKeyAcceptNew {
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
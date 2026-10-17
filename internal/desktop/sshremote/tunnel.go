package sshremote

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"whaleisle/internal/paths"
)

// TunnelConfig is the Go-only SSH session settings (userData/ssh-tunnel.json).
type TunnelConfig struct {
	Name         string `json:"name,omitempty"`
	Host         string `json:"host"`
	User         string `json:"user"`
	Port         int    `json:"port"`
	IdentityFile string `json:"identityFile"`
	Password     string `json:"password,omitempty"`
	Passphrase   string `json:"passphrase,omitempty"`
	HostKeyMode  string `json:"hostKeyMode,omitempty"`
	UseAgent     bool   `json:"useAgent,omitempty"`
	// LocalForward is optional "localPort:remoteHost:remotePort".
	LocalForward   string `json:"localForward,omitempty"`
	ReverseForward string `json:"reverseForward,omitempty"`
}

// TunnelState is the live session status.
type TunnelState string

const (
	TunnelIdle       TunnelState = "idle"
	TunnelConnecting TunnelState = "connecting"
	TunnelConnected  TunnelState = "connected"
	TunnelError      TunnelState = "error"
)

// TunnelSession manages one SSH client connection.
type TunnelSession struct {
	mu             sync.Mutex
	cfg            TunnelConfig
	state          TunnelState
	err            string
	client         *ssh.Client
	cancel         context.CancelFunc
	knownHostsFile string // empty → KnownHostsPath(paths.UserDataDir())
	tofu           bool   // Trust-On-First-Use for unknown hosts (default true)
	hostKeyMode    HostKeyMode
	forwards       map[string]*activeForward
}

type activeForward struct {
	spec     ForwardSpec
	listener net.Listener
	cancel   context.CancelFunc
}

// NewTunnelSession returns an idle session. Unknown host keys are accepted
// and stored under userData/ssh/known_hosts (TOFU) until UI prompting lands;
// call SetTrustOnFirstUse(false) to receive *UnknownHostKey instead.
func NewTunnelSession() *TunnelSession {
	return &TunnelSession{state: TunnelIdle, tofu: true, hostKeyMode: HostKeyAcceptNew, forwards: make(map[string]*activeForward)}
}

// SetKnownHostsFile sets the OpenSSH known_hosts path used for host-key
// pinning. Empty resets to KnownHostsPath(paths.UserDataDir()).
func (t *TunnelSession) SetKnownHostsFile(path string) {
	t.mu.Lock()
	t.knownHostsFile = path
	t.mu.Unlock()
}

// SetTrustOnFirstUse toggles auto-accept of unknown host keys (TOFU).
// When false, Connect fails with *UnknownHostKey so UI can prompt and call
// AcceptHostKey before retrying.
func (t *TunnelSession) SetTrustOnFirstUse(enabled bool) {
	t.mu.Lock()
	t.tofu = enabled
	if enabled {
		t.hostKeyMode = HostKeyAcceptNew
	} else {
		t.hostKeyMode = HostKeyVerify
	}
	t.mu.Unlock()
}

// SetHostKeyMode selects accept-new, verify or off.
func (t *TunnelSession) SetHostKeyMode(mode HostKeyMode) {
	t.mu.Lock()
	t.hostKeyMode = NormalizeHostKeyMode(string(mode))
	t.tofu = t.hostKeyMode == HostKeyAcceptNew
	t.mu.Unlock()
}

// ConfigPath is userData/ssh-tunnel.json.
func ConfigPath(userData string) string {
	return filepath.Join(userData, "ssh-tunnel.json")
}

// LoadTunnelConfig reads ssh-tunnel.json; missing file yields defaults.
func LoadTunnelConfig(userData string) (TunnelConfig, error) {
	cfg := TunnelConfig{Port: 22, User: "root"}
	path := ConfigPath(userData)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.HostKeyMode != "" {
		cfg.HostKeyMode = string(NormalizeHostKeyMode(cfg.HostKeyMode))
	}
	return cfg, nil
}

// SaveTunnelConfig writes ssh-tunnel.json atomically.
func SaveTunnelConfig(userData string, cfg TunnelConfig) error {
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.HostKeyMode != "" {
		cfg.HostKeyMode = string(NormalizeHostKeyMode(cfg.HostKeyMode))
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := ConfigPath(userData)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".ssh-tunnel-*.json")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)
	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Snapshot returns current state fields.
func (t *TunnelSession) Snapshot() (TunnelConfig, TunnelState, string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cfg, t.state, t.err
}

// SetConfig updates the desired tunnel config (does not connect).
func (t *TunnelSession) SetConfig(cfg TunnelConfig) {
	t.mu.Lock()
	t.cfg = cfg
	t.mu.Unlock()
}

// Connect dials SSH with the public-key identity file.
func (t *TunnelSession) Connect(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	t.mu.Lock()
	if t.client != nil {
		t.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	cfg := t.cfg
	kh := t.knownHostsFile
	tofu := t.tofu
	t.state = TunnelConnecting
	t.err = ""
	t.mu.Unlock()

	if strings.TrimSpace(cfg.Host) == "" {
		return t.fail(fmt.Errorf("host required"))
	}
	if strings.TrimSpace(cfg.User) == "" {
		return t.fail(fmt.Errorf("user required"))
	}
	var auth []ssh.AuthMethod
	var agentConn net.Conn
	if strings.TrimSpace(cfg.IdentityFile) != "" {
		key, err := os.ReadFile(cfg.IdentityFile)
		if err != nil {
			return t.fail(err)
		}
		var signer ssh.Signer
		if strings.TrimSpace(cfg.Passphrase) != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return t.fail(fmt.Errorf("parse key: %w", err))
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if cfg.Password != "" {
		auth = append(auth, ssh.Password(cfg.Password))
	}
	if cfg.UseAgent {
		sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
		if sock == "" {
			return t.fail(fmt.Errorf("SSH_AUTH_SOCK is not set"))
		}
		var err error
		agentConn, err = net.DialTimeout("unix", sock, 2*time.Second)
		if err != nil {
			return t.fail(fmt.Errorf("ssh agent: %w", err))
		}
		auth = append(auth, ssh.PublicKeysCallback(agent.NewClient(agentConn).Signers))
	}
	if len(auth) == 0 {
		return t.fail(fmt.Errorf("identityFile, password or SSH agent required"))
	}
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	if strings.TrimSpace(kh) == "" {
		kh = KnownHostsPath(paths.UserDataDir())
	}
	mode := HostKeyMode(cfg.HostKeyMode)
	if mode == "" {
		mode = HostKeyAcceptNew
		if !tofu {
			mode = HostKeyVerify
		}
	}
	hostKeyCB, err := NewHostKeyCallback(HostKeyOptions{Path: kh, TrustOnFirstUse: tofu, Mode: mode})
	if err != nil {
		if agentConn != nil {
			_ = agentConn.Close()
		}
		return t.fail(err)
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCB,
		Timeout:         12 * time.Second,
	}
	dialer := net.Dialer{Timeout: 12 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if agentConn != nil {
			_ = agentConn.Close()
		}
		return t.fail(err)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if agentConn != nil {
		_ = agentConn.Close()
	}
	if err != nil {
		_ = conn.Close()
		return t.fail(err)
	}
	client := ssh.NewClient(c, chans, reqs)

	ctx, cancel := context.WithCancel(context.Background())
	t.mu.Lock()
	t.client = client
	t.cancel = cancel
	t.state = TunnelConnected
	t.err = ""
	t.mu.Unlock()

	go t.watch(ctx, client)
	if fwd := strings.TrimSpace(cfg.LocalForward); fwd != "" {
		go t.forward(ctx, client, fwd)
	}
	if fwd := strings.TrimSpace(cfg.ReverseForward); fwd != "" {
		if !strings.HasPrefix(strings.ToLower(fwd), "reverse:") {
			fwd = "reverse:" + fwd
		}
		go t.forward(ctx, client, fwd)
	}
	return nil
}

// Disconnect closes the SSH client.
func (t *TunnelSession) Disconnect() {
	t.mu.Lock()
	cancel := t.cancel
	client := t.client
	t.cancel = nil
	t.client = nil
	t.state = TunnelIdle
	t.err = ""
	t.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil {
		_ = client.Close()
	}
}

func (t *TunnelSession) fail(err error) error {
	t.mu.Lock()
	t.state = TunnelError
	t.err = err.Error()
	t.mu.Unlock()
	return err
}

func (t *TunnelSession) watch(ctx context.Context, client *ssh.Client) {
	<-ctx.Done()
	_ = client.Close()
	t.mu.Lock()
	if t.client == client {
		t.client = nil
		t.cancel = nil
		if t.state == TunnelConnected {
			t.state = TunnelIdle
		}
	}
	t.mu.Unlock()
}

func (t *TunnelSession) forward(ctx context.Context, client *ssh.Client, spec string) {
	parsed, err := ParseForwardSpec(spec)
	if err != nil {
		t.mu.Lock()
		t.err = "forward spec: " + err.Error()
		t.mu.Unlock()
		return
	}
	stop, err := StartForward(ctx, client, parsed)
	if err != nil {
		t.mu.Lock()
		t.err = "forward listen: " + err.Error()
		t.mu.Unlock()
		return
	}
	go func() {
		<-ctx.Done()
		stop()
	}()
}

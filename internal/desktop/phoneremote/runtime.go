package phoneremote

import (
	"bufio"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// RuntimeState describes observed daemon state, never configuration readiness.
// PairingURL is ephemeral sensitive UI data: do not log or persist this struct.
type RuntimeState struct {
	Status                    string
	Listening, RelayConnected bool
	PairingURL                string
	ExpiresAt                 time.Time
	Message                   string
}
type Resources struct{ Node, Module, Home string }

// Child is the private transport boundary; tests supply pipes, never a daemon.
type Child interface {
	Input() io.Writer
	Output() io.Reader
	Wait() error
	Kill()
}

//go:embed bridge.cjs
var bridgeScript string

type Runtime struct {
	op          sync.Mutex
	mu          sync.Mutex
	state       RuntimeState
	child       Child
	done        chan struct{}
	spawn       func(Resources) (Child, error)
	resources   func(string, string) (Resources, error)
	stopTimeout time.Duration
}

func NewRuntime() *Runtime {
	return &Runtime{state: RuntimeState{Status: "offline"}, spawn: spawnBridge, resources: ResolveResources, stopTimeout: 8 * time.Second}
}

// ResolveResources accepts only the installed component or packaged vendor layout.
// The component is a complete original Electron source runtime, including its
// built vendor tree. We neither download assets nor guess a system executable.
func ResolveResources(userData, vendor string) (Resources, error) {
	root := filepath.Join(vendor, "dshd-remote")
	component := filepath.Join(userData, "components", "dshd-remote")
	if b, e := os.ReadFile(filepath.Join(component, "installed.json")); e == nil {
		var installed struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(b, &installed) != nil || installed.Version == "" || filepath.Base(installed.Version) != installed.Version || installed.Version == "." || installed.Version == ".." {
			return Resources{}, errors.New("invalid dshd-remote installed version")
		}
		root = filepath.Join(component, "versions", installed.Version)
	}
	root, absErr := filepath.Abs(root)
	if absErr != nil {
		return Resources{}, errors.New("invalid runtime root")
	}
	required := []string{"src/main/dshd-remote.js", "src/main/dshd-daemon-runner.mjs", "src/main/dshd-daemon-hooks.mjs", "src/main/mobile-web-server.js", "src/main/mobile-web.js", "mobile/web/index.html", "src/main/dshd-git-tunnel.js", "src/main/dshd-git-dispatch.js", "src/shared/official-deepseek-env.js", "src/shared/lan.js", "src/shared/offer.js", "src/shared/dshd-host-tunnel.js", "src/shared/dshd-mux-sse.js", "vendor/chisacode-remote/node_modules/@chisacode/server/dist/server/server/exports.js", "vendor/chisacode-remote/node_modules/@chisacode/server/dist/server/server/pairing-offer.js", "vendor/chisacode-remote/node_modules/@chisacode/server/dist/server/server/relay-device-credential-store.js"}
	for _, relative := range required {
		if st, e := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); e != nil || st.IsDir() {
			return Resources{}, errors.New("missing runtime resource: " + relative)
		}
	}
	node := ""
	for _, p := range []string{filepath.Join(root, "node.exe"), filepath.Join(root, "node", "node.exe"), filepath.Join(root, "node", "bin", "node")} {
		if st, e := os.Stat(p); e == nil && !st.IsDir() {
			node = p
			break
		}
	}
	if node == "" {
		return Resources{}, errors.New("missing runtime resource: bundled node executable")
	}
	return Resources{Node: node, Module: filepath.Join(root, "src", "main", "dshd-remote.js"), Home: filepath.Join(userData, "dsh-home")}, nil
}

func (r *Runtime) Start(userData, vendor string, config map[string]any) {
	r.op.Lock()
	defer r.op.Unlock()
	r.stopLocked()
	if enabled, _ := config["remoteEnabled"].(bool); !enabled {
		r.set(RuntimeState{Status: "disabled"})
		return
	}
	resources, err := r.resources(userData, vendor)
	if err != nil {
		r.set(RuntimeState{Status: "missing", Message: err.Error()})
		return
	}
	child, err := r.spawn(resources)
	if err != nil {
		r.set(RuntimeState{Status: "offline", Message: "remote bridge could not start"})
		return
	}
	done := make(chan struct{})
	r.mu.Lock()
	r.child = child
	r.done = done
	r.state = RuntimeState{Status: "starting"}
	r.mu.Unlock()
	go func() {
		scanner := bufio.NewScanner(child.Output())
		scanner.Buffer(make([]byte, 4096), 256*1024)
		for scanner.Scan() {
			r.consume(child, scanner.Bytes())
		}
		child.Kill()
		_ = child.Wait()
		r.mu.Lock()
		if r.child == child {
			message := r.state.Message
			if message == "" {
				message = "remote bridge stopped"
			}
			r.state = RuntimeState{Status: "offline", Message: message}
		}
		r.mu.Unlock()
		close(done)
	}()
	if err := writeControl(child, map[string]any{"action": "start", "config": config}); err != nil {
		r.stopLocked()
		r.set(RuntimeState{Status: "offline", Message: "remote control unavailable"})
	}
}
func (r *Runtime) Refresh(config map[string]any) {
	r.op.Lock()
	defer r.op.Unlock()
	r.mu.Lock()
	child := r.child
	r.mu.Unlock()
	if child != nil {
		if writeControl(child, map[string]any{"action": "refresh", "config": config}) != nil {
			r.stopLocked()
		}
	}
}
func (r *Runtime) Stop() { r.op.Lock(); defer r.op.Unlock(); r.stopLocked() }
func (r *Runtime) stopLocked() {
	r.mu.Lock()
	child, done := r.child, r.done
	r.child = nil
	r.done = nil
	r.state = RuntimeState{Status: "offline"}
	r.mu.Unlock()
	if child == nil {
		return
	}
	_ = writeControl(child, map[string]any{"action": "stop"})
	select {
	case <-done:
	case <-time.After(r.stopTimeout):
		child.Kill()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}
func (r *Runtime) set(s RuntimeState) { r.mu.Lock(); r.state = s; r.mu.Unlock() }
func (r *Runtime) Snapshot() RuntimeState {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.state
	if !s.ExpiresAt.IsZero() && !time.Now().Before(s.ExpiresAt) {
		s.PairingURL = ""
		s.Status = "expired"
	}
	if !s.Listening || !s.RelayConnected {
		s.PairingURL = ""
	}
	return s
}
func (r *Runtime) consume(child Child, b []byte) {
	var event struct {
		Kind           string `json:"kind"`
		Listening      bool   `json:"listening"`
		RelayConnected bool   `json:"relayConnected"`
		URL            string `json:"url"`
		Code           string `json:"code"`
	}
	if json.Unmarshal(b, &event) != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.child != child {
		return
	}
	if event.Kind == "error" {
		message := "remote startup or pairing failed; check installed runtime and retry"
		if event.Code == "runtime-load" {
			message = "original remote module or dependency could not load"
		}
		if event.Code == "snapshot" {
			message = "remote state could not be read"
		}
		r.state = RuntimeState{Status: "offline", Message: message}
		return
	}
	if event.Kind != "snapshot" {
		return
	}
	s := RuntimeState{Status: "offline", Listening: event.Listening, RelayConnected: event.RelayConnected}
	if s.Listening {
		s.Status = "relay-offline"
	}
	expires, ok := offerExpiry(event.URL)
	if ok {
		s.ExpiresAt = expires
	}
	if s.Listening && s.RelayConnected {
		s.Status = "pairing-unavailable"
		if ok {
			if time.Now().Before(expires) {
				s.Status = "ready"
				s.PairingURL = event.URL
			} else {
				s.Status = "expired"
			}
		}
	}
	r.state = s
}
func offerExpiry(raw string) (time.Time, bool) {
	if len(raw) > 128*1024 {
		return time.Time{}, false
	}
	u, e := url.Parse(raw)
	if e != nil || u == nil || u.User != nil || u.RawQuery != "" || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || !strings.HasPrefix(u.Fragment, "offer=") {
		return time.Time{}, false
	}
	b, e := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(u.Fragment, "offer="))
	if e != nil {
		return time.Time{}, false
	}
	var offer struct {
		V        int    `json:"v"`
		ServerID string `json:"serverId"`
		Key      string `json:"daemonPublicKeyB64"`
		Auth     struct {
			Version int    `json:"version"`
			Token   string `json:"pairingToken"`
			Expires int64  `json:"expiresAtMs"`
		} `json:"authBootstrap"`
	}
	if json.Unmarshal(b, &offer) != nil || offer.V != 2 || offer.ServerID == "" || offer.Key == "" || offer.Auth.Version != 1 || offer.Auth.Token == "" || offer.Auth.Expires <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(offer.Auth.Expires), true
}
func writeControl(c Child, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = c.Input().Write(append(b, '\n'))
	return e
}

type execChild struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out io.ReadCloser
}

func (c *execChild) Input() io.Writer  { return c.in }
func (c *execChild) Output() io.Reader { return c.out }
func (c *execChild) Wait() error       { return c.cmd.Wait() }
func (c *execChild) Kill()             { _ = c.in.Close(); _ = c.cmd.Process.Kill() }
func spawnBridge(r Resources) (Child, error) {
	cmd := exec.Command(r.Node, "-e", bridgeScript, r.Module, r.Home)
	cmd.Dir = filepath.Dir(r.Module)
	cmd.Stderr = io.Discard
	in, e := cmd.StdinPipe()
	if e != nil {
		return nil, e
	}
	out, e := cmd.StdoutPipe()
	if e != nil {
		in.Close()
		return nil, e
	}
	if e = cmd.Start(); e != nil {
		in.Close()
		out.Close()
		return nil, e
	}
	return &execChild{cmd, in, out}, nil
}

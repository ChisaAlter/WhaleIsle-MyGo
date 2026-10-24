package phoneremote

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
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
	PairingImage              image.Image
	ExpiresAt                 time.Time
	Message                   string
	Devices                   []Device
	BindAddresses             []string
	DevicesLoaded             bool
	DeviceAction              DeviceActionState
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
	op           sync.Mutex
	writeMu      sync.Mutex
	authorizeGit func(string, string) bool
	reviewGit    GitReviewer
	reviewCancel context.CancelFunc
	mu           sync.Mutex
	state        RuntimeState
	// lanMode is the transport the daemon was last told to run. Native always
	// builds urls[].pairingUrl and only flags relayDown in relay mode
	// (reference/electron/src/main/remote.js:344-360), so a listening LAN
	// daemon is ready without a relay connection.
	lanMode          bool
	child            Child
	done             chan struct{}
	spawn            func(Resources) (Child, error)
	resources        func(string, string) (Resources, error)
	stopTimeout      time.Duration
	deviceAction     DeviceActionState
	deviceSequence   uint64
	deviceActionDone chan struct{}
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
	r.lanMode = remoteModeLAN(config)
	r.mu.Unlock()
	go func() {
		scanner := bufio.NewScanner(child.Output())
		scanner.Buffer(make([]byte, 4096), 1024*1024)
		for scanner.Scan() {
			r.consume(child, scanner.Bytes())
		}
		child.Kill()
		_ = child.Wait()
		r.mu.Lock()
		if r.child == child {
			r.finishDeviceActionLocked("stopped", false)
			message := r.state.Message
			if message == "" {
				message = "remote bridge stopped"
			}
			r.state = RuntimeState{Status: "offline", Message: message}
		}
		r.mu.Unlock()
		close(done)
	}()
	if err := r.sendControl(child, map[string]any{"action": "start", "config": config}); err != nil {
		r.stopLocked()
		r.set(RuntimeState{Status: "offline", Message: "remote control unavailable"})
	}
}
func (r *Runtime) Refresh(config map[string]any) {
	r.op.Lock()
	defer r.op.Unlock()
	r.mu.Lock()
	child := r.child
	// A transport change takes effect on the next snapshot, never mid-event.
	r.lanMode = remoteModeLAN(config)
	r.mu.Unlock()
	if child != nil {
		if r.sendControl(child, map[string]any{"action": "refresh", "config": config}) != nil {
			r.stopLocked()
		}
	}
}
func (r *Runtime) Stop() { r.op.Lock(); defer r.op.Unlock(); r.stopLocked() }
func (r *Runtime) stopLocked() {
	r.mu.Lock()
	child, done := r.child, r.done
	r.finishDeviceActionLocked("stopped", false)
	r.child = nil
	r.done = nil
	r.state = RuntimeState{Status: "offline"}
	r.mu.Unlock()
	if child == nil {
		return
	}
	_ = r.sendControl(child, map[string]any{"action": "stop"})
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

// remoteModeLAN reports whether the daemon runs the LAN transport. Only "lan"
// is a LAN gateway; every other value is relay, matching config normalization
// (internal/config/config.go, native config.js:235).
func remoteModeLAN(config map[string]any) bool {
	mode, _ := config["remoteMode"].(string)
	return mode == "lan"
}

// relayRequired is the readiness rule for the current transport: a relay
// gateway is only ready once the relay handshake is up, a LAN gateway never
// waits for one (native remote.js:344-360 only marks relayDown in relay mode).
func relayRequired(lan bool) bool { return !lan }

func (r *Runtime) Snapshot() RuntimeState {
	r.mu.Lock()
	defer r.mu.Unlock()
	lanOnly := r.lanMode
	s := r.state
	if !s.ExpiresAt.IsZero() && !time.Now().Before(s.ExpiresAt) {
		s.PairingURL = ""
		s.PairingImage = nil
		s.Status = "expired"
	}
	if s.Status != "ready" || !s.Listening || (!s.RelayConnected && relayRequired(lanOnly)) {
		s.PairingURL = ""
		s.PairingImage = nil
	}
	r.state = s
	s.Devices = append([]Device(nil), s.Devices...)
	if s.BindAddresses != nil {
		s.BindAddresses = append([]string{}, s.BindAddresses...)
	}
	s.DeviceAction = r.deviceAction
	if s.DeviceAction.Busy && s.DeviceAction.Operation != "rename" {
		s.PairingURL = ""
		s.PairingImage = nil
		s.Status = "updating"
	}
	return s
}
func (r *Runtime) consume(child Child, b []byte) {
	var event struct {
		Kind           string   `json:"kind"`
		ID             string   `json:"id"`
		Root           string   `json:"root"`
		Action         string   `json:"action"`
		Listening      bool     `json:"listening"`
		RelayConnected bool     `json:"relayConnected"`
		URL            string   `json:"url"`
		QRPNG          string   `json:"qrPNG"`
		Code           string   `json:"code"`
		Operation      string   `json:"operation"`
		Applied        bool     `json:"applied"`
		Devices        []Device `json:"devices"`
		Addresses      []string `json:"addresses"`
		DevicesLoaded  bool     `json:"devicesLoaded"`
	}
	if json.Unmarshal(b, &event) != nil {
		return
	}
	if event.Kind == "git-review-request" {
		go r.reviewGitRequest(child, append([]byte(nil), b...))
		return
	}
	if event.Kind == "git-generate-draft" {
		go r.generateGitDraft(child, append([]byte(nil), b...))
		return
	}
	if event.Kind == "git-authorize" {
		r.mu.Lock()
		current := r.child == child
		authorize := r.authorizeGit
		r.mu.Unlock()
		allowed := current && authorize != nil && len(event.ID) < 64 && authorize(event.Root, event.Action)
		_ = r.sendControl(child, map[string]any{"action": "git-authorize-result", "id": event.ID, "allowed": allowed})
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.child != child {
		return
	}
	if event.Kind == "device-action-result" {
		if r.deviceAction.Busy && event.ID == r.deviceAction.ID && event.Operation == r.deviceAction.Operation {
			r.finishDeviceActionLocked(event.Code, event.Applied)
		}
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
	lanOnly := r.lanMode
	s := RuntimeState{Status: "offline", Listening: event.Listening, RelayConnected: event.RelayConnected,
		Devices: event.Devices, DevicesLoaded: event.DevicesLoaded, BindAddresses: event.Addresses}
	if s.Listening {
		s.Status = "relay-offline"
		if lanOnly {
			// A LAN gateway has no relay to wait for.
			s.Status = "pairing-unavailable"
		}
	}
	expires, ok := offerExpiry(event.URL)
	if ok {
		s.ExpiresAt = expires
	}
	if s.Listening && (lanOnly || s.RelayConnected) {
		s.Status = "pairing-unavailable"
		if ok {
			if time.Now().Before(expires) {
				s.Status = "ready"
				s.PairingURL = event.URL
				if event.URL == r.state.PairingURL && event.QRPNG != "" && r.state.PairingImage != nil {
					s.PairingImage = r.state.PairingImage
				} else if len(event.QRPNG) <= 128*1024 {
					if data, err := base64.StdEncoding.DecodeString(event.QRPNG); err == nil {
						if size, err := png.DecodeConfig(bytes.NewReader(data)); err == nil && size.Width > 0 && size.Width <= 1024 && size.Height == size.Width {
							s.PairingImage, _ = png.Decode(bytes.NewReader(data))
						}
					}
				}
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
	cmd    *exec.Cmd
	in     io.WriteCloser
	out    io.ReadCloser
	script string
}

func (c *execChild) Input() io.Writer  { return c.in }
func (c *execChild) Output() io.Reader { return c.out }
func (c *execChild) Wait() error       { defer os.Remove(c.script); return c.cmd.Wait() }
func (c *execChild) Kill()             { _ = c.in.Close(); _ = c.cmd.Process.Kill() }
func spawnBridge(r Resources) (Child, error) {
	if e := os.MkdirAll(r.Home, 0700); e != nil {
		return nil, e
	}
	script, e := os.CreateTemp(r.Home, ".mygo-phone-bridge-*.cjs")
	if e != nil {
		return nil, e
	}
	scriptName := script.Name()
	if _, e = script.WriteString(composedBridgeScript()); e != nil {
		script.Close()
		os.Remove(scriptName)
		return nil, e
	}
	if e = script.Close(); e != nil {
		os.Remove(scriptName)
		return nil, e
	}
	started := false
	defer func() {
		if !started {
			os.Remove(scriptName)
		}
	}()
	cmd := exec.Command(r.Node, "-e", "const file=process.argv[1];process.argv.splice(1,1);require(file)", scriptName, r.Module, r.Home)
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
	started = true
	return &execChild{cmd: cmd, in: in, out: out, script: scriptName}, nil
}

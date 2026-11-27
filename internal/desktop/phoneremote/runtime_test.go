package phoneremote

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixtureChild struct {
	input  bytes.Buffer
	output *io.PipeReader
	writer *io.PipeWriter
	done   chan struct{}
	once   sync.Once
}

func (f *fixtureChild) Input() io.Writer  { return &f.input }
func (f *fixtureChild) Output() io.Reader { return f.output }
func (f *fixtureChild) Wait() error       { <-f.done; return nil }
func (f *fixtureChild) Kill()             { f.once.Do(func() { close(f.done); f.writer.Close() }) }

func TestRuntimeFixture(t *testing.T) {
	r, w := io.Pipe()
	child := &fixtureChild{output: r, writer: w, done: make(chan struct{})}
	m := NewRuntime()
	m.stopTimeout = time.Millisecond
	m.spawn = func(Resources) (Child, error) { return child, nil }
	m.resources = func(string, string) (Resources, error) { return Resources{}, nil }
	m.Start("fixture", "fixture", map[string]any{"remoteEnabled": true})
	defer m.Stop()
	expires := time.Now().Add(time.Minute).UnixMilli()
	offer := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"v":2,"serverId":"fixture","daemonPublicKeyB64":"fictional","authBootstrap":{"version":1,"pairingToken":"fictional-only","expiresAtMs":%d},"relay":{"endpoint":"relay.example.test:443"}}`, expires)))
	var qr bytes.Buffer
	if err := png.Encode(&qr, image.NewRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	go fmt.Fprintf(w, `{"kind":"snapshot","listening":true,"relayConnected":true,"url":"https://app.example.test/#offer=%s","qrPNG":"%s"}`+"\n", offer, base64.StdEncoding.EncodeToString(qr.Bytes()))
	deadline := time.Now().Add(time.Second)
	for m.Snapshot().Status != "ready" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	s := m.Snapshot()
	if s.Status != "ready" || s.PairingURL == "" || s.PairingImage == nil {
		t.Fatal("valid offer not exposed")
	}
	m.mu.Lock()
	m.state.ExpiresAt = time.Now().Add(-time.Second)
	m.mu.Unlock()
	if s = m.Snapshot(); s.Status != "expired" || s.PairingURL != "" || s.PairingImage != nil {
		t.Fatal("expired offer leaked")
	}
	m.mu.Lock()
	m.state = RuntimeState{Status: "relay-offline", Listening: true, PairingURL: "fictional-sensitive"}
	m.mu.Unlock()
	if m.Snapshot().PairingURL != "" {
		t.Fatal("disconnected relay exposed offer")
	}
	m.consume(child, []byte(`{"kind":"error","code":"operation","error":"fictional-secret-token"}`))
	if bytes.Contains([]byte(m.Snapshot().Message), []byte("fictional-secret-token")) {
		t.Fatal("raw error leaked")
	}
	m.Stop()
	if s = m.Snapshot(); s.PairingURL != "" || s.Listening || s.PairingImage != nil {
		t.Fatal("stop retained offer")
	}
	if !bytes.Contains(child.input.Bytes(), []byte(`"action":"stop"`)) {
		t.Fatal("missing stop control")
	}
	m.Start("fixture", "fixture", map[string]any{"remoteEnabled": false})
	if _, err := ResolveResources(t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("missing runtime accepted")
	}
	if m.Snapshot().Status != "disabled" {
		t.Fatal("disabled state")
	}
}

func TestRefreshDisableWaitsForOwnedBridgeExit(t *testing.T) {
	r, w := io.Pipe()
	child := &fixtureChild{output: r, writer: w, done: make(chan struct{})}
	m := NewRuntime()
	m.stopTimeout = time.Millisecond
	m.spawn = func(Resources) (Child, error) { return child, nil }
	m.resources = func(string, string) (Resources, error) { return Resources{}, nil }
	m.Start("fixture", "fixture", map[string]any{"remoteEnabled": true})
	m.mu.Lock()
	m.state = RuntimeState{Status: "ready", Listening: true}
	m.mu.Unlock()
	m.Refresh(map[string]any{"remoteEnabled": false})
	if s := m.Snapshot(); s.Status != "disabled" || s.Listening || s.PairingURL != "" {
		t.Fatalf("disabled configuration did not settle: %+v", s)
	}
	select {
	case <-child.done:
	default:
		t.Fatal("disabled state preceded child exit")
	}
	if !bytes.Contains(child.input.Bytes(), []byte(`"action":"stop"`)) {
		t.Fatal("disable did not tell the owned daemon to stop")
	}
}

func TestHelperBridgeLeaker(t *testing.T) {
	mode := os.Getenv("WHALEISLE_TEST_BRIDGE_LEAKER")
	if mode == "" {
		return
	}
	pidFile := os.Getenv("WHALEISLE_TEST_BRIDGE_GC_PID")
	if mode == "grandchild" {
		// Inherit stdout and hold it open while sleeping.
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	if mode == "parent" {
		exe, err := os.Executable()
		if err != nil {
			os.Exit(2)
		}
		gc := exec.Command(exe, "-test.run=^TestHelperBridgeLeaker$")
		gc.Env = append(os.Environ(), "WHALEISLE_TEST_BRIDGE_LEAKER=grandchild")
		gc.Stdout = os.Stdout
		if err := gc.Start(); err != nil {
			os.Exit(3)
		}
		if pidFile != "" {
			_ = os.WriteFile(pidFile, []byte(strconv.Itoa(gc.Process.Pid)), 0o600)
		}
		// Keep parent alive ignoring stop until killed.
		time.Sleep(time.Minute)
		os.Exit(0)
	}
}

func TestRuntimeStopKillsGrandchildInheritingStdout(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "gc.pid")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WHALEISLE_TEST_BRIDGE_LEAKER", "parent")
	t.Setenv("WHALEISLE_TEST_BRIDGE_GC_PID", pidFile)

	m := NewRuntime()
	m.stopTimeout = 150 * time.Millisecond
	m.resources = func(string, string) (Resources, error) { return Resources{}, nil }
	m.spawn = func(Resources) (Child, error) {
		cmd := exec.Command(exe, "-test.run=^TestHelperBridgeLeaker$")
		prepareBridgeCmd(cmd)
		in, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		outR, outW, err := os.Pipe()
		if err != nil {
			_ = in.Close()
			return nil, err
		}
		cmd.Stdout = outW
		if err := cmd.Start(); err != nil {
			_ = in.Close()
			_ = outR.Close()
			_ = outW.Close()
			return nil, err
		}
		_ = outW.Close()
		job := bindBridgeJob(cmd)
		return newExecChild(cmd, job, in, outR, ""), nil
	}

	m.Start("fixture", "fixture", map[string]any{"remoteEnabled": true})

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(pidFile); err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	rawPID, err := os.ReadFile(pidFile)
	if err != nil {
		m.Stop()
		t.Fatalf("grandchild pid file not written: %v", err)
	}
	gcPID, err := strconv.Atoi(strings.TrimSpace(string(rawPID)))
	if err != nil || gcPID <= 0 {
		m.Stop()
		t.Fatalf("invalid grandchild pid %q", rawPID)
	}

	startStop := time.Now()
	done := m.Stop()
	took := time.Since(startStop)
	if done != nil {
		t.Fatalf("Stop returned non-nil pending channel (status=%q)", m.Snapshot().Status)
	}
	if took > m.stopTimeout+2*time.Second {
		t.Fatalf("Stop took %s, exceeded budget", took)
	}
	if st := m.Snapshot(); st.Status == "stop-failed" {
		t.Fatalf("Stop entered stop-failed: %+v", st)
	}

	// Verify grandchild was terminated along with the bridge tree.
	gcDeadline := time.Now().Add(3 * time.Second)
	for processAlive(gcPID) && time.Now().Before(gcDeadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if processAlive(gcPID) {
		if p, err := os.FindProcess(gcPID); err == nil {
			_ = p.Kill()
		}
		t.Fatalf("grandchild PID %d survived Runtime.Stop()", gcPID)
	}
}

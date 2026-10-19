package phoneremote

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFunctionalFullOwnedPhoneRuntime(t *testing.T) {
	root := os.Getenv("WHALEISLE_PHONE_ACCEPTANCE_ROOT")
	if root == "" {
		t.Skip("requires fresh own root and explicit original resources/tool dependencies/core runtime")
	}
	if os.Getenv("WHALEISLE_PHONE_SANITIZED_ENV") != "1" {
		t.Fatal("requires explicitly sanitized child process environment")
	}
	node, resources, tools, core := os.Getenv("WHALEISLE_REMOTE_NODE"), os.Getenv("WHALEISLE_PHONE_RESOURCES"), os.Getenv("WHALEISLE_PHONE_TOOLS"), os.Getenv("WHALEISLE_PHONE_CORE")
	b, err := os.ReadFile(node)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != "ba4e6d110e8c1592a1ecd390f6b05f3da124b13871a5be62b341a07a853c6c32" {
		t.Fatal("unknown Node binary")
	}
	profile := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "TEMP", "TMP"} {
		t.Setenv(key, profile)
	}
	processLog := filepath.Join(profile, "owned-processes.jsonl")
	defer func() {
		for _, file := range []string{"owned-processes.jsonl", "local-runtime-stderr.log"} {
			if data, err := os.ReadFile(filepath.Join(profile, file)); err == nil {
				_ = os.WriteFile(filepath.Join(root, file), data, 0600)
			}
		}
	}()
	preload := filepath.Join(profile, "owned-processes.cjs")
	if err = os.WriteFile(preload, []byte(ownedPhoneProcessPreload), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WHALEISLE_PHONE_PROCESS_RECORD", processLog)
	t.Setenv("NODE_OPTIONS", "--require="+preload)
	_, source, _, _ := runtime.Caller(0)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, filepath.Join(filepath.Dir(source), "full-runtime-loopback.cjs"), root, resources, tools, core, "serve")
	cmd.Dir = profile
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	errFile, err := os.Create(filepath.Join(profile, "local-runtime-stderr.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer errFile.Close()
	cmd.Stderr = errFile
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	helperPID := cmd.Process.Pid
	eventLog, err := os.Create(filepath.Join(root, "runtime-events.jsonl"))
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	defer eventLog.Close()
	events := make(chan map[string]json.RawMessage, 32)
	eventsDone := make(chan struct{})
	go func() {
		defer close(eventsDone)
		defer close(events)
		scanner := bufio.NewScanner(out)
		scanner.Buffer(make([]byte, 4096), 2<<20)
		for scanner.Scan() {
			_, _ = eventLog.Write(append(append([]byte(nil), scanner.Bytes()...), '\n'))
			var v map[string]json.RawMessage
			if json.Unmarshal(scanner.Bytes(), &v) == nil {
				events <- v
			}
		}
	}()
	var helperWaited bool
	defer func() {
		if !helperWaited {
			_, _ = io.WriteString(in, "{\"action\":\"stop\"}\n")
			in.Close()
			_ = cmd.Wait()
		}
		<-eventsDone
		if cmd.ProcessState != nil {
			data, _ := json.Marshal(map[string]any{"pid": helperPID, "exitCode": cmd.ProcessState.ExitCode(), "exited": cmd.ProcessState.Exited()})
			_ = os.WriteFile(filepath.Join(root, "helper-exit.json"), data, 0600)
		}
	}()
	wait := func(kind string) map[string]json.RawMessage {
		t.Helper()
		for {
			select {
			case v, ok := <-events:
				if !ok {
					t.Fatal("owned helper exited before " + kind)
				}
				var got string
				_ = json.Unmarshal(v["kind"], &got)
				if got == "error" {
					var msg string
					_ = json.Unmarshal(v["message"], &msg)
					t.Fatal("owned helper: " + msg)
				}
				if got == "stage" {
					var section string
					_ = json.Unmarshal(v["section"], &section)
					t.Log("private stage:", section)
				}
				if got == kind {
					return v
				}
			case <-ctx.Done():
				t.Fatal("owned helper timeout waiting " + kind)
			}
		}
	}
	ready := wait("ready")
	var vendor, endpoint string
	_ = json.Unmarshal(ready["vendor"], &vendor)
	_ = json.Unmarshal(ready["endpoint"], &endpoint)
	if host, _, err := net.SplitHostPort(endpoint); err != nil || host != "127.0.0.1" {
		t.Fatal("relay not loopback")
	}
	userData := filepath.Join(profile, "data")
	resolved, err := ResolveResources(userData, vendor)
	if err != nil {
		t.Fatal("product readiness gate: ", err)
	}
	client := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	checkHealth := func(url string) {
		t.Helper()
		res, err := client.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var data map[string]any
		if err = json.NewDecoder(res.Body).Decode(&data); err != nil || res.StatusCode != 200 || data["status"] != "ok" {
			t.Fatal("health failed", res.StatusCode, err)
		}
	}
	checkHealth("http://" + endpoint + "/health")
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	m := NewRuntime()
	defer m.Stop()
	var bridge *execChild
	m.spawn = func(r Resources) (Child, error) {
		c, err := spawnBridge(r)
		if err == nil {
			bridge = c.(*execChild)
		}
		return c, err
	}
	m.Start(userData, vendor, map[string]any{"remoteEnabled": true, "remoteMode": "away", "remoteBindAddress": "127.0.0.1", "remotePort": port, "remoteRelayEndpoint": endpoint, "remoteRelayUseTls": false, "remoteAppBaseUrl": "http://" + endpoint})
	until := time.Now().Add(45 * time.Second)
	var snapshot RuntimeState
	for time.Now().Before(until) {
		snapshot = m.Snapshot()
		if snapshot.Status == "ready" {
			break
		}
		if snapshot.Status == "offline" || snapshot.Status == "missing" {
			t.Fatalf("real Go bridge stopped: %s %s", snapshot.Status, snapshot.Message)
		}
		time.Sleep(25 * time.Millisecond)
	}
	if snapshot.Status != "ready" || !snapshot.Listening || !snapshot.RelayConnected || snapshot.PairingURL == "" {
		t.Fatalf("real bridge not ready: %s %s", snapshot.Status, snapshot.Message)
	}
	checkHealth(fmt.Sprintf("http://127.0.0.1:%d/api/health", port))
	command, _ := json.Marshal(map[string]any{"action": "handshake", "offer": snapshot.PairingURL})
	if _, err = in.Write(append(command, '\n')); err != nil {
		t.Fatal(err)
	}
	handshake := wait("handshake")
	if string(handshake["e2ee"]) != "true" || string(handshake["channelBoundChallenge"]) != "true" || string(handshake["applicationHelloSent"]) != "false" {
		t.Fatal("E2EE handshake incomplete")
	}
	m.Stop()
	if bridge == nil || bridge.cmd.ProcessState == nil || !bridge.cmd.ProcessState.Exited() {
		t.Fatal("owned bridge did not exit")
	}
	if s := m.Snapshot(); s.Listening || s.PairingURL != "" {
		t.Fatal("stopped bridge retained pairing")
	}
	if c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond); err == nil {
		c.Close()
		t.Fatal("daemon listener survived stop")
	}
	_, _ = io.WriteString(in, "{\"action\":\"stop\"}\n")
	closed := wait("closed")
	in.Close()
	err = cmd.Wait()
	helperWaited = true
	if err != nil {
		t.Fatal(err)
	}
	var children []struct {
		PID    int  `json:"pid"`
		Exited bool `json:"exited"`
	}
	if json.Unmarshal(closed["children"], &children) != nil {
		t.Fatal("invalid process cleanup record")
	}
	for _, child := range children {
		if !child.Exited {
			t.Fatalf("owned child %d did not exit", child.PID)
		}
	}
	if c, err := net.DialTimeout("tcp", endpoint, 100*time.Millisecond); err == nil {
		c.Close()
		t.Fatal("relay listener survived close")
	}
	processes, err := os.ReadFile(processLog)
	if err != nil {
		t.Fatal(err)
	}
	starts, exits := map[int]bool{}, map[int]bool{}
	for _, line := range bytesLines(processes) {
		var p struct {
			Event string `json:"event"`
			PID   int    `json:"pid"`
		}
		if json.Unmarshal(line, &p) != nil {
			t.Fatal("process record invalid")
		}
		if p.Event == "node-start" {
			starts[p.PID] = true
		}
		if p.Event == "node-exit" {
			exits[p.PID] = true
		}
	}
	for pid := range starts {
		if !exits[pid] {
			t.Fatalf("owned Node pid %d missing exit record", pid)
		}
	}
	record := map[string]any{"pass": true, "helperPID": helperPID, "bridgePID": bridge.cmd.Process.Pid, "resolvedNode": resolved.Node, "relayEnabled": true, "mode": "away", "relayUseTls": false, "relayEndpoint": endpoint, "ready": ready, "handshake": handshake, "closed": closed, "processes": string(processes), "applicationHelloSent": false, "pairingTokenConsumed": false, "notVerified": []string{"application hello/device token pairing", "provider/model discovery", "real device/QR", "mobile LAN3180", "application session/RPC"}}
	encoded, _ := json.MarshalIndent(record, "", "  ")
	if err = os.WriteFile(filepath.Join(root, "acceptance.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("real Go bridge + original daemon health/control to own original relay; signed server auth and E2EE bound challenge; no application hello; owned processes/listeners closed")
}

func bytesLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			if i > start {
				lines = append(lines, b[start:i])
			}
			start = i + 1
		}
	}
	return lines
}

const ownedPhoneProcessPreload = `const fs=require('node:fs');const file=process.env.WHALEISLE_PHONE_PROCESS_RECORD;const put=record=>fs.appendFileSync(file,JSON.stringify(record)+'\n');put({event:'node-start',pid:process.pid,ppid:process.ppid,file:process.execPath});process.on('exit',code=>put({event:'node-exit',pid:process.pid,code}));`

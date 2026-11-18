package phoneremote

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/processutil"
)

// This opt-in acceptance uses the product Go runtime and shipped daemon,
// relay Worker and phone client. Only storage and network endpoints are local.
func TestOwnedMobileE2EEPairRevokeAndStop(t *testing.T) {
	vendor, toolRoot, node, report := os.Getenv("WHALEISLE_E2EE_VENDOR"), os.Getenv("WHALEISLE_E2EE_TOOLS"), os.Getenv("WHALEISLE_E2EE_NODE"), os.Getenv("WHALEISLE_E2EE_REPORT")
	if report == "" {
		t.Skip("explicit owned E2EE acceptance required")
	}
	for _, input := range []string{vendor, toolRoot, node, report} {
		if !filepath.IsAbs(input) {
			t.Fatal("absolute E2EE inputs required")
		}
	}
	work := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "DSH_HOME", "DSHD_HOME", "CHISACODE_HOME"} {
		t.Setenv(key, work)
	}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if strings.HasSuffix(upper, "API_KEY") || strings.Contains(upper, "AUTH_TOKEN") || upper == "NODE_OPTIONS" || upper == "ELECTRON_RUN_AS_NODE" {
			t.Setenv(key, "")
		}
	}
	t.Setenv("DSH_TELEMETRY_DISABLED", "1")
	t.Setenv("DO_NOT_TRACK", "1")
	_, source, _, _ := runtime.Caller(0)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, filepath.Join(filepath.Dir(source), "e2ee_mobile_acceptance.cjs"), work, filepath.Join(vendor, "dshd-remote"), toolRoot)
	processutil.BackgroundTree(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	helperDone := make(chan error, 1)
	go func() { helperDone <- cmd.Wait() }()
	defer func() {
		_ = stdin.Close()
		select {
		case <-helperDone:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-helperDone
		}
	}()
	events := make(chan map[string]any, 16)
	go func() {
		defer close(events)
		scan := bufio.NewScanner(stdout)
		scan.Buffer(make([]byte, 4096), 1024*1024)
		for scan.Scan() {
			var event map[string]any
			if json.Unmarshal(scan.Bytes(), &event) == nil {
				events <- event
			}
		}
	}()
	waitEvent := func(kind string) map[string]any {
		t.Helper()
		for {
			select {
			case event, ok := <-events:
				if !ok {
					t.Fatal("phone driver exited before " + kind)
				}
				if event["kind"] == "error" {
					t.Fatalf("phone driver: %v", event["message"])
				}
				if event["kind"] == kind {
					return event
				}
			case <-ctx.Done():
				t.Fatal("acceptance deadline before " + kind)
			}
		}
	}
	command := func(value map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(value)
		if _, err := stdin.Write(append(raw, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	checks := []map[string]any{}
	defer func() {
		body, _ := json.MarshalIndent(map[string]any{"scope": "real Go Runtime, shipped DshdRemote daemon, shipped Cloudflare relay on loopback Miniflare, original mobile session.js and DaemonClient", "modelRequestsByDriver": 0, "physicalPhone": false, "internetRelay": false, "checks": checks, "passed": !t.Failed()}, "", "  ")
		_ = os.WriteFile(report, append(body, '\n'), 0600)
	}()
	ready := waitEvent("ready")
	endpoint, _ := ready["endpoint"].(string)
	if host, _, err := net.SplitHostPort(endpoint); err != nil || host != "127.0.0.1" {
		t.Fatal("relay is not owned loopback")
	}
	port := freePort(t)
	// Observe the real daemon's public rejection reasons without changing its
	// logger or authentication. No offer, device secret, URL or payload is saved.
	preload := filepath.Join(work, "auth-audit.cjs")
	if err := os.WriteFile(preload, []byte(e2eeAuthAuditPreload), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WHALEISLE_E2EE_AUDIT", filepath.Join(work, "auth-audit.jsonl"))
	t.Setenv("NODE_OPTIONS", "--require="+preload)
	r := NewRuntime()
	r.stopTimeout = 20 * time.Second
	defer func() {
		if done := r.Stop(); done != nil {
			select {
			case <-done:
			case <-time.After(20 * time.Second):
				t.Error("owned bridge still running")
			}
		}
	}()
	r.Start(filepath.Join(work, "desktop"), vendor, map[string]any{"remoteEnabled": true, "remoteMode": "away", "remoteBindAddress": "127.0.0.1", "remotePort": port, "remoteRelayEndpoint": endpoint, "remoteRelayUseTls": false, "remoteAppBaseUrl": "http://" + endpoint})
	waitState := func(label string, match func(RuntimeState) bool) RuntimeState {
		t.Helper()
		until := time.Now().Add(75 * time.Second)
		for time.Now().Before(until) {
			s := r.Snapshot()
			if match(s) {
				return s
			}
			if s.Status == "missing" {
				t.Fatalf("%s: %s", label, s.Message)
			}
			time.Sleep(50 * time.Millisecond)
		}
		s := r.Snapshot()
		t.Fatalf("%s: %s %s", label, s.Status, s.Message)
		return s
	}
	s := waitState("initial ready", func(s RuntimeState) bool { return s.Status == "ready" && s.RelayConnected && s.PairingURL != "" })
	offer := s.PairingURL
	command(map[string]any{"action": "pair", "offer": offer})
	paired := waitEvent("paired")
	deviceID, _ := paired["deviceId"].(string)
	if paired["hello"] != true || paired["encryptedPing"] != true || paired["secretPersisted"] != true || deviceID == "" {
		t.Fatal("phone pairing did not finish")
	}
	checks = append(checks, map[string]any{"name": "one-time-pair-application-hello-and-encrypted-ping", "pass": true})
	command(map[string]any{"action": "sticky"})
	waitEvent("sticky")
	checks = append(checks, map[string]any{"name": "formal-device-proof-reconnect-and-encrypted-ping", "pass": true})
	command(map[string]any{"action": "replay", "offer": offer})
	waitEvent("replay-rejected")
	checks = append(checks, map[string]any{"name": "consumed-offer-rejected-with-existing-device-unaffected", "pass": true})
	waitState("device listed", func(s RuntimeState) bool {
		for _, d := range s.Devices {
			if d.ID == deviceID {
				return true
			}
		}
		return false
	})
	if err = r.DeviceAction("unbind", deviceID, ""); err != nil {
		t.Fatal(err)
	}
	waitState("device revoked and gateway resumed", func(s RuntimeState) bool {
		r.mu.Lock()
		action := r.deviceAction
		r.mu.Unlock()
		if action.Error != "" {
			t.Fatalf("device action: %s", action.Error)
		}
		if action.Busy || !action.Applied || !s.RelayConnected {
			return false
		}
		for _, d := range s.Devices {
			if d.ID == deviceID {
				return false
			}
		}
		return true
	})
	command(map[string]any{"action": "revoked"})
	waitEvent("revoked-rejected")
	checks = append(checks, map[string]any{"name": "native-unbind-closes-live-device-and-rejects-new-proof", "pass": true})
	s = waitState("fresh offer after revocation", func(s RuntimeState) bool { return s.Status == "ready" && s.PairingURL != "" })
	command(map[string]any{"action": "pair", "offer": s.PairingURL})
	waitEvent("paired")
	checks = append(checks, map[string]any{"name": "fresh-offer-recovers-revoked-device", "pass": true})
	if done := r.Stop(); done != nil {
		select {
		case <-done:
		case <-time.After(20 * time.Second):
			t.Fatal("bridge did not stop")
		}
	}
	if s = r.Snapshot(); s.Listening || s.PairingURL != "" {
		t.Fatal("stopped bridge advertises listening")
	}
	if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second); err == nil {
		_ = conn.Close()
		t.Fatal("daemon listener survived stop")
	}
	command(map[string]any{"action": "stop"})
	waitEvent("closed")
	if conn, err := net.DialTimeout("tcp", endpoint, time.Second); err == nil {
		_ = conn.Close()
		t.Fatal("relay listener survived close")
	}
	checks = append(checks, map[string]any{"name": "owned-daemon-and-relay-listeners-closed", "pass": true})
	t.Log("PASS original mobile application hello, E2EE ping, sticky proof, consumed token rejection, native device revoke, re-pair, runtime stop; no model request")
}

const e2eeAuthAuditPreload = `
const cp=require('node:child_process'),fs=require('node:fs');
const original=cp.spawn;
cp.spawn=function(file,args,options){
  const child=Reflect.apply(original,this,[file,args,options]);
  if(Array.isArray(args)&&String(args[0]).endsWith('dshd-daemon-runner.mjs')&&child.stdout){
    let buffered='';
    child.stdout.on('data',chunk=>{buffered+=chunk.toString();let end;
      while((end=buffered.indexOf('\n'))>=0){const line=buffered.slice(0,end);buffered=buffered.slice(end+1);
        try{const row=JSON.parse(line);if(row.msg==='Rejected relay hello'&&['relay_pairing_token_invalid','relay_device_auth_invalid'].includes(row.reason))fs.appendFileSync(process.env.WHALEISLE_E2EE_AUDIT,JSON.stringify({reason:row.reason})+'\n');}catch{}
      }
    });
  }
  return child;
};
`

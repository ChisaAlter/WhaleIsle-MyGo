package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/phoneremote"
)

// Explicit opt-in: starts only the selected bundled daemon with a throwaway
// profile. It drives the shipped HTTP route -> Node bridge -> real Go config
// transaction, without touching a user profile or displaying desktop windows.
func TestRemoteSaveLiveDesktopTransaction(t *testing.T) {
	vendor := os.Getenv("WHALEISLE_REMOTE_SAVE_LIVE_VENDOR")
	if vendor == "" {
		t.Skip("set WHALEISLE_REMOTE_SAVE_LIVE_VENDOR for the owned runtime integration")
	}
	vendor, err := filepath.Abs(vendor)
	if err != nil {
		t.Fatal(err)
	}
	mobileListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	mobilePort := mobileListener.Addr().(*net.TCPAddr).Port
	defer mobileListener.Close()
	portListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	daemonPort := portListener.Addr().(*net.TCPAddr).Port
	_ = portListener.Close()
	root := t.TempDir()
	// Override the exported pairing-page port before the real manager imports
	// it. This preload exists only in the owned process environment; neither
	// installed runtime bytes nor an existing listener is changed.
	module := filepath.Join(vendor, "dshd-remote", "src", "main", "dshd-remote.js")
	moduleJSON, err := json.Marshal(module)
	if err != nil {
		t.Fatal(err)
	}
	launch := filepath.Join(root, "owned-node-prelude.cjs")
	preload := fmt.Sprintf(`'use strict';
const {createRequire}=require('node:module');
const originalRequire=createRequire(%s);
const web=originalRequire('./mobile-web-server.js');
if(web.MOBILE_WEB_PORT!==3180)throw Error('Unexpected bundled mobile port contract');
web.MOBILE_WEB_PORT=%d;
`, moduleJSON, mobilePort)
	if err := os.WriteFile(launch, []byte(preload), 0600); err != nil {
		t.Fatal(err)
	}
	launchJSON, err := json.Marshal(launch)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODE_OPTIONS", "--require="+string(launchJSON))
	store, err := config.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(c *config.Config) { c.Theme = "deepseek"; c.Locale = "zh" }); err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store}
	runtime := phoneremote.NewRuntime()
	runtime.SetHostActions(phoneremote.HostActions{SaveConfig: s.savePhoneConfigPatch})
	defer runtime.Stop()
	if err := mobileListener.Close(); err != nil {
		t.Fatal(err)
	}
	runtime.Start(root, vendor, map[string]any{"remoteEnabled": true, "remoteMode": "lan", "remoteBindAddress": "127.0.0.1", "remotePort": daemonPort})
	var pairing string
	for end := time.Now().Add(90 * time.Second); time.Now().Before(end); {
		current := runtime.Snapshot()
		if current.Status == "ready" && current.PairingURL != "" {
			pairing = current.PairingURL
			break
		}
		if current.Status == "offline" || current.Status == "missing" {
			t.Fatalf("owned runtime failed before ready: %s: %s", current.Status, current.Message)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if pairing == "" {
		t.Fatal("owned runtime did not publish an offer")
	}
	_, fragment, ok := strings.Cut(pairing, "#offer=")
	if !ok {
		t.Fatal("owned runtime returned an invalid offer")
	}
	if i := strings.IndexAny(fragment, "&#"); i >= 0 {
		fragment = fragment[:i]
	}
	decoded, err := base64.RawURLEncoding.DecodeString(fragment)
	if err != nil {
		t.Fatal("owned offer could not be decoded")
	}
	var offer struct {
		Auth struct {
			Token string `json:"pairingToken"`
		} `json:"authBootstrap"`
	}
	if json.Unmarshal(decoded, &offer) != nil || offer.Auth.Token == "" {
		t.Fatal("owned offer carries no authentication token")
	}
	client := &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	post := func(patch map[string]any) (int, map[string]any) {
		t.Helper()
		raw, err := json.Marshal(map[string]any{"patch": patch})
		if err != nil {
			t.Fatal(err)
		}
		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/__remote__/shell/saveConfig", mobilePort), bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("authorization", "Bearer "+offer.Auth.Token)
		request.Header.Set("content-type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			t.Fatal("owned shell request failed")
		}
		defer response.Body.Close()
		payload, err := io.ReadAll(io.LimitReader(response.Body, 16384))
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err := json.Unmarshal(payload, &result); err != nil {
			t.Fatalf("owned shell returned invalid JSON (status %d)", response.StatusCode)
		}
		return response.StatusCode, result
	}
	// Route mounting happens after the first listening snapshot. A no-op save
	// proves the entire host bridge is ready, rather than merely a TCP listener.
	ready := false
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); {
		status, result := post(map[string]any{})
		if status == http.StatusOK && result["ok"] == true {
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		t.Fatal("owned desktop save bridge did not become ready")
	}
	before := store.Get()
	baseline, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, patch := range []map[string]any{
		{"theme": 42},
		{"theme": "deepseek.dark", "locale": 42},
		{"theme": "deepseek.dark", "remotePort": 3499.5},
		{"theme": "deepseek.dark", "remoteBindAddress": "not-an-ip"},
		{"theme": "deepseek.dark", "remoteAppBaseUrl": "http://[fd00::1]/pair"},
		{"theme": "deepseek.dark", "remoteAppBaseUrl": "https://example.com/pair?token=x"},
	} {
		status, result := post(patch)
		if status != http.StatusBadRequest || result["ok"] != false || result["error"] == nil || result["result"] != nil {
			t.Fatalf("invalid HTTP patch reported success: status=%d result=%v", status, result)
		}
		after, err := os.ReadFile(store.Path())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(after, baseline) || !reflect.DeepEqual(store.Get(), before) {
			t.Fatal("invalid HTTP patch changed the on-disk or in-memory config")
		}
	}
	status, result := post(map[string]any{"theme": "deepseek.dark", "locale": "en"})
	if status != http.StatusOK || result["ok"] != true {
		t.Fatalf("valid HTTP patch failed: status=%d result=%v", status, result)
	}
	reopened, err := config.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.Get(); got.Theme != "deepseek.dark" || got.Locale != "en" {
		t.Fatal("successful HTTP response was not persisted")
	}
	t.Log("real bundled HTTP -> bridge -> desktop transaction rejects invalid and partial saves; valid values survive reopening")
}

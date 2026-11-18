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
	"strings"
	"testing"
	"time"

	"whaleisle/internal/config"
	"whaleisle/internal/desktop/phoneremote"
)

// Explicit opt-in: starts only the selected bundled daemon with a throwaway
// profile. It rejects bootstrap HTTP access, then applies the real Go config
// disable transaction without touching a user profile or desktop windows.
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
	relay, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer relay.Close()
	go func() {
		for {
			connection, err := relay.Accept()
			if err != nil {
				return
			}
			_ = connection.Close()
		}
	}()
	if err := mobileListener.Close(); err != nil {
		t.Fatal(err)
	}
	runtime.Start(root, vendor, map[string]any{"remoteEnabled": true, "remoteMode": "lan", "remoteBindAddress": "127.0.0.1", "remotePort": daemonPort, "remoteRelayEndpoint": relay.Addr().String(), "remoteRelayUseTls": false})
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
	baseline, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"", offer.Auth.Token} {
		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/__remote__/shell/saveConfig", mobilePort), strings.NewReader(`{"patch":{"theme":"deepseek.dark"}}`))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("authorization", "Bearer "+token)
		request.Header.Set("content-type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			t.Fatal("owned shell request failed")
		}
		payload, readErr := io.ReadAll(io.LimitReader(response.Body, 16384))
		response.Body.Close()
		if readErr != nil || response.StatusCode != http.StatusUnauthorized || strings.TrimSpace(string(payload)) != "unauthorized" {
			t.Fatalf("retired shell accepted bootstrap credential: status=%d", response.StatusCode)
		}
	}
	after, err := os.ReadFile(store.Path())
	if err != nil || !bytes.Equal(baseline, after) {
		t.Fatal("rejected shell request changed configuration")
	}
	// The native desktop transaction remains usable and applies a disable to
	// the exact owned runtime, independently of the retired HTTP entry point.
	s.phoneRuntime = runtime
	if err := store.Update(func(c *config.Config) { c.RemoteEnabled = true }); err != nil {
		t.Fatal(err)
	}
	saved, err := s.savePhoneConfigPatch(map[string]any{"remoteEnabled": false})
	if err != nil || saved["gatewayState"] != "applying" {
		t.Fatalf("disable admission: %v %v", saved, err)
	}
	for end := time.Now().Add(15 * time.Second); time.Now().Before(end); {
		if current := runtime.Snapshot(); current.Status == "offline" || current.Status == "disabled" {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if current := runtime.Snapshot(); current.Listening || current.Status == "stopping" || current.Status == "stop-failed" {
		t.Fatalf("saved disable did not finish: %s", current.Status)
	}
	for _, port := range []int{mobilePort, daemonPort} {
		if connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond); err == nil {
			connection.Close()
			t.Fatalf("owned listener %d survived saved disable", port)
		}
	}
	t.Log("bundled HTTP rejects bootstrap auth without saving; native config disable closes the owned gateway listeners")
}

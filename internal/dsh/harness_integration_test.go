//go:build harness

package dsh_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/credentials"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// Run with: go test -tags harness ./internal/dsh/ -count=1 -timeout 5m

type harnessFixture struct {
	sup    *harness.Supervisor
	client *dsh.Client
	mux    *dsh.Mux
	home   string
}

func startVendoredHarness(t *testing.T, port int) *harnessFixture {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Test package dir is internal/dsh; repo root is two levels up.
	repo := filepath.Clean(filepath.Join(wd, "..", ".."))
	vendor := filepath.Join(repo, "runtime", "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	bin := filepath.Join(harnessRoot, "apps", "cli", "lib", "bin.js")
	if _, err := os.Stat(bin); err != nil {
		t.Skip("vendored harness not built:", err)
	}
	nodeBin := ""
	for _, c := range []string{
		`C:\AI\tools\node-v24.21.0-win-x64\node.exe`,
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "node", "node.exe"),
	} {
		if c != "" {
			if _, err := os.Stat(c); err == nil {
				nodeBin = c
				break
			}
		}
	}
	if nodeBin == "" {
		t.Skip("Node.js 22.19+ not found (set LaunchOptions.NodeBin)")
	}
	home := t.TempDir()
	t.Setenv(paths.EnvUserData, home)
	t.Setenv(paths.EnvVendor, vendor)

	sup := harness.New(harness.Options{
		Launch: harness.LaunchOptions{
			NodeBin:         nodeBin,
			HarnessRoot:     harnessRoot,
			VendorDir:       vendor,
			Host:            "127.0.0.1",
			Port:            port,
			Home:            filepath.Join(home, "dsh-home"),
			SkipUserPlugins: true,
		},
		ReadyTimeout: 2 * time.Minute,
	})
	sup.Start()
	t.Cleanup(func() { sup.Stop() })

	deadline := time.Now().Add(90 * time.Second)
	var st harness.Status
	for time.Now().Before(deadline) {
		st = sup.Status()
		if st.State == harness.Ready {
			break
		}
		if st.State == harness.Failed {
			t.Fatalf("harness failed: %v\n%s", st.Err, strings.Join(sup.Logs(), "\n"))
		}
		time.Sleep(200 * time.Millisecond)
	}
	if st.State != harness.Ready {
		t.Fatalf("not ready: %+v\n%s", st, strings.Join(sup.Logs(), "\n"))
	}

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: st.Endpoint.Origin, Cookie: st.Endpoint.Cookie}}
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	return &harnessFixture{sup: sup, client: client, mux: mux, home: home}
}

func harnessAPIKey() string {
	for _, k := range []string{"DEEPSEEK_API_KEY", "WHALEISLE_API_KEY", "DSH_API_KEY"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	// Prefer real desktop credentials when present (before tests override user data).
	dir := filepath.Join(os.Getenv("APPDATA"), "Deepseek-Harness-Desktop")
	if dir == filepath.Join("", "Deepseek-Harness-Desktop") {
		return ""
	}
	creds, err := credentials.Load(dir)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(creds.APIKey)
}

func TestHarnessMuxAndEvents(t *testing.T) {
	fx := startVendoredHarness(t, 47890)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := fx.mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	defer fx.mux.Close()

	readyCh := make(chan dsh.EventsReady, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- fx.client.FollowEvents(ctx, fx.mux, dsh.EventsFuncs{
			Ready: func(r dsh.EventsReady) {
				select {
				case readyCh <- r:
				default:
				}
				cancel()
			},
		})
	}()
	select {
	case r := <-readyCh:
		if r.ClientID == "" {
			t.Fatal("empty clientId")
		}
	case err := <-errCh:
		t.Fatalf("FollowEvents ended early: %v", err)
	case <-time.After(25 * time.Second):
		t.Fatal("timeout waiting for $events ready; logs:\n" + strings.Join(fx.sup.Logs(), "\n"))
	}
}

// TestHarnessSessionPromptCancel exercises plan §8 smoke:
// create session → prompt → optional approval → cancel.
// Does not require a successful LLM completion.
func TestHarnessSessionPromptCancel(t *testing.T) {
	apiKey := harnessAPIKey()

	fx := startVendoredHarness(t, 47891)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := fx.mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	defer fx.mux.Close()

	readyCh := make(chan dsh.EventsReady, 1)
	var approvalAllowed atomic.Bool
	eventsDone := make(chan error, 1)
	go func() {
		eventsDone <- fx.client.FollowEvents(ctx, fx.mux, dsh.EventsFuncs{
			Ready: func(r dsh.EventsReady) {
				select {
				case readyCh <- r:
				default:
				}
			},
			Waterfall: func(wf dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) {
				if wf.Event != "approval/request" {
					return nil, nil // automatic "next"
				}
				raw, _ := json.Marshal("allowed-once")
				out := dsh.EventResultOutcome{Kind: "result", Value: raw}
				approvalAllowed.Store(true)
				return &out, nil
			},
		})
	}()

	var ready dsh.EventsReady
	select {
	case ready = <-readyCh:
		if ready.ClientID == "" {
			t.Fatal("empty clientId")
		}
	case err := <-eventsDone:
		t.Fatalf("FollowEvents ended early: %v", err)
	case <-time.After(25 * time.Second):
		t.Fatal("timeout waiting for $events ready; logs:\n" + strings.Join(fx.sup.Logs(), "\n"))
	}

	if apiKey != "" {
		_ = fx.client.Call(ctx, "credentials/set", map[string]any{
			"ref":   "apiKey",
			"value": apiKey,
		}, nil)
	}

	cwd := filepath.Join(fx.home, "workspace")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	createReq := api.SessionCreateRequest{Cwd: &cwd}
	var created api.SessionCreateValue
	if err := fx.client.Call(ctx, "session/create", map[string]any{"request": createReq}, &created); err != nil {
		t.Fatalf("session/create: %v\nlogs:\n%s", err, strings.Join(fx.sup.Logs(), "\n"))
	}
	if created.SessionId == "" {
		t.Fatal("session/create returned empty sessionId")
	}

	block, err := json.Marshal(map[string]string{"type": "text", "text": "ping"})
	if err != nil {
		t.Fatal(err)
	}
	promptReq := api.SessionPromptRequest{
		RequestId: "harness-smoke-" + created.SessionId,
		SessionId: created.SessionId,
		Mode:      "queue",
		Content:   []json.RawMessage{block},
	}
	// Prompt RPC should accept/queue quickly; do not wait for LLM completion.
	promptCtx, promptCancel := context.WithTimeout(ctx, 20*time.Second)
	defer promptCancel()
	var promptResult json.RawMessage
	promptErr := fx.client.Call(promptCtx, "session/prompt", map[string]any{"request": promptReq}, &promptResult)
	if promptErr != nil {
		msg := promptErr.Error()
		if apiKey == "" || looksLikeMissingAPIKey(msg) {
			t.Skip("session/prompt unavailable without API key: ", promptErr)
		}
		t.Fatalf("session/prompt: %v\nlogs:\n%s", promptErr, strings.Join(fx.sup.Logs(), "\n"))
	}

	// Brief window for a tool approval waterfall; absence is fine.
	approvalDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(approvalDeadline) && !approvalAllowed.Load() {
		time.Sleep(50 * time.Millisecond)
	}
	if approvalAllowed.Load() {
		t.Log("auto-allowed approval/request via PostEventResult")
	}

	var cancelOut api.SessionCancelValue
	if err := fx.client.Call(ctx, "session/cancel", map[string]any{
		"request": map[string]string{"sessionId": created.SessionId},
	}, &cancelOut); err != nil {
		t.Fatalf("session/cancel: %v\nlogs:\n%s", err, strings.Join(fx.sup.Logs(), "\n"))
	}
	t.Logf("sessionId=%s cancelAccepted=%v approval=%v", created.SessionId, cancelOut.Accepted, approvalAllowed.Load())

	cancel() // stop FollowEvents
	select {
	case <-eventsDone:
	case <-time.After(5 * time.Second):
	}
}

func looksLikeMissingAPIKey(msg string) bool {
	lower := strings.ToLower(msg)
	for _, s := range []string{"api key", "apikey", "api_key", "unauthorized", "401", "no credential", "missing credential", "authentication"} {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

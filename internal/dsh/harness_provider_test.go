//go:build harness

package dsh_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// This is the end-to-end proof for a provider created the way the settings card
// creates one: the same settings/update patch, the same apiKeyEnv mutation, the
// same credentials/set, against the real vendored Host, and then one turn that
// comes back with an answer. It reaches the operator's local gateway, so it runs
// only when asked:
//
//	WHALEISLE_E2E_GATEWAY=1 go test -tags harness ./internal/dsh/ -run TestProviderCreatedBySettingsAnswers -count=1 -timeout 5m
const (
	e2eGatewayBaseURL = "http://127.0.0.1:3425/v1"
	e2eGatewayKey     = "magpie"
	e2eModel          = "auto/gpt-6-luna"
)

func e2eVendor(t *testing.T) (vendor, harnessRoot, nodeBin string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	vendor = os.Getenv("WHALEISLE_E2E_VENDOR")
	if vendor == "" {
		vendor = filepath.Join(filepath.Clean(filepath.Join(wd, "..", "..")), "runtime", "vendor")
	}
	harnessRoot = filepath.Join(vendor, "deepseek-harness")
	if _, err := os.Stat(filepath.Join(harnessRoot, "apps", "cli", "lib", "bin.js")); err != nil {
		t.Skip("vendored harness not present:", err)
	}
	nodeBin = os.Getenv("WHALEISLE_E2E_NODE")
	if nodeBin == "" {
		for _, c := range []string{
			`C:\AI\WhaleIsle-MyGo-rt022\runtime-0.2.2-rc2\node\node.exe`,
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "node", "node.exe"),
		} {
			if _, err := os.Stat(c); err == nil {
				nodeBin = c
				break
			}
		}
	}
	if nodeBin == "" {
		t.Skip("no Node.js for the harness")
	}
	return vendor, harnessRoot, nodeBin
}

// startUserLayerHost boots the vendored Host as the app boots it: with the
// profile's own patch layer writable, so settings edits reach llm-pi-ai.
func startUserLayerHost(t *testing.T, port int) (*harness.Supervisor, *dsh.Client) {
	t.Helper()
	vendor, harnessRoot, nodeBin := e2eVendor(t)
	home := t.TempDir()
	t.Setenv(paths.EnvUserData, home)
	t.Setenv(paths.EnvVendor, vendor)
	sup := harness.New(harness.Options{
		Launch: harness.LaunchOptions{
			NodeBin:     nodeBin,
			HarnessRoot: harnessRoot,
			VendorDir:   vendor,
			Host:        "127.0.0.1",
			Port:        port,
			Home:        filepath.Join(home, "dsh-home"),
		},
		ReadyTimeout: 2 * time.Minute,
	})
	sup.Start()
	t.Cleanup(func() { sup.Stop() })
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		st := sup.Status()
		if st.State == harness.Ready {
			return sup, &dsh.Client{Endpoint: dsh.Endpoint{Origin: st.Endpoint.Origin, Cookie: st.Endpoint.Cookie}}
		}
		if st.State == harness.Failed {
			t.Fatalf("harness failed: %v\n%s", st.Err, strings.Join(sup.Logs(), "\n"))
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("harness not ready\n%s", strings.Join(sup.Logs(), "\n"))
	return nil, nil
}

func TestProviderCreatedBySettingsAnswers(t *testing.T) {
	if os.Getenv("WHALEISLE_E2E_GATEWAY") == "" {
		t.Skip("set WHALEISLE_E2E_GATEWAY=1 to reach the local gateway")
	}
	sup, client := startUserLayerHost(t, 48123)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	var described api.SettingsDescribeValue
	if err := client.Call(ctx, "settings/describe", nil, &described); err != nil {
		t.Fatalf("settings/describe: %v", err)
	}
	var revision float64
	var schema string
	found := false
	for _, ns := range described.Namespaces {
		if ns.Ns == "llm-pi-ai" {
			revision, schema, found = ns.Revision, string(ns.Schema), true
		}
	}
	if !found {
		t.Fatalf("the Host offers no llm-pi-ai namespace: %+v", described.Namespaces)
	}
	// The form's 协议 field reads this union: the Responses wire must be one of
	// the values on offer, or no card could ever ask for it.
	if !strings.Contains(schema, "openai-responses") {
		t.Fatalf("the protocol enum does not offer Responses: %s", schema)
	}

	// Exactly the patch the create card submits.
	patch := map[string]any{"providers": map[string]any{
		"magpie": map[string]any{
			"api":     "openai-responses",
			"baseURL": e2eGatewayBaseURL,
			"models":  []any{map[string]any{"id": e2eModel}},
		},
	}}
	var updated api.SettingsNamespaceView
	if err := client.Call(ctx, "settings/update", map[string]any{
		"ns": "llm-pi-ai", "patch": patch, "expectedRevision": revision,
	}, &updated); err != nil {
		t.Fatalf("settings/update: %v\n%s", err, strings.Join(sup.Logs(), "\n"))
	}

	// The credential step: the reference on the profile, then the key itself.
	var mutated api.SettingsNamespaceView
	if err := client.Call(ctx, "settings/mutate", map[string]any{
		"ns": "llm-pi-ai", "expectedRevision": updated.Revision,
		"ops": []map[string]any{{"op": "set", "path": []string{"providers", "magpie", "apiKeyEnv"}, "value": "MAGPIE_API_KEY"}},
	}, &mutated); err != nil {
		t.Fatalf("settings/mutate: %v\n%s", err, strings.Join(sup.Logs(), "\n"))
	}
	if err := client.Call(ctx, "credentials/set", map[string]any{"ref": "MAGPIE_API_KEY", "value": e2eGatewayKey}, nil); err != nil {
		t.Fatalf("credentials/set: %v", err)
	}

	// Live application: the running Host offers the route without a restart.
	catalogDeadline := time.Now().Add(45 * time.Second)
	var offered bool
	for time.Now().Before(catalogDeadline) && !offered {
		var catalog json.RawMessage
		if err := client.Call(ctx, "session/modelCatalog", map[string]any{}, &catalog); err != nil {
			t.Fatalf("session/modelCatalog: %v", err)
		}
		offered = strings.Contains(string(catalog), e2eModel) && strings.Contains(string(catalog), "magpie")
		if !offered {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if !offered {
		t.Fatalf("the running Host never offered magpie/%s; no restart should have been needed\n%s", e2eModel, strings.Join(sup.Logs(), "\n"))
	}

	var mu sync.Mutex
	var emits []string
	eventsDone := make(chan error, 1)
	readyCh := make(chan struct{}, 1)
	mux := &dsh.Mux{Endpoint: client.Endpoint}
	if err := mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	go func() {
		eventsDone <- client.FollowEvents(ctx, mux, dsh.EventsFuncs{
			Ready: func(dsh.EventsReady) {
				select {
				case readyCh <- struct{}{}:
				default:
				}
			},
			Emit: func(e dsh.EventsEmit) {
				mu.Lock()
				emits = append(emits, e.Event+" "+string(joinRaw(e.Args)))
				mu.Unlock()
			},
			Waterfall: func(dsh.EventsWaterfall) (*dsh.EventResultOutcome, error) { return nil, nil },
		})
	}()
	select {
	case <-readyCh:
	case err := <-eventsDone:
		t.Fatalf("events ended early: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatalf("no $events ready\n%s", strings.Join(sup.Logs(), "\n"))
	}

	cwd := filepath.Join(os.Getenv(paths.EnvUserData), "workspace")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	var created api.SessionCreateValue
	if err := client.Call(ctx, "session/create", map[string]any{"request": api.SessionCreateRequest{Cwd: &cwd}}, &created); err != nil {
		t.Fatalf("session/create: %v", err)
	}
	var selected json.RawMessage
	if err := client.Call(ctx, "session/selectModel", map[string]any{
		"request": map[string]any{"sessionId": created.SessionId, "provider": "magpie", "model": e2eModel},
	}, &selected); err != nil {
		t.Fatalf("session/selectModel: %v\n%s", err, strings.Join(sup.Logs(), "\n"))
	}

	block, _ := json.Marshal(map[string]string{"type": "text", "text": "用一个词回答 OK"})
	var prompted json.RawMessage
	if err := client.Call(ctx, "session/prompt", map[string]any{"request": api.SessionPromptRequest{
		RequestId: "e2e-" + created.SessionId,
		SessionId: created.SessionId,
		Mode:      "queue",
		Content:   []json.RawMessage{block},
	}}, &prompted); err != nil {
		t.Fatalf("session/prompt: %v\n%s", err, strings.Join(sup.Logs(), "\n"))
	}

	// The answer is the session's own record of the turn: the page carries every
	// event the Host persisted, assistant messages included.
	address, _ := json.Marshal(map[string]any{"kind": "session", "sessionId": created.SessionId})
	maxMessages := float64(50)
	pageRequest := api.SessionPageRequest{Address: address, MaxMessages: &maxMessages}
	answerDeadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(answerDeadline) {
		var page api.SessionEventsPage
		if err := client.Call(ctx, "session/page", map[string]any{"request": pageRequest}, &page); err == nil {
			for _, record := range page.Records {
				data := string(record.Event.Data)
				if strings.Contains(data, `"assistant"`) && strings.Contains(data, "OK") {
					shown := data
					if len(shown) > 240 {
						shown = shown[:240]
					}
					t.Logf("answered through the gateway; session %s; record %s", created.SessionId, shown)
					return
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	var page api.SessionEventsPage
	show := ""
	if err := client.Call(ctx, "session/page", map[string]any{"request": pageRequest}, &page); err == nil {
		raw, _ := json.Marshal(page)
		show = string(raw)
	}
	t.Fatalf("no assistant answer; events: %v\npage: %s\ngateway logs:\n%s", emits, show, strings.Join(sup.Logs(), "\n"))
}

func joinRaw(args []json.RawMessage) []byte {
	out := make([]byte, 0, 256)
	for _, a := range args {
		out = append(out, a...)
		out = append(out, ' ')
	}
	return out
}

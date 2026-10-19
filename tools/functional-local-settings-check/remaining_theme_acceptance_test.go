//go:build functionalacceptance && windows

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
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
	settingsstore "whaleisle/internal/store/settings"
	settingsui "whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

type themeAcceptanceReceipt struct {
	ExpectedRevision float64   `json:"expectedRevision"`
	ReturnedRevision float64   `json:"returnedRevision,omitempty"`
	Fields           []string  `json:"fields"`
	ErrorCode        string    `json:"errorCode,omitempty"`
	Sent             time.Time `json:"sent"`
}

type themeAcceptanceAudit struct {
	mu       sync.Mutex
	paths    map[string]int
	blocked  int
	receipts []themeAcceptanceReceipt
}

type themeAcceptanceTransport struct {
	origin string
	audit  *themeAcceptanceAudit
}

func (a *themeAcceptanceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	var raw []byte
	var err error
	if request.Body != nil {
		raw, err = io.ReadAll(io.LimitReader(request.Body, 64<<10))
		request.Body.Close()
		if err != nil {
			return nil, err
		}
	}
	request.Body = io.NopCloser(bytes.NewReader(raw))
	var envelope struct {
		Method  string `json:"method"`
		Payload struct {
			Args struct {
				NS       string  `json:"ns"`
				Revision float64 `json:"expectedRevision"`
				Ops      []struct {
					Op    string   `json:"op"`
					Path  []string `json:"path"`
					Value any      `json:"value"`
				} `json:"ops"`
			} `json:"args"`
		} `json:"payload"`
	}
	valid := json.Unmarshal(raw, &envelope) == nil
	path := request.URL.Path
	allowed := valid && request.Method == http.MethodPost && request.URL.Scheme+"://"+request.URL.Host == a.origin && request.URL.RawQuery == "" && request.URL.User == nil && path == "/api/"+envelope.Method
	mutation := envelope.Method == "settings/mutate"
	if mutation {
		allowed = allowed && envelope.Payload.Args.NS == "ui-theme" && len(envelope.Payload.Args.Ops) > 0 && len(envelope.Payload.Args.Ops) <= 2
		for _, op := range envelope.Payload.Args.Ops {
			if op.Op != "set" || len(op.Path) != 1 {
				allowed = false
				continue
			}
			switch op.Path[0] {
			case "fontSize":
				value, ok := op.Value.(float64)
				allowed = allowed && ok && value >= 10 && value <= 22 && value == float64(int(value))
			case "preference":
				value, ok := op.Value.(string)
				allowed = allowed && ok && (value == "light" || value == "dark" || value == "system")
			default:
				allowed = false
			}
		}
	} else {
		// The second read is intrinsic to Store.RefreshDescribe, not a preset
		// workflow. No permission write, account or credential RPC is admitted.
		allowed = allowed && (envelope.Method == "settings/describe" || envelope.Method == "permissionPresets/catalog")
	}
	a.audit.mu.Lock()
	a.audit.paths[path]++
	if !allowed {
		a.audit.blocked++
	}
	a.audit.mu.Unlock()
	if !allowed {
		return nil, errors.New("theme acceptance refused a request outside its owned settings scope")
	}
	receipt := themeAcceptanceReceipt{ExpectedRevision: envelope.Payload.Args.Revision, Sent: time.Now()}
	for _, op := range envelope.Payload.Args.Ops {
		receipt.Fields = append(receipt.Fields, op.Path[0])
	}
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil || !mutation {
		return response, err
	}
	// Keep only the scalar-write receipt's revisions and error code. Never
	// retain describe bodies, provider values, cookies or raw diagnostics.
	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	response.Body.Close()
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(data))
	var result struct {
		Result struct {
			Value struct {
				Revision float64 `json:"revision"`
			} `json:"value"`
			Error *struct {
				Code string `json:"code"`
			} `json:"error"`
		} `json:"result"`
	}
	if json.Unmarshal(data, &result) == nil {
		receipt.ReturnedRevision = result.Result.Value.Revision
		if result.Result.Error != nil {
			receipt.ErrorCode = result.Result.Error.Code
		}
	}
	a.audit.mu.Lock()
	a.audit.receipts = append(a.audit.receipts, receipt)
	a.audit.mu.Unlock()
	return response, nil
}

// TestFunctionalOwnedThemeCAS uses the same owned environment and Supervisor
// lifecycle as main_test.go, without that check's skills/preset fixtures. All
// RPCs and editor settlements below reach existing production public seams.
func TestFunctionalOwnedThemeCAS(t *testing.T) {
	stage, base, output := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME"), os.Getenv("WHALEISLE_FUNCTIONAL_ROOT"), os.Getenv("WHALEISLE_FUNCTIONAL_OUTPUT")
	for _, path := range []string{stage, base, output} {
		if !filepath.IsAbs(path) {
			t.Fatal("explicit absolute staged runtime, owned parent and evidence output required")
		}
	}
	root, err := os.MkdirTemp(base, "theme-functional-")
	if err != nil {
		t.Fatal(err)
	}
	type check struct{ Name, Status, Detail string }
	report := struct {
		Root           string                   `json:"ownedRoot"`
		Runtime        string                   `json:"runtime"`
		Scope          string                   `json:"scope"`
		Checks         []check                  `json:"checks"`
		RPCs           map[string]int           `json:"observedRPCs"`
		Receipts       []themeAcceptanceReceipt `json:"themeReceipts"`
		Blocked        int                      `json:"blockedRPCs"`
		HostStarts     int                      `json:"ownedHostStarts"`
		HostStopped    bool                     `json:"ownedHostStopped"`
		ListenerClosed bool                     `json:"ownedListenerClosed"`
	}{Root: root, Runtime: stage, Scope: "Fresh owned local Host, original ui-theme schema, production settings Store/HostThemeBinding/Editor, real debounce/CAS/retry/restart persistence. Only theme fontSize/preference mutations plus settings describe and its intrinsic read-only permission catalog. No Sessions, models, skills/preset workflow, account/credentials, external resources or user GUI; controller acceptance does not certify visual painting. Desktop onboarding remains unexecuted because real login/API-key/credit prerequisites are outside this run."}
	audit := &themeAcceptanceAudit{paths: map[string]int{}}
	write := func() {
		audit.mu.Lock()
		report.RPCs = make(map[string]int, len(audit.paths))
		for path, count := range audit.paths {
			report.RPCs[path] = count
		}
		report.Blocked, report.Receipts = audit.blocked, append([]themeAcceptanceReceipt(nil), audit.receipts...)
		audit.mu.Unlock()
		data, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(output, append(data, '\n'), 0600); err != nil {
			t.Errorf("save theme evidence: %v", err)
		}
	}
	add := func(name, status, detail string) {
		report.Checks = append(report.Checks, check{name, status, detail})
		write()
		t.Log(name, status, detail)
	}
	fail := func(name, detail string) {
		add(name, "fail", detail)
		t.Fatalf("%s: %s", name, detail)
	}
	errorLabel := func(err error) string {
		if err == nil {
			return ""
		}
		digest := sha256.Sum256([]byte(err.Error()))
		return "operation-error sha256:" + hex.EncodeToString(digest[:])
	}
	ownedHome, workspace := filepath.Join(root, "owned-home"), filepath.Join(root, "workspace")
	envs := map[string]string{"HOME": ownedHome, "USERPROFILE": ownedHome, "HOMEDRIVE": filepath.VolumeName(ownedHome), "HOMEPATH": strings.TrimPrefix(ownedHome, filepath.VolumeName(ownedHome)), "APPDATA": filepath.Join(root, "appdata"), "LOCALAPPDATA": filepath.Join(root, "local-appdata"), "XDG_CONFIG_HOME": filepath.Join(ownedHome, ".config"), "XDG_DATA_HOME": filepath.Join(ownedHome, ".local", "share"), "XDG_CACHE_HOME": filepath.Join(ownedHome, ".cache"), "DSH_AGENTS_HOME": filepath.Join(ownedHome, ".agents"), "DSH_BUNDLED_SKILL_DIR": filepath.Join(root, "bundled-skills"), "COREPACK_ENABLE_NETWORK": "0", "npm_config_offline": "true"}
	for _, path := range []string{ownedHome, workspace, filepath.Join(workspace, ".git"), envs["APPDATA"], envs["LOCALAPPDATA"], envs["DSH_AGENTS_HOME"], envs["DSH_BUNDLED_SKILL_DIR"]} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range envs {
		t.Setenv(key, value)
	}
	t.Setenv(paths.EnvUserData, filepath.Join(root, "user-data"))
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "user-data", "dsh-home"))
	vendor, home, profile := paths.VendorDir(), paths.DshHome(), paths.WebProfileDir()
	node, harnessRoot := filepath.Join(stage, "node", "node.exe"), filepath.Join(vendor, "deepseek-harness")
	if err := os.MkdirAll(profile, 0700); err != nil {
		t.Fatal(err)
	}
	supervisor := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 36820, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: profile, RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	store := settingsstore.New()
	previousPreferences := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(previousPreferences) })
	stop := func() {
		store.Detach()
		supervisor.Stop()
		report.HostStopped = supervisor.Status().State == harness.Stopped
		report.ListenerClosed = false
		if endpoint.Origin != "" {
			origin, _ := url.Parse(endpoint.Origin)
			connection, err := net.DialTimeout("tcp", origin.Host, time.Second)
			report.ListenerClosed = err != nil
			if connection != nil {
				connection.Close()
			}
		}
	}
	defer func() {
		stop()
		write()
		if report.Blocked != 0 || !report.HostStopped || report.HostStarts > 0 && !report.ListenerClosed {
			t.Errorf("theme scope/owned process cleanup failed: blocked=%d stopped=%v listenerClosed=%v", report.Blocked, report.HostStopped, report.ListenerClosed)
		}
	}()
	start := func() *dsh.Client {
		supervisor.Start()
		report.HostStarts++
		until := time.Now().Add(195 * time.Second)
		for time.Now().Before(until) {
			state := supervisor.Status()
			if state.State == harness.Ready {
				endpoint = state.Endpoint
				return &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: &http.Client{Timeout: 30 * time.Second, Transport: &themeAcceptanceTransport{origin: endpoint.Origin, audit: audit}, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("theme acceptance refuses redirects") }}}
			}
			if state.State == harness.Failed {
				fail("owned Host", errorLabel(state.Err))
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
		fail("owned Host", "readiness timeout")
		return nil
	}
	wait := func(name string, predicate func() bool) {
		until := time.Now().Add(12 * time.Second)
		for time.Now().Before(until) {
			audit.mu.Lock()
			blocked := audit.blocked
			audit.mu.Unlock()
			if blocked != 0 {
				fail(name, "RPC allowlist refused an out-of-scope operation")
			}
			if predicate() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		fail(name, "condition timeout; no raw settings or diagnostics recorded")
	}
	client := start()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	store.Attach(client, nil)
	store.RefreshDescribe(ctx)
	wait("theme schema", func() bool { return !store.Snapshot().Loading && len(store.Snapshot().Namespaces) > 0 })
	var namespace settingsstore.Namespace
	for _, row := range store.Snapshot().Namespaces {
		if row.NS == "ui-theme" {
			namespace = row
		}
	}
	fontServed, preferenceServed := false, false
	for _, field := range namespace.Fields {
		if field.Key == "fontSize" && field.Kind == settingsstore.FieldNumber {
			fontServed = true
		}
		if field.Key == "preference" && field.Kind == settingsstore.FieldEnum {
			options := " " + strings.Join(field.EnumOptions, " ") + " "
			preferenceServed = strings.Contains(options, " light ") && strings.Contains(options, " dark ")
		}
	}
	if !store.Snapshot().Writable || !fontServed || !preferenceServed {
		fail("theme schema", "actual Host did not serve writable fontSize/preference scalar fields")
	}
	binding := settingsui.BindHostTheme(store)
	if binding == nil || !binding.Snapshot().Ready {
		fail("theme binding", "production binding unavailable")
	}
	initial := binding.Snapshot()
	editor := theme.NewEditor(initial.Preferences, binding.SaveTypography)
	editor.Pending = binding.Pending
	owner := initial.Owner
	// This is the production frame settlement sequence from
	// mainState.syncHostAppearance, using public controller methods. Store.fire
	// owns the actual queue pump; there is no fixture scheduler or fake receipt.
	pump := func() settingsui.HostThemeState {
		state := binding.Snapshot()
		if state.Ready && !state.Pending && !state.Saving && !state.Loading {
			editor.AcceptHost(state.Preferences, owner != state.Owner, state.Error)
			owner = state.Owner
		}
		return state
	}
	add("theme actual schema", "pass", "fresh Host served writable ui-theme fontSize number and preference light/dark enum; production Store and HostThemeBinding loaded")
	firstSize := float32(16)
	if initial.Preferences.Chat.Size == firstSize {
		firstSize = 15
	}
	editor.Draft.Chat.Size = firstSize
	queued := time.Now()
	editor.Save() // Normal 300ms queue; no Flush/stepper bypass.
	if !binding.Pending() || editor.Saved.Chat.Size != initial.Preferences.Chat.Size || editor.Draft.Chat.Size != firstSize {
		fail("theme queued draft", "editor advanced accepted state before the Host receipt")
	}
	wait("theme debounced save", func() bool {
		state := pump()
		return !state.Pending && !state.Loading && state.Error == "" && state.Revision > initial.Revision && editor.Saved.Chat.Size == firstSize
	})
	audit.mu.Lock()
	receipts := append([]themeAcceptanceReceipt(nil), audit.receipts...)
	audit.mu.Unlock()
	if len(receipts) != 1 || receipts[0].Sent.Sub(queued) < 300*time.Millisecond || receipts[0].ExpectedRevision != initial.Revision || len(receipts[0].Fields) != 1 || receipts[0].Fields[0] != "fontSize" || receipts[0].ErrorCode != "" {
		fail("theme debounce/CAS", "first actual request did not preserve 300ms delay, scalar-only mutation and captured revision")
	}
	add("theme normal debounce/write", "pass", "production editor kept draft pending; actual scalar-only settings/mutate started after 300ms with captured revision, and public receipt pump adopted Host value")
	accepted := binding.Snapshot()
	externalPreference := "dark"
	if accepted.Scheme == externalPreference {
		externalPreference = "light"
	}
	var external api.SettingsNamespaceView
	externalClient := &dsh.Client{Endpoint: client.Endpoint, HTTP: client.HTTP}
	err = externalClient.Call(ctx, "settings/mutate", map[string]any{"ns": "ui-theme", "expectedRevision": accepted.Revision, "ops": []any{map[string]any{"op": "set", "path": []string{"preference"}, "value": externalPreference}}}, &external)
	if err != nil || external.Revision <= accepted.Revision {
		fail("theme concurrent writer", "owned independent mutation failed: "+errorLabel(err))
	}
	secondSize := float32(18)
	editor.Draft.Chat.Size = secondSize
	editor.Save() // Binding still owns accepted.Revision: Host must reject it.
	wait("theme actual CAS conflict", func() bool {
		state := pump()
		return !state.Pending && !state.Loading && !state.Saving && state.Error != "" && state.Revision == external.Revision
	})
	conflicted := binding.Snapshot()
	if editor.Draft.Chat.Size != secondSize || editor.Saved.Chat.Size != firstSize || conflicted.Scheme != externalPreference || editor.Status == "" {
		fail("theme conflict draft", "actual rejection lost pending editor draft, accepted size or independent preference")
	}
	audit.mu.Lock()
	receipts = append([]themeAcceptanceReceipt(nil), audit.receipts...)
	audit.mu.Unlock()
	if len(receipts) != 3 || receipts[2].ErrorCode != "settings/conflict" || receipts[2].ExpectedRevision != accepted.Revision {
		fail("theme actual CAS conflict", "Host did not report settings/conflict against the stale captured revision")
	}
	add("theme actual CAS rejection", "pass", "second owned writer advanced preference/revision; stale production write received settings/conflict, refreshed Host state and retained unsaved editor size without replacing preference")
	editor.Save() // Explicit user retry, still using the ordinary debounce.
	wait("theme explicit retry", func() bool {
		state := pump()
		return !state.Pending && !state.Loading && state.Error == "" && state.Revision > external.Revision && editor.Saved.Chat.Size == secondSize
	})
	final := binding.Snapshot()
	if final.Scheme != externalPreference || editor.Draft.Chat.Size != secondSize {
		fail("theme retry preservation", "retry overwrote independent preference or acknowledged the wrong size")
	}
	audit.mu.Lock()
	receipts = append([]themeAcceptanceReceipt(nil), audit.receipts...)
	audit.mu.Unlock()
	if len(receipts) != 4 || receipts[3].ExpectedRevision != external.Revision || receipts[3].ErrorCode != "" {
		fail("theme retry CAS", "retry did not use the refreshed Host revision")
	}
	add("theme explicit retry", "pass", "ordinary editor Save retried against refreshed revision; real Host accepted fontSize while preserving the independent preference")
	oldBinding, oldOwner := binding, owner
	stop()
	if !report.HostStopped || !report.ListenerClosed {
		fail("theme restart stop", "owned Host or listener did not stop")
	}
	client = start()
	store.Attach(client, nil)
	store.RefreshDescribe(ctx)
	wait("theme restart describe", func() bool { return !store.Snapshot().Loading && len(store.Snapshot().Namespaces) > 0 })
	binding = settingsui.BindHostTheme(store)
	if binding == nil {
		fail("theme restart binding", "actual namespace unavailable after owned Host restart")
	}
	editor.OnSave, editor.Pending = binding.SaveTypography, binding.Pending
	restored := pump()
	if !restored.Ready || restored.Error != "" || restored.Owner == oldOwner || restored.Preferences.Chat.Size != secondSize || restored.Scheme != externalPreference || editor.Saved.Chat.Size != secondSize || oldBinding.Pending() {
		fail("theme restart persistence", "actual restart did not restore values under a new binding owner")
	}
	if err := oldBinding.SaveScheme("system"); err == nil {
		fail("theme retired owner", "old binding retained authority after detach/restart")
	}
	add("theme owned restart persistence", "pass", "actual stopped/restarted Host restored fontSize and independent preference; public frame settlement adopted new owner and retired binding refused a late write")
	add("desktop onboarding", "not-run", "implemented source gate requires real credential-stored login or API-key metadata; account/credential/credit operations are excluded, and no status/key facts were fabricated")
}

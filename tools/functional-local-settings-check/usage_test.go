//go:build functionalacceptance && windows

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	manager "whaleisle/internal/pluginmanager"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
	usageui "whaleisle/internal/ui/usage"
)

type usageQATransport struct {
	mu      sync.Mutex
	counts  map[string]int
	blocked int
	gate    chan struct{}
}

func (a *usageQATransport) RoundTrip(r *http.Request) (*http.Response, error) {
	allowed := r.URL.Path == "/usage-stats/overview" || r.URL.Path == "/usage-stats/projects.more" || r.URL.Path == "/api/pluginManager/inspect" || r.URL.Path == "/api/pluginManager/installBundle" || r.URL.Path == "/api/pluginManager/removeBundle"
	a.mu.Lock()
	a.counts[r.URL.Path]++
	gate := a.gate
	if !allowed {
		a.blocked++
	}
	a.mu.Unlock()
	if !allowed {
		return nil, errors.New("only owned usage read and explicit fixture registration APIs admitted")
	}
	if gate != nil && r.URL.Path == "/usage-stats/overview" {
		select {
		case <-gate:
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	}
	return http.DefaultTransport.RoundTrip(r)
}

// This fixture only uses the original public durable-storage API. The synthetic
// closed turn records describe historical metering; no Agent/loop/LLM is called.
const usageSeedPlugin = `import {writeFileSync} from 'node:fs';
import {pathToFileURL} from 'node:url';
export const inject=['sessionPersistence','sessionQuery'];
export async function apply(ctx) {
  const expectedHome=__HOME__, marker=__MARKER__, rows=__ROWS__;
  if (process.env.DSH_HOME!==expectedHome) throw Error('owned data boundary mismatch');
  try {
    const {sessionFormatCatalog:catalog}=await import(pathToFileURL(__CATALOG__));
    for (const row of rows) {
      const restore=catalog.createRestore(catalog.encodeCurrentHeader(row.header,0),{recovery:'strict',validation:'current'});
      for(const event of row.events)restore.decodeRow(catalog.encodeCurrentEvent(event));
      if(restore.finish().events.length!==6)throw Error('complete public catalog preflight changed the event count');
    }
    let appended=0;
    for (const row of rows) {
      if (!await ctx.sessionPersistence.stat(row.header.id)) {
        const handle=await ctx.sessionPersistence.create(row.header);
        try { await handle.append(row.events); await handle.flush(); appended+=row.events.length; }
        finally { await handle.close(); }
      }
      const read=await ctx.sessionQuery.readSession(row.header.id);
      if(read.events.length!==6)throw Error('current-format read did not preserve six fixture events');
    }
    writeFileSync(marker,JSON.stringify({status:'accepted',sessions:rows.length,eventsAppended:appended,validatedBeforeWrite:true,agentCreated:false,executedRequests:0,source:'public strict current SessionFormatCatalog preflight, SessionPersistence create/append/flush/close; strict SessionQuery read'}));
  } catch(error) {
    writeFileSync(marker,JSON.stringify({status:'rejected',error:String(error),agentCreated:false,executedRequests:0}));throw error;
  }
}
`

func ownedUsageFixtureRows(workspace string, today time.Time) []map[string]any {
	rows := []map[string]any{}
	sums := []usagestore.Totals{{Input: 10, Output: 4, CacheRead: 2, CacheWrite: 1, Total: 17}, {Input: 20, Output: 5, CacheRead: 3, CacheWrite: 2, Total: 30}, {Input: 40, Output: 10, CacheRead: 4, CacheWrite: 1, Total: 55}}
	for i, offset := range []int{0, -10, -40} {
		stamp := today.AddDate(0, 0, offset).UnixMilli()
		model, provider := "qa-model-alpha", "qa-provider-one"
		if i == 1 {
			model, provider = "qa-model-beta", "qa-provider-two"
		}
		cwd := filepath.Join(workspace, "project-a")
		if i == 1 {
			cwd = filepath.Join(workspace, "project-b")
		}
		u := sums[i]
		events := []map[string]any{}
		addEvent := func(kind string, data any, surface bool) {
			event := map[string]any{"seq": len(events), "time": stamp, "type": kind, "data": data}
			if surface {
				event["surfaceOp"] = "append"
			}
			events = append(events, event)
		}
		addEvent("turn/start", map[string]any{"turn": 1}, false)
		addEvent("step/start", map[string]any{"turn": 1, "step": 1}, false)
		addEvent("request/context", map[string]any{"model": model, "provider": provider}, false)
		addEvent("assistant/message", map[string]any{"turn": 1, "step": 1, "message": map[string]any{"id": fmt.Sprintf("qa-usage-native-message-%d", i), "role": "assistant", "content": []any{}, "source": map[string]any{"kind": "model", "provider": provider, "model": model}}, "stream": []any{}, "usage": map[string]any{"inputTokens": u.Input, "outputTokens": u.Output, "cacheReadTokens": u.CacheRead, "cacheWriteTokens": u.CacheWrite}}, true)
		addEvent("step/end", map[string]any{"turn": 1, "step": 1}, false)
		addEvent("turn/end", map[string]any{"turn": 1, "reason": map[string]any{"kind": "completed"}}, false)
		rows = append(rows, map[string]any{"header": map[string]any{"version": 4, "id": fmt.Sprintf("qa-usage-native-%d", i), "createdAt": stamp, "isSeeded": false, "delegationDepth": 0, "cwd": cwd}, "events": events})
	}
	return rows
}

func TestFunctionalOwnedUsage(t *testing.T) {
	stage, base, output, pnpm := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME"), os.Getenv("WHALEISLE_FUNCTIONAL_ROOT"), os.Getenv("WHALEISLE_FUNCTIONAL_OUTPUT"), os.Getenv("WHALEISLE_FUNCTIONAL_PNPM")
	for _, p := range []string{stage, base, output, pnpm} {
		if !filepath.IsAbs(p) {
			t.Fatal("explicit absolute runtime, owned parent, output, and verified local pnpm required")
		}
	}
	root := os.Getenv("WHALEISLE_FUNCTIONAL_USAGE_RESUME")
	resuming := root != ""
	retainedInvalid := false
	expectedFailed := 0
	var e error
	if resuming {
		rel, err := filepath.Rel(base, root)
		if err != nil || !filepath.IsAbs(root) || strings.HasPrefix(rel, "..") || !strings.HasPrefix(filepath.Base(root), "usage-functional-") {
			t.Fatal("resume requires known owned usage root inside explicit parent")
		}
		if raw, err := os.ReadFile(filepath.Join(root, "seed-result.json")); err == nil {
			var priorSeed struct{ Status, Error string }
			if json.Unmarshal(raw, &priorSeed) != nil || priorSeed.Status != "rejected" || (!strings.Contains(priorSeed.Error, "message requires a producer-owned source kind") && !strings.Contains(priorSeed.Error, "lacks an identified message")) {
				t.Fatal("resume only at known original strict-source rejection")
			}
			retainedInvalid = true
			expectedFailed = 1
			if strings.Contains(priorSeed.Error, "lacks an identified message") {
				expectedFailed = 2
			}
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	} else {
		root, e = os.MkdirTemp(base, "usage-functional-")
		if e != nil {
			t.Fatal(e)
		}
	}
	type check struct{ Name, Status, Detail string }
	report := struct {
		Root                 string          `json:"ownedRoot"`
		Runtime              string          `json:"runtime"`
		Scope                string          `json:"scope"`
		Checks               []check         `json:"checks"`
		Seed                 json.RawMessage `json:"seed"`
		LocalPreflight       json.RawMessage `json:"localStrictPreflightBeforeHost"`
		Captures             []string        `json:"softwareBodyCaptures"`
		RPCs                 map[string]int  `json:"auditedHTTPPaths"`
		Blocked              int             `json:"blockedHTTPPaths"`
		PriorQARejections    int             `json:"priorQAOnlyRouteRejections"`
		RetainedInvalid      bool            `json:"retainedOwnedStrictlyRejectedCorpus"`
		HostStopped          bool            `json:"ownedHostStopped"`
		ListenerClosed       bool            `json:"ownedListenerClosed"`
		AgentCreated         bool            `json:"agentCreated"`
		ModelRequests        int             `json:"executedModelRequests"`
		NetworkPacketCapture bool            `json:"networkPacketCapture"`
	}{Root: root, Runtime: stage, Scope: "Default original usage-panel on a fresh owned profile/HOME/APPDATA. Actual empty Host and public SessionPersistence synthetic closed-history fixture, strict SessionQuery validation, original usage projection/aggregation and production Go usage Store. Historical token counts are invented QA data, not real model execution/billing validation. Native body software renders exercise actual returned data, not user-window/IME or original pixel parity. No Agent/prompt/tool/model execution, credentials, accounts, billing edits, log repair, external services or user data. Audited HTTP allowlist covers injected client; session ranking uses its existing separate transport. No packet capture."}
	audit := &usageQATransport{counts: map[string]int{}}
	if resuming {
		var prior struct {
			Root              string         `json:"ownedRoot"`
			Runtime           string         `json:"runtime"`
			Checks            []check        `json:"checks"`
			Captures          []string       `json:"softwareBodyCaptures"`
			RPCs              map[string]int `json:"auditedHTTPPaths"`
			Blocked           int            `json:"blockedHTTPPaths"`
			PriorQARejections int            `json:"priorQAOnlyRouteRejections"`
		}
		raw, err := os.ReadFile(output)
		decoded := err == nil && json.Unmarshal(raw, &prior) == nil
		knownSourceCheckpoint := decoded && ((len(prior.Checks) == 3 && prior.Checks[2].Name == "public storage fixture registration") || (len(prior.Checks) == 4 && prior.Checks[2].Name == "original strict-source negative" && prior.Checks[3].Name == "own metadata fixture inspection" && strings.Contains(prior.Checks[3].Detail, "already-installed")) || (len(prior.Checks) == 5 && prior.Checks[2].Name == "original strict-source negative" && prior.Checks[3].Name == "remove failed owned fixture package" && prior.Checks[4].Name == "public storage fixture registration"))
		if !decoded || prior.Root != root || prior.Runtime != stage || len(prior.Checks) < 2 || prior.Checks[0].Name != "actual empty and loading" || prior.Checks[0].Status != "pass" || (!retainedInvalid && (len(prior.Checks) != 2 || prior.Blocked != 1)) || (retainedInvalid && (!knownSourceCheckpoint || prior.Blocked != 0 || prior.PriorQARejections != 1)) {
			t.Fatal("resume requires retained empty-pass/inspection-route-rejection checkpoint")
		}
		for _, v := range prior.Checks {
			if v.Status == "pass" {
				report.Checks = append(report.Checks, v)
			}
		}
		report.Captures = prior.Captures
		report.PriorQARejections = 1
		for k, v := range prior.RPCs {
			audit.counts[k] = v
		}
		if !retainedInvalid {
			report.Checks = append(report.Checks, check{"corrected QA route guard", "pass", "Preserved first attempt separately. dsh/rpc.go posts Typert methods under /api/, unlike usage plugin control-plane routes. Initial inspection never reached Host. Resumed same owned empty profile without repeating passed empty/loading checks; admitted only original inspect/installBundle/removeBundle routes."})
		} else {
			report.RetainedInvalid = true
			if expectedFailed == 2 {
				report.Checks = append(report.Checks, check{"original identified-message negative", "pass", "Installed Session separately refused qa-usage-valid-0 for missing message.id; source was corrected only at outer validator in that attempted fixture. Both rejected own logs retained untouched. New distinct IDs carry owned message IDs and the required model source with invented qa-provider/model names, and complete original public strict/current catalog preflight runs before any append. These are synthetic historical facts, never executed model requests."})
			}
			if len(report.Checks) == 2 {
				report.Checks = append(report.Checks, check{"original strict-source negative", "pass", "Original current-format SessionQuery refused qa-usage-0 because the synthetic assistant message lacked source.kind. This records the outer-format refusal only; installed Session applies additional identified-message/model-source checks. Prior failure retained separately and malformed own fixture left unchanged for failed-coverage verification; no validator change or old-log rewrite."})
			}
		}
	}
	write := func() {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if e := os.WriteFile(output, append(raw, '\n'), 0600); e != nil {
			t.Errorf("evidence: %v", e)
		}
	}
	add := func(name, detail string) {
		report.Checks = append(report.Checks, check{name, "pass", detail})
		write()
		t.Log(name, detail)
	}
	fail := func(name, detail string) {
		report.Checks = append(report.Checks, check{name, "fail", detail})
		write()
		t.Fatalf("%s: %s", name, detail)
	}
	label := func(e error) string {
		if e == nil {
			return ""
		}
		sum := sha256.Sum256([]byte(e.Error()))
		return "operation-error sha256:" + hex.EncodeToString(sum[:])
	}
	ownedHome, workspace := filepath.Join(root, "owned-home"), filepath.Join(root, "workspace")
	envs := map[string]string{"HOME": ownedHome, "USERPROFILE": ownedHome, "HOMEDRIVE": filepath.VolumeName(ownedHome), "HOMEPATH": strings.TrimPrefix(ownedHome, filepath.VolumeName(ownedHome)), "APPDATA": filepath.Join(root, "appdata"), "LOCALAPPDATA": filepath.Join(root, "local-appdata"), "XDG_CONFIG_HOME": filepath.Join(ownedHome, ".config"), "XDG_DATA_HOME": filepath.Join(ownedHome, ".local", "share"), "XDG_CACHE_HOME": filepath.Join(ownedHome, ".cache"), "DSH_AGENTS_HOME": filepath.Join(ownedHome, ".agents"), "DSH_BUNDLED_SKILL_DIR": filepath.Join(root, "bundled-skills"), "COREPACK_ENABLE_NETWORK": "0", "npm_config_offline": "true", "NO_UPDATE_NOTIFIER": "1"}
	for _, p := range []string{ownedHome, workspace, filepath.Join(workspace, ".git"), envs["APPDATA"], envs["LOCALAPPDATA"], envs["DSH_AGENTS_HOME"], envs["DSH_BUNDLED_SKILL_DIR"]} {
		if e = os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}
	bin := filepath.Join(root, "owned-bin")
	if e = os.MkdirAll(bin, 0700); e != nil {
		t.Fatal(e)
	}
	pnpmBytes, e := os.ReadFile(pnpm)
	if e != nil {
		t.Fatal(e)
	}
	pnpmHash := sha256.Sum256(pnpmBytes)
	if hex.EncodeToString(pnpmHash[:]) != "51cc0ce092691ea0a9cf4f50b7512f8a202911696d291b2322c8be838c2fa829" {
		t.Fatal("pnpm differs from previously verified artifact")
	}
	if e = os.WriteFile(filepath.Join(bin, "pnpm.exe"), pnpmBytes, 0700); e != nil {
		t.Fatal(e)
	}
	emptyConfig := filepath.Join(root, "empty-npmrc")
	if e = os.WriteFile(emptyConfig, nil, 0600); e != nil {
		t.Fatal(e)
	}
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "owned fixture has no registry dependencies", 404)
	}))
	defer registry.Close()
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("npm_config_userconfig", emptyConfig)
	t.Setenv("npm_config_globalconfig", emptyConfig)
	t.Setenv("npm_config_store_dir", filepath.Join(root, "pnpm-store"))
	t.Setenv("npm_config_registry", registry.URL)
	t.Setenv("PNPM_HOME", bin)
	t.Setenv(paths.EnvUserData, filepath.Join(root, "user-data"))
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "user-data", "dsh-home"))
	vendor, home := paths.VendorDir(), paths.DshHome()
	hroot, node := filepath.Join(vendor, "deepseek-harness"), filepath.Join(stage, "node", "node.exe")
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 12, 0, 0, 0, time.UTC)
	rows := ownedUsageFixtureRows(workspace, today)
	encodedRows, _ := json.Marshal(rows)
	for _, p := range []string{filepath.Join(workspace, "project-a"), filepath.Join(workspace, "project-b")} {
		if e = os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	preflight := `import {pathToFileURL} from 'node:url';
const chunks=[];for await(const chunk of process.stdin)chunks.push(chunk);
const root=process.argv[1],rows=JSON.parse(Buffer.concat(chunks).toString('utf8'));
const {Session}=await import(pathToFileURL(root+'/packages/core/session/lib/index.js'));
const {sessionFormatCatalog:catalog}=await import(pathToFileURL(root+'/packages/session/session-format-catalog/lib/index.js'));
for(const row of rows){Session.fromRestore(row.header.id,row.events,row.header,0,'detached');const strict=catalog.createRestore(catalog.encodeCurrentHeader(row.header,0),{recovery:'strict',validation:'current'});for(const event of row.events)strict.decodeRow(catalog.encodeCurrentEvent(event));if(strict.finish().events.length!==6)throw Error('unexpected event count');}
console.log(JSON.stringify({records:rows.length,events:rows.reduce((sum,row)=>sum+row.events.length,0),installedSessionRestore:true,strictCurrentCatalog:true,hostStarted:false,storedRecordsWritten:0}));`
	prectx, precancel := context.WithTimeout(context.Background(), 30*time.Second)
	command := exec.CommandContext(prectx, node, "--input-type=module", "-e", preflight, hroot)
	command.Stdin = strings.NewReader(string(encodedRows))
	command.Dir = workspace
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	preflightOutput, preflightErr := command.Output()
	precancel()
	if preflightErr != nil {
		fail("local complete original decoder preflight", label(preflightErr))
	}
	if !json.Valid(preflightOutput) {
		fail("local complete original decoder preflight", "invalid metadata result")
	}
	report.LocalPreflight = append(json.RawMessage(nil), preflightOutput...)
	add("local complete original decoder preflight", "Same three future storage samples passed installed Session.fromRestore and original public strict/current catalog encode/decode/finish before starting any owned Host or writing records")
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hroot, VendorDir: vendor, Host: "127.0.0.1", Port: 36826, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: hroot, DshHome: home, ProfileDir: paths.WebProfileDir(), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	stop := func() {
		sup.Stop()
		report.HostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, _ := url.Parse(endpoint.Origin)
			conn, e := net.DialTimeout("tcp", u.Host, time.Second)
			report.ListenerClosed = e != nil
			if conn != nil {
				conn.Close()
			}
		}
	}
	defer func() {
		stop()
		audit.mu.Lock()
		report.RPCs = map[string]int{}
		for k, v := range audit.counts {
			report.RPCs[k] = v
		}
		report.Blocked = audit.blocked
		audit.mu.Unlock()
		write()
		if !report.HostStopped || !report.ListenerClosed || report.Blocked != 0 {
			t.Errorf("owned lifecycle or request boundary failed")
		}
	}()
	start := func() *dsh.Client {
		sup.Start()
		deadline := time.Now().Add(195 * time.Second)
		for time.Now().Before(deadline) {
			st := sup.Status()
			if st.State == harness.Ready {
				endpoint = st.Endpoint
				return &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: &http.Client{Timeout: 70 * time.Second, Transport: audit}}
			}
			if st.State == harness.Failed {
				fail("owned Host", label(st.Err))
			}
			time.Sleep(50 * time.Millisecond)
		}
		fail("owned Host", "readiness timeout")
		return nil
	}
	client := start()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	store := usagestore.New()
	store.SetStatsCachePath(filepath.Join(root, "go-usage-cache.json"))
	store.Attach(client, nil)
	defer func() { store.Detach() }()
	wait := func(name string, ready func() bool) {
		deadline := time.Now().Add(70 * time.Second)
		for time.Now().Before(deadline) {
			if ready() {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		fail(name, "condition timeout")
	}
	refresh := func(force bool) usagestore.StatsView {
		store.RefreshStats(ctx, force)
		wait("usage refresh completion", func() bool { return !store.Stats().Loading })
		v := store.Stats()
		if v.Err != "" || v.Overview == nil {
			fail("actual overview", label(errors.New(v.Err)))
		}
		return v
	}
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	captureDir := strings.TrimSuffix(output, filepath.Ext(output))
	if e = os.MkdirAll(captureDir, 0700); e != nil {
		t.Fatal(e)
	}
	capture := func(state, required string) {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Scroll(c).Fill().Padding(20).Children(func() { usageui.View(c, usageui.Model{Store: store, Embedded: true}) })
		}, 780, 900)
		text := strings.Join(tt.Texts(), "\n")
		if !strings.Contains(text, required) {
			fail("actual-data software body "+state, "missing expected state label "+required)
		}
		path := filepath.Join(captureDir, state+".png")
		f, e := os.Create(path)
		if e != nil {
			t.Fatal(e)
		}
		e = png.Encode(f, tt.Image())
		ce := f.Close()
		if e != nil || ce != nil {
			t.Fatal("software capture write failed")
		}
		report.Captures = append(report.Captures, path)
	}
	var v usagestore.StatsView
	if !resuming {
		audit.mu.Lock()
		gate := make(chan struct{})
		audit.gate = gate
		audit.mu.Unlock()
		store.RefreshStats(ctx, true)
		capture("loading", "Loading usage")
		close(gate)
		audit.mu.Lock()
		audit.gate = nil
		audit.mu.Unlock()
		wait("empty overview", func() bool { return !store.Stats().Loading })
		v = store.Stats()
		if v.Err != "" || v.Overview == nil || v.Overview.AllTime.Totals.Total != 0 || v.Overview.Coverage.SessionsTotal != 0 || len(v.Overview.Days) != 182 {
			fail("actual empty corpus", "unexpected fresh statistics")
		}
		capture("empty", "No usage recorded yet")
		add("actual empty and loading", "Default original service returned zero usage and 182 UTC daily buckets; production loading and empty bodies captured without executing anything")
	}
	fixture, name, marker := filepath.Join(root, "fixture-plugin"), "whaleisle-owned-usage-fixture", filepath.Join(root, "seed-result.json")
	if e = os.MkdirAll(fixture, 0700); e != nil {
		t.Fatal(e)
	}
	encodedHome, _ := json.Marshal(home)
	encodedMarker, _ := json.Marshal(marker)
	encodedCatalog, _ := json.Marshal(filepath.Join(hroot, "packages", "session", "session-format-catalog", "lib", "index.js"))
	js := strings.NewReplacer("__HOME__", string(encodedHome), "__MARKER__", string(encodedMarker), "__ROWS__", string(encodedRows), "__CATALOG__", string(encodedCatalog)).Replace(usageSeedPlugin)
	manifest, _ := json.Marshal(map[string]any{"name": name, "version": "1.0.2", "type": "module", "exports": map[string]string{".": "./index.js", "./package.json": "./package.json", "./patch.yml": "./patch.yml"}, "dsh": map[string]any{"bundle": map[string]string{"patch": "./patch.yml"}}})
	for filename, contents := range map[string][]byte{"package.json": manifest, "index.js": []byte(js), "patch.yml": []byte("- insert:\n    - id: owned-usage-fixture\n      name: '" + name + "'\n")} {
		if e = os.WriteFile(filepath.Join(fixture, filename), contents, 0600); e != nil {
			t.Fatal(e)
		}
	}
	pc := manager.Client{Caller: client}
	if retainedInvalid {
		removed, err := pc.Remove(ctx, name)
		if err != nil || removed.Error != nil {
			fail("remove failed owned fixture package", label(err))
		}
		add("remove failed owned fixture package", "Original install had persisted this own package even though applying its invalid seed failed. Removed only that package via original pluginManager API before installing corrected1.0.2; retained owned rejected logs unchanged.")
	}
	inspection, e := pc.Inspect(ctx, fixture, &registry.URL)
	if e != nil || inspection.Status != "accepted" {
		fail("own metadata fixture inspection", label(e)+" "+inspection.Problem)
	}
	change, e := pc.Install(ctx, fixture, manager.InstallOptions{Registry: &registry.URL})
	if e != nil || change.Error != nil || change.Application != "applied" {
		if raw, err := os.ReadFile(marker); err == nil {
			report.Seed = raw
		}
		fail("public storage fixture registration", label(e)+" application="+change.Application)
	}
	report.Seed, e = os.ReadFile(marker)
	if e != nil {
		fail("strict fixture read", label(e))
	}
	var seed struct {
		Status                   string
		Sessions, EventsAppended int
	}
	if json.Unmarshal(report.Seed, &seed) != nil || seed.Status != "accepted" || seed.Sessions != 3 || seed.EventsAppended != 18 {
		fail("strict fixture read", "original storage/query validation did not accept exact corpus")
	}
	add("public inert corpus", "Three explicit synthetic closed-history records, 18 events, original SessionPersistence create/append/flush/close then strict SessionQuery reads; no Agent or execution loop")
	v = refresh(true)
	o := v.Overview
	if o.AllTime.Totals.Total != 102 || o.Totals.Total != 47 || o.AllTime.SessionCount != 3 || o.SessionCount != 2 || len(o.AllTime.ByModel) != 2 || len(o.Providers) != 2 || o.Coverage.SessionsFailed != expectedFailed || o.Coverage.SessionsOk != 3 {
		fail("real projection and aggregation", fmt.Sprintf("all=%.0f recent=%.0f sessions=%d/%d coverage=%d failed=%d", o.AllTime.Totals.Total, o.Totals.Total, o.SessionCount, o.AllTime.SessionCount, o.Coverage.SessionsOk, o.Coverage.SessionsFailed))
	}
	capture("synthetic-data", "Total tokens")
	failedIDs := strings.Join(o.Coverage.FailedSessionIDs, ",")
	if retainedInvalid && (len(o.Coverage.FailedSessionIDs) != expectedFailed || !strings.Contains(failedIDs, "qa-usage-0") || (expectedFailed == 2 && !strings.Contains(failedIDs, "qa-usage-valid-0"))) {
		fail("original failed coverage", "strictly rejected owned record not identified separately")
	}
	add("real projection and dimensions", fmt.Sprintf("Original Host derived all-time102 versus recent-30-day47;3/2 usage sessions,two models/providers,3 readable records and%d known negative records; Go decoded actual payload. Failed IDs never treated as zero-success or repaired.", expectedFailed))
	for _, entry := range []struct {
		days  int
		total float64
	}{{7, 17}, {14, 47}, {30, 47}} {
		store.SelectStatsRecentDays(entry.days)
		v = store.Stats()
		if !v.Range.Active() || v.RangeError != "" || v.Selection().Totals.Total != entry.total {
			fail("UTC date presets", fmt.Sprintf("%d-day selected total mismatch", entry.days))
		}
	}
	day := today.Format("2006-01-02")
	if e = store.SetStatsDateRange(day, day); e != nil || store.Stats().Selection().Totals.Total != 17 {
		fail("inclusive UTC day", label(e))
	}
	capture("selected-day", "Daily model breakdown (selected dates)")
	prior := store.Stats().Range
	if e = store.SetStatsDateRange("2099-12-31", "2099-12-31"); e == nil || store.Stats().Range != prior {
		fail("unavailable date range", "invalid coverage replaced prior selection")
	}
	store.SetStatsRangeDraft("bad-date", day)
	store.ApplyStatsRangeDraft()
	if store.Stats().RangeError == "" {
		fail("date validation", "invalid draft lacked error")
	}
	raw, e := store.ExportStats("json")
	if e != nil || !strings.Contains(string(raw), "Selected available UTC days") || strings.Contains(string(raw), "\"allTime\"") {
		fail("selected export", label(e))
	}
	if _, e = store.ExportStats("sessions"); e == nil {
		fail("date export scope", "dated session ranking incorrectly exportable")
	}
	store.ClearStatsDateRange()
	if store.Stats().Range.Active() || store.Stats().RangeError != "" {
		fail("clear dates", "selection or error retained")
	}
	add("UTC selection and export", "7/14/30-day presets =17/47/47; inclusive today=17; unavailable/malformed dates reject without replacing valid range; selected JSON scopes data and rejects all-time session export; clear resets selection/error")
	for _, kind := range []string{"json", "days", "models", "sessions"} {
		if raw, e = store.ExportStats(kind); e != nil || len(raw) == 0 {
			fail("actual exports "+kind, label(e))
		}
	}
	store.LoadMoreSessions(ctx)
	wait("session ranking", func() bool { return !store.SessionRanking().Loading })
	sr := store.SessionRanking()
	if sr.Err != "" || len(sr.Rows) != 3 || sr.HasMore {
		fail("session ranking", label(errors.New(sr.Err)))
	}
	store.LoadMoreProjects(ctx)
	wait("project ranking", func() bool { return !store.ProjectRanking().Loading })
	pr := store.ProjectRanking()
	if pr.Err != "" || len(pr.Rows) != 2 || pr.HasMore || pr.Rows[0].Totals.Total != 72 {
		fail("project ranking", label(errors.New(pr.Err)))
	}
	add("actual read exports and rankings", "Four production exports nonempty; original sessions.more returned three sessions and projects.more grouped two owned directories with leading72 tokens; no fabricated RPC response")
	v = refresh(true)
	if v.Overview.AllTime.Totals.Total != 102 || v.Overview.Totals.Total != 47 {
		fail("forced refresh", "counts accumulated across rescans")
	}
	add("forced refresh stability", "Second original forced scan retained exactly102 all-time/47 recent, without accumulation")
	change, e = pc.Remove(ctx, name)
	if e != nil || change.Error != nil {
		fail("fixture removal", label(e))
	}
	stop()
	store.RefreshStats(ctx, true)
	wait("offline refresh", func() bool { return !store.Stats().Loading })
	v = store.Stats()
	if v.Err == "" || v.Overview == nil || v.Overview.AllTime.Totals.Total != 102 {
		fail("offline retains success", "error or retained prior data missing")
	}
	capture("offline-error", "Showing previous results; they may be stale.")
	add("actual backend loss", "Owned Host stopped/listener closed; real connection refusal surfaced while last102-token result remained visibly stale")
	store.Detach()
	store = usagestore.New()
	store.SetStatsCachePath(filepath.Join(root, "go-usage-cache.json"))
	store.Attach(client, nil)
	v = store.Stats()
	if !v.Cached || v.Overview == nil || v.Overview.AllTime.Totals.Total != 102 {
		fail("Go cache reopen", "last successful result not restored")
	}
	capture("restored-cache", "Restored cached overview.")
	add("Go cache persistence", "New production Store restored own on-disk version4 overview with Cached=true and102-token previous result while Host offline")
	client = start()
	store.Attach(client, nil)
	v = refresh(true)
	if v.Cached || v.Overview.AllTime.Totals.Total != 102 || v.Overview.Totals.Total != 47 || v.Overview.Coverage.SessionsOk != 3 {
		fail("cold original restart", "stored corpus did not reproduce totals")
	}
	capture("restarted", "Total tokens")
	add("original corpus restart persistence", "After fixture bundle removal and full owned Host restart, original strict persisted-log/projected scan still102/47; fresh success replaced Cached marker. This proves synthetic historical corpus reads, not actual paid/model metering")
}

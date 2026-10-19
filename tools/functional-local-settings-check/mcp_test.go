//go:build functionalacceptance && windows

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

// Only the original MCP configuration gateway can reach this fixture. It does
// not implement tools/call, and the QA caller admits no execution/auth RPCs.
const inertMCPServer = `import fs from 'node:fs';
import readline from 'node:readline';
const [audit, version] = process.argv.slice(2);
const record = (event, extra = {}) => fs.appendFileSync(audit, JSON.stringify({event, pid:process.pid, version, ...extra})+'\n');
record('spawn');
process.on('exit', () => record('close'));
const lines = readline.createInterface({input:process.stdin, terminal:false});
const send = (id,result) => process.stdout.write(JSON.stringify({jsonrpc:'2.0',id,result})+'\n');
lines.on('line', line => {
  const request = JSON.parse(line);
  record('rpc', {method:request.method, cursor:request.params?.cursor !== undefined});
  if (request.id === undefined) return;
  if (request.method === 'initialize') {
    send(request.id, {protocolVersion:request.params.protocolVersion, capabilities:{tools:{listChanged:false}}, serverInfo:{name:'owned-inert-metadata',version}, instructions:'Metadata-only fixture. No tool may be called.'});
  } else if (request.method === 'tools/list') {
    const second = request.params?.cursor === 'metadata-page-2';
    send(request.id, {tools:[{name:second?'metadata_two':'metadata_one',description:'Inert catalog metadata '+version,inputSchema:{type:'object',properties:{},additionalProperties:false},annotations:{readOnlyHint:true,destructiveHint:false,openWorldHint:false}}], ...second?{}:{nextCursor:'metadata-page-2'}});
  } else if (request.method === 'ping') {
    send(request.id, {});
  } else {
    process.stdout.write(JSON.stringify({jsonrpc:'2.0',id:request.id,error:{code:-32601,message:'No executable tools or other methods implemented'}})+'\n');
  }
});
lines.on('close', () => process.exit(0));
process.on('SIGTERM', () => process.exit(0));
`

type mcpRPCAudit struct {
	mu      sync.Mutex
	counts  map[string]int
	blocked int
}
type mcpQACaller struct {
	client *dsh.Client
	audit  *mcpRPCAudit
}

func (c *mcpQACaller) Call(ctx context.Context, method string, args, out any) error {
	allowed := method == "mcpServers/list" || method == "mcpServers/upsert" || method == "mcpServers/delete" || method == "mcpServers/setEnabled" || method == "mcpServers/test"
	c.audit.mu.Lock()
	c.audit.counts[method]++
	if !allowed {
		c.audit.blocked++
	}
	c.audit.mu.Unlock()
	if !allowed {
		return errors.New("only isolated MCP configuration/catalog/test RPCs admitted")
	}
	return c.client.Call(ctx, method, args, out)
}

func TestFunctionalLocalMCP(t *testing.T) {
	stage, base, output := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME"), os.Getenv("WHALEISLE_FUNCTIONAL_ROOT"), os.Getenv("WHALEISLE_FUNCTIONAL_OUTPUT")
	for _, p := range []string{stage, base, output} {
		if !filepath.IsAbs(p) {
			t.Fatal("explicit absolute runtime, owned parent and evidence output required")
		}
	}
	root := os.Getenv("WHALEISLE_FUNCTIONAL_MCP_RESUME")
	resuming := root != ""
	if resuming {
		rel, e := filepath.Rel(base, root)
		if e != nil || !filepath.IsAbs(root) || strings.HasPrefix(rel, "..") || !strings.HasPrefix(filepath.Base(root), "mcp-functional-") {
			t.Fatal("resume must reference known owned MCP fixture inside explicit parent")
		}
	} else {
		var e error
		root, e = os.MkdirTemp(base, "mcp-functional-")
		if e != nil {
			t.Fatal(e)
		}
	}
	type check struct{ Name, Status, Detail string }
	type event struct {
		Event   string `json:"event"`
		PID     int    `json:"pid"`
		Version string `json:"version"`
		Method  string `json:"method"`
		Cursor  bool   `json:"cursor"`
	}
	report := struct {
		Root                 string         `json:"ownedRoot"`
		Runtime              string         `json:"runtime"`
		Scope                string         `json:"scope"`
		Checks               []check        `json:"checks"`
		RPCs                 map[string]int `json:"observedRPCs"`
		Blocked              int            `json:"blockedRPCs"`
		Events               []event        `json:"serverMethodMetadata"`
		HostStopped          bool           `json:"ownedHostStopped"`
		ListenerClosed       bool           `json:"ownedListenerClosed"`
		AllChildrenClosed    bool           `json:"allOwnedServerInstancesClosed"`
		ToolCalls            int            `json:"toolCalls"`
		SessionCreated       bool           `json:"sessionCreated"`
		NetworkPacketCapture bool           `json:"networkPacketCapture"`
		ProcessExitChecks    map[int]string `json:"ownedProcessExitChecks"`
	}{Root: root, Runtime: stage, Scope: "Default web-app original served MCP configuration + file service, production Go mcpconfig Controller. Fresh owned profile/HOME/APPDATA/workspace, staged Node, dependency-free stdio fixture only. Actual initialize + paged tools/list + health + close; no tools/call implementation or invocation, no Session/prompt/model/account/OAuth/external service. Native GUI/IME, HTTP transport, OAuth and third-party MCP compatibility not exercised. RPC allowlist and server method audit are not packet capture."}
	audit := &mcpRPCAudit{counts: map[string]int{}}
	if resuming {
		var prior struct {
			Root    string         `json:"ownedRoot"`
			Runtime string         `json:"runtime"`
			Checks  []check        `json:"checks"`
			RPCs    map[string]int `json:"observedRPCs"`
		}
		raw, e := os.ReadFile(output)
		if e != nil || json.Unmarshal(raw, &prior) != nil || prior.Root != root || prior.Runtime != stage {
			t.Fatal("resume requires matching retained evidence")
		}
		for _, v := range prior.Checks {
			if v.Status == "pass" {
				report.Checks = append(report.Checks, v)
			}
		}
		if len(report.Checks) != 7 {
			t.Fatal("resume requires known seven-pass edited-enabled checkpoint")
		}
		for k, v := range prior.RPCs {
			audit.counts[k] = v
		}
		report.Checks = append(report.Checks, check{"corrected exit observation", "pass", "Resumed only uncompleted persistence checks in same owned profile. Prior failed JS exit-hook predicate retained separately; Windows taskkill and SDK discovery-probe termination require OS exit proof. No product change or repeated successful import/create/test/edit."})
	}
	auditFile := filepath.Join(root, "server-methods.jsonl")
	write := func() {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if e := os.WriteFile(output, append(raw, '\n'), 0600); e != nil {
			t.Errorf("save evidence: %v", e)
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
	errorLabel := func(e error) string {
		if e == nil {
			return ""
		}
		sum := sha256.Sum256([]byte(e.Error()))
		return "operation-error sha256:" + hex.EncodeToString(sum[:])
	}
	ownedHome, workspace := filepath.Join(root, "owned-home"), filepath.Join(root, "workspace")
	envs := map[string]string{"HOME": ownedHome, "USERPROFILE": ownedHome, "HOMEDRIVE": filepath.VolumeName(ownedHome), "HOMEPATH": strings.TrimPrefix(ownedHome, filepath.VolumeName(ownedHome)), "APPDATA": filepath.Join(root, "appdata"), "LOCALAPPDATA": filepath.Join(root, "local-appdata"), "XDG_CONFIG_HOME": filepath.Join(ownedHome, ".config"), "XDG_DATA_HOME": filepath.Join(ownedHome, ".local", "share"), "XDG_CACHE_HOME": filepath.Join(ownedHome, ".cache"), "DSH_AGENTS_HOME": filepath.Join(ownedHome, ".agents"), "DSH_BUNDLED_SKILL_DIR": filepath.Join(root, "bundled-skills"), "COREPACK_ENABLE_NETWORK": "0", "npm_config_offline": "true"}
	for _, p := range []string{ownedHome, workspace, filepath.Join(workspace, ".git"), envs["APPDATA"], envs["LOCALAPPDATA"], envs["DSH_AGENTS_HOME"], envs["DSH_BUNDLED_SKILL_DIR"]} {
		if e := os.MkdirAll(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	for k, v := range envs {
		t.Setenv(k, v)
	}
	t.Setenv(paths.EnvUserData, filepath.Join(root, "user-data"))
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "user-data", "dsh-home"))
	vendor, home := paths.VendorDir(), paths.DshHome()
	hroot, node := filepath.Join(vendor, "deepseek-harness"), filepath.Join(stage, "node", "node.exe")
	script := filepath.Join(root, "inert-server.mjs")
	if e := os.WriteFile(script, []byte(inertMCPServer), 0600); e != nil {
		t.Fatal(e)
	}
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hroot, VendorDir: vendor, Host: "127.0.0.1", Port: 36823, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: hroot, DshHome: home, ProfileDir: paths.WebProfileDir(), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	var controller *mcpconfig.Controller
	processHandles := map[int]windows.Handle{}
	report.ProcessExitChecks = map[int]string{}
	readEvents := func() []event {
		raw, e := os.ReadFile(auditFile)
		if os.IsNotExist(e) {
			return nil
		}
		if e != nil {
			t.Fatal(e)
		}
		var events []event
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			if line == "" {
				continue
			}
			var v event
			if e = json.Unmarshal([]byte(line), &v); e != nil {
				t.Fatal(e)
			}
			events = append(events, v)
		}
		return events
	}
	// Hold read/synchronize-only handles while owned children are live. Windows
	// taskkill /T and SDK discovery-probe termination can bypass Node exit hooks;
	// the signaled process handle, rather than an absent JS callback, proves exit.
	captureProcesses := func() {
		for _, v := range readEvents() {
			if v.Event != "spawn" || processHandles[v.PID] != 0 || report.ProcessExitChecks[v.PID] != "" {
				continue
			}
			h, e := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(v.PID))
			if errors.Is(e, windows.ERROR_INVALID_PARAMETER) {
				report.ProcessExitChecks[v.PID] = "owned audited PID no longer present before observation"
				continue
			}
			if e != nil {
				fail("observe owned child exit", errorLabel(e))
			}
			processHandles[v.PID] = h
		}
	}
	allClosed := func() bool {
		captureProcesses()
		closed := true
		for pid, handle := range processHandles {
			state, e := windows.WaitForSingleObject(handle, 0)
			if e != nil {
				fail("observe owned child exit", errorLabel(e))
			}
			if state == windows.WAIT_OBJECT_0 {
				report.ProcessExitChecks[pid] = "retained own process handle signaled"
			} else {
				closed = false
			}
		}
		return closed
	}
	stop := func() {
		if controller != nil {
			controller.StopHealthRefresh()
		}
		captureProcesses()
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
		report.Events = readEvents()
		report.AllChildrenClosed = allClosed()
		for _, handle := range processHandles {
			_ = windows.CloseHandle(handle)
		}
		for _, v := range report.Events {
			if v.Method == "tools/call" {
				report.ToolCalls++
			}
		}
		audit.mu.Lock()
		report.RPCs = map[string]int{}
		for k, v := range audit.counts {
			report.RPCs[k] = v
		}
		report.Blocked = audit.blocked
		audit.mu.Unlock()
		write()
		if !report.HostStopped || !report.ListenerClosed || !report.AllChildrenClosed || report.ToolCalls != 0 || report.Blocked != 0 {
			t.Errorf("owned lifecycle or RPC boundary failed")
		}
	}()
	start := func() *mcpQACaller {
		sup.Start()
		deadline := time.Now().Add(195 * time.Second)
		for time.Now().Before(deadline) {
			s := sup.Status()
			if s.State == harness.Ready {
				endpoint = s.Endpoint
				controller = &mcpconfig.Controller{}
				return &mcpQACaller{client: &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: &http.Client{Timeout: 45 * time.Second}}, audit: audit}
			}
			if s.State == harness.Failed {
				fail("owned Host readiness", errorLabel(s.Err))
			}
			time.Sleep(50 * time.Millisecond)
		}
		fail("owned Host readiness", "timeout")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	caller := start()
	refresh := func() {
		controller.Refresh(ctx, caller)
		if s := controller.Snapshot(); s.Err != "" || !s.Ready {
			fail("configuration read", errorLabel(errors.New(s.Err)))
		}
	}
	row := func() (api.McpServerSnapshotServersItem, bool) {
		for _, r := range controller.Snapshot().Rows {
			if r.Id == "owned_mcp_fixture" {
				return r, true
			}
		}
		return api.McpServerSnapshotServersItem{}, false
	}
	clean := func(action string) {
		s := controller.Snapshot()
		if s.Err != "" || s.Busy {
			fail(action, errorLabel(errors.New(s.Err)))
		}
	}
	wait := func(name string, ready func() bool) {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if ready() {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		fail(name, "condition not reached")
	}
	health := func(namespace string) {
		wait("connected health and qualified tools", func() bool {
			refresh()
			r, ok := row()
			return ok && r.Enabled && r.Connection != nil && r.Connection.Health == "connected" && len(r.Connection.Tools) == 2 && strings.Join(r.Connection.Tools, ",") == "mcp__"+namespace+"__metadata_one,mcp__"+namespace+"__metadata_two"
		})
		captureProcesses()
	}
	var r api.McpServerSnapshotServersItem
	var ok bool
	var s mcpconfig.Snapshot
	nextArgs, _ := json.Marshal([]string{script, auditFile, "2"})
	if !resuming {
		refresh()
		if len(controller.Snapshot().Rows) != 0 {
			fail("fresh profile", "unexpected inherited MCP record")
		}
		add("default served catalog", "Original MCP configuration gateway available with empty owned catalog; no experimental bundle or user config enabled")
		args, _ := json.Marshal([]string{script, auditFile, "1"})
		input, _ := json.Marshal(map[string]any{"mcpServers": map[string]any{"owned_mcp_fixture": map[string]any{"command": node, "args": []string{script, auditFile, "1"}, "cwd": workspace, "enabled": false}}})
		controller.SetImportInput(string(input), "")
		controller.ParseImport()
		s = controller.Snapshot()
		if !s.Import.Ready || len(s.Import.Candidates) != 1 || s.Import.Candidates[0].Enabled {
			fail("import preview", "expected one disabled stdio candidate")
		}
		controller.SelectImport(0)
		s = controller.Snapshot()
		controller.StageImportExpected(s.Import.Token, 0)
		s = controller.Snapshot()
		if s.Form == nil || s.Form.Enabled || s.Form.ArgsJSON != string(args) {
			fail("import draft", "staged form mismatch")
		}
		refresh()
		if len(controller.Snapshot().Rows) != 0 || len(readEvents()) != 0 {
			fail("import preview isolation", "preview mutated Host or spawned server")
		}
		add("import preview/staging", "Legacy mcpServers import parsed locally and staged one disabled form; no Host record or child before Save")
		s = controller.Snapshot()
		controller.SaveFormExpected(ctx, caller, *s.Form, s.EditingID)
		clean("create disabled record")
		r, ok = row()
		if !ok || r.Enabled || !r.Writable || len(readEvents()) != 0 {
			fail("create disabled record", "saved catalog or child state mismatch")
		}
		add("create disabled record", "Production form Save wrote managed stdio record without spawning it")
		controller.RowAction(ctx, caller, r.Id, "test")
		clean("explicit connection test")
		s = controller.Snapshot()
		diagnostic, ok := s.Tests[r.Id]
		if !ok || diagnostic.Err != "" || !diagnostic.Result.Ok || diagnostic.Result.ToolCount != 2 || strings.Join(diagnostic.Result.ToolNames, ",") != "metadata_one,metadata_two" {
			fail("explicit connection test", "original paginated probe result mismatch")
		}
		wait("probe closes before completion", allClosed)
		r, _ = row()
		if r.Enabled {
			fail("explicit connection test", "probe unexpectedly enabled managed record")
		}
		add("explicit connection test", "Original SDK spawned Node, initialized, walked both tools/list pages, returned two raw names and closed temporary transport; managed record remains disabled")
		controller.SetEnabled(ctx, caller, r.Id, true)
		clean("enable")
		health("owned_mcp_fixture")
		add("enable and live catalog", "Original mounted MCP client connected and published two qualified tool names without invoking a tool")
		notify := make(chan struct{}, 1)
		controller.ObserveHealthRefresh(true, caller, func() {
			select {
			case notify <- struct{}{}:
			default:
			}
		})
		select {
		case <-notify:
		case <-time.After(4 * time.Second):
			fail("visible health refresh", "no real background list completion")
		}
		controller.ObserveHealthRefresh(false, caller, nil)
		controller.StopHealthRefresh()
		add("visible health lifecycle", "Production 2-second refresh completed an actual list, then hidden/close canceled its loop; no test/retry/authorize callback used")
		controller.Edit(r.Id)
		controller.OpenForm()
		s = controller.Snapshot()
		if s.Form == nil {
			fail("edit form", "missing managed form")
		}
		form := *s.Form
		form.ServerName = "owned_mcp_edited"
		form.ArgsJSON = string(nextArgs)
		controller.SetForm(form)
		s = controller.Snapshot()
		controller.SaveFormExpected(ctx, caller, *s.Form, s.EditingID)
		clean("edit record")
		health("owned_mcp_edited")
		if _, ok = controller.Snapshot().Tests[r.Id]; ok {
			fail("edit diagnostic invalidation", "old-spec probe diagnostic retained")
		}
		add("edit and reconcile", "Production form changed namespace and version args; live tools moved to edited namespace, prior spec diagnostic cleared")
		stop()
		wait("first Host shutdown closes child", allClosed)
		caller = start()
	}
	refresh()
	health("owned_mcp_edited")
	r, _ = row()
	var persisted map[string]json.RawMessage
	_ = json.Unmarshal(r.Spec, &persisted)
	if string(persisted["args"]) != string(nextArgs) {
		var saved []string
		_ = json.Unmarshal(persisted["args"], &saved)
		if strings.Join(saved, "|") != strings.Join([]string{script, auditFile, "2"}, "|") {
			fail("enabled restart persistence", "edited args not retained")
		}
	}
	add("enabled restart persistence", "Fresh owned Host restart reopened edited managed spec and reconnected original client automatically")
	controller.SetEnabled(ctx, caller, r.Id, false)
	clean("disable")
	wait("disable disposes child", allClosed)
	stop()
	caller = start()
	refresh()
	r, ok = row()
	if !ok || r.Enabled || !allClosed() {
		fail("disabled restart persistence", "disabled record reactivated or disappeared")
	}
	add("disable and restart persistence", "Disabling disposed own child; disabled edited record survived restart without new child")
	controller.RequestDelete(r.Id)
	controller.CancelDelete()
	if controller.Snapshot().DeletingID != "" {
		fail("delete cancellation", "confirmation retained")
	}
	refresh()
	if _, ok = row(); !ok {
		fail("delete cancellation", "cancel deleted record")
	}
	controller.RequestDelete(r.Id)
	controller.Delete(ctx, caller)
	clean("delete confirmation")
	if _, ok = row(); ok {
		fail("delete confirmation", "managed record retained")
	}
	stop()
	caller = start()
	refresh()
	if len(controller.Snapshot().Rows) != 0 || !allClosed() {
		fail("delete restart persistence", "record or child reappeared")
	}
	add("delete confirmation and restart persistence", "Cancel retained own record; confirmed production deletion removed it; another restart kept catalog empty and no child")
	methodCounts := map[string]int{}
	versions := map[string]int{}
	for _, v := range readEvents() {
		if v.Event == "spawn" {
			versions[v.Version]++
		}
		if v.Event == "rpc" {
			methodCounts[v.Method]++
		}
	}
	if methodCounts["initialize"] != 4 || methodCounts["notifications/initialized"] != 4 || methodCounts["tools/list"] != 8 || methodCounts["server/discover"] != 3 || methodCounts["tools/call"] != 0 || versions["1"] != 3 || versions["2"] != 4 {
		fail("server method audit", "unexpected instance/version/handshake/page count")
	}
	for method := range methodCounts {
		if method != "initialize" && method != "notifications/initialized" && method != "tools/list" && method != "ping" && method != "server/discover" {
			fail("server method audit", "unexpected method "+method)
		}
	}
	add("method and cleanup audit", "Four initialize/initialized handshakes (one isolated test, live v1, edited v2, restarted v2), eight tools/list pages, plus three original auto-negotiation server/discover probes rejected with method-not-found before legacy fallback. Seven owned processes, zero tools/call; each exit proved by retained Windows process handle or already-absent audited PID, not JS exit-hook assumption")
}

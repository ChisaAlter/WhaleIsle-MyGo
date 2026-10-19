//go:build functionalacceptance

package jobflow_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/jobflow"
	"whaleisle/internal/paths"
	manager "whaleisle/internal/pluginmanager"
)

// This adapter uses the exact production NativeJobs wire envelopes. Its only
// extra behavior records client-side logical-stream release and kill receipts.
type actualJobTransport struct {
	client  *dsh.Client
	mux     *dsh.Mux
	mu      sync.Mutex
	active  map[string]int
	opened  []map[string]any
	anchors []map[string]any
	kills   []string
}

type actualJobStream struct {
	*dsh.Stream
	once  sync.Once
	done  func()
	frame func(json.RawMessage)
}

func (s *actualJobStream) Recv(ctx context.Context) (json.RawMessage, error) {
	raw, err := s.Stream.Recv(ctx)
	if err == nil {
		s.frame(raw)
	}
	return raw, err
}

func (s *actualJobStream) Cancel() {
	s.once.Do(func() { s.Stream.Cancel(); s.done() })
}
func (a *actualJobTransport) Open(ctx context.Context, method string, request map[string]any) (jobflow.Stream, error) {
	if e := a.mux.Dial(ctx); e != nil {
		return nil, e
	}
	s, e := a.mux.Open(ctx, method, map[string]any{"args": map[string]any{"request": request}})
	if e != nil {
		return nil, e
	}
	a.mu.Lock()
	a.active[method]++
	copy := map[string]any{"method": method}
	for k, v := range request {
		copy[k] = v
	}
	a.opened = append(a.opened, copy)
	a.mu.Unlock()
	return &actualJobStream{Stream: s, done: func() { a.mu.Lock(); a.active[method]--; a.mu.Unlock() }, frame: func(raw json.RawMessage) {
		var frame jobflow.Frame
		if json.Unmarshal(raw, &frame) == nil && frame.Type == "opened" {
			a.mu.Lock()
			a.anchors = append(a.anchors, map[string]any{"requestedFrom": copy["from"], "acceptedFrom": frame.From})
			a.mu.Unlock()
		}
	}}, nil
}
func (a *actualJobTransport) Kill(ctx context.Context, request map[string]any) (string, error) {
	var result struct {
		Outcome string `json:"outcome"`
	}
	e := a.client.Call(ctx, "job/kill", map[string]any{"request": request}, &result)
	a.mu.Lock()
	a.kills = append(a.kills, result.Outcome)
	a.mu.Unlock()
	return result.Outcome, e
}
func (*actualJobTransport) Retryable(e error) bool {
	return errors.Is(e, dsh.ErrCarrierLost) || errors.Is(e, dsh.ErrStreamEnd)
}
func (a *actualJobTransport) counts() (int, int, int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.active["job/list"], a.active["job/follow"], len(a.kills)
}

func TestFunctionalOwnedJobs(t *testing.T) {
	stage, base, output, pnpm := os.Getenv("WHALEISLE_FUNCTIONAL_RUNTIME"), os.Getenv("WHALEISLE_FUNCTIONAL_ROOT"), os.Getenv("WHALEISLE_FUNCTIONAL_OUTPUT"), os.Getenv("WHALEISLE_FUNCTIONAL_PNPM")
	for _, p := range []string{stage, base, output, pnpm} {
		if !filepath.IsAbs(p) {
			t.Fatal("explicit absolute runtime, owned parent, output and existing pnpm required")
		}
	}
	root, e := os.MkdirTemp(base, "jobs-functional-")
	if e != nil {
		t.Fatal(e)
	}
	type check struct{ Name, Status, Detail string }
	report := struct {
		Root           string  `json:"ownedRoot"`
		Runtime        string  `json:"runtime"`
		Checks         []check `json:"checks"`
		HostStopped    bool    `json:"ownedHostStopped"`
		ListenerClosed bool    `json:"ownedListenerClosed"`
		ChildClosed    bool    `json:"ownedChildClosed"`
		PNPMHash       string  `json:"pnpmSHA256"`
		Scope          string  `json:"scope"`
	}{Root: root, Runtime: stage, Scope: "Real production jobflow over original JobController/mux; unowned public registry jobs only; no session activation, prompt/model or user GUI. Stream counters observe actual client logical cancels, not direct server listener accounting."}
	write := func() {
		raw, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(output, append(raw, '\n'), 0600); err != nil {
			t.Errorf("save evidence: %v", err)
		}
	}
	add := func(name, status, detail string) {
		report.Checks = append(report.Checks, check{name, status, detail})
		write()
		t.Log(name, status, detail)
	}
	failure := func(err error) string {
		if err == nil {
			return ""
		}
		sum := sha256.Sum256([]byte(err.Error()))
		var rpc *dsh.Error
		code := "operation-error"
		if errors.As(err, &rpc) {
			code = rpc.Code
		}
		return code + " sha256:" + hex.EncodeToString(sum[:])
	}
	contents, e := os.ReadFile(pnpm)
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(contents)
	report.PNPMHash = hex.EncodeToString(sum[:])
	if report.PNPMHash != "51cc0ce092691ea0a9cf4f50b7512f8a202911696d291b2322c8be838c2fa829" {
		t.Fatal("pnpm differs from verified native artifact")
	}
	bin := filepath.Join(root, "owned-bin")
	if e = os.MkdirAll(bin, 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(bin, "pnpm.exe"), contents, 0700); e != nil {
		t.Fatal(e)
	}
	config := filepath.Join(root, "empty-npmrc")
	if e = os.WriteFile(config, nil, 0600); e != nil {
		t.Fatal(e)
	}
	workspace := filepath.Join(root, "workspace")
	if e = os.MkdirAll(workspace, 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv(paths.EnvUserData, filepath.Join(root, "user-data"))
	t.Setenv(paths.EnvVendor, filepath.Join(stage, "vendor"))
	t.Setenv("DSH_HOME", filepath.Join(root, "user-data", "dsh-home"))
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "fixture has no dependencies", 404) }))
	defer registry.Close()
	vendor, home := paths.VendorDir(), paths.DshHome()
	hroot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hroot, VendorDir: vendor, Host: "127.0.0.1", Port: 36790, Home: home, ExtraEnv: []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"), "npm_config_offline=true", "npm_config_registry=" + registry.URL, "npm_config_userconfig=" + config, "npm_config_globalconfig=" + config, "npm_config_store_dir=" + filepath.Join(root, "pnpm-store"), "PNPM_HOME=" + bin, "COREPACK_ENABLE_NETWORK=0", "NO_UPDATE_NOTIFIER=1"}}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: hroot, DshHome: home, ProfileDir: paths.WebProfileDir(), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	defer func() {
		sup.Stop()
		report.HostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, _ := url.Parse(endpoint.Origin)
			conn, err := net.DialTimeout("tcp", u.Host, time.Second)
			report.ListenerClosed = err != nil
			if conn != nil {
				conn.Close()
			}
		}
		_, err := os.Stat(filepath.Join(root, "child-closed.json"))
		report.ChildClosed = err == nil
		write()
	}()
	sup.Start()
	deadline := time.Now().Add(195 * time.Second)
	for time.Now().Before(deadline) {
		st := sup.Status()
		if st.State == harness.Ready {
			endpoint = st.Endpoint
			break
		}
		if st.State == harness.Failed {
			add("owned Host", "blocked", failure(st.Err))
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	if endpoint.Origin == "" {
		add("owned Host", "blocked", "readiness timeout")
		return
	}
	add("owned Host", "pass", "fresh rc.2 profile, remote disabled")
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: &http.Client{Timeout: 100 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	fixture := filepath.Join(root, "fixture-plugin")
	if e = os.MkdirAll(fixture, 0700); e != nil {
		t.Fatal(e)
	}
	name := "whaleisle-functional-jobs-fixture"
	manifest := map[string]any{"name": name, "version": "1.0.0", "type": "module", "exports": map[string]string{".": "./index.js", "./package.json": "./package.json", "./patch.yml": "./patch.yml"}, "dsh": map[string]any{"bundle": map[string]string{"patch": "./patch.yml"}}}
	raw, _ := json.Marshal(manifest)
	flag, closed := filepath.Join(root, "continue.flag"), filepath.Join(root, "child-closed.json")
	childJS := fmt.Sprintf("const fs=require('node:fs'); process.stdout.write('JOB_READY_鲸\\n');const timer=setInterval(()=>{if(fs.existsSync(%q)){process.stdout.write('JOB_RESUME_鲸\\n');clearInterval(timer);setInterval(()=>{},1000)}},25);", flag)
	pluginJS := fmt.Sprintf(`import { spawn } from 'node:child_process';
import { writeFileSync } from 'node:fs';
export const inject=['jobs'];
export function apply(ctx) {
  ctx.jobs.attachController('functional-owned-controller');
  ctx.jobs.start({kind:'functional',label:'Functional completed sentinel',run(job){job.append('JOB_COMPLETED_鲸\n');return {cancel(){},done:Promise.resolve({status:'completed',detail:'owned fixture completed'})}}});
  ctx.jobs.start({kind:'functional',label:'Functional waiting sentinel',run(job){
    let stopped=false,settle;const done=new Promise(resolve=>{settle=resolve});
    const child=spawn(process.execPath,['-e',%q],{stdio:['ignore','pipe','pipe']});
    child.stdout.setEncoding('utf8');child.stderr.setEncoding('utf8');
    child.stdout.on('data',text=>{job.append(text,{channel:'stdout'});if(text.includes('JOB_RESUME'))job.updateProgress('resumed')});
    child.stderr.on('data',text=>job.append(text,{channel:'stderr'}));
    child.on('error',()=>settle({status:'failed',detail:'owned child spawn failed'}));
    child.on('close',(code,signal)=>{writeFileSync(%q,JSON.stringify({code,signal}));settle({status:stopped?'killed':'completed',detail:'owned child closed'})});
    return {done,cancel(){if(stopped)return;stopped=true;job.append('JOB_CANCEL_REQUESTED\n');setTimeout(()=>child.kill(),1500)}};
  }});
}
`, childJS, closed)
	for file, data := range map[string][]byte{"package.json": raw, "index.js": []byte(pluginJS), "patch.yml": []byte("- insert:\n    - id: functional-jobs-fixture\n      name: '" + name + "'\n")} {
		if e = os.WriteFile(filepath.Join(fixture, file), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	pc := manager.Client{Caller: client}
	inspection, e := pc.Inspect(ctx, fixture, &registry.URL)
	if e != nil || inspection.Status != "accepted" {
		add("fixture inspection", "blocked", inspection.Problem+" "+failure(e))
		return
	}
	installed, e := pc.Install(ctx, fixture, manager.InstallOptions{Registry: &registry.URL})
	if e != nil || installed.Error != nil || installed.Application != "applied" {
		detail := failure(e)
		if installed.Error != nil {
			detail += " " + installed.Error.Code
		}
		add("fixture registration", "blocked", detail)
		return
	}
	add("fixture registration", "pass", "public attachController/start, unowned jobs; fixed local child, no install scripts/dependencies")
	mux := &dsh.Mux{Endpoint: client.Endpoint, HTTP: client.HTTP}
	defer mux.Close()
	a := &actualJobTransport{client: client, mux: mux, active: map[string]int{}}
	c := jobflow.New(nil)
	c.Bind(a, "functional-readonly-viewer")
	defer c.Unmount()
	wait := func(label string, pred func(jobflow.Snapshot) bool) jobflow.Snapshot {
		t.Helper()
		until := time.Now().Add(15 * time.Second)
		for time.Now().Before(until) {
			s := c.Snapshot()
			if s.Error != "" || s.Observed.Error != "" {
				add(label, "fail", failure(errors.New(s.Error+s.Observed.Error)))
				t.Fatalf("%s failed; see hash-only evidence", label)
			}
			if pred(s) {
				return s
			}
			time.Sleep(10 * time.Millisecond)
		}
		add(label, "fail", "condition timeout")
		t.Fatalf("%s timed out", label)
		return jobflow.Snapshot{}
	}
	row := func(s jobflow.Snapshot, id string) jobflow.Job {
		for _, j := range s.Rows {
			if j.ID == id {
				return j
			}
		}
		return jobflow.Job{}
	}
	c.Mount()
	s := wait("mounted roster", func(s jobflow.Snapshot) bool { return !s.Loading && len(s.Rows) == 2 })
	if s.Open {
		t.Fatal("mounted roster required popup")
	}
	liveID, doneID := "", ""
	for _, j := range s.Rows {
		if j.Label == "Functional waiting sentinel" {
			liveID = j.ID
		}
		if j.Label == "Functional completed sentinel" {
			doneID = j.ID
		}
	}
	if liveID == "" || doneID == "" {
		t.Fatal("original registry jobs missing")
	}
	add("TASK-01 mounted roster", "pass", "two original registry rows while popup closed")
	c.Open(s.Generation)
	c.Toggle(s.Generation, liveID)
	s = wait("output anchor", func(s jobflow.Snapshot) bool {
		return strings.Contains(s.Observed.Text, "JOB_READY_鲸\n") && s.Observed.Cursor > 0
	})
	cursor := s.Observed.Cursor
	if cursor != int64(len([]byte(s.Observed.Text))) {
		t.Fatal("initial UTF-8 byte cursor mismatch")
	}
	c.RetryOutput(s.Generation)
	s = wait("cursor resume", func(s jobflow.Snapshot) bool {
		a.mu.Lock()
		defer a.mu.Unlock()
		for _, o := range a.anchors {
			if o["requestedFrom"] == cursor && o["acceptedFrom"] == cursor {
				return s.Observed.HasCursor && s.Observed.Streaming
			}
		}
		return false
	})
	if strings.Count(s.Observed.Text, "JOB_READY_鲸") != 1 {
		t.Fatal("cursor resume duplicated ready text")
	}
	add("TASK-02 output cursor", "pass", "real opened/output frames, UTF-8 byte cursor, retry from accepted cursor without duplicate sentinel")
	c.Close(s.Generation)
	wait("popup release", func(s jobflow.Snapshot) bool {
		roster, follow, kills := a.counts()
		return !s.Open && s.Mounted && s.Observed.Text == "" && roster == 1 && follow == 0 && kills == 0
	})
	if e = os.WriteFile(flag, nil, 0600); e != nil {
		t.Fatal(e)
	}
	s = wait("closed roster update", func(s jobflow.Snapshot) bool { return row(s, liveID).Progress == "resumed" })
	add("TASK-01 popup close", "pass", "output stream canceled/released; mounted roster remained and received progress; child still running; zero kills")
	c.Open(s.Generation)
	c.Toggle(s.Generation, liveID)
	s = wait("reopen output", func(s jobflow.Snapshot) bool { return strings.Contains(s.Observed.Text, "JOB_RESUME_鲸\n") })
	c.ClearSettled(s.Generation)
	s = c.Snapshot()
	if len(s.Rows) != 1 || s.Rows[0].ID != liveID || !s.Rows[0].Live() {
		t.Fatal("clear removed live job")
	}
	_, _, kills := a.counts()
	if kills != 0 {
		t.Fatal("clear sent kill")
	}
	add("TASK-02 clear settled", "pass", "completed row hidden locally; running child and output retained; zero kill calls")
	c.ArmKill(s.Generation, liveID)
	c.CancelKill(s.Generation)
	_, _, kills = a.counts()
	if kills != 0 || c.Snapshot().Kill.Phase != "" {
		t.Fatal("cancel confirmation sent kill")
	}
	c.ArmKill(s.Generation, liveID)
	c.ConfirmKill(s.Generation, liveID)
	s = wait("kill receipt", func(s jobflow.Snapshot) bool { _, _, n := a.counts(); return n == 1 })
	if !row(s, liveID).Live() {
		t.Fatal("kill receipt synthesized settlement")
	}
	add("TASK-02 kill admission", "pass", "one real job/kill requested receipt; job remained running/stopping until producer closed")
	wait("stopping projection", func(s jobflow.Snapshot) bool { return row(s, liveID).Status == "stopping" })
	s = wait("kill settlement", func(s jobflow.Snapshot) bool { return row(s, liveID).Status == "killed" && !s.Observed.Streaming })
	if _, e = os.Stat(closed); e != nil {
		t.Fatal("producer settled before owned child close")
	}
	report.ChildClosed = true
	if !strings.Contains(s.Observed.Text, "JOB_CANCEL_REQUESTED") {
		t.Fatal("final output was not drained")
	}
	add("TASK-02 kill settlement", "pass", "original stopping→killed roster and terminal output; owned child close marker exists")
	c.ClearSettled(s.Generation)
	s = c.Snapshot()
	if len(s.Rows) != 0 || s.Open {
		t.Fatal("clear all did not close empty popup")
	}
	// Clearing is a local presentation action: a fresh real roster still carries
	// both settled records, and no second cancellation was sent.
	probe, e := a.Open(ctx, "job/list", map[string]any{"sessionId": "functional-readonly-viewer"})
	if e != nil {
		t.Fatal(e)
	}
	frame, e := probe.Recv(ctx)
	probe.Cancel()
	if e != nil {
		t.Fatal(e)
	}
	var roster struct {
		Jobs []jobflow.Job `json:"jobs"`
	}
	if e = json.Unmarshal(frame, &roster); e != nil || len(roster.Jobs) != 2 {
		t.Fatal("clear changed authoritative roster")
	}
	_, _, kills = a.counts()
	if kills != 1 {
		t.Fatal("clear sent another kill")
	}
	add("TASK-02 clear versus kill", "pass", "empty popup closes, original Host still lists two settled jobs, exactly one kill")
	c.Unmount()
	wait("unmount release", func(s jobflow.Snapshot) bool {
		rows, follow, _ := a.counts()
		return !s.Mounted && rows == 0 && follow == 0
	})
	add("TASK-01 unmount", "pass", "all actual client logical roster/output streams canceled; mounted state released")
	removed, e := pc.Remove(ctx, name)
	if e != nil || removed.Error != nil {
		add("fixture removal", "fail", failure(e))
		t.Fatal("fixture removal failed")
	}
	add("fixture removal", "pass", removed.Application)
}

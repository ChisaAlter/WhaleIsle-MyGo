// core-host-check is an opt-in, isolated real-Host integration check. It never
// reads user profiles/credentials. The default check sends no prompt; the
// explicit -prompt-flow mode uses only an owned deterministic loopback model.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

const fixtureText = "Core host isolated persistence fixture. No model request."

type check struct {
	Name          string `json:"name"`
	Pass          bool   `json:"pass"`
	Count         int    `json:"count"`
	SHA256        string `json:"sha256,omitempty"`
	FailureClass  string `json:"failureClass,omitempty"`
	FailureSHA256 string `json:"failureSHA256,omitempty"`
	Detail        string `json:"detail,omitempty"`
}
type report struct {
	Started             string           `json:"started"`
	Checks              []check          `json:"checks"`
	StartupLogKinds     []string         `json:"startupLogKinds,omitempty"`
	OwnedHostStopped    bool             `json:"ownedHostStopped"`
	OwnedListenerClosed bool             `json:"ownedListenerClosed"`
	ModelRequests       int              `json:"modelRequests"`
	DurationMS          int64            `json:"durationMS"`
	OwnedRunRoot        string           `json:"ownedRunRoot,omitempty"`
	SimulatedModel      bool             `json:"simulatedModel,omitempty"`
	OwnedModelStopped   bool             `json:"ownedModelStopped,omitempty"`
	ModelListenerClosed bool             `json:"modelListenerClosed,omitempty"`
	Failure             string           `json:"failure,omitempty"`
	RealModel           *realModelReport `json:"realModel,omitempty"`
}

func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func failure(err error) (string, string) {
	var rpc *dsh.Error
	class := "operation-error"
	if errors.As(err, &rpc) {
		class = "rpc-error:" + rpc.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		class = "deadline-exceeded"
	}
	if strings.Contains(err.Error(), "did not become ready within") {
		class = "host-ready-timeout"
	}
	return class, digest([]byte(err.Error()))
}

func main() {
	stage := flag.String("stage", "", "verified staged runtime directory")
	base := flag.String("root", "", "existing workspace directory for a fresh isolated run")
	out := flag.String("out", "", "JSON report destination")
	resumeRun := flag.String("resume-run", "", "owned isolated run to continue after fixture-only repair")
	contributionsOnly := flag.Bool("native-contributions-only", false, "read only the actual fixed native contribution graph in a fresh owned Host")
	promptFlow := flag.Bool("prompt-flow", false, "exercise real Host/Go Store with an owned deterministic loopback model; no external model API")
	realFlow := flag.Bool("real-model-flow", false, "two guarded official deepseek-flash synthetic turns; requires an explicit secret file and cumulative ledger")
	realPreflight := flag.Bool("real-model-preflight", false, "capture and validate the real Host's final payload, then refuse it before credentials, reservation or network")
	realSecret := flag.String("real-model-secret-file", "", "explicit T3-provided secret file; never uses an existing environment key")
	realLedger := flag.String("real-model-ledger", "", "persistent cumulative microCNY ledger; never reset or automatically recovered")
	flag.Parse()
	if *stage == "" || *base == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "-stage, -root and -out are required")
		os.Exit(2)
	}
	started := time.Now()
	r := report{Started: started.UTC().Format(time.RFC3339)}
	var err error
	if *realFlow || *realPreflight || *realSecret != "" || *realLedger != "" {
		if *promptFlow || *contributionsOnly || *resumeRun != "" {
			err = errors.New("real model modes require their own fresh isolated run")
		} else {
			err = runRealModelFlow(*stage, *base, *realSecret, *realLedger, *realFlow, *realPreflight, &r)
		}
	} else if *promptFlow {
		if *resumeRun != "" || *contributionsOnly {
			err = errors.New("prompt-flow requires its own fresh isolated run")
		} else {
			err = runPromptFlow(*stage, *base, &r)
		}
	} else {
		err = run(*stage, *base, *resumeRun, *contributionsOnly, &r)
	}
	if err != nil {
		if *realFlow || *realPreflight || *realSecret != "" || *realLedger != "" {
			r.Failure = "controlled real-model check stopped; see bounded check and guard audit codes"
			r.Checks = append(r.Checks, check{Name: "run", FailureClass: "controlled-real-model", Detail: safeRealFailure(err)})
		} else if *promptFlow {
			r.Failure = err.Error()
			r.Checks = append(r.Checks, check{Name: "run", FailureClass: "owned-prompt-flow"})
		} else {
			class, hash := failure(err)
			r.Checks = append(r.Checks, check{Name: "run", FailureClass: class, FailureSHA256: hash})
		}
	}
	r.DurationMS = time.Since(started).Milliseconds()
	raw, _ := json.MarshalIndent(r, "", "  ")
	if writeErr := os.WriteFile(*out, append(raw, '\n'), 0600); writeErr != nil {
		fmt.Fprintln(os.Stderr, "report-write-failed", digest([]byte(writeErr.Error())))
		os.Exit(1)
	}
	fmt.Println(string(raw))
	if err != nil {
		os.Exit(1)
	}
}

func run(stage, base, resumeRun string, contributionsOnly bool, r *report) error {
	if contributionsOnly && resumeRun != "" {
		return errors.New("native contribution check requires a fresh owned profile")
	}
	root, err := resumeRun, error(nil)
	if root == "" {
		root, err = os.MkdirTemp(base, "core-host-check-")
		if err != nil {
			return err
		}
	} else {
		rel, e := filepath.Rel(base, root)
		if e != nil || filepath.IsAbs(rel) || strings.Contains(rel, string(filepath.Separator)) || !strings.HasPrefix(rel, "core-host-check-") {
			return errors.New("resume directory is outside the owned run root")
		}
		info, e := os.Lstat(filepath.Join(root, "fixture.mjs"))
		if e != nil || !info.Mode().IsRegular() {
			return errors.New("resume directory lacks its owned fixture script")
		}
	}
	userData := filepath.Join(root, "user-data")
	home := filepath.Join(userData, "dsh-home")
	workspace := filepath.Join(root, "workspace")
	if err = os.MkdirAll(workspace, 0700); err != nil {
		return err
	}
	r.OwnedRunRoot = root
	if resumeRun == "" {
		// Fresh checks usually live below the developer checkout. Keep the
		// Host's repository discovery inside the synthetic workspace.
		ownedWorkspace, err := filepath.Abs(workspace)
		if err != nil {
			return err
		}
		gitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		init := exec.CommandContext(gitCtx, "git", "init", "--quiet", "--template=", ownedWorkspace)
		if output, err := init.CombinedOutput(); err != nil {
			return fmt.Errorf("initialize owned workspace repository: %w: %s", err, strings.TrimSpace(string(output)))
		}
		verify := exec.CommandContext(gitCtx, "git", "-C", ownedWorkspace, "rev-parse", "--show-toplevel")
		output, err := verify.Output()
		if err != nil {
			return fmt.Errorf("verify owned workspace repository: %w", err)
		}
		if rel, err := filepath.Rel(ownedWorkspace, strings.TrimSpace(string(output))); err != nil || rel != "." {
			return errors.New("owned workspace repository resolved outside its exact workspace")
		}
		r.Checks = append(r.Checks, check{Name: "owned-workspace-repository", Pass: true})
	}
	if err = os.Setenv(paths.EnvUserData, userData); err != nil {
		return err
	}
	if err = os.Setenv("DSH_HOME", home); err != nil {
		return err
	}
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	supervisor := harness.New(harness.Options{
		Install:      harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}),
		Launch:       harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 32180, Home: home},
		Prepare:      harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false},
		Workspace:    workspace,
		ReadyTimeout: 3 * time.Minute,
	})
	var endpoint harness.Endpoint
	stop := func() {
		supervisor.Stop()
		r.OwnedHostStopped = supervisor.Status().State == harness.Stopped
		origin := endpoint.Origin
		if origin == "" {
			for _, line := range supervisor.Logs() {
				if at := strings.Index(line, "listening on http://"); at >= 0 {
					fields := strings.Fields(line[at+len("listening on "):])
					if len(fields) > 0 {
						origin = fields[0]
					}
				}
			}
		}
		if origin != "" {
			u, _ := url.Parse(origin)
			conn, e := net.DialTimeout("tcp", u.Host, time.Second)
			r.OwnedListenerClosed = e != nil
			if conn != nil {
				conn.Close()
			}
		}
	}
	defer stop()
	start := func() error {
		fmt.Println(`{"checkpoint":"starting-owned-host"}`)
		supervisor.Start()
		deadline := time.Now().Add(195 * time.Second)
		for time.Now().Before(deadline) {
			status := supervisor.Status()
			if status.State == harness.Ready {
				endpoint = status.Endpoint
				return nil
			}
			if status.State == harness.Failed {
				return status.Err
			}
			time.Sleep(50 * time.Millisecond)
		}
		return context.DeadlineExceeded
	}
	if resumeRun == "" {
		if err = start(); err != nil {
			for _, line := range supervisor.Logs() {
				kind := "other-line-sha256:" + digest([]byte(line))
				switch {
				case strings.Contains(line, "desktop --patch overlay"):
					kind = "desktop-overlays-prepared"
				case strings.Contains(line, " --no-open"):
					kind = "node-command-started"
				case strings.Contains(line, "listening on http://"):
					kind = "tcp-listening-before-ready"
				case strings.Contains(line, "dsh web:"):
					kind = "ready-url-emitted"
				}
				r.StartupLogKinds = append(r.StartupLogKinds, kind)
			}
			r.Checks = append(r.Checks, check{Name: "blank-host-start", Count: len(supervisor.Logs()), SHA256: digest([]byte(strings.Join(supervisor.Logs(), "\n")))})
			return err
		}
		r.Checks = append(r.Checks, check{Name: "blank-host-ready-full-prepare", Pass: true})
		fmt.Println(`{"checkpoint":"blank-host-ready"}`)
		if contributionsOnly {
			return checkNativeContributions(endpoint, r)
		}
		store, mux, _ := attach(endpoint)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		store.RefreshList(ctx)
		err = await(ctx, func() (bool, error) {
			v := store.Snapshot()
			if v.ListErr != "" {
				return false, errors.New(v.ListErr)
			}
			return !v.ListLoading, nil
		})
		cancel()
		if err != nil {
			store.Detach()
			mux.Close()
			return err
		}
		count := len(store.Snapshot().Items)
		store.Detach()
		mux.Close()
		if count != 0 {
			return errors.New("fresh isolated Host list was not empty")
		}
		r.Checks = append(r.Checks, check{Name: "blank-go-store-list", Pass: true, Count: count})
		stop()
		if !r.OwnedListenerClosed {
			return errors.New("owned blank Host listener remained open")
		}
	} else {
		r.Checks = append(r.Checks, check{Name: "continue-owned-profile-after-fixture-repair", Pass: true})
	}
	script := filepath.Join(root, "fixture.mjs")
	if err = os.WriteFile(script, []byte(fixtureScript), 0600); err != nil {
		return err
	}
	build := exec.Command(node, script, harnessRoot, home, workspace)
	if resumeRun != "" {
		build.Args = append(build.Args, "--replace-owned")
	}
	build.Env = os.Environ()
	buildOutput, err := build.CombinedOutput()
	if err != nil {
		r.Checks = append(r.Checks, check{Name: "create-owned-fixtures", SHA256: digest(buildOutput)})
		return err
	}
	var fixture struct {
		Count, Events int
		SHA256        string
	}
	if err = json.Unmarshal(buildOutput, &fixture); err != nil {
		return err
	}
	if fixture.Count != 3 || fixture.Events != 22 {
		return errors.New("unexpected inert fixture counts")
	}
	r.Checks = append(r.Checks, check{Name: "create-owned-v4-job-fixtures", Pass: true, Count: fixture.Count, SHA256: fixture.SHA256})
	if err = start(); err != nil {
		return err
	}
	r.OwnedHostStopped, r.OwnedListenerClosed = false, false
	r.Checks = append(r.Checks, check{Name: "persisted-host-ready-full-prepare", Pass: true})
	fmt.Println(`{"checkpoint":"persisted-host-ready"}`)
	store, mux, client := attach(endpoint)
	defer mux.Close()
	defer store.Detach()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var wireList api.SessionListValue
	if err = client.Call(ctx, "session/list", map[string]any{"_request": api.SessionListRequest{}}, &wireList); err != nil {
		return err
	}
	r.Checks = append(r.Checks, check{Name: "persisted-session-list-rpc", Pass: len(wireList.Items) == 3, Count: len(wireList.Items)})
	store.RefreshList(ctx)
	if err = await(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.ListErr != "" {
			return false, errors.New(v.ListErr)
		}
		return !v.ListLoading, nil
	}); err != nil {
		return err
	}
	listed := len(store.Snapshot().Items)
	r.Checks = append(r.Checks, check{Name: "persisted-go-store-list", Pass: listed == 3, Count: listed})
	if listed != 3 {
		return errors.New("persisted session list count mismatch")
	}
	for i := 1; i <= 3; i++ {
		id := fmt.Sprintf("core-host-fixture-%d", i)
		// Agent-scoped lookup invokes the production resolver/resume path.
		var goal api.GoalView
		if err = client.Call(ctx, "goals/get", map[string]any{"agentId": id}, &goal); err != nil {
			return err
		}
		r.Checks = append(r.Checks, check{Name: fmt.Sprintf("fixture-%d-agent-resume", i), Pass: true})
		store.Select(id)
		if err = await(ctx, func() (bool, error) {
			v := store.Snapshot()
			if v.HistoryError != "" {
				return false, errors.New(v.HistoryError)
			}
			return v.SelectedID == id && v.HistoryReady && !v.HistoryLoading, nil
		}); err != nil {
			return err
		}
		v := store.Snapshot()
		if len(v.Messages) != 1 || digest([]byte(v.Messages[0].Body)) != digest([]byte(fixtureText)) || v.PromptBusy || v.GoalReadError != "" {
			return errors.New("actual Store history projection mismatch")
		}
		r.Checks = append(r.Checks, check{Name: fmt.Sprintf("fixture-%d-store-follow-projection", i), Pass: true, Count: len(v.Messages), SHA256: digest([]byte(v.Messages[0].Body))})
	}
	return nil
}

func attach(endpoint harness.Endpoint) (*session.Store, *dsh.Mux, *dsh.Client) {
	ep := dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}
	client := &dsh.Client{Endpoint: ep, HTTP: &http.Client{Timeout: 20 * time.Second}}
	mux := &dsh.Mux{Endpoint: ep, DialTimeout: 10 * time.Second}
	store := session.New()
	store.Attach(client, nil)
	store.AttachMux(mux)
	return store, mux, client
}
func await(ctx context.Context, condition func() (bool, error)) error {
	for {
		ok, err := condition()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

const fixtureScript = `
import fs from 'node:fs/promises';
import path from 'node:path';
import {pathToFileURL} from 'node:url';
import {zstdCompressSync,zstdDecompressSync} from 'node:zlib';
import {createHash} from 'node:crypto';
const [root,home,cwd]=process.argv.slice(2);
const {Session}=await import(pathToFileURL(path.join(root,'packages/core/session/lib/index.js')));
const {sessionFormatCatalog:catalog}=await import(pathToFileURL(path.join(root,'packages/session/session-format-catalog/lib/index.js')));
const backend=await fs.readFile(path.join(root,'packages/session/session-persistence-jsonl/lib/index.js'),'utf8');
function actualHelper(name){const start=backend.indexOf('function '+name+'(');if(start<0)throw new Error('Pinned backend helper missing');const end=backend.indexOf('\n}',start)+2;return Function(backend.slice(start,end)+';return '+name+';')();}
const projectKey=actualHelper('projectKey');
const assertZstdHeaderFrame=actualHelper('assertZstdHeaderFrame');
const hashes=[];
for(let n=1;n<=3;n++){
 const id='core-host-fixture-'+n;
 const header={version:4,id,cwd,createdAt:1,isSeeded:false,delegationDepth:0};
 const s=new Session(id,undefined,header);
 for(let seq=0;seq<=20;seq++)s.append('job/registered',{jobId:'inert-job-'+seq,owner:id,kind:'local',label:'inert stored fact',startedAt:1});
 s.append('user/message',{id:'fixture-message-'+n,role:'user',source:{kind:'user'},content:[{type:'text',text:'Core host isolated persistence fixture. No model request.'}]},{surfaceOp:'append'});
 const events=Array.from({length:s.seq},(_,i)=>({...s.eventAt(i),time:i+1}));
 Session.fromRestore(id,events,header,0,'detached');
 const records=[catalog.encodeCurrentHeader(header,0),...events.map(event=>catalog.encodeCurrentEvent(event))];
 const headerFrame=zstdCompressSync(Buffer.from(JSON.stringify(records[0])+'\n'));
 assertZstdHeaderFrame(zstdDecompressSync(headerFrame));
 const raw=Buffer.concat([headerFrame,zstdCompressSync(Buffer.from(records.slice(1).map(row=>JSON.stringify(row)).join('\n')+'\n'))]);
 const dir=path.join(home,'sessions',projectKey(cwd),id);
 if(process.argv[5]!=='--validate-only'){await fs.mkdir(dir,{recursive:true});await fs.writeFile(path.join(dir,'session.v4.jsonl.zstd'),raw,{flag:process.argv[5]==='--replace-owned'?'w':'wx'});}
 hashes.push(createHash('sha256').update(raw).digest('hex'));
}
console.log(JSON.stringify({Count:3,Events:22,SHA256:createHash('sha256').update(hashes.join('\n')).digest('hex')}));
`

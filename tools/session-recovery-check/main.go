// Opt-in real-Host acceptance. All data, sessions and destructive operations
// belong to one new run directory. No prompt or model operation is invoked.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/recovery"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/composer"
)

type result struct {
	ID     string `json:"id"`
	Layer  string `json:"layer"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail,omitempty"`
	Hash   string `json:"sha256,omitempty"`
}
type report struct {
	Started        string   `json:"started"`
	RunRoot        string   `json:"ownedRunRoot"`
	Results        []result `json:"results"`
	Failure        string   `json:"failure,omitempty"`
	ModelRequests  int      `json:"modelRequests"`
	HostStopped    bool     `json:"ownedHostStopped"`
	ListenerClosed bool     `json:"ownedListenerClosed"`
	Limits         []string `json:"limits"`
}

func hash(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func wait(ctx context.Context, condition func() (bool, error)) error {
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
func (r *report) pass(id, layer, detail string) {
	r.Results = append(r.Results, result{ID: id, Layer: layer, Pass: true, Detail: detail})
	fmt.Println("checkpoint", id)
}
func main() {
	stage := flag.String("stage", "", "verified runtime")
	base := flag.String("root", "", "owned run parent")
	out := flag.String("out", "", "evidence JSON")
	native := flag.String("native", "", "optional dedicated native recovery test executable")
	flag.Parse()
	if *stage == "" || *base == "" || *out == "" {
		panic("stage/root/out required")
	}
	r := report{Started: time.Now().UTC().Format(time.RFC3339), Limits: []string{"No model/prompt requests; copied source is entirely synthetic.", "Host RPC and Go Store checks do not prove native menu gestures or original pixel parity.", "Approval/question/Workflow/goal/plan/feedback live Agent transitions remain unverified; no live Agent trigger is fabricated.", "Import cancellation is between completed items; crash-at-swap recovery remains a separate fixture gate."}}
	err := run(*stage, *base, *native, &r)
	if err != nil {
		r.Failure = err.Error()
	}
	raw, _ := json.MarshalIndent(r, "", "  ")
	if err := os.WriteFile(*out, append(raw, '\n'), 0600); err != nil {
		panic(err)
	}
	fmt.Println(string(raw))
	if err != nil {
		os.Exit(1)
	}
}

func run(stage, base, native string, r *report) error {
	root, err := os.MkdirTemp(base, "session-recovery-check-")
	if err != nil {
		return err
	}
	r.RunRoot = root
	userData := filepath.Join(root, "user-data")
	home := filepath.Join(userData, "dsh-home")
	source := filepath.Join(root, "synthetic-source")
	workspace := filepath.Join(root, "workspace")
	ownedUser := filepath.Join(root, "owned-userhome")
	for _, dir := range []string{home, source, workspace, ownedUser} {
		if err = os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	// BuildCatalog also checks ~/.agents/skills. Redirect every process-local
	// default before using it; never inspect the caller's real home/profile.
	for key, value := range map[string]string{paths.EnvUserData: userData, "DSH_HOME": home, "HOME": ownedUser, "USERPROFILE": ownedUser, "APPDATA": filepath.Join(ownedUser, "AppData", "Roaming"), "LOCALAPPDATA": filepath.Join(ownedUser, "AppData", "Local")} {
		if err = os.Setenv(key, value); err != nil {
			return err
		}
	}
	harnessRoot := filepath.Join(stage, "vendor", "deepseek-harness")
	node := filepath.Join(stage, "node", "node.exe")
	script := filepath.Join(root, "seed.mjs")
	if err = os.WriteFile(script, []byte(seedScript), 0600); err != nil {
		return err
	}
	cmd := exec.Command(node, script, harnessRoot, source, workspace)
	if output, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("synthetic seed refused: %s (%w)", hash(output), e)
	}
	before, err := treeHash(source)
	if err != nil {
		return err
	}
	cat, err := importer.BuildCatalog(source, home)
	if err != nil {
		return err
	}
	if len(cat.Sessions) != 3 {
		return fmt.Errorf("synthetic import catalog count %d", len(cat.Sessions))
	}
	rels := []string{}
	for _, item := range cat.Sessions {
		if item.Unsupported {
			return errors.New("v4 fixture unsupported by import catalog")
		}
		rels = append(rels, item.Rel)
	}
	sort.Strings(rels)
	preview, err := importer.ApplyCatalog(cat, importer.ApplySelection{})
	if err != nil || !preview.Empty {
		return fmt.Errorf("empty import selection: %w", err)
	}
	r.pass("BOOT-05-preview-empty", "production importer/filesystem", "3 synthetic v4 rows; empty selection writes no payload")
	cctx, cancel := context.WithCancel(context.Background())
	partial, err := importer.ApplyCatalogProgress(cctx, cat, importer.ApplySelection{SelectedRels: rels}, func(importer.ApplyResult) { cancel() })
	cancel()
	if !errors.Is(err, context.Canceled) || partial.Copied != 1 {
		return fmt.Errorf("partial cancellation: copied=%d err=%v", partial.Copied, err)
	}
	if _, err = importer.RecoverInterrupted(home); err != nil {
		return err
	}
	r.pass("BOOT-05-cancel", "production importer/filesystem", "cancel after one committed item; remaining two absent and startup recovery admits completed state")
	complete, err := importer.ApplyCatalog(cat, importer.ApplySelection{SelectedRels: rels})
	if err != nil || complete.Copied != 2 || complete.Skipped != 1 {
		return fmt.Errorf("resume import: %+v %w", complete, err)
	}
	conflict, err := importer.ApplyCatalog(cat, importer.ApplySelection{SelectedRels: rels})
	if err != nil || conflict.Skipped != 3 {
		return fmt.Errorf("default conflict policy: %+v %w", conflict, err)
	}
	overwrite, err := importer.ApplyCatalog(cat, importer.ApplySelection{SelectedRels: rels[:1], Overwrite: true})
	if err != nil || overwrite.Copied != 1 {
		return fmt.Errorf("explicit overwrite: %+v %w", overwrite, err)
	}
	after, err := treeHash(source)
	if err != nil || before != after {
		return errors.New("synthetic source changed during import")
	}
	r.Results = append(r.Results, result{ID: "BOOT-05-copy-conflict-resume", Layer: "production importer/filesystem", Pass: true, Detail: "partial resume, skip conflicts and selected overwrite; no credentials or unrelated source", Hash: before})
	resume, err := recovery.Open(filepath.Join(root, "snapshot-store"))
	if err != nil {
		return err
	}
	draft := composer.NewDraftState("unfinished owned recovery draft")
	if err = resume.Update(func(v *recovery.State) {
		v.Workspace = workspace
		v.SelectedSession = "session-recovery-fixture-1"
		v.Drafts = map[string]json.RawMessage{"session-recovery-fixture-1": draft.SaveDraft()}
		v.Layout = recovery.Layout{Sidebar: 300, SidebarExpanded: 300, Right: 450, RightOpen: true, RightTab: "agents", ChatTab: "trajectory", TranscriptWidth: 860, WorkspacePanes: json.RawMessage(`{"version":1}`)}
	}); err != nil {
		return err
	}
	reopened, err := recovery.Open(filepath.Join(root, "snapshot-store"))
	if err != nil {
		return err
	}
	restored, err := composer.RestoreDraftChecked(reopened.Snapshot().Drafts["session-recovery-fixture-1"])
	if err != nil || restored.Text != draft.Text {
		return errors.New("draft disk roundtrip mismatch")
	}
	r.pass("BOOT-06-CHAT-06-snapshot", "production recovery API/filesystem", "selected ID, unsent draft and layout persist through fresh Store open; native close/reopen is separate below")
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: filepath.Join(stage, "vendor"), Host: "127.0.0.1", Port: 32181, Home: home}, Prepare: harness.PrepareOptions{VendorDir: filepath.Join(stage, "vendor"), HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var endpoint harness.Endpoint
	stop := func() {
		sup.Stop()
		r.HostStopped = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, _ := url.Parse(endpoint.Origin)
			conn, e := net.DialTimeout("tcp", u.Host, time.Second)
			r.ListenerClosed = e != nil
			if conn != nil {
				conn.Close()
			}
		}
	}
	defer stop()
	start := func() error {
		sup.Start()
		ctx, cancel := context.WithTimeout(context.Background(), 195*time.Second)
		defer cancel()
		if err := wait(ctx, func() (bool, error) {
			v := sup.Status()
			if v.State == harness.Failed {
				return false, v.Err
			}
			if v.State == harness.Ready {
				endpoint = v.Endpoint
				return true, nil
			}
			return false, nil
		}); err != nil {
			return err
		}
		return nil
	}
	if err = start(); err != nil {
		return err
	}
	r.pass("BOOT-05-imported-v4-host", "production Host", "full Prepare ready on imported synthetic v4 artifacts")
	store, mux, client := attach(endpoint)
	defer func() { store.Detach(); mux.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()
	store.SetWorkspaceCwd(workspace)
	store.RefreshList(ctx)
	store.FollowWorkspace(ctx)
	refresh := func() error {
		store.RefreshList(ctx)
		return wait(ctx, func() (bool, error) {
			v := store.Snapshot()
			if v.ListErr != "" {
				return false, errors.New(v.ListErr)
			}
			return !v.ListLoading, nil
		})
	}
	if err = refresh(); err != nil {
		return err
	}
	if len(store.Snapshot().Items) != 3 {
		return fmt.Errorf("imported Host list count %d", len(store.Snapshot().Items))
	}
	r.pass("CHAT-01-imported-list", "production Host + Go Store", "3 imported sessions visible without model requests")
	sourceID := "session-recovery-fixture-1"
	owned := map[string]bool{}
	for n := 1; n <= 3; n++ {
		owned[fmt.Sprintf("session-recovery-fixture-%d", n)] = true
	}
	store.Select(sourceID)
	if err = wait(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.HistoryError != "" {
			return false, errors.New(v.HistoryError)
		}
		for _, message := range v.Messages {
			if message.Role == "user" && message.Body == "Owned session recovery fixture. No model request." {
				return v.HistoryReady, nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	r.pass("CHAT-03-imported-follow", "production Host + Go Store", "v4 source restored and user-message projection read")
	store.UpdateSessionSearch(ctx, "Owned session recovery fixture")
	if err = wait(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.SearchErr != "" {
			return false, errors.New(v.SearchErr)
		}
		return !v.SearchLoading && v.SearchRPCQuery != "" && len(v.SearchHitIDs) == 3, nil
	}); err != nil {
		return err
	}
	r.pass("CHAT-01-content-search", "production Host + Go Store", "all three synthetic messages found by served content search")
	store.UpdateSessionSearch(ctx, "")
	store.CreateSession(ctx)
	if err = wait(ctx, func() (bool, error) {
		v := store.Snapshot()
		if v.LastAction != "" && strings.Contains(v.LastAction, "failed") {
			return false, errors.New(v.LastAction)
		}
		return v.SelectedID != "" && !owned[v.SelectedID] && v.LastAction == "session created", nil
	}); err != nil {
		return err
	}
	created := store.Snapshot().SelectedID
	owned[created] = true
	r.pass("CHAT-02-create", "production Host + Go Store", "new session/create with no prompt")
	title := "Owned acceptance renamed"
	store.BeginSessionRename(created, "")
	store.RenameSession(ctx, created, title)
	if err = wait(ctx, func() (bool, error) {
		v := store.SessionRename()
		if v.Error != "" {
			return false, errors.New(v.Error)
		}
		for _, item := range store.Snapshot().Items {
			if item.SessionID == created && item.Title == title {
				return true, nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	r.pass("CHAT-02-rename", "production Host + Go Store", "explicit title acknowledged and projected")
	store.PinSession(ctx, sourceID)
	if err = wait(ctx, func() (bool, error) {
		for _, item := range store.Snapshot().Items {
			if item.SessionID == sourceID {
				return item.Pinned, nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	r.pass("CHAT-01-pin", "production Host + Go Store", "owned source pin receipt projected")
	archived := func(id string) bool {
		for _, v := range store.Snapshot().ArchivedSessionIDs {
			if v == id {
				return true
			}
		}
		return false
	}
	store.ArchiveSession(ctx, "session-recovery-fixture-2")
	if err = wait(ctx, func() (bool, error) { return archived("session-recovery-fixture-2"), nil }); err != nil {
		return err
	}
	store.UnarchiveSession(ctx, "session-recovery-fixture-2")
	if err = wait(ctx, func() (bool, error) { return !archived("session-recovery-fixture-2"), nil }); err != nil {
		return err
	}
	r.pass("CHAT-02-archive-restore", "production Host + Go Store", "owned idle source archived and restored without stopActivity")
	store.ForkSession(ctx, sourceID)
	if err = wait(ctx, func() (bool, error) {
		v := store.Snapshot()
		return v.LastAction == "session forked" && !owned[v.SelectedID] && v.SelectedID != "", nil
	}); err != nil {
		return err
	}
	forked := store.Snapshot().SelectedID
	owned[forked] = true
	if err = refresh(); err != nil {
		return err
	}
	r.pass("CHAT-02-13-fork", "production Host + Go Store", "owned persisted source forked; no prompt continuation requested")
	// session/delete requires archive. Both IDs were generated by this run.
	if !owned[forked] {
		return errors.New("delete target is not owned")
	}
	store.ArchiveSession(ctx, forked)
	if err = wait(ctx, func() (bool, error) { return archived(forked), nil }); err != nil {
		return err
	}
	store.DeleteSession(ctx, forked)
	if err = wait(ctx, func() (bool, error) { return store.Snapshot().LastAction == "session deleted", nil }); err != nil {
		return err
	}
	if err = refresh(); err != nil {
		return err
	}
	for _, item := range store.Snapshot().Items {
		if item.SessionID == forked {
			return errors.New("deleted owned fork remained listed")
		}
	}
	r.pass("CHAT-02-delete", "production Host + Go Store", "only this run's archived fork was deleted")
	// Ensure title/pin persistence really survives a Host process boundary,
	// rather than inferring durability from a live/pending list row.
	var list api.SessionListValue
	if err = client.Call(ctx, "session/list", map[string]any{"_request": api.SessionListRequest{}}, &list); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	store.Detach()
	mux.Close()
	stop()
	if !r.ListenerClosed {
		return errors.New("owned Host listener remained open")
	}
	if err = start(); err != nil {
		return err
	}
	store, mux, client = attach(endpoint)
	store.SetWorkspaceCwd(workspace)
	store.FollowWorkspace(ctx)
	if err = refresh(); err != nil {
		return err
	}
	if err = wait(ctx, func() (bool, error) {
		v := store.Snapshot()
		var named, pinned bool
		for _, item := range v.Items {
			named = named || item.SessionID == created && item.Title == title
			pinned = pinned || item.SessionID == sourceID && item.Pinned
		}
		return v.WorkspaceReady && named && pinned, nil
	}); err != nil {
		return err
	}
	r.pass("CHAT-01-02-host-restart", "production Host + Go Store", "renamed title and pinned source survived actual process stop/restart")
	store.UnpinSession(ctx, sourceID)
	if err = wait(ctx, func() (bool, error) {
		for _, item := range store.Snapshot().Items {
			if item.SessionID == sourceID {
				return !item.Pinned, nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	r.pass("CHAT-01-unpin", "production Host + Go Store", "owned source unpinned after restart")
	store.Detach()
	mux.Close()
	stop()
	if native != "" {
		for _, phase := range []string{"save", "restore"} {
			cmd := exec.Command(native, "-test.run=TestNativeSessionRecovery", "-test.timeout=130s")
			cmd.Env = append(os.Environ(), "WHALEISLE_ISOLATED=1", "WHALEISLE_VENDOR="+filepath.Join(stage, "vendor"), "WHALEISLE_NODE="+node, "WHALEISLE_SESSION_QA_PHASE="+phase, "WHALEISLE_SESSION_QA_ID="+sourceID, "WHALEISLE_SESSION_QA_WORKSPACE="+workspace, "WHALEISLE_SESSION_QA_OUT="+filepath.Join(root, "native-"+phase+".json"))
			output, e := cmd.CombinedOutput()
			if writeErr := os.WriteFile(filepath.Join(root, "native-"+phase+".log"), output, 0600); writeErr != nil {
				return writeErr
			}
			if e != nil {
				return fmt.Errorf("native recovery %s failed (%s): %w", phase, hash(output), e)
			}
			r.pass("BOOT-06-native-"+phase, "production native mainView + real Host", "separate owned desktop process; see per-phase owned JSON")
		}
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
func treeHash(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("fixture contains link")
		}
		rel, _ := filepath.Rel(root, path)
		raw, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		fmt.Fprintln(h, filepath.ToSlash(rel), hash(raw))
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}

const seedScript = `
import fs from 'node:fs/promises';import path from 'node:path';import {pathToFileURL} from 'node:url';import {zstdCompressSync,zstdDecompressSync} from 'node:zlib';
const[root,home,cwd]=process.argv.slice(2);const{Session}=await import(pathToFileURL(path.join(root,'packages/core/session/lib/index.js')));const{sessionFormatCatalog:catalog}=await import(pathToFileURL(path.join(root,'packages/session/session-format-catalog/lib/index.js')));
const backend=await fs.readFile(path.join(root,'packages/session/session-persistence-jsonl/lib/index.js'),'utf8');function helper(name){const start=backend.indexOf('function '+name+'(');if(start<0)throw Error('missing backend helper');const end=backend.indexOf('\n}',start)+2;return Function(backend.slice(start,end)+';return '+name+';')()}const projectKey=helper('projectKey'),assertHeader=helper('assertZstdHeaderFrame');
for(let n=1;n<=3;n++){const id='session-recovery-fixture-'+n,header={version:4,id,cwd,createdAt:1,isSeeded:false,delegationDepth:0},s=new Session(id,undefined,header);for(let i=0;i<=20;i++)s.append('job/registered',{jobId:'owned-inert-'+i,owner:id,kind:'local',label:'stored inert fact',startedAt:1});s.append('turn/start',{turn:1});s.append('user/message',{id:'owned-message-'+n,role:'user',source:{kind:'user'},content:[{type:'text',text:'Owned session recovery fixture. No model request.'}]},{surfaceOp:'append'});s.append('turn/end',{turn:1,reason:{kind:'completed'}});const events=Array.from({length:s.seq},(_,i)=>({...s.eventAt(i),time:i+1}));Session.fromRestore(id,events,header,0,'detached');const head=zstdCompressSync(Buffer.from(JSON.stringify(catalog.encodeCurrentHeader(header,0))+'\n'));assertHeader(zstdDecompressSync(head));const body=zstdCompressSync(Buffer.from(events.map(e=>JSON.stringify(catalog.encodeCurrentEvent(e))).join('\n')+'\n'));const dir=path.join(home,'sessions',projectKey(cwd),id);await fs.mkdir(dir,{recursive:true});await fs.writeFile(path.join(dir,'session.v4.jsonl.zstd'),Buffer.concat([head,body]),{flag:'wx'});}
console.log(JSON.stringify({count:3,events:24}));
`

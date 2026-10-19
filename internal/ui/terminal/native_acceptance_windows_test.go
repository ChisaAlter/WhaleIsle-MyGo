//go:build windows && terminalacceptance

package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	myterm "github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/terminalruntime"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/hostterminal"
)

// Explicit invocation uses an owned cwd and cmd /D (no AutoRun) rather than a
// user's interactive PowerShell profile. The production native plugin still
// supplies the emulator, input queue, ConPTY, resize and teardown.
func TestNativeConPTYLifecycle(t *testing.T) {
	spec, err := terminalruntime.Specification("windows-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = terminalruntime.Resolve(spec, os.Getenv("MYGO_GHOSTTY_VT"), nil); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	shell := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	open := func() *myterm.Terminal {
		term, e := myterm.New(myterm.Options{Dir: root, Command: []string{shell, "/D", "/Q"}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { _ = term.Close() })
		return term
	}
	wait := func(term *myterm.Terminal, token string) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			text := strings.ReplaceAll(term.Text(), "\r", "")
			if strings.Contains("\n"+text+"\n", "\n"+token+"\n") {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("native output did not contain the expected complete sentinel line")
	}
	term := open()
	term.Resize(100, 32)
	term.Send([]byte("echo WHALEISLE_NATIVE_SENTINEL\r"))
	wait(term, "WHALEISLE_NATIVE_SENTINEL")
	term.Send([]byte("cd\r"))
	wait(term, root)
	term.Resize(92, 28)
	if cols, rows := term.Size(); cols != 92 || rows != 28 {
		t.Fatalf("resize grid=%dx%d", cols, rows)
	}
	term.Send([]byte("echo WHALEISLE_NATIVE_RESIZED\r"))
	wait(term, "WHALEISLE_NATIVE_RESIZED")
	if err = term.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-term.Done():
	case <-time.After(6 * time.Second):
		t.Fatal("ConPTY did not close")
	}
	reopened := open()
	reopened.Resize(80, 24)
	reopened.Send([]byte("echo WHALEISLE_NATIVE_REOPENED\r"))
	wait(reopened, "WHALEISLE_NATIVE_REOPENED")
	reopened.Send([]byte("exit /b 0\r"))
	select {
	case <-reopened.Done():
	case <-time.After(6 * time.Second):
		t.Fatal("natural exit did not finish")
	}
	if reopened.ExitCode() != 0 {
		t.Fatalf("natural exit code=%d", reopened.ExitCode())
	}
	t.Log("pinned native DLL + owned cmd /D ConPTY: sentinel, cwd, resize, active close, reopen, natural exit passed")
}

func TestRealHostTerminalLifecycle(t *testing.T) {
	stage := os.Getenv("WHALEISLE_TERMINAL_ACCEPTANCE_STAGE")
	if stage == "" {
		t.Fatal("explicit verified runtime stage required")
	}
	root := t.TempDir()
	home, workspace := filepath.Join(root, "dsh-home"), filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WHALEISLE_USER_DATA", filepath.Join(root, "user-data"))
	t.Setenv("DSH_HOME", home)
	node := filepath.Join(stage, "node", "node.exe")
	vendor := filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	script := filepath.Join(root, "inert-session.mjs")
	if err := os.WriteFile(script, []byte(terminalSessionFixture), 0600); err != nil {
		t.Fatal(err)
	}
	build := exec.Command(node, script, harnessRoot, home, workspace)
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("inert fixture: %v: %.300s", err, b)
	}
	shell, _ := json.Marshal(filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe"))
	patch := filepath.Join(root, "terminal.patch.yml")
	contents := fmt.Sprintf("- id: terminal-controller\n  config:\n    shell:\n      path: %s\n      name: isolated-cmd\n      args: [/D, /Q]\n    shellCandidates: []\n", shell)
	if err := os.WriteFile(patch, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	sup := harness.New(harness.Options{
		Install:   harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}),
		Launch:    harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: 32320, Home: home, PatchFiles: []string{patch}},
		Prepare:   harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false},
		Workspace: workspace, ReadyTimeout: 45 * time.Second,
	})
	sup.Start()
	defer sup.Stop()
	deadline := time.Now().Add(50 * time.Second)
	for sup.Status().State != harness.Ready {
		if sup.Status().State == harness.Failed {
			t.Fatal(sup.Status().Err)
		}
		if time.Now().After(deadline) {
			t.Fatal("owned Host readiness timeout")
		}
		time.Sleep(25 * time.Millisecond)
	}
	endpoint := sup.Status().Endpoint
	ep := dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}
	client := &dsh.Client{Endpoint: ep, HTTP: &http.Client{Timeout: 15 * time.Second}}
	mux := &dsh.Mux{Endpoint: ep, DialTimeout: 8 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	if err := mux.Dial(ctx); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()
	defer mux.Close()
	store := &hostterminal.Store{}
	store.Bind(client, mux, true, nil)
	defer store.Bind(nil, nil, false, nil)
	const session = "terminal-owned-inert"
	store.Show(session, true)
	h := &HostState{}
	defer h.drop()
	tt := ui.NewTester(func(c *ui.Context) { HostView(c, h, store, session, "light", false) }, 1000, 500)
	wait := func(what string, condition func(hostterminal.View) bool) {
		t.Helper()
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			tt.Frame()
			v := store.Snapshot(session)
			if v.Error != "" {
				t.Fatalf("%s: %s", what, v.Error)
			}
			if condition(v) {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("%s timed out", what)
	}
	owner := store.Snapshot(session).Owner
	store.Refresh(owner, session)
	wait("empty list", func(v hostterminal.View) bool { return !v.Busy && len(v.Items) == 0 })
	store.DiscoverShells(owner, session)
	wait("shell discovery", func(v hostterminal.View) bool {
		s := store.ShellSnapshot(owner, session)
		if s.Error != "" {
			t.Fatal(s.Error)
		}
		return s.Loaded
	})
	shells := store.ShellSnapshot(owner, session)
	if len(shells.Items) != 1 || strings.Join(shells.Items[0].Args, " ") != "/D /Q" {
		t.Fatal("test-only no-profile shell config was not applied")
	}
	store.SelectShell(owner, session, shells.Items[0].Path)
	store.Create(owner, session)
	wait("create and native snapshot", func(v hostterminal.View) bool { return v.Writable && h.terminal != nil })
	id := store.Snapshot(session).ActiveID
	h.terminal.Send([]byte("echo WHALEISLE_HOST_SENTINEL\r"))
	wait("host sentinel", func(v hostterminal.View) bool {
		return strings.Contains("\n"+strings.ReplaceAll(h.terminal.Text(), "\r", "")+"\n", "\nWHALEISLE_HOST_SENTINEL\n")
	})
	beforeCols, beforeRows := h.terminal.Size()
	tt.SetSize(800, 420)
	wait("host resize from native view fit", func(v hostterminal.View) bool {
		cols, rows := h.terminal.Size()
		return (cols != beforeCols || rows != beforeRows) && v.Info.Cols == cols && v.Info.Rows == rows
	})
	// A second pane displaying the same retained process takes input control,
	// while the first remains an output-only view. No second shell is created.
	secondStore := store.Occurrence("side", "", session)
	secondView := &HostState{}
	defer secondView.drop()
	secondTester := ui.NewTester(func(c *ui.Context) { HostView(c, secondView, secondStore, session, "light", false) }, 800, 420)
	wait("second pane takes input control", func(v hostterminal.View) bool {
		secondTester.Frame()
		other := secondStore.Snapshot(session)
		if other.Error != "" {
			t.Fatal(other.Error)
		}
		return !v.Writable && other.Writable && secondView.terminal != nil
	})
	oldAttachment := store.Snapshot(session).Attachment
	if _, err := oldAttachment.Write([]byte("echo MUST_NOT_EXECUTE\r")); err == nil {
		t.Fatal("read-only pane accepted input")
	}
	secondView.Observe(secondStore, session, false)
	store.Connect(owner, session, id)
	wait("explicit original pane takes input control", func(v hostterminal.View) bool { return v.Writable && h.terminal != nil })
	store.BeginRename(owner, session, id)
	store.EditRename(owner, session, id, "Owned terminal")
	store.SaveRename(owner, session, id)
	wait("rename", func(v hostterminal.View) bool { return v.Info.Title == "Owned terminal" })
	h.Observe(store, session, false)
	if h.terminal != nil {
		t.Fatal("hidden native renderer retained")
	}
	// The same production HostView resumes an existing process, never allocates.
	wait("reattach retained process", func(v hostterminal.View) bool {
		return v.ActiveID == id && v.Writable && h.terminal != nil && strings.Contains(h.terminal.Text(), "WHALEISLE_HOST_SENTINEL")
	})
	store.Close(owner, session, id)
	wait("close", func(v hostterminal.View) bool { return v.Phase == "closed" })
	store.Refresh(owner, session)
	wait("closed identity removed", func(v hostterminal.View) bool { return !v.Busy && len(v.Items) == 0 })
	store.Create(owner, session)
	wait("reopen", func(v hostterminal.View) bool { return v.ActiveID != id && v.Writable && h.terminal != nil })
	reopened := store.Snapshot(session).ActiveID
	h.terminal.Send([]byte("echo WHALEISLE_HOST_REOPENED\r"))
	wait("reopened output", func(v hostterminal.View) bool { return strings.Contains(h.terminal.Text(), "WHALEISLE_HOST_REOPENED") })
	h.terminal.Send([]byte("exit /b 0\r"))
	wait("Host natural shell exit", func(v hostterminal.View) bool {
		return v.Info.State == "exited" && !v.Writable && v.Info.ExitCode != nil && *v.Info.ExitCode == 0
	})
	store.Close(owner, session, reopened)
	wait("final close", func(v hostterminal.View) bool { return v.Phase == "closed" })
	h.drop()
	store.Bind(nil, nil, false, nil)
	mux.Close()
	sup.Stop()
	u, _ := url.Parse(endpoint.Origin)
	conn, err := net.DialTimeout("tcp", u.Host, time.Second)
	if conn != nil {
		conn.Close()
	}
	if err == nil {
		t.Fatal("owned Host listener remained open")
	}
	t.Log("real rc.2 Host + production HostView + native DLL: discover, create, snapshot, input, resize, two-pane control/read-only, rename, hide/reattach, close/list, reopen, natural exit and cleanup passed")
}

const terminalSessionFixture = `
import fs from 'node:fs/promises';import path from 'node:path';import {pathToFileURL} from 'node:url';import {zstdCompressSync} from 'node:zlib';
const [root,home,cwd]=process.argv.slice(2);
const {Session}=await import(pathToFileURL(path.join(root,'packages/core/session/lib/index.js')));
const {sessionFormatCatalog:catalog}=await import(pathToFileURL(path.join(root,'packages/session/session-format-catalog/lib/index.js')));
const backend=await fs.readFile(path.join(root,'packages/session/session-persistence-jsonl/lib/index.js'),'utf8');
const start=backend.indexOf('function projectKey(');const end=backend.indexOf('\n}',start)+2;
const projectKey=Function(backend.slice(start,end)+';return projectKey;')();
const id='terminal-owned-inert';const header={version:4,id,cwd,createdAt:1,isSeeded:false,delegationDepth:0};
const s=new Session(id,undefined,header);s.append('user/message',{id:'inert',role:'user',source:{kind:'user'},content:[{type:'text',text:'Inert terminal fixture; no prompt sent.'}]},{surfaceOp:'append'});
const ev={...s.eventAt(0),time:1};Session.fromRestore(id,[ev],header,0,'detached');
const raw=Buffer.concat([zstdCompressSync(Buffer.from(JSON.stringify(catalog.encodeCurrentHeader(header,0))+'\n')),zstdCompressSync(Buffer.from(JSON.stringify(catalog.encodeCurrentEvent(ev))+'\n'))]);
const dir=path.join(home,'sessions',projectKey(cwd),id);await fs.mkdir(dir,{recursive:true});await fs.writeFile(path.join(dir,'session.v4.jsonl.zstd'),raw,{flag:'wx'});
`

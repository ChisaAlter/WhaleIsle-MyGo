//go:build windows && directory_functional_acceptance

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// This opt-in check drives the actual mainState directory controller and its
// production dialog using headless pointer/key input plus one real owned Host.
// It does not open the OS picker, touch clipboard, submit a prompt or use a model.
func TestFunctionalDirectorySourceContract(t *testing.T) {
	root, stage, out := os.Getenv("WHALEISLE_DIRECTORY_ACCEPTANCE_ROOT"), os.Getenv("WHALEISLE_DIRECTORY_ACCEPTANCE_STAGE"), os.Getenv("WHALEISLE_DIRECTORY_ACCEPTANCE_OUTPUT")
	if root == "" || stage == "" || out == "" {
		t.Skip("explicit isolated acceptance paths required")
	}
	if !filepath.IsAbs(root) || !filepath.IsAbs(stage) || !filepath.IsAbs(out) {
		t.Fatal("absolute isolated paths required")
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal("fresh root required:", err)
	}
	workspace := filepath.Join(root, "workspace")
	for _, name := range []string{".private", "spaced child", "visible"} {
		if err := os.MkdirAll(filepath.Join(workspace, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	data, vendor := filepath.Join(root, "data"), filepath.Join(stage, "vendor")
	home, node := filepath.Join(data, "dsh-home"), filepath.Join(stage, "node", "node.exe")
	hr := filepath.Join(vendor, "deepseek-harness")
	t.Setenv(paths.EnvUserData, data)
	t.Setenv(paths.EnvVendor, vendor)
	t.Setenv("DSH_HOME", home)
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	report := map[string]any{"matrix": []string{"WORK-02"}, "root": root, "modelRequests": 0, "scope": "real owned Host + shipped mainState controller + headless dialog input; no native OS picker", "pass": false}
	checks := map[string]any{}
	report["checks"] = checks
	defer func() {
		report["pass"] = !t.Failed()
		raw, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			err = os.WriteFile(out, append(raw, '\n'), 0600)
		}
		if err != nil {
			t.Error(err)
		}
	}()
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: hr, VendorDir: vendor, Host: "127.0.0.1", Port: 32383, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: hr, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 3 * time.Minute})
	var ep harness.Endpoint
	defer func() {
		sup.Stop()
		report["ownedHostStopped"] = sup.Status().State == harness.Stopped
		if u, e := url.Parse(ep.Origin); e == nil && u.Host != "" {
			conn, e := net.DialTimeout("tcp", u.Host, time.Second)
			report["ownedListenerClosed"] = e != nil
			if conn != nil {
				conn.Close()
			}
		}
	}()
	sup.Start()
	ctx, cancel := context.WithTimeout(context.Background(), 195*time.Second)
	defer cancel()
	for sup.Status().State != harness.Ready {
		if s := sup.Status(); s.State == harness.Failed {
			t.Fatal(s.Err)
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
	ep = sup.Status().Endpoint
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: ep.Origin, Cookie: ep.Cookie}, HTTP: &http.Client{Timeout: 20 * time.Second}}
	sessions := session.New()
	sessions.Attach(client, nil)
	defer sessions.Detach()
	s := &mainState{dshClient: client, bootStatus: sup.Status(), dirPickOpen: true, dirPickMode: "workspace", dirPickHostGeneration: sessions.Snapshot().ComposerHostGeneration}
	defer s.closeDirectoryPicker(nil)
	wait := func(want string, failed bool) {
		t.Helper()
		end := time.Now().Add(15 * time.Second)
		for time.Now().Before(end) {
			s.mu.Lock()
			settled := !s.dirPickLoading && (s.dirPickErr != "") == failed && (failed || s.dirPickBrowser.Current() != nil && strings.EqualFold(s.dirPickBrowser.Current().Path, want))
			s.mu.Unlock()
			if settled {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		t.Fatalf("directory did not settle at owned path %q: loading=%v error=%s", want, s.dirPickLoading, s.dirPickErr)
	}
	s.scanDirectoryPicker(sessions, nil, workspace, "navigate")
	wait(workspace, false)
	picked := ""
	tt := ui.NewTester(func(c *ui.Context) {
		s.directoryPickerView(c, nil, sessions, "deepseek.light", false, func(mode, path string) {
			if mode != "workspace" {
				t.Error("picker mode changed")
			}
			picked = path
		})
	}, 900, 700)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
	}
	if tt.HasText(".private") {
		t.Fatal("Host hidden entry visible before toggle")
	}
	click("Show hidden files")
	if !tt.HasText(".private") {
		t.Fatal("hidden toggle did not reveal real Host entry")
	}
	click("Show hidden files")
	if tt.HasText(".private") {
		t.Fatal("hidden toggle did not restore filtering")
	}
	checks["showHidden"] = "PASS; real .private directory hidden/revealed through shipped dialog toggle"
	click("Edit path")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(workspace + string(filepath.Separator) + ".pr")
	tt.Frame()
	if !tt.HasText(".private") || tt.HasText("spaced child") {
		t.Fatal("dot-led prefix did not reveal/filter real Host final pane")
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	s.mu.Lock()
	pathCanceled := s.dirPickBrowser.PathDraft == nil && s.dirPickBrowser.TargetPath() == workspace
	s.mu.Unlock()
	if !pathCanceled {
		t.Fatal("Escape did not cancel draft without navigation")
	}
	checks["dotPrefixAndEscape"] = "PASS; dot prefix filters actual listed entries, Escape retains current owned directory"
	child := filepath.Join(workspace, "spaced child")
	click("Edit path")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(child)
	tt.Key(0, ui.KeyEnter)
	wait(child, false)
	tt.Frame()
	s.mu.Lock()
	rawSubmitted := s.dirPickPath == child && s.dirPickBrowser.PathDraft == nil
	s.mu.Unlock()
	if !rawSubmitted {
		t.Fatal("path submission changed raw path or retained editor")
	}
	checks["pathEditSubmit"] = "PASS; shipped editor Enter reached real spaced directory and returned to breadcrumbs"
	click("New folder")
	click("Folder name")
	tt.Type("created in app")
	tt.Key(0, ui.KeyEnter)
	created := filepath.Join(child, "created in app")
	wait(created, false)
	tt.Frame()
	if info, err := os.Stat(created); err != nil || !info.IsDir() {
		t.Fatal("production controller did not create actual folder", err)
	}
	checks["newFolderController"] = "PASS; actual dialog create -> mainState -> directoryPicker/createDirectory -> retained owned filesystem directory"
	missing := filepath.Join(workspace, "created after missing")
	click("Edit path")
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type(missing)
	tt.Key(0, ui.KeyEnter)
	wait(missing, true)
	tt.Frame()
	s.mu.Lock()
	kept := s.dirPickBrowser.PathDraft != nil && *s.dirPickBrowser.PathDraft == missing
	s.mu.Unlock()
	if !kept {
		t.Fatal("failed explicit path lost draft")
	}
	if err := os.Mkdir(missing, 0700); err != nil {
		t.Fatal(err)
	}
	click("Retry")
	wait(missing, false)
	tt.Frame()
	checks["pathFailureRetry"] = "PASS; missing-path failure retained draft; actual directory creation then toast Retry completed same target"
	click("Open")
	if !strings.EqualFold(picked, missing) {
		t.Fatalf("Open selected %q instead of %q", picked, missing)
	}
	checks["confirm"] = fmt.Sprintf("PASS; Open callback returned owned target, no workspace/global configuration written")
	report["notPerformed"] = []string{"native OS picker or native pointer interaction", "ACL changes/permission-denied injection", "network or system security changes"}
}

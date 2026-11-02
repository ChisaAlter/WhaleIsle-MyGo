//go:build windows && sidebaracceptance && !nativeacceptance && !appearanceacceptance && !sessionrecoveryacceptance && !settingsacceptance && !richhistoryacceptance

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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
)

const sidebarOpenID = "sidebar-acceptance-open"
const sidebarClosedID = "sidebar-acceptance-closed"
const sidebarArchiveID = "sidebar-acceptance-archive"

var sidebarAcceptanceErr error

// This separate opt-in tag owns its TestMain; MyGo's App.Run stays on the
// process main goroutine without changing the shared nativeacceptance entry.
func TestMain(m *testing.M) {
	flag.Parse()
	if root := os.Getenv("WHALEISLE_SIDEBAR_ACCEPTANCE_ROOT"); root != "" {
		sidebarAcceptanceErr = runSidebarDesktop(root, os.Getenv("WHALEISLE_SIDEBAR_ACCEPTANCE_PHASE"))
	}
	os.Exit(m.Run())
}
func TestSidebarLiveOwnedDesktop(t *testing.T) {
	if os.Getenv("WHALEISLE_SIDEBAR_ACCEPTANCE_ROOT") == "" {
		t.Skip("explicit isolated sidebar acceptance required")
	}
	if sidebarAcceptanceErr != nil {
		t.Fatal(sidebarAcceptanceErr)
	}
}

func sidebarAcceptanceUnder(root, path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	r, err := filepath.Rel(root, filepath.Clean(path))
	return err == nil && r != "." && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}
func sidebarCheckEnvironment(root, phase string) error {
	if !filepath.IsAbs(root) || !strings.HasPrefix(filepath.Base(root), "sidebar-live-acceptance-") || (phase != "initial" && phase != "restart") || os.Getenv("WHALEISLE_ISOLATED") != "1" {
		return errors.New("explicit isolated owned root/phase required")
	}
	for _, key := range []string{"WHALEISLE_USER_DATA", "DSH_HOME", "HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "TEMP", "TMP", "AGENTS_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "GIT_CONFIG_GLOBAL", "NPM_CONFIG_USERCONFIG"} {
		if !sidebarAcceptanceUnder(root, os.Getenv(key)) {
			return fmt.Errorf("%s must be inside owned root", key)
		}
	}
	if !filepath.IsAbs(os.Getenv(paths.EnvVendor)) || !filepath.IsAbs(os.Getenv("WHALEISLE_NODE")) {
		return errors.New("absolute explicit vendor and Node required")
	}
	if !strings.EqualFold(filepath.Clean(os.Getenv(paths.EnvUserData)), filepath.Join(filepath.Clean(root), "profile")) || !strings.EqualFold(filepath.Clean(os.Getenv("DSH_HOME")), filepath.Join(filepath.Clean(root), "profile", "dsh-home")) {
		return errors.New("unexpected owned profile/home")
	}
	return nil
}
func sidebarWriteJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0600)
}
func sidebarWait(ctx context.Context, predicate func() (bool, error)) error {
	tick := time.NewTicker(75 * time.Millisecond)
	defer tick.Stop()
	for {
		ok, err := predicate()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
func sidebarDigest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func sidebarPreferenceKey(row session.WorkspaceRow) string {
	return sidebarDigest([]byte(strings.Join([]string{"workspace", row.WorkspaceID, row.Path}, "\x00")))
}
func sidebarCheckSyntheticSnapshot(root string, snap session.ViewState, expectedSelection *string) error {
	allowed := map[string]string{
		sidebarOpenID:    filepath.Join(root, "workspace-open"),
		sidebarClosedID:  filepath.Join(root, "workspace-closed"),
		sidebarArchiveID: filepath.Join(root, "workspace-open"),
	}
	if snap.PromptBusy || snap.PromptAttempted {
		return errors.New("owned fixture attempted or is submitting a prompt")
	}
	if snap.SelectedID != "" && allowed[snap.SelectedID] == "" {
		return fmt.Errorf("selection is not an owned synthetic session: %q", snap.SelectedID)
	}
	if expectedSelection != nil && snap.SelectedID != *expectedSelection {
		return fmt.Errorf("driver interaction changed selection: before=%q after=%q", *expectedSelection, snap.SelectedID)
	}
	seen := map[string]bool{}
	for _, item := range snap.Items {
		cwd, ok := allowed[item.SessionID]
		if !ok || seen[item.SessionID] || !strings.EqualFold(filepath.Clean(item.Cwd), cwd) {
			return fmt.Errorf("unexpected synthetic list identity or directory: %q", item.SessionID)
		}
		seen[item.SessionID] = true
	}
	if len(seen) != len(allowed) {
		return fmt.Errorf("expected exactly three synthetic sessions, got %d", len(seen))
	}
	for _, row := range snap.Workspaces {
		if !strings.EqualFold(filepath.Clean(row.Path), filepath.Join(root, "workspace-open")) && !strings.EqualFold(filepath.Clean(row.Path), filepath.Join(root, "workspace-closed")) {
			return errors.New("workspace is outside the two owned fixture directories")
		}
		for _, id := range row.SessionIDs {
			if allowed[id] == "" || !strings.EqualFold(allowed[id], filepath.Clean(row.Path)) {
				return fmt.Errorf("workspace contains an unexpected synthetic session: %q", id)
			}
		}
	}
	if len(snap.ArchivedSessionIDs) != 1 || snap.ArchivedSessionIDs[0] != sidebarArchiveID {
		return errors.New("archive roster is not the one owned synthetic session")
	}
	return nil
}

func sidebarCheckDriverMethod(method string) error {
	switch method {
	case "workspace/create", "workspace/archiveSession", "session/list":
		return nil
	default:
		return fmt.Errorf("driver RPC is outside the fixture allowlist: %s", method)
	}
}

func sidebarRecordCurrent(root string, s *session.Store, openID, closedID, filter string, expectedSelection *string) (map[string]any, error) {
	snap := s.Snapshot()
	if !strings.EqualFold(filepath.Clean(paths.UserDataDir()), filepath.Join(root, "profile")) || !strings.EqualFold(filepath.Clean(paths.DshHome()), filepath.Join(root, "profile", "dsh-home")) {
		return nil, errors.New("recording requires this run's owned profile and DSH home")
	}
	if err := sidebarCheckSyntheticSnapshot(root, snap, expectedSelection); err != nil {
		return nil, err
	}
	if snap.WorkspaceLayout.Collapsed[openID] || !snap.WorkspaceLayout.Collapsed[closedID] {
		return nil, fmt.Errorf("workspace folds changed: %v", snap.WorkspaceLayout.Collapsed)
	}
	if snap.WorkspaceLayout.PreferenceError != "" {
		return nil, errors.New(snap.WorkspaceLayout.PreferenceError)
	}
	archived := false
	for _, id := range snap.ArchivedSessionIDs {
		if id == sidebarArchiveID {
			archived = true
		}
	}
	if !archived {
		return nil, errors.New("owned archive roster missing")
	}
	fresh, err := config.Open(paths.UserDataDir())
	if err != nil {
		return nil, err
	}
	prefs, err := fresh.WorkspaceNavigationPreferences(s.WorkspaceLayoutScope())
	if err != nil {
		return nil, err
	}
	var openKey, closedKey string
	for _, row := range snap.Workspaces {
		if row.WorkspaceID == openID {
			openKey = sidebarPreferenceKey(row)
		}
		if row.WorkspaceID == closedID {
			closedKey = sidebarPreferenceKey(row)
		}
	}
	if openKey == "" || closedKey == "" || prefs.Collapsed[openKey] || !prefs.Collapsed[closedKey] || prefs.GroupBy != "workspace" {
		return nil, fmt.Errorf("fresh config has wrong workspace folds: %v", prefs.Collapsed)
	}
	if snap.WorkspaceLayout.ArchivedFilter != filter || (prefs.ArchivedFilter != filter && !(filter == "default" && prefs.ArchivedFilter == "")) {
		return nil, fmt.Errorf("archive filter mismatch: snapshot=%q fresh config=%q expected=%q", snap.WorkspaceLayout.ArchivedFilter, prefs.ArchivedFilter, filter)
	}
	return map[string]any{"workspaceIDs": []string{openID, closedID}, "groupBy": snap.WorkspaceLayout.GroupBy, "collapsed": snap.WorkspaceLayout.Collapsed, "freshConfigCollapsed": prefs.Collapsed, "archiveFilter": snap.WorkspaceLayout.ArchivedFilter, "freshConfigArchiveFilter": prefs.ArchivedFilter, "archiveIDs": snap.ArchivedSessionIDs, "selectedSessionID": snap.SelectedID, "selectionAllowedSyntheticOrEmpty": true, "selectionMatchesBeforeInteraction": expectedSelection != nil, "promptBusy": snap.PromptBusy, "promptAttempted": snap.PromptAttempted}, nil
}
func sidebarWaitFile(ctx context.Context, path string) error {
	return sidebarWait(ctx, func() (bool, error) {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return false, nil
		}
		return err == nil, err
	})
}

func TestSidebarAcceptanceSyntheticSelectionGuard(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sidebar-live-acceptance-guard")
	fixture := func() session.ViewState {
		return session.ViewState{
			Items: []session.Item{
				{SessionID: sidebarOpenID, Cwd: filepath.Join(root, "workspace-open")},
				{SessionID: sidebarClosedID, Cwd: filepath.Join(root, "workspace-closed")},
				{SessionID: sidebarArchiveID, Cwd: filepath.Join(root, "workspace-open")},
			},
			Workspaces: []session.WorkspaceRow{
				{WorkspaceID: "open", Path: filepath.Join(root, "workspace-open"), SessionIDs: []string{sidebarOpenID, sidebarArchiveID}},
				{WorkspaceID: "closed", Path: filepath.Join(root, "workspace-closed"), SessionIDs: []string{sidebarClosedID}},
			},
			ArchivedSessionIDs: []string{sidebarArchiveID},
		}
	}
	for _, selected := range []string{"", sidebarOpenID, sidebarClosedID, sidebarArchiveID} {
		snap := fixture()
		snap.SelectedID = selected
		if err := sidebarCheckSyntheticSnapshot(root, snap, nil); err != nil {
			t.Fatalf("automatic synthetic selection %q was rejected: %v", selected, err)
		}
		if err := sidebarCheckSyntheticSnapshot(root, snap, &selected); err != nil {
			t.Fatalf("unchanged synthetic selection %q was rejected: %v", selected, err)
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*session.ViewState)
	}{
		{"foreign-selection", func(s *session.ViewState) { s.SelectedID = "foreign-session" }},
		{"foreign-history", func(s *session.ViewState) { s.Items[0].SessionID = "foreign-session" }},
		{"foreign-directory", func(s *session.ViewState) { s.Items[0].Cwd = filepath.Dir(root) }},
		{"missing-history", func(s *session.ViewState) { s.Items = s.Items[:2] }},
		{"duplicate-history", func(s *session.ViewState) { s.Items[1] = s.Items[0] }},
		{"foreign-workspace", func(s *session.ViewState) { s.Workspaces[0].Path = filepath.Dir(root) }},
		{"foreign-member", func(s *session.ViewState) { s.Workspaces[0].SessionIDs = []string{"foreign-session"} }},
		{"foreign-archive", func(s *session.ViewState) { s.ArchivedSessionIDs = []string{"foreign-session"} }},
		{"busy-prompt", func(s *session.ViewState) { s.PromptBusy = true }},
		{"completed-prompt-attempt", func(s *session.ViewState) { s.PromptAttempted = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snap := fixture()
			tc.mutate(&snap)
			if err := sidebarCheckSyntheticSnapshot(root, snap, nil); err == nil {
				t.Fatal("guard accepted a non-isolated or prompting snapshot")
			}
		})
	}
	for _, selected := range []string{"", sidebarOpenID, sidebarClosedID, sidebarArchiveID} {
		for _, after := range []string{"", sidebarOpenID, sidebarClosedID, sidebarArchiveID} {
			if selected == after {
				continue
			}
			snap := fixture()
			snap.SelectedID = after
			if err := sidebarCheckSyntheticSnapshot(root, snap, &selected); err == nil {
				t.Fatalf("navigation %q -> %q was accepted", selected, after)
			}
		}
	}
}

func TestSidebarAcceptanceEnvironmentAndDriverAllowlist(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sidebar-live-acceptance-guard")
	for _, key := range []string{"WHALEISLE_USER_DATA", "DSH_HOME", "HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "TEMP", "TMP", "AGENTS_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "GIT_CONFIG_GLOBAL", "NPM_CONFIG_USERCONFIG"} {
		t.Setenv(key, filepath.Join(root, "owned", key))
	}
	t.Setenv(paths.EnvUserData, filepath.Join(root, "profile"))
	t.Setenv("DSH_HOME", filepath.Join(root, "profile", "dsh-home"))
	t.Setenv(paths.EnvVendor, filepath.Join(root, "vendor"))
	t.Setenv("WHALEISLE_NODE", filepath.Join(root, "node.exe"))
	t.Setenv("WHALEISLE_ISOLATED", "1")
	for _, phase := range []string{"initial", "restart"} {
		if err := sidebarCheckEnvironment(root, phase); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(paths.EnvUserData, filepath.Dir(root))
	if err := sidebarCheckEnvironment(root, "initial"); err == nil {
		t.Fatal("outside profile accepted")
	}
	for _, method := range []string{"workspace/create", "workspace/archiveSession", "session/list"} {
		if err := sidebarCheckDriverMethod(method); err != nil {
			t.Fatal(err)
		}
	}
	for _, method := range []string{"session/create", "session/prompt", "session/control", "session/follow", "model/invoke", "account/read"} {
		if err := sidebarCheckDriverMethod(method); err == nil {
			t.Fatalf("unsafe driver method %q accepted", method)
		}
	}
}

func runSidebarDesktop(root, phase string) error {
	if err := sidebarCheckEnvironment(root, phase); err != nil {
		return err
	}
	root = filepath.Clean(root)
	data := filepath.Join(root, "profile")
	if phase == "initial" {
		cfg, err := config.Open(data)
		if err != nil {
			return err
		}
		if err = cfg.Update(func(v *config.Config) {
			v.Workspace = filepath.Join(root, "workspace-open")
			v.Host = "127.0.0.1"
			v.Port = 32540
			v.NodeBin = os.Getenv("WHALEISLE_NODE")
			v.Locale = "en"
			v.CloseToTray = false
			v.RemoteEnabled = false
			v.RemoteWorkspaceEnabled = false
			v.WhaleAssistantEnabled = false
			v.DshbotEnabled = false
		}); err != nil {
			return err
		}
	}
	receipt := map[string]any{"phase": phase, "appPID": os.Getpid(), "ownedRoot": root, "data": data, "dshHome": paths.DshHome(), "vendor": paths.VendorDir(), "driver": "UI Automation + targeted WM_MOUSE messages to owned MyGoSurface", "physicalMouse": false, "ownsSessionFixturesOnly": true, "driverSessionCreateOrSelect": false, "driverPromptSubmitted": false, "productionStartupAutoSelection": "may select one of the three synthetic histories and start FollowSelected; not suppressed", "productionStartupFollow": "ordinary automatic synthetic-history follow may occur; driver does not select or open a session", "networkPacketCapture": false, "startupModelMetadataReads": "ordinary production startup, not claimed absent"}
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 170*time.Second)
		defer cancel()
		var win *mygo.Window
		var sup *harness.Supervisor
		var ownedOrigin string
		finish := func(e error) {
			if win != nil {
				win.Hide()
			}
			state.mu.Lock()
			state.quitting = true
			sup = state.sup
			state.mu.Unlock()
			stopped := sup == nil
			if sup != nil {
				if ownedOrigin == "" {
					ownedOrigin = sup.Status().Endpoint.Origin
				}
				sup.Stop() // Waits for this Supervisor's own child to exit.
				stopped = sup.Status().State == harness.Stopped
			}
			receipt["ownedHostStopped"] = stopped
			listenerClosed := false
			if u, err := url.Parse(ownedOrigin); err == nil && u.Port() != "" && net.ParseIP(u.Hostname()) != nil && net.ParseIP(u.Hostname()).IsLoopback() {
				conn, err := net.DialTimeout("tcp", u.Host, 500*time.Millisecond)
				listenerClosed = err != nil
				if conn != nil {
					conn.Close()
				}
			}
			receipt["ownedListenerClosed"] = listenerClosed
			receipt["ownedListenerNotStarted"] = ownedOrigin == ""
			cleanupFinished := stopped && (listenerClosed || ownedOrigin == "")
			if !cleanupFinished {
				e = errors.Join(e, errors.New("owned Supervisor or loopback listener cleanup was not confirmed"))
			}
			// This opt-in driver already stopped its own Host above. Permit the
			// production OnClose handler only after that private teardown finishes.
			state.mu.Lock()
			state.quitCleanupFinished = cleanupFinished
			state.mu.Unlock()
			receipt["ownedQuitCleanupFinished"] = cleanupFinished
			receipt["finishedUTC"] = time.Now().UTC().Format(time.RFC3339Nano)
			receipt["pass"] = e == nil
			if e != nil {
				receipt["error"] = e.Error()
			}
			if err := sidebarWriteJSON(filepath.Join(root, phase+"-desktop.json"), receipt); e == nil && err != nil {
				e = err
			}
			done <- e
			mygo.App.Quit()
		}
		var sessions *session.Store
		err := sidebarWait(ctx, func() (bool, error) {
			state.mu.Lock()
			win, sup, sessions = state.win, state.sup, state.sessions
			status, ready := state.bootStatus, state.dshMuxReady
			state.mu.Unlock()
			if status.State == harness.Failed {
				return false, fmt.Errorf("owned Host failed: %v", status.Err)
			}
			if win == nil || sup == nil || sessions == nil || status.State != harness.Ready || !ready {
				return false, nil
			}
			snap := sessions.Snapshot()
			if snap.ListErr != "" {
				return false, errors.New(snap.ListErr)
			}
			return snap.ClientReady && !snap.ListLoading && snap.WorkspaceReady, nil
		})
		if err != nil {
			finish(fmt.Errorf("desktop readiness: %w", err))
			return
		}
		ownedOrigin = sup.Status().Endpoint.Origin
		u, err := url.Parse(ownedOrigin)
		if err != nil || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() {
			finish(errors.New("owned Host must be loopback"))
			return
		}
		state.dshMu.Lock()
		client := state.dshClient
		state.dshMu.Unlock()
		if client == nil {
			finish(errors.New("missing owned Host client"))
			return
		}
		call := func(method string, params any, out any) error {
			if err := sidebarCheckDriverMethod(method); err != nil {
				return err
			}
			return client.Call(ctx, method, params, out)
		}
		if phase == "initial" {
			// Cold title cache records make the fixtures listable. Production may
			// automatically select/follow the first synthetic history on refresh;
			// this driver never issues session/create, selection, or a prompt.
			for _, name := range []string{"workspace-open", "workspace-closed"} {
				var created api.WorkspaceCreateValue
				if err = call("workspace/create", map[string]any{"request": map[string]any{"path": filepath.Join(root, name)}}, &created); err != nil {
					finish(fmt.Errorf("workspace fixture: %w", err))
					return
				}
			}
			stop := false
			var archived api.WorkspaceArchiveValue
			if err = call("workspace/archiveSession", map[string]any{"request": api.WorkspaceArchiveSessionRequest{SessionId: sidebarArchiveID, StopActivity: &stop}}, &archived); err != nil {
				finish(fmt.Errorf("archive owned inert fixture: %w", err))
				return
			}
			sessions.RefreshList(ctx)
		}
		var openRow, closedRow session.WorkspaceRow
		nextRefresh := time.Now().Add(2 * time.Second)
		err = sidebarWait(ctx, func() (bool, error) {
			if time.Now().After(nextRefresh) {
				sessions.RefreshList(ctx)
				nextRefresh = time.Now().Add(2 * time.Second)
			}
			snap := sessions.Snapshot()
			openRow = session.WorkspaceRow{}
			closedRow = session.WorkspaceRow{}
			for _, r := range snap.Workspaces {
				if strings.EqualFold(r.Path, filepath.Join(root, "workspace-open")) {
					openRow = r
				}
				if strings.EqualFold(r.Path, filepath.Join(root, "workspace-closed")) {
					closedRow = r
				}
			}
			seen := map[string]session.Item{}
			for _, item := range snap.Items {
				seen[item.SessionID] = item
			}
			has := func(ids []string, id string) bool {
				for _, v := range ids {
					if v == id {
						return true
					}
				}
				return false
			}
			return !snap.ListLoading && openRow.WorkspaceID != "" && closedRow.WorkspaceID != "" && has(openRow.SessionIDs, sidebarOpenID) && has(openRow.SessionIDs, sidebarArchiveID) && has(closedRow.SessionIDs, sidebarClosedID) && has(snap.ArchivedSessionIDs, sidebarArchiveID) && seen[sidebarOpenID].Title == "Visible session needle" && seen[sidebarClosedID].Title == "Closed session needle" && seen[sidebarArchiveID].Title == "Archived session needle", nil
		})
		if err != nil {
			finish(fmt.Errorf("fixture/list readiness: %w; snapshot=%+v", err, sessions.Snapshot()))
			return
		}
		err = sidebarWait(ctx, func() (bool, error) { return sessions.WorkspaceLayoutScope() != "", nil })
		if err != nil {
			finish(fmt.Errorf("workspace preference binding: %w", err))
			return
		}
		if phase == "initial" {
			sessions.SetWorkspaceGrouping("workspace")
			if sessions.Snapshot().WorkspaceLayout.Collapsed[openRow.WorkspaceID] {
				sessions.ToggleWorkspaceGroup(openRow.WorkspaceID)
			}
			if !sessions.Snapshot().WorkspaceLayout.Collapsed[closedRow.WorkspaceID] {
				sessions.ToggleWorkspaceGroup(closedRow.WorkspaceID)
			}
		}
		beforeFilter := "default"
		if phase == "restart" {
			beforeFilter = "show"
		}
		before, err := sidebarRecordCurrent(root, sessions, openRow.WorkspaceID, closedRow.WorkspaceID, beforeFilter, nil)
		if err != nil {
			finish(err)
			return
		}
		receipt["before"] = before
		beforeSelection := before["selectedSessionID"].(string)
		receipt["startupSelectedSyntheticID"] = beforeSelection
		receipt["startupSelectionPresent"] = beforeSelection != ""
		state.revealMainConversation()
		win.ShowInactive()
		handle := win.NativeHandle()
		if handle == 0 {
			finish(errors.New("owned window native handle absent"))
			return
		}
		ready := map[string]any{"pid": os.Getpid(), "hwnd": uint64(handle), "phase": phase, "workspaceOpen": openRow.WorkspaceID, "workspaceClosed": closedRow.WorkspaceID, "windowShownInactive": true, "origin": ownedOrigin, "facts": before}
		if err = sidebarWriteJSON(filepath.Join(root, phase+"-ready.json"), ready); err != nil {
			finish(err)
			return
		}
		err = sidebarWaitFile(ctx, filepath.Join(root, phase+"-driver-done.json"))
		if err != nil {
			finish(fmt.Errorf("UIA driver did not finish: %w", err))
			return
		}
		raw, err := os.ReadFile(filepath.Join(root, phase+"-driver-done.json"))
		if err != nil {
			finish(err)
			return
		}
		var driver struct {
			Pass  bool   `json:"pass"`
			Error string `json:"error"`
		}
		if err = json.Unmarshal(raw, &driver); err != nil || !driver.Pass {
			finish(fmt.Errorf("UIA driver failed: %s (%v)", driver.Error, err))
			return
		}
		afterFilter := "show"
		if phase == "restart" {
			afterFilter = "default"
		}
		after, err := sidebarRecordCurrent(root, sessions, openRow.WorkspaceID, closedRow.WorkspaceID, afterFilter, &beforeSelection)
		if err != nil {
			finish(err)
			return
		}
		receipt["after"] = after
		receipt["selectionUnchangedDuringDriver"] = true
		var list api.SessionListValue
		if err = call("session/list", map[string]any{"_request": api.SessionListRequest{}}, &list); err != nil {
			finish(err)
			return
		}
		ids := map[string]bool{}
		for _, row := range list.Items {
			ids[row.SessionId] = true
		}
		if len(ids) != 3 || !ids[sidebarOpenID] || !ids[sidebarClosedID] || !ids[sidebarArchiveID] {
			finish(fmt.Errorf("unexpected Host sessions: %v", ids))
			return
		}
		receipt["hostListIDs"] = ids
		receipt["driverChecks"] = true
		if phase == "initial" {
			if png, err := win.CapturePage(); err == nil {
				_ = os.WriteFile(filepath.Join(root, "after-interaction-content.png"), png, 0600)
				receipt["contentCaptureSHA256"] = sidebarDigest(png)
				receipt["captureLayer"] = "MyGo software Content scene, not screen framebuffer"
			}
		}
		finish(nil)
	}()
	main()
	select {
	case e := <-done:
		return e
	case <-time.After(2 * time.Second):
		return errors.New("desktop exited before acceptance receipt")
	}
}

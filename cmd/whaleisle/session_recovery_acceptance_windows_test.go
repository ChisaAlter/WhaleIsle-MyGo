//go:build windows && sessionrecoveryacceptance

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/files"
)

var nativeSessionRecoveryError error

// App.Run must remain on the process main goroutine. Each invocation owns one
// hidden desktop, one fresh profile and no prompt/credential operation.
func TestMain(m *testing.M) {
	if len(os.Args) == 3 && os.Args[1] == nativeRecoveryFlag {
		os.Exit(0)
	}
	nativeSessionRecoveryError = runNativeSessionRecovery()
	os.Exit(m.Run())
}
func TestNativeSessionRecovery(t *testing.T) {
	if nativeSessionRecoveryError != nil {
		t.Fatal(nativeSessionRecoveryError)
	}
}

func runNativeSessionRecovery() error {
	phase, id, workspace, out := os.Getenv("WHALEISLE_SESSION_QA_PHASE"), os.Getenv("WHALEISLE_SESSION_QA_ID"), os.Getenv("WHALEISLE_SESSION_QA_WORKSPACE"), os.Getenv("WHALEISLE_SESSION_QA_OUT")
	data := os.Getenv(paths.EnvUserData)
	if !isolatedDesktopRun() || !strings.HasPrefix(id, "session-recovery-fixture-") || !filepath.IsAbs(data) || !filepath.IsAbs(workspace) || !filepath.IsAbs(out) || (phase != "save" && phase != "restore") {
		return fmt.Errorf("explicit owned session recovery environment required")
	}
	if phase == "save" {
		cfg, err := config.Open(data)
		if err != nil {
			return err
		}
		if err = cfg.Update(func(c *config.Config) {
			c.Workspace = workspace
			c.Host = "127.0.0.1"
			c.Port = 32182
			c.NodeBin = os.Getenv("WHALEISLE_NODE")
			c.AutoDownloadUpdates = false
		}); err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(workspace, "owned-note.txt"), []byte("owned disk bytes\n"), 0600); err != nil {
			return err
		}
	}
	result := make(chan error, 1)
	finish := func(err error) {
		result <- err
		state.mu.Lock()
		sup := state.sup
		state.mu.Unlock()
		if sup != nil {
			sup.Stop()
		}
		state.mu.Lock()
		state.quitting = true
		state.quitCleanupFinished = true
		state.mu.Unlock()
		mygo.App.Quit()
	}
	go func() {
		deadline := time.Now().Add(40 * time.Second)
		revealed, rendered, selected, projected, shown, patchedDevTools, hydratedRendered := false, false, false, false, false, false, false
		for time.Now().Before(deadline) {
			state.mu.Lock()
			win, status, sessions, ready := state.win, state.bootStatus, state.sessions, state.dshMuxReady
			resumeError := state.resumeError
			state.mu.Unlock()
			if status.State == harness.Failed {
				finish(fmt.Errorf("owned native Host failed"))
				return
			}
			if resumeError != "" {
				finish(fmt.Errorf("native resume refused: %s", resumeError))
				return
			}
			if win == nil || status.State != harness.Ready || sessions == nil || !ready {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !revealed {
				state.revealMainConversation()
				revealed = true
			}
			snap := sessions.Snapshot()
			if snap.ListLoading || snap.ListErr != "" || !snap.WorkspaceReady {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !shown {
				win.ShowInactive()
				shown = true
			}
			// Hidden windows do not receive a natural first WM_PAINT. Render the
			// production mainView so its normal resume-selection callback runs.
			if !rendered {
				win.Invalidate()
				if _, err := win.CapturePage(); err != nil {
					finish(err)
					return
				}
				rendered = true
				continue
			}
			if phase == "save" && !selected {
				sessions.Select(id)
				selected = true
				continue
			}
			if snap.SelectedID != id || !snap.HistoryReady || snap.HistoryLoading {
				win.Invalidate()
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !projected {
				win.Invalidate()
				if _, err := win.CapturePage(); err != nil {
					finish(err)
					return
				}
				want := workspaceSessionKey(sessions.WorkspaceLayoutScope(), workspace, id)
				state.mu.Lock()
				bound := state.workspaceSessions.active == want && state.workspaceSessions.BySession[want] != nil
				state.mu.Unlock()
				if !bound {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				projected = true
				continue
			}
			state.mu.Lock()
			setStore := state.settings
			state.mu.Unlock()
			if setStore == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			setStore.EnsureLoaded(context.Background())
			sv := setStore.Snapshot().Value
			if sv.Err != "" {
				finish(fmt.Errorf("settings store failed: %s", sv.Err))
				return
			}
			if !sv.Described || !sv.HasDocument || sv.Loading || sv.Saving || sv.PreferencePending["ui-settings.enabled"] {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if phase == "save" && !settingsstore.DeveloperToolsEnabled(sv) {
				if !patchedDevTools {
					setStore.PatchBool(context.Background(), "ui-settings", "enabled", true)
					patchedDevTools = true
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if phase == "restore" && !hydratedRendered {
				win.Invalidate()
				if _, err := win.CapturePage(); err != nil {
					finish(err)
					return
				}
				hydratedRendered = true
				continue
			}
			if phase == "save" {
				done := make(chan error, 1)
				win.Update(func() {
					state.mu.Lock()
					draft := state.composerDraftLocked(id)
					draft.Text = "owned native unsent draft"
					draft.Attachments = []session.ComposerAttachment{{Path: filepath.Join(workspace, "owned-note.txt"), Name: "owned-note.txt"}}
					state.layout.Sidebar = 300
					state.layout.SidebarExpanded = 300
					state.layout.TranscriptWidth = 860
					state.layout.Right = 450
					state.layout.RightTree = 210
					state.filesPaneOpen = true
					state.rightPaneTab = "files"
					state.chatPaneTab = "trajectory"
					state.todoPanelCollapsed = true
					state.fileTabs = files.Tabs{Root: workspace, Workspaces: map[string]*files.WorkspaceTabs{workspace: {Active: "owned-note.txt", Open: []*files.FileTab{{Path: "owned-note.txt", Preview: "owned disk bytes\n", Draft: "owned unsaved file buffer\n", Editing: true}}}}}
					state.captureWorkspaceSessionLocked()
					state.mu.Unlock()
					done <- state.saveResume()
				})
				if err := <-done; err != nil {
					finish(err)
					return
				}
			} else {
				state.mu.Lock()
				draft := state.composerDrafts[id]
				draftOK := draft != nil && draft.Text == "owned native unsent draft" && len(draft.Attachments) == 1 && draft.Attachments[0].Path == filepath.Join(workspace, "owned-note.txt")
				layoutOK := state.layout.Sidebar == 300 && state.layout.TranscriptWidth == 860 && state.layout.Right == 450 && state.filesPaneOpen && state.rightPaneTab == "files" && state.chatPaneTab == "trajectory" && state.todoPanelCollapsed
				filesOK := false
				if w := state.fileTabs.Workspaces[workspace]; w != nil && w.Active == "owned-note.txt" {
					for _, tab := range w.Open {
						if tab.Path == "owned-note.txt" && tab.Draft == "owned unsaved file buffer\n" {
							filesOK = true
						}
					}
				}
				state.mu.Unlock()
				if !draftOK || !layoutOK || !filesOK {
					finish(fmt.Errorf("native restore mismatch: draft=%v layout=%v (sidebar=%v tw=%v right=%v open=%v rightTab=%q chatTab=%q todo=%v) fileTabs=%v", draftOK, layoutOK, state.layout.Sidebar, state.layout.TranscriptWidth, state.layout.Right, state.filesPaneOpen, state.rightPaneTab, state.chatPaneTab, state.todoPanelCollapsed, filesOK))
					return
				}
				bytes, err := os.ReadFile(filepath.Join(workspace, "owned-note.txt"))
				if err != nil || string(bytes) != "owned disk bytes\n" {
					finish(fmt.Errorf("reopening flushed an unsaved editor buffer"))
					return
				}
			}
			contentWidth, contentHeight := win.ContentSize()
			state.mu.Lock()
			fileMode := state.fileEditViewMode
			activePane := state.workspaceSessions.active
			geometry := map[string]any{"sidebar": state.layout.Sidebar, "transcriptWidth": state.layout.TranscriptWidth, "right": state.layout.Right, "rightTree": state.layout.RightTree}
			state.mu.Unlock()
			if fileMode != files.EditViewEdit {
				finish(fmt.Errorf("fresh native source editor mode = %d, want Edit only", fileMode))
				return
			}
			proof := map[string]any{"phase": phase, "host": "ready", "historyReady": snap.HistoryReady, "messages": len(snap.Messages), "selectedOwnedID": id, "selectedWorkspaceProjectionRendered": projected, "modelRequests": 0, "productionSaveResume": phase == "save", "productionOpenResume": phase == "restore", "draftAndAttachment": true, "viewLayout": true, "fileTabsAndUnsavedBuffer": true, "workspaceDiskUntouched": true, "windowShownWithoutFocus": shown, "contentSizeDIP": []int{contentWidth, contentHeight}}
			proof["sourceFileEditViewMode"] = fileMode
			proof["selectedWorkspacePaneKey"] = activePane
			proof["adapterLayoutStateDIP"] = geometry // Persisted state, not measured native node bounds.
			if os.Getenv("WHALEISLE_SESSION_QA_UPDATE_RESPONSIVENESS") == "1" {
				observation, err := probeLauncherUpdateResponsiveness(win)
				if err != nil {
					finish(err)
					return
				}
				proof["launcherUpdateResponsiveness"] = observation
			}
			if value := os.Getenv("WHALEISLE_SESSION_QA_STABILITY_SECONDS"); value != "" {
				seconds, err := strconv.Atoi(value)
				if err != nil || seconds < 1 || seconds > 120 {
					finish(fmt.Errorf("owned stability observation requires1..120 seconds"))
					return
				}
				started := time.Now()
				for time.Since(started) < time.Duration(seconds)*time.Second {
					if win.IsDestroyed() {
						finish(fmt.Errorf("owned restored native window closed during stability observation"))
						return
					}
					time.Sleep(250 * time.Millisecond)
				}
				proof["nativeWindowStabilitySeconds"] = seconds
				proof["stabilityScope"] = "Owned restored main window idle observation; no OS input, no model request, not proof of the user's unobserved exit cause"
			}
			if phase == "restore" {
				png, err := win.CapturePage()
				if err != nil {
					finish(err)
					return
				}
				if err = os.WriteFile(strings.TrimSuffix(out, ".json")+".png", png, 0600); err != nil {
					finish(err)
					return
				}
			}
			raw, _ := json.MarshalIndent(proof, "", "  ")
			err := os.WriteFile(out, append(raw, '\n'), 0600)
			finish(err)
			return
		}
		state.mu.Lock()
		pendSess, resWs, activeKey := state.resumePendingSession, state.resumeWorkspace, state.workspaceSessions.active
		bootSt := state.bootStatus.State.String()
		muxOK := state.dshMuxReady
		var snap session.ViewState
		var scope string
		if state.sessions != nil {
			snap = state.sessions.Snapshot()
			scope = state.sessions.WorkspaceLayoutScope()
		}
		state.mu.Unlock()
		finish(fmt.Errorf("owned native session recovery readiness timeout: phase=%s boot=%s mux=%v revealed=%v shown=%v rendered=%v projected=%v wsReady=%v listLoad=%v listErr=%q sel=%q want=%q histReady=%v histLoad=%v lastAction=%q resumePend=%q resumeWs=%q activeKey=%q scope=%q items=%d",
			phase, bootSt, muxOK, revealed, shown, rendered, projected, snap.WorkspaceReady, snap.ListLoading, snap.ListErr, snap.SelectedID, id, snap.HistoryReady, snap.HistoryLoading, snap.LastAction, pendSess, resWs, activeKey, scope, len(snap.Items)))
	}()
	main()
	select {
	case err := <-result:
		return err
	case <-time.After(10 * time.Second):
		return fmt.Errorf("native desktop exited before recovery proof")
	}
}

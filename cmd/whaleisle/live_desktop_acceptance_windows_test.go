//go:build windows && livedesktopacceptance

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	pngcodec "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
)

var liveDesktopAcceptanceErr error

func TestMain(m *testing.M) {
	liveDesktopAcceptanceErr = runLiveDesktopAcceptance()
	os.Exit(m.Run())
}

func TestLiveDesktopAcceptance(t *testing.T) {
	if liveDesktopAcceptanceErr != nil {
		t.Fatal(liveDesktopAcceptanceErr)
	}
}

func liveWaitFor(what string, duration time.Duration, ready func() bool) error {
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if ready() {
			return nil
		}
		time.Sleep(80 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", what)
}

func runLiveDesktopAcceptance() error {
	out := os.Getenv("WHALEISLE_LIVE_QA_OUT")
	workspace := os.Getenv("WHALEISLE_LIVE_QA_WORKSPACE")
	data := os.Getenv(paths.EnvUserData)
	if !isolatedDesktopRun() || !filepath.IsAbs(out) || !filepath.IsAbs(workspace) || !filepath.IsAbs(data) {
		return fmt.Errorf("explicit isolated paths required: out=%q workspace=%q data=%q", out, workspace, data)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("# Live Desktop QA Workspace\n"), 0o644); err != nil {
		return err
	}
	// Remove stale resume snapshot and startup failure marker so cold-start layout and auto-start are tested cleanly.
	_ = os.Remove(filepath.Join(data, "go-resume.json"))
	_ = os.Remove(filepath.Join(data, "go-resume.json.lock"))
	_ = os.Remove(filepath.Join(data, "last-desktop-start.json"))
	// Seed custom Whale Assistant name ("Claude Duts") in dsh-home/data/whale/settings.json to match user's real profile
	whaleDataDir := filepath.Join(paths.DshHome(), "data", "whale")
	_ = os.MkdirAll(whaleDataDir, 0o755)
	_ = os.WriteFile(filepath.Join(whaleDataDir, "settings.json"), []byte("{\n  \"name\": \"Claude Duts\",\n  \"enabled\": true\n}\n"), 0o644)
	cfg, err := config.Open(data)
	if err != nil {
		return err
	}
	if err = cfg.Update(func(v *config.Config) {
		v.Workspace = workspace
		v.Host = "127.0.0.1"
		v.Port = 32189
		v.Locale = "zh"
		v.NodeBin = os.Getenv("WHALEISLE_NODE")
		v.HarnessAutoRestart = false
	}); err != nil {
		return err
	}

	result := make(chan error, 1)
	finish := func(err error) {
		select {
		case result <- err:
		default:
		}
		state.mu.Lock()
		state.quitting = true
		sup := state.sup
		win := state.win
		state.mu.Unlock()
		state.suspendPluginContributions()
		state.closePluginNativeView()
		state.closeTerminal()
		if sup != nil {
			sup.Stop()
		}
		go func() {
			time.Sleep(2500 * time.Millisecond)
			if err != nil {
				fmt.Fprintf(os.Stderr, "LIVE DESKTOP ACCEPTANCE FAILED: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}()
		if win != nil {
			mygo.RunOnMain(func() {
				win.Destroy()
				go func() {
					time.Sleep(300 * time.Millisecond)
					mygo.App.Quit()
				}()
			})
		} else {
			mygo.App.Quit()
		}
	}

	go func() {
		var bootErr error
		if err := liveWaitFor("production Host/mux/workspace ready", 105*time.Second, func() bool {
			state.mu.Lock()
			win, status, sessions, mux := state.win, state.bootStatus, state.sessions, state.dshMuxReady
			state.mu.Unlock()
			if status.State == harness.Failed {
				bootErr = fmt.Errorf("harness failed during boot: %+v", status)
				return true
			}
			if win == nil || status.State != harness.Ready || sessions == nil || !mux {
				return false
			}
			snap := sessions.Snapshot()
			return snap.ClientReady && snap.WorkspaceReady && !snap.ListLoading && snap.ListErr == ""
		}); err != nil {
			finish(err)
			return
		}
		if bootErr != nil {
			finish(bootErr)
			return
		}

		state.mu.Lock()
		win := state.win
		sessions := state.sessions
		state.mu.Unlock()

		// Wait for InitializeDefaultWorkspace + maybeStartupBlankSession to settle
		_ = liveWaitFor("default workspace and initial session", 12*time.Second, func() bool {
			snap := sessions.Snapshot()
			return len(snap.Workspaces) > 0 && snap.SelectedID != ""
		})
		bootSnap := sessions.Snapshot()
		fmt.Printf("[LIVE QA BOOT SNAP] Workspaces=%d SelectedID=%q WorkspaceCwd=%q LastAction=%q\n",
			len(bootSnap.Workspaces), bootSnap.SelectedID, sessions.WorkspaceCwd(), bootSnap.LastAction)

		win.ShowInactive()

		var drv *ui.WindowDriver
		mygo.RunOnMain(func() {
			drv = ui.DriveWindow(win)
			if drv != nil {
				drv.Frame()
			}
		})
		if drv == nil {
			finish(fmt.Errorf("ui.DriveWindow(win) returned nil"))
			return
		}

		stepsLog := []map[string]any{}
		capture := func(stepName string) error {
			mygo.RunOnMain(func() {
				drv.Frame()
			})
			win.Invalidate()
			time.Sleep(150 * time.Millisecond)
			mygo.RunOnMain(func() {
				drv.Frame()
			})
			png, err := win.CapturePage()
			if err != nil {
				return fmt.Errorf("CapturePage(%s): %w", stepName, err)
			}
			cfg, err := pngcodec.DecodeConfig(bytes.NewReader(png))
			if err != nil {
				return fmt.Errorf("DecodeConfig(%s): %w", stepName, err)
			}
			file := filepath.Join(out, stepName+".png")
			if err := os.WriteFile(file, png, 0o644); err != nil {
				return err
			}
			var texts []string
			var cursor ui.Cursor
			mygo.RunOnMain(func() {
				texts = drv.Texts()
				cursor = drv.Cursor()
			})
			stepsLog = append(stepsLog, map[string]any{
				"step":       stepName,
				"screenshot": file,
				"size":       []int{cfg.Width, cfg.Height},
				"cursor":     int(cursor),
				"textCount":  len(texts),
			})
			return nil
		}

		clickAny := func(labels ...string) error {
			var err error
			mygo.RunOnMain(func() {
				drv.Frame()
				for _, l := range labels {
					if r, ok := drv.Find(l); ok && r.W > 0 && r.H > 0 {
						drv.ClickAt(r.X+r.W/2, r.Y+r.H/2)
						err = nil
						return
					}
				}
				err = fmt.Errorf("none of %v found in live window texts: %v", labels, drv.Texts())
			})
			win.Invalidate()
			return err
		}

		hoverAny := func(labels ...string) (ui.Cursor, error) {
			var cur ui.Cursor
			var err error
			mygo.RunOnMain(func() {
				drv.Frame()
				for _, l := range labels {
					if r, ok := drv.Find(l); ok && r.W > 0 && r.H > 0 {
						drv.Move(r.X+r.W/2, r.Y+r.H/2)
						cur = drv.Cursor()
						err = nil
						return
					}
				}
				err = fmt.Errorf("none of %v found for hover in live window texts: %v", labels, drv.Texts())
			})
			win.Invalidate()
			return cur, err
		}

		hasAnyText := func(substrings ...string) bool {
			var found bool
			mygo.RunOnMain(func() {
				drv.Frame()
				for _, sub := range substrings {
					if drv.HasText(sub) {
						found = true
						return
					}
				}
			})
			return found
		}

		// 1. Main conversation initial state + hover cursor check on 新会话 (sidebar new session button)
		if cur, err := hoverAny("新会话"); err != nil {
			finish(fmt.Errorf("step 1 hover 新会话: %w", err))
			return
		} else if cur != ui.CursorPointer {
			finish(fmt.Errorf("step 1 expected CursorPointer on 新会话, got %v", cur))
			return
		}
		if err := capture("01-main-conversation"); err != nil {
			finish(err)
			return
		}

		// 2. Click 新会话 on the real window -> verify real session created/selected via deepseek-harness
		if err := clickAny("新会话"); err != nil {
			finish(fmt.Errorf("step 2 click 新会话: %w", err))
			return
		}
		if err := liveWaitFor("initial session created in deepseek-harness", 15*time.Second, func() bool {
			return strings.TrimSpace(sessions.Snapshot().SelectedID) != ""
		}); err != nil {
			finish(err)
			return
		}
		firstSessionID := strings.TrimSpace(sessions.Snapshot().SelectedID)

		// Put a draft into the current session composer in the live window and verify clicking 新会话 creates and switches to a new blank session!
		state.mu.Lock()
		if d := state.composerDraftLocked(firstSessionID); d != nil {
			d.Text = "这是一条未发送的测试草稿，用于验证新会话切换"
		}
		state.mu.Unlock()
		win.Invalidate()
		if err := capture("02-composer-draft"); err != nil {
			finish(err)
			return
		}
		// Click the sidebar "新会话" button specifically (the first "新会话" rect is in the sidebar)
		mygo.RunOnMain(func() {
			drv.Frame()
			rects := drv.FindAll("新会话")
			if len(rects) > 0 {
				r := rects[0]
				drv.ClickAt(r.X+r.W/2, r.Y+r.H/2)
			}
		})
		win.Invalidate()
		if err := liveWaitFor("second session created and switched", 15*time.Second, func() bool {
			curID := strings.TrimSpace(sessions.Snapshot().SelectedID)
			return curID != "" && curID != firstSessionID
		}); err != nil {
			finish(fmt.Errorf("step 2b new session switch failed (firstSessionID=%q, selectedID=%q): %w", firstSessionID, sessions.Snapshot().SelectedID, err))
			return
		}
		if err := capture("03-new-session-switched"); err != nil {
			finish(err)
			return
		}

		// 3. Test Directory Picker against real deepseek-harness directoryPicker/list RPC
		state.openDirectoryPicker(sessions, win, "workspace")
		if err := liveWaitFor("directory picker modal open with real directoryPicker/list entries", 15*time.Second, func() bool {
			return hasAnyText("选择工作区目录", "浏览目录", "当前路径", "取消")
		}); err != nil {
			var liveTexts []string
			mygo.RunOnMain(func() { liveTexts = drv.Texts() })
			finish(fmt.Errorf("step 3 directory picker did not open; live texts=%v: %w", liveTexts, err))
			return
		}
		if err := capture("04-directory-picker"); err != nil {
			finish(err)
			return
		}
		_ = clickAny("取消", "关闭")

		// Transition the active session to a titled conversation via real session/setPresentation RPC
		// so ConversationSessionHeader renders the active session header (团队, 更多操作, 对话, 轨迹).
		sessions.SetPresentation(context.Background(), sessions.SelectedSessionID(), "实机验收会话")
		if err := liveWaitFor("session presentation header visible", 10*time.Second, func() bool {
			return hasAnyText("团队")
		}); err != nil {
			var liveTexts []string
			mygo.RunOnMain(func() { liveTexts = drv.Texts() })
			finish(fmt.Errorf("step 3b session header did not appear after SetPresentation; live texts=%v: %w", liveTexts, err))
			return
		}

		// 4. Test Conversation Header "团队" (conversation-agents) hover + click popover against real session/projections RPC
		if cur, err := hoverAny("团队"); err != nil {
			finish(fmt.Errorf("step 4 hover 团队: %w", err))
			return
		} else if cur != ui.CursorPointer {
			finish(fmt.Errorf("step 4 expected CursorPointer on 团队, got %v", cur))
			return
		}
		if err := clickAny("团队"); err != nil {
			finish(fmt.Errorf("step 4 click 团队: %w", err))
			return
		}
		if err := liveWaitFor("real ordinaryTeam session/projections loaded and rendered in popover", 15*time.Second, func() bool {
			return hasAnyText("成员", "lead", "共享任务")
		}); err != nil {
			var liveTexts []string
			mygo.RunOnMain(func() { liveTexts = drv.Texts() })
			finish(fmt.Errorf("step 4 团队 popover did not render 成员/lead/共享任务; live window texts=%v: %w", liveTexts, err))
			return
		}
		if hasAnyText("Team 暂不可用") {
			finish(fmt.Errorf("step 4 团队 popover showed error 'Team 暂不可用'"))
			return
		}
		if err := capture("05-team-popover-open"); err != nil {
			finish(err)
			return
		}
		// Click outside to close the pinned 团队 popover
		mygo.RunOnMain(func() {
			drv.ClickAt(640, 400)
		})
		win.Invalidate()

		// 5. Test Conversation Header "轨迹" and "对话" tabs
		if err := clickAny("轨迹"); err != nil {
			finish(fmt.Errorf("step 5 click 轨迹: %w", err))
			return
		}
		if err := capture("06-trajectory-tab"); err != nil {
			finish(err)
			return
		}
		if err := clickAny("对话"); err != nil {
			finish(fmt.Errorf("step 5 click 对话: %w", err))
			return
		}

		// 6. Test Right Dock ("切换右侧栏" / "展开右侧栏" / "切换终端抽屉")
		if err := clickAny("切换右侧栏", "展开右侧栏"); err != nil {
			finish(fmt.Errorf("step 6 click 切换右侧栏: %w", err))
			return
		}
		_ = liveWaitFor("right pane kinds registered", 3*time.Second, func() bool {
			var ok bool
			mygo.RunOnMain(func() {
				drv.Frame()
				ok = drv.HasText("新建终端") && drv.HasText("工作区差异")
			})
			return ok
		})
		if err := capture("07-right-pane-open"); err != nil {
			finish(err)
			return
		}
		if err := clickAny("切换终端抽屉"); err == nil {
			if err := capture("08-terminal-drawer-open"); err != nil {
				finish(err)
				return
			}
			_ = clickAny("切换终端抽屉")
		}
		// Close right dock
		_ = clickAny("切换右侧栏", "收起右侧栏")

		// 7. Test Sidebar Navigation: first click "Claude Duts" / "鲸鱼娘" DIRECTLY from the main conversation page!
		if cur, err := hoverAny("Claude Duts", "鲸鱼娘"); err != nil {
			finish(fmt.Errorf("step 7a hover Claude Duts/鲸鱼娘 from main conversation: %w", err))
			return
		} else if cur != ui.CursorPointer {
			finish(fmt.Errorf("step 7a expected CursorPointer on Claude Duts/鲸鱼娘, got %v", cur))
			return
		}
		if err := clickAny("Claude Duts", "鲸鱼娘"); err != nil {
			finish(fmt.Errorf("step 7a click Claude Duts/鲸鱼娘 from main conversation: %w", err))
			return
		}
		if err := liveWaitFor("whale assistant session active from main conversation", 10*time.Second, func() bool {
			var ok bool
			mygo.RunOnMain(func() {
				drv.Frame()
				snap := sessions.Snapshot()
				if !strings.HasPrefix(snap.SelectedID, "session-whale-") {
					return
				}
				ok = !drv.HasText("选择一个工作区开始") && !drv.HasText("探索未至之鲸") && !drv.HasText("选择会话后开始对话") && len(drv.FindAll("Claude Duts")) >= 2 && len(drv.FindAll("助理设置")) > 0
			})
			return ok
		}); err != nil {
			var liveTexts []string
			mygo.RunOnMain(func() { liveTexts = drv.Texts() })
			finish(fmt.Errorf("step 7a whale assistant session did not open from main conversation; selectedID=%q live texts=%v: %w", sessions.Snapshot().SelectedID, liveTexts, err))
			return
		}

		if err := clickAny("插件"); err != nil {
			finish(fmt.Errorf("step 7 click 插件: %w", err))
			return
		}
		_ = liveWaitFor("plugin manager packages and contributions loaded", 10*time.Second, func() bool {
			mygo.RunOnMain(func() { drv.Frame() })
			ctrl := state.nativePluginController()
			snap := ctrl.Snapshot()
			state.mu.Lock()
			contrib := state.nativePlugins.contributions
			state.mu.Unlock()
			return snap.Status != "loading" && (contrib == nil || contrib.Snapshot().Connected)
		})
		if err := capture("09-marketplace"); err != nil {
			finish(err)
			return
		}

		if err := clickAny("自动化任务"); err != nil {
			finish(fmt.Errorf("step 7 click 自动化任务: %w", err))
			return
		}
		_ = liveWaitFor("schedule catalog loaded", 10*time.Second, func() bool {
			mygo.RunOnMain(func() { drv.Frame() })
			state.mu.Lock()
			sched := state.schedule
			state.mu.Unlock()
			return sched != nil && !sched.Snapshot().Loading
		})
		if err := capture("10-automation"); err != nil {
			finish(err)
			return
		}

		// Verify persisted session-whale-* exists on disk under dsh-home/sessions/*/<sid> before clicking Claude Duts
		sessions.Select(firstSessionID)
		if diskSID, diskName, _ := (*mainState)(nil).whaleAssistantPersistedInfo(); !strings.HasPrefix(diskSID, "session-whale-") || diskName != "Claude Duts" {
			finish(fmt.Errorf("step 7 expected persisted whale session on disk before click, got sid=%q name=%q", diskSID, diskName))
			return
		}
		if err := clickAny("Claude Duts", "鲸鱼娘"); err != nil {
			finish(fmt.Errorf("step 7 click Claude Duts/鲸鱼娘: %w", err))
			return
		}
		if err := liveWaitFor("whale assistant session active and non-inert", 10*time.Second, func() bool {
			var ok bool
			mygo.RunOnMain(func() {
				drv.Frame()
				snap := sessions.Snapshot()
				if !strings.HasPrefix(snap.SelectedID, "session-whale-") {
					return
				}
				ok = !drv.HasText("选择一个工作区开始") && !drv.HasText("探索未至之鲸") && !drv.HasText("选择会话后开始对话") && len(drv.FindAll("Claude Duts")) >= 2 && len(drv.FindAll("助理设置")) > 0
			})
			return ok
		}); err != nil {
			var liveTexts []string
			mygo.RunOnMain(func() { liveTexts = drv.Texts() })
			finish(fmt.Errorf("step 7 whale assistant session did not enter active non-inert state; live texts=%v: %w", liveTexts, err))
			return
		}
		if err := capture("11-assistant"); err != nil {
			finish(err)
			return
		}
		// Switch back to the ordinary workspace session before opening Settings
		sessions.Select(firstSessionID)
		_ = liveWaitFor("ordinary workspace session restored", 5*time.Second, func() bool {
			mygo.RunOnMain(func() { drv.Frame() })
			return sessions.Snapshot().SelectedID == firstSessionID
		})

		// 8. Test Settings Modal ("设置与账号" / "账号菜单" -> "设置" — All Sections)
		if err := clickAny("设置与账号", "账号菜单", "设置"); err != nil {
			finish(fmt.Errorf("step 8 click 设置与账号/账号菜单: %w", err))
			return
		}
		// Clicking 设置与账号/账号菜单 opens the account menu popup containing "设置"; click "设置" if settingsOpen is still false
		time.Sleep(150 * time.Millisecond)
		state.mu.Lock()
		isSettingsOpen := state.settingsOpen
		state.mu.Unlock()
		if !isSettingsOpen {
			_ = clickAny("设置", "打开设置")
		}
		if err := liveWaitFor("real settings/describe loaded", 15*time.Second, func() bool {
			state.mu.Lock()
			store := state.settings
			open := state.settingsOpen
			state.mu.Unlock()
			if !open || store == nil {
				return false
			}
			snap := store.Snapshot()
			return !snap.Loading && len(snap.Namespaces) > 0 && snap.Err == ""
		}); err != nil {
			var liveTexts []string
			mygo.RunOnMain(func() { liveTexts = drv.Texts() })
			finish(fmt.Errorf("step 8 settings modal did not open/load; live texts=%v: %w", liveTexts, err))
			return
		}
		if err := capture("12-settings-general"); err != nil {
			finish(err)
			return
		}

		settingsPages := []struct {
			file   string
			labels []string
		}{
			{"13-settings-usage", []string{"用量与费用"}},
			{"14-settings-appearance", []string{"外观"}},
			{"15-settings-interface", []string{"界面设置", "界面"}},
			{"16-settings-conversation", []string{"对话"}},
			{"17-settings-models", []string{"模型"}},
			{"18-settings-agents", []string{"Agent 预设", "智能体预设"}},
			{"19-settings-skills", []string{"技能"}},
			{"20-settings-mcp", []string{"MCP"}},
			{"21-settings-builtin-plugins", []string{"内置插件"}},
			{"22-settings-market", []string{"市场"}},
			{"23-settings-builtin", []string{"内置功能"}},
			{"24-settings-whale", []string{"鲸鱼娘"}},
			{"25-settings-im", []string{"手机远程"}},
			{"26-settings-remote", []string{"远程工作区"}},
			{"27-settings-about", []string{"关于"}},
		}
		for _, pg := range settingsPages {
			// Click the last matching label in the window (since modal overlays are rendered after background sidebar)
			var clickErr error
			mygo.RunOnMain(func() {
				drv.Frame()
				for _, l := range pg.labels {
					rects := drv.FindAll(l)
					if len(rects) > 0 {
						r := rects[len(rects)-1]
						if r.Y+r.H > 800 {
							drv.Scroll(r.X+r.W/2, 400, 0, 240)
							drv.Frame()
							if next := drv.FindAll(l); len(next) > 0 {
								r = next[len(next)-1]
							}
						}
						drv.ClickAt(r.X+r.W/2, r.Y+r.H/2)
						clickErr = nil
						return
					}
				}
				clickErr = fmt.Errorf("none of %v found in settings modal texts: %v", pg.labels, drv.Texts())
			})
			win.Invalidate()
			if clickErr != nil {
				finish(fmt.Errorf("step 8 settings section %v: %w", pg.labels, clickErr))
				return
			}
			time.Sleep(200 * time.Millisecond)
			if err := capture(pg.file); err != nil {
				finish(err)
				return
			}
		}

		// Close settings via Escape key on the live window
		mygo.RunOnMain(func() {
			drv.Key(0, ui.KeyEscape)
		})
		win.Invalidate()
		if err := liveWaitFor("settings modal closed", 5*time.Second, func() bool {
			state.mu.Lock()
			open := state.settingsOpen
			state.mu.Unlock()
			return !open
		}); err != nil {
			finish(err)
			return
		}
		if err := capture("28-back-to-workspace"); err != nil {
			finish(err)
			return
		}

		raw, _ := json.MarshalIndent(map[string]any{
			"status":         "PASS",
			"firstSessionID": firstSessionID,
			"finalSessionID": sessions.Snapshot().SelectedID,
			"steps":          stepsLog,
		}, "", "  ")
		finish(os.WriteFile(filepath.Join(out, "live-desktop-report.json"), append(raw, '\n'), 0o644))
	}()

	main()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("desktop exited before live acceptance finished")
	}
}

//go:build windows && settingsacceptance

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	pngcodec "image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/i18n"
)

var settingsAcceptanceError error

// The tag is intentionally exclusive of the other native TestMain tags.
func TestMain(m *testing.M) { settingsAcceptanceError = nativeSettingsAcceptance(); os.Exit(m.Run()) }
func TestNativeSettingsAcceptance(t *testing.T) {
	if settingsAcceptanceError != nil {
		t.Fatal(settingsAcceptanceError)
	}
}

func settingsQAWait(what string, ready func() bool) error {
	return settingsQAWaitFor(what, 20*time.Second, ready)
}
func settingsQAWaitFor(what string, duration time.Duration, ready func() bool) error {
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if ready() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("settings QA readiness: %s", what)
}

func nativeSettingsAcceptance() error {
	out, workspace := os.Getenv("WHALEISLE_SETTINGS_QA_OUT"), os.Getenv("WHALEISLE_SETTINGS_QA_WORKSPACE")
	data := os.Getenv(paths.EnvUserData)
	if !isolatedDesktopRun() || !filepath.IsAbs(out) || !filepath.IsAbs(workspace) || !filepath.IsAbs(data) {
		return fmt.Errorf("explicit owned settings QA paths required")
	}
	if _, err := os.Stat(filepath.Join(data, "credentials.json")); !os.IsNotExist(err) {
		return fmt.Errorf("settings QA requires no credential document")
	}
	cfg, err := config.Open(data)
	if err != nil {
		return err
	}
	if err = cfg.Update(func(v *config.Config) {
		v.Workspace, v.Host, v.Port, v.Locale = workspace, "127.0.0.1", 32185, "en"
		v.NodeBin = os.Getenv("WHALEISLE_NODE")
		v.HarnessAutoRestart = false
	}); err != nil {
		return err
	}
	result := make(chan error, 1)
	finish := func(err error) {
		result <- err
		state.mu.Lock()
		state.quitting = true
		sup := state.sup
		state.mu.Unlock()
		if sup != nil {
			sup.Stop()
		}
		mygo.App.Quit()
	}
	go func() {
		if err := settingsQAWaitFor("production Host/mux/workspace", 105*time.Second, func() bool {
			state.mu.Lock()
			win, status, sessions, mux := state.win, state.bootStatus, state.sessions, state.dshMuxReady
			state.mu.Unlock()
			if win == nil || status.State != harness.Ready || sessions == nil || !mux {
				return false
			}
			snap := sessions.Snapshot()
			return snap.ClientReady && snap.WorkspaceReady && !snap.ListLoading && snap.ListErr == ""
		}); err != nil {
			finish(err)
			return
		}
		state.revealMainConversation()
		state.mu.Lock()
		win := state.win
		state.mu.Unlock()
		win.SetContentSize(1185, 792)
		win.ShowInactive()
		var tt *ui.Tester
		initialWidth, initialHeight := win.ContentSize()
		mygo.RunOnMain(func() { tt = ui.NewTester(mainView, initialWidth, initialHeight) })
		proofs := []map[string]any{}
		action := func(fn func() error) error {
			var err error
			mygo.RunOnMain(func() { err = fn() })
			win.Invalidate()
			return err
		}
		click := func(label string) error { return action(func() error { return tt.Click(label) }) }
		// Several rail labels also name background workspace controls. Start at
		// the unique General rail control and use the actual focus order, so an
		// unscoped label lookup cannot accidentally dismiss the modal mask.
		navigate := func(page string) error {
			steps, ok := map[string]int{"Appearance": 1, "Interface": 2, "Plugins": 4, "Skills": 5, "MCP": 7, "About": 13}[page]
			if !ok {
				return fmt.Errorf("unapproved settings page %q", page)
			}
			return action(func() error {
				if err := tt.Click("General"); err != nil {
					return err
				}
				for range steps {
					tt.Key(0, ui.KeyTab)
				}
				if !tt.Focused(page) {
					return fmt.Errorf("rail focus did not reach %s", page)
				}
				tt.Key(0, ui.KeyEnter)
				state.mu.Lock()
				active := map[string]bool{"Appearance": state.settingsAppearanceActive, "Interface": state.settingsInterfaceActive, "Plugins": state.settingsPluginsActive, "Skills": state.settingsSkillsActive, "MCP": state.settingsMCPActive, "About": state.settingsAboutActive}[page]
				open := state.settingsOpen
				state.mu.Unlock()
				if !open || !active {
					return fmt.Errorf("production settings route not active: %s", page)
				}
				return nil
			})
		}
		if only := os.Getenv("WHALEISLE_SETTINGS_QA_ONLY"); only != "" {
			if only != "narrow" {
				finish(fmt.Errorf("unapproved settings-only mode"))
				return
			}
			finish(settingsQANarrowDiagnostic(win, tt, out, action, navigate))
			return
		}
		capture := func(name string, previous []byte) ([]byte, error) {
			var png []byte
			// This is an observation after normal painting, not native input proof.
			win.Invalidate()
			time.Sleep(time.Second)
			err := settingsQAWait("native painted "+name, func() bool {
				var captureErr error
				png, captureErr = win.CapturePage()
				if captureErr != nil || len(png) == 0 || bytes.Equal(png, previous) {
					return false
				}
				_, err := pngcodec.DecodeConfig(bytes.NewReader(png))
				if err != nil {
					return false
				}
				return true
			})
			if err != nil {
				return nil, err
			}
			if err = os.WriteFile(filepath.Join(out, name+".png"), png, 0600); err != nil {
				return nil, err
			}
			cfg, err := pngcodec.DecodeConfig(bytes.NewReader(png))
			if err != nil {
				return nil, err
			}
			w, h := win.ContentSize()
			bounds := map[string]ui.Rect{}
			mygo.RunOnMain(func() {
				tt.Frame()
				for _, label := range []string{"General", "Appearance", "Interface", "Skills", "MCP", "About", i18n.T(i18n.KeySettingsBack)} {
					if rect, ok := tt.Find(label); ok {
						bounds[label] = rect
					}
				}
			})
			proofs = append(proofs, map[string]any{"name": name, "nativePNG": name + ".png", "contentSizeDIP": []int{w, h}, "imagePixels": []int{cfg.Width, cfg.Height}, "sha256": fmt.Sprintf("%x", sha256.Sum256(png)), "productionMainViewTesterBoundsDIP": bounds, "boundsLayer": "headless production mainView; not native node bounds", "paintGate": "normal public window painting; native contents require visual review"})
			return png, nil
		}
		baseline, err := win.CapturePage()
		if err != nil {
			finish(err)
			return
		}
		if err = click("Settings"); err != nil {
			finish(err)
			return
		}
		if err = settingsQAWait("real settings describe", func() bool {
			state.mu.Lock()
			store := state.settings
			state.mu.Unlock()
			if store == nil {
				return false
			}
			snap := store.Snapshot()
			return !snap.Loading && len(snap.Namespaces) > 0 && snap.Err == ""
		}); err != nil {
			finish(err)
			return
		}
		last, err := capture("general-wide", baseline)
		if err != nil {
			finish(err)
			return
		}
		for _, page := range []string{"Appearance", "Interface", "Plugins", "Skills", "MCP", "About"} {
			if err = navigate(page); err != nil {
				finish(fmt.Errorf("%s navigation: %w", page, err))
				return
			}
			if page == "Plugins" {
				err = settingsQAWait("real plugin inventory", func() bool { s := state.settingsPluginInventory.Snapshot(); return s.Ready && !s.Busy && s.Err == "" })
			} else if page == "MCP" {
				err = settingsQAWait("real MCP empty catalog", func() bool {
					s := state.mcpController().Snapshot()
					return s.Ready && !s.Busy && s.Err == "" && len(s.Rows) == 0
				})
			}
			if err != nil {
				finish(err)
				return
			}
			last, err = capture(page+"-wide", last)
			if err != nil {
				finish(err)
				return
			}
		}
		// Close through the actual production modal callback; keyboard/focus
		// assertions belong to the Tester layer, not an OS input claim.
		if err = action(func() error { tt.Key(0, ui.KeyEscape); return nil }); err != nil {
			finish(err)
			return
		}
		state.mu.Lock()
		closed := !state.settingsOpen
		state.mu.Unlock()
		focused := false
		mygo.RunOnMain(func() { focused = tt.Focused("Settings") })
		if !closed || !focused {
			finish(fmt.Errorf("production mainView dismissal: closed=%v openerFocus=%v", closed, focused))
			return
		}
		last, err = capture("workspace-return", last)
		if err != nil {
			finish(err)
			return
		}
		if err = click("Settings"); err != nil {
			finish(err)
			return
		}
		// The SDK minimum may clamp the requested narrow size: record actual DIP.
		win.SetContentSize(620, 520)
		w, h := win.ContentSize()
		mygo.RunOnMain(func() { tt.SetSize(w, h) })
		win.Invalidate()
		last, err = capture("general-narrow", last)
		if err != nil {
			finish(err)
			return
		}
		for _, page := range []string{"Appearance", "Interface", "MCP"} {
			if err = navigate(page); err != nil {
				finish(err)
				return
			}
			last, err = capture(page+"-narrow", last)
			if err != nil {
				finish(err)
				return
			}
		}
		// Create a genuine read error only by stopping this run's owned Host.
		state.mu.Lock()
		sup := state.sup
		state.mu.Unlock()
		if sup != nil {
			sup.Stop()
		}
		if err = settingsQAWait("owned Host fully detached before error read", func() bool {
			state.dshMu.Lock()
			detached := state.dshClient == nil
			state.dshMu.Unlock()
			state.mu.Lock()
			muxReady := state.dshMuxReady
			state.mu.Unlock()
			return detached && !muxReady
		}); err != nil {
			finish(err)
			return
		}
		state.refreshMCPController()
		if err = settingsQAWait("actual offline MCP read error", func() bool { s := state.mcpController().Snapshot(); return !s.Busy && s.Err != "" }); err != nil {
			finish(err)
			return
		}
		last, err = capture("MCP-offline-error", last)
		if err != nil {
			finish(err)
			return
		}
		_ = last
		state.mu.Lock()
		setStore := state.settings
		sessions := state.sessions
		state.mu.Unlock()
		skillCount := len(sessions.Snapshot().SkillCatalog.Skills)
		proof := map[string]any{"layer": "real production native mainView and Host; production callbacks driven by public ui.Tester", "captures": proofs, "describedNamespacesAfterOwnedHostStop": len(setStore.Snapshot().Namespaces), "skillRows": skillCount, "focusReturnLayer": "production mainView ui.Tester only", "openerFocusRestored": focused, "nativeKeyboardFocusVerified": false, "modelRequests": 0, "settingsMutations": 0, "promptCalls": 0, "externalLinksOpened": false, "credentialDocumentAbsent": true, "originalPixelParityClaim": false, "nativeScreenshotsRequireIndependentVisualReview": true, "notVerified": []string{"Models/model discovery", "Account/sign-in", "Credentials and secret reveal", "remote authorization", "public market network/install", "update checking/install", "OS keyboard focus/IME", "scrolling all subpage content through native input"}}
		raw, _ := json.MarshalIndent(proof, "", "  ")
		finish(os.WriteFile(filepath.Join(out, "settings-native.json"), append(raw, '\n'), 0600))
	}()
	main()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("desktop exited before settings proof")
	}
}

// Resume only the unresolved narrow cases. Captures are diagnostic observations,
// not assertions that a pixel color or the Tester proves native input/focus.
func settingsQANarrowDiagnostic(win *mygo.Window, tt *ui.Tester, out string, action func(func() error) error, navigate func(string) error) error {
	if err := action(func() error { return tt.Click("Settings") }); err != nil {
		return err
	}
	if err := settingsQAWait("settings describe for narrow resume", func() bool {
		state.mu.Lock()
		s := state.settings
		state.mu.Unlock()
		return s != nil && !s.Snapshot().Loading && len(s.Snapshot().Namespaces) > 0
	}); err != nil {
		return err
	}
	win.SetContentSize(620, 520)
	w, h := win.ContentSize()
	mygo.RunOnMain(func() { tt.SetSize(w, h) })
	observations := []map[string]any{}
	observe := func(page string) error {
		win.Invalidate()
		for _, delay := range []time.Duration{0, 300 * time.Millisecond, 700 * time.Millisecond} {
			if delay > 0 {
				time.Sleep(delay)
			}
			png, err := win.CapturePage()
			if err != nil {
				return err
			}
			name := fmt.Sprintf("%s-native-%d.png", page, delay.Milliseconds())
			if err = os.WriteFile(filepath.Join(out, name), png, 0600); err != nil {
				return err
			}
		}
		var reference bytes.Buffer
		var bounds ui.Rect
		var focused bool
		mygo.RunOnMain(func() {
			tt.Frame()
			bounds, _ = tt.Find(page)
			focused = tt.Focused(page)
			_ = pngcodec.Encode(&reference, tt.Image())
		})
		if err := os.WriteFile(filepath.Join(out, page+"-reference.png"), reference.Bytes(), 0600); err != nil {
			return err
		}
		observations = append(observations, map[string]any{"page": page, "actualContentSizeDIP": []int{w, h}, "testerTargetNodeBoundsDIP": bounds, "testerTargetFocused": focused, "nativeKeyboardFocusVerified": false, "nativeCapturesAreObservationsRequiringVisualReview": true, "timing": "0ms,300ms,1000ms cumulative after Invalidate"})
		return nil
	}
	for _, page := range []string{"Appearance", "Interface", "MCP"} {
		if err := navigate(page); err != nil {
			return err
		}
		if page == "MCP" {
			if err := settingsQAWait("real empty MCP read", func() bool {
				s := state.mcpController().Snapshot()
				return s.Ready && !s.Busy && s.Err == "" && len(s.Rows) == 0
			}); err != nil {
				return err
			}
		}
		if err := observe(page); err != nil {
			return err
		}
	}
	state.mu.Lock()
	sup := state.sup
	state.mu.Unlock()
	if sup != nil {
		sup.Stop()
	}
	if err := settingsQAWait("owned Host detached", func() bool {
		state.dshMu.Lock()
		detached := state.dshClient == nil
		state.dshMu.Unlock()
		state.mu.Lock()
		ready := state.dshMuxReady
		state.mu.Unlock()
		return detached && !ready
	}); err != nil {
		return err
	}
	state.refreshMCPController()
	if err := settingsQAWait("current controller offline error", func() bool { s := state.mcpController().Snapshot(); return !s.Busy && s.Err != "" }); err != nil {
		return err
	}
	if err := observe("MCP-offline"); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(map[string]any{"mode": "narrow-only resumed owned profile", "observations": observations, "controllerRoutesVerified": true, "nativeScreenshotsRequireIndependentVisualReview": true, "nativeKeyboardFocusVerified": false, "settingsMutations": 0, "promptCalls": 0, "modelDiscoveryActionInvoked": false, "publicNativeAPIOnly": true, "offlineControllerError": state.mcpController().Snapshot().Err}, "", "  ")
	return os.WriteFile(filepath.Join(out, "settings-narrow-diagnostic.json"), append(raw, '\n'), 0600)
}

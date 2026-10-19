//go:build windows && appearanceacceptance && !nativeacceptance

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

var appearanceAcceptanceError error

func TestMain(m *testing.M) {
	flag.Parse()
	if out := os.Getenv("WHALEISLE_APPEARANCE_ACCEPTANCE_OUTPUT"); out != "" {
		appearanceAcceptanceError = runNativeAppearanceAcceptance(out)
	}
	os.Exit(m.Run())
}

func TestNativeAppearanceIsolatedAcceptance(t *testing.T) {
	if os.Getenv("WHALEISLE_APPEARANCE_ACCEPTANCE_OUTPUT") == "" {
		t.Fatal("explicit isolated acceptance output required")
	}
	if appearanceAcceptanceError != nil {
		t.Fatal(appearanceAcceptanceError)
	}
}

func runNativeAppearanceAcceptance(output string) (failure error) {
	stage := os.Getenv("WHALEISLE_APPEARANCE_ACCEPTANCE_STAGE")
	if !filepath.IsAbs(output) || !filepath.IsAbs(stage) {
		return errors.New("absolute output and verified runtime stage required")
	}
	root, err := os.MkdirTemp(filepath.Dir(filepath.Dir(output)), "appearance-native-fresh-")
	if err != nil {
		return err
	}
	record := map[string]any{"schemaVersion": 1, "ownedRoot": root, "runtimeStage": stage, "matrix": []string{"SET-05", "SET-06"},
		"scope":         "real isolated config/controller persistence, real Host UI-only settings and restart, public MyGo native page/state rendering; no native pointer/keyboard injection",
		"modelRequests": 0, "credentialsRead": false, "userSettingsChanged": false, "rawOSHelper": false,
		"gpuFluidImplemented": false, "originalWindowPixelComparison": false, "nativeReducedMotionToggleExercised": false}
	record["networkPacketCapture"] = false
	checks := []string{}
	defer func() {
		record["checks"], record["pass"] = checks, failure == nil
		if failure != nil {
			record["error"] = failure.Error()
		}
		raw, e := json.MarshalIndent(record, "", "  ")
		if e == nil {
			e = os.WriteFile(output, append(raw, '\n'), 0600)
		}
		if e != nil {
			failure = errors.Join(failure, e)
		}
	}()
	ownedConfig := filepath.Join(root, "config")
	if err = os.Mkdir(ownedConfig, 0700); err != nil {
		return err
	}
	cfg, err := config.Open(ownedConfig)
	if err != nil {
		return err
	}
	if err = cfg.Update(func(c *config.Config) { c.Theme = theme.IDDeepseekLight }); err != nil {
		return err
	}
	s := &mainState{cfgStore: cfg, themeID: theme.IDDeepseekLight}
	editor := s.themeFontEditor(true)
	if editor == nil || editor.Draft.Interface.Family != theme.DefaultTextFamily || editor.Draft.Code.Family != theme.DefaultCodeFamily {
		return errors.New("fresh original font defaults mismatch")
	}
	checks = append(checks, "fresh config uses original text and code fallback stacks")
	previous := theme.CurrentPreferences()
	defer theme.SetPreferences(previous)
	theme.SetPreferences(editor.Saved)
	var store *settingsstore.Store
	var stop func()
	var e error
	if prior := os.Getenv("WHALEISLE_APPEARANCE_HOST_EVIDENCE"); prior != "" {
		var saved map[string]any
		raw, readErr := os.ReadFile(prior)
		if readErr != nil {
			return readErr
		}
		if json.Unmarshal(raw, &saved) != nil || saved["hostEffectPreferencePersistence"] != true || saved["ownedHostStopped"] != true || saved["ownedListenerClosed"] != true || saved["runtimeStage"] != stage {
			return errors.New("prior owned Host persistence evidence is not complete for this runtime")
		}
		record["hostPersistenceEvidence"] = prior
		record["hostEffectPreferencePersistence"] = true
		record["hostStartedThisRun"] = false
		store = settingsstore.New()
	} else {
		record["hostStartedThisRun"] = true
		store, stop, e = appearanceHostSettings(stage, root, record)
	}
	hostIssue := e
	if hostIssue != nil {
		record["hostEffectError"] = hostIssue.Error()
		store = settingsstore.New()
	}
	if stop != nil {
		defer stop()
	}
	access := &settings.BodySettingsAccess{Owner: 1, Snapshot: store.Snapshot(), OnChange: func(ns, key, value string) {
		store.Select(ns)
		store.SetDraft(key, value)
		store.SaveUpdate(context.Background())
	}}
	mygo.App.SetName("WhaleIsle isolated appearance QA")
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(root, "native"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(root, "cache"))
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(root, "logs"))
	finished := make(chan error, 1)
	page := "appearance"
	var osReduced bool
	var draws int
	mygo.App.OnReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{Title: "Owned appearance acceptance", Width: 1185, Height: 792, Hidden: true, Content: ui.View(func(c *ui.Context) {
			s.beginAppearanceFrame(c)
			defer s.endAppearanceFrame()
			editor = s.themeFontEditor(page == "appearance")
			osReduced = c.Preferences().ReduceMotion
			draws++
			ui.Column(c).Fill().MinHeight(0).Children(func() {
				p := theme.CurrentPreferences()
				ui.Column(c).Height(154).Shrink(0).Padding(8, 16).Gap(4).Children(func() {
					for _, sample := range []struct {
						name, text string
						font       theme.FontPreference
					}{{"Interface", "WhaleIsle 界面 中文 Aa 012345", p.Interface}, {"Chat", "会话正文 hello 世界", p.Chat}, {"Code", "const 中文 = 123; iiii MMMM", p.Code}, {"Composer", "输入法 composition 中文 Aa", p.Composer}, {"Terminal", "terminal 012345 中文", p.Terminal}} {
						ui.Text(c, sample.name+": "+sample.text).Font(sample.font.Family).FontSize(sample.font.Size).SingleLine()
					}
				})
				if page == "effects" {
					ui.Column(c).Fill().Padding(24).Gap(20).Children(func() {
						ui.Text(c, "Production isolated effect previews; no session or prompt callbacks").FontSize(16)
						ui.Column(c.Key("beam-preview-seat")).FillWidth().Children(func() { composer.SettingsEffectPreview(c, appearanceBeamStyle, false) })
						ui.Column(c.Key("typing-preview-seat")).FillWidth().Children(func() { composer.SettingsEffectPreview(c, appearanceTypingStyle, true) })
					})
				} else {
					settings.View(c, settings.Model{Store: store, ThemeID: s.themeID, AppearanceActive: true, Appearance: &settings.AppearanceAccess{Host: access, Preferences: editor, Theme: &s.themeID, OnSave: func() { _ = cfg.Update(func(c *config.Config) { c.Theme = s.themeID }) }}})
				}
			})
		})})
		win.ShowInactive()
		record["nativeWindow"] = "public ShowInactive; no activation or raw OS input"
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
			defer cancel()
			call := func(fn func() error) error {
				done := make(chan error, 1)
				win.Update(func() { done <- fn() })
				select {
				case e := <-done:
					return e
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			screens := []map[string]any{}
			capture := func(name string) (string, error) {
				before := 0
				if e := call(func() error { before = draws; win.Invalidate(); return nil }); e != nil {
					return "", e
				}
				deadline := time.Now().Add(4 * time.Second)
				painted := false
				for time.Now().Before(deadline) {
					if e := call(func() error { painted = draws > before; return nil }); e != nil {
						return "", e
					}
					if painted {
						break
					}
					time.Sleep(15 * time.Millisecond)
				}
				if !painted {
					return "", errors.New("native frame did not advance before capture")
				}
				data, e := win.CapturePage()
				if e != nil {
					return "", e
				}
				img, e := png.Decode(bytes.NewReader(data))
				if e != nil {
					return "", e
				}
				if img.Bounds().Dx() < 1000 || img.Bounds().Dy() < 600 {
					return "", errors.New("native capture dimensions unexpected")
				}
				h := sha256.New()
				sample := image.Rect(0, 0, img.Bounds().Dx(), min(img.Bounds().Dy(), 154*img.Bounds().Dx()/1185))
				for y := sample.Min.Y; y < sample.Max.Y; y++ {
					for x := sample.Min.X; x < sample.Max.X; x++ {
						r, g, b, a := img.At(x, y).RGBA()
						h.Write([]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)})
					}
				}
				path := filepath.Join(filepath.Dir(output), name+".png")
				if e = os.WriteFile(path, data, 0600); e != nil {
					return "", e
				}
				hash := hex.EncodeToString(h.Sum(nil))
				screens = append(screens, map[string]any{"name": name, "path": path, "widthPx": img.Bounds().Dx(), "heightPx": img.Bounds().Dy(), "sampleSHA256": hash})
				record["screenshots"] = screens
				return hash, nil
			}
			run := func() error {
				if os.Getenv("WHALEISLE_APPEARANCE_EFFECTS_ONLY") == "1" {
					if e := call(func() error { page = "effects"; return nil }); e != nil {
						return e
					}
					time.Sleep(500 * time.Millisecond) // source layer-opacity transition is 420ms.
					if _, e := capture("native-appearance-effects-isolated"); e != nil {
						return e
					}
					checks = append(checks, "independently keyed production effect previews rendered after 420ms activation transition")
					return call(func() error {
						record["nativeOSReduceMotion"], record["nativeFrames"], record["screenshots"] = osReduced, draws, screens
						return nil
					})
				}
				baseline, e := capture("native-appearance-default")
				if e != nil {
					return e
				}
				if e = call(func() error {
					editor.Draft.Interface = theme.FontPreference{Family: "Arial, 'Microsoft YaHei'", Size: 18}
					editor.Draft.Chat.Size = 16
					editor.Draft.Composer.Size = 17
					editor.Draft.Code.Size = 15
					editor.Draft.Light = "violet"
					editor.Draft.Dark = "midnight"
					editor.ApplyPreview()
					if !editor.Preview {
						return errors.New(editor.Status)
					}
					return nil
				}); e != nil {
					return e
				}
				preview, e := capture("native-appearance-preview")
				if e != nil {
					return e
				}
				if preview == baseline {
					return errors.New("font/theme preview did not change native samples")
				}
				if e = call(func() error { editor.Cancel(); return nil }); e != nil {
					return e
				}
				canceled, e := capture("native-appearance-canceled")
				if e != nil {
					return e
				}
				if canceled != baseline {
					return errors.New("cancel did not restore original native samples")
				}
				checks = append(checks, "real native default/explicit-font/theme preview changes and cancel restores identical typography samples")
				if e = call(func() error {
					editor.Draft.Interface = theme.FontPreference{Family: "Arial, 'Microsoft YaHei'", Size: 18}
					editor.Draft.Code.Size = 15
					editor.Draft.Dark = "midnight"
					editor.Save()
					s.themeID = theme.IDDeepseekDark
					return cfg.Update(func(c *config.Config) { c.Theme = s.themeID })
				}); e != nil {
					return e
				}
				saved, e := capture("native-appearance-saved-dark")
				if e != nil {
					return e
				}
				if e = call(func() error {
					reopened, e := config.Open(ownedConfig)
					if e != nil {
						return e
					}
					p, e := reopened.Appearance()
					if e != nil {
						return e
					}
					if !reflect.DeepEqual(p, editor.Saved) || p.Interface.Family != "Arial, 'Microsoft YaHei'" || p.Interface.Size != 18 || p.Code.Size != 15 || p.Dark != "midnight" || reopened.Get().Theme != theme.IDDeepseekDark {
						return errors.New("theme/font config restart mismatch")
					}
					s.cfgStore = reopened
					s.appearanceEditor = nil
					editor = s.themeFontEditor(true)
					theme.SetPreferences(editor.Saved)
					s.themeID = reopened.Get().Theme
					return nil
				}); e != nil {
					return e
				}
				restored, e := capture("native-appearance-reopened-dark")
				if e != nil {
					return e
				}
				if restored != saved {
					return errors.New("saved/reopened native typography differs")
				}
				checks = append(checks, "real config SaveAppearance and theme mode survive reopen with explicit family/size and identical native samples")
				if e = call(func() error { page = "effects"; return nil }); e != nil {
					return e
				}
				if _, e = capture("native-appearance-effects"); e != nil {
					return e
				}
				checks = append(checks, "production SettingsEffectPreview beam/typing native paint; no typing input or GPU-fluid equivalence claimed")
				return call(func() error {
					record["nativeOSReduceMotion"], record["nativeFrames"], record["screenshots"] = osReduced, draws, screens
					return nil
				})
			}
			e := run()
			win.Update(func() { win.Destroy(); finished <- e; mygo.App.Quit() })
		}()
	})
	mygo.App.Run()
	return errors.Join(hostIssue, <-finished)
}

// Forced desktop preferences are confined to the public headless UI tester.
// This covers the real composer branch, not an OS toggle or GPU-fluid claim.
func TestAppearanceReducedMotionPublicPath(t *testing.T) {
	build := func(beam bool) (*ui.Tester, *string) {
		draft := ""
		client := composer.NewDraftClient(&draft)
		m := composer.Model{Draft: &draft, TextInput: client, UseTextInputClient: true, TypingFx: true, ComposerBeam: beam, ComposerBeamStyle: appearanceBeamStyle, Snapshot: session.ViewState{PromptBusy: true}}
		tt := ui.NewTester(func(c *ui.Context) { composer.View(c, m) }, 560, 280)
		tt.SetPreferences(ui.Preferences{ReduceMotion: true})
		return tt, &draft
	}
	enabled, draft := build(true)
	disabled, _ := build(false)
	if !bytes.Equal(enabled.Image().Pix, disabled.Image().Pix) {
		t.Fatal("reduced motion did not suppress enabled beam in production painter")
	}
	if err := enabled.Click("Composer draft"); err != nil {
		t.Fatal(err)
	}
	enabled.Type("owned 中文")
	if *draft != "owned 中文" {
		t.Fatal("reduced motion altered native draft input")
	}
	enabled.Key(ui.Cmd, ui.KeyZ)
	if *draft != "" {
		t.Fatal("reduced motion altered undo ownership")
	}
}

var appearanceBeamStyle = json.RawMessage(`{"direction":"clockwise","mode":"aurora","period":10,"palette":{"kind":"preset","id":"aurora"},"intensity":100,"bloom":100,"hue":0,"trackWidth":2,"glowBlur":8,"easing":"cubic-bezier","breathing":{"enabled":true,"amplitude":0.18,"period":8},"hueCycle":{"enabled":true,"range":90,"period":16},"night":{"enabled":false,"fromMinute":1320,"toMinute":360,"dim":40}}`)
var appearanceTypingStyle = json.RawMessage(`{"effect":"rise","cursor":"block","cursorBlink":false,"speed":120,"colors":{"kind":"preset","id":"violet"}}`)

// The only Host RPCs permitted by this fresh-profile check are settings reads
// and revision-owned UI preference writes; no provider/session APIs are called.
type appearanceRPC struct{ calls map[string]int }

func (a *appearanceRPC) RoundTrip(r *http.Request) (*http.Response, error) {
	if !strings.Contains(r.URL.Path, "/settings/") {
		return nil, errors.New("non-settings RPC excluded from appearance acceptance")
	}
	a.calls[r.URL.Path]++
	return http.DefaultTransport.RoundTrip(r)
}

func appearanceHostSettings(stage, root string, record map[string]any) (*settingsstore.Store, func(), error) {
	userData := filepath.Join(root, "user-data")
	home := filepath.Join(userData, "dsh-home")
	workspace := filepath.Join(root, "workspace")
	if e := os.MkdirAll(workspace, 0700); e != nil {
		return nil, nil, e
	}
	os.Setenv(paths.EnvUserData, userData)
	os.Setenv("DSH_HOME", home)
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		return nil, nil, e
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	node, vendor := filepath.Join(stage, "node", "node.exe"), filepath.Join(stage, "vendor")
	harnessRoot := filepath.Join(vendor, "deepseek-harness")
	sup := harness.New(harness.Options{Install: harness.NewInstallServer(harness.InstallServerOptions{NodeBin: node}), Launch: harness.LaunchOptions{NodeBin: node, HarnessRoot: harnessRoot, VendorDir: vendor, Host: "127.0.0.1", Port: port, Home: home}, Prepare: harness.PrepareOptions{VendorDir: vendor, HarnessRoot: harnessRoot, DshHome: home, ProfileDir: filepath.Join(home, "profiles", "web"), RemoteSet: true, Remote: false}, Workspace: workspace, ReadyTimeout: 100 * time.Second})
	var endpoint harness.Endpoint
	stop := func() {
		sup.Stop()
		record["ownedHostStopped"] = sup.Status().State == harness.Stopped
		if endpoint.Origin != "" {
			u, _ := url.Parse(endpoint.Origin)
			c, e := net.DialTimeout("tcp", u.Host, time.Second)
			record["ownedListenerClosed"] = e != nil
			if c != nil {
				c.Close()
			}
		}
	}
	store := settingsstore.New()
	audit := &appearanceRPC{calls: map[string]int{}}
	start := func() error {
		sup.Start()
		deadline := time.Now().Add(110 * time.Second)
		for time.Now().Before(deadline) {
			st := sup.Status()
			if st.State == harness.Failed {
				return st.Err
			}
			if st.State == harness.Ready {
				endpoint = st.Endpoint
				store.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}, HTTP: &http.Client{Transport: audit}}, nil)
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
		return errors.New("appearance owned Host ready timeout")
	}
	wait := func() error {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			v := store.Snapshot()
			if !v.Loading && !v.Saving {
				if v.Err != "" {
					return errors.New(v.Err)
				}
				return nil
			}
			time.Sleep(15 * time.Millisecond)
		}
		return errors.New("appearance settings acknowledgement timeout")
	}
	read := func() (map[string]json.RawMessage, error) {
		store.RefreshDescribe(context.Background())
		if e := wait(); e != nil {
			return nil, e
		}
		for _, ns := range store.Snapshot().Namespaces {
			if ns.NS == "ui-conversation" {
				keys := []string{}
				for _, field := range ns.Fields {
					keys = append(keys, field.Key)
				}
				record["servedConversationFieldKeys"] = keys
				var value map[string]json.RawMessage
				if e := json.Unmarshal(ns.Value, &value); e != nil {
					return nil, e
				}
				return value, nil
			}
		}
		return nil, errors.New("served ui-conversation namespace absent")
	}
	if e = start(); e != nil {
		stop()
		return nil, nil, e
	}
	value, e := read()
	if e != nil {
		stop()
		return nil, nil, e
	}
	patch := map[string]string{"composerBeam": "true", "typingFx": "true", "composerBeamStyle": string(appearanceBeamStyle), "typingFxStyle": string(appearanceTypingStyle)}
	store.Select("ui-conversation")
	for k, v := range patch {
		found := false
		for _, ns := range store.Snapshot().Namespaces {
			if ns.NS == "ui-conversation" {
				for _, field := range ns.Fields {
					if field.Key == k {
						found = true
					}
				}
			}
		}
		if !found {
			stop()
			return nil, nil, fmt.Errorf("served effect field absent: %s", k)
		}
		store.SetDraft(k, v)
	}
	store.SaveUpdate(context.Background())
	if e = wait(); e != nil {
		stop()
		return nil, nil, e
	}
	value, e = read()
	if e != nil {
		stop()
		return nil, nil, e
	}
	for k, v := range patch {
		var want, got any
		json.Unmarshal([]byte(v), &want)
		json.Unmarshal(value[k], &got)
		if !reflect.DeepEqual(want, got) {
			stop()
			return nil, nil, fmt.Errorf("effect preference acknowledgement differs: %s", k)
		}
	}
	store.Select("ui-conversation")
	store.SetDraft("composerBeam", "false")
	store.SetDraft("typingFx", "false")
	store.SaveUpdate(context.Background())
	if e = wait(); e != nil {
		stop()
		return nil, nil, e
	}
	store.Detach()
	stop()
	if e = start(); e != nil {
		stop()
		return nil, nil, e
	}
	value, e = read()
	if e != nil {
		stop()
		return nil, nil, e
	}
	for _, k := range []string{"composerBeam", "typingFx"} {
		if string(value[k]) != "false" {
			stop()
			return nil, nil, fmt.Errorf("effect disabled state not persisted: %s", k)
		}
	}
	for k, v := range map[string]json.RawMessage{"composerBeamStyle": appearanceBeamStyle, "typingFxStyle": appearanceTypingStyle} {
		var want, got any
		json.Unmarshal(v, &want)
		json.Unmarshal(value[k], &got)
		if !reflect.DeepEqual(want, got) {
			stop()
			return nil, nil, fmt.Errorf("effect style not persisted: %s", k)
		}
	}
	record["hostEffectPreferencePersistence"] = true
	record["hostStarts"] = 2
	record["observedRPCs"] = audit.calls
	return store, func() { store.Detach(); stop() }, nil
}

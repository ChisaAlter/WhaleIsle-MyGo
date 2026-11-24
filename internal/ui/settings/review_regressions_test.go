package settings

import (
	"context"
	"encoding/json"
	"errors"
	"image/color"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/shortcuts"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSettingsActionsLeaksRawError(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(locale) })

	const rawErr = "rpc error: dial tcp 127.0.0.1:43210: connect: connection refused"
	schema := json.RawMessage(`{"type":"object","dict":{"greeting":{"type":"string","description":"Greeting"}}}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": body["rpcId"],
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"writable": true,
						"namespaces": []map[string]any{{
							"ns":       "custom-ns",
							"revision": 1.0,
							"schema":   schema,
							"value":    map[string]any{"greeting": "hello"},
						}},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/update"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":  "server-response",
				"rpcId": body["rpcId"],
				"result": map[string]any{
					"ok":    false,
					"error": map[string]any{"code": "E_INTERNAL", "message": rawErr},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type":   "server-response",
				"rpcId":  body["rpcId"],
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		}
	}))
	defer srv.Close()

	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	s.RefreshDescribe(context.Background())
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if snap := s.Snapshot().Value; snap.Described && !snap.Loading {
			break
		}
		time.Sleep(time.Millisecond)
	}
	s.Select("custom-ns")
	s.SetDraft("greeting", "updated")
	s.SaveUpdate(context.Background())
	for time.Now().Before(deadline) {
		if snap := s.Snapshot().Value; !snap.Saving && snap.Err != "" {
			break
		}
		time.Sleep(time.Millisecond)
	}

	drafts := map[string]*string{}
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			Store: s,
			DraftPtr: func(key, seed string) *string {
				if p, ok := drafts[key]; ok {
					return p
				}
				v := seed
				drafts[key] = &v
				return &v
			},
			OnDraft: func(key, val string) { s.SetDraft(key, val) },
		})
	}, 1000, 700)

	for _, txt := range tt.Texts() {
		if strings.Contains(txt, "dial tcp 127.0.0.1:43210") {
			t.Fatalf("settingsActions painted raw transport error directly on screen: %q", txt)
		}
	}
}

func TestSelectHidesUnknownCurrentValue(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })

	host := &BodySettingsAccess{
		Owner: 1,
		Snapshot: store.ViewState{
			ClientReady: true,
			Writable:    true,
			Described:   true,
			Namespaces: []store.Namespace{{
				NS: "ui-chat",
				Fields: []store.Field{{
					Key:         "mode",
					Kind:        store.FieldEnum,
					EnumOptions: []string{"standard", "compact"},
				}},
				Value: json.RawMessage(`{"mode":"legacy-custom-mode"}`),
			}},
		},
		OnChange: func(ns, key, value string) {},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			bodyHostPreference(c, host, "ui-chat", "mode", "Transcript mode", "", c.Theme().Text, c.Theme().TextMuted)
		})
	}, 600, 200)
	if !tt.HasText("legacy-custom-mode") {
		t.Errorf("bodyHostPreference settingsPreferenceSelect rendered blank instead of showing %q: texts=%v", "legacy-custom-mode", tt.Texts())
	}

	draftVal := "custom-preset-v1"
	snap := store.ViewState{Writable: true, SelectedNS: "custom", Drafts: map[string]string{"preset": draftVal}}
	field := store.Field{Key: "preset", Label: "Preset", Kind: store.FieldEnum, EnumOptions: []string{"fast", "balanced"}}
	tt2 := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(16).Children(func() {
			fieldRow(c, Model{DraftPtr: func(key, seed string) *string { return &draftVal }, OnDraft: func(key, value string) {}}, snap, field, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 600, 200)
	if !tt2.HasText(draftVal) {
		t.Errorf("fieldRow kSelect rendered blank instead of showing %q: texts=%v", draftVal, tt2.Texts())
	}
}

func TestPreferenceSwitchDisabledFocusRingAndOpacity(t *testing.T) {
	value := true
	disabled := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Background(ui.RGB(255, 255, 255)).Padding(24).Children(func() {
			settingsPreferenceSwitch(c, &value, settingsSwitchLabelled).Label("Probe switch").Disabled(disabled)
		})
	}, 200, 120)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	tt.Key(0, ui.KeyTab)
	tt.Frame()
	bounds, ok := tt.Find("Probe switch")
	if !ok {
		t.Fatal("switch not found")
	}
	ringPx := func() color.RGBA {
		return tt.Image().RGBAAt(int(bounds.X)-2, int(bounds.Y+bounds.H/2))
	}
	focusedPx := ringPx()

	disabled = true
	tt.Frame()
	disabledPx := ringPx()
	if disabledPx != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("disabled switch painted focus ring pixel %v (focused=%v, want background white {255 255 255 255})", disabledPx, focusedPx)
	}
}

func TestNsListMemoStaleOnDesignTokenChange(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(locale) })

	accent := ui.RGB(220, 40, 40)
	m := fullRailModel()
	m.ActiveSection = SectionGeneral
	m.SectionContent = func(c *ui.Context) { ui.Text(c, "body") }

	filter := func(s design.Scheme, p *design.Palette) {
		p.SidebarAccent = ui.RGB(20, 200, 60)
	}
	t.Cleanup(func() { design.SetGlass(false, nil) })

	tt := ui.NewTester(func(c *ui.Context) {
		design.Bind(c, design.Light, accent, ui.RGB(255, 255, 255))
		View(c, m)
	}, 900, 600)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	var sel ui.Rect
	for _, r := range tt.FindAll("通用设置") {
		if r.X < 208 && r.W*r.H > sel.W*sel.H {
			sel = r
		}
	}
	before := tt.Image().RGBAAt(int(sel.X+sel.W)-6, int(sel.Y+sel.H/2))

	design.SetGlass(true, filter)
	tt.Frame()

	after := tt.Image().RGBAAt(int(sel.X+sel.W)-6, int(sel.Y+sel.H/2))
	if after.R != 20 || after.G != 200 || after.B != 60 {
		t.Fatalf("nsList Memo kept stale rail fill before=%v after=%v, want {20 200 60 255}", before, after)
	}
}

func TestNsListMemoStaleOnFontSizeChange(t *testing.T) {
	prev := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(prev) })
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(locale) })

	p := theme.Defaults()
	p.Interface.Size = 14
	theme.SetPreferences(p)

	m := fullRailModel()
	m.ActiveSection = SectionGeneral
	m.SectionContent = func(c *ui.Context) { ui.Text(c, "body") }

	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 900, 600)
	findTextRect := func(label string) ui.Rect {
		var best ui.Rect
		for _, r := range tt.FindAll(label) {
			if r.X < 208 && (best.H == 0 || r.H < best.H) {
				best = r
			}
		}
		return best
	}
	before := findTextRect("外观")

	p.Interface.Size = 20
	theme.SetPreferences(p)
	tt.Frame()

	after := findTextRect("外观")
	if after.H == before.H {
		t.Fatalf("nsList Memo kept stale rail text height before=%+v after=%+v after Interface.Size 14->20", before, after)
	}
}

func TestHarnessRestartPickersOverflowModal(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(locale) })

	autoRestart := true
	maxAttempts := "3"
	baseDelay := "1000"
	policy := config.RestartPreferences{AutoRestart: true, MaxAttempts: 3, BaseDelayMs: 1000}
	d := &DesktopAccess{
		Fields: DesktopFields{
			HarnessAutoRestart:        &autoRestart,
			HarnessRestartMaxAttempts: &maxAttempts,
			HarnessRestartBaseDelayMs: &baseDelay,
		},
		RestartPolicy:          &policy,
		OnSetRestartPreference: func(k, v string) (config.RestartPreferences, error) { return policy, nil },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ModalView(c, Model{DesktopActive: true, Desktop: d, OnClose: func() {}})
	}, 600, 900)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	title, _ := tt.Find("Harness 自动恢复")
	attempts := buttonRect(tt.FindAll("最大重启次数"))
	delay := buttonRect(tt.FindAll("基础重启延迟"))
	if attempts.W > 100 || delay.W > 100 || title.W < 40 {
		t.Fatalf("restartPicker .Width(88) overridden by coss.Select MinWidth(144): title=%+v attempts=%+v delay=%+v", title, attempts, delay)
	}
}

func TestSkillsSourceFilterLayoutWrap(t *testing.T) {
	locale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(locale) })

	q, filter := "", ""
	tt := ui.NewTester(func(c *ui.Context) {
		ModalView(c, Model{
			Store:        store.New(),
			SkillsActive: true,
			OnClose:      func() {},
			Skills:       &SkillsAccess{Query: &q, SourceFilter: &filter},
		})
	}, 1200, 800)
	tt.SetTime(time.Unix(1000, 0))
	tt.SetTime(time.Unix(1001, 0))

	filterRect, ok := tt.Find("Skill source filter")
	if !ok || filterRect.W > 300 {
		t.Fatalf("Skill source filter stretched to full row width: found=%v rect=%+v", ok, filterRect)
	}
}

func TestShortcutResetFailureLineOverlapsCancel(t *testing.T) {
	for _, loc := range []string{i18n.LocaleEN, i18n.LocaleZH} {
		t.Run(loc, func(t *testing.T) {
			prev := i18n.Locale()
			i18n.SetLocale(loc)
			t.Cleanup(func() { i18n.SetLocale(prev) })

			e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error {
				return errors.New("write C:/Users/example/shortcuts.json: access is denied; please check file permissions and retry")
			})
			e.RequestReset()
			e.ConfirmReset()

			tt := ui.NewTester(func(c *ui.Context) {
				p := shortcutPalette{fg: c.Theme().Text, muted: c.Theme().TextMuted, caption: c.Theme().TextMuted, border: c.Theme().Border}
				shortcutResetConfirmation(c, e, p)
			}, 900, 800)

			detailLabel := providerCopy("详情", "Details")
			sentence := providerCopy("没能恢复默认快捷键，请重试。", "The default shortcuts could not be restored. Try again.")
			if err := tt.Click(detailLabel); err != nil {
				t.Fatalf("could not click %q: %v", detailLabel, err)
			}
			tt.Frame()
			failAfter := buttonRect(tt.FindAll(sentence))
			cancelAfter, _ := tt.Find("cancel-reset-shortcuts")
			if failAfter.Y+failAfter.H > cancelAfter.Y {
				t.Fatalf("[%s] expanded settingsFailureLine (%+v, bottom=%.1f) overlaps Cancel (%+v, top=%.1f)", loc, failAfter, failAfter.Y+failAfter.H, cancelAfter, cancelAfter.Y)
			}
		})
	}
}

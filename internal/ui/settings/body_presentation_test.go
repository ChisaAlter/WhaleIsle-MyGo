package settings

import (
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/credentials"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/desktop/wallpapers"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func bodyFixtureHost() *BodySettingsAccess {
	ns := func(name string, values map[string]any, fields ...store.Field) store.Namespace {
		raw, _ := json.Marshal(values)
		return store.Namespace{NS: name, Fields: fields, Value: raw}
	}
	boolField := func(key string) store.Field {
		return store.Field{Key: key, Kind: store.FieldBool, Description: "由当前 Host 提供的功能偏好。"}
	}
	enum := func(key string, options ...string) store.Field {
		return store.Field{Key: key, Kind: store.FieldEnum, EnumOptions: options}
	}
	h := &BodySettingsAccess{Owner: 1, Snapshot: store.ViewState{ClientReady: true, Writable: true}, OnChange: func(string, string, string) {}, OnPatch: func(string, map[string]string) {}}
	h.Snapshot.Namespaces = []store.Namespace{
		ns("permission", map[string]any{"defaultPreset": "standard"}, enum("defaultPreset", "standard", "danger-full-access")),
		ns("ui-chat", map[string]any{"transcriptView": "detailed", "collapseTiming": "completion", "linkOpening": "sidebar", "performanceUsage": "compact"}, enum("transcriptView", "compact", "standard", "detailed", "verbose"), enum("collapseTiming", "completion", "next-input"), enum("linkOpening", "sidebar", "new-tab"), enum("performanceUsage", "compact", "detailed")),
		ns("ui-settings", map[string]any{"enabled": false}, boolField("enabled")),
		ns("ui-git", map[string]any{"titlebarGit": true}, boolField("titlebarGit")),
		ns("ui-titlebar", map[string]any{"terminalToggle": true, "surfacesToggle": true}, boolField("terminalToggle"), boolField("surfacesToggle")),
		ns("ui-conversation", map[string]any{"busyEnter": "queue", "customInstructions": "Use concise explanations.\nPreserve the user's working directory.", "composerBeam": true, "composerBeamStyle": effectDefaults(false), "composerBeamPresets": map[string]any{}, "typingFx": true, "typingFxStyle": effectDefaults(true), "typingFxPresets": map[string]any{}, "composerResize": true, "statsLine": true, "sessionCost": false, "officialPeakValley": true, "viewTabs": true, "nextPrompt": true}, enum("busyEnter", "queue", "steer"), store.Field{Key: "customInstructions", Kind: store.FieldString}, boolField("composerBeam"), store.Field{Key: "composerBeamStyle", Kind: store.FieldJSON}, store.Field{Key: "composerBeamPresets", Kind: store.FieldJSON}, boolField("typingFx"), store.Field{Key: "typingFxStyle", Kind: store.FieldJSON}, store.Field{Key: "typingFxPresets", Kind: store.FieldJSON}, boolField("composerResize"), boolField("statsLine"), boolField("sessionCost"), boolField("officialPeakValley"), boolField("viewTabs"), boolField("nextPrompt")),
		ns("visual-replies", map[string]any{"enabled": true}, boolField("enabled")),
		ns("session-log-deepseek", map[string]any{"enabled": false}, boolField("enabled")),
	}
	h.Snapshot.PermissionDefaults = store.PermissionDefaults{Loaded: true, DefaultPreset: "standard", Options: []store.PermissionDefaultOption{{Value: "standard", Name: "Standard"}, {Value: "danger-full-access", Name: "Full access"}}}
	return h
}

func setBodyFixtureValue(h *BodySettingsAccess, ns, key string, value any) {
	for index := range h.Snapshot.Namespaces {
		n := &h.Snapshot.Namespaces[index]
		if n.NS == ns {
			values := map[string]any{}
			_ = json.Unmarshal(n.Value, &values)
			values[key] = value
			n.Value, _ = json.Marshal(values)
			return
		}
	}
}

func saveBodySnapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, tt.Image()); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

// These are isolated render fixtures. Every writer and storage status is fake;
// they do not attach a Host, inspect user configuration, or open native windows.
func TestSettingsBodySourcePagesAndStates(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(old)
	for _, page := range []string{"general", "interface", "appearance", "pet", "about"} {
		for _, state := range []string{"wide", "narrow", "loading", "error", "empty", "read-only"} {
			t.Run(page+"/"+state, func(t *testing.T) {
				h := bodyFixtureHost()
				locale, scheme, wallpaperPath := "en", "system", ""
				enabled := true
				prefs := theme.NewEditor(theme.Defaults(), func(theme.Preferences) error { return nil })
				background := NewWallpaperEditor(wallpapers.Defaults(), "", func(wallpapers.Preferences) error { return nil }, nil, nil)
				companion := &pet.Companion{Enabled: true, Behavior: config.DefaultPetBehavior()}
				desktop := &DesktopAccess{Host: h, Preferences: prefs, Version: "0.0.0-fixture", Fields: DesktopFields{Locale: &locale, CloseToTray: &enabled}, OnSave: func() {}}
				inter := &InterfaceAccess{Host: h, ShowArchivedList: &enabled, OnSave: func() {}}
				appearance := &AppearanceAccess{Host: h, Preferences: prefs, Background: background, Theme: &scheme, WallpaperPath: &wallpaperPath, OnSave: func() {}}
				pa := &PetSettingsAccess{OnSave: func() {}}
				about := &AboutAccess{AppName: "WhaleIsle", Version: "0.0.0-fixture", HomePath: strings.Repeat("long-directory/", 12), OnRevealHome: func() error { return nil }, OnCheckUpdates: func() {}, ReadCredentialStorage: func() credentials.StorageStatus { return credentials.StorageStatus{Mode: "dpapi"} }}
				switch state {
				case "loading":
					h.Snapshot.Loading = true
					pa.Busy = true
					about.Loading = true
					about.UpdateBusy = true
				case "error":
					h.Snapshot.Err = "Synthetic settings save failed; the retained draft is available."
					pa.Err = h.Snapshot.Err
					about.Err = h.Snapshot.Err
					background.Status = h.Snapshot.Err
				case "empty":
					h.Snapshot.Namespaces = nil
					desktop.Preferences = nil
					appearance.Preferences = nil
					appearance.Background = nil
					about.AppName = ""
					about.Version = ""
					about.HomePath = ""
					about.OnRevealHome = nil
				case "read-only":
					h.Snapshot.Writable = false
					desktop.OnSave = nil
					inter.OnSave = nil
					appearance.OnSave = nil
					pa.OnSave = nil
					about.OnRevealHome = nil
				}
				width := 620
				if state == "narrow" {
					width = 340
				}
				tt := ui.NewTester(func(c *ui.Context) {
					ui.Scroll(c).Fill().Padding(24).Gap(8).Children(func() {
						switch page {
						case "general":
							desktopPane(c, Model{Desktop: desktop}, c.Theme().Text, c.Theme().TextMuted)
						case "interface":
							interfacePane(c, Model{Interface: inter}, c.Theme().Text, c.Theme().TextMuted)
						case "appearance":
							appearancePane(c, Model{Appearance: appearance}, c.Theme().Text, c.Theme().TextMuted)
						case "pet":
							petPane(c, theme.Resolve(theme.IDDeepseekLight).Light, companion, c.Theme().Text, c.Theme().TextMuted, pa)
						case "about":
							aboutPane(c, Model{About: about}, c.Theme().Text, c.Theme().TextMuted)
						}
					})
				}, width, 1000)
				if len(tt.Texts()) == 0 {
					t.Fatal("body rendered no content")
				}
				if state == "error" && !tt.HasText("Synthetic settings save failed") {
					t.Fatal("body hides save error")
				}
				if page == "general" && tt.HasText(i18n.T(i18n.KeySettingsDesktopWorkspace)) {
					t.Fatal("technical runtime controls escaped the disclosure")
				}
				saveBodySnapshot(t, tt, fmt.Sprintf("settings-body-%s-%s.png", page, state))
			})
		}
	}
}

func TestSettingsPreferenceRowsReflowLongCopy(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			bodyPreferenceRow(c, "long-row", "Long preference title that must wrap without escaping its settings column", strings.Repeat("A helpful sentence with long content. ", 6), c.Theme().Text, c.Theme().TextMuted, func() {
				ui.Button(c, "Fixture control").Label("fixture-control").FillWidth()
			})
		})
	}, 620, 500)
	wide, ok := tt.Find("fixture-control")
	if !ok || wide.X < 350 || wide.X+wide.W > 620 {
		t.Fatalf("wide control geometry %v", wide)
	}
	// Source Setting-Cells keep intrinsic controls flush to the row's trailing
	// edge; a fixed 140/180 DIP control column changes the copy's wrapping.
	if wide.W <= 0 || wide.W >= 140 || math.Abs(float64(wide.X+wide.W-596)) > .1 {
		t.Fatalf("intrinsic selector should be positive and flush-right: %v", wide)
	}
	title, titleOK := tt.Find("Long preference title that must wrap without escaping its settings column")
	if !titleOK || title.H <= 22 || title.X+title.W+48+8 > wide.X+.1 {
		t.Fatalf("long copy did not wrap in its source text column: %+v control=%+v", title, wide)
	}
	tt.SetSize(340, 700)
	narrow, ok := tt.Find("fixture-control")
	if !ok || narrow.X < 24 || narrow.X+narrow.W > 316 || narrow.Y <= wide.Y {
		t.Fatalf("narrow control did not stack %v -> %v", wide, narrow)
	}
}

func TestBodyTextDraftsStayWithHostAndAdoptAcknowledgedValues(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(old)
	h := bodyFixtureHost()
	h.Snapshot.Namespaces = append(h.Snapshot.Namespaces, store.Namespace{NS: "fixture", Fields: []store.Field{{Key: "caption", Kind: store.FieldString}}, Value: json.RawMessage(`{"caption":"Host A"}`)})
	calls := []string{}
	h.OnChange = func(ns, key, value string) { calls = append(calls, value); setBodyFixtureValue(h, ns, key, value) }
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(24).Children(func() {
			bodyHostPreference(c, h, "fixture", "caption", "Caption", "", c.Theme().Text, c.Theme().TextMuted)
		})
	}, 340, 350)
	focusCaption := func() {
		t.Helper()
		r, ok := tt.Find("Caption")
		if !ok {
			t.Fatal("caption label missing")
		}
		// The source row stacks its input below the title at this width.
		tt.ClickAt(r.X+30, r.Y+r.H+24)
	}
	readCaption := func() string {
		focusCaption()
		tt.Key(ui.Cmd, ui.KeyA)
		tt.Key(ui.Cmd, ui.KeyC)
		return tt.Clipboard()
	}
	focusCaption()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Host A unsaved")
	if len(calls) != 0 {
		t.Fatal("text input saved per keystroke")
	}
	h.Owner = 2
	setBodyFixtureValue(h, "fixture", "caption", "Host B")
	tt.Frame()
	if value := readCaption(); value != "Host B" {
		t.Fatalf("Host A draft leaked into Host B: %q", value)
	}
	if err := tt.Click("Save"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatal("Host switch enabled a save of another Host's draft")
	}
	focusCaption()
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("Host B saved")
	if err := tt.Click("Save"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0] != "Host B saved" {
		t.Fatalf("new Host draft not saved independently: %v", calls)
	}
	// The real Host acknowledgement triggers the next rendered snapshot.
	tt.Frame()
	setBodyFixtureValue(h, "fixture", "caption", "Host B refreshed")
	tt.Frame()
	if value := readCaption(); value != "Host B refreshed" {
		t.Fatalf("acknowledged text draft did not adopt refreshed value: %q", value)
	}
}

func TestEffectEditorAtomicSaveRetainsFailureAndChangesHost(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale("en")
	defer i18n.SetLocale(old)
	for _, typing := range []bool{false, true} {
		t.Run(fmt.Sprint(typing), func(t *testing.T) {
			h := bodyFixtureHost()
			key := "composerBeam"
			if typing {
				key = "typingFx"
			}
			patches := []map[string]string{}
			h.OnPatch = func(ns string, values map[string]string) {
				if ns != "ui-conversation" {
					t.Fatal(ns)
				}
				patches = append(patches, values)
				h.Snapshot.Saving = true
			}
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(24).Children(func() { bodyEffectPreference(c, h, typing, c.Theme().Text, c.Theme().TextMuted) })
			}, 900, 950)
			// The row gear replaced the old Configure button; it opens the same dialog.
			gearLabel := "Thinking glow settings"
			if typing {
				gearLabel = "Typing effect settings"
			}
			if err := tt.Click(gearLabel); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText("Color source") || !tt.HasText("Save and apply") {
				t.Fatal("source effect controls missing")
			}
			if err := tt.Click("save-" + key + "-configuration"); err != nil {
				t.Fatal(err)
			}
			if len(patches) != 1 || len(patches[0]) != 2 {
				t.Fatalf("style and presets did not save atomically: %v", patches)
			}
			if !tt.HasText("Save and apply") {
				t.Fatal("dialog closed before Host acknowledgement")
			}
			h.Snapshot.Saving = false
			h.Snapshot.Err = "Synthetic effect save failure"
			tt.Frame()
			if !tt.HasText(h.Snapshot.Err) || !tt.HasText("Save and apply") {
				t.Fatal("failed save discarded the editor draft")
			}
			saveBodySnapshot(t, tt, "settings-effect-"+key+"-failed-save.png")
			h.Snapshot.Err = ""
			if err := tt.Click("save-" + key + "-configuration"); err != nil {
				t.Fatal(err)
			}
			for field, raw := range patches[len(patches)-1] {
				var value any
				_ = json.Unmarshal([]byte(raw), &value)
				setBodyFixtureValue(h, "ui-conversation", field, value)
			}
			h.Snapshot.Saving = false
			tt.Frame()
			if tt.HasText("Save and apply") {
				t.Fatal("successful save did not acknowledge and close")
			}
			if err := tt.Click(gearLabel); err != nil {
				t.Fatal(err)
			}
			h.Owner++
			tt.Frame()
			if tt.HasText("Save and apply") {
				t.Fatal("effect dialog adopted a replacement Host")
			}
		})
	}
}

func TestEffectConfigurationRoundTripAndSourceConstraints(t *testing.T) {
	for _, typing := range []bool{false, true} {
		style := effectDefaults(typing)
		text, err := effectEnvelope(style, map[string]map[string]any{"Personal": copyEffect(style)}, typing)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parseEffectEnvelope(text, typing)
		if err != nil || len(parsed.Presets) != 1 {
			t.Fatalf("source envelope round trip: %+v %v", parsed, err)
		}
		if _, err := parseEffectEnvelope(text, !typing); err == nil {
			t.Fatal("accepted another feature's configuration")
		}
		if _, err := effectEnvelope(style, map[string]map[string]any{"__proto__": style}, typing); err == nil {
			t.Fatal("accepted reserved preset name")
		}
		if _, err := parseEffectEnvelope(strings.Repeat("x", 65537), typing); err == nil {
			t.Fatal("accepted over-size import")
		}
	}
	style := effectDefaults(false)
	style["period"] = float64(0)
	normalized, err := normalizeEffect(style, false)
	if err != nil || normalized["period"] != .8 {
		t.Fatal("source period clamp missing")
	}
	effectSet(style, "palette", map[string]any{"kind": "custom", "colors": []any{"#ffffff", "bad"}})
	if _, err := normalizeEffect(style, false); err == nil {
		t.Fatal("invalid palette accepted")
	}
}

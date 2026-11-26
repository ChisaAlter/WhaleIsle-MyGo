package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/credentials"
	"whaleisle/internal/desktop/mcpconfig"
	"whaleisle/internal/dsh"
	"whaleisle/internal/shortcuts"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

func onboardingTestController(t *testing.T, step string) *store.DesktopOnboarding {
	t.Helper()
	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: galleryTransport(func(r *http.Request) (*http.Response, error) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		value := map[string]any{
			"writable": true,
			"namespaces": []any{
				map[string]any{"ns": store.DesktopOnboardingNamespace, "schema": map[string]any{"type": "object"}, "revision": 1, "value": map[string]any{"version": 1, "step": step}, "base": map[string]any{}},
				map[string]any{"ns": "ui-chat", "schema": map[string]any{"type": "object"}, "revision": 1, "value": map[string]any{}, "base": map[string]any{}},
				map[string]any{"ns": "ui-settings", "schema": map[string]any{"type": "object"}, "revision": 1, "value": map[string]any{}, "base": map[string]any{}},
			},
		}
		rec := httptest.NewRecorder()
		_ = json.NewEncoder(rec).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return rec.Result(), nil
	})}}, nil)
	s.RefreshDescribe(context.Background())
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		if s.Snapshot().Value.Described && !s.Snapshot().Value.Loading {
			break
		}
		time.Sleep(time.Millisecond)
	}
	ob := store.NewDesktopOnboarding(s)
	ob.Sync(store.OnboardingAccount{Enabled: true, Status: "credential-stored", CreditKnown: true, CreditFunded: true})
	return ob
}

func TestOnboardingWebPDecodeNonBlocking(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)

	ob := onboardingTestController(t, "welcome")
	start := time.Now()
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DesktopOnboarding(c, ob, &AccountAccess{TopUpURL: "https://example.com/topup", OnOpenTopUp: func(string) {}}, "", false)
		})
	}, 1000, 800)
	firstFrame := time.Since(start)
	// Synchronous lossless WebP decoding previously spiked up to 2,250 ms;
	// asynchronous decode must keep the first render frame non-blocking.
	if firstFrame > 500*time.Millisecond {
		t.Fatalf("onboarding first frame blocked for %v (expected async WebP decode)", firstFrame)
	}
	if !tt.HasText("开始设置") {
		t.Fatalf("onboarding primary CTA missing on first frame: %v", tt.Texts())
	}
	// Allow background worker to finish decoding and verify subsequent frames stay fast.
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		tt.Frame()
		if onboardingImage(nil, "onboarding-welcome-zh.webp") != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	steadyStart := time.Now()
	tt.Frame()
	if elapsed := time.Since(steadyStart); elapsed > 100*time.Millisecond {
		t.Fatalf("steady onboarding frame after decode took %v", elapsed)
	}
}

func TestAboutCredentialStorageAsyncNonBlocking(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	gate := make(chan struct{})
	var reads atomic.Int32
	a := &AboutAccess{
		AppName: "Whale Isle",
		Version: "1.0.0",
		ReadCredentialStorage: func() credentials.StorageStatus {
			reads.Add(1)
			<-gate
			return credentials.StorageStatus{Mode: "encrypted", SupportedPlatform: true}
		},
	}

	done := make(chan struct{})
	var tt *ui.Tester
	go func() {
		tt = ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(16).Children(func() {
				aboutPane(c, Model{About: a}, c.Theme().Text, c.Theme().TextMuted)
			})
		}, 640, 800)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		close(gate)
		t.Fatal("aboutPane blocked UI first frame on ReadCredentialStorage")
	}
	if !tt.HasText("Checking…") {
		close(gate)
		t.Fatalf("expected async checking placeholder on first frame, got: %v", tt.Texts())
	}

	close(gate)
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		tt.Frame()
		if tt.HasText("Encrypted by the system keychain (DPAPI)") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !tt.HasText("Encrypted by the system keychain (DPAPI)") {
		t.Fatalf("async credential storage status never appeared: %v", tt.Texts())
	}
	// Subsequent frames must reuse the cached status without re-reading disk.
	for range 4 {
		tt.Frame()
	}
	if got := reads.Load(); got != 1 {
		t.Fatalf("ReadCredentialStorage called %d times, want 1 cached load", got)
	}
}

func TestOnboardingShortWindowScrollableAndTitleBarSafe(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	for _, tc := range []struct {
		step   string
		button string
		scroll bool
	}{
		{step: "welcome", button: "Get started", scroll: false},
		{step: "credit", button: "Skip setup", scroll: true},
		{step: "purpose", button: "Skip setup", scroll: true},
		{step: "process", button: "Skip setup", scroll: true},
	} {
		t.Run(tc.step, func(t *testing.T) {
			ob := onboardingTestController(t, tc.step)
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					DesktopOnboarding(c, ob, &AccountAccess{TopUpURL: "https://example.com/topup", OnOpenTopUp: func(string) {}}, "", false)
				})
			}, 640, 440)
			tt.SetTitleBar(ui.TitleBar{Height: 12, Right: 104})
			tt.Frame()

			if tc.scroll {
				// Scroll down in the short window so the bottom action row enters the viewport.
				tt.Scroll(320, 220, 0, 1200)
				tt.Frame()
			}

			rect, ok := tt.Find(tc.button)
			if !ok {
				t.Fatalf("step %q lost button %q in 640x440 window: %v", tc.step, tc.button, tt.Texts())
			}
			if rect.W <= 0 || rect.H <= 0 {
				t.Fatalf("step %q button %q collapsed/clipped off-screen: %+v", tc.step, tc.button, rect)
			}
			if rect.Y+rect.H > 440 {
				t.Fatalf("step %q button %q clipped below 440px viewport: %+v", tc.step, tc.button, rect)
			}
			if rect.Y < 44 {
				t.Fatalf("step %q button %q intruded into Windows caption band (y<44): %+v", tc.step, tc.button, rect)
			}
		})
	}
}

func rectsOverlap(a, b ui.Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

func TestSettingsModalNarrowWindowClearsWindowsCaptionButtons(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	cases := []struct {
		name string
		w, h float32
		bar  ui.TitleBar
	}{
		{"boundary-1040x856", 1040, 856, ui.TitleBar{Height: 12, Right: 104}},
		{"1080p-at-175pct", 1097, 617, ui.TitleBar{Height: 12, Right: 104}},
		{"half-screen", 960, 720, ui.TitleBar{Height: 12, Right: 104}},
		{"compact-win11", 1024, 768, ui.TitleBar{Height: 32, Right: 138}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closed := 0
			s := store.New()
			tt := ui.NewTester(func(c *ui.Context) {
				ModalView(c, Model{Store: s, OnClose: func() { closed++ }})
			}, int(tc.w), int(tc.h))
			tt.SetTitleBar(tc.bar)
			tt.SetTime(time.Unix(1000, 0))
			tt.SetTime(time.Unix(1001, 0))
			tt.Frame()

			closeLabel := i18n.T(i18n.KeySettingsBack)
			closeRect, ok := tt.Find(closeLabel)
			if !ok {
				t.Fatalf("close button %q not found: %v", closeLabel, tt.Texts())
			}
			captionH := max(tc.bar.Height, float32(44))
			captionRect := ui.Rect{X: tc.w - tc.bar.Right, Y: 0, W: tc.bar.Right, H: captionH}
			if rectsOverlap(closeRect, captionRect) {
				t.Fatalf("settings modal close %+v overlaps Windows caption buttons %+v at %.0fx%.0f", closeRect, captionRect, tc.w, tc.h)
			}
			if err := tt.Click(closeLabel); err != nil {
				t.Fatalf("failed to click close button: %v", err)
			}
			if closed != 1 {
				t.Fatalf("close callback fired %d times, want 1", closed)
			}
		})
	}
}

func TestSkillHubEnterSubmitsSearch(t *testing.T) {
	skillHubEnglish(t)
	query := "review"
	var searched []string
	hub := &SkillHubAccess{Open: true}
	a := &SkillsAccess{
		Hub:         hub,
		HubQuery:    &query,
		TargetRoot:  "user-dsh",
		OnSearchHub: func(q string) { searched = append(searched, q) },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		skillHubDialog(c, a, c.Theme().Text, c.Theme().TextMuted)
	}, 1100, 900)

	searchBox, ok := tt.Find(i18n.T(i18n.KeySettingsSkillsHubSearch))
	if !ok {
		t.Fatalf("skill hub search input not found: %v", tt.Texts())
	}
	tt.ClickAt(searchBox.X+10, searchBox.Y+searchBox.H/2)
	tt.Key(0, ui.KeyEnter)
	if len(searched) != 1 || searched[0] != "review" {
		t.Fatalf("pressing Enter in Skill Hub search triggered %v, want [review]", searched)
	}

	// Blank query or busy state must not fire search on Enter.
	query = "   "
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if len(searched) != 1 {
		t.Fatalf("blank query submitted on Enter: %v", searched)
	}
	query = "review"
	hub.Searching = true
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	if len(searched) != 1 {
		t.Fatalf("busy hub submitted on Enter: %v", searched)
	}
}

func TestMCPFormAndSkillCreateErrorsUseDestructiveColor(t *testing.T) {
	skillHubEnglish(t)

	// Verify MCP form validation errors render with destructive color (not plain foreground).
	draft := mcpconfig.FormDraft{Transport: "stdio"}
	mcpForm := &MCPFormAccess{
		Open:  true,
		Draft: &draft,
		Errors: map[string]string{
			"id":        "Enter a valid server ID.",
			"transport": "Choose a supported transport.",
			"form":      "Fix the highlighted fields before saving.",
		},
	}
	var fgColor, errColor ui.Color
	ttMCP := ui.NewTester(func(c *ui.Context) {
		t := tk(c)
		fgColor, errColor = t.Foreground, t.DestructiveForeground
		mcpFormEditor(c, mcpForm, false, "", t.Foreground, t.MutedForeground)
	}, 900, 900)
	ttMCP.SetPreferences(ui.Preferences{ReduceMotion: true})
	ttMCP.Frame()
	if errColor == fgColor {
		t.Fatal("destructive error token matches plain foreground")
	}
	for _, want := range []string{"Enter a valid server ID.", "Choose a supported transport.", "Fix the highlighted fields before saving."} {
		if !ttMCP.HasText(want) {
			t.Fatalf("MCP validation error %q missing: %v", want, ttMCP.Texts())
		}
	}
	// Verify rendered pixels of the error line contain the destructive red hue rather than neutral foreground.
	idErrRect, ok := ttMCP.Find("Enter a valid server ID.")
	if !ok {
		t.Fatal("MCP id error rect not found")
	}
	img := ttMCP.Image()
	hasRedTint := false
	for y := int(idErrRect.Y); y < int(idErrRect.Y+idErrRect.H); y++ {
		for x := int(idErrRect.X); x < int(idErrRect.X+idErrRect.W); x++ {
			px := img.RGBAAt(x, y)
			if int(px.R)-int(px.G) > 35 && int(px.R)-int(px.B) > 35 {
				hasRedTint = true
				break
			}
		}
		if hasRedTint {
			break
		}
	}
	if !hasRedTint {
		t.Fatalf("MCP form validation error at %+v did not paint in destructive red tone (fg=%+v, err=%+v)", idErrRect, fgColor, errColor)
	}

	// Verify Skill Create validation error also paints in destructive tone.
	skillDraft := SkillCreateDraft{Description: "desc", Content: "body", Root: "user-dsh"}
	create := &SkillCreateAccess{Open: true, Draft: &skillDraft, OnSave: func(SkillCreateDraft) {}}
	ttSkill := ui.NewTester(func(c *ui.Context) {
		t := tk(c)
		skillCreateDialog(c, &SkillsAccess{Create: create}, t.Foreground, t.MutedForeground)
	}, 1100, 900)
	ttSkill.SetPreferences(ui.Preferences{ReduceMotion: true})
	ttSkill.Frame()
	ttSkill.Scroll(650, 600, 0, 4000)
	if err := ttSkill.Click("Create skill"); err != nil {
		t.Fatal(err)
	}
	ttSkill.Frame()
	errRect, ok := ttSkill.Find("Use a lowercase kebab-case name, such as review-code.")
	if !ok {
		t.Fatalf("skill create validation error missing: %v", ttSkill.Texts())
	}
	skillImg := ttSkill.Image()
	skillRed := false
	for y := int(errRect.Y); y < int(errRect.Y+errRect.H); y++ {
		for x := int(errRect.X); x < int(errRect.X+errRect.W); x++ {
			px := skillImg.RGBAAt(x, y)
			if int(px.R)-int(px.G) > 35 && int(px.R)-int(px.B) > 35 {
				skillRed = true
				break
			}
		}
		if skillRed {
			break
		}
	}
	if !skillRed {
		t.Fatalf("skill create validation error at %+v did not paint in destructive red tone", errRect)
	}
}

func BenchmarkSettingsShortcutDialogSteady(b *testing.B) {
	e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12"}, nil, func(map[string]string) error { return nil })
	e.ReferenceOpen = true
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			bodyShortcutPreference(c, e, tk(c).Foreground, tk(c).MutedForeground)
		})
	}, 900, 800)
	tt.Frame()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

func BenchmarkSettingsModalSteady(b *testing.B) {
	s := store.New()
	m := Model{
		Store:   s,
		OnClose: func() {},
		About: &AboutAccess{
			AppName: "Whale Isle",
			Version: "1.0.0",
			ReadCredentialStorage: func() credentials.StorageStatus {
				return credentials.StorageStatus{Mode: "encrypted", SupportedPlatform: true}
			},
		},
		AboutActive: true,
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ModalView(c, m)
	}, 1280, 800)
	for range 3 {
		tt.Frame()
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tt.Frame()
	}
}

// Keep design imported cleanly even if tests expand.
var _ = design.RadiusLG

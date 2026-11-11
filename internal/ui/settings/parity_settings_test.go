package settings

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestSettingsParityCatalogAll94Entries(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)

	keys := []string{
		// 1. dsh-client-ui-settings-general
		i18n.KeySettingsParityNavGroupAgent,
		i18n.KeySettingsParitySearchResults,
		i18n.KeySettingsParityAboutCenterOpen,
		i18n.KeySettingsParityAboutCenterIdle,
		i18n.KeySettingsParityAboutCenterCurrent,
		i18n.KeySettingsParityAboutCenterAvailable,
		i18n.KeySettingsParityAboutCenterDownloading,
		i18n.KeySettingsParityAboutCenterPaused,
		i18n.KeySettingsParityAboutCenterVerifying,
		i18n.KeySettingsParityAboutCenterReady,
		i18n.KeySettingsParityAboutCenterInstalling,
		i18n.KeySettingsParityAboutCenterError,
		i18n.KeySettingsParityAboutCenterErrorAlt,

		// 2. dsh-client-ui-settings-remote
		i18n.KeySettingsParityRemoteRelayConnectingHint,
		i18n.KeySettingsParityRemotePairingLifetimeHint,
		i18n.KeySettingsParityRemotePairingCountdownHint,
		i18n.KeySettingsParityRemotePersistentLinkHint,
		i18n.KeySettingsParityRemotePersistentLinkWindowHint,
		i18n.KeySettingsParityRemoteRevokeLink,
		i18n.KeySettingsParityRemoteRevokeLinkConfirm,
		i18n.KeySettingsParityRemoteRevokeLinkDisconnect,
		i18n.KeySettingsParityRemoteRevokeLinkAction,
		i18n.KeySettingsParityRemoteLinkRevoked,
		i18n.KeySettingsParityRemoteLinkRevokedShort,

		// 3. dsh-client-ui-theme
		i18n.KeySettingsParityThemeEffectLowPowerGPU,
		i18n.KeySettingsParityThemeEffectLowPowerGPUAlt,
		i18n.KeySettingsParityThemePresetDefaultLight,
		i18n.KeySettingsParityThemePresetDefaultDark,
		i18n.KeySettingsParityThemePresetMidnight,
		i18n.KeySettingsParityThemePresetCeladon,
		i18n.KeySettingsParityThemePresetViolet,
		i18n.KeySettingsParityThemePresetAmber,
		i18n.KeySettingsParityThemePresetPaper,
		i18n.KeySettingsParityThemePresetContrast,

		// 4. dsh-client-ui-plugin-manager (48)
		"settings_parity.plugin_manager.loading",
		"settings_parity.plugin_manager.error",
		"settings_parity.plugin_manager.unavailable",
		"settings_parity.plugin_manager.updating",
		"settings_parity.plugin_manager.mutationError",
		"settings_parity.plugin_manager.dismissMutationError",
		"settings_parity.plugin_manager.openPackage",
		"settings_parity.plugin_manager.enablePackage",
		"settings_parity.plugin_manager.disablePackage",
		"settings_parity.plugin_manager.enableComponent",
		"settings_parity.plugin_manager.disableComponent",
		"settings_parity.plugin_manager.enableEntry",
		"settings_parity.plugin_manager.disableEntry",
		"settings_parity.plugin_manager.removePackage",
		"settings_parity.plugin_manager.moreCount",
		"settings_parity.plugin_manager.moreEntryTitle",
		"settings_parity.plugin_manager.removeTitle",
		"settings_parity.plugin_manager.removeBody",
		"settings_parity.plugin_manager.removing",
		"settings_parity.plugin_manager.factVersion",
		"settings_parity.plugin_manager.factPackage",
		"settings_parity.plugin_manager.factAuthor",
		"settings_parity.plugin_manager.factCategory",
		"settings_parity.plugin_manager.factLicense",
		"settings_parity.plugin_manager.factAvailability",
		"settings_parity.plugin_manager.factStatus",
		"settings_parity.plugin_manager.availabilityOfficial",
		"settings_parity.plugin_manager.availabilityInstalled",
		"settings_parity.plugin_manager.availabilityInstallation",
		"settings_parity.plugin_manager.statusEnabled",
		"settings_parity.plugin_manager.statusPartial",
		"settings_parity.plugin_manager.statusDisabled",
		"settings_parity.plugin_manager.componentsHint",
		"settings_parity.plugin_manager.noComponents",
		"settings_parity.plugin_manager.componentLockedHint",
		"settings_parity.plugin_manager.entryRuntimeHint",
		"settings_parity.plugin_manager.entryActive",
		"settings_parity.plugin_manager.entryInactive",
		"settings_parity.plugin_manager.installTrigger",
		"settings_parity.plugin_manager.installSubmit",
		"settings_parity.plugin_manager.installing",
		"settings_parity.plugin_manager.installStepResolve",
		"settings_parity.plugin_manager.installStepAdd",
		"settings_parity.plugin_manager.installStepEnable",
		"settings_parity.plugin_manager.installSuccess",
		"settings_parity.plugin_manager.installRegistryPlaceholder",
		"settings_parity.plugin_manager.installCurrentRegistry",
		"settings_parity.plugin_manager.installExitCode",

		// 5. dsh-client-ui-plugin-market (7)
		"settings_parity.plugin_market.factCategory",
		"settings_parity.plugin_market.versionsLabel",
		"settings_parity.plugin_market.removeSourceNamed",
		"settings_parity.plugin_market.filter",
		"settings_parity.plugin_market.openEntry",
		"settings_parity.plugin_market.installTitle",
		"settings_parity.plugin_market.installVersionTitle",
	}

	for _, locale := range []string{"zh", "en"} {
		i18n.SetLocale(locale)
		for _, k := range keys {
			val := i18n.T(k)
			if val == "" || val == k {
				t.Fatalf("[%s] key %q resolved to %q", locale, k, val)
			}
		}
	}

	// Verify local settingsVendorLocales also contain the newly added keys.
	vendorChecks := []struct {
		ns, key, zh, en string
	}{
		{"dsh-client-ui-settings-general", "about.center.paused", "{version} 下载已暂停 · {percent}%", "{version} download paused · {percent}%"},
		{"dsh-client-ui-settings-general", "about.center.ready", "{version} 已准备好，可以安装", "{version} is ready to install"},
		{"dsh-client-ui-settings-general", "about.center.error", "更新未完成：{message}", "Update not finished: {message}"},
		{"dsh-client-ui-settings-remote", "relayConnectingHint", "正在连接中继…", "Connecting to relay…"},
		{"dsh-client-ui-settings-remote", "pairingLifetimeHint", "二维码过期或使用后自动刷新；复制的链接长期有效。", "The QR refreshes after expiry or use. Copied links stay valid until revoked."},
		{"dsh-client-ui-settings-remote", "persistentLinkHint", "长期链接可重复配对，在这里撤销。已配对设备不受影响。", "Reusable links can pair new devices until revoked here. Paired devices stay connected."},
		{"dsh-client-ui-settings-remote", "revokeLinkConfirm", "撤销后，旧链接不能再配对；已配对设备保持连接。", "The old link will stop pairing new devices. Paired devices stay connected."},
		{"dsh-client-ui-settings-remote", "linkRevoked", "旧链接已撤销，可重新复制新链接。", "Old link revoked. You can copy the replacement link."},
		{"dsh-client-ui-theme", "effect.lowPowerGpu", "检测到集成显卡，背景特效默认关闭；开启后可能影响流畅度。", "An integrated GPU was detected, so the background effect starts off; turning it on may make the app less smooth."},
	}
	for _, vc := range vendorChecks {
		nsMap, ok := settingsVendorLocales[vc.ns]
		if !ok {
			t.Fatalf("missing namespace %q in settingsVendorLocales", vc.ns)
		}
		pair, ok := nsMap[vc.key]
		if !ok {
			t.Fatalf("missing key %s.%s in settingsVendorLocales", vc.ns, vc.key)
		}
		if pair.ZH != vc.zh || pair.EN != vc.en {
			t.Fatalf("mismatch for %s.%s: got %+v want [%q, %q]", vc.ns, vc.key, pair, vc.zh, vc.en)
		}
	}

	// Verify all 94 items in tmp/parity-audit-20261109/settings.json match exact Go AST string literals.
	auditJSONPath := filepath.Join("..", "..", "..", "tmp", "parity-audit-20261109", "settings.json")
	if raw, err := os.ReadFile(auditJSONPath); err == nil {
		var items []struct {
			Pkg   string `json:"pkg"`
			Key   string `json:"key"`
			Lang  string `json:"lang"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("unmarshal settings.json: %v", err)
		}
		if len(items) != 94 {
			t.Fatalf("expected 94 items in settings.json, got %d", len(items))
		}
		literals := map[string]bool{}
		fset := token.NewFileSet()
		for _, dir := range []string{
			filepath.Join("..", "i18n"),
			".",
			filepath.Join("..", "pluginmanager"),
			filepath.Join("..", "market"),
			filepath.Join("..", "remote"),
		} {
			_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
					return nil
				}
				f, err := parser.ParseFile(fset, p, nil, 0)
				if err != nil {
					return nil
				}
				ast.Inspect(f, func(n ast.Node) bool {
					if b, ok := n.(*ast.BasicLit); ok && b.Kind == token.STRING {
						if s, err := strconv.Unquote(b.Value); err == nil && s != "" {
							literals[s] = true
						}
					}
					return true
				})
				return nil
			})
		}
		for _, it := range items {
			if !literals[it.Value] {
				t.Errorf("missing exact literal for %s.%s (%s): %q", it.Pkg, it.Key, it.Lang, it.Value)
			}
		}
	}
}

func TestSettingsAboutUpdateCenterStatesAndSubtitles(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)

	cases := []struct {
		phase     string
		version   string
		percent   float64
		errText   string
		wantZH    string
		wantEN    string
		wantAltEN string
	}{
		{
			phase:  "idle",
			wantZH: "当前版本 1.2.0",
			wantEN: "Current version 1.2.0",
		},
		{
			phase:   "current",
			version: "1.2.0",
			wantZH:  "已是最新版本（1.2.0）",
			wantEN:  "Up to date (1.2.0)",
		},
		{
			phase:   "available",
			version: "1.3.0",
			wantZH:  "发现新版本 1.3.0",
			wantEN:  "Version 1.3.0 is available",
		},
		{
			phase:   "downloading",
			version: "1.3.0",
			percent: 38,
			wantZH:  "正在下载 1.3.0 · 38%",
			wantEN:  "Downloading 1.3.0 · 38%",
		},
		{
			phase:   "paused",
			version: "1.3.0",
			percent: 62,
			wantZH:  "1.3.0 下载已暂停 · 62%",
			wantEN:  "1.3.0 download paused · 62%",
		},
		{
			phase:   "verifying",
			version: "1.3.0",
			wantZH:  "正在校验 1.3.0",
			wantEN:  "Verifying 1.3.0",
		},
		{
			phase:   "ready",
			version: "1.3.0",
			wantZH:  "1.3.0 已准备好，可以安装",
			wantEN:  "1.3.0 is ready to install",
		},
		{
			phase:   "installing",
			version: "1.3.0",
			wantZH:  "正在安装 1.3.0",
			wantEN:  "Installing 1.3.0",
		},
		{
			phase:     "error",
			errText:   "checksum mismatch",
			wantZH:    "更新未完成：checksum mismatch",
			wantEN:    "Update not finished: checksum mismatch",
			wantAltEN: "Update did not finish: checksum mismatch",
		},
	}

	for _, tc := range cases {
		t.Run(tc.phase, func(t *testing.T) {
			openCenterCalls := 0
			a := &AboutAccess{
				AppName:            "Whale Isle",
				Version:            "1.2.0",
				UpdatePhase:        tc.phase,
				UpdateVersion:      tc.version,
				UpdatePercent:      tc.percent,
				Err:                tc.errText,
				OnCheckUpdates:     func() {},
				OnDownloadUpdate:   func() {},
				OnResumeUpdate:     func() {},
				OnInstallUpdate:    func() {},
				OnOpenUpdateCenter: func() { openCenterCalls++ },
			}
			m := Model{
				Desktop:     &DesktopAccess{},
				AboutActive: true,
				About:       a,
			}

			i18n.SetLocale("zh")
			ttZH := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 900)
			ttZH.Frame()
			if !ttZH.HasText(tc.wantZH) {
				t.Fatalf("[zh/%s] missing expected text %q in %v", tc.phase, tc.wantZH, ttZH.Texts())
			}
			if !ttZH.HasText("打开更新") {
				t.Fatalf("[zh/%s] missing open updates button in %v", tc.phase, ttZH.Texts())
			}
			assertInteractiveCursors(t, "about-zh-"+tc.phase, ttZH)
			if err := ttZH.Click("打开更新"); err != nil {
				t.Fatalf("[zh/%s] click open updates: %v", tc.phase, err)
			}
			if openCenterCalls != 1 {
				t.Fatalf("[zh/%s] openCenterCalls=%d want 1", tc.phase, openCenterCalls)
			}

			i18n.SetLocale("en")
			ttEN := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 900)
			ttEN.Frame()
			if !ttEN.HasText(tc.wantEN) {
				t.Fatalf("[en/%s] missing expected text %q in %v", tc.phase, tc.wantEN, ttEN.Texts())
			}
			if tc.wantAltEN != "" && !ttEN.HasText(tc.wantAltEN) {
				t.Fatalf("[en/%s] missing expected alt text %q in %v", tc.phase, tc.wantAltEN, ttEN.Texts())
			}
			if !ttEN.HasText("Open updates") {
				t.Fatalf("[en/%s] missing Open updates button in %v", tc.phase, ttEN.Texts())
			}
			assertInteractiveCursors(t, "about-en-"+tc.phase, ttEN)
		})
	}
}

func TestSettingsRemotePageStatesAndConfirmFlow(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)

	for _, locale := range []struct {
		code              string
		relayHint         string
		countdownHint     string
		vendorLifetime    string
		persistentHint    string
		persistentWinHint string
		revokeBtn         string
		confirmDisconnect string
		confirmVendor     string
		confirmAction     string
		revokedToast      string
		revokedReset      string
	}{
		{
			code:              "zh",
			relayHint:         "正在连接中继…",
			countdownHint:     "配对码与二维码将在 120 秒后刷新；配对成功后无需重复扫码。",
			vendorLifetime:    "二维码过期或使用后自动刷新；复制的链接长期有效。",
			persistentHint:    "长期链接可重复配对，在这里撤销。已配对设备不受影响。",
			persistentWinHint: "该链接长期有效，拿到链接的人可在配对有效期内连接你的桌面端。",
			revokeBtn:         "撤销长期链接",
			confirmDisconnect: "撤销后旧链接将立即失效，已连接设备也会断开。确定继续吗？",
			confirmVendor:     "撤销后，旧链接不能再配对；已配对设备保持连接。",
			confirmAction:     "撤销并生成新链接",
			revokedToast:      "旧链接已撤销，可重新复制新链接。",
			revokedReset:      "持久链接已重置",
		},
		{
			code:              "en",
			relayHint:         "Connecting to relay…",
			countdownHint:     "Pairing code and QR refresh in 120s; once paired, you won't need to scan again.",
			vendorLifetime:    "The QR refreshes after expiry or use. Copied links stay valid until revoked.",
			persistentHint:    "Reusable links can pair new devices until revoked here. Paired devices stay connected.",
			persistentWinHint: "This link stays valid; anyone with it can connect during the pairing window.",
			revokeBtn:         "Revoke persistent link",
			confirmDisconnect: "Revoking invalidates the current link immediately and disconnects paired devices. Continue?",
			confirmVendor:     "The old link will stop pairing new devices. Paired devices stay connected.",
			confirmAction:     "Revoke and replace link",
			revokedToast:      "Old link revoked. You can copy the replacement link.",
			revokedReset:      "Persistent link reset",
		},
	} {
		t.Run(locale.code, func(t *testing.T) {
			i18n.SetLocale(locale.code)
			revokedCalls := 0
			r := &RemoteAccess{
				Enabled:               true,
				Mode:                  "relay",
				RelayHost:             "relay.whaleisle.example",
				RelayConnecting:       true,
				PairingURL:            "https://whaleisle.example/pair/xyz",
				PairingRefreshSeconds: 120,
				PersistentLink:        true,
				OnCopyLink:            func() {},
				OnRefreshPairing:      func() {},
				OnRevokeLink:          func() { revokedCalls++ },
			}
			m := Model{
				Desktop:      &DesktopAccess{},
				RemoteActive: true,
				Remote:       r,
			}
			tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 900)
			tt.Frame()

			for _, expected := range []string{
				locale.relayHint,
				locale.countdownHint,
				locale.vendorLifetime,
				locale.persistentHint,
				locale.persistentWinHint,
				locale.revokeBtn,
			} {
				if !tt.HasText(expected) {
					t.Fatalf("[%s] missing text %q in %v", locale.code, expected, tt.Texts())
				}
			}
			assertInteractiveCursors(t, "remote-"+locale.code, tt)

			// Click revoke button to open confirmation flow.
			if err := tt.Click(locale.revokeBtn); err != nil {
				t.Fatalf("[%s] click revoke button: %v", locale.code, err)
			}
			tt.Frame()
			if !tt.HasText(locale.confirmDisconnect) || !tt.HasText(locale.confirmVendor) {
				t.Fatalf("[%s] missing revoke confirmation copy in %v", locale.code, tt.Texts())
			}
			assertInteractiveCursors(t, "remote-confirm-"+locale.code, tt)

			// Confirm revocation.
			if err := tt.Click(locale.confirmAction); err != nil {
				t.Fatalf("[%s] click confirm action: %v", locale.code, err)
			}
			tt.Frame()
			if revokedCalls != 1 {
				t.Fatalf("[%s] revokedCalls=%d want 1", locale.code, revokedCalls)
			}
			if !tt.HasText(locale.revokedToast) || !tt.HasText(locale.revokedReset) {
				t.Fatalf("[%s] missing link revoked feedback in %v", locale.code, tt.Texts())
			}
		})
	}
}

func TestSettingsAppearanceLowPowerGPUAndThemePresets(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)

	themeMode := "system"
	m := Model{
		Desktop:          &DesktopAccess{},
		AppearanceActive: true,
		Appearance: &AppearanceAccess{
			Preferences: &theme.Editor{},
			Background:  &WallpaperEditor{LowPowerGPU: true},
			Theme:       &themeMode,
			LowPowerGPU: true,
		},
	}

	i18n.SetLocale("zh")
	ttZH := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 900)
	ttZH.Frame()
	wantZH := "检测到集成显卡，背景特效默认关闭；开启后可能影响流畅度。"
	if !ttZH.HasText(wantZH) {
		t.Fatalf("[zh] missing lowPowerGpu hint %q in %v", wantZH, ttZH.Texts())
	}
	assertInteractiveCursors(t, "appearance-zh", ttZH)

	i18n.SetLocale("en")
	ttEN := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { View(c, m) }) }, 1000, 900)
	ttEN.Frame()
	for _, wantEN := range []string{
		"An integrated GPU was detected, so the background effect starts off; turning it on may make the app less smooth.",
		"Integrated GPU detected. Background effects are off by default and may affect smoothness when enabled.",
		"Light",
		"Dark",
		"System",
		"DeepSeek light",
		"DeepSeek dark",
		"Midnight",
		"Celadon",
		"Violet",
		"Amber",
		"Paper",
		"Contrast",
	} {
		if !ttEN.HasText(wantEN) {
			t.Fatalf("[en] missing expected appearance text %q in %v", wantEN, ttEN.Texts())
		}
	}
	assertInteractiveCursors(t, "appearance-en", ttEN)
}

func TestSettingsOnboardingStepsAndCursorAudit(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)
	i18n.SetLocale("zh")

	for _, step := range []string{"welcome", "credit", "purpose", "process"} {
		t.Run(step, func(t *testing.T) {
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
			for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
				if s.Snapshot().Described && !s.Snapshot().Loading {
					break
				}
				time.Sleep(time.Millisecond)
			}
			ob := store.NewDesktopOnboarding(s)
			ob.Sync(store.OnboardingAccount{Enabled: true, Status: "credential-stored", CreditKnown: true, CreditFunded: true})

			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					DesktopOnboarding(c, ob, &AccountAccess{TopUpURL: "https://example.com/topup", OnOpenTopUp: func(string) {}}, "", false)
				})
			}, 1000, 800)
			tt.Frame()
			assertInteractiveCursors(t, "onboarding-"+step, tt)
			if len(tt.Texts()) == 0 {
				t.Fatalf("onboarding step %q rendered no text: %v", step, strings.Join(tt.Texts(), ", "))
			}
		})
	}
}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
	settingsstore "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/settings"
)

func TestFeaturesSectionWiresDshbotToggle(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	store, err := config.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store, settings: settingsstore.New()}
	t.Cleanup(func() { _ = s.flushPendingConfigWrites() })
	// Queue an optimistic dshbot change on the async writer without flushing to
	// verify wireSettingsBodies reads effectiveConfigFor instead of stale Get().
	s.enqueueConfigUpdate(store, func(c *config.Config) { c.DshbotEnabled = true })

	m := settings.Model{
		Store:          s.settings,
		FeaturesActive: true,
		ThemeID:        "deepseek.light",
		Features:       &settings.FeaturesAccess{},
		Interface:      &settings.InterfaceAccess{},
	}
	s.wireSettingsBodies(&m)

	if m.Interface.DshbotEnabled == nil || !*m.Interface.DshbotEnabled {
		t.Fatalf("FeaturesActive did not populate m.Interface.DshbotEnabled from effectiveConfig: %v", m.Interface.DshbotEnabled)
	}
	if m.Interface.OnSetDshbot == nil {
		t.Fatal("FeaturesActive did not populate m.Interface.OnSetDshbot")
	}

	tt := ui.NewTester(func(c *ui.Context) {
		settings.View(c, m)
	}, 1100, 1200)
	if !tt.HasText("机器人（Bots）") {
		t.Fatal("expected 机器人（Bots） row to render in Settings -> 内置功能")
	}
}

func TestAsyncConfigReadsUseEffectiveConfigDuringDebounceWindow(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	dir := t.TempDir()
	store, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := &mainState{cfgStore: store}
	t.Cleanup(func() { _ = s.flushPendingConfigWrites() })

	// Enqueue locale + interface bool edits without waiting for the 120ms debounce.
	if _, err := s.setLocalePreference("en"); err != nil {
		t.Fatal(err)
	}
	if accepted, err := s.setLocalViewTabsPreference(false); err != nil || accepted {
		t.Fatalf("setLocalViewTabsPreference(false) = %v, %v", accepted, err)
	}
	if accepted, err := s.setShowArchivedListPreference(false); err != nil || accepted {
		t.Fatalf("setShowArchivedListPreference(false) = %v, %v", accepted, err)
	}

	// Re-seeding desktopDraft immediately inside the 120ms window must see the
	// queued optimistic edits rather than rolling them back to stale store.Get().
	s.desktopDraft.seeded = false
	s.seedDesktopDraft()
	if s.desktopDraft.locale != "en" || s.desktopDraft.showViewTabs || s.desktopDraft.showArchivedList {
		t.Fatalf("seedDesktopDraft rolled back pending async config edits: %+v", s.desktopDraft)
	}
	if !s.conversationViewTabsHidden(settingsstore.ViewState{}) {
		t.Fatal("conversationViewTabsHidden missed pending async HideChatViewTabs=true")
	}

	// Running an immediate desktop bool write must flush the pending async queue
	// instead of losing the locale edit.
	if accepted, err := s.setCloseToTrayPreference(true); err != nil || !accepted {
		t.Fatalf("setCloseToTrayPreference(true) = %v, %v", accepted, err)
	}
	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Get()
	if got.Locale != "en" || !got.HideChatViewTabs || !got.HideArchivedList || !got.CloseToTray {
		t.Fatalf("pending async edits lost across flush/update: %+v", got)
	}
}

func TestVoiceInputSectionExitsLoadingWhenSpeechUnmounted(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body.RPCID,
			"result": map[string]any{
				"ok": false,
				"error": map[string]any{
					"code":    "gateway/invocation-unavailable",
					"message": "speech service not registered",
				},
			},
		})
	}))
	defer srv.Close()

	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}, HTTP: srv.Client()}
	sessions := session.New()
	sessions.Attach(client, nil)
	s := &mainState{
		dshClient:  client,
		sessions:   sessions,
		bootStatus: harness.Status{State: harness.Ready},
	}

	tt := ui.NewTester(func(c *ui.Context) {
		s.renderVoiceInputSection(c)
	}, 900, 600)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		done := s.voiceDevices != nil && s.voiceDevices.probeDone
		s.mu.Unlock()
		if done {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	tt.Frame()
	if tt.HasText("正在读取语音服务…") {
		t.Fatal("voice input section stayed stuck on loading after speech endpoint reported unavailable")
	}
	if !tt.HasText("当前运行时未启用语音识别服务。可在插件管理中安装或启用语音输入扩展。") {
		t.Fatal("expected localized unmounted speech service notice")
	}
	if !tt.HasText("重新检测语音服务") {
		t.Fatal("expected retry speech service button")
	}
}

func TestWhaleConversationStatusRendersRecoveryCard(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(previous) })

	s := &mainState{whaleOpening: true}
	tt := ui.NewTester(func(c *ui.Context) {
		s.renderWhaleConversationStatus(c, nil)
	}, 900, 400)
	if !tt.HasText("正在初始化鲸鱼娘对话…") {
		t.Fatal("expected opening notice while Whale session initializes")
	}

	s.whaleOpening = false
	s.whaleOpenErr = "打开鲸鱼娘对话：当前运行时未启用此插件或功能。"
	tt.Frame()
	if !tt.HasText("重新打开鲸鱼娘对话") || !strings.Contains(s.whaleOpenErr, "未启用") {
		t.Fatal("expected friendly recovery button and localized error text")
	}
}

func TestSettingsLeaveClearsStoreTransientErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID := body["rpcId"]
		if strings.HasSuffix(r.URL.Path, "/settings/mutate") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": rpcID,
				"result": map[string]any{"ok": false, "error": map[string]any{"code": "failed", "message": "denied"}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "server-response", "rpcId": rpcID,
			"result": map[string]any{
				"ok": true,
				"value": map[string]any{
					"hasDocument": true,
					"writable":    true,
					"namespaces": []any{map[string]any{
						"ns": "ui-chat", "revision": 1,
						"schema": json.RawMessage(`{"uid":2,"refs":{"1":{"type":"boolean"},"2":{"type":"object","dict":{"flag":1}}}}`),
						"value":  map[string]any{"flag": false},
					}},
				},
			},
		})
	}))
	defer srv.Close()

	st := settingsstore.New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.RefreshDescribe(context.Background())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snap := st.Snapshot().Value
		if snap.Described && snap.Writable {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if snap := st.Snapshot().Value; !snap.Described || !snap.Writable {
		t.Fatalf("settings store never reached described+writable state: %+v", snap)
	}

	// Produce both a field-level preference failure and a page-level validation error.
	st.PatchDescribedField(context.Background(), "ui-chat", "flag", "true")
	deadline = time.Now().Add(3 * time.Second)
	for len(st.Snapshot().Value.PreferencePending) > 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	st.PatchDescribedField(context.Background(), "ui-chat", "flag", "bad")

	before := st.Snapshot().Value
	if before.Err == "" || before.PreferenceErrors["ui-chat.flag"] == "" {
		t.Fatalf("expected non-empty transient errors before leaving section, got err=%q prefErrs=%v", before.Err, before.PreferenceErrors)
	}

	s := &mainState{settings: st}
	m := settings.Model{Store: st}
	s.leaveSettingsSection(m, settings.SectionConversation)
	if snap := st.Snapshot().Value; snap.Err != "" || len(snap.PreferenceErrors) != 0 {
		t.Fatalf("leaveSettingsSection did not clear transient errors: %+v", snap)
	}
}

func TestDeepSeekAPIsRefreshDoesNotCancelOrPoisonSettingsStore(t *testing.T) {
	var describeCalls, credCalls, mutateCalls int
	flagVal := false
	rev := 1

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Small delay so a buggy defer cancel() around async calls would reliably
		// abort the request before the response writes back.
		time.Sleep(25 * time.Millisecond)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcID := body["rpcId"]
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			describeCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": rpcID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"hasDocument": true,
						"writable":    true,
						"namespaces": []any{
							map[string]any{
								"ns": "ui-chat", "revision": rev,
								"schema": json.RawMessage(`{"uid":2,"refs":{"1":{"type":"boolean"},"2":{"type":"object","dict":{"flag":1}}}}`),
								"value":  map[string]any{"flag": flagVal},
							},
							map[string]any{
								"ns": settingsstore.DeepSeekToolsNamespace, "revision": 1,
								"schema": json.RawMessage(`{"uid":3,"refs":{"1":{"type":"string"},"2":{"type":"boolean"},"3":{"type":"object","dict":{"apiKeyEnv":1,"tts":2}}}}`),
								"value":  map[string]any{"apiKeyEnv": settingsstore.CredRefAPIKey, "tts": false},
							},
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/credentials/describe"):
			credCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": rpcID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						settingsstore.CredRefAPIKey: map[string]any{
							"configured": true, "writable": true,
						},
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/mutate"):
			mutateCalls++
			rev++
			flagVal = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": rpcID,
				"result": map[string]any{
					"ok": true,
					"value": map[string]any{
						"ns": "ui-chat", "revision": rev,
						"value": map[string]any{"flag": flagVal},
						"user":  map[string]any{"flag": flagVal},
					},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "server-response", "rpcId": rpcID,
				"result": map[string]any{"ok": true, "value": map[string]any{}},
			})
		}
	}))
	defer srv.Close()

	st := settingsstore.New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.RefreshDescribe(context.Background())

	waitUntil := func(cond func() bool, msg string) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if cond() {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal(msg)
	}

	waitUntil(func() bool {
		snap := st.Snapshot().Value
		return snap.Described && snap.Writable && !snap.Loading
	}, "initial settings/describe did not complete")

	s := &mainState{settings: st}
	// Simulate entering the DeepSeek APIs settings tab.
	s.refreshDeepSeekAPIs()
	waitUntil(func() bool {
		snap := st.Snapshot().Value
		key := st.SnapshotDeepSeekKey()
		return describeCalls >= 2 && credCalls >= 1 && !snap.Loading && !key.Loading
	}, "DeepSeek APIs refresh did not finish")

	snap := st.Snapshot().Value
	if snap.Err != "" || !snap.Described {
		t.Fatalf("DeepSeek APIs refresh poisoned settings store: described=%v err=%q detail=%q", snap.Described, snap.Err, snap.ErrDetail)
	}
	if key := st.SnapshotDeepSeekKey(); key.Err != "" || !key.Configured {
		t.Fatalf("DeepSeek key refresh failed or canceled: %+v", key)
	}

	// Simulate switching tabs via OnSelectVoice (which must also be wrapped by settingsSectionModel).
	voiceSelected := false
	m := s.settingsSectionModel(settings.Model{
		Store:         st,
		OnSelectVoice: func() { voiceSelected = true },
	}, nil)
	m.OnSelectVoice()
	if !voiceSelected {
		t.Fatal("OnSelectVoice callback did not run")
	}

	// Switching back to General/Interface preferences must still allow Host writes cleanly.
	st.PatchDescribedField(context.Background(), "ui-chat", "flag", "true")
	waitUntil(func() bool {
		return mutateCalls >= 1 && len(st.Snapshot().Value.PreferencePending) == 0
	}, "Host preference write did not settle after visiting DeepSeek APIs")

	after := st.Snapshot().Value
	if after.Err != "" || len(after.PreferenceErrors) != 0 {
		t.Fatalf("Host preference write failed after DeepSeek APIs visit: err=%q prefErrs=%v", after.Err, after.PreferenceErrors)
	}
}

package im

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/errcopy"
	imstore "whaleisle/internal/store/im"
	"whaleisle/internal/ui/i18n"
)

func TestIMStandaloneRoutesReserveTitleBarSafeArea(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(old)

	cases := []struct {
		name  string
		width int
		bar   ui.TitleBar
	}{
		{name: "narrow-windows", width: 680, bar: ui.TitleBar{Height: 12, Right: 104}},
		{name: "standard-windows", width: 1024, bar: ui.TitleBar{Height: 12, Right: 104}},
		{name: "win11-wide-caption", width: 1024, bar: ui.TitleBar{Height: 32, Right: 138}},
		{name: "macos-traffic-lights", width: 900, bar: ui.TitleBar{Height: 28, Left: 72}},
		{name: "fullscreen", width: 1280, bar: ui.TitleBar{}},
	}

	for _, tc := range cases {
		t.Run("main/"+tc.name, func(t *testing.T) {
			s := imstore.New()
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Store: s, ThemeID: "deepseek", OnClose: func() {}})
			}, tc.width, 600)
			tt.SetTitleBar(tc.bar)
			backLabel := i18n.T(i18n.KeyIMBack)
			r, ok := tt.Find(backLabel)
			if !ok {
				t.Fatalf("missing %q", backLabel)
			}
			if r.X+r.W > float32(tc.width)-tc.bar.Right+0.5 {
				t.Fatalf("%s overlaps right caption safe area: right=%.1f > %.1f", backLabel, r.X+r.W, float32(tc.width)-tc.bar.Right)
			}
		})

		t.Run("onboarding/"+tc.name, func(t *testing.T) {
			s := imstore.New()
			s.Onboarding.Show("feishu")
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Store: s, ThemeID: "deepseek", OnClose: func() {}})
			}, tc.width, 600)
			tt.SetTitleBar(tc.bar)
			r, ok := tt.Find("返回 IM")
			if !ok {
				t.Fatal("missing 返回 IM")
			}
			if r.X+r.W > float32(tc.width)-tc.bar.Right+0.5 {
				t.Fatalf("返回 IM overlaps right caption safe area: right=%.1f > %.1f", r.X+r.W, float32(tc.width)-tc.bar.Right)
			}
		})

		t.Run("office/"+tc.name, func(t *testing.T) {
			s := imstore.New()
			s.Office.Show(context.Background())
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Store: s, ThemeID: "deepseek", OnClose: func() {}})
			}, tc.width, 600)
			tt.SetTitleBar(tc.bar)
			r, ok := tt.Find("Office 页面选项")
			if !ok {
				t.Fatal("missing Office 页面选项")
			}
			if r.X+r.W > float32(tc.width)-tc.bar.Right+0.5 {
				t.Fatalf("Office options button overlaps right caption safe area: right=%.1f > %.1f", r.X+r.W, float32(tc.width)-tc.bar.Right)
			}
		})
	}
}

type officeErrorTransport struct{}

func (officeErrorTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	var req struct {
		RPCID string `json:"rpcId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	payload := `{"type":"server-response","rpcId":"` + req.RPCID + `","result":{"ok":true,"value":{"schemaVersion":1,"configured":true,"connected":false,"state":"error","tokenConfigured":true,"health":{"error":{"code":"HTTP_503","message":"upstream raw panic 0xdeadbeef"}},"config":{"protocolVersion":"office-harness.v1","baseUrl":"https://office.invalid","deviceId":"device-one","maxConcurrency":1,"heartbeatSeconds":30,"workspaces":{},"instructionPresets":{}}}}}`
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
}

func TestEmbeddedOfficeOmitsStandaloneChromeSanitizesErrorsAndLocalizes(t *testing.T) {
	old := i18n.Locale()
	defer i18n.SetLocale(old)

	s := imstore.New()
	s.AttachPlugin(&pluginrpc.Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: officeErrorTransport{}}})
	s.Office.Show(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for s.Office.Snapshot().Value.Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}

	i18n.SetLocale(i18n.LocaleEN)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() {
			View(c, Model{Embedded: true, Store: s, ThemeID: "deepseek", HarnessReady: true})
		})
	}, 1000, 1400)

	joined := strings.Join(tt.Texts(), "\n")
	if strings.Contains(joined, "Office page options") || strings.Contains(joined, "Back to IM") {
		t.Fatalf("embedded Office still rendered standalone window chrome:\n%s", joined)
	}
	if strings.Contains(joined, "upstream raw panic") || strings.Contains(joined, "0xdeadbeef") || strings.Contains(joined, "office-connection-failed:") {
		t.Fatalf("raw health error or technical code leaked into Office UI:\n%s", joined)
	}
	wantErr := errcopy.SentenceFor("office-connection-failed", "")
	if !strings.Contains(joined, wantErr) {
		t.Fatalf("missing sanitized errcopy sentence %q in:\n%s", wantErr, joined)
	}
	if !strings.Contains(joined, "Device connection") || !strings.Contains(joined, "Protocol hook preview") {
		t.Fatalf("Office UI did not switch to English:\n%s", joined)
	}

	// Verify nil-store SettingsView also localizes in ZH.
	i18n.SetLocale(i18n.LocaleZH)
	nilTT := ui.NewTester(func(c *ui.Context) {
		SettingsView(c, Model{ThemeID: "deepseek"})
	}, 700, 300)
	if !nilTT.HasText("连接 Host 后即可配置消息渠道。") {
		t.Fatalf("nil-store SettingsView ignored ZH locale: %v", nilTT.Texts())
	}
}

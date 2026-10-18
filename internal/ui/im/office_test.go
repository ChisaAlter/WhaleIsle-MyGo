package im

import (
	"context"
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	imstore "whaleisle/internal/store/im"
)

type officeUITransport struct{ calls atomic.Int32 }

func (f *officeUITransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	var req struct {
		RPCID string `json:"rpcId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"type":"server-response","rpcId":"` + req.RPCID + `","result":{"ok":true,"value":{"schemaVersion":1,"configured":true,"connected":false,"state":"idle","tokenConfigured":true,"config":{"protocolVersion":"office-harness.v1","baseUrl":"https://office.invalid","deviceId":"device-one","maxConcurrency":1,"heartbeatSeconds":30,"workspaces":{},"instructionPresets":{}}}}}`)), Request: r}, nil
}
func TestOfficeNativeEntry(t *testing.T) {
	s := imstore.New()
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: s, ThemeID: "deepseek"}) }, 800, 600)
	if !tt.HasText("AI Office") {
		t.Fatal("native settings entry missing")
	}
}
func TestOfficeNativeExplicitReviewAndFallback(t *testing.T) {
	s := imstore.New()
	f := &officeUITransport{}
	s.AttachPlugin(&pluginrpc.Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: f}})
	s.Office.Show(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for s.Office.Snapshot().Busy && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !s.Office.Snapshot().Loaded {
		t.Fatal("fixture status not loaded")
	}
	fallback := false
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: s, ThemeID: "deepseek", HarnessReady: true, OnOpenPluginWeb: func() { fallback = true }})
	}, 1100, 2200)
	for _, text := range []string{"AI Office Connector", "配置已保存，Office 尚未连接", "Office Base URL", "Device ID", "Device Token（只写入 Host 凭据存储）", "保存并连接…", "打开原版插件页面"} {
		if !tt.HasText(text) {
			t.Fatalf("missing %q", text)
		}
	}
	if f.calls.Load() != 1 {
		t.Fatal("render caused RPC")
	}
	if e := tt.Click("保存并连接…"); e != nil {
		t.Fatal(e)
	}
	if !tt.HasText("确认执行") || f.calls.Load() != 1 {
		t.Fatal("save skipped review")
	}
	if e := tt.Click("取消"); e != nil {
		t.Fatal(e)
	}
	if f.calls.Load() != 1 || s.Office.Snapshot().ReviewAction != "" {
		t.Fatal("cancel mutated")
	}
	if e := tt.Click("打开原版插件页面"); e != nil {
		t.Fatal(e)
	}
	if !fallback || s.Office.Snapshot().Open {
		t.Fatal("fallback or close isolation failed")
	}
}

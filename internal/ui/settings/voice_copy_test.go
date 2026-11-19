package settings

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// gatewayEnvelope is the string a live pass found as the whole main line of the
// voice page: a transport envelope naming an RPC method and a gateway code.
const gatewayEnvelope = "gateway/invocation-unavailable: typert gateway: speech/follow: no active Remote method exports this endpoint"

// TestVoiceFailureSpeaksASentenceAndKeepsTheEnvelopeInDetails covers the voice
// page's failure row: the sentence names the service and the next step, and the
// envelope is one disclosure away instead of being the sentence. The row is
// widgets.Alert, so the copy rides the product's single failure surface.
func TestVoiceFailureSpeaksASentenceAndKeepsTheEnvelopeInDetails(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(12).Gap(12).Children(func() {
			voiceFailureLine(c, "connection", gatewayEnvelope,
				voiceText("语音服务没有响应。点「重新连接语音服务」再试；仍然连不上就稍后再试，详情见日志。", "The speech service is not responding. Press Reconnect speech service to try again; if it still fails, try later, and check the log for details."),
				"", nil)
		})
	}, 700, 260)

	shown := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(shown, "语音服务没有响应。") {
		t.Fatalf("the page does not say what failed: %q", shown)
	}
	if !strings.Contains(shown, "重新连接语音服务") {
		t.Fatalf("the page does not say what to do next: %q", shown)
	}
	if strings.Contains(shown, "gateway/") || strings.Contains(shown, "speech/follow") {
		t.Fatalf("the envelope is the main line again: %q", shown)
	}
	if !strings.Contains(shown, "详情") {
		t.Fatalf("no details affordance holds the envelope: %q", shown)
	}
	if err := tt.Click("详情"); err != nil {
		t.Fatalf("details control: %v", err)
	}
	opened := strings.Join(tt.Texts(), "\n")
	if !strings.Contains(opened, gatewayEnvelope) {
		t.Fatalf("the details disclosure lost the envelope: %q", opened)
	}
}

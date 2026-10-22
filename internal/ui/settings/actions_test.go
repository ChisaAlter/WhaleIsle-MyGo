package settings

import "testing"

func TestSettingsActionTextHidesProtocolNames(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"settings document opened", providerCopy("设置已打开", "Settings opened")},
		{"settings document updated", providerCopy("设置已更新", "Settings updated")},
		{"openSettingsDocument: rpc unavailable", providerCopy("无法打开设置文档：", "Could not open settings: ") + "rpc unavailable"},
	}
	for _, c := range cases {
		if got := settingsActionText(c.in); got != c.want {
			t.Errorf("settingsActionText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := settingsActionText("已保存"); got != "已保存" {
		t.Errorf("a user-facing action changed: %q", got)
	}
}

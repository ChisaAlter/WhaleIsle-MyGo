package webview

import (
	"net/url"
	"testing"
)

func TestSlotURL(t *testing.T) {
	base := "http://127.0.0.1:3080"
	tests := []struct {
		name      string
		base      string
		pluginID  string
		extra     url.Values
		want      string
		wantEmpty bool
	}{
		{
			name:     "root empty plugin",
			base:     base,
			pluginID: "",
			want:     "http://127.0.0.1:3080/",
		},
		{
			name:     "dshbot vendor id",
			base:     base,
			pluginID: PluginDshbot,
			want:     "http://127.0.0.1:3080/?desktopPlugin=dshbot",
		},
		{
			name:     "dshbot insert id alias",
			base:     base,
			pluginID: "dsh-bot",
			want:     "http://127.0.0.1:3080/?desktopPlugin=dshbot",
		},
		{
			name:     "dsh-whale",
			base:     base,
			pluginID: PluginWhale,
			want:     "http://127.0.0.1:3080/?desktopPlugin=dsh-whale",
		},
		{
			name:     "dsh-im maps to scoped package",
			base:     base,
			pluginID: PluginIM,
			want:     "http://127.0.0.1:3080/?desktopPlugin=%40xmanrui%2Fdsh-im",
		},
		{
			name:     "im insert id alias",
			base:     base,
			pluginID: "xmanrui-dsh-im",
			want:     "http://127.0.0.1:3080/?desktopPlugin=%40xmanrui%2Fdsh-im",
		},
		{
			name:     "usage panel",
			base:     base,
			pluginID: PluginUsagePanel,
			want:     "http://127.0.0.1:3080/?desktopPlugin=dsh-usage-panel",
		},
		{
			name:     "usage-stats insert alias",
			base:     base,
			pluginID: "usage-stats",
			want:     "http://127.0.0.1:3080/?desktopPlugin=dsh-usage-panel",
		},
		{
			name:     "market short id",
			base:     base,
			pluginID: PluginMarket,
			want:     "http://127.0.0.1:3080/?desktopPlugin=%40deepseek-ai%2Fdsh-client-ui-settings-market",
		},
		{
			name:     "market insert alias",
			base:     base,
			pluginID: "ui-settings-market",
			want:     "http://127.0.0.1:3080/?desktopPlugin=%40deepseek-ai%2Fdsh-client-ui-settings-market",
		},
		{
			name:     "unknown plugin passthrough",
			base:     base,
			pluginID: "third-party-plugin",
			want:     "http://127.0.0.1:3080/?desktopPlugin=third-party-plugin",
		},
		{
			name:     "extra query merged",
			base:     base,
			pluginID: PluginDshbot,
			extra:    url.Values{"theme": {"dark"}, "locale": {"zh"}},
			want:     "http://127.0.0.1:3080/?desktopPlugin=dshbot&locale=zh&theme=dark",
		},
		{
			name:     "trailing slash base",
			base:     "http://127.0.0.1:3080/",
			pluginID: PluginWhale,
			want:     "http://127.0.0.1:3080/?desktopPlugin=dsh-whale",
		},
		{
			name:      "blank base",
			base:      "  ",
			pluginID:  PluginDshbot,
			wantEmpty: true,
		},
		{
			name:      "reject non-http base",
			base:      "file:///tmp",
			pluginID:  PluginDshbot,
			wantEmpty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SlotURL(tt.base, tt.pluginID, tt.extra)
			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("got %q, want empty", got)
				}
				return
			}
			if got != tt.want {
				t.Fatalf("got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestPluginSlotURL(t *testing.T) {
	if got := PluginSlotURL("http://127.0.0.1:3080", ""); got != "http://127.0.0.1:3080/" {
		t.Fatalf("root = %q", got)
	}
	got := PluginSlotURL("http://127.0.0.1:3080", "dshbot")
	if got != "http://127.0.0.1:3080/?desktopPlugin=dshbot" {
		t.Fatalf("slot = %q", got)
	}
	if PluginSlotURL("  ", "x") != "" {
		t.Fatal("expected empty for blank origin")
	}
	// Alias alignment: vendor dir → package.
	got = PluginSlotURL("http://127.0.0.1:3080", "dsh-im")
	if got != "http://127.0.0.1:3080/?desktopPlugin=%40xmanrui%2Fdsh-im" {
		t.Fatalf("im = %q", got)
	}
}

func TestCanonicalPluginID(t *testing.T) {
	if CanonicalPluginID(" dsh-bot ") != "dshbot" {
		t.Fatal("dsh-bot")
	}
	if CanonicalPluginID("custom") != "custom" {
		t.Fatal("passthrough")
	}
}

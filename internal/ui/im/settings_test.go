package im

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	imstore "whaleisle/internal/store/im"
)

func TestEmbeddedChannelsConfiguration(t *testing.T) {
	m := Model{Embedded: true, Store: imstore.New()}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 780, 900)
	text := strings.Join(tt.Texts(), "\n")
	for _, name := range []string{"Telegram", "Discord", "Slack", "AI Office"} {
		if !strings.Contains(text, name) {
			t.Fatalf("missing platform %s", name)
		}
	}
	for _, forbidden := range []string{"Send message", "Message history", "消息历史", "Generated RPC"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("messaging surface leaked into channel configuration: %s", forbidden)
		}
	}
}

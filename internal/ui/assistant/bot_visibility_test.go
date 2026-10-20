package assistant

import (
	"encoding/json"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/ui/i18n"
)

func TestBotVisibilityReviewStatesScopeAndEffects(t *testing.T) {
	for _, kind := range []string{"bot", "room"} {
		item := pluginrpc.DshbotBotItem{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "exact-id", SessionID: "canonical", Kind: kind, Name: "Fixture", Hidden: true}, Profile: map[string]json.RawMessage{"hidden": json.RawMessage(`true`), "updatedAt": json.RawMessage(`2`), "future": json.RawMessage(`{"keep":true}`)}}
		cmd := pluginrpc.DshbotVisibilityCommand{ID: "exact-id", SessionID: "canonical", Expected: pluginrpc.DshbotBotCatalog{Revision: 8, Items: []pluginrpc.DshbotBotItem{item}}, Hidden: false, UpdatedAt: 100}
		text := botVisibilityReviewText("fixture-host", cmd)
		for _, needle := range []string{"fixture-host", "revision 8", "exact-id", "canonical", "true → false", "2 → 100", "future", "完整 items 数组 1 项"} {
			if !strings.Contains(text, needle) {
				t.Fatal("missing review", needle)
			}
		}
		if kind == "room" && !strings.Contains(text, i18n.T(i18n.KeyMiscAssistantRoom)) {
			t.Fatal("room identity")
		}
	}
	for _, needle := range []string{"显示隐藏项", "不会停用", "通知", "catalog/items", "revision", "requestSync", "后台工作", "不额外启动"} {
		if !strings.Contains(botVisibilityEffects, needle) {
			t.Fatal("missing impact", needle)
		}
	}
}

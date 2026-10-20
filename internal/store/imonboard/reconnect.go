package imonboard

import (
	"whaleisle/internal/store/storelocale"
)

import "encoding/json"

func reconnectBotID(payload map[string]any) string { id, _ := payload["botId"].(string); return id }

// The reconnect receipt is kept separately from the subsequent status refresh.
// Host selects the original private-chat test target; the client supplies no
// invented destination or fallback group broadcast.
func reconnectTestResult(raw json.RawMessage, botID string) (notice, failure string) {
	var result struct {
		Bots     []Bot `json:"bots"`
		Snapshot *struct {
			Bots []Bot `json:"bots"`
		} `json:"snapshot"`
		TestMessage *struct {
			Sent bool   `json:"sent"`
			Code string `json:"code"`
		} `json:"testMessage"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return "", storelocale.T(storelocale.KeyStoreImonboardConnectionCheckReceiptHasAnUnusableFor)
	}
	bots := result.Bots
	if result.Snapshot != nil {
		bots = result.Snapshot.Bots
	}
	connected := false
	for _, bot := range bots {
		if bot.BotID == botID {
			connected = bot.Connected
			break
		}
	}
	if !connected {
		return "", storelocale.T(storelocale.KeyStoreImonboardConnectionCheckFailedAndTheBotIsStill)
	}
	if result.TestMessage == nil {
		return storelocale.T(storelocale.KeyStoreImonboardConnectionCheckFinishedButTheHostRetur), ""
	}
	if result.TestMessage.Sent {
		return storelocale.T(storelocale.KeyStoreImonboardTestMessageSentConfirmItInYourFeishu), ""
	}
	if result.TestMessage.Code == "test-target-unavailable" {
		return storelocale.T(storelocale.KeyStoreImonboardConnectionCheckFinishedTheBotHasNotRe), ""
	}
	return storelocale.T(storelocale.KeyStoreImonboardConnectionCheckFinishedButTheTestMessa), ""
}

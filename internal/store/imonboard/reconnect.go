package imonboard

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
		return "", "连接检查回执格式不可用"
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
		return "", "连接检查失败，机器人仍未连接；请检查配置后重试。"
	}
	if result.TestMessage == nil {
		return "连接检查完成，Host 未返回测试消息回执。", ""
	}
	if result.TestMessage.Sent {
		return "测试消息已发送，请到飞书会话中确认。", ""
	}
	if result.TestMessage.Code == "test-target-unavailable" {
		return "连接检查完成。机器人尚未收到可用于测试的私聊消息。", ""
	}
	return "连接检查完成，但测试消息发送失败。", ""
}

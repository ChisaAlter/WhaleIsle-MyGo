package projects

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strings"
	"time"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
	"whaleisle/internal/ui/widgets/markdown"
)

func projectMarkdown(c *ui.Context, m Model, body string) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	markdown.View(c, body, markdown.Style{DocumentLayout: true, FontSize: theme.InterfaceSize(13), FG: chrome.Color(colors.FG), Muted: chrome.Color(colors.Muted), CodeBG: chrome.Color(colors.Field), Accent: chrome.Color(colors.Accent), Dark: m.SystemDark})
}

func processMessageText(raw json.RawMessage) string {
	var message struct {
		Content json.RawMessage `json:"content"`
		Text    string          `json:"text"`
	}
	if json.Unmarshal(raw, &message) != nil {
		return ""
	}
	if message.Text != "" {
		return message.Text
	}
	var literal string
	if json.Unmarshal(message.Content, &literal) == nil {
		return literal
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(message.Content, &parts) != nil {
		return ""
	}
	var out []string
	for _, p := range parts {
		if p.Type == "text" && p.Text != "" {
			out = append(out, p.Text)
		}
	}
	return strings.Join(out, "\n")
}

// This is the installed Project reader's semantic event ladder. It remains
// read-only: Markdown has no navigation, send, activation or tool callbacks.
func projectProcessRecords(c *ui.Context, m Model, scope string, records []projectflow.ProcessRecord) {
	names := map[string]string{}
	for _, record := range records {
		if record.Type == "event" && record.Event.Type == "tool/call" {
			var call struct {
				CallID string `json:"callId"`
				Name   string `json:"name"`
			}
			if json.Unmarshal(record.Event.Data, &call) == nil {
				names[call.CallID] = call.Name
			}
		}
	}
	for _, record := range records {
		if record.Type != "event" {
			continue
		}
		event := record.Event
		var data struct {
			Turn      int             `json:"turn"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
			Message   json.RawMessage `json:"message"`
			Source    struct {
				Kind string `json:"kind"`
			} `json:"source"`
			Error struct {
				Reason string `json:"reason"`
				Code   string `json:"code"`
			} `json:"error"`
			Reason struct {
				Kind  string `json:"kind"`
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			} `json:"reason"`
		}
		if json.Unmarshal(event.Data, &data) != nil {
			continue
		}
		stamp := ""
		if event.Time > 0 {
			stamp = time.UnixMilli(int64(event.Time)).Local().Format("2006-01-02 15:04:05")
		}
		row := ui.Column(c.Key(fmt.Sprintf("%s/record/%g", scope, event.Seq))).FillWidth().MinWidth(0).Gap(6).Padding(8, 0)
		row.Children(func() {
			heading := func(label string) {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, label).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).FontWeight(600)
					if stamp != "" {
						ui.Text(c, stamp).FontSize(theme.InterfaceSize(11)).TextColor(c.Theme().TextMuted)
					}
				})
			}
			disclosure := func(label string, body func()) {
				open := ui.Local(row, "open", func() bool { return false })
				widgets.DisclosureRow(c, widgets.DisclosureRowOptions{Title: label, Summary: stamp, Open: open, KeepSummaryWhenOpen: true, Size: theme.InterfaceSize(13), Body: body})
			}
			switch event.Type {
			case "turn/start":
				heading(fmt.Sprintf(tr("执行轮次 %d", "Execution round %d"), data.Turn))
			case "user/message":
				label := tr("工作请求", "Work request")
				if data.Source.Kind == "team-message" {
					label = tr("团队消息", "Team message")
				}
				disclosure(label, func() { projectMarkdown(c, m, processMessageText(event.Data)) })
			case "assistant/message":
				body := processMessageText(data.Message)
				if strings.TrimSpace(body) != "" {
					heading(tr("助手", "Assistant"))
					projectMarkdown(c, m, body)
				}
			case "tool/call":
				disclosure(tr("工具：", "Tool: ")+data.Name, func() {
					body := string(data.Arguments)
					var value string
					if json.Unmarshal(data.Arguments, &value) == nil {
						body = value
					}
					ui.Text(c, body).Font("monospace").FontSize(theme.InterfaceSize(12)).FillWidth().MinWidth(0).Selectable()
				})
			case "tool/result":
				var message struct {
					ToolCallID string `json:"toolCallId"`
					IsError    bool   `json:"isError"`
				}
				_ = json.Unmarshal(data.Message, &message)
				label := tr("结果：", "Result: ")
				if message.IsError {
					label = tr("失败：", "Failed: ")
				}
				name := names[message.ToolCallID]
				if name == "" {
					name = tr("工具", "Tool")
				}
				disclosure(label+name, func() {
					ui.Text(c, processMessageText(data.Message)).Font("monospace").FontSize(theme.InterfaceSize(12)).FillWidth().MinWidth(0).Selectable()
					errorText := data.Error.Reason
					if errorText == "" {
						errorText = data.Error.Code
					}
					if errorText != "" {
						ui.Text(c, errorText).FontSize(theme.InterfaceSize(12)).TextColor(c.Theme().Danger)
					}
				})
			case "turn/end":
				label := fmt.Sprintf(tr("执行轮次 %d · 已结束", "Execution round %d · Ended"), data.Turn)
				if data.Reason.Kind == "error" {
					label = fmt.Sprintf(tr("执行轮次 %d · 执行失败", "Execution round %d · Failed"), data.Turn)
				}
				disclosure(label, func() {
					if data.Reason.Error.Message != "" {
						ui.Text(c, data.Reason.Error.Message).FontSize(theme.InterfaceSize(12)).Selectable()
					}
				})
			}
		})
	}
}

package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"whaleisle/internal/ui/i18n"
)

var terminalOpenedReceipt = regexp.MustCompile(`^started terminal session (\S+)(?: \((.*?)\))? \[type: ([^\]]+)\]\n([\s\S]*)$`)
var terminalReadReceipt = regexp.MustCompile(`\n\[lines: (\d+)-(\d+) of (\d+)\](\n\[output truncated\])?$`)
var terminalSignalReceipt = regexp.MustCompile(`^delivered (\S+) to foreground process group (\d+)$`)

func toolDetailsTerminal(name string, args map[string]json.RawMessage, text string) *toolDetailsModel {
	target := toolDetailsModelTarget(args)
	item := toolDetailItem{Title: target, Fields: []toolDetailField{}}
	switch name {
	case "terminal_open":
		m := terminalOpenedReceipt.FindStringSubmatch(text)
		if m == nil {
			return nil
		}
		item.Title = m[1]
		if m[2] != "" {
			item.Title = m[2]
			item.Subtitle = toolDetailsModelString(m[1])
		}
		item.Badge = &toolDetailBadge{Label: i18n.T(i18n.KeyToolDetailReceiptStarted), Tone: "info"}
		item.Fields = append(item.Fields, toolDetailField{Label: toolDetailsFieldLabel("type"), Value: m[3]})
		item.Code = &toolDetailCode{Text: m[4]}
	case "terminal_read":
		m := terminalReadReceipt.FindStringSubmatchIndex(text)
		if m == nil {
			return nil
		}
		values := terminalReadReceipt.FindStringSubmatch(text)
		item.Subtitle = toolDetailsModelString(fmt.Sprintf(referenceCopy("Lines %s–%s of %s", "第 %s–%s 行，共 %s 行"), values[1], values[2], values[3]))
		item.Code = &toolDetailCode{Text: text[:m[0]]}
		if values[4] != "" {
			item.Description = toolDetailsModelString(i18n.T(i18n.KeyToolDetailOutputTruncated))
		}
	case "terminal_signal":
		m := terminalSignalReceipt.FindStringSubmatch(text)
		if m == nil {
			return nil
		}
		item.Badge = &toolDetailBadge{Label: referenceCopy("Signal delivered", "信号已送达"), Tone: "success"}
		item.Fields = append(item.Fields, toolDetailField{Label: i18n.T(i18n.KeyToolDetailFieldSignal), Value: m[1]}, toolDetailField{Label: referenceCopy("Process group", "进程组"), Value: m[2]})
	case "terminal_close":
		label := ""
		if text == "closed terminal session "+target {
			label = referenceCopy("Closed", "已关闭")
		}
		if text == "terminal session "+target+" was already closing" {
			label = referenceCopy("Closing", "正在关闭")
		}
		if label == "" {
			return nil
		}
		item.Badge = &toolDetailBadge{Label: label, Tone: "neutral"}
	default:
		return nil
	}
	return toolDetailsModelRecordedList([]toolDetailItem{item}, item.Title)
}

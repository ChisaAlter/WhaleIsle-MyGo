package interaction

import (
	"encoding/json"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/interaction"
	"whaleisle/internal/ui/i18n"
)

// Resolve at render time so an already-open request follows a locale change.
// The Host supplies all translations. No dictionary interpolation or external
// translation is applied, and explicitly empty localized strings stay empty.
func approvalReasonText(p interaction.Pending) string {
	raw := strings.TrimSpace(p.ApprovalDisplayReason)
	var plain string
	if strings.HasPrefix(raw, `"`) && json.Unmarshal([]byte(raw), &plain) == nil {
		return plain
	}
	var localized map[string]string
	if json.Unmarshal([]byte(raw), &localized) == nil {
		locale := i18n.Locale()
		keys := []string{strings.ToLower(locale), locale}
		if at := strings.IndexByte(locale, '-'); at >= 0 {
			keys = append(keys, strings.ToLower(locale[:at]))
		}
		keys = append(keys, "en")
		for _, key := range keys {
			if text, ok := localized[key]; ok {
				return text
			}
		}
	}
	if p.ApprovalOriginalReason != "" {
		return p.ApprovalOriginalReason
	}
	return p.Detail // Older/projected callers may only have the existing detail.
}

func approvalDetailAriaLabel() string {
	if i18n.Locale() == i18n.LocaleZH {
		return "审批详情"
	}
	return "Approval details"
}

func approvalEscalationText(p interaction.Pending) string {
	toolName := strings.TrimSpace(p.Title)
	if toolName == "" || toolName == interaction.EventApproval {
		toolName = "tool"
	}
	if i18n.Locale() == i18n.LocaleZH {
		return "工具 " + toolName + " 请求越权执行"
	}
	return "Tool " + toolName + " requests privileged execution"
}

func approvalReason(c *ui.Context, p interaction.Pending, muted ui.Color) {
	text := approvalReasonText(p)
	if text == "" && strings.TrimSpace(p.ApprovalDisplayReason) == "" && strings.TrimSpace(p.ApprovalOriginalReason) == "" {
		text = approvalEscalationText(p)
	}
	if text != "" {
		questionText(c, text, 12, muted)
	}
	if original := p.ApprovalOriginalReason; original != "" && original != text {
		label := "Original reason: "
		if i18n.Locale() == i18n.LocaleZH {
			label = "原始理由："
		}
		questionText(c, label+original, 11, muted)
	}
}

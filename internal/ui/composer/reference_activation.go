package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// activateDraftReference follows the source-owned preview route without editing
// the draft. Session, folder and unknown skill references have no file opener.
func activateDraftReference(m Model, mention DraftMention) bool {
	if m.Draft == nil || m.OnOpenReference == nil || mention.Start < 0 || mention.End > len(*m.Draft) ||
		mention.Start >= mention.End || (*m.Draft)[mention.Start:mention.End] != mention.Text {
		return false
	}
	if m.TextInput != nil {
		selection := m.TextInput.Selection().Range
		if selection.Start != selection.End {
			return false
		}
		if _, marked := m.TextInput.MarkedRange(); marked {
			return false
		}
	}
	var path string
	switch mention.Kind {
	case DraftMentionAt:
		if !strings.HasPrefix(mention.Text, "@") || strings.HasPrefix(mention.Text, "@[") {
			return false
		}
		path = strings.TrimPrefix(mention.Text, "@")
		if strings.HasPrefix(path, `"`) {
			if len(path) < 2 || !strings.HasSuffix(path, `"`) {
				return false
			}
			path = path[1 : len(path)-1]
		}
		if strings.HasSuffix(path, "/") || strings.HasSuffix(path, `\`) || strings.ContainsAny(path, "\"\x00\r\n") {
			return false
		}
	case DraftMentionSkill:
		for _, skill := range m.Skills {
			if mention.Text == "/"+skill.Name {
				path = skill.Path
				break
			}
		}
	default:
		return false
	}
	if strings.TrimSpace(path) == "" {
		return false
	}
	m.OnOpenReference(path)
	return true
}

// Chip editing, removal and movement share the native draft undo owner.
func replaceDraftReference(m Model, mention DraftMention, text string) bool {
	if m.Draft == nil {
		return false
	}
	resolved, valid := resolveMention(*m.Draft, mention)
	if !valid {
		return false
	}
	if m.TextInput != nil && m.TextInput.Draft == m.Draft {
		if _, marked := m.TextInput.MarkedRange(); marked {
			return false
		}
		text = strings.TrimSpace(text)
		if text == "" {
			removeDraftReference(m, resolved)
			return true
		}
		m.TextInput.ReplaceText(&ui.TextInputRange{Start: ui.UTF16Len((*m.Draft)[:resolved.Start]), End: ui.UTF16Len((*m.Draft)[:resolved.End])}, text)
	} else {
		*m.Draft = ReplaceDraftMention(*m.Draft, resolved, text)
	}
	return true
}

// Removal has its own control and remains a native draft edit with undo.
func removeDraftReference(m Model, mention DraftMention) {
	if m.Draft == nil || (m.State != nil && m.State.Sending) {
		return
	}
	before := *m.Draft
	after := RemoveDraftMention(before, mention)
	if after == before {
		return
	}
	if m.TextInput != nil && m.TextInput.Draft == m.Draft {
		if _, marked := m.TextInput.MarkedRange(); marked {
			return
		}
		resolved, valid := resolveMention(before, mention)
		if !valid {
			return
		}
		left := strings.TrimRight(before[:resolved.Start], " \t")
		right := strings.TrimLeft(before[resolved.End:], " \t")
		start, end, replacement := len(left), len(before)-len(right), ""
		if left == "" {
			start = 0
		} else if right != "" && !strings.HasSuffix(left, "\n") && !strings.HasPrefix(right, "\n") {
			replacement = " "
		}
		m.TextInput.ReplaceText(&ui.TextInputRange{Start: ui.UTF16Len(before[:start]), End: ui.UTF16Len(before[:end])}, replacement)
	} else {
		*m.Draft = after
	}
	closeChipEdit(m)
	if m.State != nil {
		m.State.EditFocus = true
	}
}

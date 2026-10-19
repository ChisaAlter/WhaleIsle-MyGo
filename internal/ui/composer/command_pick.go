package composer

import "strings"

// The menu uses the same decision table as the original command source:
// named local popup, leading-input claim, or detached bare Host command.
// Skill references never enter this path.
func pickSlashCommand(m Model, item suggestItem, token TokenRange) bool {
	if m.Draft == nil || m.Editing || !m.Snapshot.ClientReady || token.Start < 0 || token.End < token.Start || token.End > len(*m.Draft) {
		return false
	}
	line := strings.TrimSpace(item.insert)
	if m.State != nil && (line == "/model" || line == "/permission") {
		if m.State.TargetID != "" && m.State.TargetID != m.Snapshot.SelectedID {
			return true
		}
		if len(m.State.Attachments) > 0 {
			m.State.Error = line + " does not accept attachments; remove them before opening the picker"
			return true
		}
		if !replaceSuggestionToken(m, token, "") {
			return true
		}
		m.State.ModelCommand = ModelCommandState{Open: line == "/model", SessionID: m.Snapshot.SelectedID}
		m.State.PermissionCommand = ModelCommandState{Open: line == "/permission", SessionID: m.Snapshot.SelectedID}
		dismissSuggestion(m, m.TextInput)
		return true
	}
	for _, descriptor := range m.Snapshot.SlashCommands {
		if line != "/"+descriptor.Name {
			continue
		}
		if descriptor.Input != nil {
			if strings.TrimSpace((*m.Draft)[:token.Start]) != "" {
				return true
			}
			if !replaceSuggestionToken(m, token, item.insert) {
				return true
			}
			if state := suggestionState(m, m.TextInput); state != nil {
				state.claim = &leadingInputClaim{sessionID: m.Snapshot.SelectedID, hostGeneration: m.Snapshot.ComposerHostGeneration,
					token: (*m.Draft)[:token.Start] + item.insert, hint: descriptor.Input.Hint,
					attachments: descriptor.Input.Attachments != nil && *descriptor.Input.Attachments}
			}
			activeSuggestion(m, m.TextInput)
			dismissSuggestion(m, m.TextInput)
			return true
		}
		if m.OnPickCommand == nil {
			return false
		}
		if replaceSuggestionToken(m, token, "") {
			dismissSuggestion(m, m.TextInput)
			m.OnPickCommand(m.Snapshot.ComposerHostGeneration, m.Snapshot.SelectedID, line)
		}
		return true
	}
	return false
}

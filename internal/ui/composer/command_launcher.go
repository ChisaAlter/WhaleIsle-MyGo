package composer

import (
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"whaleisle/internal/ui/i18n"
)

type commandLauncher struct {
	sessionID, draft, query string
	token                   TokenRange
}

func toggleCommandLauncher(m Model) {
	if m.Draft == nil || m.Editing || m.State != nil && (m.State.Sending || m.State.Preparing > 0) {
		return
	}
	state := suggestionState(m, m.TextInput)
	if state == nil {
		return
	}
	if state.launcher != nil {
		dismissSuggestion(m, m.TextInput)
		return
	}
	text := *m.Draft
	caret := len(text)
	token := TokenRange{caret, caret}
	if m.TextInput != nil {
		selection := m.TextInput.Selection().Range
		caret = ui.UTF16ByteOffset(text, m.TextInput.Selection().Caret())
		token = TokenRange{ui.UTF16ByteOffset(text, selection.Start), ui.UTF16ByteOffset(text, selection.End)}
	}
	query := ""
	if kind, q, t, ok := ActiveSuggestAt(text, caret); ok && kind == SuggestSlash {
		query, token = q, t
	}
	state.dismissed = nil
	state.launcher = &commandLauncher{m.Snapshot.SelectedID, text, query, token}
	if m.State != nil {
		m.State.EditFocus = true
		m.State.ModelCommand.Open = false
		m.State.PermissionCommand.Open = false
	}
	if m.SuggestList != nil && m.SuggestList.Selected != nil {
		*m.SuggestList.Selected = 0
	}
}

// Commands keep source priority over same-name skills, matching Host command
// adjudication. Skill picks insert the literal reference and never execute it.
func slashMenuItems(m Model, query string) []suggestItem {
	if m.GroupMembers != nil {
		return nil
	}
	catalog, err := nativeSlashCatalog(m)
	if err != nil {
		if m.State != nil {
			m.State.Error = err.Error()
		}
		return nil
	}
	commands := filterNativeSlashCommands(catalog, query)
	out := make([]suggestItem, 0, len(commands)+len(m.Skills))
	seen := map[string]bool{}
	leading := true
	if m.Draft != nil {
		caret := len(*m.Draft)
		if m.TextInput != nil {
			caret = ui.UTF16ByteOffset(*m.Draft, m.TextInput.Selection().Caret())
		}
		if _, _, token, ok := ActiveSuggestAt(*m.Draft, caret); ok {
			leading = strings.TrimSpace((*m.Draft)[:token.Start]) == ""
		}
	}
	for _, cmd := range commands {
		if !leading {
			input := false
			for _, descriptor := range m.Snapshot.SlashCommands {
				if "/"+descriptor.Name == cmd.Line && descriptor.Input != nil {
					input = true
					break
				}
			}
			if input {
				continue
			}
		}
		seen[strings.ToLower(strings.TrimPrefix(cmd.Line, "/"))] = true
		out = append(out, suggestItem{label: slashCommandCaption(cmd), sub: cmd.Description, insert: cmd.Line + " ", kind: SuggestSlash, command: true})
	}
	child := false
	for _, item := range m.Snapshot.Items {
		if item.SessionID == m.Snapshot.SelectedID && item.Origin == "subagent" {
			child = true
		}
	}
	if !child {
		for _, sk := range FilterSkills(m.Skills, query, len(m.Skills)) {
			if seen[strings.ToLower(sk.Name)] {
				continue
			}
			seen[strings.ToLower(sk.Name)] = true
			sub := sk.Description
			if sk.UserOnly {
				if sub != "" {
					sub = " · " + sub
				}
				sub = i18n.T(i18n.KeyComposerSkillUserOnly) + sub
			}
			out = append(out, suggestItem{label: "/" + sk.Name, sub: sub, insert: SkillInsert(sk.Name), kind: SuggestSlash})
		}
	}
	if state := suggestionState(m, m.TextInput); state != nil && state.launcher != nil {
		out = append(out, composerToolItems(m, query)...)
	}
	return out
}
func isSkillOnlyInvocation(m Model, line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	name := strings.TrimPrefix(fields[0], "/")
	catalog, _ := nativeSlashCatalog(m)
	for _, cmd := range catalog {
		parts := strings.Fields(cmd.Line)
		if len(parts) > 0 && strings.EqualFold(strings.TrimPrefix(parts[0], "/"), name) {
			return false
		}
	}
	for _, sk := range m.Skills {
		if sk.Name == name {
			return true
		}
	}
	return false
}

func slashCommandCaption(c SlashCommand) string {
	if c.Label != "" {
		return c.Label
	}
	return c.Line
}
func filterNativeSlashCommands(rows []SlashCommand, query string) []SlashCommand {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		// Fixed presentation sections preserve Host order within each section.
		section := func(name string) int {
			switch name {
			case "/file", "/goal", "/plan", "/feedback":
				return 0
			case "/compact", "/permission", "/model", "/export":
				return 1
			}
			return 2
		}
		out := append([]SlashCommand(nil), rows...)
		sort.SliceStable(out, func(i, j int) bool { return section(out[i].Line) < section(out[j].Line) })
		return out
	}
	type match struct {
		row   SlashCommand
		score int
	}
	matches := []match{}
	for _, r := range rows {
		best, ok := modelNameScore(strings.TrimPrefix(r.Line, "/"), q)
		if score, found := modelNameScore(r.Label, q); found && (!ok || score > best) {
			best, ok = score, true
		}
		if ok {
			matches = append(matches, match{r, best})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].score > matches[j].score })
	out := make([]SlashCommand, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.row)
	}
	return out
}

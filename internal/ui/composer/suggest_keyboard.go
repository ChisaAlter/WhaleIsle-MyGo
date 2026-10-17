package composer

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

// A dismissal belongs to a token identity, not a painted frame. Merely
// repainting or restoring the same caret cannot reopen the dismissed menu.
type suggestIdentity struct {
	kind   SuggestKind
	query  string
	token  TokenRange
	quoted bool
}

type SuggestionState struct {
	launcher  *commandLauncher
	dismissed *suggestIdentity
	active    *suggestIdentity
}

func suggestionState(m Model, client *DraftClient) *SuggestionState {
	if m.State != nil {
		return &m.State.Suggestions
	}
	if client != nil {
		return &client.suggestions
	}
	return nil
}

func activeSuggestion(m Model, client *DraftClient) (SuggestKind, string, TokenRange, bool, bool) {
	if m.Draft == nil {
		return SuggestNone, "", TokenRange{}, false, false
	}
	draft := *m.Draft
	caret := len(draft)
	if client != nil {
		caret = ui.UTF16ByteOffset(draft, client.Selection().Caret())
	}
	kind, query, tok, open := ActiveSuggestAt(draft, caret)
	if state := suggestionState(m, client); state != nil && state.launcher != nil {
		launch := state.launcher
		if launch.sessionID != m.Snapshot.SelectedID || launch.draft != draft || m.Editing {
			state.launcher = nil
		} else {
			kind, query, tok, open = SuggestSlash, launch.query, launch.token, true
		}
	}

	if m.Editing && kind == SuggestSlash {
		open = false
	}
	quoted := open && kind == SuggestMention && strings.HasPrefix(draft[tok.Start:tok.End], `@"`)
	state := suggestionState(m, client)
	if state != nil {
		if !open {
			// Submission bookkeeping must not re-arm a dismissed token.
			if m.State == nil || !m.State.Sending && m.State.Preparing == 0 {
				state.dismissed = nil
			}
			state.active = nil
		} else {
			id := suggestIdentity{kind: kind, query: query, token: tok, quoted: quoted}
			if state.active == nil || *state.active != id {
				if m.SuggestList != nil && m.SuggestList.Selected != nil {
					*m.SuggestList.Selected = 0
				}
			}
			state.active = &id
			if state.dismissed != nil {
				if *state.dismissed == id {
					open = false
				} else {
					state.dismissed = nil
				}
			}
		}
	}
	return kind, query, tok, quoted, open
}

// handleSuggestKey runs ahead of draft submission and edit/stop Escape. Native
// input calls it directly so element shortcuts cannot bypass IME arbitration.
func handleSuggestKey(client *DraftClient, m Model, ev ui.InputEvent) bool {
	if ev.Kind != ui.InputKeyDown || client != nil && client.marked != nil ||
		m.State != nil && (m.State.ModelCommand.Visible(m.Snapshot.SelectedID) || m.State.PermissionCommand.Visible(m.Snapshot.SelectedID)) {
		return false
	}
	if ev.Mods != 0 && !(ev.Key == ui.KeyTab && ev.Mods == ui.Shift) {
		return false
	}
	switch ev.Key {
	case ui.KeyEnter, ui.KeyEscape, ui.KeyTab, ui.KeyUp, ui.KeyDown:
	default:
		return false
	}
	kind, query, tok, quoted, open := activeSuggestion(m, client)
	if !open || m.SuggestList == nil || m.SuggestList.Selected == nil {
		return false
	}
	items := buildSuggestItemsWithQuote(m, kind, query, quoted)
	if len(items) == 0 {
		return false
	}
	sel := m.SuggestList.Selected
	if *sel < 0 || *sel >= len(items) {
		*sel = 0
	}
	switch ev.Key {
	case ui.KeyEscape:
		dismissSuggestion(m, client)
	case ui.KeyTab:
		if ev.Mods == ui.Shift {
			dismissSuggestion(m, client)
		} else if !ev.Repeat {
			applySuggest(m, items[*sel], tok)
		}
	case ui.KeyEnter:
		if !ev.Repeat {
			applySuggest(m, items[*sel], tok)
		}
	case ui.KeyUp:
		if *sel > 0 {
			*sel--
		}
	case ui.KeyDown:
		if *sel < len(items)-1 {
			*sel++
		}
	}
	return true
}

func dismissSuggestion(m Model, client *DraftClient) {
	if state := suggestionState(m, client); state != nil {
		state.launcher = nil
	}
	if state := suggestionState(m, client); state != nil && state.active != nil {
		id := *state.active
		state.dismissed = &id
	}
}

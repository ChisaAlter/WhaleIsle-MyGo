package composer

import (
	"github.com/egoist/mygo/ui"
	"strings"
)

type referenceCrumb struct {
	label, insert string
	current       bool
}

func applySuggestDrill(m Model, it suggestItem, tok TokenRange) {
	if m.Draft == nil || it.drill == "" {
		return
	}
	*m.Draft = ReplaceToken(*m.Draft, tok, it.drill)
	if m.TextInput != nil {
		m.TextInput.Sync()
		caret := ui.UTF16Len((*m.Draft)[:tok.Start+len(it.drill)])
		m.TextInput.SetSelectionUTF16(caret, caret)
	}
	if state := suggestionState(m, m.TextInput); state != nil {
		state.dismissed = nil
		activeSuggestion(m, m.TextInput)
		if state.active != nil {
			id := *state.active
			state.drilled = &id
		}
	}
}

func referenceBreadcrumbs(m Model) []referenceCrumb {
	kind, query, _, quoted, open := activeSuggestion(m, m.TextInput)
	state := suggestionState(m, m.TextInput)
	if !open || kind != SuggestMention || state == nil || state.drilled == nil {
		return nil
	}
	slash := strings.LastIndex(query, "/")
	if slash < 0 {
		return nil
	}
	root := "@"
	if quoted {
		root = `@"`
	}
	out := []referenceCrumb{{label: "工作区 / Workspace", insert: root}}
	parts := strings.Split(query[:slash], "/")
	path := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if path != "" {
			path += "/"
		}
		path += part
		mention, ok := FormatFileMention(path, true, quoted)
		if !ok {
			return nil
		}
		out = append(out, referenceCrumb{label: part, insert: mention})
	}
	out[len(out)-1].current = true
	return out
}

func paintReferenceBreadcrumbs(c *ui.Context, m Model, tok TokenRange) {
	crumbs := referenceBreadcrumbs(m)
	if len(crumbs) == 0 {
		return
	}
	ui.Row(c).Gap(3).Children(func() {
		for _, crumb := range crumbs {
			if ui.Button(c, crumb.label).Disabled(crumb.current).Clicked() {
				applySuggestDrill(m, suggestItem{kind: SuggestMention, drill: crumb.insert}, tok)
			}
		}
	})
}

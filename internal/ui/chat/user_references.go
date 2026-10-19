package chat

import (
	"encoding/json"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

type userReferenceRun struct {
	Text, Raw, Path, Kind string
}

var userSessionReference = regexp.MustCompile(`^@\[([^\]\n]+)\]\(dsh-session:[^)\s]+\)`)

// Original user text is literal. Only complete reference spellings receive
// semantics; session/folder chips do not invent navigation. No discovery or
// filesystem resolution occurs while parsing or painting a historical entry.
func userReferenceRuns(text string, knownLabels ...[]string) []userReferenceRun {
	var runs []userReferenceRun
	plain := 0
	for start := 0; start < len(text); {
		r, size := utf8.DecodeRuneInString(text[start:])
		if r != '@' {
			start += size
			continue
		}
		end, kind, label, target := start, "", "", ""
		if wire := userSessionReference.FindStringSubmatch(text[start:]); wire != nil {
			end, kind, label = start+len(wire[0]), "session", wire[1]
		}
		if kind == "" && len(knownLabels) > 0 {
			for _, candidate := range knownLabels[0] {
				if candidate != "" && strings.HasPrefix(text[start+1:], candidate) && len(candidate) > len(label) {
					end, kind, label = start+1+len(candidate), "session", candidate
				}
			}
		}
		if kind == "" {
			boundary := start == 0
			if start > 0 {
				previous, _ := utf8.DecodeLastRuneInString(text[:start])
				boundary = unicode.IsSpace(previous)
			}
			if !boundary || strings.HasPrefix(text[start:], "@[") {
				start += size
				continue
			}
			if start+1 < len(text) && text[start+1] == '"' {
				if close := strings.IndexAny(text[start+2:], "\"\n"); close >= 0 && text[start+2+close] == '"' && close > 0 {
					end, target = start+3+close, text[start+2:start+2+close]
				}
			} else {
				end = start + 1
				for end < len(text) {
					next, n := utf8.DecodeRuneInString(text[end:])
					if unicode.IsSpace(next) {
						break
					}
					end += n
				}
				end -= len(text[start:end]) - len(strings.TrimRight(text[start:end], ".,;:!?，。；：！？、"))
				if end > start+1 {
					target = text[start+1 : end]
				}
			}
			if target != "" && !strings.ContainsAny(target, "\x00\r\n\"") {
				kind, label = "file", path.Base(strings.TrimRight(strings.ReplaceAll(target, `\`, "/"), "/"))
				if strings.HasSuffix(target, "/") {
					kind = "folder"
				}
			}
		}
		if kind == "" {
			start += size
			continue
		}
		if plain < start {
			runs = append(runs, userReferenceRun{Text: text[plain:start]})
		}
		runs = append(runs, userReferenceRun{Text: label, Raw: text[start:end], Path: target, Kind: kind})
		start, plain = end, end
	}
	if plain < len(text) {
		runs = append(runs, userReferenceRun{Text: text[plain:]})
	}
	return runs
}

func userReferenceText(c *ui.Context, text string, size float32, fg, muted ui.Color, open func(string), knownLabels ...[]string) {
	ui.RichText(c).Font(theme.RenderPreferences().Chat.Family).FontSize(size).FixedLineHeight(size + 8).TextColor(fg).Selectable().Children(func() {
		for _, run := range userReferenceRuns(text, knownLabels...) {
			run := run
			el := ui.Text(c, run.Text)
			if run.Kind == "" {
				continue
			}
			el = el.Tooltip(run.Raw).TextColor(muted)
			if run.Kind == "file" && open != nil {
				el = el.Label(run.Text).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleLink).Underline()
				if el.Clicked() || el.Shortcut(0, ui.KeyEnter) || el.Shortcut(0, ui.KeySpace) {
					open(run.Path)
				}
			}
		}
	})
}

// Recall labels belong only to the direct user event immediately preceding
// their sourced context event. Neither current titles nor a matching substring
// elsewhere in history establishes that relationship.
func userMessageReferenceLabels(messages []session.Message, seq float64) []string {
	if seq <= 0 {
		return nil
	}
	for _, message := range messages {
		if message.Seq != seq+1 || message.Kind != session.MessageKindContext {
			continue
		}
		var event struct {
			Data struct {
				Source  json.RawMessage `json:"source"`
				Message *struct {
					Source json.RawMessage `json:"source"`
				} `json:"message"`
			} `json:"data"`
		}
		if json.Unmarshal(message.Raw, &event) != nil {
			continue
		}
		raw := event.Data.Source
		if event.Data.Message != nil {
			raw = event.Data.Message.Source
		}
		var source struct {
			Kind       string `json:"kind"`
			References []struct {
				Label string `json:"label"`
			} `json:"references"`
		}
		if json.Unmarshal(raw, &source) != nil || source.Kind != "session-reference" {
			continue
		}
		var labels []string
		seen := map[string]bool{}
		for _, reference := range source.References {
			if reference.Label != "" && !seen[reference.Label] {
				seen[reference.Label] = true
				labels = append(labels, reference.Label)
			}
		}
		return labels
	}
	return nil
}

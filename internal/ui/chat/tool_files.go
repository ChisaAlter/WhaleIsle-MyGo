package chat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets/diff"
)

// Tool file cards consume the first-party read/edit/write wire schemas only.
// Neither output prose nor arbitrary JSON strings confer path-open authority.
type toolFileCard struct {
	Path     string
	Line     int
	Read     *toolReadBody
	Diffs    []toolFileDiff
	Intended bool
}
type toolReadBody struct {
	Path       string         `json:"path"`
	Offset     int            `json:"offset"`
	Lines      []toolReadLine `json:"lines"`
	TotalLines int            `json:"totalLines"`
	Lang       string         `json:"lang"`
}
type toolReadLine struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}
type toolFileDiff struct {
	Path    string
	OldText *string
	NewText string
}

func toolFilePath(path string) bool {
	if strings.TrimSpace(path) == "" || strings.Contains(path, "://") {
		return false
	}
	for _, r := range path {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func toolField(object map[string]json.RawMessage, key string) (string, bool) {
	raw, exists := object[key]
	var value string
	if !exists || string(raw) == "null" || json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}
func positiveToolLine(object map[string]json.RawMessage, key string) (int, bool) {
	raw, exists := object[key]
	if !exists {
		return 0, true
	}
	var value int
	if json.Unmarshal(raw, &value) != nil || value < 1 {
		return 0, false
	}
	return value, true
}

func toolFileCardFor(call *session.ToolCallView) toolFileCard {
	var card toolFileCard
	if call == nil || call.Call == nil || call.State == "preparing" || call.State == "error" || call.State == "stopped" {
		return card
	}
	var args map[string]json.RawMessage
	if json.Unmarshal([]byte(call.Arguments), &args) != nil || args == nil {
		return card
	}
	key := "file_path"
	switch call.Name {
	case "read", "read_image", "write", "edit":
	case "str_replace_editor":
		key = "path"
	default:
		return card
	}
	path, ok := toolField(args, key)
	if !ok || !toolFilePath(path) {
		return card
	}
	card.Path = path
	if call.Name == "read" {
		line, lineOK := positiveToolLine(args, "offset")
		_, limitOK := positiveToolLine(args, "limit")
		if !lineOK || !limitOK {
			return toolFileCard{}
		}
		card.Line = line
	}
	// Nested PTC results and missing call heads retain their generic evidence.
	if call.ParentCallID != "" {
		return card
	}
	var event struct {
		Data struct {
			Meta    json.RawMessage `json:"meta"`
			Message struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		} `json:"data"`
	}
	if call.Result != nil {
		if call.State != "ok" || call.Result.EventType != "tool/result" || json.Unmarshal(call.Result.Raw, &event) != nil {
			return card
		}
	}
	if call.Name == "read" && call.Result != nil {
		var fields map[string]json.RawMessage
		var body toolReadBody
		if json.Unmarshal(event.Data.Meta, &fields) != nil || fields == nil || json.Unmarshal(event.Data.Meta, &body) != nil {
			return card
		}
		// Required fields may not silently acquire Go zero values on malformed data.
		for _, key := range []string{"path", "offset", "lines", "totalLines"} {
			if raw, ok := fields[key]; !ok || string(raw) == "null" {
				return card
			}
		}
		if _, exists := fields["lang"]; exists {
			if _, ok := toolField(fields, "lang"); !ok {
				return card
			}
		}
		if !toolFilePath(body.Path) || body.Offset < 1 || body.TotalLines < 0 {
			return card
		}
		previous := body.Offset - 1
		var rawLines []map[string]json.RawMessage
		if json.Unmarshal(fields["lines"], &rawLines) != nil {
			return card
		}
		for i, line := range body.Lines {
			if _, ok := toolField(rawLines[i], "text"); !ok || line.Number <= previous || line.Number > body.TotalLines {
				return card
			}
			previous = line.Number
		}
		content := event.Data.Message.Content
		if len(content) != 1 || content[0].Type != "text" {
			return card
		}
		text := content[0].Text
		header := strings.Index(text, "</path>\n<type>file</type>\n<content>\n")
		if !strings.HasPrefix(text, "<path>") || header < 6 || strings.Contains(text[6:header], "\n") || !strings.HasSuffix(text, "\n</content>") {
			return card
		}
		card.Read = &body
		return card
	}
	intended, ok := intendedToolDiff(call.Name, args, path)
	if !ok {
		return card
	}
	if call.Result == nil {
		card.Diffs, card.Intended = []toolFileDiff{intended}, true
		return card
	}
	if call.Name == "str_replace_editor" {
		return card
	}
	var meta map[string]json.RawMessage
	if json.Unmarshal(event.Data.Meta, &meta) == nil {
		if applied, ok := appliedToolDiffs(meta["diffs"]); ok && len(applied) > 0 {
			card.Diffs = applied
			return card
		}
	}
	// A successful write can report an empty diff for create/identical overwrite.
	if call.Name == "write" {
		card.Diffs = []toolFileDiff{intended}
	}
	return card
}

func intendedToolDiff(name string, args map[string]json.RawMessage, path string) (toolFileDiff, bool) {
	d := toolFileDiff{Path: path}
	switch name {
	case "write":
		value, ok := toolField(args, "content")
		d.NewText = value
		return d, ok
	case "edit":
		old, oldOK := toolField(args, "old_string")
		next, nextOK := toolField(args, "new_string")
		if raw, exists := args["replace_all"]; exists {
			var flag bool
			if string(raw) == "null" || json.Unmarshal(raw, &flag) != nil {
				return d, false
			}
		}
		d.OldText, d.NewText = &old, next
		return d, oldOK && nextOK
	case "str_replace_editor":
		command, _ := toolField(args, "command")
		optional := func(key string) (string, bool) {
			if _, exists := args[key]; !exists {
				return "", true
			}
			return toolField(args, key)
		}
		switch command {
		case "create":
			value, ok := optional("file_text")
			d.NewText = value
			return d, ok
		case "str_replace":
			old, oldOK := optional("old_str")
			next, nextOK := optional("new_str")
			d.OldText, d.NewText = &old, next
			return d, oldOK && nextOK
		}
	}
	return d, false
}
func appliedToolDiffs(raw json.RawMessage) ([]toolFileDiff, bool) {
	var hunks []map[string]json.RawMessage
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &hunks) != nil {
		return nil, false
	}
	result := make([]toolFileDiff, 0, len(hunks))
	for _, h := range hunks {
		path, pathOK := toolField(h, "path")
		next, nextOK := toolField(h, "newText")
		oldRaw, exists := h["oldText"]
		if !pathOK || !toolFilePath(path) || !nextOK || !exists {
			return nil, false
		}
		d := toolFileDiff{Path: path, NewText: next}
		if string(oldRaw) != "null" {
			old, ok := toolField(h, "oldText")
			if !ok {
				return nil, false
			}
			d.OldText = &old
		}
		result = append(result, d)
	}
	return result, true
}

func toolFileView(c *ui.Context, call *session.ToolCallView, st ToolCardStyle, opts ToolImageGalleryOptions) {
	card := toolFileCardFor(call)
	if card.Path == "" {
		return
	}
	ui.Column(c).Gap(4).FillWidth().Children(func() {
		toolFileButton(c, card.Path, card.Line, st, opts.OnOpenFile)
		if card.Read != nil {
			body := card.Read
			ui.Text(c, fmt.Sprintf("%s · %d lines", body.Path, body.TotalLines)).FontSize(10).TextColor(st.Muted)
			for i, line := range body.Lines {
				if i == 8 && !isThinkingExpanded(opts.Expanded, "tool-file:"+call.CallID) {
					break
				}
				ui.Text(c, fmt.Sprintf("%d  %s", line.Number, line.Text)).FontSize(11).TextColor(st.FG)
			}
			if len(body.Lines) > 8 {
				toolFileDisclosure(c, call.CallID, opts)
			}
		}
		if len(card.Diffs) > 0 {
			label := "已写入差异 / Applied diff"
			if card.Intended {
				label = "拟议差异 / Proposed diff"
			}
			ui.Text(c, label).FontSize(10).TextColor(st.Muted)
			for _, d := range card.Diffs {
				if d.Path != card.Path {
					toolFileButton(c, d.Path, 0, st, opts.OnOpenFile)
				}
				// Use the existing diff widget, without consulting current disk state.
				old := ""
				if d.OldText != nil {
					old = *d.OldText
				}
				// Bound resident work; full exact text remains in the original disclosures.
				if len(old)+len(d.NewText) > 256*1024 {
					ui.Text(c, "差异过大，请展开原始记录 / Large diff: expand original payload").FontSize(11)
					continue
				}
				patch := diff.LineDiff(old, d.NewText)
				if patch == "" {
					ui.Text(c, "无文本差异 / No text changes").FontSize(11).TextColor(st.Muted)
					continue
				}
				lines := strings.Split(patch, "\n")
				if len(lines) > 9 && !isThinkingExpanded(opts.Expanded, "tool-file:"+call.CallID) {
					patch = strings.Join(lines[:9], "\n")
				}
				diff.View(c, patch, diff.Style{FG: st.FG, Muted: st.Muted, File: st.Muted, Hunk: st.Accent, Add: ui.RGB(61, 153, 112), Del: ui.RGB(224, 102, 102)})
				if len(lines) > 9 {
					toolFileDisclosure(c, call.CallID, opts)
				}
			}
		}
	})
}
func toolFileButton(c *ui.Context, path string, line int, st ToolCardStyle, open func(string)) {
	label := path
	if line > 0 {
		label += ":" + strconv.Itoa(line)
	}
	if open == nil {
		ui.Text(c, label).FontSize(11).TextColor(st.Muted)
		return
	}
	if ui.Button(c, label).FontSize(11).Clicked() {
		open(label)
	}
}
func toolFileDisclosure(c *ui.Context, id string, opts ToolImageGalleryOptions) {
	label := "展开文件内容 / Expand file content"
	if isThinkingExpanded(opts.Expanded, "tool-file:"+id) {
		label = "收起文件内容 / Collapse file content"
	}
	if ui.Button(c, label).FontSize(10).Clicked() {
		toggleThinkingExpanded(opts.Expanded, "tool-file:"+id)
	}
}

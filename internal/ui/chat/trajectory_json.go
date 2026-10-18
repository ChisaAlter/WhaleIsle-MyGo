package chat

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"io"
	"regexp"
	"sort"
	"strings"
)

var trajectoryJSONIdentifier = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

func trajectoryJSONPath(parent, key string) string {
	if trajectoryJSONIdentifier.MatchString(key) {
		return parent + "." + key
	}
	quoted, _ := json.Marshal(key)
	return parent + "[" + string(quoted) + "]"
}
func trajectoryJSONValue(raw string) (any, bool) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil {
		return nil, false
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return nil, false
	}
	switch value.(type) {
	case map[string]any, []any:
		return value, true
	}
	return nil, false
}
func trajectoryJSONCopy(value any, pretty bool) string {
	if !pretty {
		if s, ok := value.(string); ok {
			return s
		}
	}
	var raw []byte
	if pretty {
		raw, _ = json.MarshalIndent(value, "", "  ")
	} else {
		raw, _ = json.Marshal(value)
	}
	return string(raw)
}

// Raw remains the default and the outer inspector's complete copy source.
// Structured exploration never rewrites or truncates the retained event.
func trajectoryPayloadView(c *ui.Context, scope, body string, fg, muted ui.Color) {
	value, ok := trajectoryJSONValue(body)
	if !ok {
		ui.Text(c, body).FontSize(12).TextColor(fg)
		return
	}
	root := ui.Column(c.Key("trajectory-json:" + scope)).FillWidth().Gap(4)
	tree := ui.Local(root, "structured", func() bool { return false })
	root.Children(func() {
		label := "Structured JSON"
		if *tree {
			label = "Raw text"
		}
		if ui.Button(c, label).Clicked() {
			*tree = !*tree
		}
		if !*tree {
			ui.Text(c, body).FontSize(12).TextColor(fg).Selectable()
			return
		}
		budget := 200
		trajectoryJSONNode(c, "$", value, 0, &budget, fg, muted)
		if budget <= 0 {
			ui.Text(c, "展示已限量；复制保留完整内容 / Display limited; copy retains full content").FontSize(11).TextColor(muted)
		}
	})
}
func trajectoryJSONNode(c *ui.Context, path string, value any, depth int, budget *int, fg, muted ui.Color) {
	if *budget <= 0 {
		return
	}
	*budget--
	object, isObject := value.(map[string]any)
	array, isArray := value.([]any)
	container := isObject || isArray
	root := ui.Column(c.Key(path)).FillWidth().Gap(3)
	open := ui.Local(root, "expanded", func() bool { return depth == 0 })
	root.Children(func() {
		ui.Row(c).FillWidth().Gap(4).Children(func() {
			if container {
				count := len(object)
				kind := "object"
				if isArray {
					count = len(array)
					kind = "array"
				}
				if ui.Button(c, fmt.Sprintf("%s · %s (%d)", path, kind, count)).Label("Expand JSON " + path).Clicked() {
					*open = !*open
				}
			} else {
				ui.Text(c, path+": "+clipRunes(trajectoryJSONCopy(value, false), 160)).FontSize(12).TextColor(fg)
			}
		})
		ui.Row(c).Gap(4).Children(func() {
			if ui.Button(c, "Copy value").Label("Copy JSON value " + path).Clicked() {
				c.WriteClipboard(trajectoryJSONCopy(value, false))
			}
			if ui.Button(c, "Copy JSON").Label("Copy JSON pretty " + path).Clicked() {
				c.WriteClipboard(trajectoryJSONCopy(value, true))
			}
			if ui.Button(c, "Copy path").Label("Copy JSON path " + path).Clicked() {
				c.WriteClipboard(path)
			}
		})
		if !container {
			if text, ok := value.(string); ok && len([]rune(text)) > 160 {
				if ui.Button(c, "Full value").Label("Full JSON value " + path).Clicked() {
					*open = !*open
				}
				if *open {
					ui.Text(c, text).FontSize(12).TextColor(fg).Selectable()
				}
			}
			return
		}
		if !*open {
			return
		}
		if depth >= 64 {
			ui.Text(c, "深度限制 / Depth limit; copy includes subtree").TextColor(muted)
			return
		}
		if isObject {
			keys := make([]string, 0, len(object))
			for key := range object {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				if *budget <= 0 {
					break
				}
				trajectoryJSONNode(c, trajectoryJSONPath(path, key), object[key], depth+1, budget, fg, muted)
			}
		} else {
			for i, item := range array {
				if *budget <= 0 {
					break
				}
				trajectoryJSONNode(c, fmt.Sprintf("%s[%d]", path, i), item, depth+1, budget, fg, muted)
			}
		}
	})
}

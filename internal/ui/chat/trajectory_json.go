package chat

import (
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"io"
	"regexp"
	"sort"
	"strings"

	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
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
		ui.Text(c, body).FontSize(12).TextColor(fg).Selectable()
		return
	}
	root := ui.Column(c.Key("trajectory-json:" + scope)).FillWidth().Gap(design.Sp(1)).Selectable()
	tree := ui.Local(root, "structured", func() bool { return false })
	root.Children(func() {
		label := "Structured JSON"
		if *tree {
			label = "Raw text"
		}
		if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Clicked() {
			*tree = !*tree
		}
		if !*tree {
			ui.Text(c, body).FontSize(12).TextColor(fg).Selectable()
			return
		}
		budget := 200
		trajectoryJSONNode(c, "$", value, 0, &budget, fg, muted)
		if budget <= 0 {
			ui.Text(c, i18n.T(i18n.KeyChatTrajDisplayLimited)).FontSize(11).TextColor(muted)
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
	root := ui.Column(c.Key(path)).FillWidth().Gap(design.Sp(0.75))
	open := ui.Local(root, "expanded", func() bool { return depth == 0 })
	root.Children(func() {
		ui.Row(c).FillWidth().Gap(design.Sp(1)).Children(func() {
			if container {
				count := len(object)
				kind := "object"
				if isArray {
					count = len(array)
					kind = "array"
				}
				if coss.Button(c, fmt.Sprintf("%s · %s (%d)", path, kind, count), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Label("Expand JSON " + path).Clicked() {
					*open = !*open
				}
			} else {
				ui.Text(c, path+": "+clipRunes(trajectoryJSONCopy(value, false), 160)).FontSize(12).TextColor(fg)
			}
		})
		ui.Row(c).Gap(design.Sp(1)).Children(func() {
			if coss.Button(c, "Copy value", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Label("Copy JSON value " + path).Clicked() {
				c.WriteClipboard(trajectoryJSONCopy(value, false))
			}
			if coss.Button(c, "Copy JSON", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Label("Copy JSON pretty " + path).Clicked() {
				c.WriteClipboard(trajectoryJSONCopy(value, true))
			}
			if coss.Button(c, "Copy path", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Label("Copy JSON path " + path).Clicked() {
				c.WriteClipboard(path)
			}
		})
		if !container {
			if text, ok := value.(string); ok && len([]rune(text)) > 160 {
				if coss.Button(c, "Full value", coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonXS}).Label("Full JSON value " + path).Clicked() {
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
			ui.Text(c, i18n.T(i18n.KeyChatTrajDepthLimit)).TextColor(muted)
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

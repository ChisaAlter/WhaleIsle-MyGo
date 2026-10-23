package frame

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// JSON presentation state is private to the selected record, never persisted.
// Every container starts expanded, as in InspectorJsonTree in the source UI.
type inspectorJSONTreeState struct {
	raw                  string
	initialized, invalid bool
	root                 inspectorJSONTreeNode
	collapsed            map[string]bool
}

type inspectorJSONTreeNode struct {
	path, name, label string
	children          []inspectorJSONTreeNode
	container         bool
}

type inspectorJSONTreeRow struct {
	path, name, label string
	depth             int
	expandable        bool
}

func (s *inspectorJSONTreeState) setRaw(raw []byte) {
	if s.initialized && s.raw == string(raw) {
		return
	}
	s.raw, s.initialized = string(raw), true
	if s.collapsed == nil {
		s.collapsed = make(map[string]bool)
	}
	s.invalid = !json.Valid(raw)
	s.root = inspectorJSONTreeNode{}
	if !s.invalid {
		s.root = inspectorJSONNode(raw, "", "$")
	}
}

// Retain raw scalars and original field order while using the reference's
// Object / Array labels. Selecting a field never rewrites the retained event.
func inspectorJSONNode(raw json.RawMessage, path, name string) inspectorJSONTreeNode {
	node := inspectorJSONTreeNode{path: path, name: name, label: strings.TrimSpace(string(raw))}
	if node.label == "" {
		return node
	}
	switch node.label[0] {
	case '{':
		node.label, node.container = "Object", true
		for _, field := range inspectorFields(raw) {
			escaped := strings.ReplaceAll(strings.ReplaceAll(field.name, "~", "~0"), "/", "~1")
			node.children = append(node.children, inspectorJSONNode(field.raw, path+"/"+escaped, field.name))
		}
	case '[':
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return node
		}
		node.label, node.container = fmt.Sprintf("Array(%d)", len(items)), true
		for i, item := range items {
			node.children = append(node.children, inspectorJSONNode(item, path+"/"+strconv.Itoa(i), strconv.Itoa(i)))
		}
	}
	return node
}

func (s *inspectorJSONTreeState) toggle(path string) {
	if s.collapsed == nil {
		s.collapsed = make(map[string]bool)
	}
	if s.collapsed[path] {
		delete(s.collapsed, path)
	} else {
		s.collapsed[path] = true
	}
}

func (s *inspectorJSONTreeState) visibleRows() []inspectorJSONTreeRow {
	if !s.initialized || s.invalid {
		return nil
	}
	var rows []inspectorJSONTreeRow
	var visit func(inspectorJSONTreeNode, int)
	visit = func(node inspectorJSONTreeNode, depth int) {
		rows = append(rows, inspectorJSONTreeRow{path: node.path, name: node.name, label: node.label, depth: depth, expandable: node.container})
		if !s.collapsed[node.path] {
			for _, child := range node.children {
				visit(child, depth+1)
			}
		}
	}
	visit(s.root, 0)
	return rows
}

// inspectorJSONTree is hosted in the pane's details scroll. It only displays
// serialized JSON; it does not evaluate accessors or imply runtime references.
func inspectorJSONTree(c *ui.Context, state *inspectorJSONTreeState, raw []byte, colors theme.Colors) {
	state.setRaw(raw)
	if state.invalid {
		ui.Text(c, paneLabel("无法解析 JSON，显示原始内容", "Invalid JSON; showing raw content")).FontSize(11)
		ui.Text(c, state.raw).FontSize(11)
		return
	}
	ui.Column(c).FillWidth().Font(inspectorFont()).FontSize(11).LineHeight(0).Children(func() {
		for _, row := range state.visibleRows() {
			rowContext := c.Key("json-field:" + row.path)
			ui.Row(rowContext).FillWidth().Padding(2, 0, 2, float32(row.depth+1)*16).AlignItems(ui.Start).Children(func() {
				valueColor := chrome.Color(colors.Secondary)
				if row.expandable {
					valueColor = chrome.Color(colors.Muted)
				}
				spans := []ui.Span{{Text: row.name + ": ", Color: chrome.Color(colors.FG)}, {Text: row.label, Color: valueColor}}
				if row.expandable {
					marker := "▾ "
					if state.collapsed[row.path] {
						marker = "▸ "
					}
					spans = append([]ui.Span{{Text: marker, Color: chrome.Color(colors.FG)}}, spans...)
					button := inspectorButton(rowContext, row.name+": "+row.label).Expanded(!state.collapsed[row.path]).MinWidth(0)
					button.Children(func() { ui.RichText(rowContext, spans...).Font(inspectorFont()).FontSize(11).LineHeight(0).MinWidth(0) })
					if button.Clicked() {
						state.toggle(row.path)
					}
				} else {
					ui.RichText(rowContext, spans...).FontSize(11).FillWidth().MinWidth(0).Selectable()
				}
			})
		}
	})
}

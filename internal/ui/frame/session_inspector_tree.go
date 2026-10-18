package frame

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
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

// Decode containers as RawMessages so number precision and scalar spellings are
// retained. Sorted keys give object fields a stable order between renders.
func inspectorJSONNode(raw json.RawMessage, path, name string) inspectorJSONTreeNode {
	node := inspectorJSONTreeNode{path: path, name: name, label: strings.TrimSpace(string(raw))}
	switch node.label[0] {
	case '{':
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			return node
		}
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			node.label = "{}"
			return node
		}
		node.label = fmt.Sprintf("{%d}", len(keys))
		for _, key := range keys {
			escaped := strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
			label, _ := json.Marshal(key)
			node.children = append(node.children, inspectorJSONNode(fields[key], path+"/"+escaped, string(label)))
		}
	case '[':
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return node
		}
		if len(items) == 0 {
			node.label = "[]"
			return node
		}
		node.label = fmt.Sprintf("[%d]", len(items))
		for i, item := range items {
			node.children = append(node.children, inspectorJSONNode(item, path+"/"+strconv.Itoa(i), fmt.Sprintf("[%d]", i)))
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
		rows = append(rows, inspectorJSONTreeRow{path: node.path, name: node.name, label: node.label, depth: depth, expandable: len(node.children) > 0})
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
func inspectorJSONTree(c *ui.Context, state *inspectorJSONTreeState, raw []byte) {
	state.setRaw(raw)
	if state.invalid {
		ui.Text(c, paneLabel("无法解析 JSON，显示原始内容", "Invalid JSON; showing raw content")).FontSize(11)
		ui.Text(c, state.raw).FontSize(11)
		return
	}
	for _, row := range state.visibleRows() {
		rowContext := c.Key("json-field:" + row.path)
		ui.Row(rowContext).Gap(4).Children(func() {
			if row.depth > 0 {
				ui.Text(rowContext, strings.Repeat("  ", row.depth)).FontSize(11)
			}
			if row.expandable {
				marker := "▾"
				if state.collapsed[row.path] {
					marker = "▸"
				}
				if ui.Button(rowContext, marker+" "+row.name+": "+row.label).Clicked() {
					state.toggle(row.path)
				}
			} else {
				ui.Text(rowContext, row.name+": "+row.label).FontSize(11)
			}
		})
	}
}

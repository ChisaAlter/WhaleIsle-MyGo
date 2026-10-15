package chat

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

type ReadOnlyResourceOptions struct {
	OnOpenFile func(string)
	OnOpenURL  func(string)
	OnRead     func(string, bool)
	Reads      map[string]session.AgentResourceRead
}

func readOnlyResourceLink(c *ui.Context, ref string, opts ReadOnlyResourceOptions, st ToolCardStyle) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return
	}
	u, _ := url.Parse(ref)
	if u != nil && (u.Scheme == "https" || u.Scheme == "http") {
		if ui.Button(c, ref).FontSize(11).Disabled(opts.OnOpenURL == nil).Clicked() {
			opts.OnOpenURL(ref)
		}
		return
	}
	if opts.OnRead == nil {
		ui.Text(c, ref).FontSize(10).TextColor(st.Muted).Selectable()
		return
	}
	if u != nil && u.Scheme != "" && u.Scheme != "attachment" && len(u.Scheme) != 1 {
		ui.Text(c, ref).FontSize(10).TextColor(st.Muted).Selectable()
		return
	}
	read := opts.Reads[ref]
	label := "预览 / Preview: " + ref
	if read.Error != "" {
		label = "重试 / Retry: " + ref
	}
	if ui.Button(c, label).FontSize(11).Disabled(read.Busy).Clicked() {
		opts.OnRead(ref, false)
	}
}
func readOnlyToolResources(c *ui.Context, msg session.Message, opts ReadOnlyResourceOptions, st ToolCardStyle) {
	seen := map[string]bool{}
	emit := func(ref string) {
		if ref != "" && !seen[ref] {
			seen[ref] = true
			readOnlyResourceLink(c, ref, opts, st)
		}
	}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case map[string]any:
			if id, ok := v["attachmentId"].(string); ok {
				emit("attachment:" + id)
			}
			for k, x := range v {
				if k == "text" || k == "url" {
					if str, ok := x.(string); ok {
						for _, r := range readOnlyReferences(str) {
							emit(r)
						}
						if k == "url" {
							emit(str)
						}
					}
				}
				visit(x)
			}
		case []any:
			for _, x := range v {
				visit(x)
			}
		}
	}
	parse := func(raw json.RawMessage) {
		var value any
		if json.Unmarshal(raw, &value) == nil {
			visit(value)
		}
	}
	parse(msg.Raw)
	if msg.Tool != nil {
		stack := []*session.ToolCallView{msg.Tool}
		for len(stack) > 0 {
			call := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, e := range call.Events {
				parse(e.Raw)
			}
			stack = append(stack, call.Children...)
		}
	}
}

// ReadOnlyResourceResultsView exposes reads started by inline references and delivery cards.
func ReadOnlyResourceResultsView(c *ui.Context, opts ReadOnlyResourceOptions, st ToolCardStyle) {
	refs := make([]string, 0, len(opts.Reads))
	for ref := range opts.Reads {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	for _, ref := range refs {
		read := opts.Reads[ref]
		ui.Text(c, ref).FontSize(11).TextColor(st.Muted).Selectable()
		if read.Busy {
			ui.Text(c, "读取中 / Reading…").FontSize(10).TextColor(st.Muted)
		}
		if read.Error != "" {
			ui.Text(c, read.Error).FontSize(11).TextColor(st.Muted)
			if ui.Button(c, "重试 / Retry: "+ref).FontSize(10).Clicked() {
				opts.OnRead(ref, false)
			}
		}
		if read.Path != "" {
			ImageInlineView(c, read.Path, st.FG, st.Muted, nil)
		}
		if read.Text != "" {
			ui.Text(c, read.Text).FontSize(12).TextColor(st.FG).Selectable()
		}
		if !read.Busy && !read.EOF && read.Error == "" && ui.Button(c, "继续读取 / Read more: "+ref).FontSize(10).Clicked() {
			opts.OnRead(ref, true)
		}
	}
}

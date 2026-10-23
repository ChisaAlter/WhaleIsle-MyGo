package chat

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// ToolMessageView draws one complete callId lifecycle and its subcalls. The
// traversal builds nested wrappers iteratively, including malformed histories.
func ToolMessageView(c *ui.Context, msg session.Message, st ToolCardStyle, gallery ToolImageGalleryOptions) {
	if gallery.TextOnly {
		gallery.OnOpen, gallery.OnOpenURL, gallery.OnOpenFile = nil, nil, nil
	}
	if msg.Tool == nil {
		if gallery.TextOnly {
			info := ParseToolCard(msg.EventType, msg.Raw)
			info.Images, info.Links = nil, nil
			info.Detail = truncateToolText(string(msg.Raw), 200)
			toolCardInfoView(c, info, st)
			toolLifecycleDetail(c, "tool-event:"+msg.ID, i18n.T(i18n.KeyChatToolEvent), string(msg.Raw), st, gallery.Expanded, true)
			return
		}
		ToolCardView(c, msg.EventType, msg.Raw, st, gallery)
		CitationsView(c, CitationsForMessage(msg), st.Muted, st.Accent, gallery.OnOpenURL)
		return
	}
	type entry struct {
		call   *session.ToolCallView
		parent int
	}
	var tree *inspectorToolTree
	if gallery.Inspector != nil && !gallery.TextOnly {
		tree = &inspectorToolTree{owner: msg}
		gallery.Inspector.reveal.toolTrees = append(gallery.Inspector.reveal.toolTrees, tree)
	}
	// CSS border widths reserve whole device pixels (at least one). Match that
	// used width in layout too, so the connector does not drift at each level.
	scale := float64(c.DeviceScale())
	border := float32(max(1, math.Floor(.5*scale)) / scale)
	stack := []entry{{call: msg.Tool, parent: -1}}
	parents := []ui.Element{}
	seen := map[*session.ToolCallView]bool{msg.Tool: true}
	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		call := item.call
		var children ui.Element
		hasChildren := false
		for _, child := range call.Children {
			if child != nil && !seen[child] {
				hasChildren = true
				break
			}
		}
		build := func() {
			seat := ui.Column(c.Key("call:" + call.CallID)).MinWidth(0).Shrink(0).Radius(8)
			// A plain root block lets its last margin escape into the Chat
			// seat. Nested call wrappers are flex items and include that margin.
			bottom := float32(2)
			if item.parent < 0 && hasChildren {
				bottom = 0
				seat.Margin(0, 0, 2, 0)
			}
			if tree != nil {
				box := &inspectorToolBox{call: call}
				tree.boxes = append(tree.boxes, box)
				seat.ObserveLayout(func(rect ui.Rect) { box.rect = rect })
			}
			seat.Children(func() {
				childGallery := gallery
				// The same call in another owner/scope has its own disclosure.
				childGallery.ID = gallery.ID + "|tool:" + call.CallID
				toolLifecycleCard(c, call, st, childGallery)
				if hasChildren {
					children = ui.Column(c.Key("subcalls")).MinWidth(0).Shrink(0).Gap(4).
						Margin(4, 0, bottom, 22).Padding(0, 0, 0, 8).
						BorderWidth(0, 0, 0, border).BorderColor(st.Border)
				}
			})
		}
		if item.parent >= 0 {
			parents[item.parent].Children(build)
		} else {
			build()
		}
		// Re-enter the same-pass parent on the next iteration. The native tree
		// is nested without recursively building an arbitrarily deep history.
		start := len(stack)
		parent := len(parents)
		parents = append(parents, children)
		for _, child := range call.Children {
			if child == nil || seen[child] {
				continue
			}
			seen[child] = true
			stack = append(stack, entry{call: child, parent: parent})
		}
		for a, b := start, len(stack)-1; a < b; a, b = a+1, b-1 {
			stack[a], stack[b] = stack[b], stack[a]
		}
	}
}

func toolLifecycleCard(c *ui.Context, call *session.ToolCallView, st ToolCardStyle, gallery ToolImageGalleryOptions) {
	info := ToolLifecycleCardInfo(call)
	arguments := call.Arguments
	output := toolLifecycleOutputFor(call, !gallery.TextOnly)
	if gallery.TextOnly {
		info.Images, info.Links = nil, nil
		if call.Result != nil {
			info.Detail = truncateToolText(output, 200)
		}
	} else {
		arguments = prettyToolPayload(arguments)
	}
	info.Arguments = arguments
	if call.Result != nil {
		info.Output = output
		// Typed sources/fetch URL belong to the result, not the raw JSON.
		info.Web = call.Result.Web
	}
	extra := func() {
		if !gallery.TextOnly {
			toolFileView(c, call, st, gallery)
		}
		if call.ParentIssue != "" {
			ui.Text(c, i18n.T(i18n.KeyChatToolCallRelated)+": "+call.ParentIssue).FontSize(10).TextColor(st.Muted)
		}
		if !gallery.TextOnly && call.CallID != "" && gallery.OnInspectCall != nil {
			if stdButton(c, st.Colors, i18n.T(i18n.KeyChatToolInspect), buttonGhost, false).Clicked() {
				gallery.OnInspectCall(call.CallID)
			}
		}
		if !gallery.TextOnly {
			var citations []Citation
			for _, source := range call.Events {
				citations = append(citations, CitationsForMessage(source)...)
			}
			CitationsView(c, dedupeCitations(citations, 8), st.Muted, st.Accent, gallery.OnOpenURL)
		}
	}
	toolCardInfoViewExtra(c, info, st, extra, gallery)
}

// ToolLifecycleCardInfo retains the existing image/link renderers while taking
// the title and state from the authoritative paired call, never its callId.
func ToolLifecycleCardInfo(call *session.ToolCallView) ToolCardInfo {
	if call == nil {
		return ToolCardInfo{Name: "tool", Status: "event"}
	}
	source := call.Call
	if call.Result != nil {
		source = call.Result
	}
	info := ToolCardInfo{}
	if source != nil {
		info = ParseToolCard(source.EventType, source.Raw)
	} else {
		info.Detail = truncateToolText(call.Arguments, 160)
	}
	info.Name, info.Status = call.Name, call.State
	info.Links = append(markdown.Links(call.Arguments), info.Links...)
	if call.Result != nil {
		info.Detail = truncateToolText(toolLifecycleOutput(call), 200)
	}
	return info
}

func toolLifecycleDetail(c *ui.Context, key, label, body string, st ToolCardStyle, expanded *map[string]bool, textOnly bool) {
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return
	}
	open := expanded == nil || isThinkingExpanded(expanded, key)
	ui.Column(c).Key("tool-detail:" + key).Gap(3).FillWidth().MinWidth(0).Children(func() {
		if expanded != nil {
			paintDisclosureRow(c, disclosureRow{
				Key:        "tool-detail-row:" + key,
				Title:      label,
				Summary:    firstDetailLine(body),
				Open:       open,
				Expandable: true,
				OnToggle:   func() { toggleThinkingExpanded(expanded, key) },
			}, toolRowStyle(st))
		} else {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label).FontSize(11).TextColor(st.Muted).Grow(1)
				if ui.Button(c, i18n.T(i18n.KeyChatToolCopy)).FontSize(10).Label(i18n.T(i18n.KeyChatToolCopy)).Clicked() {
					if textOnly {
						c.Services().WriteClipboard(body)
					} else {
						mygo.Clipboard.WriteText(body)
					}
				}
			})
		}
		if open {
			ui.Scroll(c).FillWidth().MinWidth(0).MaxHeight(toolRowBlockMaxHeight).Children(func() {
				ui.Text(c, body).FontSize(11).Font(theme.RenderPreferences().Code.Family).TextColor(st.Muted).Selectable()
			})
		}
	})
}

func prettyToolPayload(body string) string {
	var out bytes.Buffer
	if json.Indent(&out, []byte(body), "", "  ") == nil {
		return out.String()
	}
	return body
}

func toolLifecycleOutput(call *session.ToolCallView) string {
	return toolLifecycleOutputFor(call, true)
}

func toolLifecycleOutputFor(call *session.ToolCallView, allowGallery bool) string {
	if call == nil || call.Result == nil {
		return ""
	}
	var event struct {
		Data struct {
			Content []json.RawMessage `json:"content"`
			Message struct {
				Content []json.RawMessage `json:"content"`
			} `json:"message"`
			Error *struct {
				Name   string `json:"name"`
				Code   string `json:"code"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"data"`
	}
	if json.Unmarshal(call.Result.Raw, &event) != nil {
		return string(call.Result.Raw)
	}
	content := event.Data.Content
	if call.Result.EventType == "tool/result" {
		content = event.Data.Message.Content
	}
	// Images still use the validated native gallery. On error/invalid references,
	// keep their full JSON evidence, matching the existing renderer's behavior.
	gallery := allowGallery && call.State == "ok" && len(toolImageReferences(content)) > 0
	body := toolImageText(content, gallery)
	if !allowGallery {
		body = readOnlyToolContent(content)
	}
	parts := []string{body}
	if e := event.Data.Error; e != nil {
		detail := strings.TrimSpace(e.Reason)
		if detail == "" {
			detail = strings.Trim(strings.TrimSpace(e.Name)+": "+strings.TrimSpace(e.Code), ": ")
		}
		if detail != "" {
			parts = append(parts, detail)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

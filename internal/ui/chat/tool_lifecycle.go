package chat

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/markdown"
)

// ToolMessageView draws one complete callId lifecycle and its subcalls. The
// projection and rendering both avoid recursion, including malformed histories.
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
		call  *session.ToolCallView
		depth int
	}
	stack := []entry{{msg.Tool, 0}}
	seen := map[*session.ToolCallView]bool{}
	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if item.call == nil || seen[item.call] {
			continue
		}
		seen[item.call] = true
		call := item.call
		childGallery := gallery
		// Keep the owner scope in the row key: the same call id in another
		// message/scope must not inherit an expanded seat.
		childGallery.ID = gallery.ID + "|tool:" + call.CallID
		depth := item.depth
		if depth > 6 {
			depth = 6
		}
		ui.Column(c).Padding(0, 0, 4, float32(depth)*14).Gap(4).FillWidth().Children(func() {
			if item.depth > 0 {
				ui.Text(c, "↳ "+call.ParentCallID).FontSize(10).TextColor(st.Muted)
			}
			info := ToolLifecycleCardInfo(call)
			arguments := call.Arguments
			output := toolLifecycleOutputFor(call, !childGallery.TextOnly)
			if childGallery.TextOnly {
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
				// The typed web metadata (sources / fetch URL) lives on the result
				// message and drives the web card, not the raw JSON.
				info.Web = call.Result.Web
			}
			// Everything below belongs to the expanded body: a collapsed row is
			// one 24px line, as the Original ToolRow keeps it.
			extra := func() {
				if !childGallery.TextOnly {
					toolFileView(c, call, st, childGallery)
				}
				if call.ParentIssue != "" {
					ui.Text(c, i18n.T(i18n.KeyChatToolCallRelated)+": "+call.ParentIssue).FontSize(10).TextColor(st.Muted)
				}
				if !childGallery.TextOnly && call.CallID != "" && childGallery.OnInspectCall != nil {
					if stdButton(c, st.Colors, i18n.T(i18n.KeyChatToolInspect), buttonGhost, false).Clicked() {
						childGallery.OnInspectCall(call.CallID)
					}
				}
				if !childGallery.TextOnly {
					var citations []Citation
					for _, source := range call.Events {
						citations = append(citations, CitationsForMessage(source)...)
					}
					CitationsView(c, dedupeCitations(citations, 8), st.Muted, st.Accent, childGallery.OnOpenURL)
				}
			}
			toolCardInfoViewExtra(c, info, st, extra, childGallery)
		})
		for i := len(call.Children) - 1; i >= 0; i-- {
			stack = append(stack, entry{call.Children[i], item.depth + 1})
		}
	}
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
	ui.Column(c).Key("tool-detail:"+key).Gap(3).FillWidth().MinWidth(0).Children(func() {
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

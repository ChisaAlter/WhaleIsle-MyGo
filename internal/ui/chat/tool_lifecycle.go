package chat

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets/markdown"
)

// ToolMessageView draws one complete callId lifecycle and its subcalls. The
// projection and rendering both avoid recursion, including malformed histories.
func ToolMessageView(c *ui.Context, msg session.Message, st ToolCardStyle, gallery ToolImageGalleryOptions) {
	if msg.Tool == nil {
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
		childGallery.ID = "tool:" + call.CallID
		depth := item.depth
		if depth > 6 {
			depth = 6
		}
		ui.Column(c).Padding(0, 0, 4, float32(depth)*14).Gap(4).FillWidth().Children(func() {
			if item.depth > 0 {
				ui.Text(c, "↳ "+call.ParentCallID).FontSize(10).TextColor(st.Muted)
			}
			toolCardInfoView(c, ToolLifecycleCardInfo(call), st, childGallery)
			toolLifecycleDetail(c, "tool-args:"+call.CallID, "参数 / Arguments", prettyToolPayload(call.Arguments), st, childGallery.Expanded)
			if call.Result != nil {
				toolLifecycleDetail(c, "tool-result:"+call.CallID, "结果 / Result", toolLifecycleOutput(call), st, childGallery.Expanded)
			}
			if call.ParentIssue != "" {
				ui.Text(c, "调用关系 / Call relation: "+call.ParentIssue).FontSize(10).TextColor(st.Muted)
			}
			var citations []Citation
			for _, source := range call.Events {
				citations = append(citations, CitationsForMessage(source)...)
			}
			CitationsView(c, dedupeCitations(citations, 8), st.Muted, st.Accent, childGallery.OnOpenURL)
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

func toolLifecycleDetail(c *ui.Context, key, label, body string, st ToolCardStyle, expanded *map[string]bool) {
	if body == "" {
		return
	}
	open := expanded == nil || isThinkingExpanded(expanded, key)
	arrow := "▸"
	if open {
		arrow = "▾"
	}
	ui.Column(c).Gap(3).FillWidth().Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, arrow+" "+label).FontSize(11).Clicked() {
				toggleThinkingExpanded(expanded, key)
			}
			if open && ui.Button(c, "复制 / Copy").FontSize(10).Clicked() {
				mygo.Clipboard.WriteText(body)
			}
		})
		if open {
			ui.Text(c, body).FontSize(11).TextColor(st.Muted)
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
	gallery := call.State == "ok" && len(toolImageReferences(content)) > 0
	parts := []string{toolImageText(content, gallery)}
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

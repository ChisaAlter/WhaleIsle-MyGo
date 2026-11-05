package chat

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"sync"

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
				if gallery.TodoBaselines != nil {
					childGallery.Todo = gallery.TodoBaselines[call.CallID]
				}
				native := func() { toolLifecycleCard(c, call, st, childGallery) }
				if gallery.RenderSlot != nil {
					gallery.RenderSlot(c, "toolview/body", call, native)
				} else {
					native()
				}
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

type toolLifecycleMemoKey struct {
	call        *session.ToolCallView
	name        string
	state       string
	argsLen     int
	hasCall     bool
	hasResult   bool
	resultLen   int
	cwd         string
	home        string
	textOnly    bool
	imageLoader bool
	locale      string
	todo        *todoBaseline
}

var toolLifecycleMemo struct {
	sync.Mutex
	entries map[toolLifecycleMemoKey]ToolCardInfo
}

func cachedToolLifecycleCardInfo(call *session.ToolCallView, gallery ToolImageGalleryOptions) ToolCardInfo {
	resultLen := 0
	if call.Result != nil {
		resultLen = len(call.Result.Raw)
	}
	key := toolLifecycleMemoKey{
		call:        call,
		name:        call.Name,
		state:       call.State,
		argsLen:     len(call.Arguments),
		hasCall:     call.Call != nil,
		hasResult:   call.Result != nil,
		resultLen:   resultLen,
		cwd:         gallery.Cwd,
		home:        gallery.Home,
		textOnly:    gallery.TextOnly,
		imageLoader: toolImageLoaderAvailable(),
		locale:      string(i18n.Locale()),
		todo:        gallery.Todo,
	}
	toolLifecycleMemo.Lock()
	if info, ok := toolLifecycleMemo.entries[key]; ok {
		toolLifecycleMemo.Unlock()
		return info
	}
	toolLifecycleMemo.Unlock()

	origCall := call
	var details *toolDetailsModel
	if !gallery.TextOnly {
		details = toolDetailsModelFor(call)
		// The installed write card diffs against the list preceding the call, which
		// the transcript indexes from the loaded session events.
		if todo := toolDetailsTodoFor(call, gallery.Todo); todo != nil {
			details = todo
		}
		if details != nil {
			// A settled nested receipt owns its name, even if the retained start
			// differs. Keep the store projection immutable while selecting its row.
			result, _ := terminalModelReadResult(call.Result)
			name, _ := toolDetailsModelCallName(call, result.nested)
			displayCall := *call
			displayCall.Name = name
			call = &displayCall
		}
	}
	info := ToolLifecycleCardInfo(call)
	info.details = details
	// The ask-user transcript card replaces the generic IN/OUT body, including
	// the recorded cancelled/interrupted verdicts.
	if !gallery.TextOnly {
		info.ask = askQuestionCardFor(call)
	}
	arguments := call.Arguments
	output := toolLifecycleOutputFor(call, !gallery.TextOnly)
	if gallery.TextOnly {
		info.Images, info.Links = nil, nil
		if call.Result != nil {
			info.Detail = truncateToolText(output, 200)
		}
	}
	info.Arguments = arguments
	if !gallery.TextOnly {
		summary := toolSummaryModelFor(call, gallery.Cwd, gallery.Home)
		if info.details != nil {
			switch {
			case info.details.Summary != "":
				summary.Summary = info.details.Summary
			case len(info.details.Items) > 0:
				summary.Summary = info.details.Items[0].Title
			default:
				summary.Summary = info.details.Empty
			}
		}
		info.summaryModel = &summary
		info.imageFamily = call.Name == "read_image"
		if info.imageFamily {
			info.image = toolImageModelFor(call)
		} else if toolUsesGenericImageCard(call.Name) {
			info.image = toolGenericImageModelFor(call)
		}
		if !key.imageLoader {
			info.image = nil
		}
		// Paired calls use one typed result body; malformed images remain in OUT.
		info.Images = nil
		info.readFamily = call.Name == "read"
		if info.readFamily {
			info.read = toolReadModelFor(call, gallery.Cwd, gallery.Home)
		}
		info.mutationFamily = call.Name == "write" || call.Name == "edit"
		info.diff = toolDiffModelFor(call)
		info.searchFamily = call.Name == "grep" || call.Name == "glob"
		if info.searchFamily {
			info.search = toolSearchModelFor(call)
		}
		info.webFamily = call.Name == "web_search" || call.Name == "web_fetch"
		if info.webFamily {
			info.web = toolWebModelFor(call)
			info.webHref = toolWebFetchHref(call)
		}
		info.diffSummary = toolDiffSummaryFor(call)
		if kilobytes, present := toolContentKilobytes(call.Name, arguments); present {
			info.contentKB = &kilobytes
		}
	}
	info.terminal = terminalModelFor(call, gallery.Cwd)
	if info.terminal != nil || info.diff != nil || info.read != nil {
		info.image = nil
	}
	if info.Status == "ok" && terminalModelFailed(info.terminal) {
		info.Status = "error"
	}
	if call.Result != nil {
		info.Output = output
		// Typed sources/fetch URL belong to the result, not the raw JSON.
		info.Web = call.Result.Web
	}
	_ = origCall
	toolLifecycleMemo.Lock()
	if toolLifecycleMemo.entries == nil || len(toolLifecycleMemo.entries) >= 512 {
		toolLifecycleMemo.entries = make(map[toolLifecycleMemoKey]ToolCardInfo, 64)
	}
	toolLifecycleMemo.entries[key] = info
	toolLifecycleMemo.Unlock()
	return info
}

func toolLifecycleCard(c *ui.Context, call *session.ToolCallView, st ToolCardStyle, gallery ToolImageGalleryOptions) {
	gallery.Call = call
	info := cachedToolLifecycleCardInfo(call, gallery)
	if !gallery.TextOnly {
		if call.CallID != "" && gallery.OnInspectCall != nil {
			info.inspect = func() { gallery.OnInspectCall(call.CallID) }
		}
		if info.webFamily {
			info.openURL = gallery.OnOpenURL
		}
		info.openFile = gallery.OnOpenFile
	}
	extra := func() {
		if !gallery.TextOnly && info.details == nil && !info.readFamily && !info.imageFamily && !info.mutationFamily && call.Name != "str_replace_editor" {
			toolFileView(c, call, st, gallery)
		}
		if call.ParentIssue != "" {
			ui.Text(c, i18n.T(i18n.KeyChatToolCallRelated)+": "+call.ParentIssue).FontSize(10).TextColor(st.Muted)
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
	// Only a chosen result card consumes image blocks. OUT always retains the
	// full evidence when validation or the attachment loader is unavailable.
	body := toolResultBlocksText(content, false)
	if !allowGallery {
		body = readOnlyToolContent(content)
	}
	if len(content) == 0 && event.Data.Error != nil {
		return event.Data.Error.Name + ": " + event.Data.Error.Code
	}
	return body
}

package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/widgets/markdown"
)

// Display-only limits: neither journal records nor cursor/stream state change.
const readOnlyMessageByteLimit = 64 << 10
const readOnlyHistoryByteLimit = 512 << 10

// ReadOnlyHistoryScope prevents disclosure state crossing Hosts, child address
// dimensions, or a new navigation/page. JSON avoids delimiter collisions.
func ReadOnlyHistoryScope(binding string, address session.AgentSessionAddress, token uint64) string {
	raw, _ := json.Marshal(struct {
		Binding string
		Address session.AgentSessionAddress
		Token   uint64
	}{binding, address, token})
	return string(raw)
}

// ReadOnlyHistoryView keeps chat mutation callbacks absent; optional addressed
// resources enable explicit read-only navigation without changing the main session.
func ReadOnlyHistoryView(c *ui.Context, scope string, messages []session.Message, fallback string, st ToolCardStyle, resources ...ReadOnlyResourceOptions) {
	var resource ReadOnlyResourceOptions
	if len(resources) > 0 {
		resource = resources[0]
	}
	rows, limited := readOnlyHistoryMessages(messages)
	limited = limited || (len(rows) == 0 && len(fallback) > readOnlyMessageByteLimit)
	box := ui.Column(c.Key("readonly-history:" + scope)).FillWidth().Gap(8)
	imageRequests := ui.Local(box, "image-requests", func() childImageRequests { return childImageRequests{} })
	expanded := ui.Local(box, "disclosures", func() map[string]bool { return map[string]bool{} })
	// Do not retain disclosure keys for events evicted from the bounded window.
	allowed := map[string]bool{}
	for _, msg := range rows {
		allowed[msg.ID] = true
		if msg.Tool == nil {
			allowed["tool-event:"+msg.ID] = true
			continue
		}
		stack := []*session.ToolCallView{msg.Tool}
		seen := map[*session.ToolCallView]bool{}
		for len(stack) > 0 {
			call := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if call == nil || seen[call] {
				continue
			}
			seen[call] = true
			allowed["tool-args:"+call.CallID] = true
			allowed["tool-result:"+call.CallID] = true
			stack = append(stack, call.Children...)
		}
	}
	for key := range *expanded {
		if !allowed[key] {
			delete(*expanded, key)
		}
	}
	box.Children(func() {
		ui.Text(c, "子会话只读阅读 / Child read-only history").FontSize(10).TextColor(st.Muted)
		if limited {
			ui.Text(c, "展示已截断：最多400行、每行64 KiB、合计512 KiB；原始历史与游标未变。 / Display clipped; history and cursors are unchanged.").FontSize(10).TextColor(st.Muted)
		}
		if len(rows) == 0 {
			ui.Text(c, clipReadOnlyText(fallback, readOnlyMessageByteLimit)).FontSize(12).TextColor(st.FG).Selectable()
			return
		}
		for index, msg := range rows {
			row := ui.Column(c.Key(fmt.Sprintf("message:%d:%s", index, msg.ID))).FillWidth().Gap(4)
			row.Children(func() {
				childAutoImages(c, msg, resource, *imageRequests, st)
				label := msg.Role
				if label == "" {
					label = string(msg.Kind)
				}
				if msg.Streaming {
					label += " · 实时临时输出 / live preview"
				} else {
					label += fmt.Sprintf(" #%.0f", msg.Seq)
				}
				ui.Text(c, label).FontSize(10).TextColor(st.Muted)
				if msg.Kind == session.MessageKindQuestionReply {
					QuestionReplyView(c, msg, scope, st.FG, st.Muted)
					return
				}
				if msg.Kind == session.MessageKindContext {
					ContextInjectionView(c, msg, scope, st.FG, st.Muted)
					return
				}
				if msg.Kind == session.MessageKindCompaction {
					CompactionView(c, msg, st.FG, st.Muted, st.Field)
					return
				}
				if msg.Kind == session.MessageKindTool {
					ToolMessageView(c, msg, st, ToolImageGalleryOptions{ID: msg.ID, Expanded: expanded, TextOnly: true})
					readOnlyToolResources(c, msg, resource, st)
					return
				}
				body := msg.Body
				if msg.Command != nil {
					body = strings.TrimSpace("/" + msg.Command.Name + " · " + msg.Command.State + "\n" + msg.Command.Args + "\n" + msg.Command.Text)
				}
				switch {
				case msg.Kind == session.MessageKindThinking:
					ThinkingBlockView(c, msg.ID, body, expanded, ThinkingStyle{Streaming: msg.Streaming, FG: st.FG, Muted: st.Muted, Border: st.Border, Field: st.Field, Size: secondaryChatSize()})
				case msg.Kind == session.MessageKindText && msg.Role == "assistant":
					markdown.View(c, body, markdown.Style{FontSize: 13, FG: st.FG, Muted: st.Muted, CodeBG: st.Field, Accent: st.Accent, ResolveFile: FileMentionResolver(messages, msg), OnOpenFile: resource.OnOpenFile})
					for _, ref := range readOnlyReferences(body) {
						readOnlyResourceLink(c, ref, resource, st)
					}
				default:
					ui.Text(c, body).FontSize(12).TextColor(st.FG).Selectable()
				}
				for _, path := range msg.Files {
					readOnlyResourceLink(c, path, resource, st)
				}
				for _, ref := range session.ImageRefs(body) {
					readOnlyResourceLink(c, ref, resource, st)
				}
				CitationsView(c, CitationsForMessage(msg), st.Muted, st.Accent, resource.OnOpenURL)
				if body != "" && ui.Button(c, "复制消息 / Copy message").FontSize(10).Clicked() {
					c.Services().WriteClipboard(body)
				}
			})
		}
	})
}

func readOnlyHistoryMessages(messages []session.Message) ([]session.Message, bool) {
	limited := len(messages) > projectflow.ProcessRecordLimit
	if limited {
		messages = messages[len(messages)-projectflow.ProcessRecordLimit:]
	}
	kept := make([]session.Message, 0, len(messages))
	remaining := readOnlyHistoryByteLimit
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		size := readOnlyMessageBytes(msg)
		// Canonical inputs have only leaf preparing tools; lifecycle trees are built
		// below from bounded raw events. Never borrow an externally supplied graph.
		unsupportedTree := msg.Tool != nil && (len(msg.Tool.Children) > 0 || len(msg.Tool.Events) > 0 || msg.Tool.Call != nil || msg.Tool.Result != nil)
		if size > readOnlyMessageByteLimit || unsupportedTree {
			limited = true
			var body strings.Builder
			body.WriteString("[展示截断 / display clipped]\n")
			parts := []string{msg.Body, msg.Args, string(msg.Raw[:min(len(msg.Raw), readOnlyMessageByteLimit)])}
			if msg.Tool != nil {
				parts = append(parts, msg.Tool.Name, msg.Tool.Arguments)
			}
			if msg.Command != nil {
				parts = append(parts, msg.Command.Name, msg.Command.Args, msg.Command.Text)
			}
			parts = append(parts, msg.Files...)
			for _, part := range parts {
				remaining := readOnlyMessageByteLimit - 1024 - body.Len()
				if remaining <= 0 {
					break
				}
				if part != "" {
					body.WriteString(clipReadOnlyText(part, remaining))
					body.WriteByte('\n')
				}
			}
			msg = session.Message{ID: clipReadOnlyText(msg.ID, 256), Role: clipReadOnlyText(msg.Role, 128), Kind: session.MessageKindText, Body: body.String(), Seq: msg.Seq, Streaming: msg.Streaming}
			size = readOnlyMessageBytes(msg)
		}
		if size > remaining {
			limited = true
			break
		}
		remaining -= size
		kept = append(kept, msg)
	}
	for left, right := 0, len(kept)-1; left < right; left, right = left+1, right-1 {
		kept[left], kept[right] = kept[right], kept[left]
	}
	return session.ProjectToolMessages(kept), limited
}

func readOnlyMessageBytes(msg session.Message) int {
	size := len(msg.ID) + len(msg.Role) + len(msg.Kind) + len(msg.EventType) + len(msg.Body) + len(msg.Args) + len(msg.Raw)
	for _, path := range msg.Files {
		size += len(path)
	}
	if msg.Tool != nil {
		size += len(msg.Tool.CallID) + len(msg.Tool.Name) + len(msg.Tool.Arguments) + len(msg.Tool.State)
	}
	if msg.Command != nil {
		size += len(msg.Command.Name) + len(msg.Command.Args) + len(msg.Command.Text) + len(msg.Command.State)
	}
	return size
}
func clipReadOnlyText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	end := limit
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

// Preserve all Markdown destinations, including relative/attachment/file URLs,
// as inert text. Rendering a label alone must not erase the source identity.
func readOnlyReferences(body string) []string {
	document := goldmark.New().Parser().Parse(text.NewReader([]byte(body)))
	refs := []string{}
	seen := map[string]bool{}
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination string
		switch n := node.(type) {
		case *ast.Link:
			destination = string(n.Destination)
		case *ast.Image:
			destination = string(n.Destination)
		case *ast.AutoLink:
			destination = string(n.URL([]byte(body)))
		}
		if destination != "" && !seen[destination] {
			refs = append(refs, destination)
			seen[destination] = true
		}
		return ast.WalkContinue, nil
	})
	return refs
}

// Never discard attachment metadata just because an extension/image block also
// has a text-like field. Only actual text blocks are reduced to their text.
func readOnlyToolContent(content []json.RawMessage) string {
	parts := make([]string, 0, len(content))
	for _, raw := range content {
		var block struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &block) == nil && block.Type == "text" {
			parts = append(parts, toolImageText([]json.RawMessage{raw}, false))
		} else {
			parts = append(parts, string(raw))
		}
	}
	return strings.Join(parts, "\n")
}

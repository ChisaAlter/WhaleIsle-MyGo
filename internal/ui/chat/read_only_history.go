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
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
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
	readOnlyHistoryView(c, scope, messages, fallback, st, len(messages) > projectflow.ProcessRecordLimit, resources...)
}

// ReadOnlyHistoryWithPaging keeps todo before/after receipts honest when an
// addressed child has an earlier journal page outside the current window.
func ReadOnlyHistoryWithPaging(c *ui.Context, scope string, messages []session.Message, fallback string, st ToolCardStyle, hasMore bool, resources ...ReadOnlyResourceOptions) {
	readOnlyHistoryView(c, scope, messages, fallback, st, hasMore, resources...)
}
func readOnlyHistoryView(c *ui.Context, scope string, messages []session.Message, fallback string, st ToolCardStyle, hasMore bool, resources ...ReadOnlyResourceOptions) {
	ReadOnlyHistoryWithPreferences(c, scope, messages, fallback, st, hasMore, settings.ChatPreferences{}, resources...)
}

// ReadOnlyHistoryWithPreferences keeps child turn-tail accounting under the
// same accepted display preference as the main transcript. It grants no chat
// mutation authority; resource callbacks retain their addressed read owner.
func ReadOnlyHistoryWithPreferences(c *ui.Context, scope string, messages []session.Message, fallback string, st ToolCardStyle, hasMore bool, preferences settings.ChatPreferences, resources ...ReadOnlyResourceOptions) {
	var resource ReadOnlyResourceOptions
	if len(resources) > 0 {
		resource = resources[0]
	}
	rows, limited := readOnlyHistoryMessages(messages)
	todo := todoBaselinesFor(rows, hasMore || limited)
	targets := assistantActionTargets(rows)
	model := Model{Snapshot: session.ViewState{SelectedID: scope, PlanHistoryBinding: scope, Messages: rows, TurnUsage: resource.TurnUsage}, Preferences: preferences, SystemDark: c.Theme().Dark}
	box := ui.Column(c.Key("readonly-history:" + scope)).FillWidth().Gap(8)
	imageRequests := ui.Local(box, "image-requests", func() childImageRequests { return childImageRequests{} })
	expanded := ui.Local(box, "disclosures", func() map[string]bool { return map[string]bool{} })
	// Do not retain disclosure keys for events evicted from the bounded window.
	allowed := map[string]bool{}
	for _, msg := range rows {
		allowed[msg.ID] = true
		allowed["tool-images:"+msg.ID] = true
		allowed["thinking:"+msg.ID] = true
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
			allowed["tool-file:"+call.CallID] = true
			allowed["tool-images:"+msg.ID+"|tool:"+call.CallID] = true
			// The tool row's own seat keeps its open state across frames;
			// without it the disclosure would collapse on the next render.
			allowed[msg.ID+"|tool:"+call.CallID] = true
			stack = append(stack, call.Children...)
		}
	}
	for key := range *expanded {
		if !allowed[key] {
			delete(*expanded, key)
		}
	}
	gallery := ui.Local(box, "canonical-gallery", func() childGalleryState { return childGalleryState{} })
	galleryPaths := map[string]session.AgentResourceRead{}
	for _, msg := range rows {
		refs := childDeclaredImages(msg)
		for _, a := range session.UserMessageContent(msg).Attachments {
			if a.Type == "image" {
				refs = append(refs, "attachment:"+a.AttachmentID)
			}
		}
		for _, ref := range refs {
			if path := DisplayImagePath(ref); path != "" {
				galleryPaths[path] = session.AgentResourceRead{Path: path}
			}
		}
	}
	openImage := func(path string) { gallery.Ref = path; gallery.Open = true; gallery.ActualSize = false }
	box.Children(func() {
		if limited {
			ui.Text(c, i18n.T(i18n.KeyChatReadOnlyNote)).FontSize(10).TextColor(st.Muted)
		}
		if len(rows) == 0 {
			ui.Text(c, fallback).FontSize(12).TextColor(st.FG).Selectable()
			return
		}
		for index, msg := range rows {
			row := ui.Column(c.Key(fmt.Sprintf("message:%d:%s", index, msg.ID))).FillWidth().Gap(4)
			row.Children(func() {
				if msg.EventType == "turn/end" {
					ui.Text(c, turnStatusLabel(msg)).FontSize(13).FixedLineHeight(24).TextColor(st.Muted)
					if target, ok := targets[index]; ok {
						assistantMessageActions(c, model, target, row, st.Muted)
					} else {
						TurnUsageView(c, model, msg, st.Muted)
					}
					return
				}
				content := session.UserMessageContent(msg)
				structuredUser := msg.Role == "user" && content.Structured
				if !structuredUser && msg.Kind != session.MessageKindTool && msg.Role != "user" {
					childAutoImages(c, msg, resource, *imageRequests, st)
				}

				if msg.Kind == session.MessageKindQuestionReply {
					QuestionReplyView(c, msg, scope, st.Colors, st.FG, st.Muted)
					return
				}
				if msg.Kind == session.MessageKindContext {
					ContextInjectionView(c, msg, scope, st.Colors)
					return
				}
				if msg.Kind == session.MessageKindSystemPrompt {
					SystemPromptView(c, model, msg, st.FG, st.Muted)
					return
				}
				if msg.Kind == session.MessageKindCompaction {
					CompactionView(c, msg, st.FG, st.Muted, st.Field)
					return
				}
				if msg.Kind == session.MessageKindTool {
					ToolMessageView(c, msg, st, ToolImageGalleryOptions{ID: msg.ID, Expanded: expanded, TodoBaselines: todo, OnOpenFile: resource.OnOpenFile, OnOpenURL: resource.OnOpenURL, OnOpen: openImage})
					return
				}
				if msg.Role == "user" {
					refs, body := userMessageParts(msg.Body)
					userModel := model
					userModel.Snapshot.Messages = messages
					userModel.ThinkingExpanded = expanded
					userModel.OnOpenImageLightbox = openImage
					userModel.OnOpenCitationURL = resource.OnOpenURL
					if resource.OnOpenFile != nil {
						userModel.OnOpenHistoryReference = func(path string, done func(error)) {
							resource.OnOpenFile(path)
							done(nil)
						}
					}
					userMessageView(c, userModel, msg, st.Colors, refs, body, openImage)
					return
				}
				body := msg.Body
				if msg.Command != nil {
					body = strings.TrimSpace("/" + msg.Command.Name + " 路 " + msg.Command.State + "\n" + msg.Command.Args + "\n" + msg.Command.Text)
				}
				switch {
				case msg.Kind == session.MessageKindThinking:
					ThinkingBlockView(c, msg.ID, body, expanded, ThinkingStyle{Streaming: msg.Streaming, FG: st.FG, Muted: st.Muted, Border: st.Border, Field: st.Field, Size: secondaryChatSize()})
				case msg.Kind == session.MessageKindText && msg.Role == "assistant":
					writeStyle := transcriptMarkdownStyle(st.FG, st.Muted, st.Field, st.Accent, theme.CurrentPreferences().Chat.Size, darkSurface(st.Muted))
					writeStyle.ResolveFile = FileMentionResolver(messages, msg)
					writeStyle.OnOpenFile = resource.OnOpenFile
					markdown.View(c, body, writeStyle)
					for _, ref := range readOnlyReferences(body) {
						readOnlyResourceLink(c, ref, resource, st)
					}
				default:
					ui.Text(c, body).FontSize(secondaryChatSize()).TextColor(st.FG).Selectable()
				}
				for _, path := range msg.Files {
					readOnlyResourceLink(c, path, resource, st)
				}
				for _, ref := range session.ImageRefs(body) {
					readOnlyResourceLink(c, ref, resource, st)
				}
				CitationsView(c, CitationsForMessage(msg), st.Muted, st.Accent, resource.OnOpenURL)
				if target, ok := targets[index]; ok {
					assistantMessageActions(c, model, target, row, st.Muted)
				}
			})
		}
	})
	childImageGallery(c, gallery, galleryPaths)
}

func readOnlyHistoryMessages(messages []session.Message) ([]session.Message, bool) {
	limited := len(messages) > projectflow.ProcessRecordLimit
	if limited {
		messages = messages[len(messages)-projectflow.ProcessRecordLimit:]
	}
	// Page the record window, not the contents of an individual message.
	return session.ProjectToolMessages(append([]session.Message(nil), messages...)), limited
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

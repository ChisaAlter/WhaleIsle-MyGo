package coss

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

var _ = registerSheet(func() sheet {
	var specs []specimen
	bubbleHTML := func(align, variant string, grouped bool, text string) string {
		cfg := `@when:variant === "default"`
		if variant != "default" {
			cfg = `@when:!(variant === "default")`
		}
		if !grouped {
			if align == "end" {
				cfg += `@when:!grouped && align === "end"`
			} else {
				cfg += `@when:!grouped && !(align === "end")`
			}
		}
		return `<div data-g data-slot="chat-bubble" class="{{part:chat/chat-bubble/chat-bubble@base` + cfg + `}}"><p class="{{part:chat/chat-message-text/ChatMessageText}}">` + text + `</p></div>`
	}
	for _, side := range []ChatSide{ChatIncoming, ChatOutgoing} {
		for _, grouped := range []bool{false, true} {
			side, grouped := side, grouped
			align, variant := "start", "muted"
			if side == ChatOutgoing {
				align, variant = "end", "default"
			}
			id := "chat/bubble/" + align
			if grouped {
				id += "/grouped"
			}
			specs = append(specs, specimen{id: id, w: 260, h: 64, label: "Ship it today?", html: bubbleHTML(align, variant, grouped, "Ship it today?"),
				build: func(c *ui.Context) ui.Element {
					return ChatBubble(c, ChatBubbleOptions{Side: side, Grouped: grouped}, func() { ChatMessageText(c, "Ship it today?") })
				}})
		}
	}
	specs = append(specs, specimen{id: "chat/date-pill", w: 200, h: 40, label: "Today",
		html: `<div style="width:184px"><div data-g data-slot="chat-date-pill" class="{{part:chat/chat-date-pill/chat-date-pill}}" style="position:relative;top:0">Today</div></div>`,
		build: func(c *ui.Context) ui.Element {
			var e ui.Element
			ui.Column(c).Width(184).Children(func() { e = ChatDatePill(c, "Today") })
			return e
		}})
	specs = append(specs, specimen{id: "chat/unread-divider", w: 300, h: 60, label: "2 unread",
		html: `<div data-g data-slot="chat-unread-divider" class="{{part:chat/chat-unread-divider/chat-unread-divider}}" style="width:284px"><span class="h-px flex-1 bg-input"></span>2 unread<span class="h-px flex-1 bg-input"></span></div>`,
		build: func(c *ui.Context) ui.Element {
			var e ui.Element
			ui.Column(c).Width(284).Children(func() { e = ChatUnreadDivider(c, "2 unread") })
			return e
		}})
	dot := `<span class="block size-1.5 rounded-full bg-muted-foreground" style="opacity:.4;transform:scale(.8)"></span>`
	specs = append(specs, specimen{id: "chat/typing", w: 220, h: 50, label: "Ada is typing",
		html:  `<div data-g role="status" data-slot="chat-typing-indicator" class="{{part:chat/chat-typing-indicator/chat-typing-indicator}}"><div class="flex origin-bottom-left items-center gap-1 rounded-2xl rounded-es-md bg-muted/72 px-3 py-3.5">` + dot + dot + dot + `</div><span class="min-w-0 text-muted-foreground text-xs">Ada is typing</span></div>`,
		build: func(c *ui.Context) ui.Element { return ChatTypingIndicator(c, "Ada is typing") }})
	specs = append(specs, specimen{id: "chat/jump-to-latest", w: 180, h: 44, label: "Jump to latest",
		html:  `<div data-slot="chat-jump-to-latest" class="{{part:chat/chat-jump-to-latest-button/chat-jump-to-latest}}"><button data-g type="button" class="{{cva:button.buttonVariants:variant=outline,size=xs|pointer-events-auto gap-2 rounded-full before:rounded-full}}">` + iconHTML(IconArrowDown) + `Jump to latest</button></div>`,
		build: func(c *ui.Context) ui.Element { return ChatJumpToLatest(c, "Jump to latest") }})
	specs = append(specs, specimen{id: "chat/reply-reference", w: 260, h: 56, label: "Ada",
		html: `<button data-g type="button" data-slot="chat-reply-reference" class="{{part:chat/chat-reply-reference/chat-reply-reference}}">` +
			strings.Replace(iconHTML(IconReply), "<svg", `<svg class="mt-0.5 size-3.5 shrink-0 opacity-64"`, 1) + `<span class="flex min-w-0 flex-col gap-0.5"><span class="font-medium">Ada</span><span class="line-clamp-2 text-foreground/80">Ship it today?</span></span></button>`,
		build: func(c *ui.Context) ui.Element { return ChatReplyReference(c, "Ada", "Ship it today?") }})
	specs = append(specs, specimen{id: "chat/message/incoming", w: 420, h: 110, label: "Ada", state: StateHover,
		build: func(c *ui.Context) ui.Element {
			var e ui.Element
			e = ui.Column(c).Width(400).Children(func() {
				ChatMessage(c, ChatMessageOptions{Side: ChatIncoming, Author: "Ada", Avatar: "Ada Lovelace", Text: "Ship it today?", Footer: "10:24", Actions: true})
			})
			return e
		}})
	specs = append(specs, specimen{id: "chat/message/outgoing-reply", w: 420, h: 130, label: "You",
		build: func(c *ui.Context) ui.Element {
			var e ui.Element
			e = ui.Column(c).Width(400).Children(func() {
				ChatMessage(c, ChatMessageOptions{Side: ChatOutgoing, Author: "You", Text: "Yes, after the gallery passes.", Footer: "10:25", Reply: &ChatReply{Author: "Ada", Text: "Ship it today?"}})
			})
			return e
		}})
	specs = append(specs, specimen{id: "chat/reply-preview", w: 420, h: 110, label: "Replying to Ada",
		build: func(c *ui.Context) ui.Element {
			var e ui.Element
			e = ui.Column(c).Width(400).Padding(12, 0, 0, 0).Children(func() { ChatReplyPreview(c, "Replying to Ada", "Ship it today?", false) })
			return e
		}})
	for _, k := range []ChatStateKind{ChatStateEmpty, ChatStateLoading, ChatStateReconnecting, ChatStateError, ChatStateNoAccess} {
		k := k
		name := map[ChatStateKind]string{ChatStateEmpty: "empty", ChatStateLoading: "loading", ChatStateReconnecting: "reconnecting", ChatStateError: "error", ChatStateNoAccess: "no-access"}[k]
		title := map[ChatStateKind]string{ChatStateEmpty: "No messages yet", ChatStateLoading: "Loading", ChatStateReconnecting: "Reconnecting…", ChatStateError: "Messages could not load.", ChatStateNoAccess: "No access"}[k]
		h := float32(300)
		if k == ChatStateReconnecting || k == ChatStateError {
			h = 80
		}
		specs = append(specs, specimen{id: "chat/state/" + name, w: 420, h: h,
			build: func(c *ui.Context) ui.Element {
				var e ui.Element
				e = ui.Column(c).Width(400).Children(func() {
					var act func()
					if k == ChatStateError {
						act = func() { Button(c, "Retry", ButtonOptions{Variant: ButtonOutline, Size: ButtonXS}) }
					}
					ChatState(c, k, title, "Start the conversation below.", act)
				})
				return e
			}})
	}
	return sheet{comp: "chat", specs: specs}
})

var _ = registerSheet(func() sheet {
	var specs []specimen
	for _, st := range []State{0, StateFocusVisible, StateInvalid, StateDisabled} {
		st := st
		for _, filled := range []bool{false, true} {
			filled := filled
			id := "composer/" + stateName(st)
			if filled {
				id += "/filled"
			}
			specs = append(specs, specimen{id: id, w: 440, h: 130, state: st,
				build: func(c *ui.Context) ui.Element {
					v := ""
					if filled {
						v = "Refactor the gallery harness so each sheet renders\nin one pass."
					}
					var e ui.Element
					e = ui.Column(c).Width(420).Children(func() {
						ev := Composer(c, &v, ComposerOptions{Placeholder: "Ask anything…", Invalid: st.Has(StateInvalid), Disabled: st.Has(StateDisabled),
							Start: func() {
								Button(c, "", ButtonOptions{Variant: ButtonGhost, Size: ButtonIconSM, Icon: IconPaperclip, Label: "Attach"})
								Button(c, "Model", ButtonOptions{Variant: ButtonGhost, Size: ButtonSM, TrailingIcon: IconChevronDown})
							}})
						_ = ev
					})
					return e
				}})
		}
	}
	specs = append(specs, specimen{id: "composer/suggestions", w: 300, h: 130, label: "Ada Lovelace",
		build: func(c *ui.Context) ui.Element {
			return ComposerPanel(c, []ComposerSuggestion{{ID: "ada", Label: "Ada Lovelace", Icon: IconAtSign}, {ID: "alan", Label: "Alan Turing", Icon: IconAtSign}, {ID: "plan", Label: "/plan", Detail: "Draft a plan", Icon: IconSlash}}, 1).Width(280)
		}})
	return sheet{comp: "composer", specs: specs}
})

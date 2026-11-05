package frame

import (
	"log"
	"os"

	"github.com/egoist/mygo/ui"
)

const conversationTabTraceEnv = "WHALEISLE_TRACE_CONVERSATION_TABS"

type conversationTabTraceState struct {
	bounds ui.Rect
	tab    string
	seen   bool
}

// Only these locally constructed controls use this observer. None has an input
// handler or layout observer to replace. HandleInput sets no interactive flags;
// returning false leaves MyGo's existing target and click/drag routing intact.
func observeConversationTabInput(element ui.Element, node string, tab *string) {
	if os.Getenv(conversationTabTraceEnv) != "1" {
		return
	}
	state := ui.Local(element, "conversation-tab-trace", func() conversationTabTraceState { return conversationTabTraceState{} })
	element.ObserveLayout(func(bounds ui.Rect) {
		current := conversationTabTraceValue(tab)
		if !state.seen || state.bounds != bounds || state.tab != current {
			log.Printf("[conversation-tab] layout node=%s x=%.1f y=%.1f w=%.1f h=%.1f tab=%s", node, bounds.X, bounds.Y, bounds.W, bounds.H, current)
		}
		state.bounds, state.tab, state.seen = bounds, current, true
	})
	element.HandleInput(func(event ui.InputEvent) bool {
		kind := ""
		switch event.Kind {
		case ui.InputPointerDown:
			kind = "down"
		case ui.InputPointerUp:
			kind = "up"
		}
		if kind != "" && event.Button == 0 {
			log.Printf("[conversation-tab] input node=%s kind=%s x=%.1f y=%.1f tab=%s", node, kind, state.bounds.X+event.X, state.bounds.Y+event.Y, conversationTabTraceValue(tab))
		}
		return false
	})
}

func conversationTabTraceValue(tab *string) string {
	if tab == nil {
		return "unset"
	}
	switch *tab {
	case "", "chat", "trajectory":
		if *tab == "" {
			return "unset"
		}
		return *tab
	default:
		return "other"
	}
}

func traceConversationTabSelection(before, requested string, after *string) {
	if os.Getenv(conversationTabTraceEnv) == "1" {
		log.Printf("[conversation-tab] selection requested=%s before=%s after=%s", conversationTabTraceValue(&requested), conversationTabTraceValue(&before), conversationTabTraceValue(after))
	}
}

// TraceConversationTabWriteback observes mainView's completed paint-local commit.
// Logs contain only the two fixed view values, never session IDs or content.
func TraceConversationTabWriteback(before, after string) {
	if before != after && os.Getenv(conversationTabTraceEnv) == "1" {
		log.Printf("[conversation-tab] writeback before=%s after=%s", conversationTabTraceValue(&before), conversationTabTraceValue(&after))
	}
}

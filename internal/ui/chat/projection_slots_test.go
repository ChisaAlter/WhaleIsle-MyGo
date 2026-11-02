package chat

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"whaleisle/internal/store/session"
)

func plainTranscript(selected string, bodies ...string) session.ViewState {
	snap := session.ViewState{SelectedID: selected}
	for i, body := range bodies {
		snap.Messages = append(snap.Messages, session.Message{
			ID: fmt.Sprintf("%s-%d", selected, i), Seq: float64(i + 1), Role: "assistant",
			Kind: session.MessageKindText, Body: body, Raw: json.RawMessage(`{}`),
		})
	}
	return snap
}

// A workspace draws the main conversation and a child agent's chat in one pass.
// Each must keep its warm projection while the other is drawn in between.
func TestChatProjectionsDoNotEvictEachOther(t *testing.T) {
	main := plainTranscript("slots-main", "one", "two")
	child := plainTranscript("slots-child", "alpha", "beta", "gamma")

	mainFirst := projectChatMessages(main)
	childFirst := projectChatMessages(child)
	mainAgain := projectChatMessages(main)
	childAgain := projectChatMessages(child)

	if len(mainFirst) != 2 || len(childFirst) != 3 {
		t.Fatalf("wrong projections: main=%d child=%d", len(mainFirst), len(childFirst))
	}
	if &mainFirst[0] != &mainAgain[0] {
		t.Fatal("drawing the child chat rebuilt the main conversation's projection")
	}
	if &childFirst[0] != &childAgain[0] {
		t.Fatal("drawing the main conversation rebuilt the child chat's projection")
	}
}

func TestChatProjectionSlotsAreBoundedAndStayCorrectAfterEviction(t *testing.T) {
	views := make([]session.ViewState, chatProjectionSlots+2)
	for i := range views {
		views[i] = plainTranscript(fmt.Sprintf("bounded-%d", i), fmt.Sprintf("body %d", i))
	}
	for _, v := range views {
		projectChatMessages(v)
	}
	chatProjections.Lock()
	held := len(chatProjections.slots)
	chatProjections.Unlock()
	if held > chatProjectionSlots {
		t.Fatalf("%d slots held, limit %d", held, chatProjectionSlots)
	}
	// The first view was evicted; it must still project correctly, into a fresh slot.
	got := projectChatMessages(views[0])
	if len(got) != 1 || got[0].Body != "body 0" {
		t.Fatalf("evicted transcript projected wrongly: %+v", got)
	}
}

func TestChatProjectionSourceCopyIsBounded(t *testing.T) {
	defer func(limit int) { chatSourceByteLimit = limit }(chatSourceByteLimit)
	chatSourceByteLimit = 64

	big := plainTranscript("bounded-source", strings.Repeat("x", 200))
	first := projectChatMessages(big)
	if len(first) != 1 || first[0].Body != big.Messages[0].Body {
		t.Fatalf("oversized transcript projected wrongly: %+v", first)
	}
	cp := chatProjectionOf(big)
	cp.mu.Lock()
	retained := len(cp.source)
	cp.mu.Unlock()
	if retained != 0 {
		t.Fatalf("a transcript over the limit kept %d source copies", retained)
	}

	// The same transcript within the limit is still memoized.
	chatSourceByteLimit = 1 << 20
	warm := projectChatMessages(big)
	again := projectChatMessages(big)
	if &warm[0] != &again[0] {
		t.Fatal("a transcript within the limit was not memoized")
	}
}

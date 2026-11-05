package frame

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
)

func TestNativePluginSlotsRetainOriginalSubjectContract(t *testing.T) {
	type call struct {
		slot, id string
		seq      float64
	}
	var calls []call
	fallbacks := 0
	m := Model{PluginSlot: func(_ *ui.Context, slot, id string, seq float64, fallback func()) {
		calls = append(calls, call{slot, id, seq})
		if fallback != nil {
			fallback()
		}
	}}
	fallback := func() { fallbacks++ }
	nativeComposerSlot(m)(nil, "composer", fallback)
	nativeComposerSlot(m)(nil, "conversation.composer.input-right", fallback)
	nativeMessageSlot(m)(nil, session.ViewState{}, session.Message{Seq: 17}, "conversation.chat.message.body", fallback)
	nativeReasoningSlot(m)(nil, session.Message{Seq: 0}, fallback)
	nativeToolSlot(m)(nil, "conversation.tool-call.body", &session.ToolCallView{CallID: "exact-call"}, fallback)
	nativeToolSlot(m)(nil, "conversation.tool-call.body", nil, fallback)
	want := []call{{"conversation.composer", "", -1}, {"conversation.composer.input-right", "", -1}, {"conversation.chat.message.body", "", 17}, {"conversation.chat.reasoning.body", "", 0}, {"conversation.tool-call.body", "exact-call", -1}}
	if len(calls) != len(want) || fallbacks != 6 {
		t.Fatalf("calls=%+v fallbacks=%d", calls, fallbacks)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("subject[%d]=%+v want=%+v", i, calls[i], want[i])
		}
	}
}

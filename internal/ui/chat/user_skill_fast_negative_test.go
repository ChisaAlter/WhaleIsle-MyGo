package chat

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"whaleisle/internal/store/session"
)

// Keep the complete pre-guard implementation as a semantic oracle. In
// particular sorting is stable and batch ownership follows Seq, not message ID.
func userMessageSkillsBeforeFastNegative(messages []session.Message, target session.Message) []string {
	if target.Seq <= 0 {
		return nil
	}
	ordered := append([]session.Message(nil), messages...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Seq < ordered[j].Seq })
	var names []string
	seen, contains := map[string]bool{}, false
	for _, message := range ordered {
		if message.Kind == session.MessageKindContext {
			var event struct {
				Data struct {
					Source  json.RawMessage `json:"source"`
					Message *struct {
						Source json.RawMessage `json:"source"`
					} `json:"message"`
				} `json:"data"`
			}
			if json.Unmarshal(message.Raw, &event) == nil {
				raw := event.Data.Source
				if event.Data.Message != nil {
					raw = event.Data.Message.Source
				}
				var source struct{ Kind, Name string }
				if json.Unmarshal(raw, &source) == nil && source.Kind == "skill-invocation" && source.Name != "" && !seen[source.Name] {
					seen[source.Name] = true
					names = append(names, source.Name)
				}
			}
			continue
		}
		if message.Role == "user" && message.Kind == session.MessageKindText {
			contains = contains || message.Seq == target.Seq
			continue
		}
		if contains {
			return names
		}
		names, seen = nil, map[string]bool{}
	}
	if contains {
		return names
	}
	return nil
}

func noContextSkillHistory() ([]session.Message, session.Message) {
	rows := make([]session.Message, 0, 2000)
	for turn := 1; turn <= 400; turn++ {
		base := float64((turn - 1) * 5)
		rows = append(rows,
			session.Message{ID: fmt.Sprintf("user-%d", turn), Role: "user", Kind: session.MessageKindText, Seq: base + 1, Body: fmt.Sprintf("Question %d /review 中文", turn)},
			session.Message{ID: fmt.Sprintf("process-%d", turn), Role: "assistant", EventType: "assistant/thinking", Seq: base + 2},
			session.Message{ID: fmt.Sprintf("delivery-%d", turn), EventType: "deliverables/presented", Seq: base + 3, Files: []string{fmt.Sprintf("artifact-%d.md", turn)}, TurnDelivery: &session.TurnDelivery{Turn: uint64(turn)}},
			session.Message{ID: fmt.Sprintf("final-%d", turn), Role: "assistant", Kind: session.MessageKindText, Seq: base + 4, Body: "Final answer"},
			session.Message{ID: fmt.Sprintf("end-%d", turn), EventType: "turn/end", Seq: base + 5})
	}
	return rows, rows[len(rows)-5]
}

func TestUserMessageSkillsFastNegativeMatchesOriginal(t *testing.T) {
	human := session.Message{ID: "same", Role: "user", Kind: session.MessageKindText, Seq: 10, Body: "/review /build"}
	context := func(seq float64, raw string) session.Message {
		return session.Message{ID: "same", Role: "context", Kind: session.MessageKindContext, Seq: seq, Raw: json.RawMessage(raw)}
	}
	review := context(11, `{"data":{"source":{"kind":"skill-invocation","name":"review"}}}`)
	nested := context(12, `{"data":{"source":{"kind":"skill-invocation","name":"wrong"},"message":{"source":{"kind":"skill-invocation","name":"build"}}}}`)
	boundary := session.Message{ID: "same", Role: "assistant", Kind: session.MessageKindText, Seq: 13}
	fixtures := []struct {
		name   string
		rows   []session.Message
		target session.Message
	}{
		{"empty", nil, human},
		{"plain_text", []session.Message{human, boundary}, human},
		{"literal_skill_source_without_context_kind", []session.Message{human, {Role: "assistant", Seq: 11, Raw: review.Raw}, boundary}, human},
		{"real_context", []session.Message{human, review, nested, boundary}, human},
		{"unsorted", []session.Message{nested, boundary, human, review}, human},
		{"equal_sequence_stable_order", []session.Message{human, context(10, string(review.Raw)), context(10, string(nested.Raw)), boundary}, human},
		{"duplicate_names", []session.Message{human, review, context(12, string(review.Raw)), boundary}, human},
		{"same_id_other_batch", []session.Message{{ID: "same", Role: "user", Kind: session.MessageKindText, Seq: 1}, context(2, string(nested.Raw)), {Role: "assistant", Seq: 3}, human, review, boundary, context(14, string(nested.Raw))}, human},
		{"streaming_context", []session.Message{human, {Kind: session.MessageKindContext, Streaming: true, Seq: 11, Raw: review.Raw}, boundary}, human},
		{"malformed_context", []session.Message{human, context(11, `{"data":`), boundary}, human},
		{"non_skill_context", []session.Message{human, context(11, `{"data":{"source":{"kind":"session-reference","name":"review"}}}`), boundary}, human},
		{"before_target_context", []session.Message{context(9, string(review.Raw)), human, boundary}, human},
		{"missing_target", []session.Message{human, review, boundary}, session.Message{Seq: 99}},
		{"invalid_target", []session.Message{human, review, boundary}, session.Message{Seq: 0}},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			want := userMessageSkillsBeforeFastNegative(fixture.rows, fixture.target)
			if got := userMessageSkillNames(fixture.rows, fixture.target); !reflect.DeepEqual(got, want) {
				t.Fatalf("skills=%#v, original=%#v", got, want)
			}
		})
	}
	rows, target := noContextSkillHistory()
	if got, want := userMessageSkillNames(rows, target), userMessageSkillsBeforeFastNegative(rows, target); !reflect.DeepEqual(got, want) {
		t.Fatalf("2000-row history differs: %#v / %#v", got, want)
	}
	if allocs := testing.AllocsPerRun(25, func() { skillFastNegativeSink = userMessageSkillNames(rows, target) }); allocs != 0 {
		t.Fatalf("no-context history allocates: %.1f", allocs)
	}
}

var skillFastNegativeSink []string

func BenchmarkUserMessageSkillsNoContexts(b *testing.B) {
	rows, target := noContextSkillHistory()
	b.Run("previous", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			skillFastNegativeSink = userMessageSkillsBeforeFastNegative(rows, target)
		}
	})
	b.Run("fast_negative", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			skillFastNegativeSink = userMessageSkillNames(rows, target)
		}
	})
}

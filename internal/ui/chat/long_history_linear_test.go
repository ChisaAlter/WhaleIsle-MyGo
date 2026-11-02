package chat

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
)

var linearHistorySequence atomic.Uint64

func linearHistoryMessage(id string, kind session.MessageKind, role, body string, turn *int, step int) session.Message {
	coordinates := "null"
	if turn != nil {
		coordinates = fmt.Sprint(*turn)
	}
	return session.Message{ID: id, Seq: float64(linearHistorySequence.Add(1)), Kind: kind, Role: role, Body: body, Raw: json.RawMessage(fmt.Sprintf(`{"data":{"turn":%s,"step":%d}}`, coordinates, step))}
}

func TestTranscriptLinearIndexRetainsFinalAndOrderedMembers(t *testing.T) {
	for _, mode := range []string{"", "standard", "detailed", "verbose"} {
		for _, toolStep := range []int{1, 2} {
			name := fmt.Sprintf("mode=%s/toolStep=%d", mode, toolStep)
			t.Run(name, func(t *testing.T) {
				zero, one := 0, 1
				messages := []session.Message{
					linearHistoryMessage("unknown", session.MessageKindText, "user", "Unknown location", nil, 0),
					linearHistoryMessage("", session.MessageKindThinking, "thinking", "first", &zero, 0),
					linearHistoryMessage("duplicate", session.MessageKindThinking, "thinking", "other turn", &one, 0),
					linearHistoryMessage("duplicate", session.MessageKindThinking, "thinking", "second", &zero, 0),
					linearHistoryMessage("answer", session.MessageKindText, "assistant", "Answer", &zero, 1),
					linearHistoryMessage("tool", session.MessageKindTool, "tool", "read", &zero, toolStep),
					linearHistoryMessage("end", session.MessageKindText, "status", "", &zero, 1),
				}
				messages[len(messages)-1].EventType = "turn/end"
				m := Model{Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h"}, Preferences: settings.ChatPreferences{TranscriptView: mode}}
				out, controls := projectTranscript(messages, m)
				if out[0].ID != "unknown" {
					t.Fatal("unknown coordinates became zero-turn membership")
				}
				control, folded := controls["transcript-turn:h:s:0"]
				if mode == "verbose" {
					if !reflect.DeepEqual(out, messages) || folded {
						t.Fatal("verbose presentation folded or reordered recorded occurrences")
					}
					return
				}
				wantMembers := []string{"", "duplicate", "tool"}
				if toolStep == 1 {
					wantMembers = []string{"", "duplicate", "answer", "tool"}
				}
				if !folded || !reflect.DeepEqual(control.members, wantMembers) {
					t.Fatalf("ordered member occurrences=%q want %q", control.members, wantMembers)
				}
				answerVisible := false
				for _, row := range out {
					answerVisible = answerVisible || row.ID == "answer"
				}
				if answerVisible != (toolStep != 1) {
					t.Fatal("tool on final answer's step did not invalidate only that final candidate")
				}
				choices := map[string]bool{control.key: true}
				m.ThinkingExpanded = &choices
				opened, openedControls := projectTranscript(messages, m)
				if !openedControls[control.key].open || !reflect.DeepEqual(openedControls[control.key].members, wantMembers) {
					t.Fatal("opening process control changed its stable key or member order")
				}
				if len(opened) <= len(out) {
					t.Fatal("expanded control did not reveal process rows")
				}
			})
		}
	}
}

func TestTranscriptLinearIndexPreservesFallbackAndUnfinishedTurns(t *testing.T) {
	one := 1
	for _, reason := range []string{"failed", "cancelled", "max-tokens", "completed"} {
		for _, interveningUser := range []bool{false, true} {
			messages := []session.Message{linearHistoryMessage("process", session.MessageKindThinking, "thinking", "Reasoning", nil, 0)}
			messages[0].TurnSeq = 42
			if interveningUser {
				messages = append(messages, linearHistoryMessage("steer", session.MessageKindText, "user", "Steer", &one, 0))
			}
			answer := linearHistoryMessage("answer", session.MessageKindText, "assistant", "Final", &one, 1)
			answer.TurnSeq = 42
			end := linearHistoryMessage("end", session.MessageKindText, "status", "", &one, 1)
			end.EventType = "turn/end"
			end.Raw = json.RawMessage(fmt.Sprintf(`{"data":{"turn":1,"step":1,"reason":{"kind":%q}}}`, reason))
			messages = append(messages, answer, end)
			m := Model{Snapshot: session.ViewState{SelectedID: "s", PlanHistoryBinding: "h"}}
			_, controls := projectTranscript(messages, m)
			_, whole := controls["transcript-turn:h:s:1"]
			if whole != (reason == "completed" && !interveningUser) {
				t.Fatalf("reason=%s middleUser=%v whole-turn folded=%v", reason, interveningUser, whole)
			}
			if !whole {
				key := fmt.Sprintf("transcript-group:h:s:process:%.0f", messages[0].Seq)
				if group, ok := controls[key]; !ok || group.running {
					t.Fatal("known TurnSeq fallback was treated as an unfinished/unknown turn")
				}
			}
		}
	}
}

func TestAssistantActionLinearPrefixRetainsCrossTurnOccurrences(t *testing.T) {
	text := func(id, source, role, body string) session.Message {
		return session.Message{ID: id, SourceMessageID: source, Kind: session.MessageKindText, Role: role, Body: body}
	}
	end := func(seq float64) session.Message { return session.Message{ID: "end", EventType: "turn/end", Seq: seq} }
	messages := []session.Message{
		text("shared", "", "user", "prefix"),
		text("first", "shared", "assistant", "A"),
		text("first/block", "shared", "assistant", "B"),
		end(4),
		text("shared", "", "user", "later-user"),
		text("second", "shared", "assistant", "C"),
		end(7),
		text("legacy", "", "assistant", "legacy-only"),
		text("last", "", "user", "later input"),
	}
	targets := assistantActionTargets(messages)
	for seat, want := range map[int]string{3: "prefixAB", 6: "prefixABlater-userC", 7: "legacy-only"} {
		if targets[seat].message.Body != want || targets[seat].latest {
			t.Fatalf("seat=%d got body=%q latest=%v want %q historical", seat, targets[seat].message.Body, targets[seat].latest, want)
		}
	}
	if targets[3].endSeq != 4 || targets[6].endSeq != 7 || targets[7].endSeq != 0 {
		t.Fatal("durable turn/end and legacy user-flush seats changed")
	}
	// A user with the same source is encountered before flush returns, but the
	// previous candidate must retain only the prefix ending at its own index.
	messages = []session.Message{text("a", "same", "assistant", "A"), text("u", "same", "user", "U"), text("b", "same", "assistant", "B")}
	targets = assistantActionTargets(messages)
	if targets[0].message.Body != "A" || targets[2].message.Body != "AUB" || !targets[2].latest {
		t.Fatalf("candidate prefix snapshots changed: %+v", targets)
	}
	if targets[0].message.Body != "A" { // Later Builder writes must not mutate an earlier string.
		t.Fatal("later source text mutated an already-published action")
	}
}

func TestAssistantActionLinearPrefixSkipsNonCopyableBlocks(t *testing.T) {
	messages := []session.Message{
		{ID: "", Role: "assistant", Kind: session.MessageKindText, Body: "legacy-empty-ID"},
		{ID: "end", EventType: "turn/end", Seq: 2},
		{ID: "source", Role: "user", Kind: session.MessageKindText, Body: "user"},
		{ID: "space", SourceMessageID: "source", Role: "assistant", Kind: session.MessageKindText, Body: "  "},
		{ID: "stream", SourceMessageID: "source", Role: "assistant", Kind: session.MessageKindText, Body: "stream", Streaming: true},
		{ID: "tool", SourceMessageID: "source", Role: "tool", Kind: session.MessageKindTool, Body: "tool"},
		{ID: "final", SourceMessageID: "source", Role: "assistant", Kind: session.MessageKindText, Body: "answer"},
		{ID: "end", EventType: "turn/end", Seq: 8},
	}
	targets := assistantActionTargets(messages)
	if len(targets) != 2 || targets[1].message.Body != "legacy-empty-ID" || targets[7].message.Body != "useranswer" || !targets[7].latest {
		t.Fatalf("copyability/source rules changed: %+v", targets)
	}
	// A long single lifecycle should concatenate in input order without a
	// repeated prefix scan or repeated whole-string concatenation.
	messages = messages[:0]
	for i := 0; i < 4096; i++ {
		messages = append(messages, session.Message{ID: fmt.Sprint(i), SourceMessageID: "one", Role: "assistant", Kind: session.MessageKindText, Body: "x"})
	}
	targets = assistantActionTargets(messages)
	if got := targets[len(messages)-1].message.Body; got != strings.Repeat("x", len(messages)) {
		t.Fatal("large projected lifecycle lost ordered blocks")
	}
}

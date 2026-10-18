package session

import (
	"encoding/json"
	"testing"
)

func compactFixture(t *testing.T, events ...string) []Message {
	t.Helper()
	var messages []Message
	for _, raw := range events {
		message, ok := classifyPersistenceEvent(json.RawMessage(raw))
		if !ok {
			t.Fatalf("fixture was not retained: %s", raw)
		}
		messages = append(messages, message)
	}
	return projectCommandMessages(messages)
}

const compactRun = `{"type":"command/run","seq":10,"data":{"commandId":"cmd","name":"compact"}}`
const compactStart = `{"type":"compaction/start","seq":11,"data":{"compactionId":"txn","sourceCommandId":"cmd","turn":null}}`
const compactSummary = `{"type":"compaction/summary","seq":12,"data":{"compactionId":"txn","sourceCommandId":"cmd","summary":[{"type":"text","text":"Full "},{"type":"text","text":"**summary**"}],"shadowedSeqs":[1,2],"shadowedTokenCount":42}}`
const compactCheckpoint = `{"type":"user/message","seq":13,"time":100,"surfaceOp":{"op":"replace","startSeq":1,"endSeq":2},"data":{"source":{"kind":"compact-checkpoint","compactionId":"txn","sourceCommandId":"cmd"},"content":[{"type":"text","text":"model-facing summary"}]}}`
const compactEnd = `{"type":"compaction/end","seq":14,"data":{"compactionId":"txn","sourceCommandId":"cmd","turn":null}}`
const compactDone = `{"type":"command/done","seq":15,"data":{"commandId":"cmd","kind":"success","text":"Compacted 2 items.","sourceEventSeq":12}}`

func TestManualCompactionUsesOneCheckpointCard(t *testing.T) {
	rows := compactFixture(t,
		`{"type":"user/message","seq":1,"data":{"source":{"kind":"user"},"content":[{"type":"text","text":"keep this history visible"}]}}`,
		compactRun, compactStart, compactSummary, compactCheckpoint, compactEnd, compactDone)
	if len(rows) != 2 || rows[0].Body != "keep this history visible" {
		t.Fatalf("duplicated cards or erased shadowed transcript: %+v", rows)
	}
	card := rows[1]
	if card.Kind != MessageKindCommand || card.Seq != 13 || card.ID != "command:cmd" || card.Role != "command" {
		t.Fatalf("wrong checkpoint marker: %+v", card)
	}
	command := card.Command
	if command == nil || command.State != "success" || command.Text != "Compacted 2 items." || command.Compaction == nil {
		t.Fatalf("lost lifecycle settlement: %+v", command)
	}
	c := command.Compaction
	if c.Summary != "Full **summary**" || c.SummaryEventSeq == nil || *c.SummaryEventSeq != 12 ||
		c.ShadowedItemCount == nil || *c.ShadowedItemCount != 2 || c.ShadowedTokenCount == nil || *c.ShadowedTokenCount != 42 {
		t.Fatalf("lost structured compaction evidence: %+v", c)
	}
}

func TestManualCompactionPartialHistoryKeepsIdentityAndOutput(t *testing.T) {
	for _, events := range [][]string{
		{compactCheckpoint},
		{compactCheckpoint, compactDone},
		{compactSummary, compactCheckpoint, compactDone},
	} {
		rows := compactFixture(t, events...)
		if len(rows) != 1 || rows[0].ID != "command:cmd" || rows[0].Command.Compaction == nil {
			t.Fatalf("partial page lost its marker: %+v", rows)
		}
		if len(events) == 1 && rows[0].Command.State != "running" {
			t.Fatal("checkpoint invented a command settlement")
		}
		if len(events) == 2 && (rows[0].Command.Compaction.Summary != "" || rows[0].Command.Text != "Compacted 2 items.") {
			t.Fatal("missing summary borrowed model-facing content or dropped fallback output")
		}
	}
}

func TestManualCompactionNoCheckpointKeepsFullGenericOutcome(t *testing.T) {
	for _, done := range []string{
		`{"type":"command/done","seq":15,"data":{"commandId":"cmd","kind":"error","text":"Cancelled.\nNothing was replaced."}}`,
		`{"type":"command/done","seq":15,"data":{"commandId":"cmd","kind":"success","text":"No compactable history yet."}}`,
	} {
		rows := compactFixture(t, compactRun, compactStart, compactSummary, compactEnd, done)
		if len(rows) != 1 || rows[0].Command.Compaction != nil || rows[0].Command.Text == "" {
			t.Fatalf("summary generation was mistaken for a landed checkpoint: %+v", rows)
		}
	}
	failure := `{"type":"command/done","seq":15,"data":{"commandId":"cmd","kind":"error","text":"Compaction finished, but the session could not be saved."}}`
	rows := compactFixture(t, compactRun, compactSummary, compactCheckpoint, failure)
	if len(rows) != 1 || rows[0].Command.Compaction == nil || rows[0].Command.State != "error" ||
		rows[0].Command.Text != "Compaction finished, but the session could not be saved." {
		t.Fatal("a landed checkpoint swallowed its failed-save result")
	}
}

func TestManualCompactionSourceReferenceCannotStealAnotherSummary(t *testing.T) {
	otherSummary := `{"type":"compaction/summary","seq":20,"data":{"compactionId":"other","sourceCommandId":"other-cmd","summary":[{"type":"text","text":"wrong summary"}],"shadowedSeqs":[1],"shadowedTokenCount":999}}`
	wrongDone := `{"type":"command/done","seq":21,"data":{"commandId":"cmd","kind":"success","text":"Original command output","sourceEventSeq":20}}`
	rows := compactFixture(t, compactRun, compactSummary, compactCheckpoint, otherSummary, wrongDone)
	if len(rows) != 1 || rows[0].Command.Compaction.Summary != "" || rows[0].Command.Text != "Original command output" {
		t.Fatalf("sourceEventSeq crossed command identity: %+v", rows)
	}
	// A checkpoint-looking ordinary user append cannot claim command output.
	appendOnly := `{"type":"user/message","seq":16,"surfaceOp":{"op":"append"},"data":{"source":{"kind":"compact-checkpoint","compactionId":"txn","sourceCommandId":"cmd"},"content":[{"type":"text","text":"ordinary append"}]}}`
	rows = compactFixture(t, compactRun, appendOnly, compactDone)
	if len(rows) != 2 || rows[0].Command.Compaction != nil {
		t.Fatalf("non-replacement promoted to manual compaction: %+v", rows)
	}
}

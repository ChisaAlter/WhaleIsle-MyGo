package session

import (
	"strings"
	"testing"
)

func automaticEvent(s string) string { return strings.ReplaceAll(s, `,"sourceCommandId":"cmd"`, "") }

func TestAutomaticCompactionCheckpointProjection(t *testing.T) {
	rows := compactFixture(t, automaticEvent(compactStart), automaticEvent(compactSummary), automaticEvent(compactCheckpoint), automaticEvent(compactEnd))
	if len(rows) != 1 || rows[0].Kind != MessageKindCompaction || rows[0].Body != "Full **summary**" || rows[0].Seq != 13 {
		t.Fatalf("expected one complete automatic checkpoint marker: %+v", rows)
	}
}

func TestAutomaticCompactionNeedsCheckpointAndDoesNotBorrowManual(t *testing.T) {
	if rows := compactFixture(t, automaticEvent(compactStart), automaticEvent(compactSummary), automaticEvent(compactEnd)); len(rows) != 0 {
		t.Fatalf("uncommitted lifecycle shown as cards: %+v", rows)
	}
	rows := compactFixture(t, compactSummary, automaticEvent(compactCheckpoint))
	if len(rows) != 1 || rows[0].Body != "" || rows[0].Kind != MessageKindCompaction {
		t.Fatalf("manual summary borrowed by automatic checkpoint: %+v", rows)
	}
}

func TestAutomaticCompactionKeepsFullSummary(t *testing.T) {
	full := strings.Repeat("long summary ", 1000)
	summary := strings.Replace(automaticEvent(compactSummary), "Full ", full, 1)
	rows := compactFixture(t, summary, automaticEvent(compactCheckpoint))
	if len(rows) != 1 || rows[0].Body != full+"**summary**" {
		t.Fatal("summary truncated")
	}
}

func TestAutomaticCompactionCountsIdentityAndPartialHistory(t *testing.T) {
	for _, prefix := range []string{"", automaticEvent(compactSummary)} {
		events := []string{automaticEvent(compactCheckpoint)}
		if prefix != "" {
			events = append([]string{prefix}, events...)
		}
		rows := compactFixture(t, events...)
		c := rows[0].AutomaticCompaction
		if c == nil || c.CompactionID != "txn" {
			t.Fatal("checkpoint identity lost")
		}
		if prefix == "" {
			if c.Summary != "" || c.ShadowedItemCount != nil || c.SummaryEventSeq != nil {
				t.Fatal("invented missing summary")
			}
		} else if c.ShadowedItemCount == nil || *c.ShadowedItemCount != 2 || c.ShadowedTokenCount == nil || *c.ShadowedTokenCount != 42 {
			t.Fatal("lost counts")
		}
	}
	unrelated := strings.ReplaceAll(automaticEvent(compactSummary), `"txn"`, `"other"`)
	rows := compactFixture(t, unrelated, automaticEvent(compactCheckpoint))
	if len(rows) != 1 || rows[0].Body != "" {
		t.Fatal("borrowed different compaction identity")
	}
	emptyCommand := strings.ReplaceAll(compactCheckpoint, `"sourceCommandId":"cmd"`, `"sourceCommandId":""`)
	rows = compactFixture(t, automaticEvent(compactSummary), emptyCommand)
	if len(rows) != 1 || rows[0].AutomaticCompaction == nil {
		t.Fatal("empty command identity not automatic")
	}
	appendOnly := strings.Replace(automaticEvent(compactCheckpoint), `"op":"replace"`, `"op":"append"`, 1)
	rows = compactFixture(t, appendOnly)
	if len(rows) != 1 || rows[0].Kind != MessageKindContext {
		t.Fatal("append context claimed landed checkpoint")
	}
}

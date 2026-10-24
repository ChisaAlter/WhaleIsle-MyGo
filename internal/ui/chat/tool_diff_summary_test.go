package chat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToolDiffSummaryCounts(t *testing.T) {
	for _, tc := range []struct {
		old, next      string
		added, removed int
	}{
		{"", "", 0, 0}, {"", "line\n", 1, 0}, {"line\n", "", 0, 1}, {"line", "line\n", 0, 0},
		{"same\nold\nend\n", "same\nnew\nend\n", 1, 1}, {"\n", "\n\n", 1, 0},
		{"a\r\nb\r\n", "a\nb\n", 2, 2}, {"A\nB\nA", "A\nA\nB", 1, 1},
		{"shared\n" + strings.Repeat("old\n", 128), "shared\n" + strings.Repeat("new\n", 128), 128, 128},
		{"shared\n" + strings.Repeat("old\n", 129), "shared\n" + strings.Repeat("new\n", 129), 130, 130},
		{"", strings.Repeat("new\n", 257), 257, 0},
	} {
		added, removed := toolDiffCounts(tc.old, tc.next)
		if added != tc.added || removed != tc.removed {
			t.Errorf("diff lengths%d/%d: +%d -%d want+%d -%d", len(tc.old), len(tc.next), added, removed, tc.added, tc.removed)
		}
	}
	// Compare small ambiguous/repeated-line inputs against an independent LCS
	// oracle so equal-prefix snakes and unequal lengths cannot corrupt counts.
	words := []string{"", "a", "b", "a\na", "a\nb", "b\na", "a\nb\na", "b\na\nb", "a\n\nb"}
	for _, old := range words {
		for _, next := range words {
			split := func(s string) []string {
				if s == "" {
					return nil
				}
				return strings.Split(s, "\n")
			}
			a, b := split(old), split(next)
			rows := make([][]int, len(a)+1)
			for i := range rows {
				rows[i] = make([]int, len(b)+1)
			}
			for i := 1; i <= len(a); i++ {
				for j := 1; j <= len(b); j++ {
					if a[i-1] == b[j-1] {
						rows[i][j] = rows[i-1][j-1] + 1
					} else {
						rows[i][j] = max(rows[i-1][j], rows[i][j-1])
					}
				}
			}
			lcs := rows[len(a)][len(b)]
			added, removed := toolDiffCounts(old, next)
			if added != len(b)-lcs || removed != len(a)-lcs {
				t.Fatalf("LCS oracle %q/%q got+%d -%d", old, next, added, removed)
			}
		}
	}
}

func TestToolDiffSummaryModelSelection(t *testing.T) {
	call := fileToolFixture("edit", `{"file_path":"a","old_string":"old","new_string":"new"}`, map[string]any{"diffs": []any{map[string]any{"path": "a", "oldText": "actual\nshared", "newText": "new\nnewer\nshared"}, map[string]any{"path": "b", "oldText": nil, "newText": "created"}}}, "done")
	before := append([]byte(nil), call.Result.Raw...)
	got := toolDiffSummaryFor(call)
	if got == nil || got.Added != 3 || got.Removed != 1 {
		t.Fatalf("applied totals=%+v", got)
	}
	if string(call.Result.Raw) != string(before) {
		t.Fatal("source result changed")
	}
	call.Result = nil
	call.State = "running"
	if got = toolDiffSummaryFor(call); got == nil || got.Added != 1 || got.Removed != 1 {
		t.Fatalf("intended totals=%+v", got)
	}
	for _, state := range []string{"preparing", "error", "stopped"} {
		call.State = state
		if toolDiffSummaryFor(call) != nil {
			t.Errorf("%s has stats", state)
		}
	}
	call.State = "running"
	call.ParentCallID = "outer"
	if toolDiffSummaryFor(call) != nil {
		t.Fatal("nested has diff stats")
	}
	for _, tc := range []struct {
		name, args string
		meta       any
		present    bool
		added      int
	}{
		{"write", `{"file_path":"a","content":"one\ntwo\n"}`, map[string]any{"diffs": []any{}}, true, 2},
		{"write", `{"file_path":"a","content":"one"}`, nil, true, 1},
		{"write", `{"file_path":"a","content":"one"}`, map[string]any{"diffs": []any{map[string]any{"path": "", "oldText": nil, "newText": "a\nb"}}}, true, 2},
		{"edit", `{"file_path":"a","old_string":"old","new_string":"new"}`, nil, false, 0},
		{"edit", `{"file_path":"a","old_string":"old","new_string":"new"}`, map[string]any{"diffs": []any{map[string]any{"path": "a", "newText": "new"}}}, false, 0},
		{"str_replace_editor", `{"command":"create","path":"a"}`, nil, false, 0},
		{"Read", `{"file_path":"a","content":"one"}`, nil, false, 0},
	} {
		call = fileToolFixture(tc.name, tc.args, tc.meta, "done")
		got = toolDiffSummaryFor(call)
		if (got != nil) != tc.present || got != nil && got.Added != tc.added {
			t.Errorf("%s %s stats=%+v", tc.name, tc.args, got)
		}
	}
}

func TestToolDiffSummaryDecodedContentSize(t *testing.T) {
	for _, tc := range []struct {
		args    string
		kb      int
		present bool
	}{
		{`{}`, 0, false}, {`{"content":""}`, 0, true}, {`{"content":null}`, 0, true},
		{`{"old_string":"a","new_string":"b"}`, 1, true},
	} {
		kb, present := toolContentKilobytes("edit", tc.args)
		if kb != tc.kb || present != tc.present {
			t.Errorf("%s =>%d/%v", tc.args, kb, present)
		}
	}
	for _, tc := range []struct {
		content string
		kb      int
	}{{strings.Repeat("字", 1024), 1}, {strings.Repeat("🙂", 512), 1}, {strings.Repeat("🙂", 513), 2}, {strings.Repeat("x", 1025), 2}} {
		args, _ := json.Marshal(map[string]any{"content": tc.content})
		kb, present := toolContentKilobytes("write", string(args))
		if !present || kb != tc.kb {
			t.Errorf("UTF16 count=%d/%v want%d", kb, present, tc.kb)
		}
	}
	if _, ok := toolContentKilobytes("str_replace_editor", `{"content":"x"}`); ok {
		t.Fatal("legacy row acquired mutation suffix")
	}
}

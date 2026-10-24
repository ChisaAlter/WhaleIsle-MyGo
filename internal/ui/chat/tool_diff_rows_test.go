package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// These rows/copy strings and the digests below were independently recorded by
// extracting QL..rT, Bw/QZ/eV/L8/tV from the CURRENT installed web bundle and
// evaluating only that source in a Node VM. No reference reimplementation,
// browser, installed runtime dependency or application is needed by these tests.
// apps/web/dist/assets/index-wKUmVWbt.js SHA256:
// 328188be7381f20aac74717a49c536adda26b52e12f244a36bc34507429d4398.
func TestToolDiffRowsInstalledGoldens(t *testing.T) {
	cases := []struct {
		id, old, next string
		rows          []toolDiffRow
		copy          string
	}{
		{"repeat-tie", "A\nB\nA", "A\nA\nB", []toolDiffRow{{"path", "f"}, {"context", "A"}, {"del", "B"}, {"context", "A"}, {"add", "B"}}, "f\n  A\n- B\n  A\n+ B"},
		{"reorder", "a\nb\nc\n", "b\na\nc\n", []toolDiffRow{{"path", "f"}, {"del", "a"}, {"context", "b"}, {"add", "a"}, {"context", "c"}}, "f\n- a\n  b\n+ a\n  c"},
		{"greedy", "a\na\nb\na", "b\na\nb\nb", []toolDiffRow{{"path", "f"}, {"del", "a"}, {"del", "a"}, {"context", "b"}, {"context", "a"}, {"add", "b"}, {"add", "b"}}, "f\n- a\n- a\n  b\n  a\n+ b\n+ b"},
		{"ambiguous-islands", "a\nb\na\nb\na", "b\na\nb\na\nb", []toolDiffRow{{"path", "f"}, {"del", "a"}, {"context", "b"}, {"context", "a"}, {"context", "b"}, {"context", "a"}, {"add", "b"}}, "f\n- a\n  b\n  a\n  b\n  a\n+ b"},
		{"CR-blanks-tab-nonBMP", "\t🙂\r\n\nold\r\n", "\t🙂\r\n\n新𐐷\r\n", []toolDiffRow{{"path", "f"}, {"context", "\t🙂\r"}, {"context", ""}, {"del", "old\r"}, {"add", "新𐐷\r"}}, "f\n  \t🙂\r\n  \n- old\r\n+ 新𐐷\r"},
		{"empty", "", "", []toolDiffRow{{"path", "f"}}, "f"},
		{"oneblank", "", "\n", []toolDiffRow{{"path", "f"}, {"add", ""}}, "f\n+ "},
		{"twofinal", "a\n", "a\n\n", []toolDiffRow{{"path", "f"}, {"context", "a"}, {"add", ""}}, "f\n  a\n+ "},
		{"terminator", "a\nb", "a\nb\n", []toolDiffRow{{"path", "f"}}, "f"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			diffs := []toolFileDiff{{Path: "f", OldText: &tc.old, NewText: tc.next}}
			rows := toolDiffRows(diffs)
			if !reflect.DeepEqual(rows, tc.rows) {
				t.Fatalf("rows = %#v, want %#v", rows, tc.rows)
			}
			if got := toolDiffCopyText(rows); got != tc.copy {
				t.Fatalf("copy = %q, want %q", got, tc.copy)
			}
		})
	}
}

func TestToolDiffRowsInstalledContextGaps(t *testing.T) {
	for _, count := range []int{6, 7, 8} {
		middle := make([]string, count)
		for i := range middle {
			middle[i] = fmt.Sprintf("c%d", i)
		}
		old := "start\nold-a\n" + strings.Join(middle, "\n") + "\nold-b\nend"
		next := "start\nnew-a\n" + strings.Join(middle, "\n") + "\nnew-b\nend"
		want := []toolDiffRow{{"path", "gaps"}, {"context", "start"}, {"del", "old-a"}, {"add", "new-a"}}
		if count == 6 {
			want = append(want, []toolDiffRow{{"context", "c0"}, {"context", "c1"}, {"context", "c2"}, {"context", "c3"}, {"context", "c4"}, {"context", "c5"}}...)
		} else {
			want = append(want, []toolDiffRow{{"context", "c0"}, {"context", "c1"}, {"context", "c2"}, {"gap", "⋯"}}...)
			for _, line := range middle[len(middle)-3:] {
				want = append(want, toolDiffRow{"context", line})
			}
		}
		want = append(want, []toolDiffRow{{"del", "old-b"}, {"add", "new-b"}, {"context", "end"}}...)
		if got := toolDiffRows([]toolFileDiff{{Path: "gaps", OldText: &old, NewText: next}}); !reflect.DeepEqual(got, want) {
			t.Errorf("%d shared lines: got %#v, want %#v", count, got, want)
		}
	}
	old := "before0\nbefore1\nbefore2\nbefore3\nbefore4\nold\nafter0\nafter1\nafter2\nafter3\nafter4"
	next := strings.Replace(old, "old", "new", 1)
	want := []toolDiffRow{{"path", "trim"}, {"context", "before2"}, {"context", "before3"}, {"context", "before4"}, {"del", "old"}, {"add", "new"}, {"context", "after0"}, {"context", "after1"}, {"context", "after2"}}
	if got := toolDiffRows([]toolFileDiff{{Path: "trim", OldText: &old, NewText: next}}); !reflect.DeepEqual(got, want) {
		t.Errorf("three lines of edge context = %#v, want %#v", got, want)
	}
}

func TestToolDiffRowsInstalledPathFragments(t *testing.T) {
	diffs := []toolFileDiff{
		{Path: "a", OldText: toolDiffModelStringForTest("same"), NewText: "same\n"},
		{Path: "a", NewText: "one"},
		{Path: "b", OldText: toolDiffModelStringForTest("x")},
		{Path: "a", OldText: toolDiffModelStringForTest("")},
	}
	want := []toolDiffRow{{"path", "a"}, {"gap", "⋯"}, {"add", "one"}, {"path", "b"}, {"del", "x"}, {"path", "a"}}
	rows := toolDiffRows(diffs)
	if !reflect.DeepEqual(rows, want) || toolDiffCopyText(rows) != "a\n⋯\n+ one\nb\n- x\na" {
		t.Fatalf("fragment rows/copy = %#v / %q", rows, toolDiffCopyText(rows))
	}
	if got := toolDiffRows(nil); len(got) != 0 || toolDiffCopyText(got) != "" {
		t.Fatalf("empty diffs yielded %#v", got)
	}
	rows = toolDiffRows([]toolFileDiff{{}, {}, {Path: "\tpath\r\n🙂"}})
	want = []toolDiffRow{{"path", ""}, {"gap", "⋯"}, {"path", "\tpath\r\n🙂"}}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("empty first path / exact path strings = %#v", rows)
	}
}

func toolDiffRowsTuplesForTest(rows []toolDiffRow) [][]string {
	tuples := make([][]string, len(rows))
	for i, row := range rows {
		tuples[i] = []string{row.Kind, row.Text}
	}
	return tuples
}

func toolDiffRowsSHA256ForTest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func TestToolDiffRowsInstalledEditLimitGoldens(t *testing.T) {
	for _, tc := range []struct {
		count, rowCount    int
		rowsHash, copyHash string
	}{
		{128, 259, "eddcfe770dd181b1a1ff6d78c082967c539ac4a2b20c0c47846493b200fd61c9", "3dd4832b0e50c933a959b07ef9833f8e6b36630efb9c31b449d820634a7e37e2"},
		{129, 263, "0cc5674b5e3842c381ef2bc8fe8896f64f488fcfe08066d78d6279b8b6c5d4b5", "31f39e0d3424262bfc5f5fa26a586fe788d15e8d92c670794f5d205399834626"},
	} {
		old, next := []string{"shared-prefix"}, []string{"shared-prefix"}
		for i := 0; i < tc.count; i++ {
			old, next = append(old, fmt.Sprintf("old-%d", i)), append(next, fmt.Sprintf("new-%d", i))
		}
		old, next = append(old, "shared-suffix"), append(next, "shared-suffix")
		oldText := strings.Join(old, "\n")
		rows := toolDiffRows([]toolFileDiff{{Path: "bound", OldText: &oldText, NewText: strings.Join(next, "\n")}})
		raw, err := json.Marshal(toolDiffRowsTuplesForTest(rows))
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != tc.rowCount || toolDiffRowsSHA256ForTest(raw) != tc.rowsHash || toolDiffRowsSHA256ForTest([]byte(toolDiffCopyText(rows))) != tc.copyHash {
			t.Errorf("%d edits: %d rows, rows SHA256 %s, copy SHA256 %s", tc.count*2, len(rows), toolDiffRowsSHA256ForTest(raw), toolDiffRowsSHA256ForTest([]byte(toolDiffCopyText(rows))))
		}
		if tc.count == 128 {
			if rows[1] != (toolDiffRow{"context", "shared-prefix"}) || rows[len(rows)-1] != (toolDiffRow{"context", "shared-suffix"}) {
				t.Fatal("exact 256 edits lost shared context")
			}
		} else if rows[1] != (toolDiffRow{"del", "shared-prefix"}) || rows[len(old)+1] != (toolDiffRow{"add", "shared-prefix"}) {
			t.Fatal("over 256 edits did not replace both whole fragments")
		}
	}
	for _, count := range []int{256, 257} {
		old := "shared\n"
		next := old + strings.Repeat("added\n", count)
		rows := toolDiffRows([]toolFileDiff{{Path: "insert", OldText: &old, NewText: next}})
		if count == 256 {
			if len(rows) != 258 || rows[1] != (toolDiffRow{"context", "shared"}) {
				t.Fatalf("256 insertions must remain exact: %#v", rows[:min(4, len(rows))])
			}
		} else if len(rows) != 260 || rows[1] != (toolDiffRow{"del", "shared"}) || rows[2] != (toolDiffRow{"add", "shared"}) {
			t.Fatalf("257 insertions must use whole-fragment replacement: %#v", rows[:min(4, len(rows))])
		}
	}
}

// All ordered pairs of binary-line strings through length five expose repeated
// line ambiguity. The expected digest covers the installed eV row order AND tV
// copy string for 3,969 cases, rather than only checking edit counts or an LCS.
func TestToolDiffRowsInstalledRepeatedLineMatrix(t *testing.T) {
	words := []string{""}
	for length := 1; length <= 5; length++ {
		for bits := 0; bits < 1<<length; bits++ {
			lines := make([]string, length)
			for i := range lines {
				lines[i] = "a"
				if bits>>i&1 != 0 {
					lines[i] = "b"
				}
			}
			words = append(words, strings.Join(lines, "\n"))
		}
	}
	all := make([]any, 0, len(words)*len(words))
	for _, old := range words {
		for _, next := range words {
			rows := toolDiffRows([]toolFileDiff{{Path: "f", OldText: &old, NewText: next}})
			all = append(all, []any{toolDiffRowsTuplesForTest(rows), toolDiffCopyText(rows)})
		}
	}
	raw, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	const installedHash = "fff2e1e53ec022af00c3d3bef3f3046e7be8d3616ae9de1c7f87e87ecd38d18e"
	if got := toolDiffRowsSHA256ForTest(raw); got != installedHash {
		t.Fatalf("installed 3,969-case row/copy matrix SHA256 = %s, want %s", got, installedHash)
	}
}

func TestToolDiffRowsLineContentAndLargeFragments(t *testing.T) {
	for _, tc := range []struct {
		old, next string
		want      []toolDiffRow
	}{
		{"a\r\nb\r\n", "a\nb\n", []toolDiffRow{{"path", "f"}, {"del", "a\r"}, {"del", "b\r"}, {"add", "a"}, {"add", "b"}}},
		{"\n\n", "", []toolDiffRow{{"path", "f"}, {"del", ""}, {"del", ""}}},
		{"", "+literal\n-literal\n\\ No newline at end of file\n", []toolDiffRow{{"path", "f"}, {"add", "+literal"}, {"add", "-literal"}, {"add", `\ No newline at end of file`}}},
		{"🙂\n𐐷\n\t \r\n", "🙂\n🦀\n\t \r\n", []toolDiffRow{{"path", "f"}, {"context", "🙂"}, {"del", "𐐷"}, {"add", "🦀"}, {"context", "\t \r"}}},
	} {
		if got := toolDiffRows([]toolFileDiff{{Path: "f", OldText: &tc.old, NewText: tc.next}}); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q / %q rows = %#v, want %#v", tc.old, tc.next, got, tc.want)
		}
	}
	// More than 65,535 distinct lines and more than 256KB must not trigger a
	// line-ID encoding limit, replacement fallback or a UI byte-size skip.
	lines := make([]string, 70000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line-%d", i)
	}
	old := strings.Join(lines, "\n") + "\n"
	lines[66000] = "changed-🙂𐐷\t\r"
	next := strings.Join(lines, "\n") + "\n"
	diffs := []toolFileDiff{{Path: "large", OldText: &old, NewText: next}}
	rows := toolDiffRows(diffs)
	want := []toolDiffRow{{"path", "large"}, {"context", "line-65997"}, {"context", "line-65998"}, {"context", "line-65999"}, {"del", "line-66000"}, {"add", "changed-🙂𐐷\t\r"}, {"context", "line-66001"}, {"context", "line-66002"}, {"context", "line-66003"}}
	if !reflect.DeepEqual(rows, want) || *diffs[0].OldText != old || diffs[0].NewText != next {
		t.Fatalf("large exact fragment rows = %#v, want %#v", rows, want)
	}
	longLine := strings.Repeat("🙂\t", 90000) + "\r"
	old, next = longLine+"\nold\n", longLine+"\nnew\n"
	rows = toolDiffRows([]toolFileDiff{{Path: "long-line", OldText: &old, NewText: next}})
	if len(rows) != 4 || rows[1] != (toolDiffRow{"context", longLine}) || rows[2].Text != "old" || rows[3].Text != "new" {
		t.Fatalf("long content line changed or skipped: %d rows", len(rows))
	}
}

func TestToolDiffCopyFullRowsAndPrefixes(t *testing.T) {
	rows := []toolDiffRow{{"path", "\tfile\r\n🙂"}, {"del", "-old\r"}, {"add", "+new\t𐐷"}, {"context", ""}, {"gap", "⋯"}, {"context", " last "}}
	want := "\tfile\r\n🙂\n- -old\r\n+ +new\t𐐷\n  \n⋯\n   last "
	before := append([]toolDiffRow(nil), rows...)
	if got := toolDiffCopyText(rows); got != want {
		t.Fatalf("copy = %q, want %q", got, want)
	}
	if !reflect.DeepEqual(rows, before) || toolDiffCopyText(nil) != "" {
		t.Fatal("copy changed rows or empty copy")
	}
	many := make([]toolDiffRow, 0, 100)
	for i := 0; i < 100; i++ {
		many = append(many, toolDiffRow{"add", fmt.Sprintf("line-%d", i)})
	}
	copy := toolDiffCopyText(many)
	if strings.Count(copy, "\n") != 99 || !strings.Contains(copy, "+ line-50\n") || !strings.HasSuffix(copy, "+ line-99") {
		t.Fatal("copy omitted middle or tail rows")
	}
}

func TestToolDiffCopyRejectsForgedKind(t *testing.T) {
	defer func() {
		if got := recover(); got != "unreachable diff row kind: forged" {
			t.Fatalf("forged kind panic = %#v", got)
		}
	}()
	toolDiffCopyText([]toolDiffRow{{Kind: "forged", Text: "x"}})
}

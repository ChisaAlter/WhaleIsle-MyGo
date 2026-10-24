package chat

import (
	"encoding/json"
	"strings"
	"unicode/utf16"

	"whaleisle/internal/store/session"
)

type toolDiffSummary struct {
	Added, Removed int
}

// toolDiffSummaryFor derives only first-party root mutation cards: running
// intent, or applied successful write/edit metadata with write's full-file
// fallback. Counting never reads the current file or rewrites recorded hunks.
func toolDiffSummaryFor(call *session.ToolCallView) *toolDiffSummary {
	if call == nil || call.ParentCallID != "" || call.State == "preparing" || toolSummaryHasCue(call.State) || call.Result != nil && call.Call == nil {
		return nil
	}
	args, ok := terminalModelObject(json.RawMessage(call.Arguments))
	if !ok {
		return nil
	}
	pathKey := "file_path"
	if call.Name == "str_replace_editor" {
		pathKey = "path"
	}
	path, ok := toolField(args, pathKey)
	if !ok || strings.TrimSpace(path) == "" {
		return nil
	}
	intended, ok := intendedToolDiff(call.Name, args, path)
	if !ok {
		return nil
	}
	diffs := []toolFileDiff{intended}
	if call.Result != nil {
		if call.Name == "str_replace_editor" {
			return nil
		}
		result := toolSummaryReadResult(call.Result)
		if call.Result.EventType != "tool/result" || result.nested || result.failed || result.stopped {
			return nil
		}
		meta, _ := terminalModelObject(result.meta)
		applied, valid := toolSummaryAppliedDiffs(meta["diffs"])
		if valid && len(applied) > 0 {
			diffs = applied
		} else if call.Name != "write" {
			return nil
		}
	}
	total := toolDiffSummary{}
	for _, d := range diffs {
		old := ""
		if d.OldText != nil {
			old = *d.OldText
		}
		added, removed := toolDiffCounts(old, d.NewText)
		total.Added += added
		total.Removed += removed
	}
	return &total
}

func toolSummaryAppliedDiffs(raw json.RawMessage) ([]toolFileDiff, bool) {
	var values []map[string]json.RawMessage
	if len(raw) == 0 || string(raw) == "null" || json.Unmarshal(raw, &values) != nil {
		return nil, false
	}
	var diffs []toolFileDiff
	for _, fields := range values {
		path, pathOK := toolField(fields, "path")
		next, nextOK := toolField(fields, "newText")
		oldRaw, exists := fields["oldText"]
		if !pathOK || !nextOK || !exists {
			return nil, false
		}
		diff := toolFileDiff{Path: path, NewText: next}
		if string(oldRaw) != "null" {
			old, ok := toolField(fields, "oldText")
			if !ok {
				return nil, false
			}
			diff.OldText = &old
		}
		diffs = append(diffs, diff)
	}
	return diffs, true
}

// The installed primitive uses Myers line edits with maxEditLength256. Once the
// bound is exceeded it counts a full replacement, including otherwise common
// lines. Only one final LF is a terminator; CR and interior blanks remain lines.
func toolDiffCounts(old, next string) (added, removed int) {
	lines := func(text string) []string {
		if text == "" {
			return nil
		}
		return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	}
	a, b := lines(old), lines(next)
	const limit = 256
	var frontier [2*limit + 3]int
	const origin = limit + 1
	for distance := 0; distance <= limit && distance <= len(a)+len(b); distance++ {
		for diagonal := -distance; diagonal <= distance; diagonal += 2 {
			x := 0
			if diagonal == -distance || diagonal != distance && frontier[origin+diagonal-1] < frontier[origin+diagonal+1] {
				x = frontier[origin+diagonal+1]
			} else {
				x = frontier[origin+diagonal-1] + 1
			}
			y := x - diagonal
			for x < len(a) && y < len(b) && x >= 0 && y >= 0 && a[x] == b[y] {
				x++
				y++
			}
			frontier[origin+diagonal] = x
			if x >= len(a) && y >= len(b) {
				return (distance + len(b) - len(a)) / 2, (distance + len(a) - len(b)) / 2
			}
		}
	}
	return len(b), len(a)
}

// contentKilobytes is the mutation row's decoded UTF-16 length, not bytes on
// disk. The original lazy argument reader also supplies partial preparation;
// this complete-JSON projection covers dispatched and retained complete calls.
func toolContentKilobytes(name, arguments string) (int, bool) {
	if name != "write" && name != "edit" {
		return 0, false
	}
	fields, ok := terminalModelObject(json.RawMessage(arguments))
	if !ok {
		return 0, false
	}
	length, present := 0, false
	for _, key := range []string{"content", "old_string", "new_string"} {
		if _, exists := fields[key]; exists {
			present = true
		}
		if value, ok := toolField(fields, key); ok {
			for _, r := range value {
				length += utf16.RuneLen(r)
			}
		}
	}
	return (length + 1023) / 1024, present
}

func toolSummaryHasCue(status string) bool {
	return toolRowStatus(status) == statusError || toolRowStatus(status) == statusWarning
}

package chat

import "strings"

// toolDiffRow is one full, unfolded installed diff-card row. Kind is path, del,
// add, context or gap; Text preserves content whitespace and recorded path text.
type toolDiffRow struct {
	Kind string
	Text string
}

// toolDiffRows follows localHunks/buildRows: context is three lines, consecutive
// same-path fragments and distant local hunks get a gap, and an unchanged
// fragment still contributes its path or gap. No display-height or byte cap is
// applied here; the edit search alone is bounded at 256 insertions/deletions.
func toolDiffRows(diffs []toolFileDiff) []toolDiffRow {
	rows := make([]toolDiffRow, 0)
	previous, havePrevious := "", false
	for _, diff := range diffs {
		if !havePrevious || diff.Path != previous {
			rows = append(rows, toolDiffRow{Kind: "path", Text: diff.Path})
		} else {
			rows = append(rows, toolDiffRow{Kind: "gap", Text: "⋯"})
		}
		previous, havePrevious = diff.Path, true
		for i, hunk := range toolDiffLocalHunks(diff) {
			if i > 0 {
				rows = append(rows, toolDiffRow{Kind: "gap", Text: "⋯"})
			}
			rows = append(rows, hunk...)
		}
	}
	return rows
}

// toolDiffCopyText copies every supplied row, including rows a caller may fold.
// Only content rows receive a prefix; paths and gap text remain verbatim.
func toolDiffCopyText(rows []toolDiffRow) string {
	var text strings.Builder
	for i, row := range rows {
		if i > 0 {
			text.WriteByte('\n')
		}
		switch row.Kind {
		case "del":
			text.WriteString("- ")
		case "add":
			text.WriteString("+ ")
		case "context":
			text.WriteString("  ")
		case "path", "gap":
		default:
			panic("unreachable diff row kind: " + row.Kind)
		}
		text.WriteString(row.Text)
	}
	return text.String()
}

func toolDiffContentLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

type toolDiffChange struct {
	kind  string
	lines []string
}

func toolDiffLocalHunks(diff toolFileDiff) [][]toolDiffRow {
	old := ""
	if diff.OldText != nil {
		old = *diff.OldText
	}
	oldLines, newLines := toolDiffContentLines(old), toolDiffContentLines(diff.NewText)
	changes, exact := toolDiffLineChanges(oldLines, newLines)
	if !exact {
		hunk := make([]toolDiffRow, 0, len(oldLines)+len(newLines))
		for _, line := range oldLines {
			hunk = append(hunk, toolDiffRow{Kind: "del", Text: line})
		}
		for _, line := range newLines {
			hunk = append(hunk, toolDiffRow{Kind: "add", Text: line})
		}
		return [][]toolDiffRow{hunk}
	}
	// structuredPatch closes each open range with a context component, including
	// an empty sentinel. Six shared lines join edits; seven split local hunks.
	changes = append(changes, toolDiffChange{kind: "context"})
	var hunks [][]toolDiffRow
	var current []toolDiffRow
	active := false
	context := func(lines []string) {
		for _, line := range lines {
			current = append(current, toolDiffRow{Kind: "context", Text: line})
		}
	}
	for i, change := range changes {
		if change.kind == "add" || change.kind == "del" {
			if !active {
				active = true
				if i > 0 {
					previous := changes[i-1].lines
					context(previous[max(0, len(previous)-3):])
				}
			}
			for _, line := range change.lines {
				current = append(current, toolDiffRow{Kind: change.kind, Text: line})
			}
		} else if active {
			if len(change.lines) <= 6 && i < len(changes)-2 {
				context(change.lines)
			} else {
				context(change.lines[:min(3, len(change.lines))])
				hunks = append(hunks, current)
				current, active = nil, false
			}
		}
	}
	return hunks
}

// Components retain complete lines rather than encoding line IDs as Unicode.
// Their linked history matches jsdiff's greedy path choices without copying an
// entire edit history for each diagonal or corrupting non-BMP content.
type toolDiffComponent struct {
	kind     string
	count    int
	previous *toolDiffComponent
}

type toolDiffSearchPath struct {
	oldPos int
	last   *toolDiffComponent
}

func toolDiffLineChanges(old, next []string) ([]toolDiffChange, bool) {
	const limit, origin = 256, 257
	var best [2*limit + 3]*toolDiffSearchPath
	best[origin] = &toolDiffSearchPath{oldPos: -1}
	newPos := toolDiffExtractCommon(best[origin], old, next, 0)
	if best[origin].oldPos+1 >= len(old) && newPos+1 >= len(next) {
		return toolDiffBuildChanges(best[origin].last, old, next), true
	}
	minDiagonal, maxDiagonal := -limit, limit
	for edits := 1; edits <= min(limit, len(old)+len(next)); edits++ {
		for diagonal := max(minDiagonal, -edits); diagonal <= min(maxDiagonal, edits); diagonal += 2 {
			removePath, addPath := best[origin+diagonal-1], best[origin+diagonal+1]
			if removePath != nil {
				best[origin+diagonal-1] = nil
			}
			canAdd := addPath != nil && addPath.oldPos-diagonal >= 0 && addPath.oldPos-diagonal < len(next)
			canRemove := removePath != nil && removePath.oldPos+1 < len(old)
			if !canAdd && !canRemove {
				best[origin+diagonal] = nil
				continue
			}
			var path *toolDiffSearchPath
			// Equal old positions select removal, as in the installed jsdiff.
			if !canRemove || canAdd && removePath.oldPos < addPath.oldPos {
				path = toolDiffAddToPath(addPath, "add", 0)
			} else {
				path = toolDiffAddToPath(removePath, "del", 1)
			}
			newPos = toolDiffExtractCommon(path, old, next, diagonal)
			if path.oldPos+1 >= len(old) && newPos+1 >= len(next) {
				return toolDiffBuildChanges(path.last, old, next), true
			}
			best[origin+diagonal] = path
			if path.oldPos+1 >= len(old) {
				maxDiagonal = min(maxDiagonal, diagonal-1)
			}
			if newPos+1 >= len(next) {
				minDiagonal = max(minDiagonal, diagonal+1)
			}
		}
	}
	return nil, false
}

func toolDiffAddToPath(path *toolDiffSearchPath, kind string, oldIncrement int) *toolDiffSearchPath {
	component := &toolDiffComponent{kind: kind, count: 1, previous: path.last}
	if last := path.last; last != nil && last.kind == kind {
		component.count, component.previous = last.count+1, last.previous
	}
	return &toolDiffSearchPath{oldPos: path.oldPos + oldIncrement, last: component}
}

func toolDiffExtractCommon(path *toolDiffSearchPath, old, next []string, diagonal int) int {
	oldPos, newPos, count := path.oldPos, path.oldPos-diagonal, 0
	for oldPos+1 < len(old) && newPos+1 < len(next) && old[oldPos+1] == next[newPos+1] {
		oldPos++
		newPos++
		count++
	}
	if count > 0 {
		path.last = &toolDiffComponent{kind: "context", count: count, previous: path.last}
	}
	path.oldPos = oldPos
	return newPos
}

func toolDiffBuildChanges(last *toolDiffComponent, old, next []string) []toolDiffChange {
	var components []*toolDiffComponent
	for ; last != nil; last = last.previous {
		components = append(components, last)
	}
	changes := make([]toolDiffChange, 0, len(components))
	oldPos, newPos := 0, 0
	for i := len(components) - 1; i >= 0; i-- {
		component := components[i]
		change := toolDiffChange{kind: component.kind}
		if component.kind == "del" {
			change.lines = old[oldPos : oldPos+component.count]
			oldPos += component.count
		} else {
			change.lines = next[newPos : newPos+component.count]
			newPos += component.count
			if component.kind != "add" {
				oldPos += component.count
			}
		}
		changes = append(changes, change)
	}
	return changes
}

package ui

import (
	"strings"
	"unicode/utf8"
)

// ListTextSource is the selectable text model behind a virtual list. It keeps
// text ownership with the caller: the engine does not render offscreen rows or
// retain a second document. See Element.SelectionText for the paragraph contract.
type ListTextSource struct {
	Revision   uint64
	Paragraphs func(row int) []string
}

// listTextPoint is used only while a Select All endpoint has never been mounted.
// The row key and paragraph ordinal rebind it to the actual Text when it appears.
type listTextPoint int

func (rt *engine) selectionTextSource(scope, listID uint64) (*ListState, bool) {
	if scope != listID {
		return nil, false
	}
	ls, ok := rt.listAlive(listID)
	return ls, ok && ls.TextSource.Paragraphs != nil
}

func listTextRowKey(ls *ListState, row int) any {
	if ls.Key != nil {
		return ls.Key(row)
	}
	return row
}

func (rt *engine) mountedListParagraph(scope, rowID uint64, ordinal int) *state {
	for _, p := range rt.texts {
		if p.textScope != scope || !rt.selectionStateActive(p.id) || !rt.within(p.id, rowID) {
			continue
		}
		if ordinal == 0 {
			return p
		}
		ordinal--
	}
	return nil
}

func (rt *engine) selectAllListText(scope uint64) bool {
	ls, ok := rt.selectionTextSource(scope, scope)
	if !ok {
		return false
	}
	var first, last textPoint
	var firstMeta, lastMeta textPointMeta
	var firstKey, lastKey any
	var firstText, lastText string
	endpoint := func(row, ordinal, offset int) (textPoint, textPointMeta, any) {
		key := listTextRowKey(ls, row)
		rowID := keyedID(scope, key)
		pt := textPoint{id: keyedID(rowID, listTextPoint(ordinal)), at: offset}
		if p := rt.mountedListParagraph(scope, rowID, ordinal); p != nil {
			pt.id = p.id
		}
		return pt, textPointMeta{listID: scope, rowID: rowID, rowIndex: row, inRow: ordinal}, key
	}
	for row := 0; row < ls.frame.n; row++ {
		paragraphs := ls.TextSource.Paragraphs(row)
		if len(paragraphs) == 0 {
			continue
		}
		first, firstMeta, firstKey = endpoint(row, 0, 0)
		firstText = paragraphs[0]
		break
	}
	for row := ls.frame.n - 1; row >= 0; row-- {
		paragraphs := ls.TextSource.Paragraphs(row)
		if len(paragraphs) == 0 {
			continue
		}
		ordinal := len(paragraphs) - 1
		last, lastMeta, lastKey = endpoint(row, ordinal, utf8.RuneCountInString(paragraphs[ordinal]))
		lastText = paragraphs[ordinal]
		break
	}
	if first.id == 0 {
		return true
	}
	rt.selection = textSelection{scope: scope, anchor: first, caret: last, anchorMeta: firstMeta, caretMeta: lastMeta}
	logic := rt.ensureSelectionLogic()
	logic.anchorKey, logic.caretKey = firstKey, lastKey
	logic.snapshots[first.id], logic.snapshots[last.id] = firstText, lastText
	logic.sourceRevision = ls.TextSource.Revision
	return true
}

// reconcileListTextPoint keeps a logical endpoint attached to its row even when
// that row is offscreen. Append-only streaming preserves the offset; replacement
// of endpoint text invalidates the selection rather than copying unrelated text.
func (rt *engine) reconcileListTextPoint(ls *ListState, pt *textPoint, meta *textPointMeta, key any) bool {
	row, ok := resolveListKey(ls, meta.rowIndex, key)
	if !ok {
		return false
	}
	paragraphs := ls.TextSource.Paragraphs(row)
	if meta.inRow < 0 || meta.inRow >= len(paragraphs) {
		return false
	}
	text := paragraphs[meta.inRow]
	logic := rt.ensureSelectionLogic()
	old, known := logic.snapshots[pt.id]
	if known && (ls.TextSource.Revision == 0 || logic.sourceRevision != ls.TextSource.Revision) && text != old && !strings.HasPrefix(text, old) {
		return false
	}
	meta.rowIndex = row
	meta.rowID = keyedID(meta.listID, listTextRowKey(ls, row))
	pt.at = min(max(0, pt.at), utf8.RuneCountInString(text))
	if mounted := rt.mountedListParagraph(rt.selection.scope, meta.rowID, meta.inRow); mounted != nil {
		// A source/UI disagreement must never put text different from the
		// highlighted paragraph on the clipboard.
		if mounted.editor == nil || mounted.editor.source != text {
			return false
		}
		oldID := pt.id
		pt.id = mounted.id
		if rt.focused == oldID {
			rt.focused = mounted.id
		}
	}
	logic.snapshots[pt.id] = text
	return true
}

func (rt *engine) copyListText(ls *ListState, start textPoint, startMeta textPointMeta, end textPoint, endMeta textPointMeta) (string, bool) {
	revision := ls.TextSource.Revision
	if startMeta.rowIndex < 0 || endMeta.rowIndex >= ls.frame.n {
		return "", false
	}
	var out strings.Builder
	wrote := false
	for row := startMeta.rowIndex; row <= endMeta.rowIndex; row++ {
		paragraphs := ls.TextSource.Paragraphs(row)
		lo, hi := 0, len(paragraphs)
		if row == startMeta.rowIndex {
			lo = startMeta.inRow
		}
		if row == endMeta.rowIndex {
			hi = endMeta.inRow + 1
		}
		if lo < 0 || hi > len(paragraphs) || lo > hi {
			return "", false
		}
		for ordinal := lo; ordinal < hi; ordinal++ {
			text := paragraphs[ordinal]
			from, to := 0, utf8.RuneCountInString(text)
			if row == startMeta.rowIndex && ordinal == startMeta.inRow {
				from = min(max(0, start.at), to)
			}
			if row == endMeta.rowIndex && ordinal == endMeta.inRow {
				to = min(max(0, end.at), to)
			}
			if wrote {
				out.WriteByte('\n')
			}
			out.WriteString(sliceTextRunes(text, from, to))
			wrote = true
		}
	}
	return out.String(), ls.TextSource.Revision == revision
}

// sliceTextRunes avoids allocating []rune for every copied source line.
func sliceTextRunes(text string, from, to int) string {
	if to <= from {
		return ""
	}
	start, end, at := 0, len(text), 0
	for byteIndex := range text {
		if at == from {
			start = byteIndex
		}
		if at == to {
			end = byteIndex
			break
		}
		at++
	}
	return text[start:end]
}

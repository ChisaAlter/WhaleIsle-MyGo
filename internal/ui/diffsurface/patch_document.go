package diffsurface

import (
	"path"
	"strconv"
	"strings"
)

type patchFile struct {
	Path, Source string
	Rows         []patchRow
}
type patchRow struct {
	Old, New, Meta       string
	OldNumber, NewNumber int
}
type patchViewState struct {
	Split, Wrap bool
	Closed      map[string]bool
	// Folded marks a file's collapsed hunks by hunk index; a missing entry is an
	// expanded hunk, which is the default every patch opens with.
	Folded map[string]map[int]bool
	source string
	files  []patchFile
}

func (s *patchViewState) update(source string) []patchFile {
	if s.Closed == nil {
		s.Closed = map[string]bool{}
	}
	if s.Folded == nil {
		s.Folded = map[string]map[int]bool{}
	}
	if s.source != source {
		s.source = source
		s.files = parsePatchFiles(source)
		keep := map[string]bool{}
		keepFolded := map[string]map[int]bool{}
		for _, file := range s.files {
			keep[file.Path] = s.Closed[file.Path]
			if hunks := s.Folded[file.Path]; len(hunks) > 0 {
				keepFolded[file.Path] = hunks
			}
		}
		s.Closed = keep
		s.Folded = keepFolded
	}
	return s.files
}

// foldedHunks is the collapse state of one file's hunks, created on demand.
func (s *patchViewState) foldedHunks(path string) map[int]bool {
	if s.Folded == nil {
		s.Folded = map[string]map[int]bool{}
	}
	hunks := s.Folded[path]
	if hunks == nil {
		hunks = map[int]bool{}
		s.Folded[path] = hunks
	}
	return hunks
}

// toggleHunk folds or unfolds one hunk of one file.
func (s *patchViewState) toggleHunk(path string, index int) {
	hunks := s.foldedHunks(path)
	hunks[index] = !hunks[index]
}

// setAllHunks folds or unfolds every hunk of every visible file, the per-hunk
// counterpart of setAll.
func (s *patchViewState) setAllHunks(folded bool) {
	for _, file := range s.files {
		if !folded {
			delete(s.Folded, file.Path)
			continue
		}
		hunks := s.foldedHunks(file.Path)
		index := 0
		for _, row := range file.Rows {
			if !strings.HasPrefix(row.Meta, "@@") {
				continue
			}
			hunks[index] = true
			index++
		}
	}
}
func (s *patchViewState) setAllClosed(closed bool) {
	for _, file := range s.files {
		if !closed {
			delete(s.Closed, file.Path)
			continue
		}
		s.Closed[file.Path] = true
	}
}
func safePatchPath(p string) bool {
	if p == "" || p == "." || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\\\x00:") || path.Clean(p) != p || p == ".." || strings.HasPrefix(p, "../") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if strings.EqualFold(part, ".git") {
			return false
		}
	}
	return true
}
func patchPath(raw string) string {
	if strings.HasPrefix(raw, "\"") {
		if decoded, err := strconv.Unquote(raw); err == nil {
			raw = decoded
		}
	} else if at := strings.IndexByte(raw, '\t'); at >= 0 {
		raw = raw[:at]
	}
	if raw == "/dev/null" {
		return ""
	}
	if strings.HasPrefix(raw, "a/") || strings.HasPrefix(raw, "b/") {
		return raw[2:]
	}
	return raw
}
func headerPath(line string) string {
	raw := strings.TrimPrefix(line, "diff --git ")
	if strings.HasPrefix(raw, "\"") {
		// Git C-quotes each path independently. Skip the first quoted token.
		escaped := false
		for i := 1; i < len(raw); i++ {
			if !escaped && raw[i] == '"' {
				return patchPath(strings.TrimSpace(raw[i+1:]))
			}
			if raw[i] == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		}
	}
	if at := strings.LastIndex(raw, " b/"); at >= 0 {
		return patchPath(raw[at+1:])
	}
	return ""
}
func parsePatchFiles(source string) []patchFile {
	var files []patchFile
	var lines []string
	flush := func() {
		if len(lines) == 0 {
			return
		}
		file := patchFile{Path: headerPath(lines[0]), Source: strings.Join(lines, "\n")}
		inHunk := false
		for _, line := range lines {
			if strings.HasPrefix(line, "@@") {
				inHunk = true
			}
			if inHunk {
				continue
			}
			switch {
			case strings.HasPrefix(line, "--- "):
				if p := patchPath(line[4:]); p != "" {
					file.Path = p
				}
			case strings.HasPrefix(line, "+++ "):
				if p := patchPath(line[4:]); p != "" {
					file.Path = p
				}
			case strings.HasPrefix(line, "rename to "):
				file.Path = literalPatchPath(strings.TrimPrefix(line, "rename to "))
			case strings.HasPrefix(line, "copy to "):
				file.Path = literalPatchPath(strings.TrimPrefix(line, "copy to "))
			}
		}
		file.Rows = splitPatchRows(file.Source)
		files = append(files, file)
	}
	for _, line := range strings.Split(strings.TrimSuffix(source, "\n"), "\n") {
		if strings.HasPrefix(line, "diff --git ") && len(lines) > 0 {
			flush()
			lines = nil
		}
		lines = append(lines, line)
	}
	if source != "" {
		flush()
	}
	return files
}
func splitPatchRows(source string) []patchRow {
	var rows []patchRow
	var removed, added []patchRow
	oldLine, newLine := 0, 0
	inHunk := false
	flush := func() {
		n := len(removed)
		if len(added) > n {
			n = len(added)
		}
		for i := 0; i < n; i++ {
			var r patchRow
			if i < len(removed) {
				r.Old = removed[i].Old
				r.OldNumber = removed[i].OldNumber
			}
			if i < len(added) {
				r.New = added[i].New
				r.NewNumber = added[i].NewNumber
			}
			rows = append(rows, r)
		}
		removed = nil
		added = nil
	}
	for _, line := range strings.Split(strings.TrimSuffix(source, "\n"), "\n") {
		if strings.HasPrefix(line, "@@ ") {
			flush()
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				oldLine, _ = strconv.Atoi(strings.Split(strings.TrimPrefix(fields[1], "-"), ",")[0])
				newLine, _ = strconv.Atoi(strings.Split(strings.TrimPrefix(fields[2], "+"), ",")[0])
				inHunk = true
			}
			rows = append(rows, patchRow{Meta: line})
			continue
		}
		if !inHunk || line == "" || line[0] == '\\' {
			flush()
			rows = append(rows, patchRow{Meta: line})
			continue
		}
		switch line[0] {
		case '-':
			if len(added) > 0 {
				flush()
			}
			removed = append(removed, patchRow{Old: line, OldNumber: oldLine})
			oldLine++
		case '+':
			added = append(added, patchRow{New: line, NewNumber: newLine})
			newLine++
		case ' ':
			flush()
			rows = append(rows, patchRow{Old: line, New: line, OldNumber: oldLine, NewNumber: newLine})
			oldLine++
			newLine++
		default:
			flush()
			rows = append(rows, patchRow{Meta: line})
		}
	}
	flush()
	return rows
}

func literalPatchPath(raw string) string {
	if strings.HasPrefix(raw, "\"") {
		if decoded, err := strconv.Unquote(raw); err == nil {
			return decoded
		}
	}
	return raw
}

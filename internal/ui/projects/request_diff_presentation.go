package projects

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/projectflow"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/diff"
)

const requestDiffLineLimit = 400 // dsh-project/client/client.js diffLineLimit.

type requestDiffFile struct {
	diffsurface.RecordedPatchFile
	kind           string
	added, removed int
	binary         bool
}

type requestDiffDocument struct {
	source string
	files  []diffsurface.RecordedPatchFile
}

// renderRequestChangeScope presents one recorded requirement interval. It does
// not read the current repository or offer actions against the displayed files.
func renderRequestChangeScope(c *ui.Context, m Model, scope projectflow.RequestChangeScope) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	st := diff.Style{FontSize: theme.CurrentPreferences().Code.Size, FG: chrome.Color(colors.FG),
		Muted: chrome.Color(colors.Muted), Hunk: chrome.Color(colors.Accent),
		Add: ui.RGB(61, 153, 112), Del: ui.RGB(224, 102, 102)}
	box := ui.Column(c.Key("request-diff-scope/" + scope.WorkingDirectory)).FillWidth().MinWidth(0).Gap(8)
	cache := ui.Local(box, "patch-document", func() requestDiffDocument { return requestDiffDocument{} })
	if cache.source != scope.Diff {
		cache.source, cache.files = scope.Diff, diffsurface.ParseRecordedPatch(scope.Diff)
	}
	files := requestChangeFiles(scope, cache.files)
	box.Children(func() {
		if scope.WorkingDirectory != "" {
			ui.Text(c, scope.WorkingDirectory).FontSize(theme.InterfaceSize(12)).TextColor(st.Muted).Selectable()
		}
		if len(files) == 0 {
			ui.Text(c, tr("没有改动。", "No changes.")).FontSize(theme.InterfaceSize(13)).TextColor(st.Muted)
		} else {
			added, removed := 0, 0
			for _, file := range files {
				added += file.added
				removed += file.removed
			}
			ui.Row(c).Gap(10).Children(func() {
				ui.Text(c, fmt.Sprintf(tr("%d 个文件", "%d files"), len(files))).FontSize(theme.InterfaceSize(13)).FontWeight(600).TextColor(st.FG)
				requestDiffCounts(c, added, removed, st)
			})
			for i, file := range files {
				renderRequestDiffFile(c, file, i < 3, st, chrome.Color(colors.Field), chrome.Color(colors.Border))
			}
		}
		if scope.Truncated {
			ui.Text(c, tr("差异已截断；请在工作目录中查看完整内容。", "Diff is truncated; inspect the working directory for complete contents.")).FontSize(theme.InterfaceSize(12)).TextColor(st.Muted)
		}
	})
}

// Keep the original RequestDiff union: parsed files first, then metadata-only
// entries. The native parser supplies quoted paths, renames and hunk numbers.
func requestChangeFiles(scope projectflow.RequestChangeScope, parsed []diffsurface.RecordedPatchFile) []requestDiffFile {
	var files []requestDiffFile
	known := map[string]int{}
	for _, patch := range parsed {
		if patch.Path == "" && strings.TrimSpace(patch.Source) == "" {
			continue
		}
		file := requestDiffFile{RecordedPatchFile: patch, kind: "modified"}
		for _, line := range strings.Split(patch.Source, "\n") {
			switch {
			case strings.HasPrefix(line, "new file mode "):
				file.kind = "added"
			case strings.HasPrefix(line, "deleted file mode "):
				file.kind = "deleted"
			case strings.HasPrefix(line, "rename from "):
				file.kind = "renamed"
			case strings.HasPrefix(line, "Binary files "), line == "GIT binary patch":
				file.binary = true
			}
		}
		for _, row := range patch.Rows {
			if strings.HasPrefix(row.Old, "-") {
				file.removed++
			}
			if strings.HasPrefix(row.New, "+") {
				file.added++
			}
		}
		known[patch.Path] = len(files)
		files = append(files, file)
	}
	for _, entry := range scope.Files {
		if at, exists := known[entry.Path]; exists {
			files[at].binary = files[at].binary || entry.Binary
			continue
		}
		known[entry.Path] = len(files)
		files = append(files, requestDiffFile{RecordedPatchFile: diffsurface.RecordedPatchFile{Path: entry.Path},
			kind: requestDiffStatusKind(entry.Status), added: entry.Added, removed: entry.Removed, binary: entry.Binary})
	}
	return files
}

func requestDiffStatusKind(status string) string {
	switch {
	case status == "??":
		return "untracked"
	case strings.Contains(status, "D"):
		return "deleted"
	case strings.Contains(status, "A"):
		return "added"
	case strings.Contains(status, "R"):
		return "renamed"
	default:
		return "modified"
	}
}

func renderRequestDiffFile(c *ui.Context, file requestDiffFile, initiallyOpen bool, st diff.Style, field, border ui.Color) {
	box := ui.Column(c.Key("request-diff-file/"+file.Path)).FillWidth().MinWidth(0).Gap(4).Border(1, border).Radius(8)
	open := ui.Local(box, "open", func() bool { return initiallyOpen })
	box.Children(func() {
		label, ink := tr("已修改", "Modified"), st.Muted
		switch file.kind {
		case "added":
			label, ink = tr("新增", "Added"), st.Add
		case "deleted":
			label, ink = tr("删除", "Deleted"), st.Del
		case "renamed":
			label, ink = tr("重命名", "Renamed"), st.Hunk
		case "untracked":
			label = tr("未跟踪", "Untracked")
		}
		ui.Row(c).FillWidth().MinWidth(0).Gap(8).Padding(8).Background(field).AlignItems(ui.Center).Children(func() {
			ui.Text(c, label).FontSize(theme.InterfaceSize(11)).TextColor(ink)
			path := file.Path
			if path == "" {
				path = tr("差异内容", "Diff contents")
			}
			mark := "▸ "
			if *open {
				mark = "▾ "
			}
			trigger := ui.Button(c, mark+path).Label(path).Grow(1).MinWidth(0).Justify(ui.Start).Background(ui.Transparent).BorderWidth(0).TextColor(st.FG).FontSize(theme.InterfaceSize(12)).Expanded(*open)
			if trigger.Clicked() {
				*open = !*open
				c.Invalidate()
			}
			requestDiffCounts(c, file.added, file.removed, st)
		})
		if !*open {
			return
		}
		note := ""
		switch {
		case file.kind == "untracked":
			note = tr("未跟踪文件不包含在 Git 差异中。", "Untracked file contents are not included in the Git diff.")
		case file.binary:
			note = tr("二进制文件，无法显示文本差异。", "Binary file; no text diff is available.")
		}
		lines := requestDiffLines(file.Source)
		capped := len(lines) > requestDiffLineLimit
		if capped {
			lines = lines[:requestDiffLineLimit]
		}
		if note == "" && len(lines) == 0 {
			note = tr("没有可显示的文本差异。", "No text diff is available.")
		}
		if note != "" {
			ui.Text(c, note).FontSize(theme.InterfaceSize(12)).TextColor(st.Muted).Padding(8)
			return
		}
		// The original DiffFile is unified: preserve source order and signs,
		// including the original line ranges in each @@ hunk header.
		ui.ScrollHorizontal(c).FillWidth().MinWidth(0).Children(func() {
			ui.Column(c).MinWidth(0).Padding(8).Children(func() {
				spans := diff.Spans(strings.Join(lines, "\n"), st)
				for i, line := range lines {
					if i >= len(spans) {
						spans = append(spans, ui.Span{Font: theme.RenderPreferences().Code.Family, Size: st.FontSize, Color: st.FG})
					}
					// Spans also accepts complete patches and trims their outer
					// whitespace. Here every source line is already hunk content.
					spans[i].Text = line + "\n"
					switch {
					case strings.HasPrefix(line, "+"):
						spans[i].Color = st.Add
					case strings.HasPrefix(line, "-"):
						spans[i].Color = st.Del
					case strings.HasPrefix(line, "\\"):
						spans[i].Color = st.Muted
					}
				}
				ui.RichText(c, spans...).Font(theme.RenderPreferences().Code.Family).FontSize(st.FontSize).NoWrap().Selectable()
			})
		})
		if capped {
			ui.Text(c, fmt.Sprintf(tr("每个文件仅显示前 %d 行。", "Only the first %d lines of each file are shown."), requestDiffLineLimit)).FontSize(theme.InterfaceSize(12)).TextColor(st.Muted).Padding(8)
		}
	})
}

func requestDiffCounts(c *ui.Context, added, removed int, st diff.Style) {
	if added > 0 {
		ui.Text(c, "+"+strconv.Itoa(added)).FontSize(theme.InterfaceSize(12)).TextColor(st.Add)
	}
	if removed > 0 {
		ui.Text(c, "−"+strconv.Itoa(removed)).FontSize(theme.InterfaceSize(12)).TextColor(st.Del)
	}
}

func requestDiffLines(source string) []string {
	lines := strings.Split(strings.TrimSuffix(source, "\n"), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "@@") {
			return lines[i:]
		}
	}
	return nil
}

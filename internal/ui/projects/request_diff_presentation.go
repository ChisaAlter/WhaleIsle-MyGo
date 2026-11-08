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
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	dark := th.Scheme == theme.SchemeDark || th.Scheme == theme.SchemeSystem && m.SystemDark
	panel, borderL1, removed := ui.RGB(255, 255, 255), ui.RGBA(0, 0, 0, .04), ui.RGB(236, 19, 19)
	if dark {
		panel, borderL1, removed = ui.RGB(44, 44, 46), ui.RGBA(255, 255, 255, .06), ui.RGB(242, 90, 90)
	}
	st := diff.Style{FontSize: 12, FG: chrome.Color(colors.FG),
		Muted: chrome.Color(colors.Muted), Hunk: chrome.Color(colors.Muted), File: chrome.Color(colors.Accent),
		Add: ui.RGB(34, 197, 94), Del: removed}
	box := ui.Column(c.Key("request-diff-scope/" + scope.WorkingDirectory)).FillWidth().MinWidth(0).Gap(12)
	cache := ui.Local(box, "patch-document", func() requestDiffDocument { return requestDiffDocument{} })
	if cache.source != scope.Diff {
		cache.source, cache.files = scope.Diff, diffsurface.ParseRecordedPatch(scope.Diff)
	}
	files := requestChangeFiles(scope, cache.files)
	box.Children(func() {
		if scope.WorkingDirectory != "" {
			ui.Text(c, scope.WorkingDirectory).FontSize(12).FixedLineHeight(18).TextColor(st.Muted).Selectable()
		}
		if len(files) == 0 {
			ui.Text(c, tr("没有改动。", "No changes.")).FontSize(13).FixedLineHeight(20).TextColor(st.Muted)
		} else {
			added, removed := 0, 0
			for _, file := range files {
				added += file.added
				removed += file.removed
			}
			ui.Row(c).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf(tr("%d 个文件", "%d files"), len(files))).FontSize(13).FixedLineHeight(20).FontWeight(600).TextColor(st.FG)
				requestDiffCounts(c, added, removed, 20, st)
			})
			for i, file := range files {
				renderRequestDiffFile(c, file, i < 3, st, panel, chrome.Color(colors.Border), borderL1, chrome.Color(colors.Hover))
			}
		}
		if scope.Truncated {
			requestDiffNote(c, tr("差异已截断；请在工作目录中查看完整内容。", "Diff is truncated; inspect the working directory for complete contents."), st)
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

func renderRequestDiffFile(c *ui.Context, file requestDiffFile, initiallyOpen bool, st diff.Style, panel, border, borderL1, hover ui.Color) {
	box := ui.Column(c.Key("request-diff-file/"+file.Path)).FillWidth().MinWidth(0).Gap(0).Border(1, border).Radius(12).Background(panel).Clip()
	open := ui.Local(box, "open", func() bool { return initiallyOpen })
	box.Children(func() {
		label, ink := tr("已修改", "Modified"), st.Muted
		switch file.kind {
		case "added":
			label, ink = tr("新增", "Added"), st.Add
		case "deleted":
			label, ink = tr("删除", "Deleted"), st.Del
		case "renamed":
			label, ink = tr("重命名", "Renamed"), st.File
		case "untracked":
			label = tr("未跟踪", "Untracked")
		}
		header := ui.Button(c, "").FillWidth().MinWidth(0).Gap(8).Padding(8, 12).Radius(0).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).FontSize(12).FixedLineHeight(18).Label(file.Path).Expanded(*open)
		if header.Hovered() {
			header.Background(hover)
		}
		if *open {
			header.BorderWidth(0, 0, 1, 0).BorderColor(borderL1)
		}
		if header.Clicked() {
			*open = !*open
			c.Invalidate()
		}
		header.Children(func() {
			ui.Text(c, label).FontSize(12).FixedLineHeight(18).TextColor(ink)
			path := file.Path
			if path == "" {
				path = tr("差异内容", "Diff contents")
			}
			var name []ui.Span
			font := theme.RenderPreferences().Code.Family
			if slash := strings.LastIndex(path, "/"); slash >= 0 {
				name = append(name, ui.Span{Text: path[:slash+1], Font: font, Size: 12, Color: st.Muted})
				path = path[slash+1:]
			}
			name = append(name, ui.Span{Text: path, Font: font, Size: 12, Color: st.FG})
			ui.RichText(c, name...).Font(font).FontSize(12).FixedLineHeight(18).Grow(1).MinWidth(0).SingleLine()
			requestDiffCounts(c, file.added, file.removed, 18, st)
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
			requestDiffNote(c, note, st)
			return
		}
		// The original DiffFile is unified: preserve source order and signs,
		// including the original line ranges in each @@ hunk header.
		ui.ScrollBoth(c).FillWidth().MinWidth(0).MaxHeight(360).Children(func() {
			ui.Column(c).FillWidth().Padding(4, 0).Gap(0).Selectable().Children(func() {
				for _, line := range lines {
					ink, bg := st.FG, ui.Transparent
					switch {
					case strings.HasPrefix(line, "+"):
						bg = ui.RGBA(st.Add.R, st.Add.G, st.Add.B, .1)
					case strings.HasPrefix(line, "-"):
						bg = ui.RGBA(st.Del.R, st.Del.G, st.Del.B, .1)
					case strings.HasPrefix(line, "@@"):
						ink, bg = st.Muted, hover
					case strings.HasPrefix(line, "\\"):
						ink = st.Muted
					}
					if line == "" {
						line = " "
					}
					ui.Text(c, requestDiffExpandTabs(line)).Font(theme.RenderPreferences().Code.Family).FontSize(12).FixedLineHeight(18).NoWrap().Padding(0, 12).Background(bg).TextColor(ink).Selectable()
				}
			})
		})
		if capped {
			requestDiffNote(c, fmt.Sprintf(tr("每个文件仅显示前 %d 行。", "Only the first %d lines of each file are shown."), requestDiffLineLimit), st)
		}
	})
}

func requestDiffCounts(c *ui.Context, added, removed int, lineHeight float32, st diff.Style) {
	if added > 0 {
		ui.Text(c, "+"+strconv.Itoa(added)).FontSize(12).FixedLineHeight(lineHeight).TextColor(st.Add)
	}
	if removed > 0 {
		ui.Text(c, "−"+strconv.Itoa(removed)).FontSize(12).FixedLineHeight(lineHeight).TextColor(st.Del)
	}
}

func requestDiffNote(c *ui.Context, text string, st diff.Style) {
	ui.Text(c, text).FontSize(12).FixedLineHeight(18).TextColor(st.Muted).Padding(8, 12)
}

// Native text has no tab-size option; expand at the original two-column stops.
func requestDiffExpandTabs(line string) string {
	if !strings.ContainsRune(line, '\t') {
		return line
	}
	var expanded strings.Builder
	column := 0
	for _, r := range line {
		if r == '\t' {
			spaces := 2 - column%2
			expanded.WriteString(strings.Repeat(" ", spaces))
			column += spaces
		} else {
			expanded.WriteRune(r)
			column++
		}
	}
	return expanded.String()
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

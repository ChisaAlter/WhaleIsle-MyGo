package markdown

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/theme"
)

// SearchMatch and SearchFile retain the original search result order and text.
type SearchMatch struct {
	LineNumber float64
	Line       string
}
type SearchFile struct {
	Path    string
	Matches []SearchMatch
}
type SearchData struct {
	Kind      string
	Paths     []string
	Files     []SearchFile
	Total     float64
	Truncated bool
}
type SearchLabels struct {
	PathsSummary                                    func(int, float64, bool) string
	MatchesSummary                                  func(int, float64, int, bool) string
	Copy, Copied, NoResults, Collapse, CollapseAria string
	Expand, ExpandAria                              func(int) string
}

// SearchNumber follows JavaScript's string conversion for validated nonnegative
// integer Numbers, including those outside the native int and safe-integer range.
func SearchNumber(value float64) string {
	if value == 0 {
		return "0"
	}
	if value < 1e21 {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	text := strconv.FormatFloat(value, 'e', -1, 64)
	at := strings.IndexByte(text, 'e')
	exponent := strings.TrimLeft(text[at+2:], "0")
	return text[:at] + "e+" + exponent
}

type searchRow struct {
	kind, path, line string
	number           float64
	count, index     int
	collapsed        bool
}

func searchRows(data SearchData, collapsed map[int]bool) []searchRow {
	rows := []searchRow{}
	if data.Kind == "paths" {
		for _, path := range data.Paths {
			rows = append(rows, searchRow{kind: "path", path: path})
		}
		return rows
	}
	for index, file := range data.Files {
		rows = append(rows, searchRow{kind: "file", path: file.Path, count: len(file.Matches), index: index, collapsed: collapsed[index]})
		if !collapsed[index] {
			for _, match := range file.Matches {
				rows = append(rows, searchRow{kind: "match", number: match.LineNumber, line: match.Line, index: index})
			}
		}
	}
	return rows
}
func searchCopyText(data SearchData) string {
	if data.Kind == "paths" {
		return strings.Join(data.Paths, "\n")
	}
	files := make([]string, len(data.Files))
	for i, file := range data.Files {
		lines := []string{file.Path}
		for _, match := range file.Matches {
			lines = append(lines, SearchNumber(match.LineNumber)+": "+match.Line)
		}
		files[i] = strings.Join(lines, "\n")
	}
	return strings.Join(files, "\n\n")
}
func searchVisibleRows(rows []searchRow, maxLines int, expanded bool) (head, tail []searchRow, hidden int) {
	hidden = len(rows) - maxLines
	if hidden <= 0 || expanded {
		return rows, nil, hidden
	}
	head = rows[:(maxLines+1)/2]
	tail = rows[len(rows)-maxLines/2:]
	if len(tail) > 0 && tail[0].kind == "match" {
		index := tail[0].index
		found := false
		for _, row := range head {
			if row.kind == "file" && row.index == index {
				found = true
				break
			}
		}
		if !found {
			for _, row := range rows {
				if row.kind == "file" && row.index == index {
					tail = append([]searchRow{row}, tail[1:]...)
					break
				}
			}
		}
	}
	return head, tail, hidden
}

// ToolSearchBlock draws the installed paths/grouped-matches search preset. File
// collapse and card fold are independent; clipboard text always covers all data.
func ToolSearchBlock(c *ui.Context, data SearchData, maxLines int, st Style, labels SearchLabels) ui.Element {
	root := ui.Column(c.Key("search-card")).MinWidth(0).Shrink(0).Radius(16).Background(st.CodeBG).Clip()
	expanded := ui.Local(root, "expanded", func() bool { return false })
	collapsed := ui.Local(root, "files", func() map[int]bool { return map[int]bool{} })
	copiedUntil := ui.Local(root, "copied-until", func() time.Time { return time.Time{} })
	copied := c.Now().Before(*copiedUntil)
	if copied {
		c.After(copiedUntil.Sub(c.Now()))
	}
	rows := searchRows(data, *collapsed)
	shown := len(data.Paths)
	if data.Kind == "matches" {
		shown = 0
		for _, file := range data.Files {
			shown += len(file.Matches)
		}
	}
	caption := labels.PathsSummary(shown, data.Total, data.Truncated)
	if data.Kind == "matches" {
		caption = labels.MatchesSummary(shown, data.Total, len(data.Files), data.Truncated)
	}
	prefs := theme.RenderPreferences()
	codeSize := max(float32(1), prefs.Code.Size-2)
	codeHeight := max(float32(1), prefs.Code.Size+6)
	rowHeight := max(float32(22), codeHeight)
	const headerSize, headerHeight = 13, 20
	tertiary := st.ToolCodeLanguage
	if tertiary.A == 0 {
		tertiary = st.Muted
	}
	banner := st.CodeBanner
	if banner.A == 0 {
		banner = st.CodeBG
	}
	head, tail, hidden := searchVisibleRows(rows, maxLines, *expanded)
	root.Children(func() {
		ui.Row(c.Key("banner")).MinWidth(0).Shrink(0).Padding(9, 14, 9, 14).Gap(12).AlignItems(ui.Center).Background(banner).Children(func() {
			ui.Text(c, caption).Font(prefs.Interface.Family).FontSize(headerSize).FixedLineHeight(headerHeight).TextColor(st.Muted).SingleLine().Grow(1).MinWidth(0)
			if len(rows) > 0 {
				label := labels.Copy
				if copied {
					label = labels.Copied
				}
				button := ui.ButtonBase(c.Key("copy")).Padding(0).Shrink(0).Background(ui.Transparent).Label(label).Children(func() {
					ui.Text(c, label).Font(prefs.Interface.Family).FontSize(headerSize).FixedLineHeight(headerHeight).TextColor(st.Muted).NoWrap().PassThrough()
				})
				if button.Clicked() && !copied {
					c.WriteClipboard(searchCopyText(data))
					*copiedUntil = c.Now().Add(time.Second)
					c.Invalidate()
				}
			}
		})
		if len(rows) == 0 {
			ui.Text(c, labels.NoResults).Font(prefs.Code.Family).FontSize(codeSize).FixedLineHeight(codeHeight).TextColor(tertiary).Padding(12, 14, 12, 14).MinWidth(0)
			return
		}
		ui.Column(c.Key("body")).ScrollOverflowX().MinWidth(0).Shrink(0).Padding(8, 14, 12, 0).Children(func() {
			paint := func(row searchRow) {
				switch row.kind {
				case "path":
					width, _ := c.MeasureText(0, ui.Span{Text: row.path, Font: prefs.Code.Family, Size: codeSize})
					ui.Text(c, row.path).Font(prefs.Code.Family).FontSize(codeSize).FixedLineHeight(codeHeight).MinHeight(rowHeight).Padding(0, 0, 0, 14).TextColor(st.FG).NoWrap().MinWidth(width + 14).Shrink(0).Selectable()
				case "match":
					prefix := SearchNumber(row.number) + ": "
					width, _ := c.MeasureText(0, ui.Span{Text: prefix + row.line, Font: prefs.Code.Family, Size: codeSize})
					ui.RichText(c, ui.Span{Text: prefix, Color: tertiary}, ui.Span{Text: row.line, Color: st.FG}).Font(prefs.Code.Family).FontSize(codeSize).FixedLineHeight(codeHeight).MinHeight(rowHeight).Padding(0, 0, 0, 14).NoWrap().MinWidth(width + 14).Shrink(0).Selectable()
				case "file":
					name := row.path + " " + strconv.Itoa(row.count)
					button := ui.ButtonBase(c.Key("file:"+strconv.Itoa(row.index))).FillWidth().MinWidth(0).MinHeight(rowHeight).Shrink(0).Padding(0, 14, 0, 14).Gap(8).Justify(ui.Start).AlignItems(ui.Start).Background(ui.Transparent).Role(ui.RoleDisclosure).Expanded(!row.collapsed).Label(name)
					if button.Clicked() {
						(*collapsed)[row.index] = !(*collapsed)[row.index]
						c.Invalidate()
					}
					button.Children(func() {
						pathWidth, _ := c.MeasureText(0, ui.Span{Text: row.path, Font: prefs.Code.Family, Size: codeSize, Weight: 600})
						ui.Text(c, row.path).Font(prefs.Code.Family).FontSize(codeSize).FixedLineHeight(codeHeight).FontWeight(600).TextColor(st.FG).NoWrap().MinWidth(pathWidth).Shrink(0).PassThrough()
						ui.Text(c, strconv.Itoa(row.count)).Font(prefs.Code.Family).FontSize(codeSize).FixedLineHeight(codeHeight).TextColor(tertiary).NoWrap().Shrink(0).PassThrough()
					})
				}
			}
			for _, row := range head {
				paint(row)
			}
			if hidden > 0 {
				caption, aria := labels.Expand(hidden), labels.ExpandAria(hidden)
				if *expanded {
					caption, aria = labels.Collapse, labels.CollapseAria
				}
				button := ui.ButtonBase(c.Key("fold")).FillWidth().MinWidth(0).Shrink(0).Padding(0, 14, 0, 14).Background(ui.Transparent).Justify(ui.Start).Role(ui.RoleDisclosure).Expanded(*expanded).Label(aria)
				if button.Clicked() {
					*expanded = !*expanded
					c.Invalidate()
				}
				ink := tertiary
				if button.Hovered() {
					ink = st.Muted
				}
				button.Children(func() {
					ui.Text(c, caption).Font(prefs.Code.Family).FontSize(codeSize).FixedLineHeight(codeHeight).TextColor(ink).NoWrap().PassThrough()
				})
			}
			for _, row := range tail {
				paint(row)
			}
		})
	})
	return root
}

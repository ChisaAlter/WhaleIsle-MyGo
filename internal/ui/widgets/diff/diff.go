// Package diff renders unified diff text for chat and file-patch tool results.
package diff

import (
	"strings"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Style controls diff line typography and colors.
type Style struct {
	FontSize float32
	FG       ui.Color
	Add      ui.Color
	Del      ui.Color
	Hunk     ui.Color
	File     ui.Color
	Muted    ui.Color
}

// IsUnifiedPatch reports whether s looks like a unified diff or patch hunk.
func IsUnifiedPatch(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "@@ ") || strings.Contains(s, "\n@@ ") {
		return true
	}
	hasOld := strings.Contains(s, "\n--- ") || strings.HasPrefix(s, "--- ")
	hasNew := strings.Contains(s, "\n+++ ") || strings.HasPrefix(s, "+++ ")
	return hasOld && hasNew
}

// ExtractPatchText pulls unified diff text from a message body or fenced block.
func ExtractPatchText(body string) string {
	body = strings.TrimSpace(body)
	if IsUnifiedPatch(body) {
		return body
	}
	const fence = "```"
	if i := strings.Index(body, fence); i >= 0 {
		rest := body[i+len(fence):]
		if j := strings.Index(rest, "\n"); j >= 0 {
			rest = rest[j+1:]
		}
		if k := strings.Index(rest, fence); k >= 0 {
			inner := strings.TrimSpace(rest[:k])
			if IsUnifiedPatch(inner) {
				return inner
			}
		}
	}
	return ""
}

// Spans turns unified diff text into colored RichText spans.
func Spans(source string, st Style) []ui.Span {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil
	}
	size := theme.CurrentPreferences().Code.Size
	if size <= 0 {
		size = 12
	}
	fontFamily := theme.RenderPreferences().Code.Family
	fg := st.FG
	add := st.Add
	del := st.Del
	hunk := st.Hunk
	file := st.File
	if file.R == 0 && file.G == 0 && file.B == 0 {
		file = st.Muted
	}
	out := make([]ui.Span, 0, strings.Count(source, "\n")+1)
	for _, line := range strings.Split(source, "\n") {
		color := fg
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			color = file
		case strings.HasPrefix(line, "@@"):
			color = hunk
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			color = add
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			color = del
		}
		out = append(out, ui.Span{Text: line + "\n", Size: size, Color: color, Font: fontFamily})
	}
	return out
}

type diffSpanCache struct {
	ready  bool
	source string
	style  Style
	font   string
	size   float32
	spans  []ui.Span
}

// View renders unified diff inside RichText.
func View(c *ui.Context, source string, st Style) {
	fontFamily := theme.RenderPreferences().Code.Family
	codeSize := theme.CurrentPreferences().Code.Size
	host := ui.Box(c).FillWidth().MinWidth(0)
	cache := ui.Local(host, "diff-spans-cache", func() diffSpanCache { return diffSpanCache{} })
	if !cache.ready || cache.source != source || cache.style != st || cache.font != fontFamily || cache.size != codeSize {
		patch := ExtractPatchText(source)
		if patch == "" {
			patch = source
		}
		cache.spans = Spans(patch, st)
		cache.source = source
		cache.style = st
		cache.font = fontFamily
		cache.size = codeSize
		cache.ready = true
	}
	if len(cache.spans) == 0 {
		return
	}
	host.Children(func() {
		ui.RichText(c, cache.spans...).Font(fontFamily).FontSize(codeSize).TextColor(st.FG).Selectable()
	})
}

// LineDiff produces a unified-style patch string between a and b using go-diff.
func LineDiff(a, b string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	patches := dmp.PatchMake(a, diffs)
	return dmp.PatchToText(patches)
}

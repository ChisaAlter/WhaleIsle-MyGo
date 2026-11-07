package files

import (
	"testing"

	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/widgets/markdown"
)

func TestResolveEditViewMode(t *testing.T) {
	editViewModeFallback = EditViewSplit
	m := Model{}
	if got := resolveEditViewMode(m, false); got != EditViewEdit {
		t.Fatalf("no highlight: got %v", got)
	}
	if got := resolveEditViewMode(m, true); got != EditViewSplit {
		t.Fatalf("default prefer split: got %v", got)
	}

	mode := EditViewHighlight
	m.EditViewMode = &mode
	if got := resolveEditViewMode(m, true); got != EditViewHighlight {
		t.Fatalf("pointer: got %v", got)
	}
	if got := resolveEditViewMode(m, false); got != EditViewEdit {
		t.Fatalf("force edit when not highlightable: got %v", got)
	}
}

func TestSetEditViewModeRemembers(t *testing.T) {
	editViewModeFallback = EditViewSplit
	m := Model{}
	setEditViewMode(m, EditViewEdit)
	if editViewModeFallback != EditViewEdit || resolveEditViewMode(m, true) != EditViewEdit {
		t.Fatal("fallback should remember edit")
	}

	mode := EditViewSplit
	m.EditViewMode = &mode
	setEditViewMode(m, EditViewHighlight)
	if mode != EditViewHighlight || resolveEditViewMode(m, true) != EditViewHighlight {
		t.Fatal("pointer should remember highlight")
	}
}

func TestCycleEditViewMode(t *testing.T) {
	if got := cycleEditViewMode(EditViewSplit); got != EditViewEdit {
		t.Fatalf("split→edit: %v", got)
	}
	if got := cycleEditViewMode(EditViewEdit); got != EditViewHighlight {
		t.Fatalf("edit→highlight: %v", got)
	}
	if got := cycleEditViewMode(EditViewHighlight); got != EditViewSplit {
		t.Fatalf("highlight→split: %v", got)
	}
}

func TestPreferSideBySide(t *testing.T) {
	if preferSideBySide(0) {
		t.Fatal("zero width should stack")
	}
	if preferSideBySide(sideBySideMinWidth - 1) {
		t.Fatal("narrow should stack")
	}
	if !preferSideBySide(sideBySideMinWidth) {
		t.Fatal("at threshold should side-by-side")
	}
	if !preferSideBySide(800) {
		t.Fatal("wide should side-by-side")
	}
}

func TestLanguageChipLabel(t *testing.T) {
	if got := languageChipLabel("pkg/util.go"); got != "go" {
		t.Fatalf("go: %q", got)
	}
	if got := languageChipLabel("app.py"); got != "python" {
		t.Fatalf("python: %q", got)
	}
	if got := languageChipLabel("README"); got != "text" {
		t.Fatalf("no ext: %q", got)
	}
}

func TestEditViewModeLabels(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })

	i18n.SetLocale("en")
	if got := editViewModeLabel(EditViewSplit); got != "Split preview" {
		t.Fatalf("en split: %q", got)
	}
	if got := editViewModeLabel(EditViewEdit); got != "Edit only" {
		t.Fatalf("en edit: %q", got)
	}
	if got := editViewModeLabel(EditViewHighlight); got != "Highlight only" {
		t.Fatalf("en highlight: %q", got)
	}
	if got := editViewModeCycleLabel(EditViewSplit); got != "Next: Edit only" {
		t.Fatalf("en cycle: %q", got)
	}

	i18n.SetLocale("zh")
	if got := editViewModeLabel(EditViewSplit); got != "分屏预览" {
		t.Fatalf("zh split: %q", got)
	}
	if got := editViewModeCycleLabel(EditViewHighlight); got != "切换: 分屏预览" {
		t.Fatalf("zh cycle: %q", got)
	}
}

func TestEditHighlightWiresSourceLanguage(t *testing.T) {
	rel := "pkg/util.go"
	draft := "package util\n\nfunc X() {}\n"
	if !markdown.CanSourceHighlight(rel, draft) {
		t.Fatal("expected highlightable draft")
	}
	if got := markdown.LanguageFromPath(rel); got != "go" {
		t.Fatalf("lang=%q", got)
	}
	st := markdown.Style{FontSize: 12}
	spans := markdown.SourceSpans(rel, draft, st)
	if len(spans) == 0 {
		t.Fatal("SourceView wiring: expected spans for draft")
	}
}

func TestFilesSidebarDeliverablesOpenInAppLocalesParity(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })

	i18n.SetLocale("zh")
	if got := FilesLocale("unsaved.body"); got != "关闭前可以保存修改，或继续编辑。" {
		t.Fatalf("zh unsaved.body = %q", got)
	}
	if got := SidebarFilesLocale("type.label"); got != "文件" {
		t.Fatalf("zh type.label = %q", got)
	}
	if got := DeliverablesLocale("row.title"); got != "交付文件" {
		t.Fatalf("zh row.title = %q", got)
	}
	if got := OpenInAppLocaleFormat("open.title", map[string]string{"app": "VS Code"}); got != "用 VS Code 打开" {
		t.Fatalf("zh open.title = %q", got)
	}
	if got := DocumentPreviewLocale("office.missingFontsTitle"); got != "缺失的字体" {
		t.Fatalf("zh office.missingFontsTitle = %q", got)
	}

	i18n.SetLocale("en")
	if got := FilesLocale("unsaved.body"); got != "Save your changes before closing, or keep editing." {
		t.Fatalf("en unsaved.body = %q", got)
	}
	if got := SidebarFilesLocale("type.label"); got != "Files" {
		t.Fatalf("en type.label = %q", got)
	}
	if got := DeliverablesLocale("row.title"); got != "Present files" {
		t.Fatalf("en row.title = %q", got)
	}
	if got := OpenInAppLocaleFormat("open.title", map[string]string{"app": "VS Code"}); got != "Open in VS Code" {
		t.Fatalf("en open.title = %q", got)
	}
	if got := DocumentPreviewLocale("office.missingFontsTitle"); got != "Missing fonts" {
		t.Fatalf("en office.missingFontsTitle = %q", got)
	}
}

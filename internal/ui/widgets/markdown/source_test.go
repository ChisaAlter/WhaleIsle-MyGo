package markdown

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSourceSpansGo(t *testing.T) {
	st := Style{FontSize: 12, FG: ui.RGB(0, 0, 0), CodeBG: ui.RGB(240, 240, 240)}
	spans := SourceSpans("main.go", "package main\n", st)
	if len(spans) == 0 {
		t.Fatal("expected spans")
	}
}

func TestLanguageFromPath(t *testing.T) {
	cases := map[string]string{
		"main.go":           "go",
		`C:\src\app.py`:     "python",
		"lib/index.ts":      "typescript",
		"a.tsx":             "tsx",
		"script.js":         "javascript",
		"README.md":         "markdown",
		"config.yml":        "yaml",
		"main.rs":           "rust",
		"util.hpp":          "cpp",
		"run.ps1":           "powershell",
		"data.csv":          "csv",
		"noext":             "",
		"UPPER.GO":          "go",
	}
	for path, want := range cases {
		if got := LanguageFromPath(path); got != want {
			t.Fatalf("%q: got %q want %q", path, got, want)
		}
	}
}

func TestCanSourceHighlight(t *testing.T) {
	if !CanSourceHighlight("main.go", "package main\n") {
		t.Fatal("go source should highlight")
	}
	if CanSourceHighlight("data.csv", "a,b\n") {
		t.Fatal("csv should not use chroma highlight")
	}
	if CanSourceHighlight("main.go", "") {
		t.Fatal("empty code should not highlight")
	}
	huge := make([]byte, maxSourceHighlightBytes)
	if CanSourceHighlight("main.go", string(huge)) {
		t.Fatal("huge buffer should skip highlight")
	}
}

func TestSourceSpansUsesLanguageFromPath(t *testing.T) {
	st := Style{FontSize: 12, FG: ui.RGB(0, 0, 0), CodeBG: ui.RGB(240, 240, 240)}
	py := SourceSpans("app.py", "def hello():\n    return 1\n", st)
	if len(py) == 0 {
		t.Fatal("expected python spans")
	}
	md := SourceSpans("README.md", "# Title\n", st)
	if len(md) == 0 {
		t.Fatal("expected markdown spans")
	}
}

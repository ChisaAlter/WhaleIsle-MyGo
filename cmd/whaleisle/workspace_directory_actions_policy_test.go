package main

import (
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestProjectDirectoryContributionAvailabilityLabelsRefusalAndPickerScope(t *testing.T) {
	locale := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(locale) })
	state := func(path string) (bool, bool) { return path == "/active", path == "/refused" }
	if got := projectWorkspaceDirectoryActions(false, state, func(string) {}); len(got) != 0 {
		t.Fatal("unavailable Project contributed a directory action")
	}
	for _, language := range []string{i18n.LocaleZH, i18n.LocaleEN} {
		i18n.SetLocale(language)
		var selected string
		actions := projectWorkspaceDirectoryActions(true, state, func(path string) { selected = path })
		if len(actions) != 1 {
			t.Fatalf("Project contribution count=%d", len(actions))
		}
		action := actions[0]
		for _, path := range []string{"/active", "/new", "/refused"} {
			if _, visible := action.Presentation(path, true); visible {
				t.Fatal("row-only Project contribution leaked into the generic picker")
			}
		}
		if _, visible := action.Presentation("/refused", false); visible {
			t.Fatal("explicit directory refusal was not hidden")
		}
		for _, path := range []string{"/active", "/new"} {
			want := "Create Project in this directory"
			if language == i18n.LocaleZH {
				want = "在此目录新建 Project"
			}
			if path == "/active" {
				want = "Open Project"
				if language == i18n.LocaleZH {
					want = "打开 Project"
				}
			}
			label, visible := action.Presentation(path, false)
			if !visible || label != want {
				t.Fatalf("%s path=%s label=%q visible=%v", language, path, label, visible)
			}
			action.Run(path)
			if selected != path {
				t.Fatal("directory action replaced the explicit path")
			}
		}
	}
}

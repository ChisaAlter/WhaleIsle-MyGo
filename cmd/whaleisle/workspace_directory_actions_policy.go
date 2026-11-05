package main

import (
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/workspacepicker"
)

// Project's published directory contribution is row-only. State is a pure
// projection of its listing and directory/check cache, not a render-time RPC.
func projectWorkspaceDirectoryActions(available bool, state func(path string) (active, refused bool), run func(string)) []workspacepicker.DirectoryAction {
	if !available {
		return nil
	}
	picker := false
	return []workspacepicker.DirectoryAction{{ID: "project.create", Order: 20, Picker: &picker, Run: run,
		Visible: func(path string) bool { _, refused := state(path); return !refused },
		LabelForPath: func(path string) string {
			active, _ := state(path)
			if i18n.Locale() == i18n.LocaleZH {
				if active {
					return "打开 Project"
				}
				return "在此目录新建 Project"
			}
			if active {
				return "Open Project"
			}
			return "Create Project in this directory"
		}}}
}

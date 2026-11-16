package main

import (
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/projects"
)

func TestProjectModelsKeepTheirFrameTheme(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			client := &dsh.Client{}
			state := &mainState{dshClient: client, themeID: "different-state-theme"}
			parent := frame.Model{ThemeID: "frame-theme", SystemDark: dark}
			state.bindProjectModel(nil, &parent)
			project := parent.Projects
			if project.ThemeID != "frame-theme" || project.SystemDark != dark || project.VisualPage == nil {
				t.Fatal("Project did not bind with the parent frame theme")
			}
			// An observer belongs to this rendered frame even if the caller
			// starts preparing the next frame before invoking its callback.
			parent.ThemeID, parent.SystemDark = "next-frame", !dark
			observer := project.ObserverModel(&projects.ObserverState{Section: "tasks"})
			if observer.ThemeID != "frame-theme" || observer.SystemDark != dark || observer.VisualPage == nil {
				t.Fatal("observer lost its captured frame theme")
			}
		})
	}
}

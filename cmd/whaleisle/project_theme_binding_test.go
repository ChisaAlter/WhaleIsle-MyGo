package main

import (
	"testing"

	"github.com/egoist/mygo"
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
			state.projectFlow.client = client
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

func TestProjectVisualPageUsesBoundModelAppearance(t *testing.T) {
	for _, dark := range []bool{false, true} {
		s, card, _ := projectCardOwnershipFixture(t)
		reference := card.Delivery.Pages[0]
		owner, ok := s.projectVisualOwnerLocked("request:A", reference)
		if !ok {
			t.Fatal("fixture publication was not admitted")
		}
		// A retained page with no loaded HTML exercises appearance updates
		// without starting a reader, helper or native surface.
		page := &projectVisualPage{width: 320, dark: !dark, state: projects.VisualPagePreview{Available: true}}
		s.projectFlow.visualResults = &projectVisualResultsState{pages: map[projectVisualOwner]*projectVisualPage{owner: page}}
		model := projects.Model{SystemDark: dark}
		s.bindProjectVisualResults(&mygo.Window{}, &model)
		preview := model.VisualPage("request:A", reference, 320)
		if !preview.Available || page.dark != dark {
			t.Fatal("retained publication did not use the bound model appearance")
		}
	}
}

package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/settings"
)

// Closing the settings modal clears settingsEmbeddedSection. The embedded
// section content must keep the section it was built for: reading the live
// state from the callback painted the chat behind the dialog and reused the
// transcript ListState (panic: ListState shown by two Lists).
func TestEmbeddedSectionContentKeepsTheSectionItWasBuiltFor(t *testing.T) {
	s := mainState{settingsEmbeddedSection: settings.SectionRemote}
	got := settings.SectionID("")
	m := s.settingsSectionModel(settings.Model{}, func(_ *ui.Context, section settings.SectionID) {
		got = section
	})
	if m.SectionContent == nil {
		t.Fatal("remote section did not install content")
	}
	s.settingsEmbeddedSection = "" // OnClose cleared it before the frame painted
	m.SectionContent(&ui.Context{})
	if got != settings.SectionRemote {
		t.Fatalf("content painted section %q after close, want remote", got)
	}
}

package settings

import (
	"fmt"
	fonttheme "whaleisle/internal/ui/theme"

	"github.com/egoist/mygo/ui"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// Keep an actual General namespace reachable if supplied by a Host. Older
// Hosts expose only ui-conversation; use that as the General landing page.
func generalNamespace(snap store.ViewState) string {
	for _, ns := range snap.Namespaces {
		if ns.NS == "general" {
			return ns.NS
		}
	}
	for _, ns := range snap.Namespaces {
		if ns.NS == store.CustomInstructionsNamespace {
			return ns.NS
		}
	}
	return "general"
}

func customInstructionsRow(c *ui.Context, m Model, snap store.ViewState, fg, muted ui.Color) {
	if m.DraftPtr == nil {
		return
	}
	draft := m.DraftPtr(store.CustomInstructionsField, snap.Drafts[store.CustomInstructionsField])
	ui.Column(c).Gap(4).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyCustomInstructionsTitle)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyCustomInstructionsScope)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		area := ui.TextArea(c, draft).FillWidth().Lines(4, 10).Disabled(!snap.Writable || !snap.ClientReady)
		if area.Changed() && m.OnDraft != nil {
			m.OnDraft(store.CustomInstructionsField, *draft)
		}
		count := store.CustomInstructionsLength(*draft)
		ui.Text(c, fmt.Sprintf("%d / %d", count, store.CustomInstructionsMaxLength)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if count > store.CustomInstructionsMaxLength {
			ui.Text(c, i18n.T(i18n.KeyCustomInstructionsLimit)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg)
		}
	})
}

func customInstructionsInvalid(snap store.ViewState) bool {
	return snap.SelectedNS == store.CustomInstructionsNamespace && store.CustomInstructionsLength(snap.Drafts[store.CustomInstructionsField]) > store.CustomInstructionsMaxLength
}

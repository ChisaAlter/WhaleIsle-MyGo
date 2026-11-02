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

func generalCustomInstructions(c *ui.Context, a *BodySettingsAccess, fg, muted ui.Color) {
	_, value, found := bodyHostValue(a, store.CustomInstructionsNamespace, store.CustomInstructionsField)
	if !found {
		return
	}
	anchor := ui.Column(c.Key(fmt.Sprintf("general-custom-instructions-%d", a.Owner))).FillWidth().MinWidth(0).Gap(10).Padding(16, 0)
	type draftState struct{ Value, Base string }
	draft := ui.Local(anchor, "draft", func() draftState { return draftState{value, value} })
	if draft.Value == value || draft.Value == draft.Base && draft.Base != value {
		draft.Value, draft.Base = value, value
	}
	dark := c.Theme().Dark
	colors := fonttheme.ResolveWithSystem(fonttheme.IDSystem, dark).Active(dark)
	well := ui.RGB(255, 255, 255)
	if dark {
		well = fonttheme.Color(colors.Field)
	}
	anchor.Children(func() {
		// 0.3.13: title and one-line explanation, a 12 DIP textarea on the
		// raised layer at 80%, and the counter under its right edge.
		ui.Column(c).FillWidth().MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyCustomInstructionsTitle)).FillWidth().FontSize(fonttheme.InterfaceSize(14)).FixedLineHeight(fonttheme.InterfaceSize(22)).TextColor(fg)
			ui.Text(c, providerCopy("给智能体定几条规则，后续对所有任务都生效", "Give the agent a few rules that apply to every later task")).FillWidth().FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
		})
		area := ui.TextArea(c, &draft.Value).FillWidth().Lines(6, 10).Radius(12).Border(0.667, fonttheme.Color(colors.BorderL4)).Background(well.Alpha(.8)).Padding(12, 14).FontSize(fonttheme.InterfaceSize(14)).
			Label(i18n.T(i18n.KeyCustomInstructionsTitle)).Disabled(!a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading)
		if area.Changed() && a.OnChange != nil {
			draft.Value = store.LimitCustomInstructions(draft.Value)
			if store.CustomInstructionsLength(draft.Value) <= store.CustomInstructionsMaxLength {
				a.OnChange(store.CustomInstructionsNamespace, store.CustomInstructionsField, draft.Value)
			}
		}
		ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
			ui.Text(c, fmt.Sprintf("%d / %d", store.CustomInstructionsLength(draft.Value), store.CustomInstructionsMaxLength)).FontSize(fonttheme.InterfaceSize(12)).FixedLineHeight(fonttheme.InterfaceSize(18)).TextColor(muted)
		})
		if store.CustomInstructionsLength(draft.Value) > store.CustomInstructionsMaxLength {
			ui.Text(c, i18n.T(i18n.KeyCustomInstructionsLimit)).FillWidth().TextColor(fg)
		}
	})
}

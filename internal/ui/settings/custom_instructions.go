package settings

import (
	"fmt"

	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
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
	ui.Column(c).Gap(design.Sp(1)).Children(func() {
		kLabel(c, i18n.T(i18n.KeyCustomInstructionsTitle), fg)
		kHint(c, i18n.T(i18n.KeyCustomInstructionsScope))
		count := store.CustomInstructionsLength(*draft)
		over := count > store.CustomInstructionsMaxLength
		area := kTextarea(c, draft, coss.TextareaOptions{
			Label:    i18n.T(i18n.KeyCustomInstructionsTitle),
			MaxLines: 10,
			Disabled: !snap.Writable || !snap.ClientReady,
			Invalid:  over,
		}).Lines(4, 10)
		if area.Changed() && m.OnDraft != nil {
			m.OnDraft(store.CustomInstructionsField, *draft)
		}
		kHint(c, fmt.Sprintf("%d / %d", count, store.CustomInstructionsMaxLength))
		if over {
			kError(c, i18n.T(i18n.KeyCustomInstructionsLimit))
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
	clearFailure := bodyPreferenceFieldError(c, a, store.CustomInstructionsNamespace, store.CustomInstructionsField, i18n.T(i18n.KeyCustomInstructionsTitle))
	anchor := ui.Column(c.Key(fmt.Sprintf("general-custom-instructions-%d", a.Owner))).FillWidth().MinWidth(0).Gap(design.Sp(2.5)).Padding(design.Sp(4), 0)
	settingsSearchTarget(c, "custom-instructions", i18n.T(i18n.KeyCustomInstructionsTitle), anchor)
	type draftState struct{ Value, Base string }
	draft := ui.Local(anchor, "draft", func() draftState { return draftState{value, value} })
	if draft.Value == value || draft.Value == draft.Base && draft.Base != value {
		draft.Value, draft.Base = value, value
	}
	anchor.Children(func() {
		ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(1)).Children(func() {
			kLabel(c, i18n.T(i18n.KeyCustomInstructionsTitle), fg).FillWidth()
			kHint(c, providerCopy("给智能体定几条规则，后续对所有任务都生效", "Give the agent a few rules that apply to every later task"))
		})
		over := store.CustomInstructionsLength(draft.Value) > store.CustomInstructionsMaxLength
		area := kTextarea(c, &draft.Value, coss.TextareaOptions{
			Label:       i18n.T(i18n.KeyCustomInstructionsTitle),
			Placeholder: i18n.T(i18n.KeyCustomInstructionsPlaceholder),
			MaxLines:    10,
			Disabled:    !a.Snapshot.ClientReady || !a.Snapshot.Writable || a.Snapshot.Loading,
			Invalid:     over,
		}).Lines(6, 10)
		if area.Changed() && a.OnChange != nil {
			draft.Value = store.LimitCustomInstructions(draft.Value)
			if store.CustomInstructionsLength(draft.Value) <= store.CustomInstructionsMaxLength {
				clearFailure()
				a.OnChange(store.CustomInstructionsNamespace, store.CustomInstructionsField, draft.Value)
			}
		}
		ui.Row(c).FillWidth().Justify(ui.End).Children(func() {
			kText(c, fmt.Sprintf("%d / %d", store.CustomInstructionsLength(draft.Value), store.CustomInstructionsMaxLength), design.TextXS, design.WeightNormal, muted)
		})
		if store.CustomInstructionsLength(draft.Value) > store.CustomInstructionsMaxLength {
			kError(c, i18n.T(i18n.KeyCustomInstructionsLimit))
		}
	})
	bodyPreferenceFieldAlert(c, a, store.CustomInstructionsNamespace, store.CustomInstructionsField, i18n.T(i18n.KeyCustomInstructionsTitle))
}

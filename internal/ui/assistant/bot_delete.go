package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func botDeleteConfirmLabel() string {
	return i18n.T(i18n.KeyMiscAssistantPermanentlyDeleteReviewedBot)
}
func botDeleteDangerAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIConfirmIrreversibleDeletionNow)
}
func botDeleteEffectsWarning() string {
	return i18n.T(i18n.KeyW4AssistantDangerousActionDeletesTheTargetSDataA)
}

type BotDeleteModel struct {
	State                                    dshbotstore.BotDeleteSnapshot
	HostLabel, SourceID                      string
	OnRefresh, OnReview, OnConfirm, OnCancel func()
	OnIdentity                               func(id, name string)
	OnAcknowledge                            func(bool)
}

func botDeleteBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.BotDelete.OnRefresh == nil {
		return
	}
	botDeleteView(c, m.BotDelete, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func botDeleteReviewText(host string, p pluginrpc.DshbotBotDeletePlan) string {
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantDeletePreviewDetail), host, p.Catalog.Revision, p.Target.Kind, p.Target.ID, p.Target.Name, p.Target.Title, p.Target.SessionID, string(p.Impact))
}
func botDeleteReviewConfirmed(r *dshbotstore.BotDeleteReview) bool {
	return r != nil && r.Acknowledged && r.TypedID == r.Plan.Target.ID && r.TypedName == r.Plan.Target.Name && r.TypedID != "" && r.TypedName != ""
}
func botDeleteView(c *ui.Context, m BotDeleteModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	card := ui.Column(c.Key("native-bot-delete")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
	chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
	card.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantDeleteSelectedBotOrRoom)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantReadTheFullImpactFirstCancelIsTheDef)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		widgets.Notice(c, "bot_delete:56", "", s.Notice != "", func() string { return s.Notice })
		for _, w := range s.Warnings {
			ui.Text(c, "Host warning: "+w).TextColor(accent).Selectable()
		}
		if s.Completed {
			ui.Text(c, botDeleteReviewText(m.HostLabel, s.Plan)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			return
		}
		if m.OnRefresh != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReadDeletionPreviewOrOutcome)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || s.Scope == "" || m.SourceID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Uncertain {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheOriginalAttemptStaysUnknownAndCanno)).TextColor(accent).Selectable()
			if s.Pending != nil {
				ui.Text(c, botDeleteReviewText(m.HostLabel, s.Pending.Plan)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			}
			if s.Observation != "" {
				ui.Text(c, s.Observation).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			}
			return
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantFullCurrentPreviewAndAuthorityRequi)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable4)).TextColor(accent)
		}
		if s.Review == nil {
			ui.Text(c, botDeleteReviewText(m.HostLabel, s.Plan)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			if m.OnReview != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewDestructiveDeletion)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Writable).Clicked() {
				m.OnReview()
			}
			return
		}
		r := s.Review
		effects, confirmLabel, cancelLabel, idLabel := botDeleteReviewedLabels(r.Plan.Target.Kind)
		ui.Column(c.Key("bot-delete-confirmation")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(8).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantDestructiveConfirmationCancelByDefau)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(accent)
			if m.OnCancel != nil && ui.Button(c, cancelLabel).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() {
				m.OnCancel()
			}
			ui.Text(c, botDeleteReviewText(m.HostLabel, r.Plan)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
			ui.Text(c, effects).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheNotepadDeletesAllCurrentContentByT)).FontSize(fonttheme.InterfaceSize(11)).TextColor(accent)
			name := r.TypedName
			ui.TextInput(c.Key("bot-delete-exact-name"), &name).Label(i18n.T(i18n.KeyMiscAssistantTypeExactStableName)).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(blocked).OnChange(func() {
				if m.OnIdentity != nil {
					m.OnIdentity(r.TypedID, name)
				}
			})
			id := r.TypedID
			ui.TextInput(c.Key("bot-delete-exact-id"), &id).Label(idLabel).FillWidth().Height(32).Radius(8).Border(.5, border).Disabled(blocked).OnChange(func() {
				if m.OnIdentity != nil {
					m.OnIdentity(id, r.TypedName)
				}
			})
			ack := r.Acknowledged
			if ui.Checkbox(c, &ack, botDeleteDangerAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
				m.OnAcknowledge(ack)
			}
			if m.OnConfirm != nil && ui.Button(c, confirmLabel).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Writable || !botDeleteReviewConfirmed(r)).Clicked() {
				m.OnConfirm()
			}
		})
	})
}

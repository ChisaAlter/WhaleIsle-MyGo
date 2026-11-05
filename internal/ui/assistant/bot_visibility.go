package assistant

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"sort"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
)

func botVisibilityEffects() string {
	return i18n.T(i18n.KeyW4AssistantHidingOnlyChangesVisibilityInTheContac)
}

type BotVisibilityModel struct {
	State                          dshbotstore.BotVisibilitySnapshot
	HostLabel                      string
	OnRefresh, OnConfirm, OnCancel func()
	OnShowHidden                   func(bool)
	OnReview, OnAdopt              func(id, session string)
	OnAcknowledge                  func(bool)
}

func visibilityKind(kind string) string {
	if kind == "room" {
		return i18n.T(i18n.KeyMiscAssistantRoom)
	}
	return "Bot"
}
func visibilityIdentity(v pluginrpc.DshbotBotItem) string {
	return fmt.Sprintf("%s · %s · %s\nID: %s · canonical session: %s · hidden=%t", visibilityKind(v.Kind), v.Name, v.Title, v.ID, v.SessionID, v.Hidden)
}
func botVisibilityReviewText(host string, cmd pluginrpc.DshbotVisibilityCommand) string {
	target, err := pluginrpc.DshbotVisibilityTarget(cmd)
	if err != nil {
		return err.Error()
	}
	keys := []string{}
	for k := range target.Profile {
		if k != "hidden" && k != "updatedAt" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantVisibilityDiff), host, cmd.Expected.Revision, visibilityIdentity(target), target.Hidden, cmd.Hidden, string(target.Profile["updatedAt"]), cmd.UpdatedAt, strings.Join(keys, ", "), len(cmd.Expected.Items))
}
func botVisibilityBlock(c *ui.Context, m Model, fg, muted, field, border, accent ui.Color) {
	if assiststore.NormalizeMode(m.Mode) != assiststore.ModeDshbot || m.BotVisibility.OnRefresh == nil {
		return
	}
	botVisibilityView(c, m.BotVisibility, m.Busy || m.PluginRPCBusy || !m.DshbotEnabled || !m.HarnessReady, fg, muted, field, border, accent)
}
func botVisibilityView(c *ui.Context, m BotVisibilityModel, blocked bool, fg, muted, field, border, accent ui.Color) {
	s := m.State
	blocked = blocked || s.Busy
	card := ui.Column(c.Key("bot-visibility-management")).Border(.5, border).Radius(12).Padding(12, 16).Gap(8)
	chrome.GlassSurface(card, field, chrome.MaterialPanel, 12)
	card.Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantVisibilityTitle)).FontSize(fonttheme.InterfaceSize(13)).FontWeight(600).TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantThisManagementBlockIsSeparateFromTheO)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		widgets.Notice(c, "bot_visibility:70", "", s.Notice != "", func() string { return s.Notice })
		if s.FocusID != "" {
			label := i18n.T(i18n.KeyMiscAssistantTargetAwaitsFreshCatalog)
			if s.Loaded {
				label = i18n.T(i18n.KeyMiscAssistantExactTargetNotFound)
				for _, v := range s.Catalog.Items {
					if v.ID == s.FocusID && v.SessionID == s.FocusSessionID {
						label = i18n.T(i18n.KeyMiscAssistantExactTarget) + visibilityIdentity(v)
						break
					}
				}
			}
			ui.Text(c, label+"\nID: "+s.FocusID+" · canonical session: "+s.FocusSessionID).TextColor(accent).Selectable()
		}
		if s.Result.ID != "" {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantLastConfirmedResult)+visibilityIdentity(s.Result)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
		}
		shown := s.ShowHidden
		if ui.Checkbox(c, &shown, i18n.T(i18n.KeyMiscAssistantShowHiddenLocally)).Changed() && m.OnShowHidden != nil {
			m.OnShowHidden(shown)
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReadFullCatalog)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || s.Scope == "").Clicked() && m.OnRefresh != nil {
			m.OnRefresh()
		}
		if s.Uncertain {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantTheOriginalWriteStaysUnknownAndCannot)).TextColor(accent).Selectable()
			if s.Pending != nil {
				ui.Text(c, botVisibilityReviewText(m.HostLabel, *s.Pending)).FontSize(fonttheme.InterfaceSize(11)).TextColor(fg).Selectable()
			}
			if s.Candidate != nil && ui.Button(c, i18n.T(i18n.KeyMiscAssistantAdoptExactReadback)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() && m.OnAdopt != nil {
				m.OnAdopt(s.Candidate.ID, s.Candidate.SessionID)
			}
			return
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyW4AssistantReadTheCurrentHostSFullCatalogAndWri)).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
			return
		}
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable4)).TextColor(accent)
		}
		if s.Review != nil {
			r := s.Review
			ui.Column(c.Key("visibility-confirm")).Border(.5, accent).Radius(12).Padding(12, 16).Gap(8).Children(func() {
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancel)).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked).Clicked() && m.OnCancel != nil {
					m.OnCancel()
				}
				ui.Text(c, botVisibilityReviewText(m.HostLabel, r.Command)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Text(c, botVisibilityEffects()).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent).Selectable()
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, i18n.T(i18n.KeyMiscAssistantIConfirmThisTargetAndEffects)).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				label := i18n.T(i18n.KeyMiscAssistantConfirmUnhide)
				if r.Command.Hidden {
					label = i18n.T(i18n.KeyMiscAssistantConfirmHide)
				}
				if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
					m.OnConfirm()
				}
			})
			return
		}
		count := 0
		for _, row := range s.Catalog.Items {
			if row.Hidden && !s.ShowHidden {
				continue
			}
			count++
			v := row
			ui.Column(c.Key("visibility-row-" + v.ID)).Gap(4).Children(func() {
				ui.Text(c, visibilityIdentity(v)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				label := i18n.T(i18n.KeyMiscAssistantReviewHide)
				if v.Hidden {
					label = i18n.T(i18n.KeyMiscAssistantReviewUnhide)
				}
				if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(fonttheme.InterfaceSize(12)).Disabled(blocked || !s.Writable).Clicked() && m.OnReview != nil {
					m.OnReview(v.ID, v.SessionID)
				}
			})
		}
		if count == 0 {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantNoMatchingItemsShowHiddenToRecover)).TextColor(muted)
		}
	})
}

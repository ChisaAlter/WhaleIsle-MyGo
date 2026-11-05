package assistant

import (
	"whaleisle/internal/ui/widgets"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"slices"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
	fonttheme "whaleisle/internal/ui/theme"
)

func botCapabilityAcknowledgement() string {
	return i18n.T(i18n.KeyMiscAssistantIUnderstandCapabilityAndObjectiveEx)
}

type BotCapabilityModel struct {
	State                                                dshbotstore.CapabilitySnapshot
	HostLabel                                            string
	OnRefresh, OnReview, OnConfirm, OnDismiss, OnDiscard func()
	OnChange                                             func(group, field, value string)
	OnAcknowledge                                        func(bool)
}

func botCapabilityProfileText(s pluginrpc.DshbotCapabilityStatus, key string) string {
	var value string
	if json.Unmarshal(s.Profile[key], &value) == nil {
		return value
	}
	return string(s.Profile[key])
}
func botCapabilityPolicyText(p pluginrpc.DshbotCapabilityPolicy) string {
	names := strings.Join(p.Names, ", ")
	if names == "" {
		names = "(none)"
	}
	return p.Mode + " · names: " + names
}
func botCapabilitiesBlock(c *ui.Context, m BotCapabilityModel, blocked bool, fg, muted, accent ui.Color) {
	if m.OnRefresh == nil {
		return
	}
	s := m.State
	blocked = blocked || s.Busy
	editable := !blocked && s.Loaded && s.Writable && s.Status.WriteError == "" && s.Status.Catalog.Active && s.Review == nil
	ui.Column(c.Key("bot-capabilities")).Gap(7).Children(func() {
		ui.Text(c, i18n.T(i18n.KeyMiscAssistantBotCapabilities)).FontSize(fonttheme.InterfaceSize(13)).Bold().TextColor(fg)
		ui.Text(c, i18n.T(i18n.KeyW4AssistantEditTheToolsSkillsMcpScopeTheHostAct)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantRefreshCapabilities)).Disabled(blocked || s.Scope.BotID == "").Clicked() {
			m.OnRefresh()
		}
		if s.Error != "" {
			ui.Text(c, s.Error).TextColor(accent).Selectable()
		}
		widgets.Notice(c, "bot_capabilities:58", "", s.Notice != "", func() string { return s.Notice })
		for _, warning := range s.Warnings {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantHostWarning)+warning).TextColor(accent).Selectable()
		}
		if !s.Loaded {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantReadBackBeforeSaving2)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		}
		if s.Status.Bot.ID == "" {
			return
		}
		ui.Text(c, fmt.Sprintf("Host: %s\nBot: %s (%s) · Session: %s · revision %d", m.HostLabel, s.Status.Bot.Name, s.Status.Bot.ID, s.Status.Bot.SessionID, s.Status.Revision)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
		if !s.Writable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantWriteAuthorityUnavailable2)).TextColor(accent)
		}
		if s.Status.WriteError != "" {
			ui.Text(c, s.Status.WriteError).TextColor(accent).Selectable()
		}
		if !s.Status.Catalog.Active {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantCapabilityServiceInactive)).TextColor(accent)
		}
		if !s.Status.Catalog.SkillsAvailable {
			ui.Text(c, i18n.T(i18n.KeyMiscAssistantSkillsServiceUnavailable)).TextColor(muted)
		}
		ui.Text(c, i18n.T(i18n.KeyW4AssistantAllCoversThisGroupSCurrentAndFutureC)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
		for _, group := range pluginrpc.DshbotCapabilityGroups {
			group := group
			p := s.Draft[group]
			ui.Column(c.Key("capability-group-" + group)).Gap(4).Children(func() {
				ui.Text(c, group+": "+botCapabilityPolicyText(p)).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				ui.Row(c).Gap(6).Children(func() {
					for _, mode := range []string{"all", "selected"} {
						mode := mode
						if ui.Button(c, group+": "+mode).Disabled(!editable || p.Mode == mode).Clicked() && m.OnChange != nil {
							m.OnChange(group, "mode", mode)
						}
					}
				})
				entries := append([]pluginrpc.DshbotCapabilityEntry{}, s.Status.Catalog.Entries(group)...)
				known := []string{}
				for _, e := range entries {
					known = append(known, e.Name)
				}
				for _, name := range p.Names {
					if !slices.Contains(known, name) {
						entries = append(entries, pluginrpc.DshbotCapabilityEntry{Name: name, Description: i18n.T(i18n.KeyMiscAssistantUnavailableRemoveBeforeSaving)})
					}
				}
				if p.Mode == "selected" {
					for _, e := range entries {
						e := e
						checked := slices.Contains(p.Names, e.Name)
						if ui.Checkbox(c.Key(group+"-"+e.Name), &checked, e.Name).Disabled(!editable).Changed() && m.OnChange != nil {
							m.OnChange(group, "toggle", e.Name)
						}
						if e.Description != "" {
							ui.Text(c, e.Description).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
						}
					}
				}
			})
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantDiscardCapabilityDraft)).Disabled(blocked || !s.Loaded || !s.Dirty).Clicked() && m.OnDiscard != nil {
			m.OnDiscard()
		}
		if ui.Button(c, i18n.T(i18n.KeyMiscAssistantReviewCapabilityChanges)).Disabled(!editable || !s.Dirty).Clicked() && m.OnReview != nil {
			m.OnReview()
		}
		if s.Review != nil {
			r := s.Review
			before := r.Command.Expected
			ui.Column(c.Key("capability-review")).Border(1, accent).Radius(8).Padding(10).Gap(6).Children(func() {
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantReviewCapabilitySave)).Bold().TextColor(fg)
				ui.Text(c, fmt.Sprintf(i18n.T(i18n.KeyMiscAssistantCapabilityReviewDetail), m.HostLabel, before.Bot.ID, before.Bot.SessionID, before.Revision, botCapabilityProfileText(before, "workspaceId"), botCapabilityProfileText(before, "objective"), botCapabilityProfileText(before, "objectiveMaxRounds"))).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				for _, group := range pluginrpc.DshbotCapabilityGroups {
					ui.Text(c, group+"\nBefore: "+botCapabilityPolicyText(before.Policies[group])+"\nAfter: "+botCapabilityPolicyText(r.Command.Desired[group])).FontSize(fonttheme.InterfaceSize(12)).TextColor(fg).Selectable()
				}
				ui.Text(c, i18n.T(i18n.KeyW4AssistantReturnsTheCurrentFullProfileUnchangedA)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ui.Text(c, i18n.T(i18n.KeyW4AssistantImportantTheOriginalBotUpdateAlsoSyncs)).FontSize(fonttheme.InterfaceSize(12)).TextColor(accent)
				ui.Text(c, i18n.T(i18n.KeyMiscAssistantUnknownOutcomesRequireReadback)).FontSize(fonttheme.InterfaceSize(11)).TextColor(muted)
				ack := r.Acknowledged
				if ui.Checkbox(c, &ack, botCapabilityAcknowledgement()).Disabled(blocked).Changed() && m.OnAcknowledge != nil {
					m.OnAcknowledge(ack)
				}
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantCancelCapabilitySave)).Disabled(blocked).Clicked() && m.OnDismiss != nil {
					m.OnDismiss()
				}
				if ui.Button(c, i18n.T(i18n.KeyMiscAssistantConfirmCapabilitySave)).Disabled(blocked || !s.Loaded || !s.Writable || !r.Acknowledged).Clicked() && m.OnConfirm != nil {
					m.OnConfirm()
				}
			})
		}
	})
}

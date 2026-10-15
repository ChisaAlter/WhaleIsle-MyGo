// Package schedule is the M9 native scheduled-tasks panel (dsh-schedule).
package schedule

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Model wires the schedule panel.
type Model struct {
	Store         *schedstore.Store
	SessionID     string
	SessionLinks  map[string]string
	OnOpenSession func(row schedstore.Entry)
	ThemeID       string
	SystemDark    bool
	OnClose       func()
	OnInvalidate  func()
	OnRefresh     func()
	OnDelete      func(id string)
	OnHistory     func(id string)
	OnSaveTitle   func(id, title string)
	OnSavePrompt  func(id, prompt string)
	OnCopyHistory func(text string)
	// HistoryText is the last schedule/history JSON preview for the selected id.
	HistoryText string
	// EditID / EditTitle / EditPrompt drafts for schedule/update fields.
	EditID     *string
	EditTitle  *string
	EditPrompt *string
}

// View paints the schedule panel.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	bg := chrome.Color(colors.BG)
	border := chrome.Color(colors.Border)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	accent := chrome.Color(colors.Accent)
	snap := m.Store.Snapshot()

	ui.Column(c).Grow(1).MinHeight(0).Background(bg).Children(func() {
		ui.Row(c).Padding(8, 12, 8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyScheduleTitle)).FontSize(15).Bold().TextColor(fg).Grow(1)
			if snap.Loading {
				ui.Text(c, i18n.T(i18n.KeyScheduleLoading)).FontSize(12).TextColor(muted)
			}
			if ui.Button(c, i18n.T(i18n.KeyScheduleRefresh)).Clicked() {
				if m.OnRefresh != nil {
					m.OnRefresh()
				} else {
					m.Store.Refresh(context.Background(), m.SessionID)
				}
			}
			if m.OnClose != nil && ui.Button(c, i18n.T(i18n.KeyScheduleBack)).Clicked() {
				m.OnClose()
			}
		})
		ui.Box(c).Fill().Height(1).Background(border)

		ui.Column(c).Grow(1).MinHeight(0).Padding(12, 16, 12, 16).Gap(10).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyScheduleNote)).FontSize(12).TextColor(muted)
			if snap.CatalogHint != "" {
				ui.Text(c, snap.CatalogHint).FontSize(11).TextColor(muted)
			}
			if snap.Err != "" {
				ui.Text(c, snap.Err).FontSize(12).TextColor(accent)
			}
			if len(snap.Rows) == 0 && !snap.Loading && snap.Err == "" {
				ui.Text(c, i18n.T(i18n.KeyScheduleEmpty)).FontSize(12).TextColor(muted)
				return
			}
			ui.Column(c).Fill().MinHeight(0).Background(field).Border(1, border).Radius(8).
				Padding(8, 10, 8, 10).Gap(8).Children(func() {
				for _, row := range snap.Rows {
					entry := row
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Gap(2).Grow(1).Children(func() {
							ui.Text(c, entry.Title).FontSize(12).Bold().TextColor(fg)
							if entry.Detail != "" {
								ui.Text(c, entry.Detail).FontSize(11).TextColor(muted)
							}
						})
						if entry.ID != "" && m.EditID != nil && m.EditTitle != nil {
							if ui.Button(c, i18n.T(i18n.KeyScheduleEditTitle)).FontSize(11).Clicked() {
								*m.EditID = entry.ID
								*m.EditTitle = entry.Title
								if m.EditPrompt != nil {
									*m.EditPrompt = ""
								}
							}
						}
						if entry.ID != "" && m.OnHistory != nil {
							if ui.Button(c, i18n.T(i18n.KeyScheduleHistory)).FontSize(11).Clicked() {
								m.OnHistory(entry.ID)
							}
						}
						if entry.ID != "" && m.OnDelete != nil {
							if ui.Button(c, i18n.T(i18n.KeyScheduleDelete)).Clicked() {
								m.OnDelete(entry.ID)
							}
						}
					})
				}
			})
			if m.EditID != nil && m.EditTitle != nil && strings.TrimSpace(*m.EditID) != "" {
				ui.Text(c, i18n.T(i18n.KeyScheduleEditTitle)).FontSize(11).Bold().TextColor(muted)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.TextInput(c, m.EditTitle).Grow(1)
					if m.OnSaveTitle != nil && ui.Button(c, i18n.T(i18n.KeyScheduleSaveTitle)).FontSize(11).Clicked() {
						m.OnSaveTitle(*m.EditID, *m.EditTitle)
					}
				})
				if m.EditPrompt != nil {
					ui.Text(c, i18n.T(i18n.KeyScheduleEditPrompt)).FontSize(11).Bold().TextColor(muted)
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.TextInput(c, m.EditPrompt).Grow(1)
						if m.OnSavePrompt != nil && ui.Button(c, i18n.T(i18n.KeyScheduleSavePrompt)).FontSize(11).Clicked() {
							m.OnSavePrompt(*m.EditID, *m.EditPrompt)
						}
					})
				}
			}
			if strings.TrimSpace(m.HistoryText) != "" {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, i18n.T(i18n.KeyScheduleHistory)).FontSize(11).Bold().TextColor(muted).Grow(1)
					if m.OnCopyHistory != nil {
						if ui.Button(c, i18n.T(i18n.KeyScheduleCopyHistory)).FontSize(11).Clicked() {
							m.OnCopyHistory(m.HistoryText)
						}
					}
				})
				ui.Column(c).Background(field).Border(1, border).Radius(8).
					Padding(8, 10, 8, 10).MaxHeight(160).Children(func() {
					ui.Text(c, m.HistoryText).FontSize(11).TextColor(fg)
				})
			}
		})
	})
}

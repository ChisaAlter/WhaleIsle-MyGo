// Package schedule is the M9 native scheduled-tasks panel (dsh-schedule).
package schedule

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"

	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
	"whaleisle/internal/ui/theme"
)

// Model wires the schedule panel.
type Model struct {
	DetailID         string
	DetailSessionID  string
	ViewerSessionID  string
	Store            *schedstore.Store
	SessionID        string
	SessionLinks     map[string]string
	SessionLinkState func(string) string
	OnNewTask        func()
	OnOpenSession    func(row schedstore.Entry)
	ThemeID          string
	SystemDark       bool
	OnClose          func()
	OnInvalidate     func()
	OnRefresh        func()
	OnDelete         func(id string)
	OnHistory        func(id string)
	OnSaveTitle      func(id, title string)
	OnSavePrompt     func(id, prompt string)
	OnCopyHistory    func(text string)
	// HistoryText is the last schedule/history JSON preview for the selected id.
	HistoryText string
	// EditID / EditTitle / EditPrompt drafts for schedule/update fields.
	EditID     *string
	EditTitle  *string
	EditPrompt *string
}

type storeEpoch struct {
	last *schedstore.Store
	n    uint64
}

func trackStoreEpoch(root ui.Element, store *schedstore.Store) uint64 {
	ep := ui.Local(root, "store-epoch", func() storeEpoch { return storeEpoch{} })
	if ep.last != store {
		ep.last, ep.n = store, ep.n+1
	}
	return ep.n
}

func scheduleMemoBase(epoch uint64, snap schedstore.ViewState) pagekit.Version {
	return pagekit.NewVersion().
		Uint(design.Revision()).
		Uint(theme.Revision()).
		Uint(epoch).
		Uint(snap.CatalogRevision).
		Uint(snap.ConnectionEpoch).
		String(snap.Origin).
		Bool(snap.ClientReady).
		Bool(snap.Loading)
}

// View paints the schedule panel.
func View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	snap := m.Store.Snapshot().Value
	t := design.For(c)
	root := ui.Column(c).Key("schedule-standalone").Grow(1).FillWidth().MinHeight(0).MinWidth(0)
	epoch := trackStoreEpoch(root, m.Store)
	scroll := ui.Local(root, "scroll", func() *ui.ScrollState { return &ui.ScrollState{} })

	root.Children(func() {
		pagekit.Page(c, pagekit.PageOptions{Scroll: *scroll}, func() {
			pagekit.Header(c, i18n.T(i18n.KeyScheduleTitle), i18n.T(i18n.KeyScheduleNote), func() {
				if coss.Button(c, i18n.T(i18n.KeyScheduleRefresh), coss.ButtonOptions{
					Variant: coss.ButtonOutline,
					Size:    coss.ButtonSM,
					Icon:    coss.IconRefresh,
				}).Clicked() {
					if m.OnRefresh != nil {
						m.OnRefresh()
					} else {
						m.Store.Refresh(context.Background(), m.SessionID)
					}
				}
				if m.OnClose != nil && pagekit.Back(c, i18n.T(i18n.KeyScheduleBack)) {
					m.OnClose()
				}
			})
			if snap.CatalogHint != "" {
				pagekit.Caption(c, snap.CatalogHint).FillWidth()
			}
			if snap.Err != nil {
				pagekit.FaultSentence(c, "", snap.Err.SentenceText(), "", nil)
			}
			if len(snap.Rows) == 0 && !snap.Loading && snap.Err == nil {
				coss.Empty(c, coss.EmptyOptions{
					Icon:  coss.IconClock,
					Title: i18n.T(i18n.KeyScheduleEmpty),
				}, nil)
				return
			}
			listVer := scheduleMemoBase(epoch, snap).
				Uint(uint64(len(snap.Rows))).
				Bool(m.EditID != nil).
				Bool(m.EditTitle != nil).
				Bool(m.OnHistory != nil).
				Bool(m.OnDelete != nil).
				Key()
			listCard := ui.Column(c).Key("schedule-standalone-list").FillWidth().MinWidth(0).
				Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Clip().
				Memo(listVer)
			listCard.Children(func() {
				for i, row := range snap.Rows {
					entry := row
					if i > 0 {
						coss.Separator(c, coss.Horizontal)
					}
					ui.Row(c.Key(taskKey(entry))).FillWidth().MinWidth(0).
						Padding(design.Sp(3), design.Sp(4)).Gap(design.Sp(3)).AlignItems(ui.Center).Children(func() {
						ui.Column(c).Gap(design.Sp(0.5)).Grow(1).MinWidth(0).Children(func() {
							ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
								pagekit.Strong(c, entry.Title).Shrink(1).MinWidth(0)
								if entry.Status != "" {
									variant := coss.BadgeSecondary
									if entry.Status == "active" {
										variant = coss.BadgeSuccess
									}
									coss.Badge(c, entry.Status, coss.BadgeOptions{Variant: variant})
								}
							})
							if entry.Detail != "" {
								pagekit.Muted(c, entry.Detail).FillWidth().MinWidth(0)
							}
						})
						ui.Row(c).Shrink(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
							if entry.ID != "" && m.EditID != nil && m.EditTitle != nil {
								if coss.Button(c, i18n.T(i18n.KeyScheduleEditTitle), coss.ButtonOptions{
									Variant: coss.ButtonOutline,
									Size:    coss.ButtonSM,
								}).Clicked() {
									*m.EditID = entry.ID
									*m.EditTitle = entry.Title
									if m.EditPrompt != nil {
										*m.EditPrompt = ""
									}
								}
							}
							if entry.ID != "" && m.OnHistory != nil {
								if coss.Button(c, i18n.T(i18n.KeyScheduleHistory), coss.ButtonOptions{
									Variant: coss.ButtonOutline,
									Size:    coss.ButtonSM,
								}).Clicked() {
									m.OnHistory(entry.ID)
								}
							}
							if entry.ID != "" && m.OnDelete != nil {
								if coss.Button(c, i18n.T(i18n.KeyScheduleDelete), coss.ButtonOptions{
									Variant: coss.ButtonDestructiveOutline,
									Size:    coss.ButtonSM,
								}).Clicked() {
									m.OnDelete(entry.ID)
								}
							}
						})
					})
				}
			})
			if m.EditID != nil && m.EditTitle != nil && strings.TrimSpace(*m.EditID) != "" {
				coss.Field(c, coss.FieldOptions{Label: i18n.T(i18n.KeyScheduleEditTitle)}, func() {
					ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
						titleDraft := *m.EditTitle
						ui.Row(c).Grow(1).MinWidth(0).Children(func() {
							if coss.Input(c, &titleDraft, coss.InputOptions{
								Key:   "schedule-standalone-edit-title",
								Label: i18n.T(i18n.KeyScheduleEditTitle),
							}).Changed() {
								*m.EditTitle = titleDraft
							}
						})
						if m.OnSaveTitle != nil && coss.Button(c, i18n.T(i18n.KeyScheduleSaveTitle), coss.ButtonOptions{
							Size: coss.ButtonSM,
						}).Clicked() {
							m.OnSaveTitle(*m.EditID, *m.EditTitle)
						}
					})
				})
				if m.EditPrompt != nil {
					coss.Field(c, coss.FieldOptions{Label: i18n.T(i18n.KeyScheduleEditPrompt)}, func() {
						ui.Row(c).FillWidth().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
							promptDraft := *m.EditPrompt
							ui.Row(c).Grow(1).MinWidth(0).Children(func() {
								if coss.Input(c, &promptDraft, coss.InputOptions{
									Key:   "schedule-standalone-edit-prompt",
									Label: i18n.T(i18n.KeyScheduleEditPrompt),
								}).Changed() {
									*m.EditPrompt = promptDraft
								}
							})
							if m.OnSavePrompt != nil && coss.Button(c, i18n.T(i18n.KeyScheduleSavePrompt), coss.ButtonOptions{
								Size: coss.ButtonSM,
							}).Clicked() {
								m.OnSavePrompt(*m.EditID, *m.EditPrompt)
							}
						})
					})
				}
			}
			if strings.TrimSpace(m.HistoryText) != "" {
				var copyAction func()
				if m.OnCopyHistory != nil {
					copyAction = func() {
						if coss.Button(c, i18n.T(i18n.KeyScheduleCopyHistory), coss.ButtonOptions{
							Variant: coss.ButtonOutline,
							Size:    coss.ButtonSM,
							Icon:    coss.IconCopy,
						}).Clicked() {
							m.OnCopyHistory(m.HistoryText)
						}
					}
				}
				pagekit.Section(c, i18n.T(i18n.KeyScheduleHistory), copyAction, func() {
					coss.Card(c, coss.CardOptions{}, func() {
						coss.ScrollArea(c, coss.ScrollAreaOptions{}, func() {
							pagekit.Code(c, m.HistoryText).FillWidth().Selectable()
						}).MaxHeight(design.Sp(40)).FillWidth()
					}).FillWidth()
				})
			}
		})
	})
}

package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

type deliveryHistoryState struct {
	Page            schedstore.DeliveryPage
	Loaded          bool
	Error           string
	RetryBefore     string
	RefreshRequired bool
	Dismissed       bool
}

func (h *deliveryHistoryState) finish(next schedstore.DeliveryPage, before string, err error) {
	h.Error = ""
	h.RetryBefore = ""
	h.RefreshRequired = false
	h.Dismissed = false
	if err != nil {
		h.Error = scheduleText("无法加载投递记录，请重试。", "Could not load delivery records. Try again.")
		h.RetryBefore = before
		var failure *schedstore.ResultError
		if errors.As(err, &failure) {
			switch failure.Code {
			case "delivery_cursor_not_found":
				h.Error = i18n.T(i18n.KeyMiscScheduleDeliveryCursorExpiredRefreshSavedRe)
				h.RefreshRequired = true
				h.RetryBefore = ""
			case "schedule_not_found":
				h.Error = i18n.T(i18n.KeyMiscScheduleTaskDeliveryHistoryNotFound)
				h.RetryBefore = ""
			}
		}
		return
	}
	if before != "" && h.Loaded {
		h.Page = schedstore.MergeDeliveryPages(h.Page, next)
	} else {
		h.Page = next
	}
	h.Loaded = true
}

func latestDelivery(row schedstore.Entry) string {
	var record struct {
		LastDelivery *struct {
			MessageID string `json:"messageId"`
		} `json:"lastDelivery"`
	}
	_ = json.Unmarshal(row.Raw, &record)
	if record.LastDelivery != nil {
		return record.LastDelivery.MessageID
	}
	return ""
}

func (p *Panel) loadDeliveryHistory(m Model, row schedstore.Entry, before string, refresh bool) {
	key := taskKey(row)
	epoch := m.Store.Snapshot().ConnectionEpoch
	p.mu.Lock()
	if p.historyLoading && p.historyKey == key && !refresh {
		p.mu.Unlock()
		return
	}
	if p.historyKey != key {
		p.historyState = deliveryHistoryState{}
	}
	p.historyKey = key
	p.historyLatest = latestDelivery(row)
	p.historyRow = row
	p.historyLoading = true
	p.historyRequest++
	request := p.historyRequest
	p.historyState.Error = ""
	p.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		page, err := m.Store.DeliveryHistory(ctx, row, before)
		p.mu.Lock()
		if p.historyKey == key && p.historyRequest == request && m.Store.Snapshot().ConnectionEpoch == epoch {
			p.historyLoading = false
			p.historyState.finish(page, before, err)
		}
		p.mu.Unlock()
		if m.OnInvalidate != nil {
			m.OnInvalidate()
		}
	}()
}

func deliveryTime(value, zone string) string {
	stamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	if zone != "" {
		if location, err := time.LoadLocation(zone); err == nil {
			stamp = stamp.In(location)
		}
	} else {
		stamp = stamp.In(time.Local)
	}
	now := time.Now().In(stamp.Location())
	if i18n.Locale() == i18n.LocaleZH {
		date := fmt.Sprintf("%d月%d日 %s", stamp.Month(), stamp.Day(), stamp.Format("15:04"))
		if now.Year() != stamp.Year() {
			date = fmt.Sprintf("%d年%s", stamp.Year(), date)
		}
		return date
	}
	if now.Year() != stamp.Year() {
		return stamp.Format("Jan 2, 2006, 3:04 PM")
	}
	return stamp.Format("Jan 2, 3:04 PM")
}

func (p *Panel) deliveryHistoryView(c *ui.Context, m Model, fg, muted ui.Color) {
	p.mu.Lock()
	key, row, h, loading := p.historyKey, p.historyRow, p.historyState, p.historyLoading
	p.mu.Unlock()
	if key == "" {
		return
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	border, field := chrome.Color(colors.Border), chrome.Color(colors.Field)
	ui.Column(c.Key("delivery-history")).Label("Saved delivery records").Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
		if h.Error != "" && !h.Dismissed {
			label := scheduleText("重试", "Retry")
			if h.RefreshRequired {
				label = scheduleText("刷新记录", "Refresh records")
			}
			scheduleNotice(c, "schedule-history-failure", h.Error, colors, 24,
				[]scheduleNoticeAction{{Label: label, Disabled: loading, Run: func() { p.loadDeliveryHistory(m, row, h.RetryBefore, h.RetryBefore == "") }}},
				func() { p.mu.Lock(); p.historyState.Dismissed = true; p.mu.Unlock() })
		}
		ui.Scroll(c.Key("schedule-records-scroll-" + key)).Label("Task records scroll").Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Padding(24, 24, 80).Gap(0).Children(func() {
				if loading && len(h.Page.Records) == 0 {
					ui.Row(c).Label("Loading delivery records").FillWidth().Justify(ui.Center).Padding(24, 0).Children(func() { ui.Spinner(c).Size(20, 20).TextColor(muted) })
				}
				if h.Error != "" && h.Dismissed {
					errCard := ui.Column(c).FillWidth().MinWidth(0).Padding(12, 16).Gap(8).Border(.5, border).Radius(12)
					chrome.GlassSurface(errCard, field, chrome.MaterialPanel, 12)
					errCard.Children(func() {
						message := scheduleText("记录待重新读取", "Saved records need a fresh read.")
						if h.RefreshRequired {
							message = scheduleText("记录待刷新", "Saved records need a refresh.")
						}
						ui.Text(c, message).FontSize(theme.InterfaceSize(12)).TextColor(fg).FillWidth().MinWidth(0)
						label := scheduleText("重试", "Retry")
						if h.RefreshRequired {
							label = scheduleText("刷新记录", "Refresh records")
						}
						if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(loading).Clicked() {
							p.loadDeliveryHistory(m, row, h.RetryBefore, h.RetryBefore == "")
						}
					})
				}
				if h.Loaded && len(h.Page.Records) == 0 && !loading && h.Error == "" {
					scheduleEmpty(c, scheduleText("暂无投递记录", "No delivery record available"), colors)
				}
				for index, record := range h.Page.Records {
					record := record
					top := float32(14)
					if index == 0 {
						top = 0
					}
					recCard := ui.Row(c.Key("delivery-"+record.MessageID)).FillWidth().MinWidth(0).Padding(top, 8, 14, 8).Margin(0, -8).Radius(8).Gap(12).AlignItems(ui.Start)
					// Timeline strokes belong to the record list, not a card shell.
					recCard.Children(func() {
						if index > 0 {
							ui.Box(c).Absolute().Left(15.75).Top(0).Width(.5).Height(14).Background(border)
						}
						if index+1 < len(h.Page.Records) {
							after := float32(42)
							if index == 0 {
								after = 28
							}
							ui.Box(c).Absolute().Left(15.75).Top(after).Bottom(0).Width(.5).Background(border)
						}
						ui.Icon(c, scheduleIcons["clock"]).Size(16, 16).Margin(6, 0, 0, 0).TextColor(muted).Shrink(0)
						ui.Column(c).Grow(1).MinWidth(0).Padding(2, 0).Gap(0).Children(func() {
							ui.Text(c, deliveryTime(record.ScheduledAt, row.TimeZone)).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontWeight(500).TextColor(fg).FillWidth().MinWidth(0)
							if record.Prompt != nil {
								expanded := p.deliveryExpanded[key+"/"+record.MessageID]
								font := theme.RenderPreferences().Interface.Family
								prompt := ui.Text(c, *record.Prompt).Font(font).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(22)).Margin(6, 0, 0, 0).TextColor(muted).FillWidth().MinWidth(0).Selectable()
								clamped := ui.Local(prompt, "saved-prompt-clamped", func() bool { return false })
								if !expanded {
									prompt.MaxLines(2).Ellipsis("…")
									prompt.DrawOver(func(painter *ui.Painter, rect ui.Rect) {
										_, full := painter.MeasureText(rect.W, ui.Span{Text: *record.Prompt, Font: font, Size: theme.InterfaceSize(13)})
										_, twoLines := painter.MeasureText(0, ui.Span{Text: "Ag\nAg", Font: font, Size: theme.InterfaceSize(13)})
										hidden := full > twoLines+.5
										if *clamped != hidden {
											*clamped = hidden
											c.Services().Invalidate()
										}
									})
								}
								if expanded || *clamped {
									label := scheduleText("展开", "Expand")
									if expanded {
										label = scheduleText("收起", "Collapse")
									}
									if ui.Button(c.Key("delivery-expand-"+record.MessageID), label).Height(22).Padding(1, 6).Margin(2, 0, 0, -6).Radius(6).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
										if p.deliveryExpanded == nil {
											p.deliveryExpanded = map[string]bool{}
										}
										p.deliveryExpanded[key+"/"+record.MessageID] = !expanded
									}
								}
							}
						})
					})
				}
				if h.Loaded && h.Page.NextBefore != "" && h.Error == "" {
					if ui.Button(c, scheduleText("加载更多", "Load more")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(loading).Clicked() {
						p.loadDeliveryHistory(m, row, h.Page.NextBefore, false)
					}
				}
			})
		})
		if h.Loaded && len(h.Page.Records) > 0 && h.Page.NextBefore == "" && !loading && h.Error == "" && h.Page.EarlierRecordsPruned {
			ui.Column(c).FillWidth().MinWidth(0).Padding(12, 24, 20).Gap(8).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, scheduleText("更早的投递记录已清理", "Earlier delivery records have been cleared")).FontSize(theme.InterfaceSize(12)).TextColor(muted).Grow(1).MinWidth(0)
					if ui.Button(c, "ⓘ").Label(scheduleText("保留规则", "Retention rules")).Tooltip(scheduleText("保留规则", "Retention rules")).Expanded(p.historyRetentionOpen).Size(24, 24).Padding(0).BorderWidth(0).Background(ui.Transparent).Clicked() {
						p.historyRetentionOpen = !p.historyRetentionOpen
					}
				})
				if p.historyRetentionOpen {
					if h.Page.Retention != nil {
						ui.Text(c, fmt.Sprintf(scheduleText("每个任务最多保留最近 %d 天内的 %d 条投递记录。", "Each task keeps up to %d days and %d delivery records."), h.Page.Retention.Days, h.Page.Retention.Records)).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
					}
					ui.Text(c, scheduleText("产生新记录时，超出范围的旧记录会自动清理。清理记录不影响任务继续运行。", "When a new record is saved, older records outside these limits are cleared automatically. Clearing records does not stop the task.")).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
				}
			})
		}
	})
}

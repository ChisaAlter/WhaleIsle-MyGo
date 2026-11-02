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
		h.Error = err.Error()
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
	}
	return stamp.Format("2006-01-02 15:04:05 MST")
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
		ui.Scroll(c.Key("schedule-records-scroll-" + key)).Label("Task records scroll").Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
			ui.Column(c).FillWidth().MinWidth(0).Padding(24, 24, 80).Gap(16).Children(func() {
				if loading {
					ui.Row(c).Label("Loading delivery records").FillWidth().Justify(ui.Center).Padding(24, 0).Children(func() { ui.Spinner(c).Size(20, 20).TextColor(muted) })
				}
				if h.Error != "" {
					errCard := ui.Column(c).FillWidth().MinWidth(0).Padding(12, 16).Gap(8).Border(.5, border).Radius(12)
					chrome.GlassSurface(errCard, field, chrome.MaterialPanel, 12)
					errCard.Children(func() {
						message := h.Error
						if h.Dismissed {
							message = scheduleText("可以重试读取投递记录", "Retry delivery records when ready")
						}
						ui.Text(c, message).FontSize(theme.InterfaceSize(12)).TextColor(fg).FillWidth().MinWidth(0)
						label := scheduleText("重试", "Retry")
						if h.RefreshRequired {
							label = scheduleText("刷新记录", "Refresh records")
						}
						if ui.Button(c, label).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Disabled(loading).Clicked() {
							p.loadDeliveryHistory(m, row, h.RetryBefore, h.RetryBefore == "")
						}
						if !h.Dismissed && ui.Button(c, scheduleText("关闭提示", "Dismiss")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
							p.mu.Lock()
							p.historyState.Dismissed = true
							p.mu.Unlock()
						}
					})
				}
				if h.Loaded && len(h.Page.Records) == 0 && !loading && h.Error == "" {
					scheduleEmpty(c, scheduleText("暂无投递记录", "No delivery record available"), colors)
				}
				for _, record := range h.Page.Records {
					record := record
					recCard := ui.Row(c.Key("delivery-"+record.MessageID)).FillWidth().MinWidth(0).Padding(12, 16).Border(.5, border).Radius(12).Gap(12).AlignItems(ui.Start)
					chrome.GlassSurface(recCard, field, chrome.MaterialPanel, 12)
					recCard.Children(func() {
						ui.Icon(c, scheduleIcons["clock"]).Size(16, 20).TextColor(muted).Shrink(0)
						ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
							ui.Text(c, deliveryTime(record.ScheduledAt, row.TimeZone)).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(600).TextColor(fg).FillWidth().MinWidth(0)
							if record.Prompt == nil {
								ui.Text(c, scheduleText("旧记录未保存指令内容", "Prompt unavailable in this legacy receipt")).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
							} else {
								expanded := p.deliveryExpanded[key+"/"+record.MessageID]
								font := theme.RenderPreferences().Interface.Family
								prompt := ui.Text(c, *record.Prompt).Font(font).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(20)).TextColor(fg).FillWidth().MinWidth(0).Selectable()
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
									if ui.Button(c.Key("delivery-expand-"+record.MessageID), label).Height(28).Padding(0, 10).Radius(6).FontSize(theme.InterfaceSize(12)).BorderWidth(0).Background(ui.Transparent).TextColor(muted).Clicked() {
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
				if h.Loaded && !loading {
					ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
						if ui.Button(c, scheduleText("刷新记录", "Refresh records")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
							p.loadDeliveryHistory(m, row, "", true)
						}
						if m.OnCopyHistory != nil && ui.Button(c, scheduleText("复制已加载记录", "Copy loaded records")).Height(28).Padding(0, 10).Radius(6).Border(.5, border).FontSize(theme.InterfaceSize(12)).Clicked() {
							raw, _ := json.MarshalIndent(h.Page, "", "  ")
							m.OnCopyHistory(string(raw))
						}
					})
				}
			})
		})
		if h.Loaded && h.Page.NextBefore == "" && !loading && h.Error == "" && (h.Page.EarlierRecordsPruned || h.Page.EarlierRecordsUnavailable) {
			ui.Column(c).FillWidth().MinWidth(0).Padding(0, 24, 80).Gap(4).Children(func() {
				label := scheduleText("更早的投递记录已清理", "Earlier delivery records have been cleared")
				if !h.Page.EarlierRecordsPruned {
					label = scheduleText("更早的投递记录可能不可用", "Earlier delivery records may be unavailable")
				}
				ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
				if h.Page.EarlierRecordsPruned && h.Page.Retention != nil {
					ui.Text(c, fmt.Sprintf(scheduleText("每个任务最多保留最近 %d 天内的 %d 条投递记录。", "Each task keeps up to %d days and %d delivery records."), h.Page.Retention.Days, h.Page.Retention.Records)).FontSize(theme.InterfaceSize(12)).TextColor(muted).FillWidth().MinWidth(0)
				}
			})
		}
	})
}

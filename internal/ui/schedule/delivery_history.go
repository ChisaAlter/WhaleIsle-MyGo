package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
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
		h.Error = scheduleText("无法加载任务运行记录", "Could not load delivery records. Try again.")
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
	epoch := m.Store.Snapshot().Value.ConnectionEpoch
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
		if p.historyKey == key && p.historyRequest == request && m.Store.Snapshot().Value.ConnectionEpoch == epoch {
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

func (p *Panel) deliveryHistoryView(c *ui.Context, m Model, epoch uint64) {
	p.mu.Lock()
	key, row, h, loading, measureGen := p.historyKey, p.historyRow, p.historyState, p.historyLoading, p.deliveryMeasureGen
	p.mu.Unlock()
	if key == "" {
		return
	}
	t := design.For(c)
	ui.Column(c.Key("delivery-history")).Label("Saved delivery records").Grow(1).FillWidth().MinWidth(0).MinHeight(0).Children(func() {
		if h.Error != "" {
			label := scheduleText("重试", "Retry")
			if h.RefreshRequired {
				label = scheduleText("刷新", "Refresh records")
			}
			ui.Column(c).FillWidth().Padding(design.Sp(3), design.Sp(6), 0, design.Sp(6)).Children(func() {
				pagekit.FaultSentence(c, "", h.Error, label, func() {
					p.loadDeliveryHistory(m, row, h.RetryBefore, h.RetryBefore == "")
				})
			})
		}
		coss.ScrollArea(c, coss.ScrollAreaOptions{Key: "schedule-records-scroll-" + key}, func() {
			ver := pagekit.NewVersion().
				Uint(design.Revision()).
				Uint(theme.Revision()).
				Uint(epoch).
				String(key).
				String(row.TimeZone).
				Uint(uint64(len(h.Page.Records))).
				Bool(loading).
				String(h.Error).
				String(h.Page.NextBefore).
				Bool(h.Loaded).
				Bool(h.Page.EarlierRecordsPruned).
				Bool(p.historyRetentionOpen).
				Uint(measureGen)
			for _, rec := range h.Page.Records {
				ver = ver.String(rec.MessageID).String(rec.ScheduledAt).Bool(p.deliveryExpanded[key+"/"+rec.MessageID])
			}
			ui.Column(c.Key("schedule-records-list")).FillWidth().MinWidth(0).
				Padding(design.Sp(6), design.Sp(6), design.Sp(20), design.Sp(6)).Gap(0).
				Memo(ver.Key()).Children(func() {
				if loading && len(h.Page.Records) == 0 {
					pagekit.SkeletonRows(c, 3, "Loading delivery records")
				}
				if h.Loaded && len(h.Page.Records) == 0 && !loading && h.Error == "" {
					coss.Empty(c, coss.EmptyOptions{
						Icon:        coss.IconHistory,
						Description: scheduleText("暂无任务运行记录", "No delivery record available"),
					}, nil)
				}
				for index, record := range h.Page.Records {
					record := record
					top := design.Sp(3.5)
					if index == 0 {
						top = 0
					}
					recCard := ui.Row(c.Key("delivery-"+record.MessageID)).FillWidth().MinWidth(0).
						Padding(top, design.Sp(2), design.Sp(3.5), design.Sp(2)).
						Margin(0, -design.Sp(2)).Radius(design.RadiusMD).Gap(design.Sp(3)).AlignItems(ui.Start)
					recCard.Children(func() {
						if index > 0 {
							ui.Box(c).Absolute().Left(design.Sp(4)).Top(0).Width(design.BorderWidth).Height(design.Sp(3.5)).Background(t.Border)
						}
						if index+1 < len(h.Page.Records) {
							after := design.Sp(10.5)
							if index == 0 {
								after = design.Sp(7)
							}
							ui.Box(c).Absolute().Left(design.Sp(4)).Top(after).Bottom(0).Width(design.BorderWidth).Background(t.Border)
						}
						ui.Icon(c, coss.IconClock.SVG()).Size(design.IconSM, design.IconSM).Margin(design.Sp(1.5), 0, 0, 0).TextColor(t.MutedForeground).Shrink(0)
						ui.Column(c).Grow(1).MinWidth(0).Padding(design.Sp(0.5), 0).Gap(0).Children(func() {
							pagekit.Strong(c, deliveryTime(record.ScheduledAt, row.TimeZone)).FillWidth().MinWidth(0)
							if record.Prompt != nil {
								expanded := p.deliveryExpanded[key+"/"+record.MessageID]
								font := theme.RenderPreferences().Interface.Family
								prompt := ui.Text(c, *record.Prompt).Font(font).FontSize(design.TextSM.Size).FixedLineHeight(design.TextSM.LineHeight).
									Margin(design.Sp(1.5), 0, 0, 0).TextColor(t.MutedForeground).FillWidth().MinWidth(0).Selectable()
								clamped := ui.Local(prompt, "saved-prompt-clamped", func() bool { return false })
								if !expanded {
									prompt.MaxLines(2).Ellipsis("…")
									prompt.DrawOver(func(painter *ui.Painter, rect ui.Rect) {
										_, full := painter.MeasureText(rect.W, ui.Span{Text: *record.Prompt, Font: font, Size: design.TextSM.Size})
										_, twoLines := painter.MeasureText(0, ui.Span{Text: "Ag\nAg", Font: font, Size: design.TextSM.Size})
										hidden := full > twoLines+.5
										if *clamped != hidden {
											*clamped = hidden
											p.mu.Lock()
											p.deliveryMeasureGen++
											p.mu.Unlock()
											c.Services().Invalidate()
										}
									})
								}
								if expanded || *clamped {
									label := scheduleText("展开", "Expand")
									if expanded {
										label = scheduleText("收起", "Collapse")
									}
									if coss.Button(c, label, coss.ButtonOptions{
										Key:     "delivery-expand-" + record.MessageID,
										Variant: coss.ButtonGhost,
										Size:    coss.ButtonXS,
									}).Margin(design.Sp(0.5), 0, 0, -design.Sp(1.5)).AlignSelf(ui.Start).Clicked() {
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
					if coss.Button(c, scheduleText("加载更多", "Load more"), coss.ButtonOptions{
						Variant:  coss.ButtonOutline,
						Size:     coss.ButtonSM,
						Disabled: loading,
						Loading:  loading,
					}).AlignSelf(ui.Start).Clicked() {
						p.loadDeliveryHistory(m, row, h.Page.NextBefore, false)
					}
				}
			})
		}).Label("Task records scroll").Grow(1).FillWidth().MinWidth(0).MinHeight(0)

		if h.Loaded && len(h.Page.Records) > 0 && h.Page.NextBefore == "" && !loading && h.Error == "" && h.Page.EarlierRecordsPruned {
			ui.Column(c).FillWidth().MinWidth(0).Padding(design.Sp(3), design.Sp(6), design.Sp(5), design.Sp(6)).Gap(design.Sp(2)).Children(func() {
				ui.Row(c).FillWidth().MinWidth(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
					pagekit.Caption(c, scheduleText("更早的运行记录已清理", "Earlier delivery records have been cleared")).Grow(1).MinWidth(0)
					if coss.Button(c, "", coss.ButtonOptions{
						Variant: coss.ButtonGhost,
						Size:    coss.ButtonIconXS,
						Icon:    coss.IconInfo,
						Label:   scheduleText("保留规则", "Retention rules"),
						Pressed: p.historyRetentionOpen,
					}).Tooltip(scheduleText("保留规则", "Retention rules")).Clicked() {
						p.historyRetentionOpen = !p.historyRetentionOpen
					}
				})
				if p.historyRetentionOpen {
					if h.Page.Retention != nil {
						pagekit.Caption(c, fmt.Sprintf(scheduleText("每个任务最多保留近 %d 天内的 %d 条运行记录。", "Each task keeps up to %d days and %d delivery records."), h.Page.Retention.Days, h.Page.Retention.Records)).FillWidth().MinWidth(0)
					}
					pagekit.Caption(c, scheduleText("产生新记录时，超出范围的旧记录会自动清理。清理记录不影响任务继续运行。", "When a new record is saved, older records outside these limits are cleared automatically. Clearing records does not stop the task.")).FillWidth().MinWidth(0)
				}
			})
		}
	})
}

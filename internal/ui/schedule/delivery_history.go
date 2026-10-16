package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/theme"
)

type deliveryHistoryState struct {
	Page            schedstore.DeliveryPage
	Loaded          bool
	Error           string
	RetryBefore     string
	RefreshRequired bool
}

func (h *deliveryHistoryState) finish(next schedstore.DeliveryPage, before string, err error) {
	h.Error = ""
	h.RetryBefore = ""
	h.RefreshRequired = false
	if err != nil {
		h.Error = err.Error()
		h.RetryBefore = before
		var failure *schedstore.ResultError
		if errors.As(err, &failure) {
			switch failure.Code {
			case "delivery_cursor_not_found":
				h.Error = "分页游标已失效，请刷新已保存的投递记录 / Delivery cursor expired; refresh saved records"
				h.RefreshRequired = true
				h.RetryBefore = ""
			case "schedule_not_found":
				h.Error = "找不到原任务的投递记录 / Task delivery history not found"
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
	ui.Column(c.Key("delivery-history")).Gap(8).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "已保存的投递记录 / Saved deliveries · "+row.Title).Bold().TextColor(fg).Grow(1)
			if ui.Button(c, "刷新记录 / Refresh records").Disabled(loading).Clicked() {
				p.loadDeliveryHistory(m, row, "", true)
			}
		})
		ui.Text(c, "Session: "+row.SessionID).FontSize(theme.InterfaceSize(11)).TextColor(muted)
		ui.Text(c, "这里只表示指令已保存到会话收件箱，不代表任务执行完成或成功。 / These receipts acknowledge inbox delivery, not execution outcomes.").FontSize(theme.InterfaceSize(11)).TextColor(muted)
		if loading {
			ui.Text(c, "加载中 / Loading…").TextColor(muted)
		}
		if h.Error != "" {
			ui.Text(c, h.Error).TextColor(fg)
			label := "重试 / Retry"
			if h.RefreshRequired {
				label = "刷新记录 / Refresh records"
			}
			if ui.Button(c, label).Disabled(loading).Clicked() {
				p.loadDeliveryHistory(m, row, h.RetryBefore, h.RetryBefore == "")
			}
		}
		if h.Loaded && len(h.Page.Records) == 0 && !loading && h.Error == "" {
			ui.Text(c, "暂无已保存的投递记录 / No saved deliveries").TextColor(muted)
		}
		for _, record := range h.Page.Records {
			record := record
			ui.Column(c.Key("delivery-" + record.MessageID)).Padding(8).Gap(5).Children(func() {
				ui.Text(c, "计划时间 / Scheduled: "+deliveryTime(record.ScheduledAt, row.TimeZone)).Bold().TextColor(fg)
				ui.Text(c, "投递时间 / Delivered: "+deliveryTime(record.DeliveredAt, row.TimeZone)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				ui.Text(c, "Message: "+record.MessageID).FontSize(theme.InterfaceSize(11)).TextColor(muted).Selectable()
				if record.Prompt == nil {
					ui.Text(c, "旧记录未保存指令正文 / Prompt unavailable in this legacy receipt").TextColor(muted)
				} else {
					prompt := *record.Prompt
					expanded := p.deliveryExpanded[key+"/"+record.MessageID]
					chars := []rune(prompt)
					if len(chars) > 240 && !expanded {
						prompt = string(chars[:240]) + "…"
					}
					ui.Text(c, prompt).TextColor(fg).Selectable()
					if len(chars) > 240 {
						label := "展开指令 / Expand prompt"
						if expanded {
							label = "收起指令 / Collapse prompt"
						}
						if ui.Button(c, label).Clicked() {
							if p.deliveryExpanded == nil {
								p.deliveryExpanded = map[string]bool{}
							}
							p.deliveryExpanded[key+"/"+record.MessageID] = !expanded
						}
					}
				}
			})
		}
		if h.Loaded && h.Page.NextBefore != "" && h.Error == "" {
			if ui.Button(c, "更早记录 / Older deliveries").Disabled(loading).Clicked() {
				p.loadDeliveryHistory(m, row, h.Page.NextBefore, false)
			}
		}
		if h.Loaded && h.Page.NextBefore == "" && !loading && h.Error == "" {
			if h.Page.EarlierRecordsPruned {
				ui.Text(c, "部分更早的投递记录已按保留规则清理 / Earlier delivery records were pruned").TextColor(muted)
			} else if h.Page.EarlierRecordsUnavailable {
				ui.Text(c, "部分更早记录不可用；不会重建缺失的投递回执 / Earlier records may be unavailable; missing receipts are not reconstructed").TextColor(muted)
			}
			if h.Page.EarlierRecordsPruned && h.Page.Retention != nil {
				ui.Text(c, fmt.Sprintf("每个任务最多保留近 %d 天内的 %d 条投递记录。新增记录时自动清理超限旧记录，不影响任务继续运行。", h.Page.Retention.Days, h.Page.Retention.Records)).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
		}
		if h.Loaded && m.OnCopyHistory != nil && ui.Button(c, "复制已加载记录 / Copy loaded records").Clicked() {
			raw, _ := json.MarshalIndent(h.Page, "", "  ")
			m.OnCopyHistory(string(raw))
		}
	})
}

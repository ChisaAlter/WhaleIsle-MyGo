package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Panel retains each task's unfinished draft when switching tasks or closing the panel.
// Background requests publish only status/history; they never overwrite editor inputs.
type Panel struct {
	drafts              map[string]*taskDraft
	selected            string
	query, statusFilter string
	deleting            *schedstore.Entry
	initialized         bool
	connectionEpoch     uint64
	historyRequest      uint64
	mu                  sync.Mutex
	history, status     string
	historyKey          string
	historyRow          schedstore.Entry
	historyBefore       string
	historyLoading      bool
	committed           map[string]savedDraft
}
type savedDraft struct {
	row    schedstore.Entry
	timing string
}
type taskDraft struct {
	row                                                                 schedstore.Entry
	Title, Prompt, Kind, At, Seconds, Clock, Zone, Weekdays, Expression string
	initialTiming                                                       string
}

func newTaskDraft(row schedstore.Entry) *taskDraft {
	d := &taskDraft{row: row, Title: row.Title, Prompt: row.Prompt, Zone: "UTC"}
	var r struct {
		Kind, ScheduledAt, Time, TimeZone, Expression string
		EverySeconds                                  int
		Weekdays                                      []int
	}
	_ = json.Unmarshal(row.Raw, &r)
	d.Kind = r.Kind
	if d.Kind == "after" {
		d.Kind = "at"
	}
	d.At = r.ScheduledAt
	d.Seconds = strconv.Itoa(r.EverySeconds)
	d.Clock = r.Time
	d.Expression = r.Expression
	if r.TimeZone != "" {
		d.Zone = r.TimeZone
	}
	for _, day := range r.Weekdays {
		if d.Weekdays != "" {
			d.Weekdays += ","
		}
		d.Weekdays += strconv.Itoa(day)
	}
	d.initialTiming = d.timingKey()
	return d
}
func (d *taskDraft) timingKey() string {
	return strings.Join([]string{d.Kind, d.At, d.Seconds, d.Clock, d.Zone, d.Weekdays, d.Expression}, "\x00")
}
func (d *taskDraft) change() (json.RawMessage, error) {
	if d.timingKey() == d.initialTiming {
		return nil, nil
	}
	change := map[string]any{"kind": d.Kind}
	switch d.Kind {
	case "at":
		if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(d.At)); err != nil {
			return nil, fmt.Errorf("use an RFC3339 date including its UTC offset")
		}
		change["at"] = strings.TrimSpace(d.At)
	case "every":
		n, err := strconv.Atoi(d.Seconds)
		if err != nil || n < 60 {
			return nil, fmt.Errorf("interval must be an integer of at least 60 seconds")
		}
		change["every_seconds"] = n
	case "daily", "weekly", "cron":
		if _, err := time.LoadLocation(strings.TrimSpace(d.Zone)); err != nil {
			return nil, fmt.Errorf("enter an IANA time zone, such as Asia/Shanghai or UTC")
		}
		rule := map[string]any{"time_zone": strings.TrimSpace(d.Zone)}
		if d.Kind == "cron" {
			if len(strings.Fields(d.Expression)) != 5 {
				return nil, fmt.Errorf("cron needs five fields")
			}
			rule["expression"] = strings.TrimSpace(d.Expression)
		} else {
			if _, err := time.Parse("15:04:05.999", d.Clock); err != nil {
				return nil, fmt.Errorf("time must be HH:mm:ss, optionally with milliseconds")
			}
			rule["time"] = d.Clock
			if d.Kind == "weekly" {
				days := []int{}
				seen := map[int]bool{}
				for _, v := range strings.Split(d.Weekdays, ",") {
					n, e := strconv.Atoi(strings.TrimSpace(v))
					if e != nil || n < 1 || n > 7 || seen[n] {
						return nil, fmt.Errorf("weekdays must be unique 1–7, comma separated")
					}
					seen[n] = true
					days = append(days, n)
				}
				rule["weekdays"] = days
			}
		}
		change[d.Kind] = rule
	default:
		return nil, fmt.Errorf("choose a supported rule")
	}
	return json.Marshal(change)
}
func (p *Panel) message(text string) { p.mu.Lock(); p.status = text; p.mu.Unlock() }

// View renders the full catalog, captured-owner editor, deletion confirmation and history.
func (p *Panel) View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	snap := m.Store.Snapshot()
	if !p.initialized || p.connectionEpoch != snap.ConnectionEpoch {
		p.initialized = true
		p.connectionEpoch = snap.ConnectionEpoch
		p.selected = ""
		p.query, p.statusFilter = "", "all"
		p.deleting = nil
		p.mu.Lock()
		p.historyRequest++
		p.historyKey, p.history, p.historyBefore = "", "", ""
		p.historyLoading = false
		p.committed = nil
		p.mu.Unlock()
		m.Store.Refresh(context.Background(), m.SessionID)
		snap = m.Store.Snapshot()
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	p.mu.Lock()
	status, history := p.status, p.history
	historyRow, historyBefore, historyLoading := p.historyRow, p.historyBefore, p.historyLoading
	for key, saved := range p.committed {
		if d := p.drafts[key]; d != nil {
			d.row = saved.row
			d.initialTiming = saved.timing
		}
	}
	p.committed = nil
	p.mu.Unlock()
	for _, row := range snap.Rows {
		if d := p.drafts[taskKey(row)]; d != nil {
			d.rebase(row)
		}
	}
	ui.Column(c).Grow(1).MinHeight(0).Background(chrome.Color(colors.BG)).Padding(12, 16, 12, 16).Gap(8).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "定时任务 / Scheduled tasks").Bold().Grow(1)
			if m.OnNewTask != nil && ui.Button(c, "新任务 / New task").Disabled(!snap.ClientReady).Clicked() {
				m.OnNewTask()
			}
			if ui.Button(c, "刷新 / Refresh").Disabled(snap.Loading || snap.Busy).Clicked() {
				m.Store.Refresh(context.Background(), m.SessionID)
			}
			if ui.Button(c, "返回 / Back").Clicked() && m.OnClose != nil {
				m.OnClose()
			}
		})
		ui.Text(c, "点击新任务进入新会话，用自然语言说明执行内容和时间。任务保留原始 Session。").FontSize(11).TextColor(muted)
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &p.query).Grow(1).Placeholder("搜索名称、指令或会话 / Search title, prompt or Session")
			ui.Select(c, &p.statusFilter, []string{"all", "active", "inactive"})
			if ui.Button(c, "清除筛选 / Clear filters").Clicked() {
				p.query, p.statusFilter = "", "all"
			}
		})
		if snap.CatalogHint != "" {
			ui.Text(c, snap.CatalogHint).FontSize(11).TextColor(muted)
		}
		visibleRows := schedstore.FilterEntries(snap.Rows, p.query, p.statusFilter)
		ui.Text(c, fmt.Sprintf("显示 / Showing %d / %d", len(visibleRows), len(snap.Rows))).FontSize(11).TextColor(muted)
		for _, text := range []string{snap.Err, snap.LastAction, status} {
			if text != "" {
				ui.Text(c, text).FontSize(12).TextColor(fg)
			}
		}
		if snap.Err != "" && len(snap.Rows) > 0 {
			ui.Text(c, "刷新失败，以下保留上次成功读取的数据 / Refresh failed; showing last successful rows").FontSize(11).TextColor(muted)
		}
		if !snap.ClientReady {
			ui.Text(c, "Host 未连接，连接后可管理计划。").TextColor(muted)
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(8).Children(func() {
				if len(visibleRows) == 0 && !snap.Loading && snap.Err == "" {
					text := "没有匹配的任务 / No matching tasks"
					if len(snap.Rows) == 0 {
						text = "暂无定时任务 / No scheduled tasks"
					}
					ui.Text(c, text).TextColor(muted)
				}
				for _, r := range visibleRows {
					row := r
					key := taskKey(row)
					ui.Column(c).Border(1, chrome.Color(colors.Border)).Radius(6).Padding(8, 8, 8, 8).Gap(4).Children(func() {
						ui.Text(c, row.Title).Bold().TextColor(fg)
						ui.Text(c, row.Detail).FontSize(11).TextColor(muted)
						ui.Text(c, schedstore.TimingLabel(row)).FontSize(11).TextColor(muted)
						ui.Text(c, "Session: "+row.SessionID).FontSize(11).TextColor(muted)
						ui.Row(c).Gap(8).Children(func() {
							sessionLabel, canOpen := m.SessionLinks[row.SessionID]
							if sessionLabel == "" {
								sessionLabel = row.SessionID
							}
							if m.OnOpenSession != nil && ui.Button(c, "原会话 / Session: "+sessionLabel).Disabled(!canOpen || row.Origin != snap.Origin).Clicked() {
								m.OnOpenSession(row)
							}
							if ui.Button(c, "编辑 / Edit").Disabled(row.Status == "inactive" || snap.Busy).Clicked() {
								if p.drafts == nil {
									p.drafts = map[string]*taskDraft{}
								}
								if p.drafts[key] == nil {
									p.drafts[key] = newTaskDraft(row)
								}
								p.selected = key
								p.message("")
							}
							if ui.Button(c, "历史 / History").Clicked() {
								p.mu.Lock()
								p.historyKey = key
								p.historyRequest++
								request := p.historyRequest
								epoch := snap.ConnectionEpoch
								p.history = "加载中 / Loading"
								p.historyRow = row
								p.historyBefore = ""
								p.historyLoading = true
								p.mu.Unlock()
								go func() {
									text, next, err := m.Store.HistoryPage(context.Background(), row, "")
									if err != nil {
										text = err.Error()
									}
									p.mu.Lock()
									if p.historyKey == key && p.historyRequest == request && m.Store.Snapshot().ConnectionEpoch == epoch {
										p.history = row.Title + " · " + row.SessionID + "\n" + text
										p.historyBefore = next
										p.historyLoading = false
									}
									p.mu.Unlock()
									if m.OnInvalidate != nil {
										m.OnInvalidate()
									}
								}()
							}
							if ui.Button(c, "删除 / Delete").Disabled(snap.Busy).Clicked() {
								p.deleting = &row
							}
						})
					})
				}
				if p.deleting != nil {
					row := *p.deleting
					ui.Text(c, "确认删除 “"+row.Title+"” ? Session: "+row.SessionID).TextColor(fg)
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, "确认删除 / Confirm delete").Disabled(snap.Busy).Clicked() {
							p.deleting = nil
							go func() {
								err := m.Store.DeleteEntry(context.Background(), row)
								if err != nil {
									p.message(err.Error())
								}
								if m.OnInvalidate != nil {
									m.OnInvalidate()
								}
							}()
						}
						if ui.Button(c, "取消 / Cancel").Clicked() {
							p.deleting = nil
						}
					})
				}
				if d := p.drafts[p.selected]; d != nil {
					available := false
					for _, row := range snap.Rows {
						if taskKey(row) == p.selected && row.Status != "inactive" {
							available = true
							break
						}
					}
					if !available {
						ui.Text(c, "任务已结束、被删除或未完成刷新；当前草稿不能保存。").TextColor(muted)
					}
					ui.Text(c, "编辑 · "+d.row.ID+" · Session: "+d.row.SessionID).Bold().TextColor(fg)
					input := func(label string, value *string) {
						ui.Text(c, label).FontSize(11).TextColor(muted)
						ui.TextInput(c, value).FillWidth()
					}
					input("名称 / Title", &d.Title)
					ui.Text(c, "指令 / Prompt").FontSize(11)
					ui.TextArea(c, &d.Prompt).FillWidth().Lines(3, 6)
					ui.Select(c, &d.Kind, []string{"at", "every", "daily", "weekly", "cron"})
					switch d.Kind {
					case "at":
						input("时间 / RFC3339 (2026-12-01T09:00:00+08:00)", &d.At)
					case "every":
						input("间隔秒数 / Interval seconds", &d.Seconds)
					case "daily", "weekly":
						input("时间 / HH:mm:ss", &d.Clock)
						input("时区 / IANA zone", &d.Zone)
						if d.Kind == "weekly" {
							input("星期 / Mon=1 … Sun=7, comma separated", &d.Weekdays)
						}
					case "cron":
						input("Cron (minute hour day month weekday)", &d.Expression)
						input("时区 / IANA zone", &d.Zone)
					}
					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, "保存全部 / Save all").Disabled(snap.Busy || snap.Loading || !snap.ClientReady || !available).Clicked() {
							title, prompt := strings.TrimSpace(d.Title), strings.TrimSpace(d.Prompt)
							change, err := d.change()
							if title == "" || prompt == "" || len([]rune(title)) > 120 {
								err = fmt.Errorf("title (1–120 characters) and prompt are required")
							}
							if err != nil {
								p.message(err.Error())
							} else {
								row := d.row
								key, timing := p.selected, d.timingKey()
								go func() {
									err := m.Store.Update(context.Background(), row, &title, &prompt, change)
									if err != nil {
										p.message(err.Error())
									} else {
										p.mu.Lock()
										if p.committed == nil {
											p.committed = map[string]savedDraft{}
										}
										for _, r := range m.Store.Snapshot().Rows {
											if taskKey(r) == key {
												p.committed[key] = savedDraft{row: r, timing: timing}
												break
											}
										}
										p.status = "已保存 / Saved"
										p.mu.Unlock()
									}
									if m.OnInvalidate != nil {
										m.OnInvalidate()
									}
								}()
							}
						}
						if ui.Button(c, "收起，保留草稿 / Keep draft").Clicked() {
							p.selected = ""
						}
						if ui.Button(c, "放弃草稿 / Discard draft").Disabled(snap.Busy).Clicked() {
							delete(p.drafts, p.selected)
							p.selected = ""
						}
					})
				}
				if historyBefore != "" && ui.Button(c, "更早记录 / Older deliveries").Disabled(historyLoading).Clicked() {
					p.mu.Lock()
					p.historyLoading = true
					key := p.historyKey
					p.historyRequest++
					request := p.historyRequest
					epoch := snap.ConnectionEpoch
					p.mu.Unlock()
					go func() {
						text, next, err := m.Store.HistoryPage(context.Background(), historyRow, historyBefore)
						p.mu.Lock()
						if p.historyKey == key && p.historyRequest == request && m.Store.Snapshot().ConnectionEpoch == epoch {
							p.historyLoading = false
							if err != nil {
								p.status = err.Error()
							} else {
								p.history += "\n\n" + text
								p.historyBefore = next
							}
						}
						p.mu.Unlock()
						if m.OnInvalidate != nil {
							m.OnInvalidate()
						}
					}()
				}
				if history != "" {
					ui.Text(c, "历史 / Delivery history").Bold()
					ui.Text(c, history).FontSize(11).TextColor(fg)
					if m.OnCopyHistory != nil && ui.Button(c, "复制历史 / Copy history").Clicked() {
						m.OnCopyHistory(history)
					}
				}
			})
		})
	})
}

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

// Panel retains the current detail draft while it is mounted, including failures.
// Switching identity or unmounting retires its pending UI receipts.
type Panel struct {
	recentZones         []string
	drafts              map[string]*taskDraft
	selected            string
	detailTab           string
	query, statusFilter string
	deleting            *schedstore.Entry
	initialized         bool
	connectionEpoch     uint64
	historyRequest      uint64
	mu                  sync.Mutex
	status              string
	statusOwner         string
	historyState        deliveryHistoryState
	deliveryExpanded    map[string]bool
	historyKey          string
	historyRow          schedstore.Entry
	historyLoading      bool
	historyLatest       string
	committed           map[string]savedDraft
	detailOwner         uint64
	actionRequest       uint64
	actionPending       bool
	actionFailed        bool
	deleteAck           *deleteReceipt
}
type savedDraft struct {
	row schedstore.Entry
}
type taskDraft struct {
	weekdayShortcut                                                     bool
	intervalUnit                                                        int
	picker                                                              timingPickerState
	row                                                                 schedstore.Entry
	Title, Prompt, Kind, At, Seconds, Clock, Zone, Weekdays, Expression string
	initialTiming                                                       string
}

func newTaskDraft(row schedstore.Entry) *taskDraft {
	d := &taskDraft{row: row, Title: row.Title, Prompt: row.Prompt, Zone: systemScheduleZone()}
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
	d.intervalUnit = 1
	if r.EverySeconds > 0 && r.EverySeconds%3600 == 0 {
		d.intervalUnit = 3600
	} else if r.EverySeconds > 0 && r.EverySeconds%60 == 0 {
		d.intervalUnit = 60
	}
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
	d.weekdayShortcut = d.Weekdays == "1,2,3,4,5"
	d.initialTiming = d.timingKey()
	return d
}
func (d *taskDraft) timingKey() string {
	fields := []string{d.Kind}
	switch d.Kind {
	case "at":
		fields = append(fields, d.At)
	case "every":
		fields = append(fields, d.Seconds)
	case "daily":
		fields = append(fields, d.Clock, d.Zone)
	case "weekly":
		fields = append(fields, d.Clock, d.Zone, d.Weekdays)
	case "cron":
		fields = append(fields, d.Expression, d.Zone)
	}
	return strings.Join(fields, "\x00")
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
			if _, err := parseTaskCron(d.Expression); err != nil {
				return nil, err
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
func (p *Panel) message(text string) {
	p.mu.Lock()
	p.status, p.statusOwner = text, ""
	if text == "" {
		p.actionFailed = false
	}
	p.mu.Unlock()
}

// View shares one retained task editor between the catalog and a task tab.
func (p *Panel) View(c *ui.Context, m Model) {
	if m.Store == nil {
		return
	}
	snap := m.Store.Snapshot()
	if !p.initialized || p.connectionEpoch != snap.ConnectionEpoch {
		p.closeDetail()
		p.initialized = true
		p.connectionEpoch = snap.ConnectionEpoch
		p.selected, p.detailTab = "", "rule"
		p.query, p.statusFilter = "", "all"
		p.deleting = nil
		p.mu.Lock()
		p.historyRequest++
		p.historyKey = ""
		p.historyState = deliveryHistoryState{}
		p.historyLoading = false
		p.committed = nil
		p.status, p.statusOwner = "", ""
		p.mu.Unlock()
		m.Store.Refresh(context.Background(), m.SessionID)
		snap = m.Store.Snapshot()
	}
	p.mu.Lock()
	for key, saved := range p.committed {
		if d := p.drafts[key]; d != nil {
			*d = *newTaskDraft(saved.row)
		}
	}
	p.committed = nil
	p.mu.Unlock()
	if p.reconcileDetail(snap) && m.DetailID != "" && m.OnClose != nil {
		m.OnClose()
		return
	}
	for _, row := range snap.Rows {
		if row.Status == "active" {
			c.After(time.Second)
		}
		if d := p.drafts[taskKey(row)]; d != nil {
			d.rebase(row)
		}
		if m.DetailID != "" && row.ID == m.DetailID && row.SessionID == m.DetailSessionID && p.selected != taskKey(row) {
			p.OpenTask(row, snap.ConnectionEpoch)
		}
	}
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	root := ui.Row(c.Key("schedule-page")).Label("Automation tasks page").Grow(1).FillWidth().MinWidth(0).MinHeight(0).AlignItems(ui.Stretch).Background(chrome.Color(colors.BG))
	width := root.Bounds().W
	if width <= 0 {
		width, _ = c.Size()
	}
	// Bounds are from the preceding frame. A pane resize needs one matching
	// composition frame once layout has supplied the new local viewport width.
	root.Draw(func(_ *ui.Painter, rect ui.Rect) {
		if rect.W != width {
			c.Services().Invalidate()
		}
	})
	draft := p.drafts[p.selected]
	root.Children(func() {
		if m.DetailID == "" && (draft == nil || width > 760) {
			p.catalogView(c, m, snap, colors, width)
		}
		if draft != nil {
			p.detailView(c, m, snap, draft, colors, width)
		} else if m.DetailID != "" {
			ui.Column(c.Key("schedule-unavailable")).Fill().MinWidth(0).Children(func() {
				p.detailStrip(c, m, snap, nil, colors, width)
				ui.Column(c).Fill().Justify(ui.Center).Children(func() {
					if !p.catalogFeedback(c, m, snap, colors, false) {
						scheduleEmpty(c, scheduleText("任务不可用，可能已被删除。", "Task unavailable. It may have been deleted."), colors)
					}
				})
			})
		}
	})
	if p.deleting == nil && draft != nil && root.Shortcut(0, ui.KeyEscape) {
		p.closeDetail()
		if m.DetailID != "" {
			if m.OnClose != nil {
				m.OnClose()
			}
		}
	}
	p.deleteDialog(c, m, snap, colors)
}

func (d *taskDraft) dirty() bool {
	return strings.TrimSpace(d.Title) != strings.TrimSpace(d.row.Title) ||
		strings.TrimSpace(d.Prompt) != strings.TrimSpace(d.row.Prompt) || d.timingKey() != d.initialTiming
}

func (p *Panel) saveDraft(m Model, d *taskDraft) {
	title, prompt := strings.TrimSpace(d.Title), strings.TrimSpace(d.Prompt)
	change, err := d.change()
	if title == "" || prompt == "" || len([]rune(title)) > 120 {
		err = fmt.Errorf("%s", scheduleText("请输入不超过 120 字的任务名称及任务指令。", "Enter a task name of at most 120 characters and a task instruction."))
	}
	key, row := taskKey(d.row), d.row
	var titleChange, promptChange *string
	if title != strings.TrimSpace(row.Title) {
		titleChange = &title
	}
	if prompt != strings.TrimSpace(row.Prompt) {
		promptChange = &prompt
	}
	p.mu.Lock()
	p.actionRequest++
	owner, request, epoch := p.detailOwner, p.actionRequest, p.connectionEpoch
	p.actionPending, p.actionFailed = err == nil, err != nil
	p.status, p.statusOwner = "", key
	if err != nil {
		p.status = err.Error()
	}
	p.mu.Unlock()
	if err != nil {
		return
	}
	go func() {
		err := m.Store.Update(context.Background(), row, titleChange, promptChange, change)
		snap := m.Store.Snapshot()
		p.mu.Lock()
		if !p.actionCurrentLocked(owner, request, epoch) || snap.ConnectionEpoch != epoch {
			p.mu.Unlock()
			return
		}
		p.actionPending, p.actionFailed = false, err != nil
		if err != nil {
			p.status, p.statusOwner = err.Error(), key
		} else {
			if p.committed == nil {
				p.committed = map[string]savedDraft{}
			}
			for _, r := range snap.Rows {
				if taskKey(r) == key {
					p.committed[key] = savedDraft{row: r}
					break
				}
			}
			p.status, p.statusOwner = scheduleText("已保存", "Saved"), key
		}
		p.mu.Unlock()
		if m.OnInvalidate != nil {
			m.OnInvalidate()
		}
	}()
}

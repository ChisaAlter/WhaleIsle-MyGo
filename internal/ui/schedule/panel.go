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
	"unicode/utf16"
	schedstore "whaleisle/internal/store/schedule"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// Panel retains the current detail draft while it is mounted, including failures.
// Switching identity or unmounting retires its pending UI receipts.
type Panel struct {
	recentZones          []string
	drafts               map[string]*taskDraft
	selected             string
	detailTab            string
	query, statusFilter  string
	deleting             *schedstore.Entry
	initialized          bool
	connectionEpoch      uint64
	historyRequest       uint64
	mu                   sync.Mutex
	status               string
	statusOwner          string
	statusDismissed      bool
	statusLocal          bool
	statusDraft          string
	historyState         deliveryHistoryState
	deliveryExpanded     map[string]bool
	historyKey           string
	historyRow           schedstore.Entry
	historyLoading       bool
	historyLatest        string
	historyRetentionOpen bool
	committed            map[string]savedDraft
	detailOwner          uint64
	detailReadIdentity   string
	detailReadBaseline   uint64
	actionRequest        uint64
	actionPending        bool
	actionFailed         bool
	deleteAck            *deleteReceipt
	headingFocus         ui.Handle
	detailFocus          ui.Handle
	focusDetail          bool
	returnRow            string
	rowFocus             map[string]*ui.Handle
	deleteTrigger        ui.Handle
	deleteCancel         ui.Handle
	focusDelete          bool
	catalogError         string
	catalogErrorRequest  uint64
	catalogDismissed     bool
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
			return nil, fmt.Errorf("%s", scheduleText("请输入有效的日期和时间，或检查时间字段", "Enter a valid date and time."))
		}
		change["at"] = strings.TrimSpace(d.At)
	case "every":
		n, err := strconv.Atoi(d.Seconds)
		if err != nil || n < 60 {
			return nil, fmt.Errorf("%s", scheduleText("请输入至少 60 秒的间隔", "Enter an interval of at least 60 seconds."))
		}
		change["every_seconds"] = n
	case "daily", "weekly", "cron":
		if _, err := time.LoadLocation(strings.TrimSpace(d.Zone)); err != nil {
			return nil, fmt.Errorf("%s", scheduleText("请输入有效的 IANA 时区，例如 Asia/Shanghai", "Choose a valid time zone, for example Asia/Shanghai."))
		}
		rule := map[string]any{"time_zone": strings.TrimSpace(d.Zone)}
		if d.Kind == "cron" {
			if _, err := parseTaskCron(d.Expression); err != nil {
				return nil, fmt.Errorf("%s", scheduleText("请输入 5 字段的 cron 表达式，例如 0 9 * * 1-5", "Enter a five-field cron expression, for example 0 9 * * 1-5."))
			}
			rule["expression"] = strings.TrimSpace(d.Expression)
		} else {
			if _, err := time.Parse("15:04:05.999", d.Clock); err != nil {
				return nil, fmt.Errorf("%s", scheduleText("请输入有效的时、分、秒。", "Enter a valid hour, minute, and second."))
			}
			rule["time"] = d.Clock
			if d.Kind == "weekly" {
				days := []int{}
				seen := map[int]bool{}
				for _, v := range strings.Split(d.Weekdays, ",") {
					n, e := strconv.Atoi(strings.TrimSpace(v))
					if e != nil || n < 1 || n > 7 || seen[n] {
						return nil, fmt.Errorf("%s", scheduleText("请至少选择一个星期，且不要重复。", "Choose at least one weekday without duplicates."))
					}
					seen[n] = true
					days = append(days, n)
				}
				rule["weekdays"] = days
			}
		}
		change[d.Kind] = rule
	default:
		return nil, fmt.Errorf("%s", scheduleText("请选择运行规则。", "Choose a run rule."))
	}
	return json.Marshal(change)
}
func (p *Panel) message(text string) {
	p.mu.Lock()
	p.status, p.statusOwner = text, ""
	p.statusDismissed, p.statusLocal = false, text != ""
	p.statusDraft = ""
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
	initialized := false
	if !p.initialized || p.connectionEpoch != snap.ConnectionEpoch {
		p.closeDetail()
		p.initialized = true
		p.connectionEpoch = snap.ConnectionEpoch
		p.detailReadIdentity = ""
		p.rowFocus, p.returnRow = nil, ""
		initialized = true
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
	}
	// A retained catalog sent before this navigation cannot establish that the
	// task is missing. Like ScheduleTaskTab, request a read newer than this seat's
	// navigation baseline and keep loading/error visible until that read settles.
	detailIdentity := m.DetailSessionID + "\x00" + m.DetailID
	if m.DetailID != "" && p.detailReadIdentity != detailIdentity {
		p.closeDetail()
		p.detailReadIdentity = detailIdentity
		p.detailReadBaseline = snap.CatalogRequest
		m.Store.Refresh(context.Background(), m.SessionID)
		snap = m.Store.Snapshot()
	} else if initialized {
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
	previousSelection := p.selected
	if p.reconcileDetail(snap) && m.DetailID != "" && m.OnClose != nil {
		m.OnClose()
		return
	}
	if m.DetailID == "" && previousSelection != "" && p.selected == "" {
		p.focusCatalog(c, snap)
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
	root := ui.Row(c.Key("schedule-page")).Label("Automation tasks page").Grow(1).FillWidth().MinWidth(0).MinHeight(0).AlignItems(ui.Stretch).Background(ui.Transparent)
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
					answered := snap.CatalogSettled && snap.CatalogReadSettled > p.detailReadBaseline
					if !answered && snap.ClientReady && snap.Err == "" {
						ui.Spinner(c).Size(20, 20).TextColor(chrome.Color(colors.Muted))
					} else if !p.catalogFeedback(c, m, snap, colors, false) {
						scheduleEmpty(c, scheduleText("任务已不可用，可能已被删除", "Task unavailable. It may have been deleted."), colors)
					}
				})
			})
		}
	})
	if p.deleting == nil && draft != nil && root.Shortcut(0, ui.KeyEscape) {
		if draft.picker.calendarOpen {
			draft.picker.calendarOpen = false
		} else if draft.picker.clock.open {
			draft.picker.clock.open = false
		} else if draft.picker.zone.open {
			draft.picker.zone.open = false
		} else {
			p.dismissDetails(c, m, snap)
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
	change, err := d.validatedChange()
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
	p.statusDismissed, p.statusLocal = false, err != nil
	p.statusDraft = d.draftKey()
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
		p.statusDismissed, p.statusLocal = false, false
		if err != nil {
			p.status, p.statusOwner = scheduleUpdateFailure(err), key
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
			p.status, p.statusOwner = "", key
		}
		p.mu.Unlock()
		if m.OnInvalidate != nil {
			m.OnInvalidate()
		}
	}()
}

func (d *taskDraft) validatedChange() (json.RawMessage, error) {
	title := strings.TrimSpace(d.Title)
	// Match the installed Client's JS string length for the 120-character limit.
	if title == "" || len(utf16.Encode([]rune(title))) > 120 {
		return nil, fmt.Errorf("%s", scheduleText("请输入不超过 120 个字符的任务名称", "Enter a task name of at most 120 characters."))
	}
	if strings.TrimSpace(d.Prompt) == "" {
		return nil, fmt.Errorf("%s", scheduleText("请输入任务内容", "Enter a task instruction."))
	}
	return d.change()
}

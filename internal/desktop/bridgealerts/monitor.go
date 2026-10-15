// Package bridgealerts consumes the managed component's native reminder queue.
// Claim is stateful: the server advances its shared native cursor before delivery.
package bridgealerts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

// ErrNotClaimed is reserved for a failure established before dispatching POST.
var ErrNotClaimed = errors.New("reminder claim was not dispatched")
var errStale = errors.New("鲸桥实例或窗口已改变，请重新查看提醒")

type Scalar string

func (s *Scalar) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*s = ""
		return nil
	}
	var v string
	if len(b) > 0 && b[0] == '"' {
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
	} else {
		var n json.Number
		if err := json.Unmarshal(b, &n); err != nil {
			return err
		}
		if _, err := n.Float64(); err != nil {
			return err
		}
		v = n.String()
	}
	*s = Scalar(v)
	return nil
}

type Alert struct {
	Name, Provider, User, Kind, Window, ResetsAt string
	Credits, Used, Balance                       Scalar
}
type Record struct {
	Sequence int64 `json:"sequence"`
	Alert    Alert `json:"alert"`
}
type Batch struct {
	Sequence int64    `json:"sequence"`
	Alerts   []Record `json:"alerts"`
}
type Source interface {
	Current(context.Context) (string, error)
	Claim(context.Context, string) (Batch, error)
	Open(context.Context, string) error
}
type Ticket struct {
	Scope      string
	Generation uint64
	Sequence   int64
}
type Notice struct {
	Ticket      Ticket
	Title, Body string
}
type Snapshot struct {
	Manage           Ticket
	Alerts           []Notice
	Status           string
	Paused, Fallback bool
	Dropped          int
}
type Monitor struct {
	mu         sync.Mutex
	source     Source
	ctx        context.Context
	cancel     context.CancelFunc
	closed     atomic.Bool
	reading    bool
	scope      string
	generation uint64
	cursors    map[string]int64
	blocked    map[string]bool
	snapshot   Snapshot
}

func New(source Source) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{source: source, ctx: ctx, cancel: cancel, cursors: map[string]int64{}, blocked: map[string]bool{}}
}
func (m *Monitor) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.snapshot
	out.Alerts = append([]Notice(nil), out.Alerts...)
	return out
}
func (m *Monitor) currentLocked(t Ticket) bool {
	if m.closed.Load() || t.Scope == "" || t.Scope != m.scope || t.Generation != m.generation {
		return false
	}
	if t.Sequence == 0 {
		return true
	}
	for _, n := range m.snapshot.Alerts {
		if n.Ticket == t {
			return true
		}
	}
	return false
}
func (m *Monitor) Current(t Ticket) bool { m.mu.Lock(); defer m.mu.Unlock(); return m.currentLocked(t) }
func (m *Monitor) Close() {
	m.closed.Store(true)
	m.cancel()
	m.mu.Lock()
	m.generation++
	m.scope = ""
	m.snapshot = Snapshot{}
	m.mu.Unlock()
}
func (m *Monitor) DeliveryFailed(t Ticket) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentLocked(t) {
		m.snapshot.Fallback = true
	}
}
func (m *Monitor) Open(t Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.currentLocked(t) {
		return errStale
	}
	return m.source.Open(m.ctx, t.Scope)
}

// Continue resumes future queue consumption only after an explicit user action.
// It does not replay the unknown batch or invent an acknowledgment endpoint.
func (m *Monitor) Continue(t Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.currentLocked(t) || !m.snapshot.Paused {
		return errStale
	}
	key, err := m.source.Current(m.ctx)
	if err != nil || key != t.Scope {
		return errStale
	}
	delete(m.blocked, key)
	m.snapshot.Paused = false
	m.snapshot.Status = "已继续接收后续提醒。之前可能已认领但未显示的提醒，请在鲸桥管理页的提醒历史中查看。"
	return nil
}
func (m *Monitor) bindLocked(key string) {
	if key == m.scope {
		return
	}
	m.scope = key
	m.generation++
	m.snapshot = Snapshot{}
	if key != "" {
		m.snapshot.Manage = Ticket{Scope: key, Generation: m.generation}
		m.snapshot.Paused = m.blocked[key]
	}
}

// Observe retires stopped/replaced instance tickets without consuming the queue.
func (m *Monitor) Observe() {
	key, err := m.source.Current(m.ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed.Load() {
		return
	}
	if err != nil {
		m.snapshot.Status = "无法核验本机鲸桥实例，尚未认领新提醒；稍后会重新检查。"
		return
	}
	m.bindLocked(key)
}
func (m *Monitor) Poll() {
	m.mu.Lock()
	if m.closed.Load() || m.reading {
		m.mu.Unlock()
		return
	}
	m.reading = true
	m.mu.Unlock()
	defer func() { m.mu.Lock(); m.reading = false; m.mu.Unlock() }()
	key, err := m.source.Current(m.ctx)
	m.mu.Lock()
	if m.closed.Load() {
		m.mu.Unlock()
		return
	}
	if err != nil {
		m.snapshot.Status = "无法核验本机鲸桥实例，尚未认领新提醒；稍后会重新检查。"
		m.mu.Unlock()
		return
	}
	m.bindLocked(key)
	gen := m.generation
	if key == "" {
		m.mu.Unlock()
		return
	}
	if m.blocked[key] {
		m.snapshot.Paused = true
		m.snapshot.Status = uncertainStatus
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	batch, err := m.source.Claim(m.ctx, key)
	if err == nil {
		err = validateBatch(batch)
	}
	// Once a POST may have happened, any missing receipt is an unknown outcome.
	m.mu.Lock()
	if err != nil && !errors.Is(err, ErrNotClaimed) {
		m.blocked[key] = true
	}
	m.mu.Unlock()
	live, liveErr := m.source.Current(m.ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed.Load() || m.generation != gen || m.scope != key {
		return
	}
	if liveErr != nil || live != key {
		m.bindLocked("")
		return
	}
	if err != nil {
		if errors.Is(err, ErrNotClaimed) {
			m.snapshot.Status = "鲸桥提醒预检未完成，未认领新提醒；稍后会重新检查。"
		} else {
			m.snapshot.Paused = true
			m.snapshot.Status = uncertainStatus
		}
		return
	}
	sort.SliceStable(batch.Alerts, func(i, j int) bool { return batch.Alerts[i].Sequence < batch.Alerts[j].Sequence })
	cursor := m.cursors[key]
	for _, record := range batch.Alerts {
		if record.Sequence <= cursor {
			continue
		}
		title, body := Format(record.Alert)
		m.snapshot.Alerts = append(m.snapshot.Alerts, Notice{Ticket: Ticket{key, gen, record.Sequence}, Title: title, Body: body})
		cursor = record.Sequence
	}
	if batch.Sequence > cursor {
		cursor = batch.Sequence
	}
	m.cursors[key] = cursor
	if len(m.snapshot.Alerts) > 100 {
		n := len(m.snapshot.Alerts) - 100
		m.snapshot.Dropped += n
		m.snapshot.Alerts = append([]Notice(nil), m.snapshot.Alerts[n:]...)
	}
	if m.snapshot.Status != "" && !strings.HasPrefix(m.snapshot.Status, "已继续") {
		m.snapshot.Status = ""
	}
}

const uncertainStatus = "提醒队列认领结果未确认，已暂停接收。上一批可能已认领但未显示；鲸桥管理页保留提醒历史。确认后可继续接收后续提醒。"

func validateBatch(b Batch) error {
	if b.Sequence < 0 || b.Alerts == nil || len(b.Alerts) > 100 {
		return errors.New("invalid claim receipt")
	}
	for _, r := range b.Alerts {
		if r.Sequence <= 0 || r.Sequence > b.Sequence {
			return errors.New("invalid notice sequence")
		}
	}
	return nil
}
func plain(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s))
}

// Format preserves returned values, including zero and timestamps/time zones;
// absent fields are labeled rather than invented as zero, a date, or a balance.
func Format(a Alert) (string, string) {
	name := plain(a.Name)
	if name == "" {
		name = plain(a.Provider)
	}
	if user := plain(a.User); user != "" {
		if name != "" {
			name += " · "
		}
		name += user
	}
	title := "鲸桥"
	if name != "" {
		title += " · " + name
	}
	when := plain(a.ResetsAt)
	if when == "" {
		when = "时间未提供"
	}
	used := "用量未提供"
	if a.Used != "" {
		used = "已使用 " + plain(string(a.Used)) + "%"
	}
	window := plain(a.Window)
	if window == "" {
		window = "额度窗口"
	}
	switch plain(a.Kind) {
	case "expires":
		credits := plain(string(a.Credits))
		if credits == "" {
			credits = "数量未提供"
		}
		return title, fmt.Sprintf("%s 次额度重置将在 %s 到期。", credits, when)
	case "renews":
		return title, fmt.Sprintf("%s 将在 %s 重置，当前%s。", window, when, used)
	default:
		if a.Balance != "" {
			return title, "余额已降至 " + plain(string(a.Balance)) + "。"
		}
		body := window + " " + used + "。"
		if a.ResetsAt != "" {
			body += "下次重置：" + when + "。"
		}
		return title, body
	}
}

// Package botnotify observes real Bot activity. It performs no Host mutations,
// model work, permission changes, or native notification calls.
package botnotify

import (
	"context"
	"strings"
	"time"
)

const (
	MaxBots     = 200                     // original profile-ops.js activity request limit
	Interval    = 1500 * time.Millisecond // original contacts-view timer
	ReadTimeout = 10 * time.Second
)

type Run struct {
	RoutineID, RunID, Name, Status string
	Silent                         bool
	EndedAt                        int64
}

func (r Run) key() string {
	if r.RoutineID == "" || r.RunID == "" {
		return ""
	}
	return r.RoutineID + "\x00" + r.RunID
}
func (r Run) terminal() bool {
	return r.Status == "completed" || r.Status == "failed" || r.Status == "interrupted"
}

type Row struct {
	ID, SessionID, Name, Kind string
	Notify, Attention, Hidden bool
	ActivitySeq               int
	GoalPhase                 string
	Run                       Run
}
type Snapshot struct {
	Contacts    []ContactActivity
	Revision    int
	Unavailable int
	Rows        []Row
}
type Ticket struct {
	BotID, SessionID string
	Revision         int
	epoch, event     uint64
}
type Notice struct {
	Ticket Ticket
	Name   string
	Events []string
}
type observed struct {
	row       Row
	revision  int
	event     uint64
	mask      uint8
	runKey    string
	highRunAt int64
	runs      map[string]bool
}

// Tracker is owned by the window UI thread. Epochs and catalog revisions are
// local/source ownership fences, not invented server-side activity CAS values.
type Tracker struct {
	epoch, sequence uint64
	revision        int
	ready           bool
	rows            map[string]observed
}

func (t *Tracker) Clear() { t.epoch++; t.ready = false; t.rows = nil }
func (t *Tracker) Current(ticket Ticket) bool {
	o, ok := t.rows[ticket.BotID]
	return ok && t.ready && ticket.epoch == t.epoch && ticket.event != 0 && o.event == ticket.event && o.revision == ticket.Revision && o.row.SessionID == ticket.SessionID && o.row.Notify && o.mask != 0
}

// Target is a verified in-memory catalog target for navigation only.
func (t *Tracker) Target(ticket Ticket) (Row, bool) {
	if !t.Current(ticket) {
		return Row{}, false
	}
	return t.rows[ticket.BotID].row, true
}
func (t *Tracker) Observe(s Snapshot, focused, supported bool) []Notice {
	if s.Revision < 0 || len(s.Rows) > MaxBots || (t.ready && s.Revision < t.revision) {
		return nil
	}
	if !t.ready {
		t.epoch++
		t.rows = map[string]observed{}
		t.ready = true
	}
	t.revision = s.Revision
	live := map[string]bool{}
	var notices []Notice
	for _, row := range s.Rows {
		if row.ID == "" || row.SessionID == "" || live[row.ID] {
			continue
		}
		live[row.ID] = true
		old, exists := t.rows[row.ID]
		if exists && old.row.SessionID != row.SessionID {
			exists = false
		}
		if exists && row.ActivitySeq < old.row.ActivitySeq {
			// bot/activity has no CAS revision. A backwards inspection must not
			// reset a known activity baseline or create new transitions.
			if !row.Notify {
				old.row.Notify = false
				old.mask = 0
				t.rows[row.ID] = old
			}
			continue
		}
		current := old
		if !exists {
			current = observed{runs: map[string]bool{}}
		}
		current.row = row
		current.revision = s.Revision
		// A cleared condition retires the corresponding click ticket immediately.
		if !row.Attention {
			current.mask &^= 1
		}
		if row.GoalPhase != "blocked" {
			current.mask &^= 2
		}
		if row.Run.key() != current.runKey || !row.Run.terminal() || row.Run.Silent {
			current.mask &^= 4
		}
		if !row.Notify {
			current.mask = 0
		}
		var events []string
		var mask uint8
		if exists && row.Attention && !old.row.Attention {
			events = append(events, "Bot 需要处理待确认事项 / Bot needs your attention")
			mask |= 1
		}
		if exists && row.GoalPhase == "blocked" && old.row.GoalPhase != "blocked" {
			events = append(events, "Bot 目标遇到阻碍 / Bot objective is blocked")
			mask |= 2
		}
		run := row.Run
		key := run.key()
		if run.terminal() && key != "" {
			if run.EndedAt > current.highRunAt {
				current.highRunAt = run.EndedAt
				current.runs = map[string]bool{}
			}
			// A bounded high-water bucket suppresses replay after a transient empty run.
			// Terminal records are durable/immutable. Older timestamps cannot be new.
			unseen := run.EndedAt >= current.highRunAt && !current.runs[key] && len(current.runs) < MaxBots
			if unseen {
				current.runs[key] = true
			}
			if exists && unseen && !run.Silent && key != old.row.Run.key() {
				message := "例行任务结束但未成功 / Routine did not complete successfully"
				if run.Status == "completed" {
					message = "例行任务已完成 / Routine completed"
				}
				if name := strings.TrimSpace(run.Name); name != "" {
					message += ": " + name
				}
				events = append(events, message)
				mask |= 4
			} else if exists && unseen && !run.Silent && !old.row.Run.terminal() {
				message := "例行任务结束但未成功 / Routine did not complete successfully"
				if run.Status == "completed" {
					message = "例行任务已完成 / Routine completed"
				}
				if name := strings.TrimSpace(run.Name); name != "" {
					message += ": " + name
				}
				events = append(events, message)
				mask |= 4
			}
		}
		// Baselines advance even when foreground/disabled/unsupported. No replay on
		// blur, permission grant, or notify re-enable.
		if len(events) > 0 && row.Notify && !focused && supported {
			t.sequence++
			current.event = t.sequence
			current.mask = mask
			current.runKey = key
			notices = append(notices, Notice{Ticket: Ticket{row.ID, row.SessionID, s.Revision, t.epoch, t.sequence}, Name: row.Name, Events: events})
		}
		t.rows[row.ID] = current
	}
	for id := range t.rows {
		if !live[id] {
			delete(t.rows, id)
		}
	}
	return notices
}

// Poll is one serial reader, never a goroutine-per-tick. Its cadence starts only
// after the previous read/publish returns, and cancellation fences publication.
func Poll(ctx context.Context, interval, timeout time.Duration, read func(context.Context) (Snapshot, error), publish func(Snapshot, error)) {
	if interval <= 0 {
		interval = Interval
	}
	if timeout <= 0 {
		timeout = ReadTimeout
	}
	for {
		if ctx.Err() != nil {
			return
		}
		round, cancel := context.WithTimeout(ctx, timeout)
		snapshot, err := read(round)
		cancel()
		if ctx.Err() != nil {
			return
		}
		publish(snapshot, err)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// Package startup owns the application's readiness levels and the plan that
// decides when each subsystem starts. The shell reports facts (Host
// transport, $events, workbench baselines); this package turns them into one
// observable level per Host generation and releases deferred work against it
// (ARCHITECTURE.md §8.2: native first frame, Host Ready, workspace Ready).
package startup

import (
	"slices"
	"sync"
	"time"

	"whaleisle/internal/startuptrace"
	"whaleisle/internal/state"
)

// Level is how far one Host generation has come. Levels are ordered: a
// generation at EventsReady also has its transport.
type Level int

const (
	// NotReady: no Host generation, or one still starting.
	NotReady Level = iota
	// TransportReady: dsh web printed its URL and the token exchange set a
	// session cookie. RPC works; the mux route may still be registering.
	TransportReady
	// EventsReady: the $events stream delivered its ready frame on the
	// current carrier. Lost again when the carrier drops.
	EventsReady
	// WorkbenchReady: the default workspace is initialized and the first
	// session list of this events generation loaded.
	WorkbenchReady
)

func (l Level) String() string {
	switch l {
	case TransportReady:
		return "transport_ready"
	case EventsReady:
		return "events_ready"
	case WorkbenchReady:
		return "workbench_ready"
	default:
		return "not_ready"
	}
}

// Readiness is one published level for one Host generation.
type Readiness struct {
	Generation uint64
	Level      Level
	// At is when this level was reached.
	At time.Time
	// Reached holds the first time each level was reached in this generation,
	// indexed by Level; a reconnect does not reset it.
	Reached [WorkbenchReady + 1]time.Time
}

// Tracker publishes the readiness of the current Host generation. Reports for
// a superseded generation are ignored, so a late callback of a stopped Host
// cannot raise the new one's level.
type Tracker struct {
	view state.Published[Readiness]

	mu        sync.Mutex
	listeners []func(Readiness)
}

// NewTracker starts at NotReady, generation 0.
func NewTracker() *Tracker {
	t := &Tracker{}
	t.view.Publish(Readiness{})
	return t
}

// Current returns the published readiness without copying.
func (t *Tracker) Current() Readiness {
	if v := t.view.Load(); v != nil {
		return v.Value
	}
	return Readiness{}
}

// Changed is closed on the next publication.
func (t *Tracker) Changed() <-chan struct{} { return t.view.Changed() }

// OnChange registers fn for every later transition. fn runs on the reporting
// goroutine after the publication and must not block.
func (t *Tracker) OnChange(fn func(Readiness)) {
	if fn == nil {
		return
	}
	t.mu.Lock()
	t.listeners = append(t.listeners, fn)
	t.mu.Unlock()
}

// Begin opens Host generation gen at TransportReady. A zero gen, or a stop,
// is Reset.
func (t *Tracker) Begin(gen uint64) {
	t.set(func(cur Readiness) (Readiness, bool) {
		if gen == 0 || cur.Generation == gen && cur.Level >= TransportReady {
			return cur, false
		}
		next := Readiness{Generation: gen}
		return reach(next, TransportReady), true
	})
}

// Reset drops to NotReady: the Host stopped or failed.
func (t *Tracker) Reset() {
	t.set(func(cur Readiness) (Readiness, bool) {
		if cur.Level == NotReady {
			return cur, false
		}
		return Readiness{Generation: cur.Generation, At: time.Now()}, true
	})
}

// Reach raises generation gen to level. A lower or equal level, or another
// generation, is ignored. WorkbenchReady requires EventsReady first.
func (t *Tracker) Reach(gen uint64, level Level) bool {
	changed := false
	t.set(func(cur Readiness) (Readiness, bool) {
		if gen == 0 || cur.Generation != gen || level <= cur.Level || cur.Level == NotReady {
			return cur, false
		}
		if level == WorkbenchReady && cur.Level < EventsReady {
			return cur, false
		}
		changed = true
		return reach(cur, level), true
	})
	return changed
}

// Lose lowers generation gen to TransportReady when its events carrier
// drops; the next $events ready raises it again.
func (t *Tracker) Lose(gen uint64) {
	t.set(func(cur Readiness) (Readiness, bool) {
		if cur.Generation != gen || cur.Level <= TransportReady {
			return cur, false
		}
		cur.Level, cur.At = TransportReady, time.Now()
		return cur, true
	})
}

func reach(r Readiness, level Level) Readiness {
	now := time.Now()
	r.Level, r.At = level, now
	if r.Reached[level].IsZero() {
		r.Reached[level] = now
	}
	return r
}

func (t *Tracker) set(next func(Readiness) (Readiness, bool)) {
	t.mu.Lock()
	cur := t.Current()
	r, ok := next(cur)
	if !ok {
		t.mu.Unlock()
		return
	}
	t.view.Publish(r)
	listeners := slices.Clone(t.listeners)
	t.mu.Unlock()
	startuptrace.Mark("startup."+r.Level.String(), r.Level.String())
	for _, fn := range listeners {
		fn(r)
	}
}

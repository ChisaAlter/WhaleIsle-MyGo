package schedule

import (
	"context"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/storelocale"
)

// EventsOwner binds one existing Host $events stream, including reconnects that
// reuse a Client. Notifications only invalidate a catalog already opened by UI.
type EventsOwner struct {
	store      *Store
	client     *dsh.Client
	ctx        context.Context
	generation uint64
}

func (s *Store) BeginEvents(ctx context.Context, client *dsh.Client) *EventsOwner {
	if s == nil || ctx == nil || ctx.Err() != nil || client == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != client || ctx.Err() != nil {
		return nil
	}
	s.eventsGeneration++
	s.generation++ // Retire reads issued for the preceding stream.
	s.loading = false
	return &EventsOwner{store: s, client: client, ctx: ctx, generation: s.eventsGeneration}
}

func (s *Store) eventsOwnerCurrentLocked(owner *EventsOwner) bool {
	return owner != nil && owner.store == s && owner.client == s.client && owner.ctx != nil &&
		owner.ctx.Err() == nil && owner.generation == s.eventsGeneration
}

// OnEventsReady refreshes the retained catalog after a transport reset.
func (s *Store) OnEventsReady(owner *EventsOwner) {
	if s == nil || owner == nil {
		return
	}
	s.refresh(owner.ctx, "", owner)
}

// OnEmit consumes the real Cordis notification, not the per-session durable
// schedule/change event. No scheduling or write operation is initiated here.
func (s *Store) OnEmit(owner *EventsOwner, emit dsh.EventsEmit) {
	if emit.Type == dsh.EventEmit && emit.Event == "schedule/changed" {
		s.OnEventsReady(owner)
	}
}

// FilterEntries derives a presentation copy; original identity and CAS records
// remain untouched, including rows hidden by the current query.
func FilterEntries(rows []Entry, query, status string) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]Entry, 0, len(rows))
	for _, row := range rows {
		if status != "" && status != "all" && row.Status != status {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(row.Title), query) &&
			!strings.Contains(strings.ToLower(row.Prompt), query) && !strings.Contains(strings.ToLower(row.SessionID), query) {
			continue
		}
		out = append(out, row)
	}
	sort.SliceStable(out, func(i, j int) bool {
		left, le := time.Parse(time.RFC3339Nano, out[i].ScheduledAt)
		right, re := time.Parse(time.RFC3339Nano, out[j].ScheduledAt)
		if le != nil {
			return false
		}
		if re != nil {
			return true
		}
		return left.Before(right)
	})
	return out
}

// TimingLabel displays the Host's committed next instant. It never calculates
// another occurrence locally, which could disagree with Host DST/cron rules.
func TimingLabel(row Entry) string {
	if row.Status == "inactive" {
		return storelocale.T(storelocale.KeyStoreScheduleEndedNoFutureDeliveries)
	}
	if row.Status != "active" {
		return storelocale.T(storelocale.KeyStoreScheduleUnknownStatusRefreshToVerify)
	}
	at, err := time.Parse(time.RFC3339Nano, row.ScheduledAt)
	if err != nil {
		return storelocale.T(storelocale.KeyStoreScheduleNextRunUnavailableRefresh)
	}
	zone := strings.TrimSpace(row.TimeZone)
	if zone == "" {
		zone = "UTC"
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return storelocale.T(storelocale.KeyStoreScheduleNextRun) + at.UTC().Format(time.RFC3339Nano) + storelocale.T(storelocale.KeyStoreScheduleRuleZoneUnavailable) + zone
	}
	return storelocale.T(storelocale.KeyStoreScheduleNextRun) + at.In(loc).Format("2006-01-02 15:04:05.999 -07:00") + " · " + zone
}

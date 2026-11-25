package main

import (
	"sync"
	"time"
)

const settingsTabSWRTTL = 30 * time.Second

type settingsTabOwner struct {
	gen   int
	ref   any
	scope string
}

type settingsTabSWREntry struct {
	owner       any
	refreshedAt time.Time
}

// settingsTabSWR implements a short-TTL stale-while-revalidate gate for
// Settings tabs (models, plugins, skills, presets, market, usage, MCP).
// Returning to a tab within TTL renders its cached store data immediately
// without firing redundant RPC storms; explicit refresh buttons bypass the
// gate and advance the timestamp.
type settingsTabSWR struct {
	mu      sync.Mutex
	ttl     time.Duration
	now     func() time.Time
	entries map[string]settingsTabSWREntry
}

func newSettingsTabSWR(ttl time.Duration) *settingsTabSWR {
	if ttl <= 0 {
		ttl = settingsTabSWRTTL
	}
	return &settingsTabSWR{
		ttl:     ttl,
		now:     time.Now,
		entries: map[string]settingsTabSWREntry{},
	}
}

// ShouldRefresh reports whether tab should fetch fresh data for owner.
// When it returns true it also records the refresh timestamp so immediate
// re-entries within TTL reuse the in-flight/cached store state.
func (c *settingsTabSWR) ShouldRefresh(tab string, owner any) bool {
	if c == nil {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]settingsTabSWREntry{}
	}
	now := c.clock()
	if entry, ok := c.entries[tab]; ok && entry.owner == owner && now.Sub(entry.refreshedAt) < c.effectiveTTL() {
		return false
	}
	c.entries[tab] = settingsTabSWREntry{owner: owner, refreshedAt: now}
	return true
}

// MarkRefreshed records an explicit refresh for tab/owner so switching away
// and returning within TTL reuses the freshly fetched data.
func (c *settingsTabSWR) MarkRefreshed(tab string, owner any) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]settingsTabSWREntry{}
	}
	c.entries[tab] = settingsTabSWREntry{owner: owner, refreshedAt: c.clock()}
}

// Invalidate forces the next visit to tab to refresh.
func (c *settingsTabSWR) Invalidate(tab string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.entries, tab)
	c.mu.Unlock()
}

// Reset clears all cached tab timestamps (for example on Host restart).
func (c *settingsTabSWR) Reset() {
	if c == nil {
		return
	}
	c.mu.Lock()
	clear(c.entries)
	c.mu.Unlock()
}

func (c *settingsTabSWR) clock() time.Time {
	if c.now != nil {
		return c.now()
	}
	return time.Now()
}

func (c *settingsTabSWR) effectiveTTL() time.Duration {
	if c.ttl > 0 {
		return c.ttl
	}
	return settingsTabSWRTTL
}

func (s *mainState) settingsTabSWRStore() *settingsTabSWR {
	s.settingsTabCacheOnce.Do(func() {
		if s.settingsTabCache == nil {
			s.settingsTabCache = newSettingsTabSWR(settingsTabSWRTTL)
		}
	})
	return s.settingsTabCache
}

func (s *mainState) settingsTabShouldRefresh(tab string, owner any) bool {
	return s.settingsTabSWRStore().ShouldRefresh(tab, owner)
}

func (s *mainState) settingsTabMarkRefreshed(tab string, owner any) {
	s.settingsTabSWRStore().MarkRefreshed(tab, owner)
}

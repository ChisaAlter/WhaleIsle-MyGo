package pluginmanager

import (
	"context"
	"errors"
	"strings"
)

func (c *Controller) EnsureDiscover(owner uint64) {
	c.mu.Lock()
	needed := c.state.Owner == owner && c.state.Discover.Status == "idle"
	if needed {
		c.state.Discover.Status = "loading"
	}
	c.mu.Unlock()
	if needed {
		go c.LoadIndexes(context.Background(), owner)
	}
}
func (a Actions) ReloadIndexes() { go a.c.LoadIndexes(context.Background(), a.owner) }
func (c *Controller) LoadIndexes(ctx context.Context, expected uint64) error {
	client, owner, ctx, done, err := c.capture(ctx)
	if err != nil {
		return err
	}
	defer done()
	if owner != expected {
		return errors.New("Plugin owner changed")
	}
	var read uint64
	c.patch(owner, func(s *State) {
		c.indexRead++
		read = c.indexRead
		s.Discover.Status = "loading"
		s.Discover.Error = ""
	})
	indexes, err := client.Indexes(ctx)
	selected := ""
	load := false
	c.patch(owner, func(s *State) {
		if c.indexRead != read {
			return
		}
		d := &s.Discover
		if err != nil {
			d.Status = "error"
			d.Error = err.Error()
			return
		}
		d.Status = "ready"
		d.Indexes = indexes
		for _, index := range indexes {
			if index.URL == d.Selected {
				selected = d.Selected
			}
		}
		if selected == "" && len(indexes) > 0 {
			selected = indexes[0].URL
		}
		load = selected != "" && (d.Listing == nil || d.Selected != selected)
		d.Selected = selected
		if selected == "" {
			d.Listing = nil
			d.SelectedEntry = ""
		}
	})
	if err == nil && load {
		return c.loadSearch(ctx, owner, selected, Query{Sort: MarketSort{Kind: "index"}, Limit: 50}, false)
	}
	return err
}
func (a Actions) SelectIndex(url string) {
	a.c.mu.Lock()
	if a.c.state.Owner != a.owner {
		a.c.mu.Unlock()
		return
	}
	valid := false
	for _, index := range a.c.state.Discover.Indexes {
		if index.URL == url {
			valid = true
			break
		}
	}
	a.c.mu.Unlock()
	if valid {
		a.c.queueSearch(a.owner, url, Query{Sort: MarketSort{Kind: "index"}, Limit: 50}, false)
	}
}
func (a Actions) Search(query Query) {
	s := a.c.Snapshot()
	if s.Owner != a.owner || s.Discover.Selected == "" {
		return
	}
	query.Offset = 0
	query.Limit = 50
	query.FetchedAt = 0
	a.c.queueSearch(a.owner, s.Discover.Selected, query, false)
}
func (a Actions) RefreshDiscover() {
	s := a.c.Snapshot()
	if s.Owner != a.owner || s.Discover.Selected == "" {
		return
	}
	q := s.Discover.Query
	q.Offset = 0
	q.Limit = 50
	q.FetchedAt = 0
	q.Refresh = true
	a.c.queueSearch(a.owner, s.Discover.Selected, q, false)
}
func (a Actions) More() {
	s := a.c.Snapshot()
	d := s.Discover
	if s.Owner != a.owner || d.Listing == nil || d.Listing.NextOffset == nil || d.Loading || d.MoreLoading {
		return
	}
	q := d.Query
	q.Offset = *d.Listing.NextOffset
	q.Limit = 50
	q.Refresh = false
	q.FetchedAt = d.Listing.FetchedAt
	a.c.queueSearch(a.owner, d.Selected, q, true)
}

// Reserve the request in the input callback, before starting its goroutine.
// Rapid edits therefore keep user order even if goroutines start out of order.
func (c *Controller) queueSearch(owner uint64, url string, q Query, more bool) {
	c.mu.Lock()
	if c.state.Owner != owner || c.client == nil {
		c.mu.Unlock()
		return
	}
	c.listingRead++
	read := c.listingRead
	if c.state.Discover.Selected != url {
		c.state.Discover.Listing = nil
		c.state.Discover.SelectedEntry = ""
	}
	c.state.Discover.Selected = url
	visible := q
	visible.Offset = 0
	visible.FetchedAt = 0
	visible.Refresh = false
	c.state.Discover.Query = visible
	c.state.Discover.Loading = !more
	c.state.Discover.MoreLoading = more
	c.mu.Unlock()
	c.fire()
	go c.loadSearch(context.Background(), owner, url, q, more, read)
}
func (c *Controller) loadSearch(ctx context.Context, expected uint64, url string, q Query, more bool, reserved ...uint64) error {
	client, owner, ctx, done, err := c.capture(ctx)
	if err != nil {
		return err
	}
	defer done()
	if owner != expected {
		return errors.New("Plugin owner changed")
	}
	c.mu.Lock()
	if c.state.Owner != owner {
		c.mu.Unlock()
		return errors.New("Plugin owner changed")
	}
	if len(reserved) > 0 {
		if c.listingRead != reserved[0] {
			c.mu.Unlock()
			return nil
		}
	} else {
		c.listingRead++
	}
	read := c.listingRead
	d := &c.state.Discover
	if d.Selected != url {
		d.Listing = nil
		d.SelectedEntry = ""
	}
	d.Selected = url
	visible := q
	visible.Offset = 0
	visible.FetchedAt = 0
	visible.Refresh = false
	d.Query = visible
	d.Error = ""
	d.Loading = !more
	d.MoreLoading = more
	c.mu.Unlock()
	c.fire()
	result, err := client.Search(ctx, url, q)
	c.mu.Lock()
	if c.state.Owner != owner || read != c.listingRead {
		c.mu.Unlock()
		return err
	}
	d = &c.state.Discover
	d.Loading = false
	d.MoreLoading = false
	if err != nil {
		d.Error = err.Error()
	} else if more && d.Listing != nil && d.Listing.FetchedAt != result.FetchedAt {
		err = errors.New("Plugin index changed while loading more. Refresh the listing.")
		d.Error = err.Error()
	} else {
		if more && d.Listing != nil {
			result.Entries = append(append([]MarketEntry{}, d.Listing.Entries...), result.Entries...)
		}
		d.Listing = &result
		d.Status = "ready"
	}
	c.mu.Unlock()
	c.fire()
	return err
}
func (a Actions) OpenEntry(name string) {
	a.edit(func(s *State) {
		if s.Discover.Listing != nil {
			for _, entry := range s.Discover.Listing.Entries {
				if entry.Name == name {
					s.Discover.SelectedEntry = name
					return
				}
			}
		}
	})
}
func (a Actions) CloseEntry() { a.edit(func(s *State) { s.Discover.SelectedEntry = "" }) }
func (a Actions) ToggleSources() {
	a.edit(func(s *State) { s.Discover.SourcesOpen = !s.Discover.SourcesOpen })
}
func (a Actions) EditSource(value string) { a.edit(func(s *State) { s.Discover.Draft = value }) }
func (a Actions) AddSource() {
	s := a.c.Snapshot()
	if s.Owner != a.owner || strings.TrimSpace(s.Discover.Draft) == "" {
		return
	}
	urls := make([]string, 0, len(s.Discover.Indexes)+1)
	for _, index := range s.Discover.Indexes {
		urls = append(urls, index.URL)
	}
	urls = append(urls, strings.TrimSpace(s.Discover.Draft))
	a.setSources(urls)
}
func (a Actions) RemoveSource(url string) {
	s := a.c.Snapshot()
	if s.Owner != a.owner {
		return
	}
	urls := []string{}
	for _, index := range s.Discover.Indexes {
		if index.URL != url {
			urls = append(urls, index.URL)
		}
	}
	a.setSources(urls)
}
func (a Actions) setSources(urls []string) {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner || c.client == nil || c.state.Busy {
		c.mu.Unlock()
		return
	}
	client, ctx := Client{c.client}, c.life
	c.state.Busy = true
	c.mu.Unlock()
	c.fire()
	go func() {
		err := client.SetIndexes(ctx, urls)
		if !c.patch(a.owner, func(s *State) {
			s.Busy = false
			if err != nil {
				s.Discover.Error = err.Error()
			} else {
				s.Discover.Draft = ""
			}
		}) {
			return
		}
		if err == nil {
			_ = c.LoadIndexes(ctx, a.owner)
		}
	}()
}
func (a Actions) InstallVersion(name, version string) {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner || c.state.Discover.Listing == nil {
		c.mu.Unlock()
		return
	}
	for _, entry := range c.state.Discover.Listing.Entries {
		if entry.Name == name {
			for _, v := range entry.Versions {
				if v.Version == version && (v.Compatibility.Compatible || v.Compatibility.Exempted) {
					spec, registry := v.Install.Spec, ""
					if v.Install.Registry != nil {
						registry = *v.Install.Registry
					}
					enabled := false
					c.mu.Unlock()
					c.openInstall(a.owner, spec, registry, &enabled, true)
					return
				}
			}
		}
	}
	c.mu.Unlock()
}

package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
	"whaleisle/internal/dsh"
)

const discoverPageSize = 50

type discoverRead struct {
	url   string
	query Query
	more  bool
}

func initialQuery() Query { return Query{Sort: MarketSort{Kind: "index"}, Limit: discoverPageSize} }
func visibleQuery(q Query) Query {
	q.Offset, q.FetchedAt, q.Refresh = 0, 0, false
	q.Limit = discoverPageSize
	return q
}
func sameQuery(a, b Query) bool { return reflect.DeepEqual(visibleQuery(a), visibleQuery(b)) }
func WithText(q Query, text string) Query {
	was, blank := strings.TrimSpace(q.Text) == "", strings.TrimSpace(text) == ""
	if was && !blank && q.Sort.Kind == "index" {
		q.Sort = MarketSort{Kind: "relevance"}
	} else if blank && q.Sort.Kind == "relevance" {
		q.Sort = MarketSort{Kind: "index"}
	}
	q.Text = text
	return q
}
func declaredQuery(q Query, metadata MarketMetadata) Query {
	if q.Category != "" {
		if _, ok := metadata.Categories[q.Category]; !ok {
			q.Category = ""
		}
	}
	labels := []string(nil)
	for _, id := range q.Labels {
		if _, ok := metadata.Labels[id]; ok {
			labels = append(labels, id)
		}
	}
	q.Labels = labels
	if q.Sort.Kind == "key" {
		if _, ok := metadata.SortKeys[q.Sort.ID]; !ok {
			q.Sort = MarketSort{Kind: "index"}
		}
	}
	return q
}
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
func (c *Controller) SourcesChanged(owner uint64) {
	s := c.Snapshot()
	if s.Owner == owner && s.Discover.Status != "idle" {
		go c.LoadIndexes(context.Background(), owner)
	}
}
func (c *Controller) BundlesChanged(owner uint64) {
	s := c.Snapshot()
	d := s.Discover
	if s.Owner != owner || d.Listing == nil || d.Loading || d.MoreLoading || d.Error != "" {
		return
	}
	q := d.Query
	q.Limit = max(discoverPageSize, len(d.Listing.Entries))
	c.queueSearch(owner, d.Selected, q, false)
}
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
		if s.Discover.Status == "idle" {
			s.Discover.Status = "loading"
		}
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
			d.IndexesError = err.Error()
			return
		}
		d.Status = "ready"
		d.IndexesError = ""
		d.Indexes = indexes
		if c.selectAdded && len(indexes) > 0 {
			selected = indexes[len(indexes)-1].URL
		} else {
			for _, index := range indexes {
				if index.URL == d.Selected {
					selected = d.Selected
				}
			}
		}
		c.selectAdded = false
		if selected == "" && len(indexes) > 0 {
			selected = indexes[0].URL
		}
		load = selected != "" && (d.Listing == nil || d.Selected != selected)
		if d.Selected != selected {
			d.Listing = nil
			d.Answered = nil
			d.SelectedEntry = ""
			d.Opened = nil
		}
		d.Selected = selected
		if selected == "" {
			c.listingRead++
			d.Listing = nil
			d.Answered = nil
			d.SelectedEntry = ""
			d.Opened = nil
			d.Loading = false
			d.MoreLoading = false
			d.SourcesOpen = false
		}
	})
	if err == nil && load {
		return c.loadSearch(ctx, owner, selected, initialQuery(), false)
	}
	return err
}
func (a Actions) SelectIndex(url string) {
	s := a.c.Snapshot()
	if s.Owner != a.owner || s.Discover.Selected == url {
		return
	}
	for _, index := range s.Discover.Indexes {
		if index.URL == url {
			a.c.queueSearch(a.owner, url, initialQuery(), false)
			return
		}
	}
}
func (a Actions) Search(q Query) {
	s := a.c.Snapshot()
	if s.Owner != a.owner || s.Discover.Selected == "" {
		return
	}
	q = visibleQuery(q)
	a.c.queueSearch(a.owner, s.Discover.Selected, q, false)
}

// Typing is local for 250 ms, as in Discover's browser input effect.
func (a Actions) SearchText(text string) {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner || c.state.Discover.Selected == "" {
		c.mu.Unlock()
		return
	}
	c.cancelDiscoverText()
	serial := c.textRead
	c.state.Discover.TextDraft = text
	c.discoverTextTimer = time.AfterFunc(250*time.Millisecond, func() {
		c.mu.Lock()
		if c.state.Owner != a.owner || c.textRead != serial {
			c.mu.Unlock()
			return
		}
		q := WithText(c.state.Discover.Query, text)
		url := c.state.Discover.Selected
		c.listingRead++
		read := c.listingRead
		c.beginDiscover(url, q, false)
		c.mu.Unlock()
		c.fire()
		go c.loadSearch(context.Background(), a.owner, url, q, false, read)
	})
	c.mu.Unlock()
	c.fire()
}

// Called with the controller mutex held.
func (c *Controller) cancelDiscoverText() {
	c.textRead++
	if c.discoverTextTimer != nil {
		c.discoverTextTimer.Stop()
		c.discoverTextTimer = nil
	}
}
func (a Actions) RefreshDiscover() {
	s := a.c.Snapshot()
	if s.Owner != a.owner || s.Discover.Selected == "" {
		return
	}
	q := visibleQuery(s.Discover.Query)
	q.Refresh = true
	a.c.queueSearch(a.owner, s.Discover.Selected, q, false)
}
func (a Actions) RetryDiscover() {
	c := a.c
	c.mu.Lock()
	var retry *discoverRead
	if c.state.Owner == a.owner && c.failedDiscover != nil {
		value := *c.failedDiscover
		retry = &value
	}
	c.mu.Unlock()
	if retry != nil {
		c.queueSearch(a.owner, retry.url, retry.query, retry.more)
	}
}
func (a Actions) More() {
	s := a.c.Snapshot()
	d := s.Discover
	if s.Owner != a.owner || d.Listing == nil || d.Listing.NextOffset == nil || d.Loading || d.MoreLoading || d.Answered == nil || !sameQuery(d.Query, *d.Answered) {
		return
	}
	q := d.Query
	q.Offset = len(d.Listing.Entries)
	q.Limit = discoverPageSize
	q.FetchedAt = d.Listing.FetchedAt
	a.c.queueSearch(a.owner, d.Selected, q, true)
}

// Reserve in the input callback; a late goroutine cannot overtake a newer edit.
func (c *Controller) queueSearch(owner uint64, url string, q Query, more bool) {
	c.mu.Lock()
	if c.state.Owner != owner || c.client == nil {
		c.mu.Unlock()
		return
	}
	c.listingRead++
	read := c.listingRead
	c.cancelDiscoverText()
	c.beginDiscover(url, q, more)
	c.mu.Unlock()
	c.fire()
	go c.loadSearch(context.Background(), owner, url, q, more, read)
}
func (c *Controller) beginDiscover(url string, q Query, more bool) {
	d := &c.state.Discover
	if d.Selected != url {
		d.Listing = nil
		d.Answered = nil
		d.SelectedEntry = ""
		d.Opened = nil
	}
	d.Selected = url
	d.Query = visibleQuery(q)
	d.TextDraft = q.Text
	d.Error = ""
	d.Loading = !more
	d.MoreLoading = more
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
		return context.Canceled
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
	c.beginDiscover(url, q, more)
	kept := c.state.Discover.Listing
	c.mu.Unlock()
	c.fire()
	count := q.Limit
	if count <= 0 {
		count = discoverPageSize
	}
	var value *SearchResult
	restarted := false
	if more && kept != nil {
		copy := *kept
		copy.Entries = append([]MarketEntry(nil), kept.Entries...)
		value = &copy
		count = len(kept.Entries) + discoverPageSize
	}
	for {
		offset, fetched := 0, int64(0)
		if value != nil {
			offset, fetched = len(value.Entries), value.FetchedAt
		}
		page := q
		page.Offset = offset
		page.Limit = min(count-offset, 100)
		page.FetchedAt = fetched
		page.Refresh = q.Refresh && value == nil && !restarted
		result, e := client.Search(ctx, url, page)
		c.mu.Lock()
		current := c.state.Owner == owner && c.listingRead == read
		c.mu.Unlock()
		if !current {
			return e
		}
		if e != nil {
			err = e
			break
		}
		if value != nil && value.FetchedAt != result.FetchedAt {
			if more {
				more = false
				value = nil
				q.Limit = count
				q.Offset = 0
				q.FetchedAt = 0
				q.Refresh = false
				continue
			}
			if restarted {
				err = errors.New("Plugin index changed while loading pages. Retry the listing.")
				break
			}
			restarted = true
			value = nil
			continue
		}
		if value != nil {
			result.Entries = append(value.Entries, result.Entries...)
		}
		value = &result
		if len(value.Entries) >= count || value.NextOffset == nil {
			break
		}
	}
	if err == nil {
		declared := declaredQuery(visibleQuery(q), value.Metadata)
		if !sameQuery(declared, q) {
			declared.Limit = count
			c.mu.Lock()
			if c.state.Owner == owner && c.listingRead == read {
				c.mu.Unlock()
				return c.loadSearch(ctx, owner, url, declared, false, read)
			}
			c.mu.Unlock()
			return nil
		}
	}
	c.mu.Lock()
	if c.state.Owner != owner || c.listingRead != read {
		c.mu.Unlock()
		return err
	}
	d := &c.state.Discover
	d.Loading = false
	d.MoreLoading = false
	if err != nil {
		d.Error = err.Error()
		c.failedDiscover = &discoverRead{url, q, more}
	} else {
		d.Listing = value
		answered := visibleQuery(q)
		d.Answered = &answered
		d.Status = "ready"
		c.failedDiscover = nil
		if d.SelectedEntry != "" {
			for _, entry := range value.Entries {
				if entry.Name == d.SelectedEntry {
					copy := entry
					d.Opened = &copy
					break
				}
			}
		}
	}
	for i := range d.Indexes {
		if d.Indexes[i].URL == url {
			if err != nil {
				d.Indexes[i].Error = err.Error()
			} else {
				d.Indexes[i].Title = value.Title
				d.Indexes[i].Description = value.Description
				d.Indexes[i].Error = ""
			}
		}
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
					a.c.cancelDiscoverText()
					s.Discover.SelectedEntry = name
					copy := entry
					s.Discover.Opened = &copy
					return
				}
			}
		}
	})
}
func (a Actions) CloseEntry() {
	a.edit(func(s *State) { s.Discover.SelectedEntry = ""; s.Discover.Opened = nil })
}
func (a Actions) ToggleSources() {
	a.edit(func(s *State) { s.Discover.SourcesOpen = !s.Discover.SourcesOpen; s.Discover.SourceError = "" })
}
func (a Actions) EditSource(value string) {
	a.edit(func(s *State) { s.Discover.Draft = value; s.Discover.SourceError = "" })
}
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
	a.setSources(urls, true)
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
	a.setSources(urls, false)
}
func sourceError(err error) string {
	var remote *dsh.Error
	if errors.As(err, &remote) {
		switch remote.Code {
		case "plugin-market/invalid-index-url":
			var details struct {
				Reason string `json:"reason"`
			}
			_ = json.Unmarshal(remote.Details, &details)
			if details.Reason == "duplicate" {
				return "This index source is already configured."
			}
			return "Enter a valid HTTP or HTTPS index URL."
		case "plugin-market/settings-unavailable":
			return "Index sources cannot be changed in this profile."
		}
	}
	return err.Error()
}
func (a Actions) setSources(urls []string, adding bool) {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner || c.client == nil || c.state.Discover.Saving {
		c.mu.Unlock()
		return
	}
	client, ctx := Client{c.client}, c.life
	c.state.Discover.Saving = true
	c.state.Discover.SourceError = ""
	c.selectAdded = adding
	c.mu.Unlock()
	c.fire()
	go func() {
		err := client.SetIndexes(ctx, urls)
		if !c.patch(a.owner, func(s *State) {
			s.Discover.Saving = false
			if err != nil {
				c.selectAdded = false
				s.Discover.SourceError = sourceError(err)
				s.Discover.SourceErrorSequence++
			} else if adding {
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
	entries := c.state.Discover.Listing.Entries
	if opened := c.state.Discover.Opened; opened != nil && opened.Name == name {
		entries = []MarketEntry{*opened}
	}
	for _, entry := range entries {
		if entry.Name == name && entry.Installation.Status != "provided" {
			for _, v := range entry.Versions {
				if v.Version == version && entry.Installation.InstalledVersion != version && (v.Compatibility.Compatible || v.Compatibility.Exempted) {
					c.state.Discover.Choice = &MarketChoice{entry, v}
					c.mu.Unlock()
					c.fire()
					return
				}
			}
		}
	}
	c.mu.Unlock()
}
func (a Actions) CancelMarketChoice() { a.edit(func(s *State) { s.Discover.Choice = nil }) }
func (a Actions) ConfirmMarketChoice() {
	c := a.c
	c.mu.Lock()
	if c.state.Owner != a.owner || c.state.Discover.Choice == nil {
		c.mu.Unlock()
		return
	}
	choice := *c.state.Discover.Choice
	c.state.Discover.Choice = nil
	c.mu.Unlock()
	registry := ""
	if choice.Version.Install.Registry != nil {
		registry = *choice.Version.Install.Registry
	}
	if !c.openInstall(a.owner, choice.Version.Install.Spec, registry, nil, false) {
		return
	}
	c.patch(a.owner, func(s *State) {
		s.Install.Confirmed = true
		s.Install.Title = choice.Entry.Name
		s.Install.ConfirmedTitle = choice.Entry.Title
		s.Install.Version = choice.Version.Version
	})
	a.startConfirmedInstall()
}

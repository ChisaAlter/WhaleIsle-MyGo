package dshbot

import (
	"sort"
	"strings"
	"sync"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/store/storelocale"
)

type ContactFilters struct {
	Query, Kind, Source, Activity string
	ShowHidden                    bool
}
type ContactSource struct{ ID, Label string }
type ContactGroup struct {
	ID, Name string
	Items    []pluginrpc.DshbotBotItem
}
type ContactSnapshot struct {
	Scope                                                      string
	Generation                                                 uint64
	Loaded, Writable, Busy                                     bool
	Catalog                                                    pluginrpc.DshbotContactCatalog
	Filters                                                    ContactFilters
	EditingSection, SectionName, AssignmentItem, DeleteSection string
	Undo                                                       *pluginrpc.DshbotSectionSnapshot
	Pending                                                    *pluginrpc.DshbotContactCommand
	Error, Notice                                              string
}
type ContactTicket struct {
	Scope                string
	Generation, Sequence uint64
}
type contactEntry struct {
	state    ContactSnapshot
	sequence uint64
	attempt  uint64
}
type ContactControls struct {
	mu                   sync.Mutex
	active, binding      string
	generation, sequence uint64
	entries              map[string]*contactEntry
}

func (c *ContactControls) entry(k string) *contactEntry {
	if c.entries == nil {
		c.entries = map[string]*contactEntry{}
	}
	if c.entries[k] == nil {
		c.entries[k] = &contactEntry{state: ContactSnapshot{Scope: k}}
	}
	return c.entries[k]
}
func (c *ContactControls) Bind(k, b string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == k && c.binding == b {
		return
	}
	c.generation++
	c.active, c.binding = k, b
	if k != "" {
		e := c.entry(k)
		e.state.Loaded = false
		e.state.Busy = false
		e.sequence = 0
	}
}
func (c *ContactControls) Suspend() { c.Bind("", "") }
func (c *ContactControls) ViewCurrent(k string, g uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return k != "" && k == c.active && c.binding != "" && g == c.generation
}
func (c *ContactControls) Snapshot(k string) ContactSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := ContactSnapshot{Scope: k, Generation: c.generation}
	if e := c.entries[k]; e != nil {
		out = e.state
		out.Generation = c.generation
	}
	out.Catalog = pluginrpc.CloneDshbotContactCatalog(out.Catalog)
	out.Undo = pluginrpc.CloneDshbotSectionSnapshot(out.Undo)
	if out.Pending != nil {
		cmd := pluginrpc.CloneDshbotContactCommand(*out.Pending)
		out.Pending = &cmd
	}
	return out
}
func (c *ContactControls) Change(k, field, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k == "" || k != c.active {
		return
	}
	s := &c.entry(k).state
	switch field {
	case "query":
		s.Filters.Query = value
	case "kind":
		s.Filters.Kind = value
	case "source":
		s.Filters.Source = value
	case "activity":
		s.Filters.Activity = value
	case "hidden":
		s.Filters.ShowHidden = value == "true"
	case "reset":
		s.Filters = ContactFilters{}
	case "name":
		s.SectionName = value
	case "edit":
		s.EditingSection = value
		s.SectionName = ""
		for _, row := range s.Catalog.Sections {
			if row.ID == value {
				s.SectionName = row.Name
				break
			}
		}
	case "assign":
		s.AssignmentItem = value
	case "delete":
		s.DeleteSection = value
	}
}
func (c *ContactControls) BeginLoad(k string) (ContactTicket, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k == "" || k != c.active || c.binding == "" {
		return ContactTicket{}, false
	}
	e := c.entry(k)
	if e.state.Busy {
		return ContactTicket{}, false
	}
	c.sequence++
	e.sequence = c.sequence
	e.state.Busy = true
	e.state.Error = ""
	return ContactTicket{k, c.generation, c.sequence}, true
}
func (c *ContactControls) ticket(t ContactTicket) *contactEntry {
	e := c.entries[t.Scope]
	if t.Scope == "" || t.Scope != c.active || t.Generation != c.generation || e == nil || e.sequence != t.Sequence {
		return nil
	}
	return e
}
func (c *ContactControls) FinishLoad(t ContactTicket, v pluginrpc.DshbotContactCatalog, w bool, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.ticket(t)
	if e == nil {
		return false
	}
	e.state.Busy = false
	e.state.Writable = w
	if err != nil {
		e.state.Error = err.Error()
		return true
	}
	e.state.Catalog = pluginrpc.CloneDshbotContactCatalog(v)
	e.state.Loaded = true
	e.state.Error = ""
	if p := e.state.Pending; p != nil {
		if pluginrpc.DshbotContactApplied(*p, v) {
			c.applySuccess(e, p, nil)
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotRefreshedCatalogConfirmsTheOperation)
		} else {
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotOutcomeUnknownRefreshWithoutResendin)
		}
	}
	return true
}
func (c *ContactControls) Begin(k string, cmd pluginrpc.DshbotContactCommand) (ContactTicket, pluginrpc.DshbotContactCommand, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k == "" || k != c.active || c.binding == "" {
		return ContactTicket{}, cmd, false
	}
	e := c.entry(k)
	if e.state.Busy || !e.state.Loaded || !e.state.Writable || e.state.Pending != nil || cmd.Expected.Revision != e.state.Catalog.Revision {
		return ContactTicket{}, cmd, false
	}
	c.sequence++
	e.sequence = c.sequence
	e.state.Busy = true
	e.state.Error = ""
	e.state.Notice = ""
	cmd = pluginrpc.CloneDshbotContactCommand(cmd)
	e.state.Pending = &cmd
	e.attempt = c.sequence
	return ContactTicket{k, c.generation, c.sequence}, cmd, true
}
func (c *ContactControls) applySuccess(e *contactEntry, cmd *pluginrpc.DshbotContactCommand, snapshot *pluginrpc.DshbotSectionSnapshot) {
	switch cmd.Action {
	case "delete":
		if snapshot == nil {
			snapshot = pluginrpc.DshbotContactSectionSnapshot(cmd.Expected, cmd.ID)
		}
		e.state.Undo = pluginrpc.CloneDshbotSectionSnapshot(snapshot)
		e.state.DeleteSection = ""
	case "restore":
		e.state.Undo = nil
	case "create", "rename":
		e.state.EditingSection = ""
		e.state.SectionName = ""
	case "assign":
		e.state.AssignmentItem = ""
	}
	e.state.Pending = nil
	e.attempt = 0
}
func (c *ContactControls) Finish(t ContactTicket, cmd pluginrpc.DshbotContactCommand, result pluginrpc.DshbotContactResult, err error) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[t.Scope]
	if e == nil || e.attempt != t.Sequence {
		return false
	}
	e.state.Busy = false
	if err != nil {
		e.state.Error = err.Error()
		if !result.Attempted || result.Rejected {
			e.attempt = 0
			e.state.Pending = nil
			e.state.Loaded = false
		} else {
			e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotOutcomeUnknownRefreshToReconcile)
		}
		return true
	}
	e.state.Catalog = pluginrpc.CloneDshbotContactCatalog(result.Catalog)
	e.state.Loaded = true
	c.applySuccess(e, &cmd, result.Snapshot)
	e.state.Notice = storelocale.T(storelocale.KeyStoreDshbotDirectorySaved)
	return true
}
func ContactSources(v pluginrpc.DshbotContactCatalog) []ContactSource {
	seen := map[string]bool{}
	out := []ContactSource{}
	for _, row := range v.Items {
		id, label := pluginrpc.DshbotContactSource(row)
		if id != "" && !seen[id] {
			seen[id] = true
			out = append(out, ContactSource{id, label})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Label == out[j].Label {
			return out[i].ID < out[j].ID
		}
		return out[i].Label < out[j].Label
	})
	return out
}
func ContactDisplayName(v pluginrpc.DshbotBotItem) string {
	if v.Kind != "room" && strings.TrimSpace(v.Title) != "" {
		return strings.TrimSpace(v.Title)
	}
	if strings.TrimSpace(v.Name) != "" {
		return strings.TrimSpace(v.Name)
	}
	return v.ID
}
func ContactGroups(v pluginrpc.DshbotContactCatalog, f ContactFilters, activity map[string]ContactActivity) []ContactGroup {
	now := time.Now().UnixMilli()
	groups := make([]ContactGroup, 0, len(v.Sections)+1)
	byID := map[string]int{}
	for _, section := range v.Sections {
		byID[section.ID] = len(groups)
		groups = append(groups, ContactGroup{ID: section.ID, Name: section.Name})
	}
	groups = append(groups, ContactGroup{Name: storelocale.T(storelocale.KeyStoreDshbotUnassigned)})
	needle := strings.ToLower(strings.TrimSpace(f.Query))
	for _, row := range v.Items {
		if row.Hidden && !f.ShowHidden {
			continue
		}
		if f.Kind == "bots" && row.Kind == "room" || f.Kind == "rooms" && row.Kind != "room" {
			continue
		}
		source, _ := pluginrpc.DshbotContactSource(row)
		if f.Source != "" && source != f.Source {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(strings.Join([]string{row.Name, row.Title, pluginrpc.DshbotContactText(row, "description")}, "\n")), needle) {
			continue
		}
		if !ContactActivityMatches(row, activity[row.ID], f.Activity, now) {
			continue
		}
		index, ok := byID[pluginrpc.DshbotContactText(row, "sectionId")]
		if !ok {
			index = len(groups) - 1
		}
		groups[index].Items = append(groups[index].Items, row)
	}
	for i := range groups {
		rows := groups[i].Items
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Pinned != rows[j].Pinned {
				return rows[i].Pinned
			}
			a, b := ContactActivityAt(rows[i], activity[rows[i].ID]), ContactActivityAt(rows[j], activity[rows[j].ID])
			if a != b {
				return a > b
			}
			aName, bName := strings.ToLower(ContactDisplayName(rows[i])), strings.ToLower(ContactDisplayName(rows[j]))
			if aName != bName {
				return aName < bName
			}
			return strings.ToLower(rows[i].Name) < strings.ToLower(rows[j].Name)
		})
	}
	return groups
}

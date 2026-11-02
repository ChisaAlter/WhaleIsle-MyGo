package shortcutprefs

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// Snapshot is an accepted document or a diagnostic retaining the last good one.
// Sequence is monotonic within an owner lifetime; clients discard older replies.
type Snapshot struct {
	Revision      string   `json:"revision"`
	Sequence      uint64   `json:"sequence"`
	Document      Document `json:"document"`
	Status        string   `json:"status"`
	Error         *string  `json:"error"`
	UsingDefaults bool     `json:"usingDefaults"`
}

// Result classifies a transaction without replacing accepted state on failure.
type Result struct {
	Status    string   `json:"status"`
	Snapshot  Snapshot `json:"snapshot"`
	Issue     string   `json:"issue,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
}

// Store serializes all reads, catalog changes, and writes for one owner process.
// The injected writer must atomically commit or leave the previous bytes intact.
// It must additionally enforce any underlying config.Store baseline protection.
// Creating a second independent Store for the same persistence target is unsafe.
type Store struct {
	mu            sync.Mutex
	profile       string
	read          func() ([]byte, error)
	write         func([]byte) error
	snapshot      Snapshot
	raw           []byte
	observed      bool
	readError     string
	catalogErrors map[string]bool
	definitions   map[string][]Definition
	subscribers   map[uint64]func(Snapshot)
	subscriberID  uint64
	epoch         string
}

// New reads the current configuration; nil bytes and nil error mean no saved
// document. Read/parse failures are returned through the diagnostic snapshot.
func New(profile string, read func() ([]byte, error), write func([]byte) error) *Store {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		panic("shortcutprefs: cannot create revision identity")
	}
	doc, _ := ParseDocument(nil)
	s := &Store{profile: profile, read: read, write: write, definitions: map[string][]Definition{}, catalogErrors: map[string]bool{}, subscribers: map[uint64]func(Snapshot){}, epoch: hex.EncodeToString(token), snapshot: Snapshot{Document: doc, Status: "loading", UsingDefaults: true}}
	s.readLocked()
	return s
}

// Get rereads the injected source and returns a detached accepted snapshot.
func (s *Store) Get() Snapshot {
	s.mu.Lock()
	before := s.snapshot.Sequence
	s.readLocked()
	out := cloneSnapshot(s.snapshot)
	listeners := s.listenersLocked(before)
	s.mu.Unlock()
	publish(listeners, out)
	return out
}

// Accepted returns the accepted snapshot without rereading the persisted
// source. The revision changes on every accepted document or catalog change, so
// a render path can compare Sequence instead of paying a file read per frame;
// Get and ReloadSource remain the ways to notice an external document.
func (s *Store) Accepted() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneSnapshot(s.snapshot)
}

// ReloadSource replaces a failed initialization reader only after its owner
// explicitly reloads persistence. Registered catalogs and write ownership stay
// on this store, so existing native and plugin consumers keep one authority.
func (s *Store) ReloadSource(read func() ([]byte, error)) Snapshot {
	s.mu.Lock()
	before := s.snapshot.Sequence
	s.read = read
	s.readLocked()
	out := cloneSnapshot(s.snapshot)
	listeners := s.listenersLocked(before)
	s.mu.Unlock()
	publish(listeners, out)
	return out
}

// SetDefinitions installs or revokes one trusted client's catalog. Invalid
// catalogs disable writes with an invalid diagnostic; saved preferences remain.
// Empty definitions revoke only that owner's catalog, never its overrides.
func (s *Store) SetDefinitions(owner string, defs []Definition) Snapshot {
	s.mu.Lock()
	before := s.snapshot.Sequence
	if owner == "" {
		s.catalogErrors[owner] = true
		delete(s.definitions, owner)
	} else if len(defs) == 0 {
		delete(s.definitions, owner)
		delete(s.catalogErrors, owner)
	} else if validateDefinitions(defs) != nil {
		s.catalogErrors[owner] = true
		delete(s.definitions, owner)
	} else {
		s.definitions[owner] = cloneDefinitions(defs)
		delete(s.catalogErrors, owner)
	}
	s.acceptLocked()
	out := cloneSnapshot(s.snapshot)
	listeners := s.listenersLocked(before)
	s.mu.Unlock()
	publish(listeners, out)
	return out
}

// Subscribe receives accepted changes after the owner mutex is released. Each
// callback receives an independent copy. Unsubscribe does not cancel an event
// already captured for delivery. Consumers must compare Sequence for ordering.
func (s *Store) Subscribe(listener func(Snapshot)) func() {
	if listener == nil {
		return func() {}
	}
	s.mu.Lock()
	s.subscriberID++
	id := s.subscriberID
	s.subscribers[id] = listener
	s.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { s.mu.Lock(); delete(s.subscribers, id); s.mu.Unlock() }) }
}

// Effective returns one owner's rows with conflicts computed over every current
// catalog. It does not register, intercept, or execute any keyboard combination.
func (s *Store) Effective(owner string) []EffectiveShortcut {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := map[string]bool{}
	for _, d := range s.definitions[owner] {
		if d.Fixed == nil {
			ids[d.ID] = true
		}
	}
	rows := Effective(s.catalogLocked(), s.snapshot.Document, s.profile)
	out := []EffectiveShortcut{}
	for _, row := range rows {
		if ids[row.ID] {
			out = append(out, row)
		}
	}
	return out
}

// Describe previews a candidate against every current owner, including fixed
// reservations, without publishing a revision or writing preferences.
func (s *Store) Describe(owner, id string, binding *Binding) (string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	owned := false
	for _, definition := range s.definitions[owner] {
		if definition.ID == id && definition.Fixed == nil {
			owned = true
		}
	}
	if !owned {
		return "not-ready", nil
	}
	doc := cloneDocument(s.snapshot.Document)
	if doc.Profiles[s.profile] == nil {
		doc.Profiles[s.profile] = map[string]*Binding{}
	}
	doc.Profiles[s.profile][id] = cloneBinding(binding)
	for _, row := range Effective(s.catalogLocked(), doc, s.profile) {
		if row.ID == id {
			issue := ""
			if row.Issue != nil {
				issue = *row.Issue
			}
			return issue, append([]string{}, row.Conflicts...)
		}
	}
	return "not-ready", nil
}

// Apply commits an atomic batch against the reviewed revision. Callers cannot
// write unknown IDs, another owner's commands, or that owner's fixed bindings.
// Reset-all removes only current editable catalog entries, preserving dormant IDs.
func (s *Store) Apply(owner, revision string, edits []Edit) Result {
	s.mu.Lock()
	before := s.snapshot.Sequence
	s.readLocked()
	result := s.applyLocked(owner, revision, edits)
	out := cloneSnapshot(s.snapshot)
	result.Snapshot = out
	listeners := s.listenersLocked(before)
	s.mu.Unlock()
	publish(listeners, out)
	return result
}
func (s *Store) applyLocked(owner, revision string, edits []Edit) Result {
	result := func(status string) Result { return Result{Status: status} }
	defs, ready := s.definitions[owner]
	if !ready || s.snapshot.Status == "loading" {
		return result("not-ready")
	}
	if revision != s.snapshot.Revision {
		return result("stale")
	}
	if s.snapshot.Status == "unreadable" {
		return result("unreadable")
	}
	owned := map[string]bool{}
	for _, d := range defs {
		if d.Fixed == nil {
			owned[d.ID] = true
		}
	}
	for _, edit := range edits {
		if validateEdit(edit) != nil {
			return Result{Status: "conflict", Issue: "unsupported-key"}
		}
		if edit.Type != "reset-all" && !owned[edit.ID] {
			return result("not-ready")
		}
	}
	if len(edits) == 0 {
		return result("saved")
	}
	candidate := cloneDocument(s.snapshot.Document)
	candidate.SchemaVersion = 2
	overrides := candidate.Profiles[s.profile]
	if overrides == nil {
		overrides = map[string]*Binding{}
		candidate.Profiles[s.profile] = overrides
	}
	touched := map[string]bool{}
	set := map[string]bool{}
	for _, edit := range edits {
		switch edit.Type {
		case "set":
			overrides[edit.ID] = cloneBinding(edit.Binding)
			touched[edit.ID] = true
			set[edit.ID] = true
		case "reset":
			delete(overrides, edit.ID)
			touched[edit.ID] = true
			delete(set, edit.ID)
		case "reset-all":
			for id := range owned {
				delete(overrides, id)
				touched[id] = true
				delete(set, id)
			}
		}
	}
	rows := Effective(s.catalogLocked(), candidate, s.profile)
	for _, row := range rows {
		if touched[row.ID] && (row.Issue != nil || len(row.Conflicts) > 0) {
			r := Result{Status: "conflict", Conflicts: append([]string{}, row.Conflicts...)}
			if row.Issue != nil {
				r.Issue = *row.Issue
			}
			return r
		}
	}
	// An explicit edit cannot silently displace another command's default.
	for _, row := range rows {
		for _, id := range row.Conflicts {
			if set[id] {
				return Result{Status: "conflict", Conflicts: []string{row.ID}}
			}
		}
	}
	if s.write == nil {
		return result("write-failed")
	}
	raw, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return result("write-failed")
	}
	raw = append(raw, '\n')
	if err = s.write(append([]byte(nil), raw...)); err != nil {
		return result("write-failed")
	}
	s.raw = raw
	s.observed = true
	s.readError = ""
	s.snapshot.Document = candidate
	s.snapshot.UsingDefaults = false
	s.acceptLocked()
	return result("saved")
}
func (s *Store) readLocked() {
	if s.read == nil {
		if s.readError != "read" || s.snapshot.Status == "loading" {
			s.readError = "read"
			s.acceptLocked()
		}
		return
	}
	raw, err := s.read()
	if err != nil {
		if s.readError != "read" || s.snapshot.Status == "loading" {
			s.readError = "read"
			s.acceptLocked()
		}
		return
	}
	if s.observed && (raw == nil) == (s.raw == nil) && bytes.Equal(raw, s.raw) && s.readError != "read" {
		return
	}
	s.raw = append([]byte(nil), raw...)
	if raw != nil && s.raw == nil {
		s.raw = []byte{}
	}
	s.observed = true
	doc, issue := ParseDocument(raw)
	s.readError = issue
	if issue == "" {
		s.snapshot.Document = doc
		s.snapshot.UsingDefaults = raw == nil
	}
	s.acceptLocked()
}
func (s *Store) acceptLocked() {
	issue := s.readError
	if issue == "" && (!profilePattern.MatchString(s.profile) || len(s.catalogErrors) > 0 || !s.sharedDefinitionsAgreeLocked()) {
		issue = "invalid"
	}
	s.snapshot.Status = "ready"
	s.snapshot.Error = nil
	if issue != "" {
		s.snapshot.Status = "unreadable"
		s.snapshot.Error = &issue
	}
	s.snapshot.Sequence++
	s.snapshot.Revision = fmt.Sprintf("%s:%d", s.epoch, s.snapshot.Sequence)
}
func (s *Store) sharedDefinitionsAgreeLocked() bool {
	seen := map[string]*Binding{}
	for _, defs := range s.definitions {
		for _, d := range defs {
			if d.Fixed != nil {
				continue
			}
			var current *Binding
			if b, ok := d.Defaults[s.profile]; ok {
				normalized := normalize(b, s.profile)
				current = &normalized
			}
			if old, ok := seen[d.ID]; ok {
				if (old == nil) != (current == nil) {
					return false
				}
				if old != nil && !sameBinding(*old, *current) {
					return false
				}
			} else {
				seen[d.ID] = current
			}
		}
	}
	return true
}
func sameBinding(a, b Binding) bool {
	return a.Code == b.Code && a.SecondCode == b.SecondCode && bytes.Equal([]byte(fmt.Sprint(a.Modifiers)), []byte(fmt.Sprint(b.Modifiers)))
}
func (s *Store) catalogLocked() []Definition {
	owners := make([]string, 0, len(s.definitions))
	for owner := range s.definitions {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	out := []Definition{}
	for _, owner := range owners {
		out = append(out, s.definitions[owner]...)
	}
	return out
}
func (s *Store) listenersLocked(before uint64) []func(Snapshot) {
	if before == s.snapshot.Sequence {
		return nil
	}
	out := make([]func(Snapshot), 0, len(s.subscribers))
	for _, fn := range s.subscribers {
		out = append(out, fn)
	}
	return out
}
func cloneSnapshot(s Snapshot) Snapshot {
	out := s
	out.Document = cloneDocument(s.Document)
	if s.Error != nil {
		issue := *s.Error
		out.Error = &issue
	}
	return out
}
func publish(listeners []func(Snapshot), snapshot Snapshot) {
	for _, listener := range listeners {
		func() { defer func() { _ = recover() }(); listener(cloneSnapshot(snapshot)) }()
	}
}

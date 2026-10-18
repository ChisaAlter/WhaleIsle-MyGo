package shortcutprefs

import (
	"encoding/json"
	"sort"
)

// ImportEntry compares one explicit source override with the target override.
// HasExisting distinguishes an absent override from an explicit nil/unbinding.
// Conflict requires a user choice; an omitted choice always keeps Existing.
type ImportEntry struct {
	ID          string   `json:"id"`
	Incoming    *Binding `json:"incoming"`
	Existing    *Binding `json:"existing"`
	HasExisting bool     `json:"hasExisting"`
	Conflict    bool     `json:"conflict"`
}

// ImportPreview is a read-only plan for importing one profile's overrides.
// Status is ready, invalid-import, or unreadable. Error classifies source errors
// as invalid, future, or profile; target diagnostics remain in Snapshot.Error.
// Entries and Conflicts are sorted by ID. No source defaults are synthesized.
type ImportPreview struct {
	Status        string        `json:"status"`
	Snapshot      Snapshot      `json:"snapshot"`
	SourceProfile string        `json:"sourceProfile"`
	TargetProfile string        `json:"targetProfile"`
	Entries       []ImportEntry `json:"entries"`
	Conflicts     []string      `json:"conflicts"`
	Error         string        `json:"error,omitempty"`
}

// PreviewImport compares a strictly parsed source document with current saved
// overrides without writing either one. SourceProfile is a schema profile name,
// never a filesystem path. The target profile always belongs to this Store.
// Callers retain the source and use Snapshot.Revision when submitting choices.
func (s *Store) PreviewImport(raw []byte, sourceProfile string) ImportPreview {
	s.mu.Lock()
	before := s.snapshot.Sequence
	s.readLocked()
	plan := s.previewImportLocked(raw, sourceProfile)
	plan.Snapshot = cloneSnapshot(s.snapshot)
	listeners := s.listenersLocked(before)
	s.mu.Unlock()
	publish(listeners, plan.Snapshot)
	return plan
}

func (s *Store) previewImportLocked(raw []byte, sourceProfile string) ImportPreview {
	plan := ImportPreview{Status: "ready", SourceProfile: sourceProfile, TargetProfile: s.profile, Entries: []ImportEntry{}, Conflicts: []string{}}
	if s.snapshot.Status != "ready" {
		plan.Status = "unreadable"
		return plan
	}
	if !profilePattern.MatchString(sourceProfile) {
		plan.Status = "invalid-import"
		plan.Error = "profile"
		return plan
	}
	document, issue := ParseDocument(raw)
	if issue != "" {
		plan.Status = "invalid-import"
		plan.Error = issue
		return plan
	}
	incoming := document.Profiles[sourceProfile]
	existing := s.snapshot.Document.Profiles[s.profile]
	ids := make([]string, 0, len(incoming))
	for id := range incoming {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		current, present := existing[id]
		entry := ImportEntry{ID: id, Incoming: cloneBinding(incoming[id]), Existing: cloneBinding(current), HasExisting: present, Conflict: present && !equivalentImportBinding(current, incoming[id], s.profile)}
		plan.Entries = append(plan.Entries, entry)
		if entry.Conflict {
			plan.Conflicts = append(plan.Conflicts, id)
		}
	}
	return plan
}

func equivalentImportBinding(left, right *Binding, profile string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return sameBinding(normalize(*left, profile), normalize(*right, profile))
}

// Import explicitly imports sourceProfile overrides into this Store's profile.
// Missing targets are inserted; equivalent values and unselected conflicts keep
// the existing representation. ReplaceIDs must name distinct current conflicts.
// Unknown/dormant valid IDs are preserved as data, never registered as commands.
// Invalid input returns invalid-import with issue invalid/future/profile/selection.
// A stale revision, failed write, or effective-binding conflict changes no data.
// Neither the supplied bytes nor the source's localStorage entry is modified.
func (s *Store) Import(raw []byte, sourceProfile, revision string, replaceIDs []string) Result {
	s.mu.Lock()
	before := s.snapshot.Sequence
	s.readLocked()
	result := s.importLocked(raw, sourceProfile, revision, replaceIDs)
	result.Snapshot = cloneSnapshot(s.snapshot)
	listeners := s.listenersLocked(before)
	s.mu.Unlock()
	publish(listeners, result.Snapshot)
	return result
}

func (s *Store) importLocked(raw []byte, sourceProfile, revision string, replaceIDs []string) Result {
	if s.snapshot.Status == "loading" {
		return Result{Status: "not-ready"}
	}
	if revision != s.snapshot.Revision {
		return Result{Status: "stale"}
	}
	if s.snapshot.Status == "unreadable" {
		return Result{Status: "unreadable"}
	}
	plan := s.previewImportLocked(raw, sourceProfile)
	if plan.Status != "ready" {
		return Result{Status: plan.Status, Issue: plan.Error}
	}
	conflicts := make(map[string]bool, len(plan.Conflicts))
	for _, id := range plan.Conflicts {
		conflicts[id] = true
	}
	replace := make(map[string]bool, len(replaceIDs))
	for _, id := range replaceIDs {
		if !conflicts[id] || replace[id] {
			return Result{Status: "invalid-import", Issue: "selection"}
		}
		replace[id] = true
	}
	candidate := cloneDocument(s.snapshot.Document)
	overrides := candidate.Profiles[s.profile]
	if overrides == nil {
		overrides = map[string]*Binding{}
		candidate.Profiles[s.profile] = overrides
	}
	touched := map[string]bool{}
	for _, entry := range plan.Entries {
		if entry.HasExisting && (!entry.Conflict || !replace[entry.ID]) {
			continue
		}
		overrides[entry.ID] = cloneBinding(entry.Incoming)
		touched[entry.ID] = true
	}
	// Keeping every conflict or importing an empty profile is a genuine no-op.
	if len(touched) == 0 {
		return Result{Status: "saved"}
	}
	candidate.SchemaVersion = 2
	rows := Effective(s.catalogLocked(), candidate, s.profile)
	conflictIDs := map[string]bool{}
	issue := ""
	for _, row := range rows {
		if touched[row.ID] && (row.Issue != nil || len(row.Conflicts) > 0) {
			conflictIDs[row.ID] = true
			for _, id := range row.Conflicts {
				conflictIDs[id] = true
			}
			if issue == "" && row.Issue != nil {
				issue = *row.Issue
			}
		}
		// Preserve another command's default even when an explicit incoming override
		// would otherwise win the normal effective-binding precedence calculation.
		for _, id := range row.Conflicts {
			if touched[id] {
				conflictIDs[row.ID] = true
				conflictIDs[id] = true
			}
		}
	}
	if len(conflictIDs) > 0 {
		ids := make([]string, 0, len(conflictIDs))
		for id := range conflictIDs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return Result{Status: "conflict", Issue: issue, Conflicts: ids}
	}
	if s.write == nil {
		return Result{Status: "write-failed"}
	}
	encoded, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return Result{Status: "write-failed"}
	}
	encoded = append(encoded, '\n')
	if err = s.write(append([]byte(nil), encoded...)); err != nil {
		return Result{Status: "write-failed"}
	}
	s.raw = encoded
	s.observed = true
	s.readError = ""
	s.snapshot.Document = candidate
	s.snapshot.UsingDefaults = false
	s.acceptLocked()
	return Result{Status: "saved"}
}

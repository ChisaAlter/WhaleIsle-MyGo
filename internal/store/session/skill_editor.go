package session

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// SkillEditTarget pins a draft to the catalog Host, selected workspace, file and
// read revision. Its data is inert and can only be written by explicit Save.
type SkillEditTarget struct {
	client                                                     *dsh.Client
	sessionID, cwd, scopeSession, scopeCwd, name, source, path string
	version, hostGeneration                                    uint64
	original                                                   api.SkillInventoryDetail
	fingerprint                                                [32]byte
	loaded, writable                                           bool
}

type SkillEditorFields struct {
	Description, WhenToUse, Content string
	Groups                          []string
	ModelInvocable, UserInvocable   bool
}

func (t SkillEditTarget) Fields() SkillEditorFields {
	f := SkillEditorFields{Description: t.original.Description, Content: t.original.Content, Groups: append([]string(nil), t.original.Groups...), ModelInvocable: t.original.ModelInvocable, UserInvocable: t.original.UserInvocable}
	if t.original.WhenToUse != nil {
		f.WhenToUse = *t.original.WhenToUse
	}
	return f
}

func (t SkillEditTarget) Writable() bool { return t.loaded && t.writable }

func (t SkillEditTarget) Label() string { return t.name + " · " + t.source + "\n" + t.path }

func (s *Store) CaptureCatalogSkillEdit(name string) (SkillEditTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.skillCatalogClient != s.client || s.skillCatalogLoading || s.skillCatalog.Busy || s.skillCatalogSession != s.selectedID || s.skillCatalogCwd != s.cwd {
		return SkillEditTarget{}, fmt.Errorf("Skill catalog changed; refresh before editing")
	}
	var row *api.SkillInventorySnapshotSkillsItem
	for i := range s.skillCatalog.Skills {
		if s.skillCatalog.Skills[i].Name == name {
			if row != nil {
				return SkillEditTarget{}, fmt.Errorf("Skill identity is ambiguous")
			}
			row = &s.skillCatalog.Skills[i]
		}
	}
	if row == nil || (row.Writable && (row.Path == nil || *row.Path == "" || row.Source == "")) {
		return SkillEditTarget{}, fmt.Errorf("Skill has no confirmed source identity")
	}
	target := SkillEditTarget{client: s.client, sessionID: s.selectedID, cwd: s.cwd, version: s.skillCatalogVersion, hostGeneration: s.eventsGeneration, name: row.Name, source: row.Source, writable: row.Writable}
	if row.Path != nil {
		target.path = *row.Path
	}
	if s.skillCatalogScope.SessionId != nil {
		target.scopeSession = *s.skillCatalogScope.SessionId
	}
	if s.skillCatalogScope.Cwd != nil {
		target.scopeCwd = *s.skillCatalogScope.Cwd
	}
	return target, nil
}

func (s *Store) skillEditCurrentLocked(t SkillEditTarget, version uint64) bool {
	return s.eventsGeneration == t.hostGeneration && s.skillCatalogCurrentLocked(t.client, t.sessionID, t.cwd, version) && !s.skillCatalogLoading
}

func (s *Store) SkillEditorCurrent(t SkillEditTarget) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return t.Writable() && !s.skillCatalog.Busy && s.skillEditCurrentLocked(t, t.version)
}

func (t SkillEditTarget) request() map[string]any {
	req := map[string]any{"name": t.name}
	if t.scopeSession != "" {
		req["sessionId"] = t.scopeSession
	}
	if t.scopeCwd != "" {
		req["cwd"] = t.scopeCwd
	}
	return req
}

func skillEditFingerprint(doc api.SkillInventoryDetail) [32]byte {
	data, _ := json.Marshal(doc)
	return sha256.Sum256(data)
}
func (t SkillEditTarget) matchesFile(doc api.SkillInventoryDetail) bool {
	path := ""
	if doc.Path != nil {
		path = *doc.Path
	}
	return doc.Writable == t.writable && doc.Name == t.name && doc.Source == t.source && path == t.path
}

func (s *Store) LoadCatalogSkillEdit(ctx context.Context, t SkillEditTarget) (SkillEditTarget, error) {
	s.mu.Lock()
	current := !s.skillCatalog.Busy && s.skillEditCurrentLocked(t, t.version)
	s.mu.Unlock()
	if !current {
		return SkillEditTarget{}, fmt.Errorf("Skill edit target changed; inspect again")
	}
	var doc api.SkillInventoryDetail
	if err := t.client.Call(ctx, "skillInventory/get", map[string]any{"request": t.request()}, &doc); err != nil {
		return SkillEditTarget{}, err
	}
	s.mu.Lock()
	current = !s.skillCatalog.Busy && s.skillEditCurrentLocked(t, t.version)
	s.mu.Unlock()
	if !current || !t.matchesFile(doc) {
		return SkillEditTarget{}, fmt.Errorf("Skill source or target changed; inspect again")
	}
	t.original = doc
	t.fingerprint = skillEditFingerprint(doc)
	t.loaded = true
	return t, nil
}

// SaveCatalogSkillEditor consumes one read revision, even on an unknown result.
// A read-before-write guard detects observed edits but the Host has no atomic CAS.
// Failure leaves the caller's draft intact; re-inspect before any further Save.
func (s *Store) SaveCatalogSkillEditor(ctx context.Context, t SkillEditTarget, f SkillEditorFields) error {
	if strings.TrimSpace(f.Description) == "" || strings.TrimSpace(f.Content) == "" {
		return fmt.Errorf("Description and content are required")
	}
	s.mu.Lock()
	if !t.Writable() || s.skillCatalog.Busy || !s.skillEditCurrentLocked(t, t.version) {
		s.mu.Unlock()
		return fmt.Errorf("Skill edit target or revision changed; inspect again before saving")
	}
	s.skillCatalogVersion++
	version := s.skillCatalogVersion
	s.skillCatalog.Busy = true
	s.skillCatalog.Action = ""
	s.invalidateViewLocked()
	s.mu.Unlock()
	s.fire()
	finish := func(err error) error {
		s.mu.Lock()
		current := s.skillEditCurrentLocked(t, version)
		if current {
			s.invalidateViewLocked()
			s.skillCatalog.Busy = false
			if err != nil {
				s.skillCatalog.Action = err.Error()
			} else {
				s.skillCatalog.Action = "Skill saved: " + t.name
				for i := range s.skillCatalog.Skills {
					row := &s.skillCatalog.Skills[i]
					if row.Name == t.name && row.Source == t.source && row.Path != nil && *row.Path == t.path {
						row.Description = strings.TrimSpace(f.Description)
						row.Groups = NormalizeSkillGroups(f.Groups)
						row.ModelInvocable = f.ModelInvocable
						row.UserInvocable = f.UserInvocable
						row.WhenToUse = nil
						if when := strings.TrimSpace(f.WhenToUse); when != "" {
							row.WhenToUse = &when
						}
					}
				}
			}
		} else {
			s.retireSkillCatalogMutationLocked(version)
		}
		s.mu.Unlock()
		s.fire()
		if !current && err == nil {
			return fmt.Errorf("Skill saved on its original Host; the current target changed. Inspect before continuing")
		}
		return err
	}
	// fire may re-enter selection or reconnect before this first RPC starts.
	s.mu.Lock()
	valid := ctx.Err() == nil && s.skillEditCurrentLocked(t, version)
	s.mu.Unlock()
	if !valid {
		return finish(fmt.Errorf("Skill target changed before save; draft retained"))
	}
	var current api.SkillInventoryDetail
	if err := t.client.Call(ctx, "skillInventory/get", map[string]any{"request": t.request()}, &current); err != nil {
		return finish(fmt.Errorf("Skill pre-save check failed; draft retained. Inspect again: %w", err))
	}
	if !t.matchesFile(current) || skillEditFingerprint(current) != t.fingerprint {
		return finish(fmt.Errorf("Skill source changed since it was read; draft retained. Inspect again before saving"))
	}
	s.mu.Lock()
	valid = ctx.Err() == nil && s.skillEditCurrentLocked(t, version)
	s.mu.Unlock()
	if !valid {
		return finish(fmt.Errorf("Skill target changed before save; draft retained"))
	}
	req := t.request()
	req["description"] = strings.TrimSpace(f.Description)
	req["content"] = f.Content
	req["groups"] = NormalizeSkillGroups(f.Groups)
	req["modelInvocable"] = f.ModelInvocable
	req["userInvocable"] = f.UserInvocable
	// The Host removes whenToUse when it is omitted or empty. Groups must be [].
	if when := strings.TrimSpace(f.WhenToUse); when != "" {
		req["whenToUse"] = when
	}
	var out api.UpdateResult
	if err := t.client.Call(ctx, "skillInventory/update", map[string]any{"request": req}, &out); err != nil {
		return finish(fmt.Errorf("Skill save was not confirmed; draft retained. Inspect before retrying: %w", err))
	}
	return finish(nil)
}

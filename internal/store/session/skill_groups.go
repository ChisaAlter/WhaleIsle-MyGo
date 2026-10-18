package session

import (
	"context"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
)

// SkillCatalog retains inventory permissions and groups independently of the
// composer skills/list result, which intentionally has a smaller wire shape.
type SkillCatalog struct {
	Skills              []api.SkillInventorySnapshotSkillsItem
	Err, Notice, Action string
	Errors              map[string]string
	Busy                bool
}

func (c SkillCatalog) clone() SkillCatalog {
	c.Skills = append([]api.SkillInventorySnapshotSkillsItem(nil), c.Skills...)
	for i := range c.Skills {
		c.Skills[i].Groups = append([]string(nil), c.Skills[i].Groups...)
	}
	errors := make(map[string]string, len(c.Errors))
	for k, v := range c.Errors {
		errors[k] = v
	}
	c.Errors = errors
	return c
}

func NormalizeSkillGroups(groups []string) []string {
	out := make([]string, 0, len(groups))
	seen := map[string]bool{}
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group != "" && !seen[group] {
			out = append(out, group)
			seen[group] = true
		}
	}
	return out
}

func (s *Store) RefreshSkillCatalog(ctx context.Context) {
	s.mu.Lock()
	client, sid, cwd := s.client, s.selectedID, s.cwd
	if client == nil || (s.skillCatalog.Busy && s.skillCatalogClient == client && s.skillCatalogSession == sid && s.skillCatalogCwd == cwd) {
		s.mu.Unlock()
		return
	}
	if s.skillCatalogClient != client || s.skillCatalogSession != sid || s.skillCatalogCwd != cwd {
		s.skillCatalog = SkillCatalog{}
	}
	s.skillCatalogClient, s.skillCatalogLoading = client, true
	s.skillCatalogSession, s.skillCatalogCwd = sid, cwd
	s.skillCatalogVersion++
	version := s.skillCatalogVersion
	s.mu.Unlock()
	go func() {
		scope := api.SkillInventoryScope{}
		if sid != "" {
			scope.SessionId = &sid
		}
		if cwd != "" {
			scope.Cwd = &cwd
		}
		var inv api.SkillInventorySnapshot
		err := client.Call(ctx, "skillInventory/list", map[string]any{"request": scope}, &inv)
		notice := ""
		if err != nil && sid != "" {
			scope.SessionId = nil
			err = client.Call(ctx, "skillInventory/list", map[string]any{"request": scope}, &inv)
			if err == nil {
				notice = "Session skill catalog unavailable; showing global skills."
			}
		}
		s.mu.Lock()
		if s.client != client || s.skillCatalogVersion != version || s.selectedID != sid || s.cwd != cwd {
			s.mu.Unlock()
			return
		}
		s.skillCatalogLoading = false
		if err != nil {
			s.skillCatalog.Err = err.Error()
			s.skillCatalog.Skills = nil
		} else {
			s.skillCatalog.Skills = inv.Skills
			s.skillCatalog.Err = ""
			s.skillCatalog.Notice = notice
			s.skillCatalogScope = scope
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) GetCatalogSkillDetail(ctx context.Context, name string) (api.SkillInventoryDetail, error) {
	s.mu.Lock()
	client, scope := s.client, s.skillCatalogScope
	s.mu.Unlock()
	if client == nil {
		return api.SkillInventoryDetail{}, fmt.Errorf("harness not ready")
	}
	req := map[string]any{"name": name}
	if scope.SessionId != nil {
		req["sessionId"] = *scope.SessionId
	}
	if scope.Cwd != nil {
		req["cwd"] = *scope.Cwd
	}
	var doc api.SkillInventoryDetail
	err := client.Call(ctx, "skillInventory/get", map[string]any{"request": req}, &doc)
	return doc, err
}

func (s *Store) SetSkillGroupInvocation(ctx context.Context, group string, enabled bool) {
	s.setCatalogInvocation(ctx, func(sk api.SkillInventorySnapshotSkillsItem) bool {
		groups := NormalizeSkillGroups(sk.Groups)
		if group == "" {
			return len(groups) == 0
		}
		for _, g := range groups {
			if g == group {
				return true
			}
		}
		return false
	}, enabled)
}

func (s *Store) SetCatalogSkillInvocation(ctx context.Context, name string, enabled bool) {
	s.setCatalogInvocation(ctx, func(sk api.SkillInventorySnapshotSkillsItem) bool { return sk.Name == name }, enabled)
}

func (s *Store) setCatalogInvocation(ctx context.Context, matches func(api.SkillInventorySnapshotSkillsItem) bool, enabled bool) {
	s.mu.Lock()
	client, scope := s.client, s.skillCatalogScope
	if client == nil || s.skillCatalogClient != client || s.skillCatalogLoading || s.skillCatalog.Busy || s.skillCatalogSession != s.selectedID || s.skillCatalogCwd != s.cwd {
		s.mu.Unlock()
		return
	}
	var targets []api.SkillInventorySnapshotSkillsItem
	for i, sk := range s.skillCatalog.Skills {
		if sk.Writable && matches(sk) && sk.ModelInvocable != enabled {
			targets = append(targets, sk)
			s.skillCatalog.Skills[i].ModelInvocable = enabled
		}
	}
	if len(targets) == 0 {
		s.mu.Unlock()
		return
	}
	s.skillCatalogVersion++
	version := s.skillCatalogVersion
	s.skillCatalog.Busy = true
	s.skillCatalog.Errors = map[string]string{}
	s.skillCatalog.Action = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		failures := map[string]string{}
		for _, sk := range targets {
			req := api.SkillInventoryInvocationRequest{Name: sk.Name, ModelInvocable: enabled, UserInvocable: sk.UserInvocable, SessionId: scope.SessionId, Cwd: scope.Cwd}
			var out api.SetInvocationResult
			if err := client.Call(ctx, "skillInventory/setInvocation", map[string]any{"request": req}, &out); err != nil {
				failures[sk.Name] = err.Error()
			}
		}
		s.mu.Lock()
		if s.skillCatalogVersion == version {
			for _, sk := range targets {
				if failures[sk.Name] != "" {
					for i := range s.skillCatalog.Skills {
						if s.skillCatalog.Skills[i].Name == sk.Name {
							s.skillCatalog.Skills[i].ModelInvocable = sk.ModelInvocable
						}
					}
				}
			}
			s.skillCatalog.Errors = failures
			s.skillCatalog.Busy = false
			s.skillCatalog.Action = fmt.Sprintf("Skill invocation: %d updated, %d failed", len(targets)-len(failures), len(failures))
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SaveCatalogSkill sends groups explicitly, including [] to clear all tags.
// The generated API type omits an empty slice, so use an explicit map here.
func (s *Store) SaveCatalogSkill(ctx context.Context, doc api.SkillInventoryDetail) {
	s.mu.Lock()
	client, scope := s.client, s.skillCatalogScope
	if client == nil || s.skillCatalogClient != client || s.skillCatalogLoading || s.skillCatalog.Busy || !doc.Writable || doc.Name == "" || s.skillCatalogSession != s.selectedID || s.skillCatalogCwd != s.cwd {
		s.mu.Unlock()
		return
	}
	found := false
	for _, sk := range s.skillCatalog.Skills {
		if sk.Name == doc.Name && sk.Writable {
			found = true
			doc.ModelInvocable, doc.UserInvocable = sk.ModelInvocable, sk.UserInvocable
			break
		}
	}
	if !found {
		s.mu.Unlock()
		return
	}
	s.skillCatalogVersion++
	version := s.skillCatalogVersion
	s.skillCatalog.Busy = true
	s.skillCatalog.Action = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		req := map[string]any{"name": doc.Name, "description": doc.Description, "content": doc.Content, "groups": NormalizeSkillGroups(doc.Groups), "modelInvocable": doc.ModelInvocable, "userInvocable": doc.UserInvocable}
		if doc.WhenToUse != nil {
			req["whenToUse"] = *doc.WhenToUse
		}
		if scope.SessionId != nil {
			req["sessionId"] = *scope.SessionId
		}
		if scope.Cwd != nil {
			req["cwd"] = *scope.Cwd
		}
		var out api.UpdateResult
		err := client.Call(ctx, "skillInventory/update", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if s.skillCatalogVersion == version {
			s.skillCatalog.Busy = false
			if err != nil {
				s.skillCatalog.Action = "Skill save failed: " + err.Error()
			} else {
				s.skillCatalog.Action = "Skill saved: " + doc.Name
				for i := range s.skillCatalog.Skills {
					if s.skillCatalog.Skills[i].Name == doc.Name {
						s.skillCatalog.Skills[i].Groups = NormalizeSkillGroups(doc.Groups)
						s.skillCatalog.Skills[i].Description = doc.Description
					}
				}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

package session

import (
	"context"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
)

// SkillHubHit is one skillInventory/searchHub row.
type SkillHubHit struct {
	Repo        string
	Path        string
	SkillName   string
	Description string
	Stars       float64
	Installed   bool
}

// UpdateSkillInventory calls skillInventory/update with the given fields.
func (s *Store) UpdateSkillInventory(ctx context.Context, name, description, content string, modelInvocable, userInvocable bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil || name == "" {
		return
	}
	go func() {
		req := api.SkillInventoryUpdateRequest{
			Name:           name,
			Description:    description,
			Content:        content,
			ModelInvocable: modelInvocable,
			UserInvocable:  userInvocable,
		}
		if sessionID != "" {
			req.SessionId = &sessionID
		}
		if cwd != "" {
			req.Cwd = &cwd
		}
		var out api.UpdateResult
		err := client.Call(ctx, "skillInventory/update", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = "skill update: " + err.Error()
		} else {
			s.lastAction = "skill updated: " + name
		}
		s.mu.Unlock()
		s.fire()
		if err == nil {
			s.RefreshSkills(ctx)
		}
	}()
}

// GetSkillInventoryDetail calls skillInventory/get for one skill name.
func (s *Store) GetSkillInventoryDetail(ctx context.Context, name string) (api.SkillInventoryDetail, error) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil || name == "" {
		return api.SkillInventoryDetail{}, fmt.Errorf("harness not ready")
	}
	req := map[string]any{"name": name}
	if sessionID != "" {
		req["sessionId"] = sessionID
	}
	if cwd != "" {
		req["cwd"] = cwd
	}
	var out api.SkillInventoryDetail
	err := client.Call(ctx, "skillInventory/get", map[string]any{"request": req}, &out)
	return out, err
}

// SearchSkillHub calls skillInventory/searchHub and folds hits into Snapshot.
func (s *Store) SearchSkillHub(ctx context.Context, query, root string) {
	query = strings.TrimSpace(query)
	root = strings.TrimSpace(root)
	if root == "" {
		root = "user"
	}
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil || query == "" {
		return
	}
	go func() {
		req := api.SkillHubSearchRequest{Query: query, Root: root}
		if sessionID != "" {
			req.SessionId = &sessionID
		}
		if cwd != "" {
			req.Cwd = &cwd
		}
		var out api.SearchHubResult
		err := client.Call(ctx, "skillInventory/searchHub", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
			s.skillHubHits = nil
		} else {
			hits := make([]SkillHubHit, 0, len(out))
			for _, it := range out {
				installed := it.InstallationStatus != nil && strings.EqualFold(*it.InstallationStatus, "installed")
				hits = append(hits, SkillHubHit{
					Repo:        it.Repo,
					Path:        it.Path,
					SkillName:   it.SkillName,
					Description: it.Description,
					Stars:       it.Stars,
					Installed:   installed,
				})
			}
			s.skillHubHits = hits
			s.lastAction = "skill hub " + query
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// InstallSkillHub calls skillInventory/installHub then refreshes skills/list.
func (s *Store) InstallSkillHub(ctx context.Context, hit SkillHubHit, root string) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "user"
	}
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil || hit.Repo == "" || hit.SkillName == "" {
		return
	}
	go func() {
		req := api.SkillHubInstallRequest{
			Repo:      hit.Repo,
			Path:      hit.Path,
			SkillName: hit.SkillName,
			Root:      root,
		}
		if sessionID != "" {
			req.SessionId = &sessionID
		}
		if cwd != "" {
			req.Cwd = &cwd
		}
		var out api.SkillHubInstallResult
		err := client.Call(ctx, "skillInventory/installHub", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "skill installed " + hit.SkillName
		}
		s.mu.Unlock()
		s.fire()
		if err == nil {
			s.RefreshSkills(ctx)
		}
	}()
}

// CreateSkillInventory calls skillInventory/create with a minimal user skill.
func (s *Store) CreateSkillInventory(ctx context.Context, name, description, content, root string) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	content = strings.TrimSpace(content)
	root = strings.TrimSpace(root)
	if root == "" {
		root = "user"
	}
	if content == "" {
		content = "# " + name + "\n"
	}
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil || name == "" {
		return
	}
	go func() {
		req := api.SkillInventoryCreateRequest{
			Name:           name,
			Description:    description,
			Content:        content,
			Root:           root,
			ModelInvocable: true,
			UserInvocable:  true,
		}
		if sessionID != "" {
			req.SessionId = &sessionID
		}
		if cwd != "" {
			req.Cwd = &cwd
		}
		var out api.CreateResult
		err := client.Call(ctx, "skillInventory/create", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "skill created " + name
		}
		s.mu.Unlock()
		s.fire()
		if err == nil {
			s.RefreshSkills(ctx)
		}
	}()
}

// SetSkillInvocation calls skillInventory/setInvocation.
func (s *Store) SetSkillInvocation(ctx context.Context, name string, modelInvocable, userInvocable bool) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil || name == "" {
		return
	}
	go func() {
		req := api.SkillInventoryInvocationRequest{
			Name:           name,
			ModelInvocable: modelInvocable,
			UserInvocable:  userInvocable,
		}
		if sessionID != "" {
			req.SessionId = &sessionID
		}
		if cwd != "" {
			req.Cwd = &cwd
		}
		var out api.SetInvocationResult
		err := client.Call(ctx, "skillInventory/setInvocation", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "skill invocation " + name
		}
		s.mu.Unlock()
		s.fire()
		if err == nil {
			s.RefreshSkills(ctx)
		}
	}()
}

// DeleteSkillInventory calls skillInventory/delete for a writable skill.
func (s *Store) DeleteSkillInventory(ctx context.Context, name string) {
	name = strings.TrimSpace(name)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil || name == "" {
		return
	}
	go func() {
		req := api.SkillInventoryRemoveRequest{Name: name}
		if sessionID != "" {
			req.SessionId = &sessionID
		}
		if cwd != "" {
			req.Cwd = &cwd
		}
		var out api.DeleteResult2
		err := client.Call(ctx, "skillInventory/delete", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = "skill deleted " + name
		}
		s.mu.Unlock()
		s.fire()
		if err == nil {
			s.RefreshSkills(ctx)
		}
	}()
}

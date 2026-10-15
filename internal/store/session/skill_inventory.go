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
	Status      string
}

// SkillInventoryTarget returns a validated wire target from the existing workspace.
// It never accepts an arbitrary incoming path or changes directory authorization.
func (s *Store) SkillInventoryTarget(root string) (SkillTarget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	target, err := resolveSkillTarget(root, s.cwd, s.selectedID)
	target.client = s.client
	return target, err
}

func (s *Store) skillInventoryTargetError(err error) {
	s.mu.Lock()
	s.lastAction = err.Error()
	s.mu.Unlock()
	s.fire()
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
	s.mu.Lock()
	client := s.client
	target, targetErr := resolveSkillTarget(root, s.cwd, s.selectedID)
	s.mu.Unlock()
	if targetErr != nil {
		s.skillInventoryTargetError(targetErr)
		return
	}
	if client == nil || query == "" {
		return
	}
	go func() {
		req := api.SkillHubSearchRequest{Query: query, Root: target.Root}
		if target.SessionID != "" {
			req.SessionId = &target.SessionID
		}
		if target.Cwd != "" {
			req.Cwd = &target.Cwd
		}
		var out api.SearchHubResult
		err := client.Call(ctx, "skillInventory/searchHub", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if s.client != client || s.selectedID != target.SessionID || strings.TrimSpace(s.cwd) != target.Cwd {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = err.Error()
			s.skillHubHits = nil
		} else {
			hits := make([]SkillHubHit, 0, len(out))
			for _, it := range out {
				status := ""
				if it.InstallationStatus != nil {
					status = *it.InstallationStatus
				}
				installed := status == "installed"
				hits = append(hits, SkillHubHit{
					Repo:        it.Repo,
					Path:        it.Path,
					SkillName:   it.SkillName,
					Description: it.Description,
					Stars:       it.Stars,
					Installed:   installed,
					Status:      status,
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
	s.mu.Lock()
	client := s.client
	target, targetErr := resolveSkillTarget(root, s.cwd, s.selectedID)
	s.mu.Unlock()
	if targetErr != nil {
		s.skillInventoryTargetError(targetErr)
		return
	}
	if client == nil || hit.Repo == "" || hit.SkillName == "" {
		return
	}
	if hit.Status == "conflict" {
		s.skillInventoryTargetError(fmt.Errorf("skill %q conflicts with an existing skill", hit.SkillName))
		return
	}
	go func() {
		req := api.SkillHubInstallRequest{
			Repo:      hit.Repo,
			Path:      hit.Path,
			SkillName: hit.SkillName,
			Root:      target.Root,
		}
		if target.SessionID != "" {
			req.SessionId = &target.SessionID
		}
		if target.Cwd != "" {
			req.Cwd = &target.Cwd
		}
		var out api.SkillHubInstallResult
		err := client.Call(ctx, "skillInventory/installHub", map[string]any{"request": req}, &out)
		message := ""
		if err == nil {
			message, err = SkillHubInstallMessage(out.Status, out.SkillName, out.Directory)
		}
		s.mu.Lock()
		if s.client != client || s.selectedID != target.SessionID || strings.TrimSpace(s.cwd) != target.Cwd {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = err.Error()
		} else {
			s.lastAction = message
			for i := range s.skillHubHits {
				row := &s.skillHubHits[i]
				if row.Repo == hit.Repo && row.Path == hit.Path && row.SkillName == hit.SkillName {
					row.Status, row.Installed = "installed", true
				}
			}
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
	if content == "" {
		content = "# " + name + "\n"
	}
	s.mu.Lock()
	client := s.client
	target, targetErr := resolveSkillTarget(root, s.cwd, s.selectedID)
	s.mu.Unlock()
	if targetErr != nil {
		s.skillInventoryTargetError(targetErr)
		return
	}
	if client == nil || name == "" {
		return
	}
	go func() {
		req := api.SkillInventoryCreateRequest{
			Name:           name,
			Description:    description,
			Content:        content,
			Root:           target.Root,
			ModelInvocable: true,
			UserInvocable:  true,
		}
		if target.SessionID != "" {
			req.SessionId = &target.SessionID
		}
		if target.Cwd != "" {
			req.Cwd = &target.Cwd
		}
		var out api.CreateResult
		err := client.Call(ctx, "skillInventory/create", map[string]any{"request": req}, &out)
		s.mu.Lock()
		if s.client != client || s.selectedID != target.SessionID || strings.TrimSpace(s.cwd) != target.Cwd {
			s.mu.Unlock()
			return
		}
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

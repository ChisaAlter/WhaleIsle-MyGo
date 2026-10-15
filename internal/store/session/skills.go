package session

import (
	"context"
	"fmt"

	"whaleisle/internal/dsh/api"
)

// SkillOption is one row in the composer skills picker.
type SkillOption struct {
	Name        string
	Description string
	UserOnly    bool // modelInvocable false (disable-model-invocation skills)
}

func flattenSkillList(list api.SkillListValue) []SkillOption {
	out := make([]SkillOption, 0, len(list.Skills))
	for _, sk := range list.Skills {
		if sk.Name == "" {
			continue
		}
		desc := sk.Description
		if sk.WhenToUse != nil && *sk.WhenToUse != "" && desc == "" {
			desc = *sk.WhenToUse
		}
		out = append(out, SkillOption{
			Name:        sk.Name,
			Description: desc,
			UserOnly:    !sk.ModelInvocable,
		})
	}
	return out
}

// ListSkills calls skills/list for the selected session when a client is attached.
func (s *Store) ListSkills(ctx context.Context) []SkillOption {
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cached := append([]SkillOption(nil), s.skillOptions...)
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return cached
	}
	var out api.SkillListValue
	err := client.Call(ctx, "skills/list", map[string]any{
		"request": map[string]any{"sessionId": sessionID},
	}, &out)
	if err != nil {
		return cached
	}
	return flattenSkillList(out)
}

// RefreshSkills loads skills/list for the composer skills menu.
// Falls back to skillInventory/list when session skills/list is unavailable.
func (s *Store) RefreshSkills(ctx context.Context) {
	s.RefreshSkillCatalog(ctx)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	cwd := s.cwd
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var out api.SkillListValue
		var err error
		if sessionID != "" {
			err = client.Call(ctx, "skills/list", map[string]any{
				"request": map[string]any{"sessionId": sessionID},
			}, &out)
		} else {
			err = fmt.Errorf("no session")
		}
		opts := flattenSkillList(out)
		if err != nil || len(opts) == 0 {
			scope := api.SkillInventoryScope{}
			if sessionID != "" {
				scope.SessionId = &sessionID
			}
			if cwd != "" {
				scope.Cwd = &cwd
			}
			var inv api.SkillInventorySnapshot
			if invErr := client.Call(ctx, "skillInventory/list", map[string]any{"request": scope}, &inv); invErr == nil {
				opts = flattenSkillInventory(inv)
				err = nil
			} else if err == nil {
				err = invErr
			}
		}
		s.mu.Lock()
		if s.client != client || s.selectedID != sessionID || s.cwd != cwd {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.skillsErr = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.skillsErr = nil
		s.skillOptions = opts
		s.mu.Unlock()
		s.fire()
	}()
}

func flattenSkillInventory(inv api.SkillInventorySnapshot) []SkillOption {
	out := make([]SkillOption, 0, len(inv.Skills))
	for _, sk := range inv.Skills {
		if sk.Name == "" || !sk.UserInvocable {
			continue
		}
		out = append(out, SkillOption{
			Name:        sk.Name,
			Description: sk.Description,
			UserOnly:    !sk.ModelInvocable,
		})
	}
	return out
}

package session

import (
	"context"
	"strings"

	"whaleisle/internal/dsh/api"
)

// ListFileReferences calls fileReferences/list for the selected session.
// Returns workspace-relative paths matching query (Electron @ mention search).
// Also merges sessionReferenceResolver/candidates mentions when available.
func (s *Store) ListFileReferences(ctx context.Context, query string) []string {
	query = strings.TrimSpace(query)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return nil
	}
	paths := make([]string, 0, 16)
	seen := map[string]bool{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}
	var out api.ListResult
	if err := client.Call(ctx, "fileReferences/list", map[string]any{
		"agentId": sessionID,
		"query":   query,
	}, &out); err == nil {
		for _, item := range out {
			add(item.Path)
		}
	}
	var refs api.CandidatesResult
	if err := client.Call(ctx, "sessionReferenceResolver/candidates", map[string]any{
		"agentId": sessionID,
		"query":   query,
	}, &refs); err == nil {
		for _, it := range refs {
			m := strings.TrimSpace(it.Mention)
			if m == "" {
				m = strings.TrimSpace(it.Label)
			}
			add(m)
		}
	}
	return paths
}

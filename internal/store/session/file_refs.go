package session

import (
	"context"
	"strings"
	"sync"
	"whaleisle/internal/dsh/api"
)

// ReferenceCandidate keeps opaque session identity separate from its display
// title. Mention is Host-produced text and must be inserted without another @.
type ReferenceCandidate struct {
	Kind, Path, Label, Mention, SessionID, Cwd string
	SameWorkspace                              bool
}

// ListReferenceCandidates reads both existing Host namespaces concurrently and
// drops results after a Host or selection-generation change. Quoted paths only
// ask for files, matching the reference client's grammar.
func (s *Store) ListReferenceCandidates(ctx context.Context, sessionID, query string, quoted bool) []ReferenceCandidate {
	s.mu.Lock()
	client, generation := s.client, s.followGeneration
	valid := sessionID != "" && sessionID == s.selectedID
	s.mu.Unlock()
	if client == nil || !valid {
		return nil
	}
	var files api.ListResult
	var refs api.CandidatesResult
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if client.Call(ctx, "fileReferences/list", map[string]any{"agentId": sessionID, "query": query}, &files) != nil {
			files = nil
		}
	}()
	if !quoted {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if client.Call(ctx, "sessionReferenceResolver/candidates", map[string]any{"agentId": sessionID, "query": query}, &refs) != nil {
				refs = nil
			}
		}()
	}
	wg.Wait()
	s.mu.Lock()
	valid = s.client == client && s.selectedID == sessionID && s.followGeneration == generation
	s.mu.Unlock()
	if !valid || ctx.Err() != nil {
		return nil
	}
	out := make([]ReferenceCandidate, 0, len(files)+len(refs))
	seen := map[string]bool{}
	for _, item := range files {
		if item.Path == "" || (item.Kind != "file" && item.Kind != "directory") {
			continue
		}
		key := item.Kind + ":" + item.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ReferenceCandidate{Kind: item.Kind, Path: item.Path, Label: item.Path})
	}
	for _, item := range refs {
		if item.SessionId == "" || item.Mention == "" {
			continue
		}
		key := "session:" + item.SessionId
		if seen[key] {
			continue
		}
		seen[key] = true
		label := item.Label
		if item.DisplayTitle != nil && *item.DisplayTitle != "" {
			label = *item.DisplayTitle
		}
		row := ReferenceCandidate{Kind: "session", Label: label, Mention: item.Mention, SessionID: item.SessionId, SameWorkspace: item.SameWorkspace}
		if item.Cwd != nil {
			row.Cwd = *item.Cwd
		}
		out = append(out, row)
	}
	return out
}

// ListFileReferences preserves the legacy path-only caller surface.
func (s *Store) ListFileReferences(ctx context.Context, query string) []string {
	s.mu.Lock()
	id := s.selectedID
	s.mu.Unlock()
	candidates := s.ListReferenceCandidates(ctx, id, strings.TrimSpace(query), false)
	out := make([]string, 0, len(candidates))
	for _, item := range candidates {
		path := item.Path
		if item.Kind == "session" {
			path = item.Mention
		}
		if item.Kind == "directory" {
			path = strings.TrimSuffix(path, "/") + "/"
		}
		out = append(out, path)
	}
	return out
}

package session

import (
	"context"
	"errors"
	"strings"
	"sync"
	"whaleisle/internal/dsh"
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
	rows, _ := queryReferenceCandidates(ctx, client, sessionID, query, quoted)
	s.mu.Lock()
	valid = s.client == client && s.selectedID == sessionID && s.followGeneration == generation
	s.mu.Unlock()
	if !valid || ctx.Err() != nil {
		return nil
	}
	return rows
}

// QueryReferenceCandidates is the asynchronous composer's cancellable reader.
func (s *Store) QueryReferenceCandidates(ctx context.Context, query string, quoted bool) ([]ReferenceCandidate, error) {
	s.mu.Lock()
	client, id, generation := s.client, s.selectedID, s.followGeneration
	s.mu.Unlock()
	if client == nil || id == "" {
		return nil, errors.New("Host is not connected")
	}
	rows, err := queryReferenceCandidates(ctx, client, id, query, quoted)
	s.mu.Lock()
	current := s.client == client && s.selectedID == id && s.followGeneration == generation
	s.mu.Unlock()
	if !current {
		return nil, context.Canceled
	}
	return rows, err
}

func (s *Store) QueryAgentReferences(ctx context.Context, token uint64, address AgentSessionAddress, query string, quoted bool) ([]ReferenceCandidate, error) {
	s.mu.Lock()
	valid := s.agentPromptActionAdmissionLocked(token, address, true)
	client := s.agentNavigation.client
	generation := s.agentNavigation.generation
	s.mu.Unlock()
	if !valid || client == nil {
		return nil, context.Canceled
	}
	rows, err := queryReferenceCandidates(ctx, client, address.ChildSessionID, query, quoted)
	s.mu.Lock()
	current := s.agentPromptActionAdmissionLocked(token, address, true) && s.agentNavigation.client == client && s.agentNavigation.generation == generation
	s.mu.Unlock()
	if !current {
		return nil, context.Canceled
	}
	return rows, err
}

func queryReferenceCandidates(ctx context.Context, client *dsh.Client, id, query string, quoted bool) ([]ReferenceCandidate, error) {
	var files api.ListResult
	var refs api.CandidatesResult
	var fileErr, refErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fileErr = client.Call(ctx, "fileReferences/list", map[string]any{"agentId": id, "query": query}, &files)
	}()
	if !quoted {
		wg.Add(1)
		go func() {
			defer wg.Done()
			refErr = client.Call(ctx, "sessionReferenceResolver/candidates", map[string]any{"agentId": id, "query": query}, &refs)
		}()
	}
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
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
	return out, errors.Join(fileErr, refErr)
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

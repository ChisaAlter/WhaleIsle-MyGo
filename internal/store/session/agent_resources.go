package session

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"whaleisle/internal/desktop/workspacefs"
	"whaleisle/internal/dsh/api"
)

type AgentResourceRead struct {
	Table             *workspacefs.XLSXPage
	Busy              bool
	Path, Text, Error string
	EOF               bool
	Lines             float64
}

func cloneAgentResources(in map[string]AgentResourceRead) map[string]AgentResourceRead {
	out := make(map[string]AgentResourceRead, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func cloneAgentTurnResults(in map[TurnFileRequest]TurnFileResult) map[TurnFileRequest]TurnFileResult {
	out := make(map[TurnFileRequest]TurnFileResult, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func (s *Store) agentResourceCurrentLocked(token uint64, address AgentSessionAddress) bool {
	return s.agentNavigationOwnedLocked() && s.agentNavigation.view.Token == token && s.agentNavigation.view.Address == address && address.validChild()
}

// ReadAgentResource never borrows selectedID or the main attachment cache key.
// Both image authorization and workspace resolution happen on the child's Host.
func (s *Store) ReadAgentResource(ctx context.Context, token uint64, address AgentSessionAddress, ref string, more bool) {
	s.mu.Lock()
	if !s.agentResourceCurrentLocked(token, address) || s.agentNavigation.view.Resources[ref].Busy {
		s.mu.Unlock()
		return
	}
	n := &s.agentNavigation
	previous := n.view.Resources[ref]
	client := n.client
	captured := *client
	endpoint := client.Endpoint
	if n.view.Resources == nil {
		n.view.Resources = map[string]AgentResourceRead{}
	}
	n.view.Resources[ref] = AgentResourceRead{Busy: true, Text: previous.Text, Lines: previous.Lines}
	s.mu.Unlock()
	s.fire()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result := AgentResourceRead{}
	var err error
	if id, ok := strings.CutPrefix(ref, "attachment:"); ok {
		identity := fmt.Sprintf("%s\x00%v\x00%d\x00%s", turnFileHistoryBinding(client, address.ChildSessionID), address, token, id)
		result.Path, err = fetchSessionAttachment(ctx, &captured, address.ChildSessionID, id, identity)
		result.EOF = true
	} else {
		offset, limit := float64(0), float64(2000)
		if more {
			offset = previous.Lines
		}
		var out api.WorkspaceFileText
		err = captured.Call(ctx, "workspaceFiles/read", map[string]any{"workspaceFileScopeId": address.ChildSessionID, "path": ref, "range": api.WorkspaceFileRange{Offset: &offset, Limit: &limit}}, &out)
		result.Text = out.Text
		result.EOF = out.Eof
		result.Lines = out.Offset + out.Lines
		if more {
			result.Text = previous.Text + out.Text
		}
	}
	if err != nil {
		result = previous
		result.Busy = false
		result.Error = err.Error()
	}
	s.mu.Lock()
	if s.agentResourceCurrentLocked(token, address) && s.client == client && client.Endpoint == endpoint {
		s.agentNavigation.view.Resources[ref] = result
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) AgentTurnFiles(token uint64) TurnFilesSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := &s.agentNavigation
	if !s.agentResourceCurrentLocked(token, n.view.Address) {
		return TurnFilesSnapshot{}
	}
	return TurnFilesSnapshot{Scope: TurnFileScope{Binding: turnFileHistoryBinding(n.client, n.view.SessionID), SessionID: n.view.SessionID, Generation: token}, HostLabel: n.client.Endpoint.Origin, Results: cloneAgentTurnResults(n.view.TurnFiles.Results)}
}
func (s *Store) agentTurnTargetLocked(target TurnFileTarget) (string, bool) {
	v := s.agentNavigation.view
	d, ok := TurnDeliveries(v.Messages)[target.Turn]
	if !ok {
		return "", false
	}
	if target.Kind == "presented" {
		for _, f := range d.Files {
			if f.Target == target {
				return f.Path, true
			}
		}
	}
	if target.Kind == "changes" && d.Changes != nil && target.Seq == d.Changes.Seq {
		if target.Index == -1 {
			return "", true
		}
		sum := v.TurnFiles.Results[TurnFileRequest{Target: *d.Changes, Action: "summary"}].Summary
		if sum != nil && target.Index >= 0 && target.Index < len(sum.Files) {
			return sum.Files[target.Index].Path, true
		}
	}
	return "", false
}
func (s *Store) AgentTurnPreviewPath(token uint64, address AgentSessionAddress, target TurnFileTarget) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.agentResourceCurrentLocked(token, address) {
		return "", false
	}
	return s.agentTurnTargetLocked(target)
}

// RunAgentTurnFileAction reuses the main delivery protocol with child coordinates.
func (s *Store) RunAgentTurnFileAction(ctx context.Context, token uint64, address AgentSessionAddress, request TurnFileRequest) {
	s.mu.Lock()
	path, ok := s.agentTurnTargetLocked(request.Target)
	if !ok || !s.agentResourceCurrentLocked(token, address) {
		s.mu.Unlock()
		return
	}
	n := &s.agentNavigation
	key := request.StateKey()
	if n.view.TurnFiles.Results[key].Busy {
		s.mu.Unlock()
		return
	}
	route, method := "", http.MethodGet
	switch request.Action {
	case "host":
		route = "present.host"
	case "summary":
		if request.Target.Kind == "changes" && request.Target.Index == -1 {
			route = "changes.summary"
		}
	case "diff":
		if request.Target.Kind == "changes" && request.Target.Index >= 0 {
			route = "changes.diff"
		}
	case "open", "reveal":
		if request.Target.Index >= 0 {
			method = http.MethodPost
			if request.Target.Kind == "presented" {
				route = "present.open"
			} else if request.Target.Kind == "changes" {
				route = "changes.open"
			}
		}
	}
	if route == "" {
		s.mu.Unlock()
		return
	}
	client := n.client
	captured := *client
	endpoint := client.Endpoint
	scope := TurnFileScope{Binding: turnFileHistoryBinding(client, address.ChildSessionID), SessionID: address.ChildSessionID, Generation: token}
	if n.view.TurnFiles.Results == nil {
		n.view.TurnFiles.Results = map[TurnFileRequest]TurnFileResult{}
	}
	n.view.TurnFiles.Results[key] = TurnFileResult{Busy: true, Action: request.Action}
	s.mu.Unlock()
	s.fire()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	result := runTurnFileHTTP(ctx, &captured, scope, request, route, method, path)
	s.mu.Lock()
	if s.agentResourceCurrentLocked(token, address) && s.client == client && client.Endpoint == endpoint {
		if _, valid := s.agentTurnTargetLocked(request.Target); valid {
			s.agentNavigation.view.TurnFiles.Results[key] = result
		}
	}
	s.mu.Unlock()
	s.fire()
}

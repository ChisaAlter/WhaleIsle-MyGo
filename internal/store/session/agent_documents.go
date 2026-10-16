package session

import (
	"context"
	"errors"
	"time"

	"whaleisle/internal/desktop/mediapreview"
	"whaleisle/internal/desktop/workspacefs"
)

// AgentMediaPreviewSource captures the complete child navigation identity;
// ordinary selection remains on the parent and is never used as file scope.
func (s *Store) AgentMediaPreviewSource(token uint64, address AgentSessionAddress) (mediapreview.Caller, func() bool, error) {
	s.mu.Lock()
	if !s.agentResourceCurrentLocked(token, address) {
		s.mu.Unlock()
		return nil, nil, errors.New("child document is no longer selected")
	}
	client := s.agentNavigation.client
	captured := *client
	endpoint := client.Endpoint
	s.mu.Unlock()
	valid := func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.agentResourceCurrentLocked(token, address) && s.client == client && client.Endpoint == endpoint
	}
	return &captured, valid, nil
}
func (s *Store) SetAgentResourceStatus(token uint64, address AgentSessionAddress, path string, busy bool, err error) {
	s.mu.Lock()
	if s.agentResourceCurrentLocked(token, address) {
		if s.agentNavigation.view.Resources == nil {
			s.agentNavigation.view.Resources = map[string]AgentResourceRead{}
		}
		read := s.agentNavigation.view.Resources[path]
		read.Busy = busy
		read.Error = ""
		read.EOF = true
		if err != nil {
			read.Error = err.Error()
		}
		s.agentNavigation.view.Resources[path] = read
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) ReadAgentWorkbook(ctx context.Context, token uint64, address AgentSessionAddress, path string, request workspacefs.XLSXPageRequest) {
	reader, valid, err := s.AgentMediaPreviewSource(token, address)
	if err != nil {
		return
	}
	s.mu.Lock()
	if s.agentNavigation.view.Resources[path].Busy {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	s.SetAgentResourceStatus(token, address, path, true, nil)
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	data, err := mediapreview.ReadSnapshot(ctx, reader, address.ChildSessionID, path, 16<<20, valid)
	var page *workspacefs.XLSXPage
	if err == nil {
		page, err = workspacefs.ReadWorkbookBytesPage(ctx, data, request)
	}
	if !valid() {
		return
	}
	s.mu.Lock()
	if s.agentResourceCurrentLocked(token, address) {
		previous := s.agentNavigation.view.Resources[path]
		result := AgentResourceRead{Table: page, EOF: true}
		if err != nil {
			result.Table = previous.Table
			result.Error = err.Error()
		}
		s.agentNavigation.view.Resources[path] = result
	}
	s.mu.Unlock()
	s.fire()
}

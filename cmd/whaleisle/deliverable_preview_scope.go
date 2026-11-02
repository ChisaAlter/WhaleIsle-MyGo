package main

import (
	"fmt"
	"net"
	"net/url"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/store/session"
)

// deliverablePreviewOwner keeps the process and session identity from the frame
// that rendered the card. A same-address Host restart is a different owner.
type deliverablePreviewOwner struct {
	supervisor *harness.Supervisor
	client     *dsh.Client
	generation int
	endpoint   harness.Endpoint
	store      *session.Store
	scope      session.TurnFileScope
	binding    string
	cwd        string
}

func (s *mainState) captureDeliverablePreviewOwner(sessions *session.Store, scope session.TurnFileScope, binding, cwd string, host harness.Status, client *dsh.Client) deliverablePreviewOwner {
	if s == nil {
		return deliverablePreviewOwner{}
	}
	s.mu.Lock()
	sup := s.sup
	s.mu.Unlock()
	return deliverablePreviewOwner{supervisor: sup, client: client, generation: host.Generation, endpoint: host.Endpoint,
		store: sessions, scope: scope, binding: binding, cwd: cwd}
}

// Loopback is not itself proof of locality: an SSH tunnel can also use it.
// Require the exact supervised local process generation and selected file owner.
func (s *mainState) localDeliverablePreviewScope(owner deliverablePreviewOwner) bool {
	if s == nil || owner.store == nil || owner.supervisor == nil || owner.client == nil || owner.generation == 0 || owner.scope.Binding == "" || owner.scope.SessionID == "" || owner.cwd == "" {
		return false
	}
	s.mu.Lock()
	current := s.sessions == owner.store && s.sup == owner.supervisor && !s.quitting &&
		s.bootStatus.State == harness.Ready && s.bootStatus.Generation == owner.generation && s.bootStatus.Endpoint == owner.endpoint
	s.mu.Unlock()
	if !current {
		return false
	}
	live := owner.supervisor.Status()
	if live.State != harness.Ready || live.Generation != owner.generation || live.Endpoint != owner.endpoint {
		return false
	}
	u, err := url.Parse(live.Endpoint.Origin)
	if err != nil || (u.Hostname() != "localhost" && !net.ParseIP(u.Hostname()).IsLoopback()) {
		return false
	}
	s.dshMu.Lock()
	client, endpoint := s.dshClient, s.dshEndpoint
	s.dshMu.Unlock()
	if client != owner.client || endpoint != live.Endpoint.Origin+"\x00"+live.Endpoint.Cookie || owner.binding != fmt.Sprintf("%p\x00%s", client, owner.scope.SessionID) {
		return false
	}
	originBinding, id, selectedCwd := owner.store.WorkspaceFileReferenceOrigin()
	if originBinding != owner.binding || id != owner.scope.SessionID || selectedCwd != owner.cwd {
		return false
	}
	return owner.store.TurnFilesScope() == owner.scope
}

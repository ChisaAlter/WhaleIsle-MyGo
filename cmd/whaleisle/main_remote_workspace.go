package main

import (
	"context"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	remotestore "whaleisle/internal/store/remote"
	"whaleisle/internal/store/session"
)

// A mirror completion is a receipt from one machine, not an instruction to
// adopt its pathname on whichever Host happens to be current later.
type remoteWorkspaceAdoption struct {
	state            *mainState
	win              *mygo.Window
	sessions         *session.Store
	remote           *remotestore.Store
	client           *dsh.Client
	host             harness.Status
	lease, hostEpoch uint64
	navigation       *session.ComposerTarget
	config           *config.Store
}

func (s *mainState) captureRemoteWorkspaceAdoption(win *mygo.Window, sessions *session.Store, remote *remotestore.Store) *remoteWorkspaceAdoption {
	if sessions == nil || remote == nil {
		return nil
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.mu.Lock()
	host, lease, cfg := s.bootStatus, s.remoteWorkspaceLease.Load(), s.cfgStore
	s.mu.Unlock()
	s.dshMu.Unlock()
	if client == nil || host.State != harness.Ready || client.Endpoint.Origin != host.Endpoint.Origin || client.Endpoint.Cookie != host.Endpoint.Cookie {
		return nil
	}
	navigation, err := sessions.CaptureComposerTarget(sessions.SelectedSessionID())
	if err != nil {
		return nil
	}
	return &remoteWorkspaceAdoption{state: s, win: win, sessions: sessions, remote: remote, client: client, host: host, lease: lease, hostEpoch: sessions.ComposerHostGeneration(), navigation: navigation, config: cfg}
}

func (op *remoteWorkspaceAdoption) current(choice remotestore.DirectoryChoice) bool {
	if op == nil || op.state.remoteWorkspaceLease.Load() != op.lease || !op.navigation.NavigationCurrent() || op.sessions.ComposerHostGeneration() != op.hostEpoch || !op.remote.DirectoryChoiceCurrent(choice) {
		return false
	}
	s := op.state
	s.mu.Lock()
	current := s.sessions == op.sessions && s.remoteMachines == op.remote && s.win == op.win && !s.quitting && s.bootStatus.State == harness.Ready && s.bootStatus.Generation == op.host.Generation && s.bootStatus.Endpoint == op.host.Endpoint
	s.mu.Unlock()
	s.dshMu.Lock()
	sameClient := s.dshClient == op.client
	s.dshMu.Unlock()
	return current && sameClient
}

// RegisterWorkspace calls persistence under its own Store lock. This last
// boundary therefore uses the atomic Host/client lease, not a main/session
// lock or a newly resolved transport. Its token already pins the Store client.
func (op *remoteWorkspaceAdoption) persist(choice remotestore.DirectoryChoice, path string) error {
	if op.state.remoteWorkspaceLease.Load() != op.lease || !op.remote.DirectoryChoiceCurrent(choice) {
		return context.Canceled
	}
	if op.config == nil {
		return session.ErrNoClient
	}
	return op.config.UpdateChecked(func(c *config.Config) error {
		if op.state.remoteWorkspaceLease.Load() != op.lease || !op.remote.DirectoryChoiceCurrent(choice) {
			return context.Canceled
		}
		c.Workspace = path
		return nil
	})
}

func (op *remoteWorkspaceAdoption) register(choice remotestore.DirectoryChoice) bool {
	if !op.current(choice) {
		return false
	}
	s := op.state
	adoption := s.captureWorkspaceAdoption(op.sessions)
	if !op.current(choice) {
		return false
	}
	op.sessions.OpenWorkspacePicker()
	request := op.sessions.Snapshot().WorkspaceNavigation
	if request == nil || request.Mode != "pick" || request.Busy {
		return false
	}
	// Recheck after minting the Store token too: a replacement Store Host must
	// never receive even workspace/create for the previous Host's mirror path.
	if !op.current(choice) {
		op.sessions.CancelWorkspaceNavigation(request.Token)
		return false
	}
	op.sessions.RegisterWorkspace(context.Background(), request.Token, choice.LocalMirror, func(path string) error { return op.persist(choice, path) }, func(row session.WorkspaceRow) { s.finishWorkspaceAdoption(op.win, op.sessions, row, adoption) })
	return true
}

func (s *mainState) remoteWorkspaceChoiceHandler(win *mygo.Window, sessions *session.Store, remote *remotestore.Store) func(remotestore.DirectoryChoice) {
	op := s.captureRemoteWorkspaceAdoption(win, sessions, remote)
	return func(choice remotestore.DirectoryChoice) {
		if op == nil || win == nil || win.IsDestroyed() {
			return
		}
		// The remote Store completes on its worker. Native adoption and draft
		// capture run on the UI owner, with the original receipt checked there.
		win.Update(func() {
			if !win.IsDestroyed() {
				op.register(choice)
			}
		})
	}
}

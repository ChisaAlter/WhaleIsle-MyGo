package session

import (
	"context"
	"encoding/json"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type commandCatalog struct {
	sessionID string
	client    *dsh.Client
	revision  uint64
	cancel    context.CancelFunc
	loaded    bool
	loading   bool
	err       error
	items     []api.ListResultItem2
}

func (s *Store) invalidateCommandsLocked() {
	revision := s.commandCatalog.revision + 1
	if s.commandCatalog.cancel != nil {
		s.commandCatalog.cancel()
	}
	s.commandCatalog = commandCatalog{revision: revision}
}

// EnsureSlashCommands never performs network work on the paint thread. A failed
// load stays visible until explicit retry or a catalog/session/Host change.
func (s *Store) EnsureSlashCommands(ctx context.Context) {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	client, id := s.client, s.selectedID
	if client == nil || id == "" {
		s.mu.Unlock()
		return
	}
	catalog := &s.commandCatalog
	if catalog.client != client || catalog.sessionID != id {
		s.invalidateCommandsLocked()
		catalog = &s.commandCatalog
		catalog.client, catalog.sessionID = client, id
	}
	if catalog.loading || catalog.loaded {
		s.mu.Unlock()
		return
	}
	requestCtx, cancel := context.WithCancel(ctx)
	catalog.cancel = cancel
	catalog.loading = true
	revision := catalog.revision
	s.mu.Unlock()
	go func() {
		defer cancel()
		s.fire()
		var out api.ListResult3
		err := client.Call(requestCtx, "commands/list", map[string]any{"agentId": id}, &out)
		s.mu.Lock()
		catalog := &s.commandCatalog
		if s.client != client || s.selectedID != id || catalog.revision != revision ||
			catalog.client != client || catalog.sessionID != id {
			s.mu.Unlock()
			return
		}
		catalog.loading, catalog.loaded, catalog.cancel = false, true, nil
		catalog.err = err
		if err == nil && requestCtx.Err() == nil {
			catalog.items = cloneCommandDescriptors(out)
		} else if err == nil {
			catalog.err = requestCtx.Err()
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// RefreshSlashCommands is the explicit retry action, also useful after a local
// preset-selection acknowledgment whose registry notification may arrive later.
func (s *Store) RefreshSlashCommands(ctx context.Context) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.invalidateCommandsLocked()
	s.mu.Unlock()
	s.EnsureSlashCommands(ctx)
	s.fire()
}

func cloneCommandDescriptors(items []api.ListResultItem2) []api.ListResultItem2 {
	out := append([]api.ListResultItem2(nil), items...)
	for i := range out {
		if out[i].DefinitionId != nil {
			id := *out[i].DefinitionId
			out[i].DefinitionId = &id
		}
		if out[i].Input != nil {
			input := *out[i].Input
			if input.Attachments != nil {
				accepts := *input.Attachments
				input.Attachments = &accepts
			}
			out[i].Input = &input
		}
	}
	return out
}

func (s *Store) snapshotCommandsLocked(st *ViewState) {
	catalog := &s.commandCatalog
	if catalog.client != s.client || catalog.sessionID != s.selectedID {
		return
	}
	st.SlashCommands = cloneCommandDescriptors(catalog.items)
	st.SlashCommandsLoading = catalog.loading
	if catalog.err != nil {
		st.SlashCommandsErr = catalog.err.Error()
	}
}

// Caller already validated the $events owner while holding s.mu.
func (s *Store) applyCommandCatalogEmitLocked(emit dsh.EventsEmit) bool {
	switch emit.Event {
	case "commands/change":
		if len(emit.Args) != 0 {
			return false
		}
	case "agent-preset/selected":
		if len(emit.Args) < 1 {
			return false
		}
		var id string
		if json.Unmarshal(emit.Args[0], &id) != nil || id != s.selectedID {
			return false
		}
	default:
		return false
	}
	s.invalidateCommandsLocked()
	return true
}

package settings

import (
	"context"
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh"
)

// OpenSettingsDocument opens the Host configuration document in an external editor,
// then refreshes settings/describe. Use RefreshDescribe to load an in-app settings
// panel without launching an editor.
func (s *Store) OpenSettingsDocument(ctx context.Context) {
	s.mu.Lock()
	client := s.client
	generation := s.generation
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		var out json.RawMessage
		err := client.Call(ctx, "settings/openSettingsDocument", map[string]any{}, &out)
		s.mu.Lock()
		if s.client != client || s.generation != generation {
			s.mu.Unlock()
			return
		}
		if err != nil {
			s.lastAction = "openSettingsDocument: " + err.Error()
		} else {
			s.lastAction = "settings document opened"
		}
		s.mu.Unlock()
		s.fire()
		s.RefreshDescribe(ctx)
	}()
}

// IsDocumentUpdatedEmit reports whether a $events emit likely signals that the
// settings document changed (Electron parity: settings/document-updated).
func IsDocumentUpdatedEmit(event string, argsPreview string) bool {
	e := strings.ToLower(strings.TrimSpace(event))
	blob := e + " " + strings.ToLower(argsPreview)
	return strings.Contains(blob, "document-updated") ||
		strings.Contains(blob, "settings/document") ||
		strings.Contains(blob, "settings.document") ||
		(strings.Contains(blob, "settings") && strings.Contains(blob, "updated"))
}

// OnEmitMaybeRefresh refreshes describe when emit looks like a settings document update.
func (s *Store) OnEmitMaybeRefresh(ctx context.Context, emit dsh.EventsEmit) {
	if emit.Event == "credentials/reference-updated" || emit.Event == "llm/adapters-updated" {
		s.mu.Lock()
		s.onboardingCredentialEpoch++
		s.mu.Unlock()
	}
	if emit.Event == "llm/adapters-updated" {
		s.InvalidateModelConfigurationCatalog(ctx)
	}
	if emit.Event == "credentials/reference-updated" || emit.Event == "credentials/record-updated" {
		s.mu.Lock()
		key := s.webSearchKey
		if key != nil && key.currentLocked() {
			key.state.Known = false
			key.state.Error = ""
		}
		s.mu.Unlock()
		if key != nil {
			key.Refresh(ctx)
		}
	}
	preview := ""
	if len(emit.Args) > 0 {
		preview = string(emit.Args[0])
		if len(preview) > 200 {
			preview = preview[:200]
		}
	}
	if !IsDocumentUpdatedEmit(emit.Event, preview) {
		return
	}
	s.InvalidateModelConfigurationCatalog(ctx)
	s.mu.Lock()
	s.lastAction = "settings document updated"
	s.mu.Unlock()
	s.RefreshDescribe(ctx)
}

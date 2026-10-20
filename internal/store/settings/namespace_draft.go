package settings

import (
	"context"
	"errors"
)

// The reasons a per-card save may refuse to start. Everything else the save can
// run into — a validation failure, a refused write, a Host that did not accept
// the staged values — is reported through Snapshot's Err and LastAction, as the
// page's own saves are.
var (
	// ErrSettingsNotReady is a save before a Host is attached and described.
	ErrSettingsNotReady = errors.New("settings: no Host is described")
	// ErrNamespaceUnknown is a namespace this Host has not described.
	ErrNamespaceUnknown = errors.New("settings: the Host has not described this namespace")
	// ErrSettingsBusy is a save while a read or a write is already in flight.
	ErrSettingsBusy = errors.New("settings: a settings read or write is in flight")
	// ErrSettingsReadOnly is a save to a Host whose settings are read-only.
	ErrSettingsReadOnly = errors.New("settings: the Host settings are read-only")
)

// NamespaceDraft returns a copy of one namespace's field drafts and whether it
// has unsaved changes, without touching the namespace the generic settings
// editor has selected: a provider card can show its own 「取消」「保存」 from it
// while every other card keeps its drafts.
//
// An unknown namespace reports no drafts. The drafts are what describe seeded
// plus what was staged since; they are not the stored values.
func (s *Store) NamespaceDraft(namespace string) (map[string]string, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.findNamespaceLocked(namespace) == nil {
		return nil, false
	}
	return copyDrafts(s.drafts[namespace]), len(s.dirty[namespace]) > 0
}

// SaveNamespace submits only one namespace's staged fields, leaving every other
// namespace's drafts, dirty flags and revisions untouched: the save a provider
// card's own button makes, without selecting that namespace in the rail.
//
// It reuses the namespace binding's save sequence — the same validation, secret
// preservation and revision handling every plugin body and the generic editor
// use — so nothing is applied twice or by a second path. The write finishes in
// the background: a nil error means it started, and Snapshot's Saving,
// LastAction and Err, or a binding's Snapshot, report the outcome. A namespace
// with nothing staged is a no-op.
//
//	// One card, one save: the other cards keep their drafts.
//	app.Store.SaveNamespace(ctx, "llm-deepseek")
func (s *Store) SaveNamespace(ctx context.Context, namespace string) error {
	if s == nil {
		return ErrSettingsNotReady
	}
	s.mu.Lock()
	described := len(s.namespaces) > 0
	known := s.findNamespaceLocked(namespace) != nil
	connected := s.client != nil
	busy := s.saving || s.loading
	writable := s.writable
	s.mu.Unlock()
	switch {
	case !connected || !described:
		return ErrSettingsNotReady
	case !known:
		return ErrNamespaceUnknown
	case busy:
		return ErrSettingsBusy
	case !writable:
		return ErrSettingsReadOnly
	}
	binding := s.BindNamespace(namespace)
	if binding == nil {
		return ErrNamespaceUnknown
	}
	binding.Save(ctx)
	return nil
}

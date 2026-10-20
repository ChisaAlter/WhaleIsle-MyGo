package session

import (
	"context"
	"errors"
	"sort"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

// NativeCommandOwner binds an invocation to one retained session and one native
// registration. Identical session IDs from another Host are not interchangeable.
type NativeCommandOwner struct {
	HostGeneration, SessionGeneration, PermissionCatalogRevision uint64
	RegistrationGeneration, PortGeneration                       uint64
	SessionID, Revision, ReleaseVersion, ReleaseCommit           string
}

type NativeCommandOption struct {
	ID, Label, CatalogLabel, Group, GroupLabel, Detail, Badge string
	Active, Confirmation                                      bool
	Model                                                     *api.ModelCatalogDefault
	Failure                                                   string
}

func (s *Store) nativeCommandCurrentLocked(o NativeCommandOwner, name string) bool {
	g := s.nativeContributions.snapshot
	index := nativeCommandIndex(name)
	if index < 0 || o.RegistrationGeneration != s.nativeContributions.generation || o.PortGeneration != s.nativeContributions.ports[index] {
		return false
	}
	if s.client == nil || o.SessionID == "" || s.selectedID != o.SessionID || s.followGeneration != o.SessionGeneration || s.composerHostGeneration != o.HostGeneration || !g.Known || g.CommandRevision(name) != o.Revision || g.ReleaseVersion != o.ReleaseVersion || g.ReleaseCommit != o.ReleaseCommit {
		return false
	}
	switch name {
	case "file":
		return g.FileCommand && s.modelSelectionAvailableLocked()
	case "model":
		return g.ModelCommand && s.modelSelectionAvailableLocked()
	case "permission", "feedback":
		if name == "permission" && (!g.PermissionCommand || s.permissionCatalogRevision != o.PermissionCatalogRevision || !s.permissionProjectionCurrentLocked()) {
			return false
		}
		if name == "permission" {
			return true
		}
		if name == "feedback" && !g.FeedbackCommand {
			return false
		}
		if s.commandCatalog.client != s.client || s.commandCatalog.sessionID != o.SessionID || !s.commandCatalog.loaded || s.commandCatalog.err != nil {
			return false
		}
		for _, d := range s.commandCatalog.items {
			if d.Name == name {
				return true
			}
		}
	}
	return false
}
func (s *Store) NativeCommandCurrent(o NativeCommandOwner, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nativeCommandCurrentLocked(o, name)
}

// LoadNativeCommandOptions keeps catalog failures in the command's own retry
// surface. Rows carry opaque selections; captions and IDs are never parsed.
func (s *Store) LoadNativeCommandOptions(ctx context.Context, o NativeCommandOwner, name string) ([]NativeCommandOption, error) {
	s.mu.Lock()
	if ctx.Err() != nil || !s.nativeCommandCurrentLocked(o, name) {
		s.mu.Unlock()
		return nil, errors.New(storelocale.T(storelocale.KeyStoreSessionCommandOwnerChanged))
	}
	client, follow := s.client, s.followGeneration
	request := s.modelState.CatalogRequest
	if name == "model" {
		s.modelState.CatalogRequest++
		request = s.modelState.CatalogRequest
	}
	s.mu.Unlock()
	var rows []NativeCommandOption
	switch name {
	case "model":
		var cat api.ModelCatalog
		if err := client.Call(ctx, "session/modelCatalog", map[string]any{}, &cat); err != nil {
			return nil, err
		}
		s.mu.Lock()
		if !s.nativeCommandCurrentLocked(o, name) || !s.sessionResultCurrentLocked(ctx, client, o.SessionID, follow) || s.modelState.CatalogRequest != request {
			s.mu.Unlock()
			return nil, errors.New("Command owner changed")
		}
		s.modelOptions = flattenModelCatalog(cat)
		s.modelState.Default = cloneModelSelection(cat.Default)
		s.modelState.CatalogReady = true
		current, known := s.currentModelSelectionLocked()
		options := append([]ModelOption(nil), s.modelOptions...)
		s.mu.Unlock()
		priority := func(id string) int {
			if id == "deepseek-account" {
				return 0
			}
			if id == "deepseek-official" {
				return 1
			}
			return 2
		}
		sort.SliceStable(cat.Groups, func(i, j int) bool { return priority(cat.Groups[i].Id) < priority(cat.Groups[j].Id) })
		for _, g := range cat.Groups {
			for _, m := range g.Models {
				selection := api.ModelCatalogDefault{Provider: g.Id, Model: m.Id}
				if m.Reasoning != nil {
					selection.ReasoningEffort = m.Reasoning.DefaultEffort
				}
				active := known && current.Provider == g.Id && current.Model == m.Id
				if active && current.ReasoningEffort != nil {
					selection.ReasoningEffort = current.ReasoningEffort
				}
				selection = cloneModelSelection(selection)
				rows = append(rows, NativeCommandOption{ID: g.Id + "/" + m.Id, Label: m.Name, CatalogLabel: labelForSelection(selection, options), Group: g.Id, GroupLabel: g.Name, Active: active, Model: &selection})
			}
		}
		for _, f := range cat.Failures {
			rows = append(rows, NativeCommandOption{ID: "failure/" + f.Id, Label: f.Name, Detail: f.Message, Failure: "this provider's catalog failed to load — pick a model from a loaded group"})
		}
	case "permission":
		var cat api.PermissionCatalog
		if err := client.Call(ctx, "permissionPresets/catalog", map[string]any{}, &cat); err != nil {
			return nil, err
		}
		s.mu.Lock()
		if !s.nativeCommandCurrentLocked(o, name) || !s.sessionResultCurrentLocked(ctx, client, o.SessionID, follow) {
			s.mu.Unlock()
			return nil, errors.New("Command owner changed")
		}
		current := s.permissionProjection.Value
		s.permissionOptions = flattenPermissionCatalog(cat)
		s.mu.Unlock()
		for _, p := range cat.Options {
			row := NativeCommandOption{ID: p.Value, Label: p.Name, Active: p.Value == current, Confirmation: permissionNeedsConfirmation(p.Value)}
			if p.Description != nil {
				row.Detail = *p.Description
			}
			rows = append(rows, row)
		}
	default:
		return nil, errors.New("Command has no popup")
	}
	s.fire()
	return rows, nil
}

// SelectNativeCommandModel returns a receipt only after the captured Host has
// confirmed its projection. A retired registration cannot clear a new draft.
func (s *Store) SelectNativeCommandModel(ctx context.Context, o NativeCommandOwner, selection api.ModelCatalogDefault) error {
	done := make(chan error, 1)
	s.selectModelSelectionWithReceipt(ctx, o.SessionID, selection, false, &o, func(err error) { done <- err })
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// acknowledged is the command popup's explicit risk checkbox, not inferred
// from opening a menu or choosing the same row in another surface.
func (s *Store) SelectNativeCommandPermission(ctx context.Context, o NativeCommandOwner, value string, acknowledged bool) error {
	s.mu.Lock()
	if ctx.Err() != nil || !s.nativeCommandCurrentLocked(o, "permission") {
		s.mu.Unlock()
		return errors.New("Command owner changed")
	}
	opt, ok := permissionOptionByValue(value, s.permissionOptions)
	if !ok || s.permissionProjection.Pending {
		s.mu.Unlock()
		return errors.New("Permission selection unavailable or busy")
	}
	if permissionNeedsConfirmation(value) && !acknowledged {
		s.mu.Unlock()
		return errors.New(storelocale.T(storelocale.KeyStoreSessionExplicitPermissionRiskAcknowledgement))
	}
	if value == s.permissionProjection.Value {
		s.mu.Unlock()
		return nil
	}
	client, follow := s.client, s.followGeneration
	s.permissionProjection.Pending = true
	s.permissionsErr = nil
	s.mu.Unlock()
	s.fire()
	return s.settlePermissionPreset(ctx, client, o.SessionID, follow, opt, &o)
}

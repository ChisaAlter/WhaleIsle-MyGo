package session

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/storelocale"
)

// PermissionOption is one row in the composer permission preset picker.
type PermissionOption struct {
	Label string
	Value string
}

func flattenPermissionCatalog(cat api.PermissionCatalog) []PermissionOption {
	var out []PermissionOption
	seen := map[string]struct{}{}
	appendItems := func(items []api.PermissionCatalogOptionsItem) {
		for _, it := range items {
			if it.Value == "" {
				continue
			}
			if _, ok := seen[it.Value]; ok {
				continue
			}
			seen[it.Value] = struct{}{}
			label := it.Name
			if label == "" {
				label = it.Value
			}
			out = append(out, PermissionOption{Label: label, Value: it.Value})
		}
	}
	// DefaultOptions carries the authoritative label for the default preset;
	// the general options list may repeat the same value with a stale label.
	appendItems(cat.DefaultOptions)
	appendItems(cat.Options)
	return out
}

func labelForPresetValue(value string, options []PermissionOption) string {
	for _, o := range options {
		if o.Value == value {
			return o.Label
		}
	}
	if value != "" {
		return value
	}
	return ""
}

func permissionOptionByLabel(label string, options []PermissionOption) (PermissionOption, bool) {
	var match PermissionOption
	found := false
	for _, o := range options {
		if o.Label == label {
			if found {
				return PermissionOption{}, false
			}
			match, found = o, true
		}
	}
	return match, found
}

// RefreshPermissionCatalog reads process-level choices only. The catalog's
// defaultPreset belongs to future sessions, never the composer's active value.
func (s *Store) RefreshPermissionCatalog(ctx context.Context) {
	s.mu.Lock()
	run := s.beginPermissionCatalogLocked(ctx)
	s.mu.Unlock()
	if run != nil {
		go run()
	}
}

// The events callback must retain its actual delivery owner through admission;
// an old notification cannot begin a read against a replacement Host.
func (s *Store) RefreshPermissionCatalogForEvents(owner *EventsOwner) {
	s.mu.Lock()
	if !s.eventsOwnerCurrentLocked(owner) {
		s.mu.Unlock()
		return
	}
	run := s.beginPermissionCatalogLocked(owner.ctx)
	s.mu.Unlock()
	if run != nil {
		go run()
	}
}

func (s *Store) beginPermissionCatalogLocked(ctx context.Context) func() {
	s.permissionCatalogRevision++
	s.invalidateViewLocked()
	revision := s.permissionCatalogRevision
	client, generation := s.client, s.eventsGeneration
	if client == nil || ctx.Err() != nil {
		return nil
	}
	return func() {
		s.fire()
		var cat api.PermissionCatalog
		err := client.Call(ctx, "permissionPresets/catalog", map[string]any{}, &cat)
		s.mu.Lock()
		if s.client != client || s.eventsGeneration != generation || s.permissionCatalogRevision != revision || ctx.Err() != nil {
			s.mu.Unlock()
			return
		}
		var options []PermissionOption
		if err == nil {
			options = flattenPermissionCatalog(cat)
		}
		oldErr, newErr := "", ""
		if s.permissionsErr != nil {
			oldErr = s.permissionsErr.Error()
		}
		if err != nil {
			newErr = err.Error()
		}
		if oldErr != newErr || !slices.Equal(s.permissionOptions, options) {
			s.invalidateViewLocked()
		}
		s.permissionsErr, s.permissionOptions = err, options
		s.mu.Unlock()
		beforeSnapshotNotification("permission-catalog")
		s.fire()
	}
}

// SelectPermissionPreset switches the captured current session through the
// host's /permission command. The permissions projection is the only value
// committed to the picker; a command acknowledgement is not a projected value.
// Future-session defaults remain in the independent Settings namespace editor.
func (s *Store) SelectPermissionPreset(ctx context.Context, label string) {
	s.mu.Lock()
	opt, ok := permissionOptionByLabel(label, s.permissionOptions)
	id := s.selectedID
	s.mu.Unlock()
	if ok {
		s.SelectPermissionValueForSession(ctx, id, opt.Value)
	}
}

// SelectPermissionValueForSession accepts an opaque catalog value, never a caption.
// The captured visible session must still own the permission projection.
func (s *Store) SelectPermissionValueForSession(ctx context.Context, targetID, value string) {
	s.mu.Lock()
	opt, ok := permissionOptionByValue(value, s.permissionOptions)
	client, sessionID, generation := s.client, s.selectedID, s.followGeneration
	if sessionID != targetID {
		s.mu.Unlock()
		return
	}
	if !ok || client == nil || sessionID == "" || ctx.Err() != nil ||
		!s.permissionProjectionCurrentLocked() || s.permissionProjection.Pending ||
		opt.Value == s.permissionProjection.Value {
		s.mu.Unlock()
		return
	}
	s.resetPermissionRiskLocked()
	if permissionNeedsConfirmation(opt.Value) {
		s.permissionRisk = &permissionRiskOperation{view: PermissionRiskConfirmation{Token: s.permissionRiskToken, SessionID: sessionID, Label: opt.Label, Value: opt.Value}, client: client, generation: generation}
		s.mu.Unlock()
		s.fire()
		return
	}
	s.permissionProjection.Pending = true
	s.permissionsErr = nil
	s.mu.Unlock()
	s.fire()
	go s.runPermissionPreset(ctx, client, sessionID, generation, opt)
}

func (s *Store) runPermissionPreset(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, opt PermissionOption) {
	_ = s.settlePermissionPreset(ctx, client, sessionID, generation, opt, nil)
}

func (s *Store) settlePermissionPreset(ctx context.Context, client *dsh.Client, sessionID string, generation uint64, opt PermissionOption, owner *NativeCommandOwner) error {
	s.mu.Lock()
	current := s.sessionResultCurrentLocked(ctx, client, sessionID, generation) && (owner == nil || s.nativeCommandCurrentLocked(*owner, "permission"))
	s.mu.Unlock()
	var err error
	var out api.SessionProjectionsValue
	if current {
		err = executeCommand(ctx, client, sessionID, "/permission "+opt.Value, nil)
		s.mu.Lock()
		current = s.sessionResultCurrentLocked(ctx, client, sessionID, generation) && (owner == nil || s.nativeCommandCurrentLocked(*owner, "permission"))
		s.mu.Unlock()
		if err == nil && current {
			err = client.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": sessionID}}, &out)
		}
	}
	s.mu.Lock()
	if s.client != client || s.selectedID != sessionID || s.followGeneration != generation {
		s.mu.Unlock()
		return fmt.Errorf("Permission selection owner changed")
	}
	wasPending := s.permissionProjection.Pending
	s.permissionProjection.Pending = false
	if wasPending {
		s.invalidateViewLocked()
	}
	if !current || ctx.Err() != nil || (owner != nil && !s.nativeCommandCurrentLocked(*owner, "permission")) {
		s.mu.Unlock()
		s.fire()
		return fmt.Errorf("Permission selection owner changed")
	}
	if err == nil {
		s.applyPermissionValuesLocked(out.Values, out.AsOfSeq)
		if !s.permissionProjectionCurrentLocked() || s.permissionProjection.Value != opt.Value {
			err = errors.New(storelocale.T(storelocale.KeyStoreSessionTheHostDidNotConfirmTheCurrentSession))
		}
	}
	oldErr, newErr := "", ""
	if s.permissionsErr != nil {
		oldErr = s.permissionsErr.Error()
	}
	if err != nil {
		newErr = err.Error()
	}
	action := storelocale.T(storelocale.KeyStoreSessionCurrentSessionPermissionUpdated) + s.permissionLabelLocked()
	if err != nil {
		action = newErr
	}
	if oldErr != newErr || s.lastAction != action {
		s.invalidateViewLocked()
	}
	s.permissionsErr = err
	s.lastAction = action
	s.mu.Unlock()
	beforeSnapshotNotification("permission-settled")
	s.fire()
	return err
}

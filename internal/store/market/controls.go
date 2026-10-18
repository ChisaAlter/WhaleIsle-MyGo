package market

import (
	"fmt"

	"whaleisle/internal/dsh/api"
)

// CanToggle follows the Host's bundle-level restrictions. Package names must
// never be retried as per-entry identifiers when an operation fails.
func (e Entry) CanToggle() bool {
	return e.IdentityKnown && e.Enabled != nil && e.ReadOnlyReason == "" && (*e.Enabled || e.Error == "")
}

func bundleError(err *api.ChangeResultError) string {
	if err == nil {
		return ""
	}
	if err.Diagnostic != nil && *err.Diagnostic != "" {
		return err.Code + ": " + *err.Diagnostic
	}
	return err.Code
}

// ValidateBundleChange rechecks the inventory after a UI click. Cached,
// disconnected or failed refreshes must not enable stale destructive controls.
func (s *Store) ValidateBundleChange(name string, removing bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.loading || s.err != nil {
		return fmt.Errorf("plugin inventory is unavailable; refresh and try again")
	}
	for _, row := range s.rows {
		if row.Label != name {
			continue
		}
		if !row.IdentityKnown || row.Enabled == nil {
			return fmt.Errorf("plugin bundle identity is unknown; refresh and try again")
		}
		if row.ReadOnlyReason != "" {
			return fmt.Errorf("%s: %s", name, row.ReadOnlyReason)
		}
		if removing {
			if !row.Removable {
				return fmt.Errorf("%s cannot be removed", name)
			}
		} else if !row.CanToggle() {
			return fmt.Errorf("%s: %s", name, row.Error)
		}
		return nil
	}
	return fmt.Errorf("plugin bundle is no longer in the inventory: %s", name)
}

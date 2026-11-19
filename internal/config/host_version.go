package config

import "strings"

// ReconcileHostVersion keeps the stored npm fallback version naming the Host
// runtime this build actually ships.
//
// The field only feeds a launch that falls back to npm, and a value left over
// from an older build would ask npm for a Host this app was not built against.
// Nobody should have to know the field exists to get a working start, so the
// shell corrects it instead of reporting it.
func (s *Store) ReconcileHostVersion(installed string) error {
	installed = strings.TrimSpace(installed)
	if installed == "" {
		return nil
	}
	if strings.TrimSpace(s.Get().DshVersion) == installed {
		return nil
	}
	return s.Update(func(c *Config) { c.DshVersion = installed })
}

//go:build windows && inspectorfixture

package session

import "encoding/json"

// ApplyInspectorFixtureFrame drives the normal follow parser in an opt-in local
// desktop fixture. It is absent from production builds and opens no connection.
func (s *Store) ApplyInspectorFixtureFrame(raw json.RawMessage) { s.applyFollowFrame(raw) }

package session

import (
	"context"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
)

// RenderOfficePDF calls officeToPdf/render for a workspace-relative path and
// returns the AbsolutePath of the generated PDF when present.
func (s *Store) RenderOfficePDF(ctx context.Context, relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "", fmt.Errorf("empty path")
	}
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil {
		return "", fmt.Errorf("harness not ready")
	}
	if sessionID == "" {
		return "", fmt.Errorf("no session selected")
	}
	// Electron probes officeToPdf/generation before render (cache key / engine bump).
	var gen api.OfficeToPdfGeneration
	_ = client.Call(ctx, "officeToPdf/generation", map[string]any{}, &gen)
	var out api.RenderedDocumentBytes
	err := client.Call(ctx, "officeToPdf/render", map[string]any{
		"workspaceFileScopeId": sessionID,
		"path":                 relPath,
		"priority":             "interactive",
	}, &out)
	if err != nil {
		return "", err
	}
	abs := strings.TrimSpace(out.AbsolutePath)
	if abs == "" {
		return "", fmt.Errorf("officeToPdf/render: empty absolutePath")
	}
	if g := strings.TrimSpace(string(gen)); g != "" {
		s.mu.Lock()
		s.lastAction = "office pdf " + g
		s.mu.Unlock()
		s.fire()
	}
	return abs, nil
}

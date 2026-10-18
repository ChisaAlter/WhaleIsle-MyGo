package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"whaleisle/internal/dsh/api"
)

// OfficePDFDocument contains the converted PDF and the original Office identity.
// SourcePath belongs to the Host; it is not a local path to the generated PDF.
type OfficePDFDocument struct {
	Data         []byte
	MissingFonts []string
	SourcePath   string
	Version      string
	Generation   string
}

// RenderOfficePDF returns the source identity reported by officeToPdf/render.
// Preview callers must use RenderOfficePDFBytesForSession for the PDF content.
func (s *Store) RenderOfficePDF(ctx context.Context, relPath string) (string, error) {
	s.mu.Lock()
	sid := s.selectedID
	s.mu.Unlock()
	return s.RenderOfficePDFForSession(ctx, sid, relPath)
}

// RenderOfficePDFForSession retains the path-only metadata API. The Host returns
// the original Office path here, never a temporary PDF path.
func (s *Store) RenderOfficePDFForSession(ctx context.Context, sessionID, relPath string) (string, error) {
	out, err := s.renderOfficePDFForSession(ctx, sessionID, relPath)
	if err != nil {
		return "", err
	}
	return out.AbsolutePath, nil
}

// RenderOfficePDFBytesForSession decodes the existing multipart binary result.
// Conversion and source authorization stay on the selected Session's Host.
func (s *Store) RenderOfficePDFBytesForSession(ctx context.Context, sessionID, path string) (*OfficePDFDocument, error) {
	out, err := s.renderOfficePDFForSession(ctx, sessionID, path)
	if err != nil {
		return nil, err
	}
	var data []byte
	if err := json.Unmarshal(out.Data, &data); err != nil {
		return nil, fmt.Errorf("Office conversion returned unreadable PDF bytes: %w", err)
	}
	if len(data) == 0 || out.Offset != 0 || !out.Eof {
		return nil, fmt.Errorf("Office conversion returned no complete PDF")
	}
	if out.Bytes != nil && *out.Bytes != float64(len(data)) {
		return nil, fmt.Errorf("Office conversion returned an incomplete PDF")
	}
	return &OfficePDFDocument{
		Data: data, MissingFonts: out.MissingFonts, SourcePath: out.AbsolutePath,
		Version: out.Version, Generation: out.Generation,
	}, nil
}

func (s *Store) renderOfficePDFForSession(ctx context.Context, sessionID, path string) (api.RenderedDocumentBytes, error) {
	var out api.RenderedDocumentBytes
	path = strings.TrimSpace(path)
	if path == "" {
		return out, fmt.Errorf("empty path")
	}
	s.mu.Lock()
	client := s.client
	same := sessionID != "" && s.selectedID == sessionID
	s.mu.Unlock()
	if !same {
		return out, fmt.Errorf("Office conversion session changed")
	}
	if client == nil {
		return out, fmt.Errorf("harness not ready")
	}
	// Preserve the provider-generation probe. Render carries its own generation
	// and remains authoritative when the Host has no cached generation response.
	var gen api.OfficeToPdfGeneration
	_ = client.Call(ctx, "officeToPdf/generation", map[string]any{}, &gen)
	err := client.Call(ctx, "officeToPdf/render", map[string]any{
		"workspaceFileScopeId": sessionID,
		"path":                 path,
		"priority":             "foreground",
	}, &out)
	if err != nil {
		return out, err
	}
	if strings.TrimSpace(out.AbsolutePath) == "" {
		return out, fmt.Errorf("officeToPdf/render: empty source absolutePath")
	}
	return out, nil
}

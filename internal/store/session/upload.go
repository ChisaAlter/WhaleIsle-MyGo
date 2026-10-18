package session

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"

	"whaleisle/internal/dsh/api"
)

// UploadEncodedFile posts fileUploads/upload for the selected session.
// Returns the attachmentId on success.
func (s *Store) UploadEncodedFile(ctx context.Context, absPath string) (string, error) {
	absPath = strings.TrimSpace(absPath)
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" || absPath == "" {
		return "", os.ErrInvalid
	}
	raw, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	name := filepath.Base(absPath)
	req := api.EncodedFileUploadRequest{
		Data: base64.StdEncoding.EncodeToString(raw),
		Name: &name,
	}
	var out api.FileUploadValue
	err = client.Call(ctx, "fileUploads/upload", map[string]any{
		"agentId": sessionID,
		"request": req,
	}, &out)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(out.File.AttachmentId)
	if id == "" {
		return "", os.ErrInvalid
	}
	s.mu.Lock()
	s.lastAction = "file uploaded " + id
	s.mu.Unlock()
	s.fire()
	return id, nil
}

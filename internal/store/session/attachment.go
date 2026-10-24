package session

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

var attachmentCache struct {
	mu    sync.Mutex
	dir   string
	paths map[string]string // attachmentId -> local path
}

// SetAttachmentCacheDir installs the directory used for session/attachment blobs.
func SetAttachmentCacheDir(dir string) {
	attachmentCache.mu.Lock()
	attachmentCache.dir = strings.TrimSpace(dir)
	if attachmentCache.paths == nil {
		attachmentCache.paths = map[string]string{}
	}
	attachmentCache.mu.Unlock()
}

// CachedAttachmentPath returns a previously fetched local path for attachmentId.
func CachedAttachmentPath(attachmentID string) string {
	if attachmentID == "" {
		return ""
	}
	attachmentCache.mu.Lock()
	defer attachmentCache.mu.Unlock()
	return attachmentCache.paths[attachmentID]
}

// FetchAttachment calls session/attachment with the exact provider-owned ID and
// writes decoded bytes to the cache dir. Whitespace is part of an opaque ID.
func (s *Store) FetchAttachment(ctx context.Context, attachmentID string) (string, error) {
	if attachmentID == "" {
		return "", os.ErrInvalid
	}
	if local := CachedAttachmentPath(attachmentID); local != "" {
		if _, err := os.Stat(local); err == nil {
			return local, nil
		}
	}
	s.mu.Lock()
	client := s.client
	sessionID := s.selectedID
	s.mu.Unlock()
	if client == nil || sessionID == "" {
		return "", ErrNoClient
	}
	return fetchSessionAttachment(ctx, client, sessionID, attachmentID, attachmentID)
}

// fetchSessionAttachment shares decoding and storage while callers choose the cache namespace.
func fetchSessionAttachment(ctx context.Context, client *dsh.Client, sessionID, attachmentID, identity string) (string, error) {
	var out api.SessionAttachmentValue
	err := client.Call(ctx, "session/attachment", map[string]any{
		"request": api.SessionAttachmentRequest{
			SessionId:    sessionID,
			AttachmentId: attachmentID,
		},
	}, &out)
	if err != nil {
		return "", err
	}
	raw, err := decodeAttachmentData(out.Data)
	if err != nil {
		return "", err
	}
	attachmentCache.mu.Lock()
	dir := attachmentCache.dir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "whaleisle-attachments")
	}
	attachmentCache.mu.Unlock()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ext := ".bin"
	if mt := strings.ToLower(out.Attachment.MediaType); strings.Contains(mt, "png") {
		ext = ".png"
	} else if strings.Contains(mt, "jpeg") || strings.Contains(mt, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(mt, "gif") {
		ext = ".gif"
	} else if strings.Contains(mt, "webp") {
		ext = ".webp"
	} else if name := out.Attachment.Name; name != nil && filepath.Ext(*name) != "" {
		ext = filepath.Ext(*name)
	}
	// IDs are provider-owned and opaque; never interpret them as local paths.
	cacheKey := sha256.Sum256([]byte(sessionID + "\x00" + identity))
	local := filepath.Join(dir, fmt.Sprintf("%x", cacheKey)+ext)
	if err := os.WriteFile(local, raw, 0o644); err != nil {
		return "", err
	}
	attachmentCache.mu.Lock()
	if attachmentCache.paths == nil {
		attachmentCache.paths = map[string]string{}
	}
	attachmentCache.paths[identity] = local
	attachmentCache.mu.Unlock()
	return local, nil
}

func decodeAttachmentData(data string) ([]byte, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, os.ErrInvalid
	}
	if i := strings.Index(data, ","); strings.HasPrefix(data, "data:") && i > 0 {
		data = data[i+1:]
	}
	return base64.StdEncoding.DecodeString(data)
}

// ForgetCachedAttachment allows an explicit retry after missing/corrupt cache
// content. The next read still goes through the session-authorized endpoint.
func ForgetCachedAttachment(attachmentID string) {
	attachmentCache.mu.Lock()
	delete(attachmentCache.paths, attachmentID)
	attachmentCache.mu.Unlock()
}

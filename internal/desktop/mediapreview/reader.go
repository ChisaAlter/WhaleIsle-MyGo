// Package mediapreview serves a bounded immutable media snapshot, never a
// filesystem directory, Host proxy, arbitrary URL or attachment-ID cache.
package mediapreview

import (
	"context"
	"errors"
	"math"
	"strings"
)

const MaxBytes = 32 << 20
const readWindow = 1 << 20

type Caller interface {
	Call(context.Context, string, any, any) error
}
type fileStat struct {
	AbsolutePath string   `json:"absolutePath"`
	Version      string   `json:"version"`
	Bytes        *float64 `json:"bytes"`
}
type fileBytes struct {
	fileStat
	Data   []byte  `json:"data"`
	Offset float64 `json:"offset"`
	Eof    bool    `json:"eof"`
}

// ValidPath accepts a single explicit Host path, never a URL or traversal. Host
// paths are opaque and are NEVER resolved, opened, or statted on this computer.
func ValidPath(p string) bool { return validFilePath(p) && Supports(p) }
func validFilePath(p string) bool {
	if p == "" || strings.TrimSpace(p) != p || strings.ContainsAny(p, "\x00\r\n") || strings.Contains(p, "://") || strings.HasPrefix(p, `\\`) || strings.HasPrefix(p, "//") {
		return false
	}
	q := strings.ReplaceAll(p, `\`, "/")
	for _, part := range strings.Split(q, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	// Only a leading Windows drive separator may contain a colon (no ADS/URLs).
	if len(q) > 2 && ((q[0] >= 'a' && q[0] <= 'z') || (q[0] >= 'A' && q[0] <= 'Z')) && q[1] == ':' && q[2] == '/' {
		q = q[2:]
	}
	return !strings.Contains(q, ":")
}
func (s fileStat) size() (int, error) {
	if s.AbsolutePath == "" || s.Version == "" || s.Bytes == nil || math.IsNaN(*s.Bytes) || math.IsInf(*s.Bytes, 0) || *s.Bytes < 1 || *s.Bytes > MaxBytes || math.Trunc(*s.Bytes) != *s.Bytes {
		return 0, errors.New("media requires a versioned complete file of 1 byte–32 MiB")
	}
	return int(*s.Bytes), nil
}
func (s fileStat) matches(other fileStat) bool {
	return s.AbsolutePath == other.AbsolutePath && s.Version == other.Version && s.Bytes != nil && other.Bytes != nil && *s.Bytes == *other.Bytes
}

// Read uses the existing session-scoped Remote contract. Each bounded range
// fits the existing 8 MiB RPC wire cap; no generic transport limit is enlarged.
// One immutable path/version/size is required across stat, every range and the
// final stat. Any unsupported Host/backend, cancellation or mismatch fails
// closed, with no local-path or unscoped attachment-cache fallback.
func Read(ctx context.Context, client Caller, sid, path string, valid func() bool) ([]byte, error) {
	if !ValidPath(path) {
		return nil, errors.New("invalid media path")
	}
	data, err := ReadSnapshot(ctx, client, sid, path, MaxBytes, valid)
	if err != nil {
		return nil, err
	}
	if _, _, err = Validate(path, data); err != nil {
		return nil, err
	}
	return data, nil
}

// ReadSnapshot reuses the media transport for a selected document's real bytes.
// The caller chooses its existing document size limit and validates its format.
func ReadSnapshot(ctx context.Context, client Caller, sid, path string, limit int, valid func() bool) ([]byte, error) {
	current := func() bool { return ctx.Err() == nil && valid != nil && valid() }
	if client == nil || sid == "" || !validFilePath(path) || !current() {
		return nil, errors.New("media request has no current Host/session or valid file")
	}
	args := map[string]any{"workspaceFileScopeId": sid, "path": path}
	var first fileStat
	if err := client.Call(ctx, "workspaceFiles/stat", args, &first); err != nil {
		return nil, err
	}
	size, err := first.size()
	if err != nil {
		return nil, err
	}
	if size > limit {
		return nil, errors.New("document exceeds its preview size limit")
	}
	data := make([]byte, 0, size)
	for offset := 0; offset < size; {
		if !current() {
			return nil, errors.New("media Host/session changed")
		}
		length := min(readWindow, size-offset)
		var part fileBytes
		req := map[string]any{"workspaceFileScopeId": sid, "path": path, "options": map[string]any{"range": map[string]any{"offset": offset, "length": length}}}
		if err = client.Call(ctx, "workspaceFiles/readBytes", req, &part); err != nil {
			return nil, err
		}
		if !current() || !first.matches(part.fileStat) || part.Offset != float64(offset) || len(part.Data) != length || part.Eof != (offset+length == size) {
			return nil, errors.New("media path, version, size or byte range changed; reopen the file")
		}
		data = append(data, part.Data...)
		offset += length
	}
	var last fileStat
	if err = client.Call(ctx, "workspaceFiles/stat", args, &last); err != nil {
		return nil, err
	}
	if !current() || !first.matches(last) {
		return nil, errors.New("media changed while reading; reopen the file")
	}
	return data, nil
}

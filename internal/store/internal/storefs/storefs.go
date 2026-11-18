// Package storefs provides atomic file writes and corrupt-JSON preservation
// shared across local-persistence stores under internal/store.
package storefs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WriteFileAtomic writes data to a unique temp file next to path, flushes it
// to disk, and atomically renames it into place. It retries os.Rename briefly
// for transient Windows share locks and never falls back to truncating path.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	var renameErr error
	for attempt := 0; attempt < 8; attempt++ {
		renameErr = os.Rename(tmpName, path)
		if renameErr == nil {
			cleanup = false
			return nil
		}
		// Windows search indexers / AV scanners can briefly lock newly closed files.
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return renameErr
}

// ReadJSON reads and decodes path into dst. Missing files return (false, nil).
// Read failures or invalid JSON fail closed without modifying path; decode
// failures also save a best-effort .corrupt backup copy alongside path.
func ReadJSON[T any](storeName, path string, dst *T, allowEmpty bool) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("%s: read %s: %w", storeName, filepath.Base(path), err)
	}
	if allowEmpty && len(bytes.TrimSpace(raw)) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return false, PreserveCorrupt(storeName, path, raw, err)
	}
	return true, nil
}

// PreserveCorrupt keeps path untouched, saves a best-effort timestamped
// .corrupt backup beside it, and returns a contextual parse error.
func PreserveCorrupt(storeName, path string, raw []byte, parseErr error) error {
	ts := time.Now().UTC().Format("20060102T150405.000000000")
	backupPath := fmt.Sprintf("%s.corrupt-%s", path, ts)
	for n := 1; ; n++ {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = fmt.Sprintf("%s.corrupt-%s-%d", path, ts, n)
	}
	_ = WriteFileAtomic(backupPath, raw, 0o644)
	return fmt.Errorf("%s: parse %s: %w", storeName, filepath.Base(path), parseErr)
}

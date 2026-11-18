package credentials

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const lockFileName = ".credentials.lock"

// ErrSharedProfileBusy leaves the edit recoverable while another desktop owns
// Chromium's Windows profile lock. It never prevents read-only profile use.
var ErrSharedProfileBusy = errors.New("共享凭据正由另一个桌面使用，请关闭原版桌面后重试保存；本页编辑已保留。 / Shared credentials are in use by another desktop. Close that desktop and retry saving; your edits are retained")

// ErrSharedProfileWriteUnavailable means this platform has no verified write
// lease shared with Electron. Closing a peer alone cannot establish that lease.
var ErrSharedProfileWriteUnavailable = errors.New("当前平台尚不支持安全写入与鲸屿原版共享的凭据。请在原版中保存后重新读取；本页编辑已保留。 / Safe shared credential writes are not available on this platform. Save in the original desktop, then reload here; your edits are retained")

var (
	dirMutexes         sync.Map // cleaned dir path -> *sync.Mutex
	lockAcquireTimeout = 1500 * time.Millisecond
	staleLockAge       = 30 * time.Second
)

type lockMetadata struct {
	PID       int    `json:"pid"`
	Token     string `json:"token"`
	CreatedAt int64  `json:"createdAt"`
}

func dirLock(dir string) *sync.Mutex {
	cleaned := filepath.Clean(dir)
	if v, ok := dirMutexes.Load(cleaned); ok {
		return v.(*sync.Mutex)
	}
	mu := &sync.Mutex{}
	actual, _ := dirMutexes.LoadOrStore(cleaned, mu)
	return actual.(*sync.Mutex)
}

// withDirLock acquires the in-process directory mutex and, when the directory
// exists (or createDir is true), acquires the cross-process .credentials.lock
// with bounded backoff retry and stale lock cleanup.
func withDirLock(dir string, createDir bool, fn func() error) error {
	return withProfileDirLock(dir, createDir, false, fn)
}

// shared marks the desktop profile, whose file may also be written by Electron.
// Private stores retain their existing participating-writer lock contract.
func withProfileDirLock(dir string, createDir, shared bool, fn func() error) error {
	mu := dirLock(dir)
	mu.Lock()
	defer mu.Unlock()

	if !createDir {
		if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
			return fn()
		}
		_, lockErr := os.Stat(filepath.Join(dir, lockFileName))
		_, fileErr := os.Stat(filepath.Join(dir, FileName))
		if errors.Is(lockErr, fs.ErrNotExist) && errors.Is(fileErr, fs.ErrNotExist) {
			return fn()
		}
	} else {
		if shared {
			if err := sharedProfileWriteSupport(); err != nil {
				return err
			}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		acquire := acquirePrivateProfileWrite
		if shared {
			acquire = acquireProfileWrite
		}
		releaseProfile, err := acquire(dir)
		if err != nil {
			return err
		}
		defer releaseProfile()
	}

	release, err := acquireCredentialsLock(dir)
	if err != nil {
		return err
	}
	defer release()
	return fn()
}

func acquireCredentialsLock(dir string) (func(), error) {
	lockPath := filepath.Join(dir, lockFileName)
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	deadline := time.Now().Add(lockAcquireTimeout)
	backoff := 2 * time.Millisecond

	for attempt := 0; ; attempt++ {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			meta, marshalErr := json.Marshal(lockMetadata{
				PID:       os.Getpid(),
				Token:     token,
				CreatedAt: time.Now().UnixMilli(),
			})
			if marshalErr == nil {
				_, _ = f.Write(append(meta, '\n'))
				_ = f.Sync()
			}
			if closeErr := f.Close(); closeErr != nil {
				_ = os.Remove(lockPath)
				return nil, closeErr
			}
			return func() {
				if current, readErr := os.ReadFile(lockPath); readErr == nil {
					var cur lockMetadata
					if json.Unmarshal(current, &cur) == nil && cur.Token == token {
						_ = os.Remove(lockPath)
					}
				}
			}, nil
		}
		exists := errors.Is(err, fs.ErrExist)
		if !exists && !lockCreationContended(err) {
			return nil, fmt.Errorf("credentials: acquire lock: %w", err)
		}

		if exists && tryCleanupStaleLock(lockPath) {
			continue
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			if !exists {
				return nil, fmt.Errorf("credentials: acquire lock: %w", err)
			}
			return nil, fmt.Errorf("credentials: lock busy (%s)", lockPath)
		}
		wait := jitteredDuration(backoff)
		if wait > remaining {
			wait = remaining
		}
		time.Sleep(wait)
		if backoff < 40*time.Millisecond {
			backoff *= 2
			if backoff > 40*time.Millisecond {
				backoff = 40 * time.Millisecond
			}
		}
		_ = attempt
	}
}

func tryCleanupStaleLock(lockPath string) bool {
	info, statErr := os.Stat(lockPath)
	if statErr != nil {
		return false
	}
	raw, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		return false
	}
	if !isLockMetadataStale(raw, info.ModTime(), time.Now()) {
		return false
	}
	// Re-check bytes immediately before removal to narrow concurrent cleanup races.
	confirm, err := os.ReadFile(lockPath)
	if err != nil || !bytes.Equal(raw, confirm) {
		return false
	}
	return os.Remove(lockPath) == nil
}

func isLockMetadataStale(raw []byte, modTime, now time.Time) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return now.Sub(modTime) > 2*time.Second
	}
	var meta lockMetadata
	if err := json.Unmarshal(trimmed, &meta); err != nil {
		return now.Sub(modTime) > staleLockAge
	}
	if meta.PID > 0 && !isProcessAlive(meta.PID) {
		return true
	}
	if meta.CreatedAt > 0 && now.Sub(time.UnixMilli(meta.CreatedAt)) > staleLockAge {
		return true
	}
	if meta.PID <= 0 && meta.CreatedAt <= 0 {
		return now.Sub(modTime) > staleLockAge
	}
	return false
}

func jitteredDuration(base time.Duration) time.Duration {
	if base <= 0 {
		return time.Millisecond
	}
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		return base
	}
	fraction := float64(binary.LittleEndian.Uint16(b[:])) / 65535.0
	extra := time.Duration(float64(base) * 0.5 * fraction)
	return base + extra
}

package config

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrConflict reports that config.json changed on disk since this Store last
// read or wrote it. Update refuses to overwrite in that case: the caller must
// reopen the store and deliberately redo the change.
var ErrConflict = errors.New("config: configuration changed on disk since it was read")

// commitMu serializes the final compare-and-replace step across every Store in
// this process. It does not coordinate with other processes (for example the
// Electron shell), which remain a separate, unverified gate.
var commitMu sync.Mutex

// fileBaseline is the exact on-disk state a Store last observed. It is
// compared byte-for-byte so permissive reparsing or timestamps cannot mask an
// external edit.
type fileBaseline struct {
	exists bool
	data   []byte
}

func readBaseline(path string) (fileBaseline, error) {
	var data []byte
	var err error
	for attempt := 0; attempt < 8; attempt++ {
		data, err = os.ReadFile(path)
		if err == nil {
			return fileBaseline{exists: true, data: data}, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			return fileBaseline{}, nil
		}
		// Windows can briefly share-lock the file during a concurrent rename.
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return fileBaseline{}, err
}

// matches reports whether the file on disk is still identical to the baseline.
// A read error other than "not exist" is returned so callers stop instead of
// guessing.
func (b fileBaseline) matches(path string) (bool, error) {
	cur, err := readBaseline(path)
	if err != nil {
		return false, err
	}
	return cur.exists == b.exists && bytes.Equal(cur.data, b.data), nil
}

// writeAtomic writes data to a temporary file next to path and renames it into
// place. rename is injectable so tests can force a deterministic failure.
func writeAtomic(path string, data []byte, rename func(oldpath, newpath string) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if rename == nil {
		rename = os.Rename
	}
	var errRename error
	for attempt := 0; attempt < 8; attempt++ {
		errRename = rename(tmp.Name(), path)
		if errRename == nil {
			return nil
		}
		time.Sleep(time.Duration(attempt+1) * 5 * time.Millisecond)
	}
	return errRename
}

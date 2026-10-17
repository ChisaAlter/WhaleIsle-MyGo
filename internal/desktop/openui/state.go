package openui

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var stateMu sync.Mutex

func statePath(root, id string) (string, error) {
	if len(id) != 64 {
		return "", errors.New("invalid OpenUI state key")
	}
	if _, e := hex.DecodeString(id); e != nil {
		return "", e
	}
	return filepath.Join(root, id+".json"), nil
}
func ReadState(root, id string) (json.RawMessage, error) {
	path, e := statePath(root, id)
	if e != nil {
		return nil, e
	}
	f, e := os.Open(path)
	if os.IsNotExist(e) {
		return json.RawMessage(`{}`), nil
	}
	if e != nil {
		return nil, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, MaxStateBytes+1))
	if e != nil {
		return nil, e
	}
	if e = ValidateState(data); e != nil {
		return nil, e
	}
	return json.RawMessage(data), nil
}
func SaveState(root, id string, state json.RawMessage) error {
	if e := ValidateState(state); e != nil {
		return e
	}
	path, e := statePath(root, id)
	if e != nil {
		return e
	}
	stateMu.Lock()
	defer stateMu.Unlock()
	if e = os.MkdirAll(root, 0700); e != nil {
		return e
	}
	entries, e := os.ReadDir(root)
	if e != nil {
		return e
	}
	if _, e = os.Stat(path); os.IsNotExist(e) && len(entries) >= 256 {
		return errors.New("OpenUI state storage limit reached")
	}
	f, e := os.CreateTemp(root, ".state-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(state); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}

package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"whaleisle/internal/dsh/api"
)

// ThemeBinding follows the original ui-theme service's 300ms continuous-control
// queue over the one served namespace. It contains no desktop config mirror.
type ThemeBinding struct {
	*NamespaceBinding
	pending         map[string]json.RawMessage
	inFlightFields  map[string]json.RawMessage
	timer           *time.Timer
	ready, inFlight bool
	queueRevision   uint64
	writeError      string
}

func (t *ThemeBinding) Snapshot() ViewState {
	view := t.NamespaceBinding.Snapshot()
	t.store.mu.Lock()
	defer t.store.mu.Unlock()
	if t.currentLocked() && t.writeError != "" {
		view.Err = t.writeError
	}
	return view
}

func (s *Store) BindTheme() *ThemeBinding {
	b := s.BindNamespace("ui-theme")
	if b == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.themeBinding == nil || s.themeBinding.client != b.client || s.themeBinding.generation != b.generation {
		s.themeBinding = &ThemeBinding{NamespaceBinding: b, pending: map[string]json.RawMessage{}}
	}
	return s.themeBinding
}

func (t *ThemeBinding) Pending() bool {
	s := t.store
	s.mu.Lock()
	defer s.mu.Unlock()
	return t.currentLocked() && (len(t.pending) > 0 || t.inFlight)
}

func (t *ThemeBinding) Flush() {
	s := t.store
	s.mu.Lock()
	if !t.currentLocked() {
		s.mu.Unlock()
		return
	}
	if t.timer != nil {
		t.timer.Stop()
	}
	t.ready = true
	s.mu.Unlock()
	s.fire()
}

func (t *ThemeBinding) EffectiveValue() json.RawMessage {
	s := t.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !t.currentLocked() {
		return nil
	}
	if s.findNamespaceLocked("ui-theme") == nil {
		return nil
	}
	var values map[string]json.RawMessage
	_ = json.Unmarshal(s.findNamespaceLocked("ui-theme").Value, &values)
	if values == nil {
		values = map[string]json.RawMessage{}
	}
	for key, value := range t.inFlightFields {
		values[key] = value
	}
	for key, value := range t.pending {
		values[key] = value
	}
	data, _ := json.Marshal(values)
	return data
}

// Queue updates only explicitly edited, actually served fields. Host schema
// authority determines availability; no offline values are copied into it.
func (t *ThemeBinding) Queue(values map[string]any) error {
	s := t.store
	s.mu.Lock()
	if !t.currentLocked() || s.client == nil || s.describedClient != s.client || !s.writable {
		s.mu.Unlock()
		return fmt.Errorf("Host appearance is unavailable or read-only")
	}
	ns := s.findNamespaceLocked("ui-theme")
	known := map[string]bool{}
	for _, field := range ns.Fields {
		known[field.Key] = true
	}
	for key := range values {
		if !known[key] {
			s.mu.Unlock()
			return fmt.Errorf("Host does not expose appearance field %s", key)
		}
	}
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			s.mu.Unlock()
			return err
		}
		t.pending[key] = raw
	}
	t.ready = false
	t.queueRevision++
	queuedRevision := t.queueRevision
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = time.AfterFunc(300*time.Millisecond, func() {
		s.mu.Lock()
		if !t.currentLocked() || t.queueRevision != queuedRevision {
			s.mu.Unlock()
			return
		}
		t.ready = true
		s.mu.Unlock()
		s.fire()
	})
	s.mu.Unlock()
	s.fire()
	return nil
}

func (s *Store) flushQueuedTheme() {
	s.mu.Lock()
	t := s.themeBinding
	if t == nil || !t.currentLocked() || !t.ready || t.inFlight || len(t.pending) == 0 || s.saving || s.loading || !s.writable {
		s.mu.Unlock()
		return
	}
	ns := s.findNamespaceLocked("ui-theme")
	revision := ns.Revision
	fields := t.pending
	t.pending = map[string]json.RawMessage{}
	t.ready = false
	t.inFlight = true
	t.inFlightFields = fields
	t.writeError = ""
	s.saving, s.err = true, nil
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ops := []json.RawMessage{}
	for _, key := range keys {
		ops = append(ops, fieldOperation("set", []string{key}, fields[key]))
	}
	s.mu.Unlock()
	go func() {
		var out api.SettingsNamespaceView
		err := t.client.Call(context.Background(), "settings/mutate", map[string]any{"ns": "ui-theme", "ops": ops, "expectedRevision": revision}, &out)
		s.mu.Lock()
		if s.client != t.client || s.generation != t.generation {
			s.mu.Unlock()
			return
		}
		t.inFlight, s.saving = false, false
		t.inFlightFields = nil
		if s.findNamespaceLocked("ui-theme") == nil {
			s.err = fmt.Errorf("Host appearance is no longer available")
			s.mu.Unlock()
			s.fire()
			return
		}
		if err == nil {
			current := s.findNamespaceLocked("ui-theme")
			current.Revision = out.Revision
			current.Value = append(json.RawMessage(nil), out.Value...)
			current.Base = append(json.RawMessage(nil), out.Base...)
			current.User = append(json.RawMessage(nil), out.User...)
			current.HasSecrets = hasSetSecrets(out.Secrets)
			current.SecretPaths = configuredSecretPaths(out.Secrets)
			s.seedDraftsLocked(*current)
			if !namespaceWriteLanded(out, nil, ops) {
				err = fmt.Errorf("Host did not accept the appearance values")
			}
		}
		if err != nil {
			t.writeError = err.Error()
			s.err, s.lastAction = err, err.Error()
		} else {
			s.err, s.lastAction = nil, "appearance saved"
		}
		s.mu.Unlock()
		if err != nil {
			s.RefreshDescribe(context.Background())
		} else {
			s.fire()
		}
	}()
}

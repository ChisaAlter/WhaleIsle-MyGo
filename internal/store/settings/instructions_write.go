package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

type instructionsWrite struct {
	client          *dsh.Client
	generation      uint64
	pending         *string
	ready, inFlight bool
	timer           *time.Timer
	queueRevision   uint64
}

// The existing conversation producer debounces custom instructions by 400ms.
// A newer edit joins the same owner queue while another settings write settles.
func (s *Store) queueInstructions(client *dsh.Client, generation uint64, text string) {
	s.mu.Lock()
	if s.client != client || s.generation != generation || s.describedClient != client || !s.writable || s.findNamespaceLocked(CustomInstructionsNamespace) == nil {
		s.mu.Unlock()
		return
	}
	if err := validateCustomInstructions(CustomInstructionsNamespace, map[string]string{CustomInstructionsField: text}); err != nil {
		s.err = err
		s.mu.Unlock()
		s.fire()
		return
	}
	w := s.instructionsWrite
	if w == nil || w.client != client || w.generation != generation {
		w = &instructionsWrite{client: client, generation: generation}
		s.instructionsWrite = w
	}
	w.pending = &text
	w.queueRevision++
	queuedRevision := w.queueRevision
	w.ready = false
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(400*time.Millisecond, func() {
		s.mu.Lock()
		if s.instructionsWrite != w || s.client != client || s.generation != generation || w.queueRevision != queuedRevision {
			s.mu.Unlock()
			return
		}
		w.ready = true
		s.mu.Unlock()
		s.fire()
	})
	s.mu.Unlock()
	s.fire()
}

func (s *Store) flushQueuedInstructions() {
	s.mu.Lock()
	w := s.instructionsWrite
	if w == nil || w.client != s.client || w.generation != s.generation || !w.ready || w.pending == nil || w.inFlight || s.saving || s.loading || !s.writable {
		s.mu.Unlock()
		return
	}
	ns := s.findNamespaceLocked(CustomInstructionsNamespace)
	if ns == nil {
		s.mu.Unlock()
		return
	}
	text, revision := *w.pending, ns.Revision
	w.pending, w.ready, w.inFlight = nil, false, true
	s.saving, s.err = true, nil
	raw, _ := json.Marshal(text)
	ops := []json.RawMessage{fieldOperation("set", []string{CustomInstructionsField}, raw)}
	s.mu.Unlock()
	go func() {
		var out api.SettingsNamespaceView
		err := w.client.Call(context.Background(), "settings/mutate", map[string]any{"ns": CustomInstructionsNamespace, "expectedRevision": revision, "ops": ops}, &out)
		s.mu.Lock()
		if s.client != w.client || s.generation != w.generation {
			s.mu.Unlock()
			return
		}
		w.inFlight, s.saving = false, false
		if current := s.findNamespaceLocked(CustomInstructionsNamespace); err == nil && current != nil {
			current.Revision = out.Revision
			current.Value = append(json.RawMessage(nil), out.Value...)
			current.Base = append(json.RawMessage(nil), out.Base...)
			current.User = append(json.RawMessage(nil), out.User...)
			s.seedDraftsLocked(*current)
			if !namespaceWriteLanded(out, nil, ops) {
				err = fmt.Errorf("Custom instructions were not accepted")
			}
		} else if err == nil {
			err = fmt.Errorf("Custom instructions settings are no longer available")
		}
		if err != nil {
			s.err, s.lastAction = err, err.Error()
		} else {
			s.err, s.lastAction = nil, "custom instructions saved"
		}
		s.mu.Unlock()
		if err != nil {
			s.RefreshDescribe(context.Background())
		} else {
			s.fire()
		}
	}()
}

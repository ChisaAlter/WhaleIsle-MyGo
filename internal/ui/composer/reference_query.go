package composer

import (
	"context"
	"strconv"
	"sync"
	"time"
	"whaleisle/internal/store/session"
)

// Reference discovery never runs on the renderer. Each draft owns one query;
// changing the token, Session or Host revokes the old receipt.
type referenceQueryState struct {
	mu      sync.Mutex
	key     string
	epoch   uint64
	cancel  context.CancelFunc
	pending bool
	err     string
	rows    []session.ReferenceCandidate
}

func (s *referenceQueryState) cancelQuery() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.key != "" {
		s.epoch++
		s.key = ""
		s.rows = nil
		s.pending = false
		s.err = ""
	}
}
func (s *referenceQueryState) query(m Model, q string, quoted bool) []session.ReferenceCandidate {
	key := strconv.FormatUint(m.Snapshot.ComposerHostGeneration, 10) + ":" + strconv.FormatUint(m.Snapshot.ComposerSessionGeneration, 10) + "\x00" + m.Snapshot.SelectedID + "\x00" + q
	if quoted {
		key += "\x00quoted"
	}
	s.mu.Lock()
	if s.key != key {
		if s.cancel != nil {
			s.cancel()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		s.key = key
		s.epoch++
		epoch := s.epoch
		s.cancel = cancel
		s.pending = true
		s.err = ""
		s.rows = nil
		go func() {
			rows, err := m.OnQueryReferences(ctx, q, quoted)
			cancel()
			s.mu.Lock()
			if s.epoch != epoch || s.key != key {
				s.mu.Unlock()
				return
			}
			s.pending = false
			s.cancel = nil
			s.rows = append([]session.ReferenceCandidate(nil), rows...)
			if err != nil {
				s.err = err.Error()
			}
			s.mu.Unlock()
			if m.referenceInvalidate != nil {
				m.referenceInvalidate()
			}
		}()
	}
	rows := append([]session.ReferenceCandidate(nil), s.rows...)
	s.mu.Unlock()
	return rows
}
func referenceQueryActive(m Model) bool { return m.OnQueryReferences != nil && m.State != nil }
func referenceQueryStatus(m Model) (bool, string) {
	if !referenceQueryActive(m) {
		return false, ""
	}
	s := &m.State.references
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending, s.err
}

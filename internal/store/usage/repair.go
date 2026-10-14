package usage

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

// RepairResult is the vendored usage-stats repair.session contract. Only the
// host touches artifacts; the native client never reads or rewrites log files.
type RepairResult struct {
	Repaired    int64  `json:"repaired"`
	Backup      string `json:"backup"`
	BytesBefore int64  `json:"bytesBefore"`
	BytesAfter  int64  `json:"bytesAfter"`
}
type RepairView struct {
	PendingID, SessionID, Err string
	Busy                      bool
	Result                    *RepairResult
}
type repairState struct {
	RepairView
	cancel    context.CancelFunc
	completed map[string]bool
}

func (s *Store) resetRepairLocked() {
	if s.repair.cancel != nil {
		s.repair.cancel()
	}
	s.repair = repairState{}
}
func (s *Store) Repair() RepairView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.repair.RepairView
	if v.Result != nil {
		copy := *v.Result
		v.Result = &copy
	}
	return v
}
func (s *Store) repairCandidateLocked(id string) bool {
	if id == "" || s.client == nil || s.stats.Loading || s.repair.Busy || s.repair.completed[id] {
		return false
	}
	var o Overview
	if json.Unmarshal(s.stats.Raw, &o) != nil {
		return false
	}
	for _, candidate := range o.Coverage.FailedSessionIDs {
		if candidate == id {
			return true
		}
	}
	return false
}
func (s *Store) CanRepair(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repairCandidateLocked(id)
}

// PrepareRepair only opens native confirmation. It never sends a request.
func (s *Store) PrepareRepair(id string) {
	s.mu.Lock()
	if !s.repairCandidateLocked(id) {
		s.mu.Unlock()
		return
	}
	s.repair.PendingID = id
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CancelRepair() {
	s.mu.Lock()
	s.repair.PendingID = ""
	s.mu.Unlock()
	s.fire()
}

// ConfirmRepair must follow the explicit native confirmation for this exact ID.
// A refresh, detach, changed selection, or duplicate click invalidates it.
func (s *Store) ConfirmRepair(ctx context.Context, id string) {
	s.mu.Lock()
	if s.repair.PendingID != id || !s.repairCandidateLocked(id) {
		s.mu.Unlock()
		return
	}
	ep, epoch := s.client.Endpoint, s.statsEpoch
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	s.repair.PendingID = ""
	s.repair.SessionID, s.repair.Err, s.repair.Result = id, "", nil
	s.repair.Busy, s.repair.cancel = true, cancel
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		rpc := &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie}
		var result RepairResult
		err := rpc.Call(ctx, "usage-stats", "repair.session", map[string]any{"sessionId": id}, &result)
		if err == nil && (result.Repaired <= 0 || strings.TrimSpace(result.Backup) == "" || result.BytesBefore <= 0 || result.BytesAfter <= 0) {
			err = errors.New("incomplete repair result")
		}
		s.mu.Lock()
		if epoch != s.statsEpoch {
			s.mu.Unlock()
			return
		}
		s.repair.Busy, s.repair.cancel = false, nil
		if err != nil {
			s.repair.Err = err.Error()
		} else {
			s.repair.Result = &result
			if s.repair.completed == nil {
				s.repair.completed = map[string]bool{}
			}
			s.repair.completed[id] = true
		}
		s.mu.Unlock()
		s.fire()
	}()
}

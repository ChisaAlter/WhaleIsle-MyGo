package usage

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
)

// RepairResult is the existing usage-stats repair.session contract. Only the
// host touches artifacts; this client never reads or rewrites log files.
type RepairResult struct {
	Repaired    int64  `json:"repaired"`
	Backup      string `json:"backup"`
	BytesBefore int64  `json:"bytesBefore"`
	BytesAfter  int64  `json:"bytesAfter"`
}
type RepairOutcome struct {
	SessionID string
	Result    *RepairResult
	Err       string
}
type RepairView struct {
	PendingID, SessionID, Err                                           string
	PendingIDs, Targets, Unattempted, RemainingFailed                   []string
	Busy, Batch, StopRequested, Rescanning, RescanComplete, RescanStale bool
	RescanErr                                                           string
	Result                                                              *RepairResult // Most recent confirmed result, retained for single-repair callers.
	Outcomes                                                            []RepairOutcome
}
type repairState struct {
	RepairView
	cancel       context.CancelFunc
	completed    map[string]bool
	generation   uint64
	pendingBatch bool
}

func (s *Store) resetRepairLocked() {
	if s.repair.cancel != nil {
		s.repair.cancel()
	}
	s.repair = repairState{generation: s.repair.generation + 1}
}
func (s *Store) clearRepairConfirmationLocked() {
	s.repair.PendingID = ""
	s.repair.PendingIDs = nil
	s.repair.pendingBatch = false
}
func (s *Store) Repair() RepairView {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.repair.RepairView
	v.PendingIDs = slices.Clone(v.PendingIDs)
	v.Targets = slices.Clone(v.Targets)
	v.Unattempted = slices.Clone(v.Unattempted)
	v.RemainingFailed = slices.Clone(v.RemainingFailed)
	clone := func(r *RepairResult) *RepairResult {
		if r == nil {
			return nil
		}
		c := *r
		return &c
	}
	v.Result = clone(v.Result)
	v.Outcomes = slices.Clone(v.Outcomes)
	for i := range v.Outcomes {
		v.Outcomes[i].Result = clone(v.Outcomes[i].Result)
	}
	return v
}
func (s *Store) repairCandidatesLocked() []string {
	if s.client == nil || s.stats.Loading || s.repair.Busy || s.repair.RescanErr != "" || s.repair.RescanStale {
		return nil
	}
	var o Overview
	if json.Unmarshal(s.stats.Raw, &o) != nil {
		return nil
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, id := range o.Coverage.FailedSessionIDs {
		if id != "" && !seen[id] && !s.repair.completed[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	return ids
}
func (s *Store) repairCandidateLocked(id string) bool {
	return slices.Contains(s.repairCandidatesLocked(), id)
}
func (s *Store) CanRepair(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repairCandidateLocked(id)
}
func (s *Store) RepairCandidates() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.repairCandidatesLocked()
}

// Prepare methods only open native confirmation for a snapshot of failed IDs.
func (s *Store) PrepareRepair(id string) {
	s.mu.Lock()
	if !s.repairCandidateLocked(id) {
		s.mu.Unlock()
		return
	}
	s.clearRepairConfirmationLocked()
	s.repair.PendingID = id
	s.repair.PendingIDs = []string{id}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) PrepareRepairAll() {
	s.mu.Lock()
	ids := s.repairCandidatesLocked()
	if len(ids) == 0 {
		s.mu.Unlock()
		return
	}
	s.clearRepairConfirmationLocked()
	s.repair.PendingIDs = ids
	s.repair.pendingBatch = true
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CancelRepair() {
	s.mu.Lock()
	s.clearRepairConfirmationLocked()
	s.mu.Unlock()
	s.fire()
}

// StopRepairQueue stops only unstarted requests. The in-flight RPC is allowed
// to report its actual result, then one read-only forced rescan verifies state.
// There is no host cancellation endpoint and no promise to undo a completed repair.
func (s *Store) StopRepairQueue() {
	s.mu.Lock()
	if s.repair.Busy && !s.repair.Rescanning {
		s.repair.StopRequested = true
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) ConfirmRepair(ctx context.Context, id string) {
	s.confirmRepairs(ctx, []string{id}, false)
}
func (s *Store) ConfirmRepairAll(ctx context.Context, ids []string) { s.confirmRepairs(ctx, ids, true) }
func (s *Store) confirmRepairs(ctx context.Context, ids []string, batch bool) {
	s.mu.Lock()
	if len(ids) == 0 || s.repair.pendingBatch != batch || !slices.Equal(s.repair.PendingIDs, ids) || (!batch && s.repair.PendingID != ids[0]) {
		s.mu.Unlock()
		return
	}
	for _, id := range ids {
		if !s.repairCandidateLocked(id) {
			s.clearRepairConfirmationLocked()
			s.mu.Unlock()
			s.fire()
			return
		}
	}
	client, epoch := s.client, s.statsEpoch
	generation := s.repair.generation + 1
	completed := s.repair.completed
	ctx, cancel := context.WithCancel(ctx)
	targets := slices.Clone(ids)
	s.repair = repairState{RepairView: RepairView{Busy: true, Batch: batch, Targets: targets, Unattempted: slices.Clone(targets)}, cancel: cancel, completed: completed, generation: generation}
	s.mu.Unlock()
	s.fire()
	go s.runRepairs(ctx, cancel, client, epoch, generation, targets)
}
func (s *Store) repairCurrentLocked(client *dsh.Client, epoch, generation uint64) bool {
	return s.client == client && s.statsEpoch == epoch && s.repair.generation == generation
}
func decodeRepairResult(raw json.RawMessage) (RepairResult, error) {
	var wire struct {
		Repaired    *int64  `json:"repaired"`
		Backup      *string `json:"backup"`
		BytesBefore *int64  `json:"bytesBefore"`
		BytesAfter  *int64  `json:"bytesAfter"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return RepairResult{}, err
	}
	if wire.Repaired == nil || wire.Backup == nil || wire.BytesBefore == nil || wire.BytesAfter == nil || *wire.Repaired <= 0 || strings.TrimSpace(*wire.Backup) == "" || *wire.BytesBefore <= 0 || *wire.BytesAfter <= 0 {
		return RepairResult{}, errors.New("incomplete repair result")
	}
	return RepairResult{*wire.Repaired, *wire.Backup, *wire.BytesBefore, *wire.BytesAfter}, nil
}
func (s *Store) runRepairs(ctx context.Context, cancel context.CancelFunc, client *dsh.Client, epoch, generation uint64, ids []string) {
	defer cancel()
	rpc := &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
	for i, id := range ids {
		s.mu.Lock()
		if !s.repairCurrentLocked(client, epoch, generation) {
			s.mu.Unlock()
			return
		}
		if s.repair.StopRequested || ctx.Err() != nil {
			s.mu.Unlock()
			break
		}
		s.repair.SessionID = id
		s.repair.Unattempted = slices.Clone(ids[i+1:])
		s.mu.Unlock()
		s.fire()
		callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
		var raw json.RawMessage
		err := rpc.Call(callCtx, "usage-stats", "repair.session", map[string]any{"sessionId": id}, &raw)
		callCancel()
		var result RepairResult
		if err == nil {
			result, err = decodeRepairResult(raw)
		}
		s.mu.Lock()
		if !s.repairCurrentLocked(client, epoch, generation) {
			s.mu.Unlock()
			return
		}
		outcome := RepairOutcome{SessionID: id}
		if err != nil {
			outcome.Err = err.Error()
			s.repair.Err = err.Error()
		} else {
			outcome.Result = &result
			s.repair.Result = &result
			if s.repair.completed == nil {
				s.repair.completed = map[string]bool{}
			}
			s.repair.completed[id] = true
		}
		s.repair.Outcomes = append(s.repair.Outcomes, outcome)
		s.mu.Unlock()
		s.fire()
		// A failed/ambiguous write stops the sequence; never retry automatically.
		if err != nil {
			break
		}
	}
	s.mu.Lock()
	if !s.repairCurrentLocked(client, epoch, generation) {
		s.mu.Unlock()
		return
	}
	if len(s.repair.Outcomes) == 0 {
		s.repair.Busy = false
		s.repair.cancel = nil
		if ctx.Err() != nil {
			s.repair.Err = ctx.Err().Error()
		}
		s.mu.Unlock()
		s.fire()
		return
	}
	// Keep repair Busy throughout verification so neither another write nor a
	// manual refresh can race the follow-up scan. Generation guards also fence
	// host replacement and any newer scan before its result is committed.
	scanCtx, scanCancel := context.WithTimeout(ctx, 65*time.Second)
	s.statsCancel = scanCancel
	s.stats.generation++
	scanGeneration := s.stats.generation
	s.stats.Loading = true
	s.stats.Err = ""
	s.resetRankingLocked()
	s.repair.Rescanning = true
	s.mu.Unlock()
	s.fire()
	var raw json.RawMessage
	err := rpc.Call(scanCtx, "usage-stats", "overview", map[string]any{"force": true}, &raw)
	scanCancel()
	if err == nil {
		err = decodeOverview(raw)
	}
	s.mu.Lock()
	if !s.repairCurrentLocked(client, epoch, generation) || s.stats.generation != scanGeneration {
		s.mu.Unlock()
		return
	}
	s.stats.Loading = false
	s.statsCancel = nil
	if err != nil {
		s.stats.Err = "修复后的统计重扫失败 / Post-repair statistics rescan failed: " + err.Error()
	} else {
		s.stats.Raw = append(json.RawMessage(nil), raw...)
		s.reconcileStatsRangeLocked()
	}
	s.finishRepairScanLocked(err)
	s.repair.Busy = false
	s.repair.Rescanning = false
	s.repair.cancel = nil
	s.mu.Unlock()
	s.fire()
}

// Both automatic and later manual scans update verification, without erasing
// individual write results. A failed scan keeps previous statistics visible.
func (s *Store) finishRepairScanLocked(err error) {
	if len(s.repair.Targets) == 0 {
		return
	}
	s.repair.RescanComplete = false
	s.repair.RescanErr = ""
	s.repair.RescanStale = false
	if err != nil {
		s.repair.RescanErr = err.Error()
		return
	}
	var o Overview
	var coverage struct {
		Coverage struct {
			FailedIDs *[]string `json:"failedSessionIds"`
			Failed    *int      `json:"sessionsFailed"`
		} `json:"coverage"`
	}
	if json.Unmarshal(s.stats.Raw, &o) != nil || json.Unmarshal(s.stats.Raw, &coverage) != nil || coverage.Coverage.FailedIDs == nil || coverage.Coverage.Failed == nil || *coverage.Coverage.Failed < 0 || (*coverage.Coverage.Failed > 0 && len(*coverage.Coverage.FailedIDs) == 0) {
		s.repair.RescanErr = "Incomplete failure coverage in rescan; repair verification unavailable"
		return
	}
	s.repair.RescanComplete = true
	s.repair.RemainingFailed = slices.Clone(o.Coverage.FailedSessionIDs)
	s.repair.RescanStale = o.Stale
}

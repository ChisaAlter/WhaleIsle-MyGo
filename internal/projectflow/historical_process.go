package projectflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"whaleisle/internal/dsh"
)

// HistoricalTarget is captured from a selected, already-read exact report.
// Run/member expectations are supplied only when that source actually records them.
type HistoricalTarget struct {
	Assignment                                                               bool
	Terminal                                                                 bool
	RequestID                                                                string
	RequestRevision                                                          int
	ProjectID, WorkstreamID, DelegationRef, ReportID, RunID, WorkerSessionID string
	Report                                                                   Report
}
type HistoricalDelegation struct {
	Ref             string  `json:"ref"`
	MessageID       string  `json:"messageId"`
	RunID           string  `json:"runId"`
	CreatedAt       float64 `json:"createdAt"`
	Phase           string  `json:"phase"`
	Scope           string  `json:"scope"`
	RequestID       string  `json:"requestId"`
	RequestRevision int     `json:"requestRevision"`
}
type HistoricalBinding struct {
	Target                                HistoricalTarget
	Delegation                            HistoricalDelegation
	CoordinatorSessionID, WorkerSessionID string
	Materialized                          bool
}

func (t HistoricalTarget) validate() error {
	if t.Assignment {
		if t.ProjectID == "" || t.WorkstreamID == "" || t.DelegationRef == "" || t.RequestID == "" {
			return fmt.Errorf("select an exact requirement assignment before locating its process")
		}
		return nil
	}
	if t.Terminal {
		if t.ProjectID == "" || t.WorkstreamID == "" || t.DelegationRef == "" || t.RequestID == "" || t.RequestRevision < 1 {
			return fmt.Errorf("select an exact recorded settlement before locating its assignment")
		}
		return nil
	}
	if t.ProjectID == "" || t.WorkstreamID == "" || t.DelegationRef == "" || t.Report.DelegationRef != t.DelegationRef || t.ReportID == "" || t.Report.CallID != t.ReportID {
		return fmt.Errorf("select an exact recorded report before locating its assignment")
	}
	return nil
}
func (b HistoricalBinding) validate() error {
	if err := b.Target.validate(); err != nil {
		return err
	}
	d := b.Delegation
	if d.Ref != b.Target.DelegationRef || b.WorkerSessionID == "" || b.CoordinatorSessionID == "" || b.Target.WorkerSessionID != "" && b.Target.WorkerSessionID != b.WorkerSessionID || b.Target.RunID != "" && b.Target.RunID != d.RunID {
		return fmt.Errorf("original Project / member / delegation / run identity does not match")
	}
	if b.Target.Terminal && (d.RequestID != b.Target.RequestID || d.RequestRevision != b.Target.RequestRevision) {
		return fmt.Errorf("settlement requirement/version does not match the original assignment")
	}
	if b.Target.Assignment && d.RequestID != b.Target.RequestID {
		return fmt.Errorf("the recorded assignment belongs to another requirement")
	}
	if len(b.Target.Report.Request) > 0 && string(b.Target.Report.Request) != "null" {
		var request struct {
			ID       string `json:"id"`
			Revision int    `json:"revision"`
		}
		if json.Unmarshal(b.Target.Report.Request, &request) != nil {
			return fmt.Errorf("recorded report request is invalid")
		}
		if request.ID != "" && (request.ID != d.RequestID || request.Revision > 0 && request.Revision != d.RequestRevision) {
			return fmt.Errorf("original assignment requirement/version does not match the selected report")
		}
	}
	return nil
}

// HistoricalAssignment reads the original per-delegation receipt, never the
// current work brief/latestReport. Neither read activates a Session.
func (c Client) HistoricalAssignment(ctx context.Context, target HistoricalTarget) (HistoricalBinding, error) {
	out := HistoricalBinding{Target: cloneHistoricalTarget(target)}
	if err := target.validate(); err != nil {
		return out, err
	}
	var data struct {
		Workstream struct {
			ID              string                 `json:"id"`
			ProjectID       string                 `json:"projectId"`
			WorkerSessionID string                 `json:"workerSessionId"`
			Delegations     []HistoricalDelegation `json:"delegations"`
		} `json:"workstream"`
		Worker *struct {
			SessionID    string `json:"sessionId"`
			ProjectID    string `json:"projectId"`
			WorkstreamID string `json:"workstreamId"`
			Materialized bool   `json:"materialized"`
		} `json:"worker"`
	}
	if err := c.call(ctx, "workstreams/get", map[string]any{"projectId": target.ProjectID, "workstreamId": target.WorkstreamID}, &data); err != nil {
		return out, err
	}
	if data.Workstream.ID != target.WorkstreamID || data.Workstream.ProjectID != target.ProjectID {
		return out, fmt.Errorf("historical work does not belong to this Project")
	}
	matches := 0
	for _, d := range data.Workstream.Delegations {
		if d.Ref == target.DelegationRef {
			out.Delegation = d
			matches++
		}
	}
	if matches != 1 {
		return out, fmt.Errorf("original delegation receipt is missing or ambiguous; no current assignment is substituted")
	}
	out.WorkerSessionID = data.Workstream.WorkerSessionID
	if data.Worker != nil {
		if data.Worker.SessionID != out.WorkerSessionID || data.Worker.ProjectID != "" && data.Worker.ProjectID != target.ProjectID || data.Worker.WorkstreamID != "" && data.Worker.WorkstreamID != target.WorkstreamID {
			return out, fmt.Errorf("original member identity does not match")
		}
		out.Materialized = data.Worker.Materialized
	}
	lead, err := c.Open(ctx, target.ProjectID)
	if err != nil {
		return out, err
	}
	out.CoordinatorSessionID = lead
	return out, out.validate()
}
func (b HistoricalBinding) Address() (map[string]any, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}
	if !b.Materialized || b.Delegation.MessageID == "" {
		return nil, fmt.Errorf("original member Session or assignment message is unavailable; no replacement is created")
	}
	return map[string]any{"kind": "subagent", "parentSessionId": b.CoordinatorSessionID, "childSessionId": b.WorkerSessionID, "mode": "continuable"}, nil
}

type HistoricalWindow struct {
	Target                 HistoricalTarget
	Records                []ProcessRecord
	Cursor                 float64
	NextAssignmentBoundary bool
}
type HistoricalPage func(context.Context, float64, float64) (ProcessSnapshot, error)

func historicalVisible(r ProcessRecord) bool {
	if r.Type != "event" {
		return false
	}
	switch r.Event.Type {
	case "turn/start", "user/message", "assistant/message", "tool/call", "tool/result", "turn/end":
		return true
	}
	return false
}
func historicalMessage(r ProcessRecord) (id, message, kind, sender string) {
	var d struct {
		ID     string `json:"id"`
		Source struct {
			MessageID string `json:"messageId"`
			Kind      string `json:"kind"`
			SenderID  string `json:"senderId"`
		} `json:"source"`
	}
	if r.Event.Type == "user/message" {
		_ = json.Unmarshal(r.Event.Data, &d)
	}
	return d.ID, d.Source.MessageID, d.Source.Kind, d.Source.SenderID
}

// LocateHistoricalAssignment searches backward at one frozen Session cursor.
// It returns only the original assignment turn range, ending before the next
// Lead assignment. Missing history and no-progress pages fail explicitly.
func LocateHistoricalAssignment(ctx context.Context, b HistoricalBinding, first ProcessSnapshot, page HistoricalPage) (HistoricalWindow, error) {
	if _, err := b.Address(); err != nil {
		return HistoricalWindow{}, err
	}
	if first.Type != "snapshot" || first.Records == nil || first.Cursor <= 0 || math.IsNaN(first.Cursor) || math.IsInf(first.Cursor, 0) {
		return HistoricalWindow{}, fmt.Errorf("invalid original Session snapshot")
	}
	cursor := first.Cursor
	events := map[float64]ProcessRecord{}
	bytesRead := 0
	merge := func(records []ProcessRecord) error {
		for _, r := range records {
			if r.Type != "event" {
				continue
			}
			seq := r.Event.Seq
			if seq <= 0 || seq > cursor || math.IsNaN(seq) || math.IsInf(seq, 0) {
				return fmt.Errorf("history event falls outside the captured Session cursor")
			}
			if old, ok := events[seq]; ok {
				if old.Event.Type != r.Event.Type || !bytes.Equal(old.Event.Data, r.Event.Data) {
					return fmt.Errorf("conflicting history event at one sequence")
				}
				continue
			}
			events[seq] = r
			bytesRead += len(r.Event.Data)
			if len(events) > 20000 || bytesRead > 32*1024*1024 {
				return fmt.Errorf("historical search exceeds the bounded reader limit; no partial assignment is substituted")
			}
		}
		return nil
	}
	current := first
	for {
		if err := ctx.Err(); err != nil {
			return HistoricalWindow{}, err
		}
		if err := merge(current.Records); err != nil {
			return HistoricalWindow{}, err
		}
		var seqs []float64
		for seq := range events {
			seqs = append(seqs, seq)
		}
		sort.Float64s(seqs)
		var records []ProcessRecord
		for _, seq := range seqs {
			if r := events[seq]; historicalVisible(r) {
				records = append(records, r)
			}
		}
		target := -1
		for i, r := range records {
			id, message, _, _ := historicalMessage(r)
			if r.Event.Type == "user/message" && (id == b.Delegation.MessageID || message == b.Delegation.MessageID) {
				if target >= 0 {
					return HistoricalWindow{}, fmt.Errorf("assignment message is ambiguous in original Session")
				}
				target = i
			}
		}
		start := -1
		if target >= 0 {
			for i := 0; i < target; i++ {
				if records[i].Event.Type == "turn/start" {
					start = i
				}
			}
		}
		if target >= 0 && (start >= 0 || !current.HasMore) {
			if start < 0 {
				start = 0
			}
			end := len(records)
			bounded := false
			for i := target + 1; i < len(records); i++ {
				_, _, kind, sender := historicalMessage(records[i])
				if kind == "team-message" && sender == b.CoordinatorSessionID {
					end = i
					for j := target + 1; j < i; j++ {
						if records[j].Event.Type == "turn/start" {
							end = j
						}
					}
					bounded = true
					break
				}
			}
			return HistoricalWindow{Target: cloneHistoricalTarget(b.Target), Records: cloneHistoricalRecords(records[start:end]), Cursor: cursor, NextAssignmentBoundary: bounded}, nil
		}
		if !current.HasMore {
			return HistoricalWindow{}, fmt.Errorf("original assignment message is not present in retained Session history")
		}
		if len(seqs) == 0 || page == nil {
			return HistoricalWindow{}, fmt.Errorf("earlier original Session history is unavailable")
		}
		before := seqs[0]
		next, err := page(ctx, cursor, before)
		if err != nil {
			return HistoricalWindow{}, err
		}
		if next.Type != "snapshot" || next.Records == nil || next.Cursor != cursor {
			return HistoricalWindow{}, fmt.Errorf("earlier page does not belong to the captured history cursor")
		}
		progressed := false
		for _, r := range next.Records {
			if r.Type == "event" && r.Event.Seq < before {
				progressed = true
			}
		}
		if !progressed {
			return HistoricalWindow{}, fmt.Errorf("earlier history made no progress; original assignment was not located")
		}
		current = next
	}
}

// ReadHistoricalProcess is called only after the user explicitly opens process.
// A subagent follow observes and validates the original Session; it does not
// activate it, send input, create a replacement, or restart an old run. The carrier is released after
// its opening snapshot; all older pages stay at that snapshot's cursor.
func (c Client) ReadHistoricalProcess(ctx context.Context, mux *dsh.Mux, b HistoricalBinding) (HistoricalWindow, error) {
	address, err := b.Address()
	if err != nil {
		return HistoricalWindow{}, err
	}
	if mux == nil {
		return HistoricalWindow{}, fmt.Errorf("original Session transport is unavailable")
	}
	if err = mux.Dial(ctx); err != nil {
		return HistoricalWindow{}, err
	}
	stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": map[string]any{"address": address, "maxMessages": 30}}})
	if err != nil {
		return HistoricalWindow{}, err
	}
	raw, err := stream.Recv(ctx)
	stream.Cancel()
	if err != nil {
		return HistoricalWindow{}, err
	}
	first, err := ParseProcessSnapshot(raw)
	if err != nil {
		return HistoricalWindow{}, err
	}
	var opening struct {
		Header struct {
			ID            string `json:"id"`
			Origin        string `json:"origin"`
			ParentSession string `json:"parentSession"`
		} `json:"header"`
	}
	if err = json.Unmarshal(raw, &opening); err != nil {
		return HistoricalWindow{}, err
	}
	if opening.Header.ID != b.WorkerSessionID || opening.Header.Origin != "subagent" || opening.Header.ParentSession != b.CoordinatorSessionID {
		return HistoricalWindow{}, fmt.Errorf("opening snapshot does not belong to the original member and Lead")
	}
	api := dsh.Client{Endpoint: dsh.Endpoint{Origin: c.Origin, Cookie: c.Cookie}, HTTP: c.HTTP}
	return LocateHistoricalAssignment(ctx, b, first, func(ctx context.Context, through, before float64) (ProcessSnapshot, error) {
		var out ProcessSnapshot
		err := api.Call(ctx, "session/page", map[string]any{"request": map[string]any{"address": address, "throughSeq": through, "beforeSeq": before, "maxMessages": 30}}, &out)
		// session/page carries records/hasMore; the requested frozen cursor is local.
		if out.Cursor != 0 && out.Cursor != through {
			return out, fmt.Errorf("Host returned a different history cursor")
		}
		out.Cursor = through
		out.Type = "snapshot"
		return out, err
	})
}
func cloneHistoricalRecords(in []ProcessRecord) []ProcessRecord {
	out := append([]ProcessRecord(nil), in...)
	for i := range out {
		out[i].Event.Data = append(json.RawMessage(nil), out[i].Event.Data...)
	}
	return out
}
func cloneHistoricalTarget(t HistoricalTarget) HistoricalTarget {
	t.Report = cloneDeliveryReport(t.Report)
	return t
}

type HistoricalSnapshot struct {
	Host             string
	Generation, View uint64
	Target           HistoricalTarget
	Binding          *HistoricalBinding
	Window           *HistoricalWindow
	Page             int
	Busy             bool
	Error            string
}
type HistoricalBrowser struct {
	state  HistoricalSnapshot
	serial uint64
	closed bool
}
type HistoricalRequest struct {
	owner  *HistoricalBrowser
	serial uint64
	kind   string
}

func NewHistoricalBrowser(host string, generation uint64, t HistoricalTarget) *HistoricalBrowser {
	return &HistoricalBrowser{state: HistoricalSnapshot{Host: host, Generation: generation, Target: cloneHistoricalTarget(t)}}
}
func (b *HistoricalBrowser) Snapshot() HistoricalSnapshot {
	s := b.state
	s.Target = cloneHistoricalTarget(s.Target)
	if s.Binding != nil {
		v := *s.Binding
		v.Target = cloneHistoricalTarget(v.Target)
		s.Binding = &v
	}
	if s.Window != nil {
		v := *s.Window
		v.Target = cloneHistoricalTarget(v.Target)
		v.Records = cloneHistoricalRecords(v.Records)
		s.Window = &v
	}
	return s
}
func (b *HistoricalBrowser) View() uint64 { return b.state.View }
func (b *HistoricalBrowser) Begin(kind string) (HistoricalRequest, error) {
	if b.closed || b.state.Busy || b.state.Host == "" || (kind != "record" && kind != "process") {
		return HistoricalRequest{}, fmt.Errorf("historical reader is unavailable")
	}
	if err := b.state.Target.validate(); err != nil {
		return HistoricalRequest{}, err
	}
	if kind == "process" {
		if b.state.Binding == nil {
			return HistoricalRequest{}, fmt.Errorf("load the exact assignment first")
		}
		if _, err := b.state.Binding.Address(); err != nil {
			return HistoricalRequest{}, err
		}
	}
	b.serial++
	b.state.View++
	b.state.Busy = true
	b.state.Error = ""
	return HistoricalRequest{b, b.serial, kind}, nil
}
func (b *HistoricalBrowser) Pending(r HistoricalRequest) bool {
	return !b.closed && r.owner == b && r.serial == b.serial && b.state.Busy
}
func (b *HistoricalBrowser) CompleteRecord(r HistoricalRequest, v HistoricalBinding, err error) bool {
	if !b.Pending(r) || r.kind != "record" {
		return false
	}
	if err == nil {
		err = v.validate()
		if err == nil && !sameHistoricalTarget(v.Target, b.state.Target) {
			err = fmt.Errorf("historical receipt belongs to another source")
		}
	}
	if err == nil && b.state.Binding != nil {
		old := b.state.Binding
		if old.Delegation.Ref != v.Delegation.Ref || old.Delegation.MessageID != v.Delegation.MessageID || old.Delegation.RunID != v.Delegation.RunID || old.Delegation.RequestID != v.Delegation.RequestID || old.Delegation.RequestRevision != v.Delegation.RequestRevision || old.CoordinatorSessionID != v.CoordinatorSessionID || old.WorkerSessionID != v.WorkerSessionID {
			err = fmt.Errorf("original assignment binding changed; keeping the previously read receipt and process")
		}
	}
	b.state.Busy = false
	b.state.View++
	if err != nil {
		b.state.Error = err.Error()
		return true
	}
	v.Target = cloneHistoricalTarget(v.Target)
	b.state.Binding = &v
	return true
}
func sameHistoricalTarget(a, b HistoricalTarget) bool {
	return a.Terminal == b.Terminal && a.RequestID == b.RequestID && a.RequestRevision == b.RequestRevision && a.ProjectID == b.ProjectID && a.WorkstreamID == b.WorkstreamID && a.DelegationRef == b.DelegationRef && a.ReportID == b.ReportID && a.RunID == b.RunID && a.WorkerSessionID == b.WorkerSessionID && bytes.Equal(a.Report.Request, b.Report.Request)
}
func (b *HistoricalBrowser) CompleteProcess(r HistoricalRequest, v HistoricalWindow, err error) bool {
	if !b.Pending(r) || r.kind != "process" {
		return false
	}
	b.state.Busy = false
	b.state.View++
	if err == nil && !sameHistoricalTarget(v.Target, b.state.Target) {
		err = fmt.Errorf("process window belongs to another historical source")
	}
	if err != nil {
		b.state.Error = err.Error()
		return true
	}
	v.Target = cloneHistoricalTarget(v.Target)
	v.Records = cloneHistoricalRecords(v.Records)
	b.state.Window = &v
	b.state.Page = 0
	return true
}
func (b *HistoricalBrowser) MovePage(delta int) {
	if b.closed || b.state.Busy || b.state.Window == nil {
		return
	}
	n := b.state.Page + delta
	if n >= 0 && n*50 < len(b.state.Window.Records) {
		b.state.Page = n
		b.state.View++
	}
}
func (b *HistoricalBrowser) Close() {
	b.closed = true
	b.serial++
	b.state.View++
	b.state.Busy = false
}

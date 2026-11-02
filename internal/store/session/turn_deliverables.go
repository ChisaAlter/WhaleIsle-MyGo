package session

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/dsh"
)

const MessageKindTurnDelivery MessageKind = "turn-delivery"

// Coordinates always belong to the viewed session (also for inherited fork
// events). Paths/descriptions are display data, never action route arguments.
type TurnFileTarget struct {
	Kind      string
	Turn, Seq uint64
	Index     int
}
type TurnPresentedFile struct {
	Target            TurnFileTarget
	Path, Description string
}
type TurnDelivery struct {
	Turn    uint64
	Changes *TurnFileTarget
	Files   []TurnPresentedFile
}
type TurnFileScope struct {
	Binding, SessionID string
	Generation         uint64
}
type TurnFileRequest struct {
	Target      TurnFileTarget
	Action      string
	Application string
}

// TurnFileApplicationAction carries an opaque Host application ID through the
// existing addressed UI action callback. It is never a path or shell command.
func TurnFileApplicationAction(id string) string { return "application:" + id }

func normalizeTurnFileRequest(r TurnFileRequest) TurnFileRequest {
	if id, ok := strings.CutPrefix(r.Action, "application:"); ok && id != "" {
		r.Action, r.Application = "open", id
	}
	return r
}

func (r TurnFileRequest) StateKey() TurnFileRequest {
	r = normalizeTurnFileRequest(r)
	if r.Action == "open" || r.Action == "reveal" {
		r.Action = "native"
		r.Application = ""
	}
	return r
}

type turnDeliveryEvent struct {
	Type string   `json:"type"`
	Seq  *float64 `json:"seq"`
	Data struct {
		Turn   *float64          `json:"turn"`
		CallID string            `json:"callId"`
		Files  []json.RawMessage `json:"files"`
	} `json:"data"`
}

func validTurnCoordinate(n *float64) bool {
	return n != nil && *n >= 1 && *n <= 9007199254740991 && math.Trunc(*n) == *n
}
func readTurnDeliveryEvent(raw json.RawMessage) (turnDeliveryEvent, bool) {
	if len(raw) == 0 {
		return turnDeliveryEvent{}, false
	}
	if typ, ok := peekEventType(raw); ok {
		switch typ {
		case "turn/end", "workspace/changes", "deliverables/presented", "assistant/message":
		default:
			return turnDeliveryEvent{}, false
		}
	} else if !bytes.Contains(raw, []byte("turn/end")) && !bytes.Contains(raw, []byte("workspace/changes")) && !bytes.Contains(raw, []byte("deliverables/presented")) && !bytes.Contains(raw, []byte("assistant/message")) && !bytes.Contains(raw, []byte(`\u`)) {
		return turnDeliveryEvent{}, false
	}
	var e turnDeliveryEvent
	ok := json.Unmarshal(raw, &e) == nil && validTurnCoordinate(e.Seq) && validTurnCoordinate(e.Data.Turn)
	return e, ok
}

func hasTurnDeliveryCandidates(messages []Message) bool {
	for _, m := range messages {
		if len(m.Raw) == 0 {
			continue
		}
		if m.EventType == "workspace/changes" || m.EventType == "deliverables/presented" {
			return true
		}
		if typ, ok := peekEventType(m.Raw); ok {
			if typ == "workspace/changes" || typ == "deliverables/presented" {
				return true
			}
			continue
		}
		if bytes.Contains(m.Raw, []byte("workspace/changes")) || bytes.Contains(m.Raw, []byte("deliverables/presented")) || bytes.Contains(m.Raw, []byte(`\u`)) {
			return true
		}
	}
	return false
}
func presentedFromEvent(e turnDeliveryEvent) []TurnPresentedFile {
	if e.Type != "deliverables/presented" || e.Data.CallID == "" || e.Data.Files == nil {
		return nil
	}
	var out []TurnPresentedFile
	for index, raw := range e.Data.Files {
		var f struct {
			Path        string  `json:"path"`
			Description *string `json:"description"`
		}
		if json.Unmarshal(raw, &f) != nil || strings.TrimSpace(f.Path) == "" {
			continue
		}
		// Explicit null is not the optional string contract.
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		if d, exists := fields["description"]; exists && string(d) == "null" {
			continue
		}
		description := ""
		if f.Description != nil {
			description = *f.Description
		}
		out = append(out, TurnPresentedFile{Target: TurnFileTarget{Kind: "presented", Turn: uint64(*e.Data.Turn), Seq: uint64(*e.Seq), Index: index}, Path: f.Path, Description: description})
	}
	return out
}

// turnDeliveryRecord is one message's decoded delivery coordinate. Both
// builders below read it once so a frame does not decode every raw twice.
type turnDeliveryRecord struct {
	event       turnDeliveryEvent
	ok          bool
	streaming   bool
	bodyPresent bool
}

func readTurnDeliveryRecords(messages []Message) []turnDeliveryRecord {
	records := make([]turnDeliveryRecord, len(messages))
	for i, m := range messages {
		e, ok := readTurnDeliveryEvent(m.Raw)
		records[i] = turnDeliveryRecord{event: e, ok: ok, streaming: m.Streaming, bodyPresent: strings.TrimSpace(m.Body) != ""}
	}
	return records
}

// TurnDeliveries derives only durable, explicitly owned facts. The turn/end
// anchor also supports turns with no closing prose; live/incomplete turns have
// no delivery actions. Same-path deliveries keep the latest original index.
//
// The events are decoded once and grouped by turn: rescanning every message for
// every turn decoded each raw once per turn (quadratic), which showed up as the
// second largest cost of a frame in tmp/stalls.
func TurnDeliveries(messages []Message) map[uint64]TurnDelivery {
	if !hasTurnDeliveryCandidates(messages) {
		return map[uint64]TurnDelivery{}
	}
	return turnDeliveriesFromRecords(readTurnDeliveryRecords(messages))
}

func turnDeliveriesFromRecords(records []turnDeliveryRecord) map[uint64]TurnDelivery {
	closing := map[uint64]uint64{}
	byTurn := make(map[uint64][]turnDeliveryRecord)
	for _, r := range records {
		if !r.ok {
			continue
		}
		e := r.event
		turn := uint64(*e.Data.Turn)
		if e.Type == "turn/end" && !r.streaming {
			closing[turn] = uint64(*e.Seq)
		}
		byTurn[turn] = append(byTurn[turn], r)
	}
	out := map[uint64]TurnDelivery{}
	for turn, end := range closing {
		events := byTurn[turn]
		// The original tail filters presented declarations at its last text
		// assistant's sequence, falling back to turn/end without closing prose.
		presentedBefore := end
		var lastReply uint64
		for _, item := range events {
			e := item.event
			if e.Type == "assistant/message" && !item.streaming && item.bodyPresent && uint64(*e.Seq) < end && uint64(*e.Seq) > lastReply {
				lastReply = uint64(*e.Seq)
			}
		}
		if lastReply != 0 {
			presentedBefore = lastReply
		}
		d := TurnDelivery{Turn: turn}
		positions := map[string]int{}
		// Follow history is ordered; compare sequences as well so late replay
		// cannot replace a newer path or change announcement.
		for _, item := range events {
			e := item.event
			if item.streaming || uint64(*e.Seq) >= end {
				continue
			}
			if e.Type == "workspace/changes" && (d.Changes == nil || uint64(*e.Seq) > d.Changes.Seq) {
				d.Changes = &TurnFileTarget{Kind: "changes", Turn: turn, Seq: uint64(*e.Seq), Index: -1}
			}
			if uint64(*e.Seq) >= presentedBefore {
				continue
			}
			for _, f := range presentedFromEvent(e) {
				if i, exists := positions[f.Path]; exists {
					if d.Files[i].Target.Seq <= f.Target.Seq {
						d.Files[i] = f
					}
				} else {
					positions[f.Path] = len(d.Files)
					d.Files = append(d.Files, f)
				}
			}
		}
		if d.Changes != nil || len(d.Files) > 0 {
			out[turn] = d
		}
	}
	return out
}

func ProjectTurnDeliveryMessages(messages []Message) []Message {
	if !hasTurnDeliveryCandidates(messages) {
		return append([]Message(nil), messages...)
	}
	records := readTurnDeliveryRecords(messages)
	deliveries := turnDeliveriesFromRecords(records)
	out := make([]Message, 0, len(messages))
	for i, m := range messages {
		r := records[i]
		if r.ok && (r.event.Type == "workspace/changes" || r.event.Type == "deliverables/presented") {
			continue
		}
		out = append(out, m)
		if !r.ok || r.event.Type != "turn/end" {
			continue
		}
		e := r.event
		if d, exists := deliveries[uint64(*e.Data.Turn)]; exists {
			copy := d
			out = append(out, Message{ID: fmt.Sprintf("turn-delivery:%d:%g", d.Turn, *e.Seq), Kind: MessageKindTurnDelivery, Role: "assistant", Seq: *e.Seq, TurnDelivery: &copy})
			delete(deliveries, uint64(*e.Data.Turn))
		}
	}
	return out
}

type TurnChangedFile struct {
	Path      string `json:"path"`
	Display   string `json:"display"`
	Added     int    `json:"added"`
	Deleted   int    `json:"deleted"`
	Binary    bool   `json:"binary"`
	Oversized bool   `json:"oversized"`
}
type TurnChangesSummary struct {
	Turn    uint64            `json:"turn"`
	Files   []TurnChangedFile `json:"files"`
	Total   int               `json:"total"`
	Added   int               `json:"added"`
	Deleted int               `json:"deleted"`
}
type TurnDiffHunk struct {
	OldStart int      `json:"oldStart"`
	OldLines int      `json:"oldLines"`
	NewStart int      `json:"newStart"`
	NewLines int      `json:"newLines"`
	Lines    []string `json:"lines"`
}
type TurnFileDiff struct {
	Kind    string         `json:"kind"`
	Path    string         `json:"path"`
	Display string         `json:"display"`
	Before  *bool          `json:"before"`
	After   *bool          `json:"after"`
	Coarse  *bool          `json:"coarse"`
	Hunks   []TurnDiffHunk `json:"hunks"`
}

func (d TurnFileDiff) Patch() string {
	var out strings.Builder
	for _, h := range d.Hunks {
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
		for _, line := range h.Lines {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	return out.String()
}

type TurnPresentedHost struct {
	Name        string  `json:"name"`
	Available   bool    `json:"available"`
	FileManager *string `json:"fileManager"`
}
type TurnFileResult struct {
	Busy, Ready, Unavailable bool
	Action, Error            string
	Summary                  *TurnChangesSummary
	Diff                     *TurnFileDiff
	Host                     *TurnPresentedHost
	Applications             []TurnFileApplication
}

type TurnFileApplication struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Default bool    `json:"default"`
	Icon    *string `json:"icon"`
}
type TurnFilesSnapshot struct {
	Scope     TurnFileScope
	HostLabel string
	Results   map[TurnFileRequest]TurnFileResult
}
type turnFilesState struct {
	scope   TurnFileScope
	results map[TurnFileRequest]TurnFileResult
	pending map[TurnFileRequest]context.CancelFunc
}

func turnFileHistoryBinding(client *dsh.Client, sessionID string) string {
	if client == nil || sessionID == "" {
		return ""
	}
	identity := sha256.Sum256([]byte(client.Endpoint.Origin + "\x00" + client.Endpoint.Cookie))
	return fmt.Sprintf("%p\x00%s\x00%x", client, sessionID, identity)
}
func (s *Store) turnFilesScopeLocked() TurnFileScope {
	binding := turnFileHistoryBinding(s.client, s.selectedID)
	if binding == "" || s.turnFilesHistoryBinding != binding {
		return TurnFileScope{}
	}
	return TurnFileScope{Binding: binding, SessionID: s.selectedID, Generation: s.followGeneration}
}
func (s *Store) resetTurnFilesLocked() {
	s.invalidateViewLocked()
	for _, cancel := range s.turnFiles.pending {
		cancel()
	}
	s.turnFiles = turnFilesState{}
}
func (s *Store) turnFilesSnapshotLocked() TurnFilesSnapshot {
	scope := s.turnFilesScopeLocked()
	if scope != s.turnFiles.scope {
		s.resetTurnFilesLocked()
		s.turnFiles.scope = scope
	}
	out := TurnFilesSnapshot{Scope: scope, Results: map[TurnFileRequest]TurnFileResult{}}
	if scope.Binding != "" {
		out.HostLabel = s.client.Endpoint.Origin
	}
	for key, r := range s.turnFiles.results {
		r.Applications = append([]TurnFileApplication(nil), r.Applications...)
		if r.Summary != nil {
			v := *r.Summary
			v.Files = append([]TurnChangedFile(nil), v.Files...)
			r.Summary = &v
		}
		if r.Diff != nil {
			v := *r.Diff
			v.Hunks = append([]TurnDiffHunk(nil), v.Hunks...)
			for i := range v.Hunks {
				v.Hunks[i].Lines = append([]string(nil), v.Hunks[i].Lines...)
			}
			r.Diff = &v
		}
		if r.Host != nil {
			v := *r.Host
			r.Host = &v
		}
		out.Results[key] = r
	}
	return out
}
func (s *Store) turnFileTargetLocked(scope TurnFileScope, target TurnFileTarget) (string, bool) {
	if scope.Binding == "" || scope != s.turnFilesScopeLocked() {
		return "", false
	}
	d, ok := TurnDeliveries(s.messages)[target.Turn]
	if !ok {
		return "", false
	}
	switch target.Kind {
	case "presented":
		for _, f := range d.Files {
			if f.Target == target {
				return f.Path, true
			}
		}
	case "changes":
		if d.Changes != nil && d.Changes.Seq == target.Seq {
			if target.Index == -1 {
				return "", true
			}
			key := TurnFileRequest{Target: *d.Changes, Action: "summary"}
			summary := s.turnFiles.results[key].Summary
			if summary != nil && target.Index >= 0 && target.Index < len(summary.Files) {
				return summary.Files[target.Index].Path, true
			}
		}
	}
	return "", false
}

// TurnFilePreviewPath is the gate for the existing Files preview callback. A
// response's path never becomes arbitrary local-open authority.
func (s *Store) TurnFilePreviewPath(scope TurnFileScope, target TurnFileTarget) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if target.Kind != "presented" {
		return "", false
	}
	return s.turnFileTargetLocked(scope, target)
}

// RunTurnFileAction runs exactly the clicked read or file handoff. No retry,
// prompt, tool execution, question answer, approval or document action exists.
// Caller runs this off the UI thread. Both native buttons share one busy key.
func (s *Store) RunTurnFileAction(ctx context.Context, scope TurnFileScope, request TurnFileRequest) {
	request = normalizeTurnFileRequest(request)
	s.mu.Lock()
	_ = s.turnFilesSnapshotLocked()
	expectedPath, valid := s.turnFileTargetLocked(scope, request.Target)
	if !valid || ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	route, method := "", http.MethodGet
	switch request.Action {
	case "summary":
		if request.Target.Kind == "changes" && request.Target.Index == -1 {
			route = "changes.summary"
		}
	case "diff":
		if request.Target.Kind == "changes" && request.Target.Index >= 0 {
			route = "changes.diff"
		}
	case "host":
		route = "present.host"
	case "open", "reveal", "applications":
		if request.Target.Index >= 0 {
			if request.Action != "applications" {
				method = http.MethodPost
			}
			if request.Target.Kind == "presented" {
				route = "present.open"
			} else if request.Target.Kind == "changes" {
				route = "changes.open"
			}
		}
	}
	key := request.StateKey()
	if route == "" || s.turnFiles.results[key].Busy {
		s.mu.Unlock()
		return
	}
	client := s.client
	captured := *client
	endpoint := client.Endpoint
	child, cancel := context.WithTimeout(ctx, 30*time.Second)
	if s.turnFiles.results == nil {
		s.turnFiles.results = map[TurnFileRequest]TurnFileResult{}
		s.turnFiles.pending = map[TurnFileRequest]context.CancelFunc{}
	}
	s.turnFiles.results[key] = TurnFileResult{Busy: true, Action: request.Action}
	s.turnFiles.pending[key] = cancel
	s.mu.Unlock()
	s.fire()
	result := runTurnFileHTTP(child, &captured, scope, request, route, method, expectedPath)
	cancel()
	s.mu.Lock()
	if s.client != client || client.Endpoint != endpoint || scope != s.turnFilesScopeLocked() || scope != s.turnFiles.scope {
		s.mu.Unlock()
		return
	}
	delete(s.turnFiles.pending, key)
	// A history replacement can revoke the targeted declaration during a read.
	if _, ok := s.turnFileTargetLocked(scope, request.Target); !ok {
		delete(s.turnFiles.results, key)
		s.mu.Unlock()
		s.fire()
		return
	}
	s.turnFiles.results[key] = result
	s.mu.Unlock()
	s.fire()
}
func runTurnFileHTTP(ctx context.Context, client *dsh.Client, scope TurnFileScope, request TurnFileRequest, route, method, expectedPath string) TurnFileResult {
	result := TurnFileResult{Action: request.Action}
	fail := func(err error) TurnFileResult { result.Error = err.Error(); return result }
	origin, err := url.Parse(client.Endpoint.Origin)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" {
		return fail(fmt.Errorf("invalid Host origin"))
	}
	address, err := url.JoinPath(client.Endpoint.Origin, "api", route)
	if err != nil {
		return fail(err)
	}
	u, err := url.Parse(address)
	if err != nil {
		return fail(err)
	}
	if route != "present.host" {
		q := url.Values{"sessionId": {scope.SessionID}, "seq": {strconv.FormatUint(request.Target.Seq, 10)}}
		if request.Target.Index >= 0 {
			q.Set("index", strconv.Itoa(request.Target.Index))
		}
		if request.Action == "reveal" {
			q.Set("action", "reveal")
		}
		if request.Action == "open" && request.Application != "" {
			q.Set("application", request.Application)
		}
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return fail(err)
	}
	req.Header.Set("Cookie", client.Endpoint.Cookie)
	req.Header.Set("Accept", "application/json")
	httpClient := http.Client{}
	if client.HTTP != nil {
		httpClient = *client.HTTP
	}
	// Never replay a native action or forward auth to a redirect destination,
	// including same-origin redirects. A redirect is a visible explicit failure.
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := httpClient.Do(req)
	if err != nil {
		return fail(err)
	}
	defer response.Body.Close()
	if response.StatusCode == 404 {
		result.Unavailable = true
		return fail(fmt.Errorf("recorded file or turn summary is no longer available on this Host"))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fail(fmt.Errorf("%s: HTTP %d; retry only this file action", route, response.StatusCode))
	}
	if method == http.MethodPost {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		result.Ready = true
		return result
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil {
		return fail(err)
	}
	if len(raw) > 8<<20 {
		return fail(fmt.Errorf("turn file response exceeds 8 MiB"))
	}
	switch request.Action {
	case "applications":
		var apps []TurnFileApplication
		var rows []json.RawMessage
		if json.Unmarshal(raw, &apps) != nil || apps == nil || json.Unmarshal(raw, &rows) != nil {
			return fail(fmt.Errorf("invalid native application list"))
		}
		for i, app := range apps {
			if !turnFileRequiredFields(rows[i], "id", "name", "default") || app.ID == "" {
				return fail(fmt.Errorf("invalid native application entry"))
			}
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(rows[i], &fields)
			if fields["icon"] == nil || app.Icon != nil && !validTurnFileApplicationIcon(*app.Icon) {
				return fail(fmt.Errorf("invalid native application icon"))
			}
		}
		result.Applications = apps
	case "summary":
		var v TurnChangesSummary
		if !turnFileRequiredFields(raw, "turn", "files", "total", "added", "deleted") || json.Unmarshal(raw, &v) != nil || v.Turn != request.Target.Turn || v.Files == nil || v.Total < len(v.Files) || v.Added < 0 || v.Deleted < 0 {
			return fail(fmt.Errorf("invalid or foreign turn summary"))
		}
		var rows struct {
			Files []json.RawMessage `json:"files"`
		}
		_ = json.Unmarshal(raw, &rows)
		for i, f := range v.Files {
			if !turnFileRequiredFields(rows.Files[i], "path", "display", "added", "deleted") || strings.TrimSpace(f.Path) == "" || f.Display == "" || f.Added < 0 || f.Deleted < 0 {
				return fail(fmt.Errorf("invalid changed-file row"))
			}
		}
		result.Summary = &v
	case "diff":
		var v TurnFileDiff
		if json.Unmarshal(raw, &v) != nil || v.Path != expectedPath || v.Display == "" {
			return fail(fmt.Errorf("invalid or mismatched comparison"))
		}
		switch v.Kind {
		case "binary", "oversized":
		case "text":
			if v.Before == nil || v.After == nil || v.Coarse == nil || v.Hunks == nil {
				return fail(fmt.Errorf("incomplete text comparison"))
			}
			var rows struct {
				Hunks []json.RawMessage `json:"hunks"`
			}
			_ = json.Unmarshal(raw, &rows)
			for i, h := range v.Hunks {
				if !turnFileRequiredFields(rows.Hunks[i], "oldStart", "oldLines", "newStart", "newLines", "lines") || h.OldStart < 0 || h.OldLines < 0 || h.NewStart < 0 || h.NewLines < 0 || h.Lines == nil {
					return fail(fmt.Errorf("invalid comparison hunk"))
				}
				for _, line := range h.Lines {
					if line == "" || !strings.ContainsRune("+ -", rune(line[0])) {
						return fail(fmt.Errorf("invalid comparison line"))
					}
				}
			}
		default:
			return fail(fmt.Errorf("unsupported comparison kind"))
		}
		result.Diff = &v
	case "host":
		var v TurnPresentedHost
		var fields map[string]json.RawMessage
		if !turnFileRequiredFields(raw, "name", "available") || json.Unmarshal(raw, &v) != nil || json.Unmarshal(raw, &fields) != nil || fields["fileManager"] == nil {
			return fail(fmt.Errorf("invalid Host desktop metadata"))
		}
		if v.FileManager != nil && *v.FileManager != "finder" && *v.FileManager != "explorer" && *v.FileManager != "directory" {
			return fail(fmt.Errorf("invalid Host file manager"))
		}
		result.Host = &v
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	result.Ready = true
	return result
}

func validTurnFileApplicationIcon(value string) bool {
	data, ok := strings.CutPrefix(value, "data:image/png;base64,")
	if !ok {
		data, ok = strings.CutPrefix(value, "data:image/svg+xml;base64,")
	}
	if !ok || data == "" {
		return false
	}
	for _, c := range data {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/' || c == '=' {
			continue
		}
		return false
	}
	return true
}

func turnFileRequiredFields(raw json.RawMessage, names ...string) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return false
	}
	for _, name := range names {
		if value := fields[name]; len(value) == 0 || string(value) == "null" {
			return false
		}
	}
	return true
}

package session

import (
	"bytes"
	"encoding/json"
	"math"
	"sort"
	"strings"
)

type TrajectoryProviderMetadata struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}
type TrajectoryAttemptEvidence struct {
	Seq  float64         `json:"seq"`
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"event"`
}
type TrajectoryLiveAttempt struct {
	Usage           map[string]float64 `json:"usage,omitempty"`
	AttemptID       string             `json:"attemptId"`
	StartedAfterSeq float64            `json:"startedAfterSeq"`
	NextIndex       int                `json:"nextIndex"`
	Stream          []json.RawMessage  `json:"stream"`
}

// A request is a recorded step/compaction operation, never an estimated model
// call. Retry identities and durable attempt sequence numbers remain explicit.
type TrajectoryRequest struct {
	ID               string                      `json:"id"`
	Purpose          string                      `json:"purpose"`
	Turn             *float64                    `json:"turn"`
	Step             *float64                    `json:"step,omitempty"`
	CompactionID     string                      `json:"compactionId,omitempty"`
	StartSeq         float64                     `json:"startSeq"`
	StartedAt        *float64                    `json:"startedAt"`
	CompletedAt      *float64                    `json:"completedAt"`
	Status           string                      `json:"status"`
	ResultSeq        *float64                    `json:"resultSeq,omitempty"`
	ReplacementSeq   *float64                    `json:"replacementSeq,omitempty"`
	ProviderMetadata *TrajectoryProviderMetadata `json:"providerMetadata,omitempty"`
	Usage            json.RawMessage             `json:"usage,omitempty"`
	RequestConfig    json.RawMessage             `json:"requestConfig,omitempty"`
	Header           *TrajectoryRequestHeader    `json:"header,omitempty"`
	Summary          string                      `json:"summary,omitempty"`
	RawOutput        *string                     `json:"rawOutput,omitempty"`
	SummaryBlocks    json.RawMessage             `json:"summaryBlocks,omitempty"`
	RawOutputBlocks  json.RawMessage             `json:"rawOutputBlocks,omitempty"`
	Error            *string                     `json:"error,omitempty"`
	Retries          []RetryAttempt              `json:"retries,omitempty"`
	Attempts         []TrajectoryAttemptEvidence `json:"attempts,omitempty"`
	LiveAttempt      *TrajectoryLiveAttempt      `json:"liveAttempt,omitempty"`
	Events           []json.RawMessage           `json:"events"`
}
type trajectoryRequestEvent struct {
	Type string   `json:"type"`
	Seq  float64  `json:"seq"`
	Time *float64 `json:"time"`
	Data struct {
		Turn, Step      *float64
		CompactionID    string          `json:"compactionId"`
		Usage           json.RawMessage `json:"usage"`
		Interrupted     bool            `json:"interrupted"`
		Error           *string         `json:"error"`
		Provider, Model string
		Summary         json.RawMessage `json:"summary"`
		RawOutput       json.RawMessage `json:"rawOutput"`
		MaxTokens       *float64        `json:"maxTokens"`
		Message         struct {
			Source TrajectoryProviderMetadata `json:"source"`
		} `json:"message"`
		Source struct{ Kind, CompactionID string } `json:"source"`
	} `json:"data"`
}

type trajectoryRequestCacheEntry struct {
	raw   string
	event trajectoryRequestEvent
	ok    bool
}

func isTrajectoryRequestCandidate(raw json.RawMessage) bool {
	if typ, ok := peekEventType(raw); ok {
		switch typ {
		case "step/start", "step/end", "assistant/message", "assistant/attempt",
			"llm/retry", "llm/retry-started",
			"compaction/start", "compaction/summary", "compaction/end",
			"user/message", "session/end", "session/end-seed":
			return true
		default:
			return false
		}
	}
	if len(raw) <= 512 {
		return true
	}
	return bytes.Contains(raw, []byte("step/")) ||
		bytes.Contains(raw, []byte("assistant/")) ||
		bytes.Contains(raw, []byte("llm/retry")) ||
		bytes.Contains(raw, []byte("compaction")) ||
		bytes.Contains(raw, []byte("session/end")) ||
		bytes.Contains(raw, []byte(`\u`))
}

func (s *Store) trajectoryRequestsLocked() []TrajectoryRequest {
	return cloneTrajectoryRequests(s.trajectoryRequestsViewLocked(), false)
}

func (s *Store) trajectoryRequestsViewLocked() []TrajectoryRequest {
	fingerprint := s.trajectoryFingerprintLocked()
	journal := s.trajJournalCache
	if s.trajEventsCache == nil {
		s.trajEventsCache = make(map[float64]trajectoryRequestCacheEntry, len(s.trajectoryBoundaries)+len(s.messages))
	}
	if s.trajCacheRev != fingerprint || journal == nil {
		journal = make(map[float64]json.RawMessage, len(s.trajectoryBoundaries)+len(s.messages))
		for seq, raw := range s.trajectoryBoundaries {
			journal[seq] = raw
			if !isTrajectoryRequestCandidate(raw) {
				continue
			}
			if entry, cached := s.trajEventsCache[seq]; !cached || !bytes.Equal([]byte(entry.raw), raw) {
				var parsed trajectoryRequestEvent
				var ok bool
				if json.Unmarshal(raw, &parsed) == nil {
					ok = true
				}
				s.trajEventsCache[seq] = trajectoryRequestCacheEntry{raw: string(raw), event: parsed, ok: ok}
			}
		}
		for _, msg := range s.messages {
			if len(msg.Raw) == 0 || !isTrajectoryRequestCandidate(msg.Raw) {
				continue
			}
			if msg.Seq > 0 {
				if entry, cached := s.trajEventsCache[msg.Seq]; cached && bytes.Equal([]byte(entry.raw), msg.Raw) {
					if entry.ok && entry.event.Seq > 0 {
						journal[entry.event.Seq] = msg.Raw
					}
					continue
				}
			}
			var ev trajectoryRequestEvent
			if json.Unmarshal(msg.Raw, &ev) == nil && ev.Seq > 0 {
				journal[ev.Seq] = msg.Raw
				s.trajEventsCache[ev.Seq] = trajectoryRequestCacheEntry{raw: string(msg.Raw), event: ev, ok: true}
			}
		}
		s.trajCacheRev = fingerprint
		s.trajJournalCache = journal
		s.trajRequestsCache = nil
	}
	headers := s.trajHeadersCache
	if headers == nil || !s.boundariesVerifiedLocked() {
		headers = s.trajectoryRequestHeadersLocked()
		s.trajRequestsCache = nil
	}
	if s.trajRequestsCache == nil {
		seqs := make([]float64, 0, len(journal))
		for seq := range journal {
			seqs = append(seqs, seq)
		}
		sort.Float64s(seqs)
		out := []TrajectoryRequest{}
		byID := map[string]int{}
		for _, seq := range seqs {
			raw := journal[seq]
			entry, ok := s.trajEventsCache[seq]
			if !ok || !entry.ok {
				continue
			}
			e := entry.event
			d := e.Data
			id := ""
			switch e.Type {
			case "step/start", "step/end", "assistant/message", "assistant/attempt", "llm/retry", "llm/retry-started":
				id = trajectoryStepID(d.Turn, d.Step)
			case "compaction/start", "compaction/summary", "compaction/end":
				if d.CompactionID != "" {
					id = "compaction:" + d.CompactionID
				}
			case "user/message":
				if d.Source.Kind == "compact-checkpoint" && d.Source.CompactionID != "" {
					id = "compaction:" + d.Source.CompactionID
				}
			case "session/end":
				for i := range out {
					if out[i].Purpose == "compaction" && out[i].Status == "running" {
						r := &out[i]
						r.Status = "error"
						r.CompletedAt = trajectoryTimestamp(e.Time)
						reason := "compaction interrupted by session/end"
						r.Error = &reason
						r.Events = append(r.Events, append(json.RawMessage(nil), raw...))
					}
				}
			case "session/end-seed":
				// The original seed boundary interrupts the latest unfinished
				// compaction preceding it; earlier unrelated requests keep their
				// own recorded outcomes.
				for i := len(out) - 1; i >= 0; i-- {
					if out[i].Purpose == "compaction" && out[i].Status == "running" {
						r := &out[i]
						r.Status, r.CompletedAt = "error", trajectoryTimestamp(e.Time)
						reason := "trajectory.compaction-interrupted"
						r.Error = &reason
						r.Events = append(r.Events, append(json.RawMessage(nil), raw...))
						break
					}
				}
			}
			if id == "" {
				continue
			}
			index, found := byID[id]
			if e.Type == "step/start" || e.Type == "compaction/start" {
				if !found {
					index = len(out)
					byID[id] = index
					purpose := "assistant"
					if e.Type == "compaction/start" {
						purpose = "compaction"
					}
					out = append(out, TrajectoryRequest{ID: id, Purpose: purpose, Turn: d.Turn, Step: d.Step, CompactionID: d.CompactionID, StartSeq: seq, StartedAt: trajectoryTimestamp(e.Time), Status: "running"})
					found = true
				}
			}
			// A partial window cannot invent the start, request identity or timestamps.
			if !found {
				continue
			}
			r := &out[index]
			r.Events = append(r.Events, append(json.RawMessage(nil), raw...))
			switch e.Type {
			case "assistant/attempt":
				r.Attempts = append(r.Attempts, TrajectoryAttemptEvidence{Seq: seq, Type: e.Type, Raw: append(json.RawMessage(nil), raw...)})
			case "assistant/message":
				r.Status = "complete"
				if d.Interrupted {
					r.Status = "error"
				}
				r.ResultSeq = &e.Seq
				r.CompletedAt = trajectoryTimestamp(e.Time)
				if d.Message.Source.Model != "" || d.Message.Source.Provider != "" {
					source := d.Message.Source
					r.ProviderMetadata = &source
				}
				if len(d.Usage) > 0 && string(d.Usage) != "null" {
					r.Usage = append(json.RawMessage(nil), d.Usage...)
				}
			case "step/end":
				if r.ResultSeq == nil {
					r.Status = "error"
					r.CompletedAt = trajectoryTimestamp(e.Time)
				}
			case "llm/retry":
				var envelope struct {
					Data RetryAttempt `json:"data"`
				}
				if json.Unmarshal(raw, &envelope) == nil && envelope.Data.RetryID != "" && envelope.Data.Retry > 0 {
					retry := envelope.Data
					retry.Seq = seq
					retry.State = "scheduled"
					r.Retries = append(r.Retries, retry)
				}
			case "llm/retry-started":
				var envelope struct {
					Data RetryAttempt `json:"data"`
				}
				if json.Unmarshal(raw, &envelope) == nil {
					for i := range r.Retries {
						retry := &r.Retries[i]
						if retry.RetryID == envelope.Data.RetryID && retry.Retry == envelope.Data.Retry && retry.State == "scheduled" {
							retry.State = "started"
						}
					}
				}
			case "compaction/summary":
				r.ResultSeq = &e.Seq
				r.Summary = trajectorySummaryText(d.Summary)
				r.SummaryBlocks = append(json.RawMessage(nil), d.Summary...)
				if len(d.RawOutput) > 0 && string(d.RawOutput) != "null" {
					text := trajectorySummaryText(d.RawOutput)
					r.RawOutput = &text
					r.RawOutputBlocks = append(json.RawMessage(nil), d.RawOutput...)
				}
				if d.Provider != "" || d.Model != "" {
					r.ProviderMetadata = &TrajectoryProviderMetadata{Provider: d.Provider, Model: d.Model}
					config := map[string]any{"provider": d.Provider, "model": d.Model, "purpose": "compaction"}
					if d.MaxTokens != nil {
						config["maxTokens"] = *d.MaxTokens
					}
					r.RequestConfig, _ = json.Marshal(config)
				}
				if len(d.Usage) > 0 && string(d.Usage) != "null" {
					r.Usage = append(json.RawMessage(nil), d.Usage...)
				}
			case "compaction/end":
				r.Status = "complete"
				r.CompletedAt = trajectoryTimestamp(e.Time)
				r.Error = d.Error
				if d.Error != nil {
					r.Status = "error"
				}
			case "user/message":
				r.ReplacementSeq = &e.Seq
			}
		}
		for i := range out {
			r := &out[i]
			if r.Status != "running" {
				for j := range r.Retries {
					if r.Retries[j].State == "scheduled" {
						r.Retries[j].State = "cancelled"
					}
				}
			}
			if r.Purpose != "assistant" {
				continue
			}
			upper := float64(9007199254740991)
			if r.ResultSeq != nil {
				upper = *r.ResultSeq
			} else {
				for _, next := range out {
					if next.Purpose == "assistant" && next.StartSeq > r.StartSeq && next.StartSeq < upper {
						upper = next.StartSeq - 0.5
					}
				}
			}
			for j := range headers {
				h := &headers[j]
				if h.Seq <= upper && (r.Header == nil || h.Seq > r.Header.Seq) {
					copy := *h
					r.Header = &copy
				}
			}
			if r.Header != nil && r.Header.Header.Config != nil {
				r.RequestConfig, _ = json.Marshal(r.Header.Header.Config)
			}
		}
		s.trajRequestsCache = cloneTrajectoryRequests(out, true)
	}
	a := s.assistantStream.active
	if a == nil {
		return cloneSlice(s.trajRequestsCache)
	}
	targetID := trajectoryStepID(&a.Turn, &a.Step)
	out := append([]TrajectoryRequest(nil), s.trajRequestsCache...)
	for i := range out {
		r := &out[i]
		if r.Purpose == "assistant" && r.Status == "running" && targetID == r.ID {
			stream := make([]json.RawMessage, len(a.Stream))
			for j := range a.Stream {
				stream[j] = append(json.RawMessage(nil), a.Stream[j]...)
			}
			r.LiveAttempt = &TrajectoryLiveAttempt{AttemptID: a.AttemptID, StartedAfterSeq: a.StartedAfterSeq, NextIndex: a.NextIndex, Stream: stream, Usage: a.currentLiveUsage()}
		}
	}
	return out
}

// Compaction persists ContentBlock[], including reasoning in rawOutput. Keep
// the original blocks for inspection while projecting readable text for the
// compact ledger. Existing string-valued history remains readable as well.
func trajectorySummaryText(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var blocks []struct{ Type, Text string }
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, block := range blocks {
		if (block.Type == "text" || block.Type == "reasoning") && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func (a *assistantStreamAttempt) currentLiveUsage() map[string]float64 {
	if a == nil {
		return nil
	}
	if a.liveUsageStreamLen > len(a.Stream) {
		a.liveUsage = nil
		a.liveUsageStreamLen = 0
		a.liveUsageInvalid = false
	}
	if !a.liveUsageInvalid && a.liveUsageStreamLen < len(a.Stream) {
		chunks, ok := expandAssistantStream(a.Stream[a.liveUsageStreamLen:])
		if !ok {
			a.liveUsageInvalid = true
			a.liveUsage = nil
		} else {
			addTrajectoryLiveUsage(&a.liveUsage, chunks)
			a.liveUsageStreamLen = len(a.Stream)
		}
	}
	if a.liveUsageInvalid || len(a.liveUsage) == 0 {
		return nil
	}
	out := make(map[string]float64, len(a.liveUsage))
	for k, v := range a.liveUsage {
		out[k] = v
	}
	return out
}

// Usage chunks are additive within this exact active attempt. Missing fields
// remain absent; this is not a fabricated aggregate across previous retries.
func trajectoryLiveUsage(stream []json.RawMessage) map[string]float64 {
	var result map[string]float64
	chunks, _ := expandAssistantStream(stream)
	addTrajectoryLiveUsage(&result, chunks)
	return result
}

func addTrajectoryLiveUsage(result *map[string]float64, chunks []assistantTimedChunk) {
	for _, item := range chunks {
		var chunk struct {
			Type  string             `json:"type"`
			Usage map[string]float64 `json:"usage"`
		}
		if json.Unmarshal(item.chunk, &chunk) != nil || chunk.Type != "usage" {
			continue
		}
		for _, key := range []string{"inputTokens", "outputTokens", "cacheReadTokens", "cacheWriteTokens", "reasoningTokens"} {
			value, ok := chunk.Usage[key]
			if !ok || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				continue
			}
			if *result == nil {
				*result = map[string]float64{}
			}
			(*result)[key] += value
		}
	}
}

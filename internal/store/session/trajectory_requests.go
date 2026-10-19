package session

import (
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

func (s *Store) trajectoryRequestsLocked() []TrajectoryRequest {
	journal := map[float64]json.RawMessage{}
	for seq, raw := range s.trajectoryBoundaries {
		journal[seq] = raw
	}
	for _, msg := range s.messages {
		var ev trajectoryRequestEvent
		if json.Unmarshal(msg.Raw, &ev) == nil && ev.Seq > 0 {
			journal[ev.Seq] = msg.Raw
		}
	}
	seqs := make([]float64, 0, len(journal))
	for seq := range journal {
		seqs = append(seqs, seq)
	}
	sort.Float64s(seqs)
	out := []TrajectoryRequest{}
	byID := map[string]int{}
	headers := s.trajectoryRequestHeadersLocked()
	for _, seq := range seqs {
		raw := journal[seq]
		var e trajectoryRequestEvent
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
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
		a := s.assistantStream.active
		if r.Status == "running" && a != nil && trajectoryStepID(&a.Turn, &a.Step) == r.ID {
			stream := make([]json.RawMessage, len(a.Stream))
			for j := range a.Stream {
				stream[j] = append(json.RawMessage(nil), a.Stream[j]...)
			}
			r.LiveAttempt = &TrajectoryLiveAttempt{AttemptID: a.AttemptID, StartedAfterSeq: a.StartedAfterSeq, NextIndex: a.NextIndex, Stream: stream, Usage: trajectoryLiveUsage(stream)}
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

// Usage chunks are additive within this exact active attempt. Missing fields
// remain absent; this is not a fabricated aggregate across previous retries.
func trajectoryLiveUsage(stream []json.RawMessage) map[string]float64 {
	var result map[string]float64
	for _, raw := range stream {
		var chunk struct {
			Type  string             `json:"type"`
			Usage map[string]float64 `json:"usage"`
		}
		if json.Unmarshal(raw, &chunk) != nil || chunk.Type != "usage" {
			continue
		}
		for _, key := range []string{"inputTokens", "outputTokens", "cacheReadTokens", "cacheWriteTokens", "reasoningTokens"} {
			value, ok := chunk.Usage[key]
			if !ok || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
				continue
			}
			if result == nil {
				result = map[string]float64{}
			}
			result[key] += value
		}
	}
	return result
}

package session

import (
	"encoding/json"
	"sort"
	"strings"
	"whaleisle/internal/dsh/events"
)

// RequestInspectionHeader is an allowlist of the actual persisted model request
// configuration and tool declarations. It never retains unknown adapter/auth fields.
type RequestInspectionHeader struct {
	Config *events.RequestHeaderDataHeaderConfig     `json:"config,omitempty"`
	Tools  []events.RequestHeaderDataHeaderToolsItem `json:"tools"`
}
type TrajectoryRequestHeader struct {
	Seq    float64                 `json:"seq"`
	Reason string                  `json:"reason"`
	Header RequestInspectionHeader `json:"header"`
}

func sanitizeRequestInspectionHeader(h *RequestInspectionHeader) *RequestInspectionHeader {
	if h == nil {
		return nil
	}
	if h.Config != nil && (strings.TrimSpace(h.Config.Model) == "" || strings.TrimSpace(h.Config.Provider) == "") {
		h.Config = nil
	}
	tools := make([]events.RequestHeaderDataHeaderToolsItem, 0, len(h.Tools))
	for _, tool := range h.Tools {
		if strings.TrimSpace(tool.Name) != "" && tool.Parameters != nil {
			tools = append(tools, tool)
		}
	}
	h.Tools = tools
	return h
}
func (s *Store) trajectoryRequestHeadersLocked() []TrajectoryRequestHeader {
	result := []TrajectoryRequestHeader{}
	for _, raw := range s.trajectoryBoundaries {
		var ev systemPromptEvent
		if json.Unmarshal(raw, &ev) != nil || ev.Type != "request/header" || ev.Data.Header == nil {
			continue
		}
		result = append(result, TrajectoryRequestHeader{Seq: ev.Seq, Reason: ev.Data.Reason, Header: *ev.Data.Header})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Seq < result[j].Seq })
	return result
}

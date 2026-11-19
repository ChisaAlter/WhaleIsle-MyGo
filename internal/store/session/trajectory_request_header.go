package session

import (
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
	for seq, raw := range s.trajectoryBoundaries {
		if typ, ok := peekEventType(raw); ok && typ != "request/header" {
			continue
		}
		ev, ok := s.promptEventLocked(seq, raw)
		if !ok || ev.Type != "request/header" || ev.Data.Header == nil {
			continue
		}
		// The decode is shared with the system-prompt rows, so the header handed
		// out is a copy: a caller that edits a schema must not edit the cache.
		result = append(result, cloneTrajectoryRequestHeader(TrajectoryRequestHeader{Seq: ev.Seq, Reason: ev.Data.Reason, Header: *ev.Data.Header}, true))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Seq < result[j].Seq })
	return result
}

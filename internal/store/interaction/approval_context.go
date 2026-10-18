package interaction

import (
	"bytes"
	"encoding/json"
	"strings"
)

// approvalRequester is the Host-read delegated call. It must never be replaced
// by a similarly named call in the presenting parent's transcript.
type approvalRequester struct {
	SessionID string `json:"sessionId"`
	Label     string `json:"label"`
	Cwd       string `json:"cwd"`
	Call      struct {
		CallID    string `json:"callId"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"call"`
}

func approvalArgsPreview(req approvalRequest) string {
	if req.Requester != nil {
		if req.CallID == "" || req.Requester.Call.CallID != req.CallID {
			return ""
		}
		return formatApprovalArgs(json.RawMessage(req.Requester.Call.Arguments))
	}
	for _, raw := range []json.RawMessage{req.Input, req.Args, req.Parameters} {
		if result := formatApprovalArgs(raw); result != "" {
			return result
		}
	}
	if req.Command != "" {
		return req.Command
	}
	return req.Description
}

// Preserve every argument byte for inspection and copying. The UI bounds the
// viewport, never the value; JSON strings from tool/call are decoded once.
func formatApprovalArgs(raw json.RawMessage) string {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return ""
	}
	var encoded string
	if json.Unmarshal(raw, &encoded) == nil {
		raw = json.RawMessage(encoded)
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, raw, "", "  ") == nil {
		return pretty.String()
	}
	return string(raw)
}

func approvalRequesterContext(req approvalRequest) string {
	r := req.Requester
	if r == nil {
		return ""
	}
	if req.CallID == "" || r.Call.CallID != req.CallID {
		return "Requester call identity does not match; original arguments unavailable."
	}
	cwd := r.Cwd
	var args struct {
		Workdir string `json:"workdir"`
	}
	if json.Unmarshal([]byte(r.Call.Arguments), &args) == nil && args.Workdir != "" {
		cwd = approvalWorkingDirectory(cwd, args.Workdir)
	}
	var lines []string
	if r.Label != "" {
		lines = append(lines, "Requester: "+r.Label)
	}
	if r.SessionID != "" {
		lines = append(lines, "Session: "+r.SessionID)
	}
	if cwd != "" {
		lines = append(lines, "Working directory: "+cwd)
	}
	return strings.Join(lines, "\n")
}

// Display-only counterpart of the original portable resolveWorkspacePath. Do
// not clean, resolve, open or execute Host paths on this client's filesystem.
func approvalWorkingDirectory(cwd, workdir string) string {
	windows := func(s string) bool {
		return strings.HasPrefix(s, `\\`) || (len(s) >= 3 && ((s[0] >= 'A' && s[0] <= 'Z') || (s[0] >= 'a' && s[0] <= 'z')) && s[1] == ':' && (s[2] == '/' || s[2] == '\\'))
	}
	if workdir == "" {
		return cwd
	}
	if cwd == "" || strings.HasPrefix(workdir, "/") || windows(workdir) {
		return workdir
	}
	sep := "/"
	if windows(cwd) && strings.Contains(cwd, `\`) {
		sep = `\`
	}
	return strings.TrimRight(cwd, `/\`) + sep + strings.TrimLeft(workdir, `/\`)
}

// SyncApprovalCalls correlates pending approvals with fully loaded start
// records from the same session. It is presentation-only and cannot resolve an
// approval, change policy or substitute a parent's call for a delegated call.
func (s *Store) SyncApprovalCalls(agentID string, events []json.RawMessage) {
	if agentID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	wanted := map[string]bool{}
	for _, p := range s.pending {
		if p.AgentID == agentID && p.Event == EventApproval && !p.approvalRequester && !p.approvalResolved && p.approvalCallID != "" {
			wanted[p.approvalCallID] = true
		}
	}
	if len(wanted) == 0 {
		return
	}
	calls := map[string]string{}
	for _, raw := range events {
		var e struct {
			Type string `json:"type"`
			Data struct {
				CallID    string          `json:"callId"`
				SubCallID string          `json:"subCallId"`
				Arguments json.RawMessage `json:"arguments"`
			} `json:"data"`
		}
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		id := e.Data.CallID
		switch e.Type {
		case "tool/call":
		case "tool/ptc-dispatch-start":
			id = e.Data.SubCallID
		default:
			continue
		}
		if !wanted[id] {
			continue
		}
		if _, exists := calls[id]; exists {
			continue
		}
		if args := formatApprovalArgs(e.Data.Arguments); args != "" {
			calls[id] = args
		}
	}
	for i := range s.pending {
		p := &s.pending[i]
		if p.AgentID != agentID || p.Event != EventApproval || p.approvalRequester || p.approvalResolved {
			continue
		}
		if args, ok := calls[p.approvalCallID]; ok {
			p.Args, p.approvalResolved = args, true
		}
	}
}

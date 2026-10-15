package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"whaleisle/internal/dsh"
	"whaleisle/internal/projectflow"
)

// workflowMemberStream borrows the existing, verified assistant presentation
// reducer in a transport-free Store. It never shares the parent's messages,
// attempts, cursors, composer, client or mux. Process alone is durable history.
type workflowMemberStream struct {
	process      projectflow.ProcessSnapshot
	presentation *Store
	floor        float64
}

func newWorkflowMemberStream() *workflowMemberStream {
	return &workflowMemberStream{presentation: New()}
}
func (p *workflowMemberStream) apply(raw json.RawMessage) (rebaseline bool, err error) {
	var head struct {
		Type  string `json:"type"`
		Event struct {
			Seq float64 `json:"seq"`
		} `json:"event"`
	}
	if err = json.Unmarshal(raw, &head); err != nil {
		return false, err
	}
	if p.process.Type != "snapshot" && head.Type != "snapshot" {
		return true, nil
	}
	// Old journal duplicates outside the bounded display window cannot revive an
	// omitted row. Within the window the canonical reducer owns duplicate and
	// pending-settlement checks, including duplicate-pending rebaseline.
	if head.Type == "event" && p.floor > 0 && head.Event.Seq < p.floor {
		return false, nil
	}
	next, err := projectflow.FoldProcessFrame(p.process, raw)
	if err != nil {
		return false, err
	}
	p.presentation.applyFollowFrameLocked(raw)
	if p.presentation.assistantStream.rebaseline {
		return true, nil
	}
	p.process = next
	p.prune()
	return false, nil
}
func (p *workflowMemberStream) prune() {
	floor := float64(0)
	for _, record := range p.process.Records {
		if record.Type == "event" && (floor == 0 || record.Event.Seq < floor) {
			floor = record.Event.Seq
		}
	}
	p.floor = floor
	if floor <= 0 {
		return
	}
	s := p.presentation
	kept := s.messages[:0]
	for _, message := range s.messages {
		if message.Streaming || message.Seq >= floor {
			kept = append(kept, message)
		}
	}
	s.messages = kept
	for seq := range s.assistantStream.published {
		if seq < floor {
			delete(s.assistantStream.published, seq)
		}
	}
	for seq := range s.workflowBoundaries {
		if seq < floor {
			delete(s.workflowBoundaries, seq)
		}
	}
	for seq := range s.trajectoryBoundaries {
		if seq < floor {
			delete(s.trajectoryBoundaries, seq)
		}
	}
}
func (p *workflowMemberStream) messages() []Message {
	s := p.presentation
	messages := projectCommandMessages(s.projectAssistantMessagesLocked(projectMessageEdits(s.messages)))
	return s.projectPreparingToolsLocked(messages)
}
func workflowMemberFollowPayload(address map[string]any) map[string]any {
	return map[string]any{"args": map[string]any{"request": map[string]any{"address": address, "maxMessages": 30, "assistantStream": true}}}
}

// followWorkflowMember reopens only its own logical follow on a reducer fault.
// A fresh opening snapshot restores the attempt prefix/revision/index; no suffix
// is guessed and no model/tool execution is requested by this repair.
func (s *Store) followWorkflowMember(ctx context.Context, token uint64, mux *dsh.Mux, address map[string]any) {
	for {
		if err := mux.Dial(ctx); err != nil {
			s.finishWorkflowMember(ctx, token, err)
			return
		}
		stream, err := mux.Open(ctx, "session/follow", workflowMemberFollowPayload(address))
		if err != nil {
			s.finishWorkflowMember(ctx, token, err)
			return
		}
		reducer := newWorkflowMemberStream()
		restart := false
		for {
			raw, recvErr := stream.Recv(ctx)
			if recvErr != nil {
				err = recvErr
				break
			}
			restart, err = reducer.apply(raw)
			if err != nil || restart {
				break
			}
			s.mu.Lock()
			if !s.workflowMemberCurrentLocked(ctx, token) {
				s.mu.Unlock()
				stream.Cancel()
				return
			}
			w := &s.workflowMember
			notify := !w.notifyPending
			w.notifyPending = true
			w.view.Process = reducer.process
			w.view.Messages = reducer.messages()
			w.view.PresentationReady = true
			w.view.Busy = false
			s.mu.Unlock()
			if notify {
				s.fire()
			}
		}
		stream.Cancel()
		if !restart {
			s.finishWorkflowMember(ctx, token, err)
			return
		}
		s.mu.Lock()
		if !s.workflowMemberCurrentLocked(ctx, token) {
			s.mu.Unlock()
			return
		}
		w := &s.workflowMember
		w.view.Process.Live = false
		w.view.Process.Error = "实时序列不连续，正在重新同步… / Live sequence changed; resynchronizing…"
		w.view.Busy = true
		notify := !w.notifyPending
		w.notifyPending = true
		s.mu.Unlock()
		if notify {
			s.fire()
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// WorkflowMemberDisplayText renders the canonical transient-or-settled view.
// Streaming rows are labelled and never contribute to Process or page cursors.
func WorkflowMemberDisplayText(v WorkflowMemberView) string {
	if !v.PresentationReady {
		return WorkflowProcessText(v)
	}
	var out strings.Builder
	for _, message := range v.Messages {
		label := message.Role
		if label == "" {
			label = string(message.Kind)
		}
		if message.Streaming {
			label += " · 实时临时输出 / live preview"
		} else if message.Seq > 0 {
			label += fmt.Sprintf(" #%.0f", message.Seq)
		}
		body := message.Body
		if message.Tool != nil {
			body = message.Tool.Name + " · " + message.Tool.State
			if message.Tool.Arguments != "" {
				body += "\n" + message.Tool.Arguments
			}
		} else if message.Args != "" && message.Args != body {
			body += "\n" + message.Args
		}
		if body != "" {
			fmt.Fprintf(&out, "[%s]\n%s\n\n", label, body)
		}
	}
	return out.String()
}

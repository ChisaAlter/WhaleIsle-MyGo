package session

import (
	"encoding/json"
	"fmt"
	"whaleisle/internal/projectflow"
)

// ProjectAgentNavigationHistory validates a fixed-cut durable page and projects
// its original raw records through the isolated child reducer. It has no
// transport, selection, permissions or attachment resolver. Do not remarshal
// ProcessSnapshot to obtain Messages: that would discard event metadata.
func ProjectAgentNavigationHistory(raw json.RawMessage, through, before float64) (projectflow.ProcessSnapshot, []Message, string, error) {
	var page projectflow.ProcessSnapshot
	if err := json.Unmarshal(raw, &page); err != nil {
		return page, nil, "", err
	}
	if page.Records == nil {
		return page, nil, "", fmt.Errorf("Invalid child history records")
	}
	for _, r := range page.Records {
		if r.Type != "event" || r.Event.Seq <= 0 || r.Event.Seq > through || (before > 0 && r.Event.Seq >= before) {
			return projectflow.ProcessSnapshot{}, nil, "", fmt.Errorf("Child history page crossed its fixed cursor")
		}
	}
	if len(page.Records) == 0 && page.HasMore {
		return projectflow.ProcessSnapshot{}, nil, "", fmt.Errorf("Child history made no backward progress")
	}
	// Reuse the verified, transport-free history/assistant reducer. This does not
	// open the Workflow observer or borrow any of its state or main-chat cursors.
	reducer := newWorkflowMemberStream()
	restart, err := reducer.apply(raw)
	if err != nil {
		return projectflow.ProcessSnapshot{}, nil, "", err
	}
	if restart {
		return projectflow.ProcessSnapshot{}, nil, "", fmt.Errorf("Invalid child history presentation")
	}
	process := reducer.process
	process.Live = false
	messages := reducer.messages()
	body := WorkflowMemberDisplayText(WorkflowMemberView{Process: process, Messages: messages, PresentationReady: true})
	if body == "" {
		body = WorkflowProcessText(WorkflowMemberView{Process: process})
	}
	return process, messages, body, nil
}

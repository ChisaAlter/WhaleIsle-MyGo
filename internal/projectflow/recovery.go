package projectflow

import (
	"context"
	"fmt"
)

// Settlement identifies an already-recorded native terminal outcome. It never
// requests a new activation or changes the worker's saved state.
type Settlement struct {
	RunID          string   `json:"runId"`
	DelegationRefs []string `json:"delegationRefs"`
	At             float64  `json:"at"`
	StopReason     string   `json:"stopReason"`
}

func recoveryProject(selected, current Project) error {
	if selected.ID == "" || selected.CoordinatorSessionID == "" || current.ID != selected.ID || current.CoordinatorSessionID != selected.CoordinatorSessionID {
		return fmt.Errorf("the original Project Lead changed; refresh before preparing a recovery draft")
	}
	if current.Lifecycle != "ready" || current.Archiving {
		return fmt.Errorf("restore the Project before preparing a recovery draft")
	}
	return nil
}

// PrepareSummaryRetry re-reads the Project, work and saved request before it
// returns text for the original Lead composer. No prompt or retry RPC is sent.
func (c Client) PrepareSummaryRetry(ctx context.Context, selected Project, work WorkDetail) (string, error) {
	current, err := c.LifecycleProject(ctx, selected.ID)
	if err != nil {
		return "", err
	}
	if err = recoveryProject(selected, current); err != nil {
		return "", err
	}
	fresh, err := c.WorkDetail(ctx, current.ID, work.Workstream.ID, "")
	if err != nil {
		return "", err
	}
	if request := fresh.Workstream.Request; request.ID != "" {
		card, err := c.Request(ctx, current.ID, request.ID)
		if err != nil {
			return "", err
		}
		if card.Revision != request.Revision || card.AuthorityError != "" {
			return "", fmt.Errorf("the saved request changed or its original authorization is unavailable; review it in the Lead conversation")
		}
	}
	return SummaryRetryDraft(current, work, fresh)
}

func SummaryRetryDraft(project Project, selected, current WorkDetail) (string, error) {
	if err := recoveryProject(project, project); err != nil {
		return "", err
	}
	before, failure := selected.Workstream.SummaryFailure, current.Workstream.SummaryFailure
	if selected.Workstream.ProjectID != project.ID || current.Workstream.ProjectID != project.ID ||
		selected.Workstream.ID == "" || current.Workstream.ID != selected.Workstream.ID ||
		selected.Workstream.WorkerSessionID == "" || current.Workstream.WorkerSessionID != selected.Workstream.WorkerSessionID ||
		selected.Workstream.Request.ID != current.Workstream.Request.ID || selected.Workstream.Request.Revision != current.Workstream.Request.Revision ||
		current.Workstream.Request.ID != "" && current.Workstream.Request.Revision <= 0 ||
		before == nil || failure == nil || before.DelegationRef == "" || *before != *failure ||
		current.Workstream.CurrentDelegationRef != failure.DelegationRef {
		return "", fmt.Errorf("the saved summary failure changed; refresh the original work before preparing a draft")
	}
	if current.Workstream.Request.AuthorityError != "" {
		return "", fmt.Errorf("the saved request has no verified original authorization; revise it in the Lead conversation")
	}
	evidence := ""
	if report := current.Workstream.LatestReport; report != nil && report.DelegationRef == failure.DelegationRef {
		evidence = fmt.Sprintf("saved report %s (recorded at %g)", report.CallID, report.At)
	}
	if evidence == "" && failure.RunID != "" {
		for _, terminal := range current.Workstream.RecentSettlements {
			if terminal.RunID != failure.RunID {
				continue
			}
			for _, ref := range terminal.DelegationRefs {
				if ref == failure.DelegationRef {
					evidence = fmt.Sprintf("saved terminal outcome %s (recorded at %g)", terminal.RunID, terminal.At)
					break
				}
			}
			if evidence != "" {
				break
			}
		}
	}
	if evidence == "" {
		return "", fmt.Errorf("the original saved report or terminal outcome is unavailable; refresh before preparing a summary draft")
	}
	return fmt.Sprintf("Summarize the following saved result again, including changes, verification and remaining issues. Read the existing records without rerunning background work, creating a replacement member or expanding the authorized scope.\nProject: %s\nWork: %s\nworkstreamId: %s\nOriginal member session: %s\nDelegation: %s\nEvidence: %s", project.ID, current.Workstream.Title, current.Workstream.ID, current.Workstream.WorkerSessionID, failure.DelegationRef, evidence), nil
}

// PrepareHandoverRetry verifies the same saved request revision and failure.
func (c Client) PrepareHandoverRetry(ctx context.Context, selected Project, card RequestCard) (string, error) {
	current, err := c.LifecycleProject(ctx, selected.ID)
	if err != nil {
		return "", err
	}
	if err = recoveryProject(selected, current); err != nil {
		return "", err
	}
	fresh, err := c.Request(ctx, current.ID, card.ID)
	if err != nil {
		return "", err
	}
	return HandoverRetryDraft(current, card, fresh)
}

func HandoverRetryDraft(project Project, selected, current RequestCard) (string, error) {
	if err := recoveryProject(project, project); err != nil {
		return "", err
	}
	if selected.ID == "" || selected.ProjectID != project.ID || current.ProjectID != project.ID || current.ID != selected.ID ||
		selected.Revision <= 0 || current.Revision != selected.Revision || selected.HandoverError == "" ||
		current.HandoverError != selected.HandoverError || current.Status == "completed" {
		return "", fmt.Errorf("the saved handover changed; refresh the original user request before preparing a draft")
	}
	if current.AuthorityError != "" {
		return "", fmt.Errorf("the saved request has no verified original authorization; revise it in the Lead conversation")
	}
	return fmt.Sprintf("Summarize the saved results for this user request again, including changes, verification and remaining issues. Read the existing reports without rerunning background work, creating replacement members or expanding the authorized scope.\nProject: %s\nRequest: %s\nrequestId: %s\nRevision: %d\nGoal: %s", project.ID, current.Title, current.ID, current.Revision, current.Goal), nil
}

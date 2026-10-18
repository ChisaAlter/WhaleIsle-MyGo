package projectflow

import (
	"context"
	"fmt"
)

// RequestCard is a user requirement and its delivery, not an agentTeam task.
type RequestCard struct {
	ID             string   `json:"id"`
	ProjectID      string   `json:"projectId"`
	Title          string   `json:"title"`
	Revision       int      `json:"revision"`
	Status         string   `json:"status"`
	Goal           string   `json:"goal"`
	GoalPreview    string   `json:"goalPreview"`
	ArchivedAt     *float64 `json:"archivedAt"`
	Conditions     []string `json:"conditions"`
	Constraints    []string `json:"constraints"`
	AuthorityError string   `json:"authorityError"`
	HandoverError  string   `json:"handoverError"`
	Delivery       *struct {
		Artifacts       []Artifact `json:"artifacts"`
		ID              string     `json:"id"`
		Outcome         string     `json:"outcome"`
		Applicable      bool       `json:"applicable"`
		Summary         string     `json:"summary"`
		SummaryPreview  string     `json:"summaryPreview"`
		Evidence        []string   `json:"evidence"`
		RemainingIssues []string   `json:"remainingIssues"`
	} `json:"delivery"`
	Versions []struct {
		Revision int    `json:"revision"`
		Title    string `json:"title"`
		Goal     string `json:"goal"`
	} `json:"versions"`
	Work []struct {
		WorkstreamID    string `json:"workstreamId"`
		DelegationRef   string `json:"delegationRef"`
		Title           string `json:"title"`
		Status          string `json:"status"`
		MemberName      string `json:"memberName"`
		WorkerSessionID string `json:"workerSessionId"`
		Current         bool   `json:"current"`
		Summary         string `json:"summary"`
		BlockedReason   string `json:"blockedReason"`
	} `json:"work"`
}
type RequestPage struct {
	Items      []RequestCard `json:"items"`
	Total      int           `json:"total"`
	NextCursor *string       `json:"nextCursor"`
}

func (c Client) Requests(ctx context.Context, projectID, kind, cursor string) (RequestPage, error) {
	var out RequestPage
	if projectID == "" || (kind != "current" && kind != "completed" && kind != "archived") {
		return out, fmt.Errorf("select current, completed or archived user requests")
	}
	payload := map[string]any{"projectId": projectID, "kind": kind}
	if cursor != "" {
		payload["cursor"] = cursor
	}
	err := c.call(ctx, "requests/list", payload, &out)
	return out, err
}
func (c Client) Request(ctx context.Context, projectID, requestID string) (RequestCard, error) {
	var out struct {
		Card RequestCard `json:"card"`
	}
	if projectID == "" || requestID == "" {
		return out.Card, fmt.Errorf("select a user request")
	}
	err := c.call(ctx, "requests/get", map[string]any{"projectId": projectID, "requestId": requestID}, &out)
	if err == nil && (out.Card.ID != requestID || out.Card.ProjectID != projectID) {
		err = fmt.Errorf("user request identity does not match")
	}
	return out.Card, err
}

// The Host additionally checks current delivery applicability, active work and
// project lifecycle atomically; a local completed badge never overrides it.
func (c Client) SetRequestArchived(ctx context.Context, projectID, requestID string, archived bool) error {
	fresh, err := c.Request(ctx, projectID, requestID)
	if err != nil {
		return err
	}
	method := "requests/restore"
	if archived {
		if fresh.Status != "completed" {
			return fmt.Errorf("only a completed and current user request can be archived")
		}
		method = "requests/archive"
	}
	return c.call(ctx, method, map[string]any{"projectId": projectID, "requestId": requestID}, nil)
}

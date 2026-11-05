package projectflow

import (
	"context"
	"fmt"
)

// RequestCard is a user requirement and its delivery, not an agentTeam task.
type RequestCard struct {
	ID             string                `json:"id"`
	ProjectID      string                `json:"projectId"`
	Title          string                `json:"title"`
	Revision       int                   `json:"revision"`
	UpdatedAt      float64               `json:"updatedAt"`
	Status         string                `json:"status"`
	Goal           string                `json:"goal"`
	GoalPreview    string                `json:"goalPreview"`
	ArchivedAt     *float64              `json:"archivedAt"`
	Conditions     []string              `json:"conditions"`
	Constraints    []string              `json:"constraints"`
	AuthorityError string                `json:"authorityError"`
	HandoverError  string                `json:"handoverError"`
	Queue          *RequestQueueEntry    `json:"queue"`
	Changes        *RequestChangeSummary `json:"changes"`
	WorkPreview    []RequestWork         `json:"workPreview"`
	Counts         *RequestWorkCounts    `json:"counts"`
	Delivery       *struct {
		Artifacts       []Artifact              `json:"artifacts"`
		Pages           []DeliveryPageReference `json:"pages"`
		ID              string                  `json:"id"`
		Outcome         string                  `json:"outcome"`
		Applicable      bool                    `json:"applicable"`
		Summary         string                  `json:"summary"`
		SummaryPreview  string                  `json:"summaryPreview"`
		ArtifactCount   int                     `json:"artifactCount"`
		Evidence        []string                `json:"evidence"`
		RemainingIssues []string                `json:"remainingIssues"`
	} `json:"delivery"`
	Versions []struct {
		Revision    int      `json:"revision"`
		Title       string   `json:"title"`
		Goal        string   `json:"goal"`
		Constraints []string `json:"constraints"`
		Conditions  []string `json:"conditions"`
		At          float64  `json:"at"`
	} `json:"versions"`
	Work []RequestWork `json:"work"`
}

type RequestWorkCounts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
}

type RequestWork struct {
	WorkstreamID    string         `json:"workstreamId"`
	DelegationRef   string         `json:"delegationRef"`
	Title           string         `json:"title"`
	Status          string         `json:"status"`
	MemberName      string         `json:"memberName"`
	WorkerSessionID string         `json:"workerSessionId"`
	Current         *bool          `json:"current"`
	Summary         string         `json:"summary"`
	BlockedReason   string         `json:"blockedReason"`
	MemberRole      string         `json:"memberRole"`
	ReportID        string         `json:"reportId"`
	WaitingReason   *WaitingReason `json:"waitingReason"`
}

type RequestQueueEntry struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Revision int    `json:"revision"`
	State    string `json:"state"`
	Released bool   `json:"released"`
	Position int    `json:"position"`
	Total    int    `json:"total"`
}

type RequestQueue struct {
	Active          *RequestCard        `json:"active"`
	Queued          []RequestQueueEntry `json:"queued"`
	Paused          bool                `json:"paused"`
	DiscussingCount int                 `json:"discussingCount"`
	CancelledCount  int                 `json:"cancelledCount"`
}

type RequestQueueChange struct {
	RequestID       string `json:"requestId"`
	RequestRevision int    `json:"requestRevision"`
	Action          string `json:"action"`
	BeforeRequestID string `json:"beforeRequestId,omitempty"`
}

type RequestQueueResult struct {
	RequestID    string            `json:"requestId"`
	Queue        RequestQueueEntry `json:"queue"`
	ProjectQueue RequestQueue      `json:"projectQueue"`
}

// RequestQueue reads the existing detail projection; there is no queue-read RPC.
func (c Client) RequestQueue(ctx context.Context, projectID string) (*RequestQueue, error) {
	if projectID == "" {
		return nil, fmt.Errorf("select a Project")
	}
	var out struct {
		Project Project `json:"project"`
	}
	if err := c.call(ctx, "detail", map[string]any{"projectId": projectID}, &out); err != nil {
		return nil, err
	}
	if out.Project.ID != projectID {
		return nil, fmt.Errorf("Project queue identity does not match")
	}
	if out.Project.Queue == nil {
		return nil, fmt.Errorf("Project request queue is unavailable on this Host")
	}
	return out.Project.Queue, nil
}

// ChangeRequestQueue preserves the displayed revision. The Host atomically
// checks lifecycle, current requirement authority and ordering before mutation.
func (c Client) ChangeRequestQueue(ctx context.Context, projectID string, input RequestQueueChange) (RequestQueueResult, error) {
	var out RequestQueueResult
	if projectID == "" || input.RequestID == "" || input.RequestRevision <= 0 {
		return out, fmt.Errorf("select a current requirement revision")
	}
	if input.Action != "enqueue" && input.Action != "move" && input.Action != "cancel" && input.Action != "resume" {
		return out, fmt.Errorf("choose enqueue, move, cancel or resume")
	}
	if input.BeforeRequestID != "" && (input.Action != "move" && input.Action != "enqueue" || input.BeforeRequestID == input.RequestID) {
		return out, fmt.Errorf("select a different queued requirement")
	}
	err := c.call(ctx, "requests/queue", struct {
		ProjectID string `json:"projectId"`
		RequestQueueChange
	}{projectID, input}, &out)
	if err == nil && out.RequestID != input.RequestID {
		err = fmt.Errorf("request queue response identity does not match")
	}
	return out, err
}

type RequestPage struct {
	Items      []RequestCard `json:"items"`
	Total      int           `json:"total"`
	NextCursor *string       `json:"nextCursor"`
}

func (c Client) Requests(ctx context.Context, projectID, kind, cursor string) (RequestPage, error) {
	var out RequestPage
	if projectID == "" || (kind != "current" && kind != "completed" && kind != "archived" && kind != "cancelled") {
		return out, fmt.Errorf("select current, completed, archived or cancelled user requests")
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

package projectflow

import (
	"context"
	"fmt"
	"whaleisle/internal/dsh"
)

type TeamMember struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Role        string `json:"role"`
	Phase       string `json:"phase"`
	Error       string `json:"error"`
}
type TeamTask struct {
	ID                 string   `json:"id"`
	Subject            string   `json:"subject"`
	Description        string   `json:"description"`
	Status             string   `json:"status"`
	OwnerName          string   `json:"ownerName"`
	BlockedBy          []string `json:"blockedBy"`
	WriteScopes        []string `json:"writeScopes"`
	WriteScopeWarnings []string `json:"writeScopeWarnings"`
	Ready              bool     `json:"ready"`
}
type TeamProjection struct {
	Members []TeamMember `json:"members"`
	Tasks   []TeamTask   `json:"tasks"`
	Failure string       `json:"failure"`
}
type Worker struct {
	SessionID   string `json:"sessionId"`
	Phase       string `json:"phase"`
	Stopped     bool   `json:"stopped"`
	ActiveRunID string `json:"activeRunId"`
	Error       string `json:"error"`
	Jobs        []struct {
		ID string `json:"id"`
	} `json:"jobs"`
}
// WaitingReason is the Host-provided reason a queued assignment cannot start.
type WaitingReason struct {
	Kind       string   `json:"kind"`
	WorkTitles []string `json:"workTitles"`
	Code       string   `json:"code"`
}

type Workstream struct {
	Group                string `json:"group"`
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Status               string `json:"status"`
	WorkerSessionID      string `json:"workerSessionId"`
	BlockedReason        string `json:"blockedReason"`
	PendingSummary       bool   `json:"pendingSummary"`
	HasReport            bool   `json:"hasReport"`
	PrerequisitesChanged bool   `json:"prerequisitesChanged"`
	CurrentDelegationRef string `json:"currentDelegationRef"`
	Delegation           struct {
		Phase         string         `json:"phase"`
		WaitingReason *WaitingReason `json:"waitingReason"`
	} `json:"delegation"`
	SummaryFailure *struct {
		DelegationRef string  `json:"delegationRef"`
		RunID         string  `json:"runId"`
		At            float64 `json:"at"`
		Error         struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"summaryFailure"`
}
type WorkPage struct {
	Workstreams []Workstream `json:"workstreams"`
	Workers     []Worker     `json:"workers"`
	NextCursor  *string      `json:"nextCursor"`
	HasMore     bool         `json:"hasMore"`
	Total       int          `json:"total"`
}
type Overview struct {
	Project     Project      `json:"project"`
	Workstreams []Workstream `json:"workstreams"`
	Workers     []Worker     `json:"workers"`
	WorkPage    struct {
		CurrentTotal      int     `json:"currentTotal"`
		CurrentCursor     *string `json:"currentCursor"`
		CurrentNextCursor *string `json:"currentNextCursor"`
	} `json:"workPage"`
	Team *TeamProjection `json:"-"`
	// CurrentCursors is local replacement-page navigation; it is never sent to the Host.
	CurrentCursors []string `json:"-"`
}

func (c Client) TeamOverview(ctx context.Context, id string) (Overview, error) {
	return c.TeamOverviewPage(ctx, id, "")
}

// TeamOverviewPage lets detail resolve a retired page anchor back to page one.
// Refreshing a page also refreshes the authoritative Project and native roster.
func (c Client) TeamOverviewPage(ctx context.Context, id, cursor string) (Overview, error) {
	var out Overview
	if id == "" {
		return out, fmt.Errorf("select a Project")
	}
	payload := map[string]any{"projectId": id}
	if cursor != "" {
		payload["currentCursor"] = cursor
	}
	if err := c.call(ctx, "detail", payload, &out); err != nil {
		return out, err
	}
	if out.Project.ID != id || out.Project.CoordinatorSessionID == "" {
		return out, fmt.Errorf("Project Lead session is unavailable")
	}
	if out.WorkPage.CurrentCursor != nil && *out.WorkPage.CurrentCursor != "" && *out.WorkPage.CurrentCursor != cursor {
		return out, fmt.Errorf("Project current-work page identity does not match")
	}
	out.CurrentCursors = []string{""}
	if out.WorkPage.CurrentCursor != nil && *out.WorkPage.CurrentCursor != "" {
		out.CurrentCursors = append(out.CurrentCursors, *out.WorkPage.CurrentCursor)
	}
	var projection struct {
		Values struct {
			Team *TeamProjection `json:"agentTeam"`
		} `json:"values"`
	}
	api := dsh.Client{Endpoint: dsh.Endpoint{Origin: c.Origin, Cookie: c.Cookie}, HTTP: c.HTTP}
	if err := api.Call(ctx, "session/projections", map[string]any{"request": map[string]any{"sessionId": out.Project.CoordinatorSessionID}}, &projection); err != nil {
		return out, err
	}
	out.Team = projection.Values.Team
	if out.Team == nil {
		return out, fmt.Errorf("native Team projection is unavailable on this Host")
	}
	if out.Team.Failure != "" {
		return out, fmt.Errorf("native Team projection is invalid: %s", out.Team.Failure)
	}
	return out, nil
}
func (c Client) CurrentWork(ctx context.Context, id, cursor string) (WorkPage, error) {
	var out WorkPage
	payload := map[string]any{"projectId": id, "kind": "current"}
	if cursor != "" {
		payload["cursor"] = cursor
	}
	err := c.call(ctx, "workstreams/list", payload, &out)
	return out, err
}

// Control deliberately excludes Team task CRUD and model-only provenance tools.
func (c Client) Control(ctx context.Context, id, action, workID string) (string, error) {
	if id == "" || (action != "stop" && action != "pause" && action != "resume") || (workID != "" && action != "stop") {
		return "", fmt.Errorf("unsupported Project control")
	}
	payload := map[string]any{"projectId": id}
	if workID != "" {
		payload["workstreamId"] = workID
	}
	var out struct {
		Note  string `json:"note"`
		State string `json:"state"`
	}
	err := c.call(ctx, action, payload, &out)
	if out.Note != "" {
		return out.Note, err
	}
	return out.State, err
}

// MemberState never treats durable active/idle lifecycle as proof of execution or completion.
func (o Overview) MemberState(m TeamMember) string {
	if m.Role == "lead" {
		if o.Project.Activity.CoordinatorRunning {
			return "running"
		}
		switch o.Project.Activity.State {
		case "idle":
			return "ready"
		case "done":
			return "delivered"
		case "":
			return "unavailable"
		default:
			return o.Project.Activity.State
		}
	}
	var worker *Worker
	var work *Workstream
	for i := range o.Workers {
		if o.Workers[i].SessionID == m.ID {
			worker = &o.Workers[i]
			break
		}
	}
	for i := range o.Workstreams {
		if o.Workstreams[i].WorkerSessionID == m.ID {
			work = &o.Workstreams[i]
			break
		}
	}
	if worker != nil && worker.Phase == "stopping" {
		return "stopping"
	}
	if m.Phase == "failed" || worker != nil && worker.Phase == "failed" {
		return "failed"
	}
	if worker != nil && (worker.ActiveRunID != "" || len(worker.Jobs) > 0) {
		return "running"
	}
	if worker != nil && worker.Stopped && (work == nil || work.Status != "done") {
		return "paused"
	}
	if m.Phase == "provisioning" || work != nil && work.Delegation.Phase == "preparing" {
		return "provisioning"
	}
	if work != nil {
		if work.Delegation.Phase == "queued" {
			return "queued"
		}
		if work.SummaryFailure != nil && (work.CurrentDelegationRef == "" || work.SummaryFailure.DelegationRef == work.CurrentDelegationRef) {
			return "summaryFailed"
		}
		if work.PendingSummary && !o.Project.Paused && (worker == nil || !worker.Stopped) {
			return "pendingSummary"
		}
		if work.PrerequisitesChanged {
			return "needsReview"
		}
		if work.Status == "blocked" {
			return "blocked"
		}
		if work.HasReport {
			return "reported"
		}
		if work.Status == "done" {
			return "completed"
		}
	}
	assigned := false
	allComplete := true
	if o.Team != nil {
		for _, task := range o.Team.Tasks {
			if m.Name != "" && task.OwnerName == m.Name || work != nil && task.ID == work.ID {
				assigned = true
				allComplete = allComplete && task.Status == "completed"
			}
		}
	}
	if assigned && allComplete {
		return "completed"
	}
	if assigned || work != nil && work.Status == "running" {
		return "assigned"
	}
	if worker != nil {
		return "ready"
	}
	return "unavailable"
}

// Members includes Host workers that do not yet have a native roster row.
// Their original Session identities remain the only targets for inspection.
func (o Overview) Members() []TeamMember {
	if o.Team == nil {
		return nil
	}
	members := append([]TeamMember(nil), o.Team.Members...)
	seen := make(map[string]bool, len(members))
	for _, member := range members {
		seen[member.ID] = true
	}
	for _, worker := range o.Workers {
		if worker.SessionID == "" || seen[worker.SessionID] {
			continue
		}
		member := TeamMember{ID: worker.SessionID, Role: "member", Phase: worker.Phase, Error: worker.Error}
		for _, work := range o.Workstreams {
			if work.WorkerSessionID == worker.SessionID {
				member.Description = work.Title
				break
			}
		}
		members = append(members, member)
		seen[member.ID] = true
	}
	return members
}

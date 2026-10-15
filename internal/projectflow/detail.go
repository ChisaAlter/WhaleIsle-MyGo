package projectflow

import (
	"context"
	"encoding/json"
	"fmt"
	"whaleisle/internal/dsh"
)

type WorkDetail struct {
	Workstream struct {
		Workstream
		ProjectID string `json:"projectId"`
		Brief     string `json:"brief"`
		Request   struct {
			ID             string `json:"id"`
			Revision       int    `json:"revision"`
			AuthorityError string `json:"authorityError"`
		} `json:"request"`
		LatestReport      *Report      `json:"latestReport"`
		RecentSettlements []Settlement `json:"recentSettlements"`
		Delegations       []struct {
			Ref   string `json:"ref"`
			RunID string `json:"runId"`
			Phase string `json:"phase"`
		} `json:"delegations"`
	} `json:"workstream"`
	Worker *struct {
		Worker
		Materialized bool   `json:"materialized"`
		CWD          string `json:"cwd"`
		Branch       string `json:"branch"`
	} `json:"worker"`
}

func (c Client) WorkDetail(ctx context.Context, projectID, workID, memberID string) (WorkDetail, error) {
	var out WorkDetail
	if projectID == "" || (workID == "") == (memberID == "") {
		return out, fmt.Errorf("choose one workstream or member identity")
	}
	payload := map[string]any{"projectId": projectID}
	if workID != "" {
		payload["workstreamId"] = workID
	} else {
		payload["workerSessionId"] = memberID
	}
	if err := c.call(ctx, "workstreams/get", payload, &out); err != nil {
		return out, err
	}
	if out.Workstream.ProjectID != projectID || out.Workstream.ID == "" || workID != "" && out.Workstream.ID != workID || memberID != "" && out.Workstream.WorkerSessionID != memberID {
		return out, fmt.Errorf("Project work identity does not match")
	}
	return out, nil
}
func (d WorkDetail) ProcessAddress(p Project) (map[string]any, error) {
	if p.ID == "" || p.CoordinatorSessionID == "" || d.Workstream.ProjectID != p.ID || d.Worker == nil || !d.Worker.Materialized || d.Worker.SessionID == "" || d.Worker.SessionID != d.Workstream.WorkerSessionID {
		return nil, fmt.Errorf("the original member Session is unavailable; no replacement will be created")
	}
	return map[string]any{"kind": "subagent", "parentSessionId": p.CoordinatorSessionID, "childSessionId": d.Worker.SessionID, "mode": "continuable"}, nil
}
func RetryDraft(p Project, d WorkDetail) (string, error) {
	if p.ID == "" || p.CoordinatorSessionID == "" || d.Workstream.ProjectID != p.ID || d.Workstream.ID == "" || d.Worker == nil || d.Worker.Phase != "failed" || d.Worker.SessionID == "" || d.Worker.SessionID != d.Workstream.WorkerSessionID {
		return "", fmt.Errorf("refresh a failed original member before preparing a retry request")
	}
	return fmt.Sprintf("Please retry the failed original Project work %q (workstreamId: %s, member session / childId: %s). Inspect the failure first and continue this existing workstream with its original member identity and authorized scope. Do not create a duplicate task or replacement member. If the original identity cannot be resumed safely, explain the blocker before taking another approach.", d.Workstream.Title, d.Workstream.ID, d.Worker.SessionID), nil
}

type ProcessRecord struct {
	Type  string `json:"type"`
	Event struct {
		Type string          `json:"type"`
		Seq  float64         `json:"seq"`
		Data json.RawMessage `json:"data"`
	} `json:"event"`
}
type ProcessSnapshot struct {
	Type    string          `json:"type"`
	Cursor  float64         `json:"cursor"`
	Records []ProcessRecord `json:"records"`
	HasMore bool            `json:"hasMore"`
}

func ParseProcessSnapshot(raw []byte) (ProcessSnapshot, error) {
	var out ProcessSnapshot
	err := json.Unmarshal(raw, &out)
	if err == nil && (out.Type != "snapshot" || out.Records == nil) {
		err = fmt.Errorf("invalid member history opening snapshot")
	}
	return out, err
}

// ReadProcess captures one non-activating continuable follow opening and releases
// it immediately. Refresh is explicit; it never prompts or replaces a member.
func ReadProcess(ctx context.Context, mux *dsh.Mux, p Project, d WorkDetail) (ProcessSnapshot, error) {
	address, err := d.ProcessAddress(p)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	if mux == nil {
		return ProcessSnapshot{}, fmt.Errorf("member history transport is unavailable")
	}
	if err = mux.Dial(ctx); err != nil {
		return ProcessSnapshot{}, err
	}
	stream, err := mux.Open(ctx, "session/follow", map[string]any{"args": map[string]any{"request": map[string]any{"address": address, "maxMessages": 30}}})
	if err != nil {
		return ProcessSnapshot{}, err
	}
	defer stream.Cancel()
	raw, err := stream.Recv(ctx)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	return ParseProcessSnapshot(raw)
}
func (c Client) EarlierProcess(ctx context.Context, p Project, d WorkDetail, current ProcessSnapshot) (ProcessSnapshot, error) {
	address, err := d.ProcessAddress(p)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	before := float64(0)
	for _, r := range current.Records {
		if r.Type == "event" && r.Event.Seq > 0 && (before == 0 || r.Event.Seq < before) {
			before = r.Event.Seq
		}
	}
	if current.Cursor <= 0 || before <= 0 || !current.HasMore {
		return ProcessSnapshot{}, fmt.Errorf("no earlier member history cursor")
	}
	api := dsh.Client{Endpoint: dsh.Endpoint{Origin: c.Origin, Cookie: c.Cookie}, HTTP: c.HTTP}
	var page ProcessSnapshot
	err = api.Call(ctx, "session/page", map[string]any{"request": map[string]any{"address": address, "throughSeq": current.Cursor, "beforeSeq": before, "maxMessages": 30}}, &page)
	page.Type = "snapshot"
	page.Cursor = current.Cursor
	return page, err
}

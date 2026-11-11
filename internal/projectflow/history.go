package projectflow

import (
	"context"
	"encoding/json"
	"fmt"
)

type ReportSource struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type Report struct {
	DelegationRef        string          `json:"delegationRef"`
	CallID               string          `json:"callId"`
	At                   float64         `json:"at"`
	Outcome              string          `json:"outcome"`
	Summary              string          `json:"summary"`
	Evidence             []string        `json:"evidence"`
	RemainingIssues      []string        `json:"remainingIssues"`
	Sources              []ReportSource  `json:"sources,omitempty"`
	Artifacts            []Artifact      `json:"artifacts"`
	Request              json.RawMessage `json:"request"`
	PrerequisitesChanged bool            `json:"prerequisitesChanged"`
}
type ReportEntry struct {
	WorkstreamID string          `json:"workstreamId"`
	Title        string          `json:"title"`
	Current      bool            `json:"current"`
	ReportCount  int             `json:"reportCount"`
	Report       Report          `json:"report"`
	Request      json.RawMessage `json:"request"`
}
type ReportListing struct {
	Results    []ReportEntry `json:"results"`
	Total      int           `json:"resultsTotal"`
	NextCursor string        `json:"nextResultsCursor"`
}
type ReportHistory struct {
	Items      []Report        `json:"items"`
	Total      int             `json:"total"`
	NextCursor string          `json:"nextCursor"`
	HasMore    bool            `json:"hasMore"`
	Request    json.RawMessage `json:"request"`
}

func (c Client) Reports(ctx context.Context, id, cursor string) (ReportListing, error) {
	var out ReportListing
	if id == "" {
		return out, fmt.Errorf("select a Project")
	}
	p := map[string]any{"projectId": id}
	if cursor != "" {
		p["resultsCursor"] = cursor
	}
	err := c.call(ctx, "store/list", p, &out)
	if err == nil && out.Results == nil {
		err = fmt.Errorf("Host returned no Project report list")
	}
	return out, err
}
func (c Client) ReportHistory(ctx context.Context, id, work, cursor, ref string) (ReportHistory, error) {
	var out ReportHistory
	if id == "" || work == "" || (cursor != "" && ref != "") {
		return out, fmt.Errorf("select one work and history cursor or exact report")
	}
	p := map[string]any{"projectId": id, "workstreamId": work}
	if cursor != "" {
		p["cursor"] = cursor
	}
	if ref != "" {
		p["delegationRef"] = ref
	}
	err := c.call(ctx, "store/results", p, &out)
	if err == nil && (out.Items == nil || (ref != "" && (len(out.Items) != 1 || out.Items[0].DelegationRef != ref))) {
		err = fmt.Errorf("Host did not return the requested recorded report")
	}
	return out, err
}
func (r Report) File(work, path string) (Artifact, error) {
	for _, f := range r.Artifacts {
		if f.Path == path && path != "" && work != "" && r.DelegationRef != "" {
			f.WorkstreamID = work
			f.DelegationRef = r.DelegationRef
			return f, nil
		}
	}
	return Artifact{}, fmt.Errorf("file does not belong to this recorded report")
}

type ReportBrowser struct {
	ProjectID, WorkID, Title string
	Listing                  ReportListing
	History                  ReportHistory
	Selected                 *Report
	Work                     *WorkDetail
	ListCursors, WorkCursors []string
	Error, Notice            string
}

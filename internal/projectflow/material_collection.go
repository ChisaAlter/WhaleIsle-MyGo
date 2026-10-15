package projectflow

import "context"

// MaterialRequests is the recorded authority-recovery collection returned by
// store/list. It is separate from current/completed/archived request cards.
type MaterialRequests struct {
	Items      []RequestCard `json:"items"`
	Total      int           `json:"total"`
	NextCursor string        `json:"nextCursor"`
}
type MaterialCollection struct {
	ProjectID, Title string
	Requests         MaterialRequests
	Cursors          []string
	Loaded           bool
	Error            string
}

func (c Client) MaterialRequestHistory(ctx context.Context, projectID, cursor string) (MaterialRequests, error) {
	var out struct {
		Requests MaterialRequests `json:"requests"`
	}
	payload := map[string]any{"projectId": projectID, "kind": "requests"}
	if cursor != "" {
		payload["cursor"] = cursor
	}
	err := c.call(ctx, "store/list", payload, &out)
	return out.Requests, err
}

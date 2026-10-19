package projectflow

import (
	"context"
	"fmt"
)

type RequestChangeSummary struct {
	IsGit      bool     `json:"isGit"`
	FileCount  int      `json:"fileCount"`
	Added      int      `json:"added"`
	Removed    int      `json:"removed"`
	Revertible bool     `json:"revertible"`
	RevertedAt *float64 `json:"revertedAt"`
}

type RequestChangeScope struct {
	WorkingDirectory string `json:"workingDirectory"`
	Diff             string `json:"diff"`
	Truncated        bool   `json:"truncated"`
	Files            []struct {
		Path    string `json:"path"`
		Added   int    `json:"added"`
		Removed int    `json:"removed"`
		Binary  bool   `json:"binary"`
		Status  string `json:"status"`
	} `json:"files"`
}

// RequestChanges is the requirement's recorded development interval, distinct
// from a workstream's current workspace diff.
type RequestChanges struct {
	RequestChangeScope
	ProjectID  string               `json:"-"`
	RequestID  string               `json:"-"`
	Workspaces []RequestChangeScope `json:"workspaces"`
	Final      bool                 `json:"final"`
	Isolated   bool                 `json:"isolated"`
	RevertedAt *float64             `json:"revertedAt"`
}

func (c Client) RequestChanges(ctx context.Context, projectID, requestID string) (RequestChanges, error) {
	out := RequestChanges{ProjectID: projectID, RequestID: requestID}
	if projectID == "" || requestID == "" {
		return out, fmt.Errorf("select a Project requirement")
	}
	err := c.call(ctx, "requests/changes", map[string]any{"projectId": projectID, "requestId": requestID}, &out)
	return out, err
}

func (c Client) RevertRequest(ctx context.Context, projectID, requestID string, revision int) error {
	if projectID == "" || requestID == "" || revision < 1 {
		return fmt.Errorf("select a current Project requirement revision")
	}
	return c.call(ctx, "requests/revert", map[string]any{"projectId": projectID, "requestId": requestID, "requestRevision": revision}, nil)
}

func (c Client) PrepareRevertRequest(ctx context.Context, card RequestCard) (string, error) {
	fresh, err := c.Request(ctx, card.ProjectID, card.ID)
	if err != nil {
		return "", err
	}
	if fresh.Revision != card.Revision || fresh.Changes == nil || fresh.Changes.RevertedAt != nil {
		return "", fmt.Errorf("the requirement changed; refresh its details before continuing")
	}
	return fmt.Sprintf("Please revert the changes made by requirement %q. Automatic revert found conflicts because files changed after delivery. Inspect the conflicting files, revert only this requirement's changes, and preserve later edits.", fresh.Title), nil
}

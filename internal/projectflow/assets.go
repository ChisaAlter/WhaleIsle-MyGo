package projectflow

import (
	"context"
	"fmt"
)

type Artifact struct {
	Title         string `json:"title"`
	Path          string `json:"path"`
	WorkstreamID  string `json:"workstreamId"`
	DelegationRef string `json:"delegationRef"`
	Location      string `json:"location"`
}
type ResolvedArtifact struct {
	CanonicalPath       string `json:"canonicalPath"`
	Root                string `json:"root"`
	ModifiedSinceReport bool   `json:"modifiedSinceReport"`
}
type WorkDiff struct {
	Snapshot struct {
		IsGit     *bool  `json:"isGit"`
		Diff      string `json:"diff"`
		Truncated bool   `json:"truncated"`
		Files     []struct {
			Path string `json:"path"`
		} `json:"files"`
	} `json:"snapshot"`
	Worker struct {
		CWD    string `json:"cwd"`
		Mode   string `json:"mode"`
		Branch string `json:"branch"`
	} `json:"worker"`
}

// ResolveArtifact keeps the immutable report reference. The Host validates the
// reported file and desktop bridge canonicalizes it before native navigation.
func (c Client) ResolveArtifact(ctx context.Context, id string, file Artifact) (ResolvedArtifact, error) {
	var out ResolvedArtifact
	if id == "" || file.WorkstreamID == "" || file.DelegationRef == "" || file.Path == "" {
		return out, fmt.Errorf("select a file from a specific delivery report")
	}
	err := c.call(ctx, "open", map[string]any{"projectId": id, "workstreamId": file.WorkstreamID, "delegationRef": file.DelegationRef, "path": file.Path}, &out)
	if err == nil && out.CanonicalPath == "" {
		err = fmt.Errorf("Project desktop bridge returned no canonical file path")
	}
	return out, err
}

// WorkDiff is the current workspace snapshot, never a historical rollback.
func (c Client) WorkDiff(ctx context.Context, id, workID string) (WorkDiff, error) {
	var out WorkDiff
	if id == "" || workID == "" {
		return out, fmt.Errorf("select a Project workstream")
	}
	err := c.call(ctx, "diff", map[string]any{"projectId": id, "workstreamId": workID}, &out)
	if err == nil && out.Snapshot.IsGit == nil {
		err = fmt.Errorf("Project workspace snapshot is unavailable")
	}
	return out, err
}
func (c Client) OpenProjectDirectory(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("select a Project")
	}
	return c.call(ctx, "open", map[string]any{"projectId": id}, nil)
}

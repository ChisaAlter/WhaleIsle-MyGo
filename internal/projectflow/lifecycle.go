package projectflow

import (
	"context"
	"fmt"
)

type LifecycleResult struct {
	Project Project `json:"project"`
	Note    string  `json:"note"`
}

func (p Project) HasActiveWork() bool {
	return p.Activity.CoordinatorRunning || p.Activity.Running > 0 || p.Activity.Queued > 0 || p.Activity.Provisioning > 0 || p.Activity.State == "stopping"
}
func (c Client) LifecycleProject(ctx context.Context, id string) (Project, error) {
	var out LifecycleResult
	if id == "" {
		return out.Project, fmt.Errorf("select a Project")
	}
	err := c.call(ctx, "detail", map[string]any{"projectId": id}, &out)
	if err == nil && out.Project.ID != id {
		err = fmt.Errorf("Project lifecycle identity does not match")
	}
	return out.Project, err
}
func (c Client) SetProjectLifecycle(ctx context.Context, id, action string) (LifecycleResult, error) {
	var out LifecycleResult
	if id == "" || (action != "archive" && action != "restore") {
		return out, fmt.Errorf("unsupported Project lifecycle action")
	}
	err := c.call(ctx, action, map[string]any{"projectId": id}, &out)
	expected := "archived"
	if action == "restore" {
		expected = "ready"
	}
	if err == nil && (out.Project.ID != id || out.Project.Lifecycle != expected) {
		err = fmt.Errorf("Project lifecycle response is unavailable; refresh before retrying")
	}
	return out, err
}

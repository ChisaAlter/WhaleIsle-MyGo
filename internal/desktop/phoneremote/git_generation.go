package phoneremote

import (
	"context"
	"encoding/json"
	"time"
	desktopgit "whaleisle/internal/desktop/git"
)

func (r *Runtime) generateGitDraft(child Child, data []byte) {
	var event struct {
		ID    string `json:"id"`
		Root  string `json:"root"`
		Input struct {
			CommitSummary string `json:"commitSummary"`
			DiffSummary   string `json:"diffSummary"`
			DiffPatch     string `json:"diffPatch"`
			FallbackTitle string `json:"fallbackTitle"`
			Template      string `json:"changeRequestTemplate"`
		} `json:"input"`
	}
	if json.Unmarshal(data, &event) != nil {
		return
	}
	r.mu.Lock()
	current, authorize := r.child == child, r.authorizeGit
	r.mu.Unlock()
	result := map[string]any{"action": "git-generation-result", "id": event.ID}
	if !current || authorize == nil || !authorize(event.Root, "gitCreateChangeRequest") {
		result["error"] = "Mobile Git desktop session permission expired"
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		g, err := desktopgit.GenerateChangeRequestText(ctx, desktopgit.PRGenerationInput{
			CommitSummary: event.Input.CommitSummary, DiffSummary: event.Input.DiffSummary,
			DiffPatch: event.Input.DiffPatch, FallbackTitle: event.Input.FallbackTitle, Template: event.Input.Template,
		})
		if err != nil {
			result["error"] = err.Error()
		} else {
			result["value"] = map[string]string{"title": g.Title, "body": g.Body, "source": g.Source}
		}
	}
	_ = r.sendControl(child, result)
}

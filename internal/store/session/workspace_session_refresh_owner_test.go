package session

import (
	"context"
	"net/http"
	"testing"

	"whaleisle/internal/dsh"
)

// A UI callback may navigate between any two refresh admissions. Each existing
// adapter must check the original selection under its own mutation lock.
func TestWorkspaceSessionViewRefreshRejectsStaleOwner(t *testing.T) {
	for _, transition := range []string{"selection", "selection-aba", "events", "host-aba", "cwd"} {
		t.Run(transition, func(t *testing.T) {
			s := New()
			calls := make(chan string, 8)
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://stale-refresh.invalid"}, HTTP: &http.Client{Transport: lifecycleAdmissionTransport(func(r *http.Request) (*http.Response, error) {
				calls <- r.URL.Path
				return nil, context.Canceled
			})}}
			s.Attach(client, nil)
			s.mu.Lock()
			s.selectedID = "created"
			s.cwd = "/original"
			owner := &workspaceSelectedOwner{client: client, hostGeneration: s.eventsGeneration, composerGeneration: s.composerHostGeneration,
				followGeneration: s.followGeneration, sessionID: s.selectedID, cwd: s.cwd}
			s.mu.Unlock()
			switch transition {
			case "selection":
				s.mu.Lock()
				s.selectSessionLocked("later")
				s.mu.Unlock()
			case "selection-aba":
				s.mu.Lock()
				s.selectSessionLocked("later")
				s.selectSessionLocked("created")
				s.mu.Unlock()
			case "events":
				s.BeginEvents(context.Background(), client)
			case "host-aba":
				s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://other.invalid"}}, nil)
				s.Attach(client, nil)
			case "cwd":
				s.SetWorkspaceCwd("/later")
			}
			s.mu.Lock()
			beforeSkill, beforeCatalog, beforeFeedback := s.skillOptionsEpoch, s.skillCatalogVersion, s.feedback.epoch
			s.mu.Unlock()
			s.refreshProjectionsForOwner(context.Background(), owner)
			s.refreshSkillsForOwner(context.Background(), owner)
			s.refreshFeedbackForOwner(context.Background(), owner)
			s.mu.Lock()
			if s.skillOptionsEpoch != beforeSkill || s.skillCatalogVersion != beforeCatalog || s.feedback.epoch != beforeFeedback || s.feedback.Loading {
				s.mu.Unlock()
				t.Fatal("stale refresh mutated current UI state")
			}
			s.mu.Unlock()
			select {
			case method := <-calls:
				t.Fatalf("stale owner started %s", method)
			default:
			}
		})
	}
}

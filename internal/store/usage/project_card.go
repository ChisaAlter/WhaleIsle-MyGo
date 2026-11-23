package usage

import (
	"context"
	"errors"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

// The first-page card and the More modal have separate sorting/read owners.
type projectCardState struct {
	ProjectRanking
	cancel context.CancelFunc
}

func (s *Store) LoadProjectCard(ctx context.Context, sortBy string) {
	s.mu.Lock()
	if s.client == nil || s.stats.Loading {
		s.unlockPublish()
		return
	}
	if sortBy != "cost" {
		sortBy = "tokens"
	}
	old := s.projectRanking.card
	if old != nil && old.Loading {
		s.unlockPublish()
		return
	}
	client, epoch := s.client, s.attachL.Generation()
	ctx, cancel := context.WithTimeout(ctx, 65*time.Second)
	state := &projectCardState{ProjectRanking: ProjectRanking{Sort: sortBy, Loading: true}, cancel: cancel}
	s.projectRanking.card = state
	s.unlockPublish()
	s.fire()
	go func() {
		defer cancel()
		rpc := &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
		var page struct {
			Rows    []ProjectUsage `json:"rows"`
			HasMore *bool          `json:"hasMore"`
		}
		err := rpc.Call(ctx, "usage-stats", "projects.more", map[string]any{"offset": 0, "sort": sortBy}, &page)
		if err == nil && (page.Rows == nil || page.HasMore == nil) {
			err = errors.New("incomplete project ranking response")
		}
		s.mu.Lock()
		if s.client != client || s.attachL.Generation() != epoch || s.projectRanking.card != state {
			s.unlockPublish()
			return
		}
		state.Loading = false
		state.cancel = nil
		state.Started = true
		if err != nil {
			state.Err = faultOf(err)
		} else {
			state.Rows = page.Rows
			state.HasMore = *page.HasMore
		}
		s.unlockPublish()
		s.fire()
	}()
}

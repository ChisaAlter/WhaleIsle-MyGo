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

func (s *Store) ProjectCardRanking() ProjectRanking {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.projectRanking.card == nil {
		return ProjectRanking{Sort: "tokens"}
	}
	v := s.projectRanking.card.ProjectRanking
	v.Rows = append([]ProjectUsage(nil), v.Rows...)
	return v
}
func (s *Store) LoadProjectCard(ctx context.Context, sortBy string) {
	s.mu.Lock()
	if s.client == nil || s.stats.Loading {
		s.mu.Unlock()
		return
	}
	if sortBy != "cost" {
		sortBy = "tokens"
	}
	old := s.projectRanking.card
	if old != nil && old.Loading {
		s.mu.Unlock()
		return
	}
	client, epoch := s.client, s.statsEpoch
	ctx, cancel := context.WithTimeout(ctx, 65*time.Second)
	state := &projectCardState{ProjectRanking: ProjectRanking{Sort: sortBy, Loading: true}, cancel: cancel}
	s.projectRanking.card = state
	s.mu.Unlock()
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
		if s.client != client || s.statsEpoch != epoch || s.projectRanking.card != state {
			s.mu.Unlock()
			return
		}
		state.Loading = false
		state.cancel = nil
		state.Started = true
		if err != nil {
			state.Err = err.Error()
		} else {
			state.Rows = page.Rows
			state.HasMore = *page.HasMore
		}
		s.mu.Unlock()
		s.fire()
	}()
}

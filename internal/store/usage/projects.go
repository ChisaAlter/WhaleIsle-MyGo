package usage

import (
	"context"
	"errors"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

type ProjectUsage struct {
	Project *string      `json:"project"`
	Name    string       `json:"name"`
	Totals  Totals       `json:"totals"`
	Models  []ModelUsage `json:"models"`
}

func (r ProjectUsage) key() string {
	if r.Project == nil {
		return "null"
	}
	return "path:" + *r.Project
}

type ProjectRanking struct {
	Rows                      []ProjectUsage
	Sort                      string
	Started, Loading, HasMore bool
	Err                       string
}
type projectRankingState struct {
	ProjectRanking
	card   *projectCardState
	offset int
	epoch  uint64
	cancel context.CancelFunc
}

func (s *Store) resetProjectRankingLocked() {
	if s.projectRanking.card != nil && s.projectRanking.card.cancel != nil {
		s.projectRanking.card.cancel()
	}
	if s.projectRanking.cancel != nil {
		s.projectRanking.cancel()
	}
	s.projectRanking = projectRankingState{epoch: s.projectRanking.epoch + 1}
}
func (s *Store) ProjectRanking() ProjectRanking {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.projectRanking.ProjectRanking
	if v.Sort == "" {
		v.Sort = "tokens"
	}
	v.Rows = append([]ProjectUsage(nil), v.Rows...)
	return v
}

// ResetProjectRanking explicitly restarts the official all-time working-directory
// aggregation from offset zero. The Host, not the UI, sorts by tokens/cost.
func (s *Store) ResetProjectRanking(ctx context.Context, sort string) {
	if sort != "tokens" && sort != "cost" {
		return
	}
	s.mu.Lock()
	if s.client == nil || s.stats.Loading {
		s.mu.Unlock()
		return
	}
	card := s.projectRanking.card
	if s.projectRanking.cancel != nil {
		s.projectRanking.cancel()
	}
	s.projectRanking = projectRankingState{epoch: s.projectRanking.epoch + 1, card: card}
	s.projectRanking.Sort = sort
	s.mu.Unlock()
	s.LoadMoreProjects(ctx)
}
func (s *Store) LoadMoreProjects(ctx context.Context) {
	s.mu.Lock()
	r := &s.projectRanking
	if s.client == nil || s.stats.Loading || r.Loading || (r.Started && !r.HasMore && r.Err == "") {
		s.mu.Unlock()
		return
	}
	if r.Sort == "" {
		r.Sort = "tokens"
	}
	client := s.client
	offset, epoch, sort := r.offset, r.epoch, r.Sort
	ctx, cancel := context.WithTimeout(ctx, 65*time.Second)
	r.cancel = cancel
	r.Loading = true
	r.Err = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		rpc := &pluginrpc.Client{BaseURL: client.Endpoint.Origin, Cookie: client.Endpoint.Cookie, HTTPClient: client.HTTP}
		var page struct {
			Rows    []ProjectUsage `json:"rows"`
			HasMore *bool          `json:"hasMore"`
		}
		err := rpc.Call(ctx, "usage-stats", "projects.more", map[string]any{"offset": offset, "sort": sort}, &page)
		if err == nil && (page.Rows == nil || page.HasMore == nil) {
			err = errors.New("incomplete project ranking response")
		}
		if err == nil && len(page.Rows) == 0 && *page.HasMore {
			err = errors.New("project page did not advance; refresh and retry")
		}
		s.mu.Lock()
		if s.client != client || s.projectRanking.epoch != epoch {
			s.mu.Unlock()
			return
		}
		r := &s.projectRanking
		r.Loading = false
		r.cancel = nil
		if err != nil {
			r.Err = err.Error()
		} else {
			seen := map[string]bool{}
			for _, row := range r.Rows {
				seen[row.key()] = true
			}
			for _, row := range page.Rows {
				if !seen[row.key()] {
					r.Rows = append(r.Rows, row)
					seen[row.key()] = true
				}
			}
			r.offset += len(page.Rows)
			r.Started = true
			r.HasMore = *page.HasMore
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ProjectEstimate only prices returned period buckets with the existing saved
// user prices. Unknown rows remain explicitly unpriced, never synthetic zero.
func ProjectEstimate(row ProjectUsage, b BillingView) (float64, bool) {
	if !b.Loaded || len(row.Models) == 0 {
		return 0, false
	}
	total := 0.0
	complete := true
	for _, m := range row.Models {
		n, ok := EstimateModel(m, b.Settings)
		total += n
		complete = complete && ok
	}
	return total, complete
}

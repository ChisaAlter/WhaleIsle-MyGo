package usage

import (
	"context"
	"errors"
	"net/http"
	"time"
	"whaleisle/internal/desktop/pluginrpc"
)

const rankPageSize = 10

// SessionRanking is all-time only: sessions.more has no date-range parameters.
type SessionRanking struct {
	Rows                      []SessionUsage
	Sort                      string
	Started, Loading, HasMore bool
	Err                       string
}
type rankingState struct {
	SessionRanking
	offset     int // Wire rows consumed, independent of deduplicated display rows.
	epoch      uint64
	cancel     context.CancelFunc
	modelLimit int
}

func (s *Store) resetRankingLocked() {
	s.resetProjectRankingLocked()
	s.resetSessionRankingLocked()
}
func (s *Store) resetSessionRankingLocked() {
	sort := s.ranking.Sort
	if s.ranking.cancel != nil {
		s.ranking.cancel()
	}
	s.ranking = rankingState{SessionRanking: SessionRanking{Sort: sort}, epoch: s.ranking.epoch + 1}
}
func (s *Store) SessionRanking() SessionRanking {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.ranking.SessionRanking
	if v.Sort == "" {
		v.Sort = "tokens"
	}
	v.Rows = append([]SessionUsage(nil), v.Rows...)
	for i := range v.Rows {
		v.Rows[i].Models = append([]ModelUsage(nil), v.Rows[i].Models...)
	}
	return v
}
func (s *Store) ModelRankLimit() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return max(rankPageSize, s.ranking.modelLimit)
}
func (s *Store) ShowMoreModels() {
	s.mu.Lock()
	s.ranking.modelLimit = max(rankPageSize, s.ranking.modelLimit) + rankPageSize
	s.mu.Unlock()
	s.fire()
}

// LoadMoreSessions starts at zero instead of using overview.TopSessions as a
// cursor: overview's top-N is not the paging contract. Failed requests retain
// their offset; refresh/reconnect invalidates in-flight pages and cursors.
func (s *Store) LoadMoreSessions(ctx context.Context) {
	s.mu.Lock()
	r := &s.ranking
	if s.client == nil || s.stats.Loading || r.Loading || (r.Started && !r.HasMore && r.Err == "") {
		s.mu.Unlock()
		return
	}
	ep := s.client.Endpoint
	if r.Sort == "" {
		r.Sort = "tokens"
	}
	offset, epoch, sort := r.offset, r.epoch, r.Sort
	ctx, cancel := context.WithTimeout(ctx, 65*time.Second)
	r.cancel = cancel
	r.Loading = true
	r.Err = ""
	s.mu.Unlock()
	s.fire()
	go func() {
		defer cancel()
		rpc := &pluginrpc.Client{BaseURL: ep.Origin, Cookie: ep.Cookie, HTTPClient: &http.Client{Timeout: 60 * time.Second}}
		var page struct {
			Sessions []SessionUsage `json:"sessions"`
			HasMore  *bool          `json:"hasMore"`
		}
		err := rpc.Call(ctx, "usage-stats", "sessions.more", map[string]any{"offset": offset, "sort": sort}, &page)
		if err == nil && (page.Sessions == nil || page.HasMore == nil) {
			err = errors.New("incomplete session page contract")
		}
		if err == nil && len(page.Sessions) == 0 && *page.HasMore {
			err = errors.New("session page did not advance; refresh and retry")
		}
		if err == nil {
			for _, row := range page.Sessions {
				if row.ID == "" {
					err = errors.New("session page contains an empty ID")
					break
				}
			}
		}
		s.mu.Lock()
		if s.ranking.epoch != epoch {
			s.mu.Unlock()
			return
		}
		r := &s.ranking
		r.Loading = false
		r.cancel = nil
		if err != nil {
			r.Err = err.Error()
		} else {
			seen := make(map[string]bool, len(r.Rows))
			for _, row := range r.Rows {
				seen[row.ID] = true
			}
			for _, row := range page.Sessions {
				if !seen[row.ID] {
					r.Rows = append(r.Rows, row)
					seen[row.ID] = true
				}
			}
			r.offset += len(page.Sessions)
			r.Started = true
			r.HasMore = *page.HasMore
		}
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) ResetSessionRanking(ctx context.Context, sort string) {
	if sort != "tokens" && sort != "cost" {
		return
	}
	s.mu.Lock()
	if s.client == nil || s.stats.Loading {
		s.mu.Unlock()
		return
	}
	s.resetSessionRankingLocked()
	s.ranking.Sort = sort
	s.mu.Unlock()
	s.LoadMoreSessions(ctx)
}

// SessionEstimate requires complete host-provided model buckets and saved
// user prices. An unpriced model makes the whole session explicitly unpriced.
func SessionEstimate(row SessionUsage, billing BillingView) (float64, bool) {
	return ProjectEstimate(ProjectUsage{Models: row.Models}, billing)
}

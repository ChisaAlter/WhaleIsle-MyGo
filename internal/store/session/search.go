package session

import (
	"context"
	"strings"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

const sessionSearchDebounce = 300 * time.Millisecond

// SessionSearchResultLimit is the original Host-owned merged search bound.
const SessionSearchResultLimit = 20

// UpdateSessionSearch keeps immediate local matches while debouncing content search.
func (s *Store) UpdateSessionSearch(ctx context.Context, query string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.searchInput == query {
		s.mu.Unlock()
		return
	}
	s.clearSearchLocked()
	s.searchInput = query
	trimmed := strings.TrimSpace(query)
	if trimmed != "" {
		s.searchLoading = true
		gen := s.searchSeq
		// Install under the same lock as the query so an older timer cannot replace
		// the timer belonging to newer input.
		s.searchDebounce = time.AfterFunc(sessionSearchDebounce, func() { s.startSessionSearch(ctx, trimmed, gen) })
	}
	s.mu.Unlock()
	s.fire()
}

// SearchSessions refreshes the current query without changing input ownership.
func (s *Store) SearchSessions(ctx context.Context, query string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	gen := s.searchSeq
	s.mu.Unlock()
	s.startSessionSearch(ctx, strings.TrimSpace(query), gen)
}

func (s *Store) startSessionSearch(ctx context.Context, query string, expected uint64) {
	s.mu.Lock()
	if query == "" || s.searchSeq != expected || strings.TrimSpace(s.searchInput) != query {
		s.mu.Unlock()
		return
	}
	if s.searchDebounce != nil {
		s.searchDebounce.Stop()
		s.searchDebounce = nil
	}
	if s.searchCancel != nil {
		s.searchCancel()
		s.searchCancel = nil
	}
	s.searchSeq++
	gen, client := s.searchSeq, s.client
	s.searchResults = nil
	s.searchHits = nil
	s.searchHasMore = false
	s.searchRPCQuery = ""
	s.searchErr = nil
	if client == nil {
		s.searchLoading = false
		s.searchUseLocal = true
		s.mu.Unlock()
		s.fire()
		return
	}
	requestCtx, cancel := context.WithCancel(ctx)
	s.searchCancel = cancel
	s.searchLoading = true
	s.searchUseLocal = false
	s.mu.Unlock()
	s.fire()
	go s.runSessionSearch(requestCtx, client, gen, query)
}

func (s *Store) runSessionSearch(ctx context.Context, client *dsh.Client, gen uint64, query string) {
	req := api.SessionSearchRequest{Query: query}
	var out api.SessionSearchValue
	err := client.Call(ctx, "session/search", map[string]any{"request": req}, &out)
	s.mu.Lock()
	if s.client != client || s.searchSeq != gen || strings.TrimSpace(s.searchInput) != query {
		s.mu.Unlock()
		return
	}
	// A transport may finish at the same time as cancellation. Never publish its
	// success after the owner has canceled the query.
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if s.searchCancel != nil {
		s.searchCancel()
		s.searchCancel = nil
	}
	s.searchLoading = false
	if err != nil {
		s.searchUseLocal = true
		s.searchErr = err
		s.searchRPCQuery = ""
		s.searchHits = nil
		s.searchResults = nil
		s.searchHasMore = false
	} else {
		s.searchUseLocal = false
		s.searchErr = nil
		s.searchRPCQuery = query
		s.searchResults = append([]api.SessionSearchValueItemsItem(nil), out.Items...)
		s.searchHasMore = out.HasMore
		s.searchHits = make(map[string]struct{}, len(out.Items))
		for _, it := range out.Items {
			if it.SessionId != "" {
				s.searchHits[it.SessionId] = struct{}{}
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}

func (s *Store) clearSearchLocked() {
	if s.searchDebounce != nil {
		s.searchDebounce.Stop()
		s.searchDebounce = nil
	}
	if s.searchCancel != nil {
		s.searchCancel()
		s.searchCancel = nil
	}
	s.searchSeq++
	s.searchLoading = false
	s.searchUseLocal = false
	s.searchErr = nil
	s.searchRPCQuery = ""
	s.searchHits = nil
	s.searchInput = ""
	s.searchResults = nil
	s.searchHasMore = false
}

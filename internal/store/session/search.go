package session

import (
	"context"
	"strings"
	"time"

	"whaleisle/internal/dsh/api"
)

const sessionSearchDebounce = 300 * time.Millisecond

// UpdateSessionSearch debounces sidebar queries and calls session/search when non-empty.
func (s *Store) UpdateSessionSearch(ctx context.Context, query string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.searchInput == query {
		s.mu.Unlock()
		return
	}
	s.searchInput = query
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		if s.searchDebounce != nil {
			s.searchDebounce.Stop()
			s.searchDebounce = nil
		}
		s.searchSeq++
		s.clearSearchLocked()
		s.mu.Unlock()
		s.fire()
		return
	}
	s.searchLoading = true
	s.searchUseLocal = false
	s.searchErr = nil
	s.searchRPCQuery = ""
	s.searchHits = nil
	gen := s.searchSeq + 1
	s.searchSeq = gen
	if s.searchDebounce != nil {
		s.searchDebounce.Stop()
	}
	s.mu.Unlock()
	s.fire()

	s.mu.Lock()
	s.searchDebounce = time.AfterFunc(sessionSearchDebounce, func() {
		s.mu.Lock()
		if s.searchSeq != gen {
			s.mu.Unlock()
			return
		}
		q := strings.TrimSpace(s.searchInput)
		if q == "" {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()
		s.SearchSessions(ctx, q)
	})
	s.mu.Unlock()
}

// SearchSessions calls session/search and stores hit session ids for the sidebar tree.
func (s *Store) SearchSessions(ctx context.Context, query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	gen := s.searchSeq
	s.mu.Unlock()
	if client == nil {
		s.mu.Lock()
		if s.searchSeq == gen && strings.TrimSpace(s.searchInput) == query {
			s.searchLoading = false
			s.searchUseLocal = true
			s.searchErr = nil
		}
		s.mu.Unlock()
		s.fire()
		return
	}

	go func() {
		req := api.SessionSearchRequest{Query: query}
		var out api.SessionSearchValue
		err := client.Call(ctx, "session/search", map[string]any{"request": req}, &out)

		s.mu.Lock()
		if s.searchSeq != gen || strings.TrimSpace(s.searchInput) != query {
			s.mu.Unlock()
			return
		}
		s.searchLoading = false
		if err != nil {
			s.searchUseLocal = true
			s.searchErr = err
			s.searchRPCQuery = ""
			s.searchHits = nil
			s.mu.Unlock()
			s.fire()
			return
		}
		s.searchUseLocal = false
		s.searchErr = nil
		s.searchRPCQuery = query
		hits := make(map[string]struct{}, len(out.Items))
		for _, it := range out.Items {
			if it.SessionId != "" {
				hits[it.SessionId] = struct{}{}
			}
		}
		s.searchHits = hits
		s.mu.Unlock()
		s.fire()
	}()
}

func (s *Store) clearSearchLocked() {
	s.searchLoading = false
	s.searchUseLocal = false
	s.searchErr = nil
	s.searchRPCQuery = ""
	s.searchHits = nil
	s.searchInput = ""
}

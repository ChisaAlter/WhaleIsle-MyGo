package main

import (
	"context"
	"sync"
	"time"

	"whaleisle/internal/desktop/whale"
	"whaleisle/internal/harness"
)

// whaleCatalogState caches one Host's dsh-whale catalog. The settings page
// renders from this snapshot synchronously while refreshes run in the
// background, so a slow or offline Host never blocks the window.
type whaleCatalogState struct {
	mu      sync.Mutex
	key     string
	snap    *whale.Catalog
	err     string
	loading bool
	lastTry time.Time
}

// whaleRetryInterval bounds automatic retries: a failed catalog call must not
// turn every frame into a new RPC.
const whaleRetryInterval = 5 * time.Second

func whaleHostKey(ep harness.Endpoint) string { return ep.Origin + "\x00" + ep.Cookie }

// whalePetEnabled reports whether the dsh-whale page can talk to this Host.
func whalePetEnabled(ep harness.Endpoint, ready bool) bool {
	// The cookie is optional: pluginrpc simply omits the header when the local
	// Harness runs without auth.
	return ready && ep.Origin != ""
}

// whaleCatalogSnapshot returns the cached catalog for this exact Host.
func (s *mainState) whaleCatalogSnapshot(ep harness.Endpoint) (*whale.Catalog, string) {
	st := &s.whaleCatalog
	key := whaleHostKey(ep)
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.key != key {
		return nil, ""
	}
	return st.snap, st.err
}

// requestWhaleCatalog loads the catalog once per Host (force reloads after a
// mutation). Failures are cached with a retry interval.
func (s *mainState) requestWhaleCatalog(ep harness.Endpoint, force bool) {
	key := whaleHostKey(ep)
	st := &s.whaleCatalog
	now := time.Now()
	st.mu.Lock()
	if st.loading || (!force && st.key == key && (st.snap != nil || st.err != "" && now.Sub(st.lastTry) < whaleRetryInterval)) {
		st.mu.Unlock()
		return
	}
	st.loading, st.lastTry = true, now
	st.mu.Unlock()
	client, err := whale.New(ep.Origin, ep.Cookie)
	if err != nil {
		st.mu.Lock()
		st.loading, st.key, st.err = false, key, err.Error()
		st.mu.Unlock()
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		catalog, callErr := client.Catalog(ctx)
		live := s.currentEndpoint()
		st.mu.Lock()
		st.loading = false
		switch {
		case live.Origin != "" && key != whaleHostKey(live):
			// A different Host answered while this call was in flight; drop it.
		case callErr != nil:
			st.key, st.snap, st.err = key, nil, callErr.Error()
		default:
			st.key, st.snap, st.err = key, &catalog, ""
		}
		st.mu.Unlock()
		s.refreshWhaleSettingsUI()
	}()
}

// currentEndpoint reads the live Host endpoint under the window lock.
func (s *mainState) currentEndpoint() harness.Endpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bootStatus.Endpoint
}

// runWhaleMutation applies one write and reloads the catalog afterwards. The
// page stays on the previous snapshot while the call is in flight.
func (s *mainState) runWhaleMutation(ep harness.Endpoint, apply func(context.Context, *whale.Client) error) {
	client, err := whale.New(ep.Origin, ep.Cookie)
	if err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if callErr := apply(ctx, client); callErr != nil {
			st := &s.whaleCatalog
			st.mu.Lock()
			st.key, st.err = whaleHostKey(ep), callErr.Error()
			st.mu.Unlock()
			s.refreshWhaleSettingsUI()
			return
		}
		s.requestWhaleCatalog(ep, true)
	}()
}

// whalePetAccess builds the settings page's Whale data and callbacks. Every
// return value is nil when the Host is not ready, so the page keeps its current
// behaviour instead of showing an empty form.
func (s *mainState) whalePetAccess(ep harness.Endpoint, ready bool) (snapshot *whale.Catalog, onLoad func(), onUpdate func(whale.Patch), onReplaceMemory func(string), onClearMemory func(), onToggleSkill func(string, bool)) {
	if !whalePetEnabled(ep, ready) {
		return nil, nil, nil, nil, nil, nil
	}
	if _, err := whale.New(ep.Origin, ep.Cookie); err != nil {
		return nil, nil, nil, nil, nil, nil
	}
	snapshot, _ = s.whaleCatalogSnapshot(ep)
	s.requestWhaleCatalog(ep, false)
	return snapshot,
		func() { s.requestWhaleCatalog(ep, true) },
		func(patch whale.Patch) {
			s.runWhaleMutation(ep, func(ctx context.Context, client *whale.Client) error {
				_, err := client.Update(ctx, patch)
				return err
			})
		},
		func(text string) {
			s.runWhaleMutation(ep, func(ctx context.Context, client *whale.Client) error {
				_, err := client.ReplaceMemory(ctx, text)
				return err
			})
		},
		func() {
			s.runWhaleMutation(ep, func(ctx context.Context, client *whale.Client) error {
				return client.ClearMemory(ctx)
			})
		},
		func(name string, enabled bool) {
			s.runWhaleMutation(ep, func(ctx context.Context, client *whale.Client) error {
				return client.ToggleSkill(ctx, name, enabled)
			})
		}
}

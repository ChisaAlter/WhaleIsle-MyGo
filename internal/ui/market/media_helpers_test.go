package market

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
	"whaleisle/internal/desktop/media"
)

// useTestMedia gives the Market a media service this test drives: no
// background decode workers, downloads through the test's own cache.
func useTestMedia(t *testing.T, cache *imgcache.Cache) *media.Service {
	t.Helper()
	cfg := media.Config{Workers: -1}
	if cache != nil {
		cfg.Fetch = cache.Fetch
	}
	s := media.New(cfg)
	previous := marketMedia
	marketMedia = func() *media.Service { return s }
	t.Cleanup(func() { marketMedia = previous })
	return s
}

// settleMedia runs queued decodes and frames until done reports true.
func settleMedia(t *testing.T, tt *ui.Tester, s *media.Service, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s.RunPending()
		tt.Frame()
		if done() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("media did not settle: texts=%v", tt.Texts())
}

package settings

import (
	"net/url"
	"os"
	"sync"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
)

var accountImages struct {
	sync.Mutex
	cache   *imgcache.Cache
	bitmaps map[string]*ui.Bitmap
}

// SetAccountImageCache reuses the application's bounded, credential-free image
// cache; profile fields never become local filesystem image paths.
func SetAccountImageCache(cache *imgcache.Cache) {
	accountImages.Lock()
	accountImages.cache = cache
	accountImages.Unlock()
}

func RetryAccountAvatar(raw string, invalidate func()) {
	accountImages.Lock()
	cache := accountImages.cache
	accountImages.Unlock()
	if cache != nil {
		cache.Retry(raw, invalidate)
	}
}

func accountAvatar(c *ui.Context, raw string) *ui.Bitmap {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Host == "" {
		return nil
	}
	accountImages.Lock()
	cache := accountImages.cache
	bitmap := accountImages.bitmaps[raw]
	accountImages.Unlock()
	if bitmap != nil || cache == nil {
		return bitmap
	}
	status, path, _ := cache.State(raw)
	if status == imgcache.StatusMissing {
		cache.Ensure(raw, c.Services().Invalidate)
		return nil
	}
	if status != imgcache.StatusReady {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	bitmap, err = ui.DecodeBitmap(data)
	if err != nil {
		return nil
	}
	accountImages.Lock()
	if accountImages.bitmaps == nil {
		accountImages.bitmaps = map[string]*ui.Bitmap{}
	}
	accountImages.bitmaps[raw] = bitmap
	accountImages.Unlock()
	return bitmap
}

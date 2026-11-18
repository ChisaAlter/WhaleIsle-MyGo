package settings

import (
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/imgcache"
)

const (
	maxSettingsAvatarCache = 32
	settingsAvatarNegTTL   = 30 * time.Second
)

type settingsAvatarEntry struct {
	bmp      *ui.Bitmap
	failedAt time.Time
}

var accountImages struct {
	sync.Mutex
	cache    *imgcache.Cache
	entries  map[string]settingsAvatarEntry
	order    []string
	inFlight map[string]bool
}

func touchAvatarLRU(order []string, key string) []string {
	for i, existing := range order {
		if existing == key {
			order = append(order[:i], order[i+1:]...)
			break
		}
	}
	return append(order, key)
}

func storeSettingsAvatarLocked(key string, bmp *ui.Bitmap) {
	if accountImages.entries == nil {
		accountImages.entries = make(map[string]settingsAvatarEntry, maxSettingsAvatarCache)
	}
	entry := settingsAvatarEntry{bmp: bmp}
	if bmp == nil {
		entry.failedAt = time.Now()
	}
	accountImages.entries[key] = entry
	accountImages.order = touchAvatarLRU(accountImages.order, key)
	for len(accountImages.order) > maxSettingsAvatarCache {
		oldest := accountImages.order[0]
		accountImages.order = accountImages.order[1:]
		delete(accountImages.entries, oldest)
	}
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
	delete(accountImages.entries, raw)
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
	var invalidate func()
	if c != nil {
		invalidate = c.Services().Invalidate
	}
	now := time.Now()
	accountImages.Lock()
	cache := accountImages.cache
	if entry, ok := accountImages.entries[raw]; ok {
		if entry.bmp != nil {
			accountImages.order = touchAvatarLRU(accountImages.order, raw)
			accountImages.Unlock()
			return entry.bmp
		}
		if now.Sub(entry.failedAt) < settingsAvatarNegTTL {
			accountImages.order = touchAvatarLRU(accountImages.order, raw)
			accountImages.Unlock()
			return nil
		}
		delete(accountImages.entries, raw)
	}
	inFlight := accountImages.inFlight[raw]
	accountImages.Unlock()
	if cache == nil || inFlight {
		return nil
	}
	status, path, _ := cache.State(raw)
	if status == imgcache.StatusMissing {
		cache.Ensure(raw, invalidate)
		return nil
	}
	if status != imgcache.StatusReady {
		return nil
	}
	accountImages.Lock()
	if accountImages.inFlight[raw] {
		accountImages.Unlock()
		return nil
	}
	if accountImages.inFlight == nil {
		accountImages.inFlight = make(map[string]bool)
	}
	accountImages.inFlight[raw] = true
	accountImages.Unlock()

	go func() {
		var bmp *ui.Bitmap
		if path != "" {
			if data, err := os.ReadFile(path); err == nil {
				if decoded, err := ui.DecodeBitmap(data); err == nil {
					bmp = decoded
				}
			}
		}
		accountImages.Lock()
		delete(accountImages.inFlight, raw)
		storeSettingsAvatarLocked(raw, bmp)
		accountImages.Unlock()
		if bmp != nil && invalidate != nil {
			invalidate()
		}
	}()
	return nil
}

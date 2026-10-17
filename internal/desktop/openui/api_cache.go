package openui

import (
	"encoding/json"
	"sync"
)

// Public adapter results are reused for the application process lifetime.
// Only response JSON is stored: instance IDs, credentials and bitmap payloads
// never enter this cache. Eviction follows insertion order, not access order.
type publicResponseCache struct {
	mu       sync.Mutex
	entries  map[string]json.RawMessage
	order    []string
	capacity int
}

type resourceCaches struct {
	wiki, images, routes publicResponseCache
}

var processResourceCaches = resourceCaches{
	wiki:   publicResponseCache{capacity: 500},
	images: publicResponseCache{capacity: 500},
	routes: publicResponseCache{capacity: 200},
}

func (c *publicResponseCache) read(key string, e Event) (APIResponse, bool) {
	if c == nil {
		return APIResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	body, ok := c.entries[key]
	if !ok {
		return APIResponse{}, false
	}
	return APIResponse{InstanceID: e.InstanceID, RequestID: e.RequestID, Status: 200, Body: append(json.RawMessage(nil), body...)}, true
}

func (c *publicResponseCache) retain(key string, result APIResponse) APIResponse {
	if c == nil || result.Status != 200 {
		return result
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[string]json.RawMessage)
	}
	if _, exists := c.entries[key]; !exists {
		if len(c.order) >= c.capacity {
			delete(c.entries, c.order[0])
			c.order = c.order[1:]
		}
		c.order = append(c.order, key)
	}
	c.entries[key] = append(json.RawMessage(nil), result.Body...)
	return result
}

func (a *ResourceAPI) cacheFor(path string) *publicResponseCache {
	if a.caches == nil {
		return nil
	}
	switch path {
	case "/api/wiki":
		return &a.caches.wiki
	case "/api/image-search":
		return &a.caches.images
	case "/api/street-route":
		return &a.caches.routes
	}
	return nil
}

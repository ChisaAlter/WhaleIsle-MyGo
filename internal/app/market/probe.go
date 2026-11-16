package marketctl

import (
	"sync"
	"time"
)

// bgProbe caches a key-indexed value for a TTL and refreshes stale keys in
// the background at most once concurrently per key.
type bgProbe[T any] struct {
	mu      sync.Mutex
	key     string
	value   T
	at      time.Time
	running bool
	has     bool
}

func (p *bgProbe[T]) get(key string, ttl time.Duration, load func() T, notify func()) T {
	p.mu.Lock()
	if p.has && p.key == key && (ttl <= 0 || time.Since(p.at) < ttl) {
		val := p.value
		p.mu.Unlock()
		return val
	}
	val := p.value
	if !p.running {
		p.running = true
		go func(probeKey string) {
			fresh := load()
			now := time.Now()
			p.mu.Lock()
			p.key = probeKey
			p.value = fresh
			p.at = now
			p.has = true
			p.running = false
			p.mu.Unlock()
			if notify != nil {
				notify()
			}
		}(key)
	}
	p.mu.Unlock()
	return val
}

package main

import (
	"reflect"
	"sync"
	"time"
)

// backgroundProbe answers a file-system question from its last result and
// refreshes it off the UI thread once stale, so a render pass never waits on
// the file system (a busy on-access scanner can hold a single stat or PATH
// lookup for many milliseconds). Only the first question for a key is answered
// synchronously. changed runs, off the UI thread, when a refresh differs.
type backgroundProbe[V any] struct {
	mu    sync.Mutex
	key   string
	value V
	known bool
	at    time.Time
	// gen advances each time the answer is set by anything but a refresh (put, or
	// the synchronous first question for a key). A refresh keeps the generation it
	// started under and drops its result if that has moved, so a probe that began
	// before a fresher read cannot put the older answer back.
	gen     uint64
	probing bool
}

// set records an answer and retires every refresh started before it. Caller
// holds p.mu.
func (p *backgroundProbe[V]) set(key string, v V) {
	p.key, p.value, p.known, p.at = key, v, true, time.Now()
	p.gen++
}

// put records an answer read elsewhere, such as a fresh read an action needed.
func (p *backgroundProbe[V]) put(key string, v V) {
	p.mu.Lock()
	p.set(key, v)
	p.mu.Unlock()
}

func (p *backgroundProbe[V]) get(key string, maxAge time.Duration, probe func() V, changed func()) V {
	p.mu.Lock()
	if !p.known || p.key != key {
		gen := p.gen
		p.mu.Unlock()
		v := probe()
		p.mu.Lock()
		if p.gen == gen {
			p.set(key, v)
		} else if p.known && p.key == key {
			// Another reader answered this key while the probe ran; theirs is newer.
			v = p.value
		}
		p.mu.Unlock()
		return v
	}
	if !p.probing && time.Since(p.at) >= maxAge {
		p.probing = true
		gen := p.gen
		go func() {
			v := probe()
			p.mu.Lock()
			current := p.known && p.key == key && p.gen == gen
			differs := current && !reflect.DeepEqual(p.value, v)
			if current {
				p.value, p.at = v, time.Now()
			}
			p.probing = false
			p.mu.Unlock()
			if differs && changed != nil {
				changed()
			}
		}()
	}
	v := p.value
	p.mu.Unlock()
	return v
}

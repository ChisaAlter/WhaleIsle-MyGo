package pluginmanager

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// HostInventory owns the three inventory reads for one attached transport.
// Runtime rows are read at attach; bundle and writable-row metadata are read
// only when a consumer asks. Concurrent consumers share each completed read.
// Real change events and writes invalidate the generation, never a TTL.
type HostInventory struct {
	mu      sync.Mutex
	caller  Caller
	life    context.Context
	cancel  context.CancelFunc
	version uint64
	reads   map[string]*hostInventoryRead
	changed func(string)
}

type hostInventoryRead struct {
	version uint64
	done    chan struct{}
	raw     json.RawMessage
	err     error
}

func NewHostInventory(caller Caller, changed func(string)) *HostInventory {
	life, cancel := context.WithCancel(context.Background())
	return &HostInventory{caller: caller, life: life, cancel: cancel, version: 1, reads: make(map[string]*hostInventoryRead), changed: changed}
}

func (h *HostInventory) Close()                   { h.cancel() }
func (h *HostInventory) InventoryVersion() uint64 { h.mu.Lock(); defer h.mu.Unlock(); return h.version }
func (h *HostInventory) InvalidateInventory() {
	h.mu.Lock()
	h.version++
	h.reads = make(map[string]*hostInventoryRead)
	h.mu.Unlock()
}

func inventoryReadMethod(method string) bool {
	return method == "pluginInventory/list" || method == "pluginManager/listBundles" || method == "pluginManager/listPlugins"
}

func (h *HostInventory) Call(ctx context.Context, method string, args, out any) error {
	if !inventoryReadMethod(method) {
		err := h.caller.Call(ctx, method, args, out)
		switch method {
		case "pluginManager/installBundle", "pluginManager/removeBundle", "pluginManager/setBundleEnabled", "pluginManager/setPluginEnabled", "pluginManager/setVersionExemption":
			h.InvalidateInventory()
		}
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		h.mu.Lock()
		read := h.reads[method]
		if read == nil {
			read = &hostInventoryRead{version: h.version, done: make(chan struct{})}
			h.reads[method] = read
			go h.read(method, read)
		}
		h.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-h.life.Done():
			return h.life.Err()
		case <-read.done:
		}
		h.mu.Lock()
		current := h.version == read.version
		h.mu.Unlock()
		if !current {
			continue
		}
		if read.err != nil {
			return read.err
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(read.raw, out)
	}
}

func (h *HostInventory) read(method string, read *hostInventoryRead) {
	ctx, cancel := context.WithTimeout(h.life, 30*time.Second)
	defer cancel()
	read.err = h.caller.Call(ctx, method, nil, &read.raw)
	if read.err == nil {
		read.err = ctx.Err()
	}
	h.mu.Lock()
	current := h.version == read.version
	close(read.done)
	if current && read.err != nil {
		delete(h.reads, method)
	}
	h.mu.Unlock()
	if current && h.changed != nil {
		h.changed(method)
	}
}

// Cached decodes a completed successful read without doing work. This is the
// distribution boundary for settings and sidebar projections.
func (h *HostInventory) Cached(method string, out any) (uint64, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	read := h.reads[method]
	if read == nil || read.version != h.version {
		return h.version, false
	}
	select {
	case <-read.done:
	default:
		return h.version, false
	}
	if read.err != nil {
		return h.version, false
	}
	return read.version, json.Unmarshal(read.raw, out) == nil
}

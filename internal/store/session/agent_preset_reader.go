package session

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"unicode/utf8"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// AgentPresetContentLimit bounds native text layout, not the source declaration.
const AgentPresetContentLimit = 256 * 1024

type AgentPresetReaderState struct {
	Ticket                                              uint64
	SelectedID, Host, Name, Description, Content, Error string
	Open, Busy, Loaded, Truncated, MetadataTruncated    bool
	TotalBytes                                          int
}

// AgentPresetReader only calls agentPresets/read. Its content is inert text.
// The owner includes the Host epoch so an A -> B -> A reconnect stays stale.
type AgentPresetReader struct {
	mu        sync.Mutex
	source    *Store
	isVisible func() bool
	visible   bool
	client    *dsh.Client
	endpoint  dsh.Endpoint
	epoch     uint64
	state     AgentPresetReaderState
	cancel    context.CancelFunc
	claimed   bool
}

func NewAgentPresetReader(source *Store, isVisible func() bool) *AgentPresetReader {
	return &AgentPresetReader{source: source, isVisible: isVisible}
}
func (r *AgentPresetReader) clearLocked(keepSelection bool) {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	id := ""
	if keepSelection {
		id = r.state.SelectedID
	}
	r.state = AgentPresetReaderState{Ticket: r.state.Ticket + 1, SelectedID: id, Host: r.state.Host}
	r.claimed = false
}
func (r *AgentPresetReader) hostCurrentLocked() bool {
	if r.source == nil {
		return false
	}
	r.source.mu.Lock()
	defer r.source.mu.Unlock()
	return r.client != nil && r.source.client == r.client && r.source.client.Endpoint == r.endpoint && r.source.composerHostGeneration == r.epoch
}
func (r *AgentPresetReader) currentLocked(ticket uint64) bool {
	return r.visible && r.state.Ticket == ticket && (r.isVisible == nil || r.isVisible()) && r.hostCurrentLocked()
}
func (r *AgentPresetReader) Snapshot() AgentPresetReaderState {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.source != nil {
		r.source.mu.Lock()
		client, epoch := r.source.client, r.source.composerHostGeneration
		endpoint := dsh.Endpoint{}
		if client != nil {
			endpoint = client.Endpoint
		}
		r.source.mu.Unlock()
		if client != r.client || endpoint != r.endpoint || epoch != r.epoch {
			r.clearLocked(false)
			r.client, r.endpoint, r.epoch = client, endpoint, epoch
			r.state.Host = ""
			// Never expose URL credentials, query parameters or authentication cookies.
			if u, err := url.Parse(endpoint.Origin); err == nil && u.Host != "" {
				r.state.Host = u.Scheme + "://" + u.Host
			}
		}
	}
	if r.state.Open && (!r.visible || (r.isVisible != nil && !r.isVisible())) {
		r.clearLocked(true)
	}
	return r.state
}
func (r *AgentPresetReader) ObserveVisible(visible bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.visible && !visible {
		r.clearLocked(false)
	}
	r.visible = visible
}

// Select is a local reading choice, independent of the persisted default.
func (r *AgentPresetReader) Select(ticket uint64, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.currentLocked(ticket) || r.state.SelectedID == id {
		return
	}
	r.clearLocked(false)
	r.state.SelectedID = id
}
func (r *AgentPresetReader) Close(ticket uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state.Ticket == ticket {
		r.clearLocked(true)
	}
}

// Begin runs on the UI thread, before the worker starts, to order clicks.
func (r *AgentPresetReader) Begin(ticket uint64) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.currentLocked(ticket) || r.state.SelectedID == "" || r.state.Busy {
		return 0
	}
	r.clearLocked(true)
	r.state.Open, r.state.Busy = true, true
	return r.state.Ticket
}

// Load rejects obsolete admissions and obsolete replies; it never resolves labels.
func (r *AgentPresetReader) Load(ctx context.Context, ticket uint64) {
	r.mu.Lock()
	if !r.currentLocked(ticket) || !r.state.Busy || r.claimed {
		r.mu.Unlock()
		return
	}
	r.claimed = true
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	id := r.state.SelectedID
	rpc := &dsh.Client{Endpoint: r.endpoint, HTTP: r.client.HTTP}
	r.mu.Unlock()
	defer cancel()
	var doc api.AgentPresetDocument
	err := rpc.Call(ctx, "agentPresets/read", map[string]any{"agentPreset": id}, &doc)
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.currentLocked(ticket) {
		return
	}
	r.state.Busy = false
	if err != nil {
		r.state.Error, _ = presetReaderBound(err.Error(), 4096)
		return
	}
	if doc.AgentPreset != id {
		r.state.Error = fmt.Sprintf("预设身份不一致 / Preset identity mismatch: requested %q", id)
		return
	}
	r.state.Loaded = true
	r.state.TotalBytes = len(doc.Content)
	r.state.Content, r.state.Truncated = presetReaderBound(doc.Content, AgentPresetContentLimit)
	r.state.Name = id
	nameCut, descriptionCut := false, false
	if doc.Name != nil {
		r.state.Name, nameCut = presetReaderBound(*doc.Name, 1024)
	}
	if doc.Description != nil {
		r.state.Description, descriptionCut = presetReaderBound(*doc.Description, 16*1024)
	}
	r.state.MetadataTruncated = nameCut || descriptionCut
}

// CopyText supplies only the currently displayed bytes, guarded against stale UI.
func (r *AgentPresetReader) CopyText(ticket uint64) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.currentLocked(ticket) || !r.state.Open || !r.state.Loaded || r.state.Busy {
		return "", false
	}
	return r.state.Content, true
}
func presetReaderBound(text string, limit int) (string, bool) {
	if len(text) <= limit {
		return text, false
	}
	end := limit
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end], true
}

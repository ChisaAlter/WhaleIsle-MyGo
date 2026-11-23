package pad

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/state"
)

// Store keeps the mutable working view under mu and hands readers a
// published, immutable Snapshot version. Writers copy only the containers
// they change; untouched slices and maps stay shared with the last
// publication. Attach/stream lifetimes and per-panel model requests are
// state.Lease generations instead of ad-hoc counters; disk work runs
// serialized under persistMu with no state lock held.
type Store struct {
	mu        sync.Mutex
	path      string
	view      Snapshot
	pub       state.Published[Snapshot]
	client    *Client
	attach    state.LeaseOwner
	attachL   state.Lease
	stream    state.LeaseOwner
	streamL   state.Lease
	watchKey  string
	persistMu sync.Mutex
	owners    map[string]*state.LeaseOwner
	writeFn   func(path string, l Layout) error
	notify    func()
}

func Open(userData string, notify func()) (*Store, error) {
	s := &Store{path: filepath.Join(userData, "pad", "layout.json"), notify: notify, owners: map[string]*state.LeaseOwner{}}
	s.writeFn = writeLayoutFile
	s.view = Snapshot{Layout: defaultLayout(), PanelErrors: map[string]*state.Fault{}, ActionErrors: map[string]*state.Fault{}, ModelActions: map[string]ModelAction{}}
	raw, err := os.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		if err = json.Unmarshal(raw, &s.view.Layout); err != nil {
			return nil, err
		}
	}
	if s.view.Layout.Heights == nil {
		s.view.Layout.Heights = map[string]float32{}
	}
	if s.view.Layout.States == nil {
		s.view.Layout.States = map[string]map[string]json.RawMessage{}
	}
	if s.view.Layout.Width < 120 {
		s.view.Layout.Width = 352
	}
	s.publishLocked()
	return s, nil
}
func (s *Store) fire() {
	if s.notify != nil {
		s.notify()
	}
}

// publishLocked must run under s.mu: Attached derives from s.client and the
// whole view goes out as one revision.
func (s *Store) publishLocked() {
	s.view.Attached = s.client != nil
	s.pub.Publish(s.view)
}

// sameFault keeps SetError/SetPanelError from refiring for an unchanged
// classified failure.
func sameFault(a, b *state.Fault) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
func (s *Store) SetError(err error) {
	f := state.FaultOf(err)
	s.mu.Lock()
	changed := !sameFault(s.view.Error, f)
	s.view.Error = f
	if changed {
		s.publishLocked()
	}
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}
func (s *Store) SetPanelError(id string, err error) {
	f := state.FaultOf(err)
	s.mu.Lock()
	errs := maps.Clone(s.view.PanelErrors)
	if sameFault(errs[id], f) {
		s.mu.Unlock()
		return
	}
	if f == nil {
		delete(errs, id)
	} else {
		errs[id] = f
	}
	s.view.PanelErrors = errs
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SetNotice(err error) {
	s.mu.Lock()
	s.view.Notice = state.FaultOf(err)
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Attach(c *Client, generation uint64) {
	s.mu.Lock()
	if s.client != nil && c != nil && s.client.Origin == c.Origin && s.client.Cookie == c.Cookie && s.view.Generation == generation {
		s.mu.Unlock()
		return
	}
	s.attachL = s.attach.Renew(context.Background())
	s.client = c
	s.view.Generation = generation
	s.view.Panels = nil
	s.view.Connected = false
	s.view.Disabled = false
	s.view.Loaded = false
	s.view.Error = nil
	s.view.Notice = nil
	s.view.Catalog = nil
	s.view.ModelsLoading = false
	s.view.ModelsError = nil
	s.view.ModelActions = map[string]ModelAction{}
	s.view.PanelErrors = map[string]*state.Fault{}
	s.view.ActionErrors = map[string]*state.Fault{}
	s.owners = map[string]*state.LeaseOwner{}
	s.watchKey = "\x00"
	s.restartWatchLocked()
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Detach() { s.Attach(nil, 0) }

// Snapshot returns the current published version. The same *Version is
// shared by every reader until a writer publishes; Value is immutable —
// panel History maps, the layout tree and the error maps must not be
// mutated through it.
func (s *Store) Snapshot() *state.Version[Snapshot] { return s.pub.Load() }

// Changed closes when the next version is published.
func (s *Store) Changed() <-chan struct{} { return s.pub.Changed() }

// Panel returns the shared immutable panel value.
func (s *Store) Panel(id string) (Panel, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.view.Panels {
		if p.ID == id {
			return p, true
		}
	}
	return Panel{}, false
}
func (s *Store) watchIDsLocked() []string {
	ids := []string{}
	if s.view.Layout.Expanded {
		for _, p := range s.view.Panels {
			if !p.Collapsed && !p.NeedsApproval {
				ids = append(ids, p.ID)
			}
		}
	}
	return ids
}
func (s *Store) restartWatchLocked() {
	if s.client == nil {
		return
	}
	ids := s.watchIDsLocked()
	key := strings.Join(ids, ",")
	if key == s.watchKey {
		return
	}
	s.watchKey = key
	s.streamL = s.stream.Renew(s.attachL.Context())
	go s.follow(s.streamL.Context(), s.client, s.attachL, s.streamL, ids)
}

// Reconnect is called after plugin-manager/changed, including runtime disabling
// or enabling Pad without replacing the Host generation.
func (s *Store) Reconnect() { s.mu.Lock(); s.watchKey = "\x00"; s.restartWatchLocked(); s.mu.Unlock() }
func (s *Store) follow(ctx context.Context, c *Client, attachL, streamL state.Lease, ids []string) {
	for {
		err := c.stream(ctx, ids, func(e streamEvent) { s.receive(attachL, streamL, e) })
		if ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		if s.attachL != attachL || s.streamL != streamL {
			s.mu.Unlock()
			return
		}
		s.view.Connected = false
		var status *HTTPError
		s.view.Disabled = errors.As(err, &status) && status.Status == http.StatusNotFound
		if err != nil {
			s.view.Error = state.FaultOf(err)
		}
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
func (s *Store) receive(attachL, streamL state.Lease, e streamEvent) {
	s.mu.Lock()
	if s.attachL != attachL || s.streamL != streamL {
		s.mu.Unlock()
		return
	}
	switch e.Type {
	case "panels":
		old := map[string]Panel{}
		for _, p := range s.view.Panels {
			old[p.ID] = p
		}
		created := false
		for i := range e.Panels {
			p := &e.Panels[i]
			if previous, ok := old[p.ID]; ok {
				if len(p.History) == 0 {
					p.History = maps.Clone(previous.History)
				}
			} else if s.view.Loaded {
				created = true
			}
			record(p, p.Data)
		}
		s.view.Panels = e.Panels
		s.view.Loaded = true
		if created {
			s.view.Layout.Expanded = true
		}
		s.view.Layout.Tree = normalizeTree(s.view.Layout.Tree, s.view.Panels)
	case "data":
		for i := range s.view.Panels {
			if s.view.Panels[i].ID == e.PanelID {
				panels := slices.Clone(s.view.Panels)
				panels[i].History = maps.Clone(panels[i].History)
				panels[i].Data = e.Data
				record(&panels[i], e.Data)
				s.view.Panels = panels
				break
			}
		}
	default:
		s.mu.Unlock()
		return
	}
	s.view.Connected = true
	s.view.Disabled = false
	s.view.Error = nil
	s.restartWatchLocked()
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
}
func record(p *Panel, data Data) {
	if p.History == nil {
		p.History = map[string][]Point{}
	}
	for id, r := range data.Sources {
		if !r.OK || len(r.Value) == 0 {
			continue
		}
		points := p.History[id]
		if len(points) > 0 && points[len(points)-1].At == r.At {
			continue
		}
		// Clone before appending: the previous point slice may be shared
		// with a published view and must never be written through.
		points = append(slices.Clone(points), Point{Value: append(json.RawMessage(nil), r.Value...), At: r.At})
		if len(points) > 120 {
			points = points[len(points)-120:]
		}
		p.History[id] = points
	}
}
func (s *Store) operation(generation uint64) (context.Context, *Client, state.Lease, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.view.Disabled {
		return nil, nil, state.Lease{}, errors.New("Pad: Host unavailable")
	}
	if generation != 0 && generation != s.view.Generation {
		return nil, nil, state.Lease{}, errors.New("Pad: Host changed")
	}
	return s.attachL.Context(), s.client, s.attachL, nil
}
func (s *Store) current(attachL state.Lease) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attachL == attachL && s.client != nil
}
func (s *Store) Act(ctx context.Context, a Action) error {
	parent, c, attachL, err := s.operation(a.Generation)
	if err != nil {
		return err
	}
	owned, cancel := requestContext(parent)
	defer cancel()
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	err = c.Act(owned, a)
	if !s.current(attachL) {
		return errors.New("Pad: Host changed")
	}
	return err
}
func (s *Store) Do(a Action) {
	parent, c, attachL, err := s.operation(a.Generation)
	if err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		err := c.Act(ctx, a)
		s.mu.Lock()
		if s.attachL != attachL {
			s.mu.Unlock()
			return
		}
		errs := maps.Clone(s.view.ActionErrors)
		errs[a.PanelID] = state.FaultOf(err)
		s.view.ActionErrors = errs
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) RefreshAll() {
	snap := s.Snapshot().Value
	for _, p := range snap.Panels {
		s.Do(Action{Op: "refresh", PanelID: p.ID, Generation: snap.Generation})
	}
}
func (s *Store) LoadModels() {
	s.mu.Lock()
	if s.view.ModelsLoading || s.view.Catalog != nil || s.client == nil {
		s.mu.Unlock()
		return
	}
	s.view.ModelsLoading = true
	s.view.ModelsError = nil
	c, attachL, parent := s.client, s.attachL, s.attachL.Context()
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		catalog, err := c.Models(ctx)
		s.mu.Lock()
		if s.attachL != attachL {
			s.mu.Unlock()
			return
		}
		s.view.ModelsLoading = false
		if err == nil {
			s.view.Catalog = &catalog
		} else {
			s.view.ModelsError = state.FaultOf(err)
		}
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
	}()
}

// ChooseModel keeps the editor's original busy/success/failure contract. It is
// separate from fire-and-forget chrome actions: the picker closes only after
// this exact Host/panel request has succeeded. Each panel owns a LeaseOwner,
// so a fresh pick supersedes the panel's earlier request while other panels
// keep theirs running.
func (s *Store) ChooseModel(generation uint64, id string, model *ModelSelection) uint64 {
	s.mu.Lock()
	if s.client == nil || s.view.Disabled || generation != s.view.Generation || s.view.ModelActions[id].Busy {
		s.mu.Unlock()
		return 0
	}
	owner := s.owners[id]
	if owner == nil {
		owner = &state.LeaseOwner{}
		s.owners[id] = owner
	}
	attachL := s.attachL
	lease := owner.Renew(attachL.Context())
	request := lease.Generation()
	actions := maps.Clone(s.view.ModelActions)
	actions[id] = ModelAction{Request: request, Busy: true}
	s.view.ModelActions = actions
	c := s.client
	s.publishLocked()
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(lease.Context(), 30*time.Second)
		defer cancel()
		err := c.Act(ctx, Action{Generation: generation, Op: "model", PanelID: id, Model: model})
		s.mu.Lock()
		if !lease.Valid() || s.attachL != attachL {
			s.mu.Unlock()
			return
		}
		actions := maps.Clone(s.view.ModelActions)
		actions[id] = ModelAction{Request: request, Succeeded: err == nil, Error: state.FaultOf(err)}
		s.view.ModelActions = actions
		s.publishLocked()
		s.mu.Unlock()
		s.fire()
	}()
	return request
}
func (s *Store) Run(ctx context.Context, id, prompt, system string) (RunResult, error) {
	return s.RunForGeneration(ctx, 0, id, prompt, system)
}
func (s *Store) RunForGeneration(ctx context.Context, generation uint64, id, prompt, system string) (RunResult, error) {
	parent, c, attachL, err := s.operation(generation)
	if err != nil {
		return RunResult{}, err
	}
	owned, cancel := requestContext(parent)
	defer cancel()
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	result, err := c.Run(owned, id, prompt, system)
	if !s.current(attachL) {
		return RunResult{}, errors.New("Pad: Host changed")
	}
	return result, err
}

func writeLayoutFile(path string, value Layout) error {
	raw, err := json.Marshal(value)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	if err == nil {
		err = os.WriteFile(path+".tmp", raw, 0600)
	}
	if err == nil {
		err = os.Rename(path+".tmp", path)
	}
	return err
}

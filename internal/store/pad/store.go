package pad

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu, persistMu               sync.Mutex
	path                        string
	layout                      Layout
	client                      *Client
	generation, epoch, streamID uint64
	ctx                         context.Context
	cancel, streamCancel        context.CancelFunc
	watchKey                    string
	panels                      []Panel
	connected, loaded           bool
	disabled                    bool
	modelCatalog                *Catalog
	modelsLoading               bool
	modelsError                 string
	modelActions                map[string]ModelAction
	nextModelAction             uint64
	err                         string
	notice                      string
	panelErrors                 map[string]string
	actionErrors                map[string]string
	notify                      func()
}

func Open(userData string, notify func()) (*Store, error) {
	s := &Store{path: filepath.Join(userData, "pad", "layout.json"), layout: defaultLayout(), notify: notify, panelErrors: map[string]string{}, actionErrors: map[string]string{}}
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(raw, &s.layout); err != nil {
		return nil, err
	}
	if s.layout.Heights == nil {
		s.layout.Heights = map[string]float32{}
	}
	if s.layout.States == nil {
		s.layout.States = map[string]map[string]json.RawMessage{}
	}
	if s.layout.Width < 120 {
		s.layout.Width = 352
	}
	return s, nil
}
func (s *Store) fire() {
	if s.notify != nil {
		s.notify()
	}
}
func (s *Store) SetError(err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	s.mu.Lock()
	changed := s.err != message
	s.err = message
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}
func (s *Store) SetPanelError(id, message string) {
	s.mu.Lock()
	changed := s.panelErrors[id] != message
	s.panelErrors[id] = message
	s.mu.Unlock()
	if changed {
		s.fire()
	}
}
func (s *Store) SetNotice(err error) {
	text := ""
	if err != nil {
		text = err.Error()
	}
	s.mu.Lock()
	s.notice = text
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Attach(c *Client, generation uint64) {
	s.mu.Lock()
	if s.client != nil && c != nil && s.client.Origin == c.Origin && s.client.Cookie == c.Cookie && s.generation == generation {
		s.mu.Unlock()
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.epoch++
	s.client = c
	s.generation = generation
	s.panels = nil
	s.connected = false
	s.disabled = false
	s.loaded = false
	s.err = ""
	s.notice = ""
	s.modelCatalog = nil
	s.modelsLoading = false
	s.modelsError = ""
	s.modelActions = map[string]ModelAction{}
	s.panelErrors = map[string]string{}
	s.actionErrors = map[string]string{}
	s.watchKey = "\x00"
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.restartWatchLocked()
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Detach() { s.Attach(nil, 0) }
func (s *Store) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{Attached: s.client != nil, Connected: s.connected, Loaded: s.loaded, Disabled: s.disabled, Generation: s.generation, Panels: clone(s.panels), Layout: clone(s.layout), Catalog: clone(s.modelCatalog), ModelsLoading: s.modelsLoading, ModelsError: s.modelsError, ModelActions: clone(s.modelActions), Error: s.err, Notice: s.notice, PanelErrors: clone(s.panelErrors), ActionErrors: clone(s.actionErrors)}
}
func (s *Store) Panel(id string) (Panel, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.panels {
		if p.ID == id {
			return clone(p), true
		}
	}
	return Panel{}, false
}
func (s *Store) watchIDsLocked() []string {
	ids := []string{}
	if s.layout.Expanded {
		for _, p := range s.panels {
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
	if s.streamCancel != nil {
		s.streamCancel()
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.streamCancel = cancel
	s.streamID++
	id, epoch, c := s.streamID, s.epoch, s.client
	go s.follow(ctx, c, epoch, id, ids)
}

// Reconnect is called after plugin-manager/changed, including runtime disabling
// or enabling Pad without replacing the Host generation.
func (s *Store) Reconnect() { s.mu.Lock(); s.watchKey = "\x00"; s.restartWatchLocked(); s.mu.Unlock() }
func (s *Store) follow(ctx context.Context, c *Client, epoch, id uint64, ids []string) {
	for {
		err := c.stream(ctx, ids, func(e streamEvent) { s.receive(epoch, id, e) })
		if ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		if s.epoch != epoch || s.streamID != id {
			s.mu.Unlock()
			return
		}
		s.connected = false
		var status *HTTPError
		s.disabled = errors.As(err, &status) && status.Status == http.StatusNotFound
		if err != nil {
			s.err = err.Error()
		}
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
func (s *Store) receive(epoch, id uint64, e streamEvent) {
	s.mu.Lock()
	if s.epoch != epoch || s.streamID != id {
		s.mu.Unlock()
		return
	}
	switch e.Type {
	case "panels":
		old := map[string]Panel{}
		for _, p := range s.panels {
			old[p.ID] = p
		}
		created := false
		for i := range e.Panels {
			p := &e.Panels[i]
			if previous, ok := old[p.ID]; ok {
				if len(p.History) == 0 {
					p.History = previous.History
				}
			} else if s.loaded {
				created = true
			}
			record(p, p.Data)
		}
		s.panels = e.Panels
		s.loaded = true
		if created {
			s.layout.Expanded = true
		}
		s.layout.Tree = normalizeTree(s.layout.Tree, s.panels)
	case "data":
		for i := range s.panels {
			if s.panels[i].ID == e.PanelID {
				s.panels[i].Data = e.Data
				record(&s.panels[i], e.Data)
			}
		}
	default:
		s.mu.Unlock()
		return
	}
	s.connected = true
	s.disabled = false
	s.err = ""
	s.restartWatchLocked()
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
		points = append(points, Point{Value: append(json.RawMessage(nil), r.Value...), At: r.At})
		if len(points) > 120 {
			points = points[len(points)-120:]
		}
		p.History[id] = points
	}
}
func (s *Store) operation(generation uint64) (context.Context, *Client, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil || s.disabled {
		return nil, nil, 0, errors.New("Pad: Host unavailable")
	}
	if generation != 0 && generation != s.generation {
		return nil, nil, 0, errors.New("Pad: Host changed")
	}
	return s.ctx, s.client, s.epoch, nil
}
func (s *Store) current(epoch uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.epoch == epoch && s.client != nil
}
func (s *Store) Act(ctx context.Context, a Action) error {
	parent, c, epoch, err := s.operation(a.Generation)
	if err != nil {
		return err
	}
	owned, cancel := requestContext(parent)
	defer cancel()
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	err = c.Act(owned, a)
	if !s.current(epoch) {
		return errors.New("Pad: Host changed")
	}
	return err
}
func (s *Store) Do(a Action) {
	parent, c, epoch, err := s.operation(a.Generation)
	if err != nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		err := c.Act(ctx, a)
		s.mu.Lock()
		if s.epoch != epoch {
			s.mu.Unlock()
			return
		}
		message := ""
		if err != nil {
			message = err.Error()
		}
		s.actionErrors[a.PanelID] = message
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) RefreshAll() {
	snap := s.Snapshot()
	for _, p := range snap.Panels {
		s.Do(Action{Op: "refresh", PanelID: p.ID, Generation: snap.Generation})
	}
}
func (s *Store) LoadModels() {
	s.mu.Lock()
	if s.modelsLoading || s.modelCatalog != nil || s.client == nil {
		s.mu.Unlock()
		return
	}
	s.modelsLoading = true
	s.modelsError = ""
	c, epoch, parent := s.client, s.epoch, s.ctx
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		catalog, err := c.Models(ctx)
		s.mu.Lock()
		if s.epoch != epoch {
			s.mu.Unlock()
			return
		}
		s.modelsLoading = false
		if err == nil {
			s.modelCatalog = &catalog
		} else {
			s.modelsError = err.Error()
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ChooseModel keeps the editor's original busy/success/failure contract. It is
// separate from fire-and-forget chrome actions: the picker closes only after
// this exact Host/panel request has succeeded.
func (s *Store) ChooseModel(generation uint64, id string, model *ModelSelection) uint64 {
	s.mu.Lock()
	if s.client == nil || s.disabled || generation != s.generation || s.modelActions[id].Busy {
		s.mu.Unlock()
		return 0
	}
	if s.modelActions == nil {
		s.modelActions = map[string]ModelAction{}
	}
	s.nextModelAction++
	request := s.nextModelAction
	s.modelActions[id] = ModelAction{Request: request, Busy: true}
	c, epoch, parent := s.client, s.epoch, s.ctx
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		defer cancel()
		err := c.Act(ctx, Action{Generation: generation, Op: "model", PanelID: id, Model: model})
		s.mu.Lock()
		if s.epoch != epoch || s.modelActions[id].Request != request {
			s.mu.Unlock()
			return
		}
		result := ModelAction{Request: request, Succeeded: err == nil}
		if err != nil {
			result.Error = err.Error()
		}
		s.modelActions[id] = result
		s.mu.Unlock()
		s.fire()
	}()
	return request
}
func (s *Store) Run(ctx context.Context, id, prompt, system string) (RunResult, error) {
	return s.RunForGeneration(ctx, 0, id, prompt, system)
}
func (s *Store) RunForGeneration(ctx context.Context, generation uint64, id, prompt, system string) (RunResult, error) {
	parent, c, epoch, err := s.operation(generation)
	if err != nil {
		return RunResult{}, err
	}
	owned, cancel := requestContext(parent)
	defer cancel()
	stop := context.AfterFunc(ctx, cancel)
	defer stop()
	result, err := c.Run(owned, id, prompt, system)
	if !s.current(epoch) {
		return RunResult{}, errors.New("Pad: Host changed")
	}
	return result, err
}

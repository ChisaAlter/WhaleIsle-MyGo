package chat

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
	"whaleisle/internal/paths"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/theme"
)

type whalePanelOp struct {
	Source string          `json:"source"`
	Order  [3]float64      `json:"order"`
	Append bool            `json:"append"`
	Spec   json.RawMessage `json:"spec"`
}
type whalePanelState struct {
	mu         sync.Mutex
	Path       string          `json:"-"`
	Snapshot   json.RawMessage `json:"snapshot"`
	Base       json.RawMessage `json:"-"`
	Barrier    *[3]float64     `json:"barrier"`
	Operations []whalePanelOp  `json:"-"`
	Collapsed  bool            `json:"collapsed"`
	Version    int             `json:"version"`
	Err        string          `json:"-"`
	Generation uint64          `json:"-"`
	LastUsed   time.Time       `json:"-"`
	Retired    bool            `json:"-"`
}

var whalePanels = struct {
	sync.Mutex
	values map[string]*whalePanelState
}{values: map[string]*whalePanelState{}}

func whalePanelFor(m Model) *whalePanelState {
	if m.Snapshot.SelectedID == "" {
		return nil
	}
	root := paths.UserDataDir()
	scope := "local"
	if m.FeedbackStore != nil && m.FeedbackStore.WhaleUIStorageScope() != "" {
		scope = m.FeedbackStore.WhaleUIStorageScope()
	}
	id := openui.Identity(scope, m.Snapshot.SelectedID, "panel", 0)
	key := filepath.Join(root, "whale-ui-panels-v2", id+".json")
	whalePanels.Lock()
	defer whalePanels.Unlock()
	if p := whalePanels.values[key]; p != nil {
		p.mu.Lock()
		p.LastUsed = time.Now()
		p.mu.Unlock()
		return p
	}
	// Cache only idle state; persisted panels are never removed from disk.
	if len(whalePanels.values) >= 64 {
		var oldest string
		var at time.Time
		for k, v := range whalePanels.values {
			v.mu.Lock()
			used := v.LastUsed
			v.mu.Unlock()
			if oldest == "" || used.Before(at) {
				oldest = k
				at = used
			}
		}
		stale := whalePanels.values[oldest]
		stale.mu.Lock()
		stale.Retired = true
		stale.Generation++
		stale.mu.Unlock()
		delete(whalePanels.values, oldest)
	}
	p := &whalePanelState{Version: 2, Generation: 1, Path: key, LastUsed: time.Now()}
	if raw, err := os.ReadFile(p.Path); err == nil {
		if len(raw) > openui.MaxProgramBytes*2 || json.Unmarshal(raw, p) != nil || p.Version != 2 || (len(p.Snapshot) > 0 && string(p.Snapshot) != "null" && !validWhalePanelSpec(p.Snapshot)) {
			p.Snapshot = nil
			p.Barrier = nil
			p.Err = "Saved Whale UI panel is invalid"
		}
		p.Base = append(json.RawMessage(nil), p.Snapshot...)
	} else if !os.IsNotExist(err) {
		p.Err = err.Error()
	}
	whalePanels.values[key] = p
	return p
}
func orderCmp(a, b [3]float64) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
func validWhalePanelSpec(raw json.RawMessage) bool {
	var s struct {
		Items []json.RawMessage `json:"items"`
	}
	return len(raw) <= openui.MaxProgramBytes && json.Unmarshal(raw, &s) == nil && s.Items != nil
}
func mergeWhalePanel(a, b json.RawMessage) (json.RawMessage, error) {
	if len(a) == 0 || string(a) == "null" {
		return append(json.RawMessage(nil), b...), nil
	}
	var left, right map[string]json.RawMessage
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return nil, errors.New("invalid panel spec")
	}
	var x, y []json.RawMessage
	_ = json.Unmarshal(left["items"], &x)
	_ = json.Unmarshal(right["items"], &y)
	// Match desktop tab append: same labelled tabs receive additional children.
	var lt, rt struct {
		Type string                       `json:"type"`
		Tabs []map[string]json.RawMessage `json:"tabs"`
	}
	tabs := len(x) == 1 && len(y) == 1 && json.Unmarshal(x[0], &lt) == nil && json.Unmarshal(y[0], &rt) == nil && lt.Type == "tabs" && rt.Type == "tabs"
	if tabs {
		for _, t := range rt.Tabs {
			found := false
			for _, u := range lt.Tabs {
				if string(t["label"]) == string(u["label"]) {
					var ui, ti []json.RawMessage
					_ = json.Unmarshal(u["items"], &ui)
					_ = json.Unmarshal(t["items"], &ti)
					u["items"], _ = json.Marshal(append(ui, ti...))
					found = true
					break
				}
			}
			if !found {
				lt.Tabs = append(lt.Tabs, t)
			}
		}
		x[0], _ = json.Marshal(lt)
	} else {
		x = append(x, y...)
	}
	left["items"], _ = json.Marshal(x)
	if _, ok := left["title"]; !ok {
		if title, ok := right["title"]; ok {
			left["title"] = title
		}
	}
	return json.Marshal(left)
}
func (p *whalePanelState) persistLocked() error {
	if p.Path == "" {
		return errors.New("panel storage unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(p.Path), 0700); err != nil {
		return err
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(p.Path), ".panel-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err = file.Write(raw); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, p.Path)
}
func (p *whalePanelState) Publish(source string, order [3]float64, raw json.RawMessage) error {
	op, err := openui.ValidatePanelOperation(raw)
	if err != nil {
		return err
	}
	if source == "" || order[0] < 0 || order[1] < 0 || order[2] < 0 {
		return errors.New("panel source has no conversation position")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Retired {
		return errors.New("Whale UI panel cache owner retired")
	}
	for _, old := range p.Operations {
		if old.Source == source && old.Append == op.Append && orderCmp(old.Order, order) == 0 && string(old.Spec) == string(op.Spec) {
			return nil
		}
	}
	if p.Barrier != nil && orderCmp(order, *p.Barrier) <= 0 {
		return nil
	}
	ops := make([]whalePanelOp, 0, len(p.Operations)+1)
	for _, old := range p.Operations {
		if old.Source != source {
			ops = append(ops, old)
		}
	}
	ops = append(ops, whalePanelOp{source, order, op.Append, op.Spec})
	sort.SliceStable(ops, func(i, j int) bool {
		c := orderCmp(ops[i].Order, ops[j].Order)
		return c < 0 || c == 0 && ops[i].Source < ops[j].Source
	})
	snapshot := p.Base
	for _, item := range ops {
		if item.Append {
			snapshot, err = mergeWhalePanel(snapshot, item.Spec)
			if err != nil {
				return err
			}
		} else {
			snapshot = item.Spec
		}
	}
	if !validWhalePanelSpec(snapshot) || len("root = Col([Genui("+string(snapshot)+")])") > openui.MaxProgramBytes {
		return errors.New("merged Whale UI panel exceeds source budget")
	}
	old, oldOps := p.Snapshot, p.Operations
	p.Snapshot = snapshot
	p.Operations = ops
	// Persist folded state with replay barrier; live operations remain replayable
	// in memory so a source update replaces its old contribution.
	savedBarrier := p.Barrier
	if len(ops) > 0 {
		v := ops[len(ops)-1].Order
		p.Barrier = &v
	}
	err = p.persistLocked()
	p.Barrier = savedBarrier
	if err != nil {
		p.Snapshot = old
		p.Operations = oldOps
		p.Err = err.Error()
		return err
	}
	p.Generation++
	p.Err = ""
	return nil
}
func (p *whalePanelState) Current() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.Retired && !p.Collapsed && len(p.Snapshot) > 0 && string(p.Snapshot) != "null"
}
func (p *whalePanelState) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	old, base, ops, barrier := p.Snapshot, p.Base, p.Operations, p.Barrier
	if len(p.Operations) > 0 {
		v := p.Operations[len(p.Operations)-1].Order
		p.Barrier = &v
	}
	p.Snapshot = nil
	p.Base = nil
	p.Operations = nil
	if err := p.persistLocked(); err != nil {
		p.Snapshot, p.Base, p.Operations, p.Barrier = old, base, ops, barrier
		p.Err = err.Error()
		return err
	}
	p.Generation++
	p.Err = ""
	return nil
}
func whaleUIPanelView(c *ui.Context, m Model) {
	p := whalePanelFor(m)
	if p == nil {
		return
	}
	p.mu.Lock()
	raw := append(json.RawMessage(nil), p.Snapshot...)
	collapsed, errText := p.Collapsed, p.Err
	generation := p.Generation
	p.mu.Unlock()
	if len(raw) == 0 || string(raw) == "null" {
		if errText != "" {
			ui.Text(c, errText).FontSize(12)
		}
		return
	}
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Row(c).FillWidth().Gap(8).Children(func() {
		ui.Text(c, whaleLabel("会话面板", "Conversation panel")).FontSize(13)
		if stdButton(c, palette, whaleLabel("折叠 / 展开", "Collapse / expand"), buttonGhost, false).Clicked() {
			p.mu.Lock()
			p.Collapsed = !p.Collapsed
			if err := p.persistLocked(); err != nil {
				p.Collapsed = !p.Collapsed
				p.Err = err.Error()
			} else {
				p.Generation++
			}
			p.mu.Unlock()
		}
		if stdButton(c, palette, whaleLabel("清空面板", "Clear panel"), buttonGhost, false).Clicked() {
			if err := p.Clear(); err != nil {
				p.mu.Lock()
				p.Err = err.Error()
				p.mu.Unlock()
			}
		}
	})
	if errText != "" {
		ui.Text(c, errText).FontSize(12)
	}
	if collapsed {
		return
	}
	program := "root = Col([Genui(" + string(raw) + ")])"
	msg := session.Message{ID: "whale-ui-panel", Role: "assistant"}
	capture := captureWhalePanel(m, p)
	guarded := func() (*session.ComposerTarget, func() bool, error) {
		target, valid, err := capture()
		if err != nil {
			return nil, nil, err
		}
		return target, func() bool {
			if !valid() {
				return false
			}
			p.mu.Lock()
			defer p.mu.Unlock()
			return p.Generation == generation
		}, nil
	}
	openUIBlockView(c, m, msg, "whale-ui", program, int(generation), "whale-ui-panel", true, guarded)
}

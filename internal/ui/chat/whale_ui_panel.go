package chat

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"whaleisle/internal/paths"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

const whalePanelDefaultDebounce = 25 * time.Millisecond

type whalePanelOp struct {
	Source string          `json:"source"`
	Order  [3]float64      `json:"order"`
	Append bool            `json:"append"`
	Spec   json.RawMessage `json:"spec"`
}

type whalePanelDurable struct {
	Snapshot   json.RawMessage
	Base       json.RawMessage
	Barrier    *[3]float64
	Operations []whalePanelOp
	Collapsed  bool
	Generation uint64
}

type whalePanelDisk struct {
	Snapshot  json.RawMessage `json:"snapshot"`
	Barrier   *[3]float64     `json:"barrier"`
	Collapsed bool            `json:"collapsed"`
	Version   int             `json:"version"`
}

type whalePersistWaiter struct {
	seq  uint64
	done chan error
}

type whalePanelState struct {
	mu                   sync.Mutex
	Path                 string          `json:"-"`
	Snapshot             json.RawMessage `json:"snapshot"`
	Base                 json.RawMessage `json:"-"`
	Barrier              *[3]float64     `json:"barrier"`
	Operations           []whalePanelOp  `json:"-"`
	Collapsed            bool            `json:"collapsed"`
	Version              int             `json:"version"`
	Err                  string          `json:"-"`
	Generation           uint64          `json:"-"`
	LastUsed             time.Time       `json:"-"`
	Retired              bool            `json:"-"`
	loading              bool
	loadDone             chan struct{}
	durable              whalePanelDurable
	persistSeq           uint64
	persistedSeq         uint64
	lastMarshaled        []byte
	lastWriteSucceeded   bool
	persistWorkerStarted bool
	persistWake          chan struct{}
	flushWake            chan struct{}
	persistWaiters       []whalePersistWaiter
	debounce             time.Duration
	onInvalidate         func()
	persistWrites        uint64
	writeFile            func(path string, raw []byte) error
	readFile             func(path string) ([]byte, error)
}

var whalePanels = struct {
	sync.Mutex
	values map[string]*whalePanelState
}{values: map[string]*whalePanelState{}}

func whalePanelFor(m Model) *whalePanelState {
	return whalePanelForUI(m, nil)
}

func whalePanelForUI(m Model, invalidate func()) *whalePanelState {
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
	if p := whalePanels.values[key]; p != nil {
		p.mu.Lock()
		p.LastUsed = time.Now()
		if invalidate != nil {
			p.onInvalidate = invalidate
		}
		p.mu.Unlock()
		whalePanels.Unlock()
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
	p := &whalePanelState{
		Version:      2,
		Generation:   1,
		Path:         key,
		LastUsed:     time.Now(),
		loading:      true,
		loadDone:     make(chan struct{}),
		persistWake:  make(chan struct{}, 1),
		flushWake:    make(chan struct{}, 1),
		debounce:     whalePanelDefaultDebounce,
		onInvalidate: invalidate,
		durable:      whalePanelDurable{Generation: 1},
	}
	whalePanels.values[key] = p
	whalePanels.Unlock()

	go p.runInitialLoad()
	return p
}

func cloneRawJSON(src json.RawMessage) json.RawMessage {
	if src == nil {
		return nil
	}
	return append(json.RawMessage(nil), src...)
}

func cloneBarrier(b *[3]float64) *[3]float64 {
	if b == nil {
		return nil
	}
	v := *b
	return &v
}

func clonePanelOps(ops []whalePanelOp) []whalePanelOp {
	if len(ops) == 0 {
		return nil
	}
	out := make([]whalePanelOp, len(ops))
	for i, op := range ops {
		out[i] = whalePanelOp{
			Source: op.Source,
			Order:  op.Order,
			Append: op.Append,
			Spec:   cloneRawJSON(op.Spec),
		}
	}
	return out
}

func (p *whalePanelState) WaitLoaded() {
	if p != nil && p.loadDone != nil {
		<-p.loadDone
	}
}

var (
	whalePanelReadFileFn  = os.ReadFile
	whalePanelWriteFileFn = writeWhalePanelFile
)

func (p *whalePanelState) runInitialLoad() {
	p.mu.Lock()
	path := p.Path
	readFn := p.readFile
	p.mu.Unlock()
	if readFn == nil {
		readFn = whalePanelReadFileFn
	}

	raw, err := readFn(path)

	p.mu.Lock()
	var inv func()
	// Apply initial disk state only if no newer in-memory mutation arrived while loading.
	if p.persistSeq == 0 && !p.Retired {
		if err == nil {
			var loaded whalePanelDisk
			if len(raw) > openui.MaxProgramBytes*2 || json.Unmarshal(raw, &loaded) != nil || loaded.Version != 2 || (len(loaded.Snapshot) > 0 && string(loaded.Snapshot) != "null" && !validWhalePanelSpec(loaded.Snapshot)) {
				p.Snapshot = nil
				p.Barrier = nil
				p.Err = "Saved Whale UI panel is invalid"
			} else {
				p.Snapshot = cloneRawJSON(loaded.Snapshot)
				p.Barrier = cloneBarrier(loaded.Barrier)
				p.Collapsed = loaded.Collapsed
				p.lastMarshaled, _ = json.Marshal(loaded)
				p.lastWriteSucceeded = true
			}
			p.Base = cloneRawJSON(p.Snapshot)
		} else if !os.IsNotExist(err) {
			p.Err = errcopy.Sentence(err)
		}
		p.durable = whalePanelDurable{
			Snapshot:   cloneRawJSON(p.Snapshot),
			Base:       cloneRawJSON(p.Base),
			Barrier:    cloneBarrier(p.Barrier),
			Collapsed:  p.Collapsed,
			Generation: p.Generation,
		}
	}
	p.loading = false
	close(p.loadDone)
	inv = p.onInvalidate
	p.mu.Unlock()

	if inv != nil {
		inv()
	}
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
func (p *whalePanelState) marshalDiskLocked() ([]byte, error) {
	diskBarrier := p.Barrier
	if len(p.Operations) > 0 {
		v := p.Operations[len(p.Operations)-1].Order
		diskBarrier = &v
	}
	disk := whalePanelDisk{
		Snapshot:  p.Snapshot,
		Barrier:   diskBarrier,
		Collapsed: p.Collapsed,
		Version:   p.Version,
	}
	return json.Marshal(disk)
}

func writeWhalePanelFile(path string, raw []byte) error {
	if path == "" {
		return errors.New("panel storage unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".panel-*")
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
	return os.Rename(name, path)
}

func (p *whalePanelState) ensureChannelsLocked() {
	if p.persistWake == nil {
		p.persistWake = make(chan struct{}, 1)
	}
	if p.flushWake == nil {
		p.flushWake = make(chan struct{}, 1)
	}
}

func notifyNonBlocking(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func (p *whalePanelState) schedulePersistLocked() {
	p.ensureChannelsLocked()
	p.persistSeq++
	if !p.persistWorkerStarted {
		p.persistWorkerStarted = true
		go p.runPersistWorker()
	}
	notifyNonBlocking(p.persistWake)
}

func (p *whalePanelState) resolveWaitersLocked(upTo uint64, err error) {
	kept := p.persistWaiters[:0]
	for _, w := range p.persistWaiters {
		if w.seq <= upTo {
			w.done <- err
		} else {
			kept = append(kept, w)
		}
	}
	p.persistWaiters = kept
}

func (p *whalePanelState) snapshotDurableLocked() whalePanelDurable {
	return whalePanelDurable{
		Snapshot:   cloneRawJSON(p.Snapshot),
		Base:       cloneRawJSON(p.Base),
		Barrier:    cloneBarrier(p.Barrier),
		Operations: clonePanelOps(p.Operations),
		Collapsed:  p.Collapsed,
		Generation: p.Generation,
	}
}

func (p *whalePanelState) restoreDurableLocked() {
	p.Snapshot = cloneRawJSON(p.durable.Snapshot)
	p.Base = cloneRawJSON(p.durable.Base)
	p.Barrier = cloneBarrier(p.durable.Barrier)
	p.Operations = clonePanelOps(p.durable.Operations)
	p.Collapsed = p.durable.Collapsed
	if !p.Retired {
		p.Generation = p.durable.Generation
	}
}

func (p *whalePanelState) runPersistWorker() {
	for {
		p.mu.Lock()
		if p.persistSeq == p.persistedSeq && len(p.persistWaiters) == 0 {
			p.persistWorkerStarted = false
			p.mu.Unlock()
			return
		}
		debounce := p.debounce
		flushing := len(p.persistWaiters) > 0
		p.ensureChannelsLocked()
		wakeCh := p.persistWake
		flushCh := p.flushWake
		p.mu.Unlock()

		if debounce > 0 && !flushing {
			deadline := time.Now().Add(debounce * 4)
			timer := time.NewTimer(debounce)
		debounceWait:
			for {
				select {
				case <-timer.C:
					break debounceWait
				case <-flushCh:
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					break debounceWait
				case <-wakeCh:
					rem := time.Until(deadline)
					if rem <= 0 {
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						break debounceWait
					}
					wait := debounce
					if rem < wait {
						wait = rem
					}
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(wait)
				}
			}
		}

		// Drain any coalesced wake signals before reading the latest state.
	drainWake:
		for {
			select {
			case <-wakeCh:
			case <-flushCh:
			default:
				break drainWake
			}
		}

		p.mu.Lock()
		if p.persistSeq == p.persistedSeq {
			p.resolveWaitersLocked(p.persistedSeq, nil)
			p.mu.Unlock()
			continue
		}
		targetSeq := p.persistSeq
		path := p.Path
		writeFn := p.writeFile
		if writeFn == nil {
			writeFn = whalePanelWriteFileFn
		}
		raw, err := p.marshalDiskLocked()
		candidate := p.snapshotDurableLocked()
		if err == nil && p.lastWriteSucceeded && bytes.Equal(raw, p.lastMarshaled) {
			p.persistedSeq = targetSeq
			p.durable = candidate
			p.Err = ""
			p.resolveWaitersLocked(targetSeq, nil)
			p.mu.Unlock()
			continue
		}
		p.mu.Unlock()

		if err == nil {
			err = writeFn(path, raw)
		}

		p.mu.Lock()
		p.persistWrites++
		p.persistedSeq = targetSeq
		var inv func()
		if err == nil {
			p.lastMarshaled = raw
			p.lastWriteSucceeded = true
			p.durable = candidate
			if p.persistSeq == targetSeq {
				p.Err = ""
			}
		} else {
			if p.persistSeq == targetSeq {
				p.restoreDurableLocked()
				p.Err = errcopy.Sentence(err)
				inv = p.onInvalidate
			}
		}
		p.resolveWaitersLocked(targetSeq, err)
		p.mu.Unlock()

		if inv != nil {
			inv()
		}
	}
}

// Flush immediately runs pending debounced writes and returns the persist result.
func (p *whalePanelState) Flush() error {
	if p == nil {
		return nil
	}
	p.WaitLoaded()
	p.mu.Lock()
	if p.persistSeq == p.persistedSeq {
		p.mu.Unlock()
		return nil
	}
	p.ensureChannelsLocked()
	ch := make(chan error, 1)
	p.persistWaiters = append(p.persistWaiters, whalePersistWaiter{seq: p.persistSeq, done: ch})
	if !p.persistWorkerStarted {
		p.persistWorkerStarted = true
		go p.runPersistWorker()
	}
	notifyNonBlocking(p.flushWake)
	notifyNonBlocking(p.persistWake)
	p.mu.Unlock()
	return <-ch
}

func (p *whalePanelState) Publish(source string, order [3]float64, raw json.RawMessage) error {
	op, err := openui.ValidatePanelOperation(raw)
	if err != nil {
		return err
	}
	if source == "" || order[0] < 0 || order[1] < 0 || order[2] < 0 {
		return errors.New("panel source has no conversation position")
	}
	p.WaitLoaded()
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
	p.Snapshot = snapshot
	p.Operations = ops
	p.Generation++
	p.Err = ""
	p.schedulePersistLocked()
	return nil
}

func (p *whalePanelState) Current() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.Retired && !p.Collapsed && len(p.Snapshot) > 0 && string(p.Snapshot) != "null"
}

func (p *whalePanelState) Clear() error {
	p.WaitLoaded()
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.Operations) > 0 {
		v := p.Operations[len(p.Operations)-1].Order
		p.Barrier = &v
	}
	p.Snapshot = nil
	p.Base = nil
	p.Operations = nil
	p.Generation++
	p.Err = ""
	p.schedulePersistLocked()
	return nil
}

func (p *whalePanelState) toggleCollapsed() {
	p.mu.Lock()
	p.Collapsed = !p.Collapsed
	p.Generation++
	p.Err = ""
	p.schedulePersistLocked()
	p.mu.Unlock()
}

func whaleUIPanelView(c *ui.Context, m Model) bool {
	p := whalePanelForUI(m, c.Services().Invalidate)
	if p == nil {
		return false
	}
	p.mu.Lock()
	raw := append(json.RawMessage(nil), p.Snapshot...)
	collapsed, errText := p.Collapsed, p.Err
	generation := p.Generation
	p.mu.Unlock()
	if len(raw) == 0 || string(raw) == "null" {
		if errText != "" {
			ui.Text(c, errText).FontSize(12)
			return true
		}
		return false
	}
	palette := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	ui.Row(c).FillWidth().Gap(design.Sp(2)).Children(func() {
		ui.Text(c, whaleLabel("会话面板", "Conversation panel")).FontSize(13)
		if stdButton(c, palette, whaleLabel("折叠 / 展开", "Collapse / expand"), buttonGhost, false).Clicked() {
			p.toggleCollapsed()
		}
		if stdButton(c, palette, whaleLabel("清空面板", "Clear panel"), buttonGhost, false).Clicked() {
			_ = p.Clear()
		}
	})
	if errText != "" {
		ui.Text(c, errText).FontSize(12)
	}
	if collapsed {
		return true
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
	return true
}

func isWhaleAssistantConversation(snap session.ViewState) bool {
	if snap.SelectedID == "" {
		return false
	}
	if strings.HasPrefix(snap.SelectedID, "session-whale-") {
		return true
	}
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID {
			return item.Presentation != nil && item.Presentation.Owner == "dsh-whale:assistant"
		}
	}
	return false
}

func whaleEmptyWelcomeView(c *ui.Context, snap session.ViewState, colors theme.Colors, muted ui.Color) {
	title := whaleLabel("鲸鱼娘", "Whale Assistant")
	for _, item := range snap.Items {
		if item.SessionID == snap.SelectedID && item.Presentation != nil && strings.TrimSpace(item.Presentation.Title) != "" {
			title = strings.TrimSpace(item.Presentation.Title)
			break
		}
	}
	subtitle := whaleLabel(
		"我是你的专属桌面助理。在下方输入问题，或让我协助处理工作区任务、定时提醒与工具操作。",
		"Your personal desktop assistant. Ask anything below, or have me help with workspace tasks, schedules, and tools.",
	)
	ui.Column(c.Key("whale-empty-welcome")).FillWidth().Grow(1).AlignItems(ui.Center).Justify(ui.Center).Padding(design.Sp(6)).Gap(design.Sp(2)).Children(func() {
		ui.Text(c, title).FontSize(theme.InterfaceSize(design.TextXL.Size)).FontWeight(design.WeightSemibold).TextColor(theme.Color(colors.FG))
		ui.Text(c, subtitle).FontSize(theme.InterfaceSize(design.TextSM.Size)).TextColor(muted).MaxWidthPercent(70)
	})
}

// Package hostterminal owns retained Host terminals. It never launches a local
// process. RPC writes are one-shot and every output attachment has its own input
// capability, so queued input cannot cross a reconnect or session change.
package hostterminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

type Shell struct {
	Path string   `json:"path"`
	Args []string `json:"args"`
	Name string   `json:"name"`
}
type Info struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Shell        Shell  `json:"shell"`
	Cwd          string `json:"cwd"`
	Cols         int    `json:"cols"`
	Rows         int    `json:"rows"`
	State        string `json:"state"`
	ExitCode     *int   `json:"exitCode"`
	Error        string `json:"error,omitempty"`
	ControllerID string `json:"controllerId,omitempty"`
}
type Environment struct {
	Cwd           string `json:"cwd"`
	MaxInputBytes int    `json:"maxInputBytes"`
	MaxCols       int    `json:"maxCols"`
	MaxRows       int    `json:"maxRows"`
	Scrollback    int    `json:"scrollback"`
}
type Stream interface {
	Recv(context.Context) (json.RawMessage, error)
	Cancel()
}
type Remote interface {
	Call(context.Context, string, any, any) error
	Open(context.Context, string, any) (Stream, error)
}

type Render struct {
	Revision   uint64
	Snapshot   bool
	Data       string
	Cols, Rows int
}
type View struct {
	Owner                              uint64
	ClientReady, Ready, Busy, Writable bool
	Items                              []Info
	ActiveID, Phase, Error             string
	Info                               Info
	Environment                        Environment
	Attachment                         *Attachment
	Render                             *Render
}
type sessionState struct {
	items       []Info
	active      string
	busy        bool
	error       string
	environment Environment
	operation   uint64
}
type terminalKey struct{ session, id string }
type entry struct {
	key                    terminalKey
	info                   Info
	desired, held, closing bool
	phase, error           string
	holdCancel             context.CancelFunc
	holdSerial             uint64
	attachment             *Attachment
	rename                 renameState
}

type Store struct {
	mu       sync.Mutex
	owner    any
	remote   Remote
	ready    bool
	epoch    uint64
	ctx      context.Context
	cancel   context.CancelFunc
	notify   func()
	sessions map[string]*sessionState
	entries  map[terminalKey]*entry
	visible  string
	serial   uint64
}

func (s *Store) bind(owner any, r Remote, ready bool, notify func()) {
	s.mu.Lock()
	if s.owner == owner && s.ctx != nil {
		s.notify = notify
		s.mu.Unlock()
		s.transport(owner, ready)
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	for _, e := range s.entries {
		s.stopHoldLocked(e)
	}
	s.owner, s.remote, s.ready, s.notify = owner, r, ready, notify
	s.epoch++
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.sessions = make(map[string]*sessionState)
	s.entries = make(map[terminalKey]*entry)
	s.visible = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) transport(owner any, ready bool) {
	s.mu.Lock()
	if s.owner != owner || s.remote == nil || s.ready == ready {
		s.mu.Unlock()
		return
	}
	s.ready = ready
	for _, e := range s.entries {
		s.stopHoldLocked(e)
		if e.desired && !e.closing {
			if ready {
				s.startHoldLocked(e)
			} else {
				e.phase = "disconnected"
			}
		}
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) fire() {
	s.mu.Lock()
	f := s.notify
	s.mu.Unlock()
	if f != nil {
		f()
	}
}
func (s *Store) sessionLocked(id string) *sessionState {
	if s.sessions == nil {
		s.sessions = make(map[string]*sessionState)
	}
	v := s.sessions[id]
	if v == nil {
		v = &sessionState{}
		s.sessions[id] = v
	}
	return v
}
func (s *Store) entryLocked(key terminalKey) *entry {
	if s.entries == nil {
		s.entries = make(map[terminalKey]*entry)
	}
	e := s.entries[key]
	if e == nil {
		e = &entry{key: key, info: Info{ID: key.id}, phase: "idle"}
		s.entries[key] = e
	}
	return e
}
func identity() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("terminal attachment entropy unavailable")
	}
	return hex.EncodeToString(b[:])
}

// Show only changes the visible output attachment. Window holds survive hiding
// and session navigation; neither of those actions closes a Host process.
func (s *Store) Show(session string, visible bool) {
	if !visible {
		session = ""
	}
	s.mu.Lock()
	if s.visible == session {
		s.mu.Unlock()
		return
	}
	if old := s.sessions[s.visible]; old != nil {
		if e := s.entries[terminalKey{s.visible, old.active}]; e != nil {
			s.stopFollowLocked(e)
		}
	}
	s.visible = session
	if v := s.sessions[session]; v != nil {
		if e := s.entries[terminalKey{session, v.active}]; e != nil && e.held && !e.closing {
			s.startFollowLocked(e)
		}
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) Snapshot(session string) View {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := View{Owner: s.epoch, ClientReady: s.remote != nil, Ready: s.ready}
	st := s.sessions[session]
	if st == nil {
		return v
	}
	v.Busy, v.Error, v.Environment, v.ActiveID = st.busy, st.error, st.environment, st.active
	v.Items = append([]Info(nil), st.items...)
	if e := s.entries[terminalKey{session, st.active}]; e != nil {
		v.Info, v.Phase = e.info, e.phase
		if e.error != "" {
			v.Error = e.error
		} else if e.info.Error != "" {
			v.Error = e.info.Error
		}
		v.Attachment = e.attachment
		if a := e.attachment; a != nil {
			v.Writable = s.writableLocked(a)
			if a.render != nil {
				r := *a.render
				v.Render = &r
			}
		}
	}
	return v
}

// Refresh lists retained identities only. It does not activate an Agent, allocate
// a shell, or attach/take control of any terminal.
func (s *Store) Refresh(owner uint64, session string) {
	s.mu.Lock()
	if s.epoch != owner || s.remote == nil || session == "" {
		s.mu.Unlock()
		return
	}
	st := s.sessionLocked(session)
	if st.busy {
		s.mu.Unlock()
		return
	}
	st.busy = true
	st.error = ""
	st.operation++
	op := st.operation
	r, epoch, ctx := s.remote, s.epoch, s.ctx
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		var items []Info
		err := r.Call(ctx, "terminal/list", map[string]any{"sessionId": session}, &items)
		s.mu.Lock()
		if s.epoch != epoch || st.operation != op {
			s.mu.Unlock()
			return
		}
		st.busy = false
		if err != nil {
			st.error = err.Error()
		} else {
			st.items = items
			for _, info := range items {
				e := s.entryLocked(terminalKey{session, info.ID})
				if e.attachment == nil {
					e.info = info
				}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// Create is called only by the explicit New Host terminal button. A failed or
// lost response leaves its generated identity visible for refresh/explicit close;
// retries never implicitly allocate a replacement process.
func (s *Store) Create(owner uint64, session string) {
	s.mu.Lock()
	if s.epoch != owner || s.remote == nil || !s.ready || session == "" || s.visible != session {
		s.mu.Unlock()
		return
	}
	st := s.sessionLocked(session)
	if st.busy {
		s.mu.Unlock()
		return
	}
	st.busy = true
	st.error = ""
	st.operation++
	op := st.operation
	id := identity()
	r, epoch, ctx := s.remote, s.epoch, s.ctx
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		var env Environment
		err := r.Call(ctx, "terminal/environment", map[string]any{"agentId": session}, &env)
		var info Info
		attempted := false
		if err == nil {
			err = validEnvironment(env)
		}
		if err == nil {
			s.mu.Lock()
			valid := s.epoch == epoch && st.operation == op
			s.mu.Unlock()
			if !valid {
				return
			}
			attempted = true
			err = r.Call(ctx, "terminal/create", map[string]any{"agentId": session, "request": map[string]any{"id": id, "cols": min(80, env.MaxCols), "rows": min(24, env.MaxRows)}}, &info)
		}
		if err == nil && info.ID != id {
			err = errors.New("Host returned a different terminal identity")
		}
		s.mu.Lock()
		if s.epoch != epoch || st.operation != op {
			s.mu.Unlock()
			return
		}
		st.busy = false
		if err != nil {
			st.error = err.Error()
			if attempted {
				e := s.entryLocked(terminalKey{session, id})
				e.phase = "failed"
				e.error = "Creation outcome uncertain; refresh the list or explicitly close this identity. " + err.Error()
				st.active = id
				st.items = append(st.items, e.info)
			}

		} else {
			st.environment = env
			st.items = append(st.items, info)
			e := s.entryLocked(terminalKey{session, id})
			e.info = info
			s.selectLocked(st, e)
			e.desired = true
			s.startHoldLocked(e)
		}
		s.mu.Unlock()
		s.fire()
	}()
}
func validEnvironment(v Environment) error {
	if v.MaxInputBytes < 1 || v.MaxInputBytes > 4<<20 || v.MaxCols < 2 || v.MaxCols > 4096 || v.MaxRows < 1 || v.MaxRows > 4096 || v.Scrollback < 0 {
		return errors.New("Host terminal limits unavailable")
	}
	return nil
}

// Connect is explicit for listed terminals and uses a fresh input capability.
func (s *Store) Connect(owner uint64, session, id string) {
	s.mu.Lock()
	st := s.sessionLocked(session)
	e := s.entries[terminalKey{session, id}]
	if s.epoch != owner || s.remote == nil || !s.ready || e == nil || e.closing || st.busy || s.visible != session {
		s.mu.Unlock()
		return
	}
	st.busy = true
	st.error = ""
	st.operation++
	op := st.operation
	r, epoch, ctx := s.remote, s.epoch, s.ctx
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		var env Environment
		err := r.Call(ctx, "terminal/environment", map[string]any{"agentId": session}, &env)
		if err == nil {
			err = validEnvironment(env)
		}
		s.mu.Lock()
		if s.epoch != epoch || st.operation != op || e.closing {
			s.mu.Unlock()
			return
		}
		st.busy = false
		if err != nil {
			st.error = err.Error()
		} else {
			st.environment = env
			s.selectLocked(st, e)
			e.desired = true
			e.error = ""
			if e.held {
				s.startFollowLocked(e)
			} else {
				s.startHoldLocked(e)
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) selectLocked(st *sessionState, e *entry) {
	if old := s.entries[terminalKey{e.key.session, st.active}]; old != nil {
		s.stopFollowLocked(old)
	}
	st.active = e.key.id
}

func (s *Store) stopFollowLocked(e *entry) {
	if a := e.attachment; a != nil {
		a.cancel()
		e.attachment = nil
	}
	if e.phase == "connected" || e.phase == "connecting" {
		e.phase = "retained"
	}
}
func (s *Store) stopHoldLocked(e *entry) {
	s.stopFollowLocked(e)
	e.held = false
	e.holdSerial++
	if e.holdCancel != nil {
		e.holdCancel()
		e.holdCancel = nil
	}
}
func (s *Store) startHoldLocked(e *entry) {
	if !s.ready || !e.desired || e.closing || e.holdCancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	e.holdCancel = cancel
	e.holdSerial++
	serial, epoch, r := e.holdSerial, s.epoch, s.remote
	e.phase = "retaining"
	go func() {
		st, err := r.Open(ctx, "terminal/retain", map[string]any{"sessionId": e.key.session, "id": e.key.id})
		if err == nil {
			defer st.Cancel()
			var raw json.RawMessage
			raw, err = st.Recv(ctx)
			var frame struct {
				Type string `json:"type"`
			}
			if err == nil {
				err = json.Unmarshal(raw, &frame)
				if err == nil && frame.Type != "retained" {
					err = errors.New("invalid terminal retention acknowledgement")
				}
			}
			if err == nil {
				s.mu.Lock()
				valid := s.epoch == epoch && e.holdSerial == serial && !e.closing && ctx.Err() == nil
				if valid {
					e.held = true
					e.phase = "retained"
					s.startFollowLocked(e)
				}
				s.mu.Unlock()
				s.fire()
				if !valid {
					return
				}
				for err == nil {
					_, err = st.Recv(ctx)
				}
			}
		}
		s.mu.Lock()
		if s.epoch == epoch && e.holdSerial == serial && ctx.Err() == nil {
			s.stopFollowLocked(e)
			e.held = false
			e.holdCancel = nil
			e.phase = "disconnected"
			if err != nil {
				e.error = err.Error()
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}
func (s *Store) startFollowLocked(e *entry) {
	st := s.sessions[e.key.session]
	if !s.ready || !e.held || e.closing || s.visible != e.key.session || st == nil || st.active != e.key.id {
		return
	}
	s.stopFollowLocked(e)
	ctx, cancel := context.WithCancel(s.ctx)
	a := &Attachment{ID: identity(), store: s, entry: e, remote: s.remote, epoch: s.epoch, ctx: ctx, cancel: cancel, ack: make(chan uint64, 1)}
	e.attachment = a
	e.phase = "connecting"
	e.error = ""
	go a.follow()
}
func (s *Store) currentLocked(a *Attachment) bool {
	return s.epoch == a.epoch && a.ctx.Err() == nil && a.entry.attachment == a && !a.entry.closing
}
func (s *Store) writableLocked(a *Attachment) bool {
	return s.currentLocked(a) && s.ready && a.parsed && !a.failed && a.entry.held && s.visible == a.entry.key.session && a.entry.info.State == "running" && a.entry.info.ControllerID == a.ID
}

// Acknowledge is called after native Ghostty has parsed the entire frame. No
// further output is consumed before this local render barrier.
func (s *Store) Acknowledge(a *Attachment, revision uint64) {
	s.mu.Lock()
	if a == nil || !s.currentLocked(a) || a.render == nil || a.render.Revision != revision {
		s.mu.Unlock()
		return
	}
	a.render = nil
	a.parsed = true
	select {
	case a.ack <- revision:
	default:
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) RenderFailed(a *Attachment, err error) {
	if a == nil || err == nil {
		return
	}
	a.fail(err)
	a.cancel()
}

// Close terminates exactly this captured Host/session/terminal. Hiding, input
// failure and Host replacement never call this method automatically.
func (s *Store) Close(owner uint64, session, id string) {
	s.mu.Lock()
	e := s.entries[terminalKey{session, id}]
	if s.epoch != owner || s.remote == nil || e == nil || e.closing || s.visible != session {
		s.mu.Unlock()
		return
	}
	s.cancelRenameLocked(e)
	e.closing = true
	if st := s.sessions[session]; st != nil {
		st.operation++
		st.busy = false
	}
	e.phase = "closing"
	e.error = ""
	s.stopFollowLocked(e)
	r, epoch, ctx := s.remote, s.epoch, s.ctx
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		err := r.Call(ctx, "terminal/close", map[string]any{"agentId": session, "id": id}, nil)
		s.mu.Lock()
		if s.epoch != epoch {
			s.mu.Unlock()
			return
		}
		if err != nil {
			e.closing = false
			e.phase = "failed"
			e.error = "Close outcome uncertain; retry close explicitly. " + err.Error()
		} else {
			s.stopHoldLocked(e)
			e.desired = false
			e.phase = "closed"
			e.info.State = "exited"
			if st := s.sessions[session]; st != nil {
				for i, v := range st.items {
					if v.ID == id {
						st.items = append(st.items[:i], st.items[i+1:]...)
						break
					}
				}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// Attachment is a native emulator connection for exactly one follow generation.
// Read stays open while output is delivered through acknowledged Render frames.
// Its Close releases only the view/transport, never the Host process.
type Attachment struct {
	ID             string
	store          *Store
	entry          *entry
	remote         Remote
	epoch          uint64
	ctx            context.Context
	cancel         context.CancelFunc
	ioMu           sync.Mutex
	parsed, failed bool    // Store.mu
	render         *Render // Store.mu
	ack            chan uint64
	sequence       int64
}

func (a *Attachment) Read(p []byte) (int, error) { <-a.ctx.Done(); return 0, io.EOF }
func (a *Attachment) Close() error               { a.cancel(); return nil }
func (a *Attachment) Write(p []byte) (int, error) {
	a.ioMu.Lock()
	defer a.ioMu.Unlock()
	s := a.store
	s.mu.Lock()
	valid := s.writableLocked(a)
	maxBytes := s.sessionLocked(a.entry.key.session).environment.MaxInputBytes
	s.mu.Unlock()
	if !valid {
		return 0, errors.New("Host terminal attachment is not writable")
	}
	if len(p) > maxBytes {
		a.fail(errors.New("Host terminal input exceeds its byte limit"))
		return 0, errors.New("Host terminal input exceeds its byte limit")
	}
	if len(p) == 0 {
		return 0, nil
	}
	err := a.call("terminal/write", map[string]any{"data": string(p)})
	if err != nil {
		a.fail(fmt.Errorf("Input outcome uncertain; it will not be replayed: %w", err))
		return 0, err
	}
	return len(p), nil
}
func (a *Attachment) Resize(cols, rows int) error {
	a.ioMu.Lock()
	defer a.ioMu.Unlock()
	s := a.store
	s.mu.Lock()
	if !s.writableLocked(a) {
		s.mu.Unlock()
		return errors.New("Host terminal attachment is not writable")
	}
	env := s.sessionLocked(a.entry.key.session).environment
	cols, rows = min(cols, env.MaxCols), min(rows, env.MaxRows)
	same := a.entry.info.Cols == cols && a.entry.info.Rows == rows
	s.mu.Unlock()
	if cols < 2 || rows < 1 || same {
		return nil
	}
	err := a.call("terminal/resize", map[string]any{"cols": cols, "rows": rows})
	if err != nil {
		a.fail(err)
	}
	return err
}
func (a *Attachment) call(method string, args map[string]any) error {
	s := a.store
	s.mu.Lock()
	valid := s.writableLocked(a)
	s.mu.Unlock()
	if !valid {
		return errors.New("Host terminal attachment changed")
	}
	args["agentId"], args["id"], args["attachmentId"] = a.entry.key.session, a.entry.key.id, a.ID
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	return a.remote.Call(ctx, method, args, nil)
}
func (a *Attachment) fail(err error) {
	s := a.store
	s.mu.Lock()
	if s.currentLocked(a) {
		a.failed = true
		a.entry.error = err.Error()
		a.entry.phase = "failed"
	}
	s.mu.Unlock()
	s.fire()
}
func (a *Attachment) follow() {
	stream, err := a.remote.Open(a.ctx, "terminal/follow", map[string]any{"agentId": a.entry.key.session, "id": a.entry.key.id, "attachmentId": a.ID})
	if err != nil {
		a.fail(err)
		return
	}
	defer stream.Cancel()
	first := true
	for {
		raw, err := stream.Recv(a.ctx)
		if err != nil {
			if a.ctx.Err() == nil {
				a.fail(err)
			}
			return
		}
		var f wireFrame
		if err = json.Unmarshal(raw, &f); err != nil {
			a.fail(err)
			return
		}
		if err = validateFrame(f, first, a.sequence, a.entry.key.id); err != nil {
			a.fail(err)
			return
		}
		s := a.store
		s.mu.Lock()
		if !s.currentLocked(a) {
			s.mu.Unlock()
			return
		}
		if f.Type != "output" {
			env := s.sessionLocked(a.entry.key.session).environment
			if f.Info.Cols > env.MaxCols || f.Info.Rows > env.MaxRows {
				s.mu.Unlock()
				a.fail(errors.New("Host terminal frame exceeds its grid limits"))
				return
			}
			a.entry.info = *f.Info
			a.entry.phase = "connected"
		}
		if f.Type == "snapshot" || f.Type == "output" {
			a.sequence = *f.Sequence
			s.serial++
			data := f.Data
			if first {
				data = f.Screen
			}
			a.render = &Render{Revision: s.serial, Snapshot: first, Data: data, Cols: a.entry.info.Cols, Rows: a.entry.info.Rows}
		}
		render := a.render
		s.mu.Unlock()
		s.fire()
		first = false
		if render != nil {
			select {
			case <-a.ctx.Done():
				return
			case <-a.ack:
			}
		}
	}
}

type wireFrame struct {
	Type     string `json:"type"`
	Sequence *int64 `json:"sequence"`
	Screen   string `json:"screen"`
	Data     string `json:"data"`
	Info     *Info  `json:"info"`
}

func validateFrame(f wireFrame, first bool, sequence int64, id string) error {
	if first && f.Type != "snapshot" {
		return errors.New("Host terminal stream must begin with a screen snapshot")
	}
	if f.Type != "snapshot" && f.Type != "output" && f.Type != "state" {
		return errors.New("invalid Host terminal frame")
	}
	if !first && f.Type == "snapshot" {
		return errors.New("unexpected Host terminal snapshot")
	}
	if f.Type != "state" && (f.Sequence == nil || *f.Sequence < 0 || *f.Sequence > 9007199254740991) {
		return errors.New("invalid Host terminal sequence")
	}
	if f.Type == "output" && *f.Sequence != sequence+1 {
		return errors.New("Host terminal output sequence gap")
	}
	if f.Type != "output" && (f.Info == nil || f.Info.ID != id || f.Info.Cols < 2 || f.Info.Rows < 1 || (f.Info.State != "running" && f.Info.State != "exited" && f.Info.State != "failed")) {
		return errors.New("invalid Host terminal identity or state")
	}
	return nil
}

// TerminalSize is the optional native renderer grid constraint. A read-only or
// recovering attachment must use the Host grid to preserve cursor positions.
// Writable views may propose a new bounded size, sent by Resize separately.
func (a *Attachment) TerminalSize(cols, rows int) (int, int) {
	s := a.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.writableLocked(a) {
		return max(2, a.entry.info.Cols), max(1, a.entry.info.Rows)
	}
	env := s.sessionLocked(a.entry.key.session).environment
	return max(2, min(cols, env.MaxCols)), max(1, min(rows, env.MaxRows))
}

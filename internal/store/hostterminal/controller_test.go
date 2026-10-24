package hostterminal

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

type fixtureStream struct {
	q        chan json.RawMessage
	canceled chan struct{}
	once     sync.Once
	remote   *fixtureRemote
}

func stream() *fixtureStream {
	return &fixtureStream{q: make(chan json.RawMessage, 16), canceled: make(chan struct{})}
}
func (s *fixtureStream) Recv(ctx context.Context) (json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.canceled:
		return nil, io.EOF
	case b := <-s.q:
		return b, nil
	}
}
func (s *fixtureStream) Cancel()    { s.once.Do(func() { close(s.canceled) }) }
func (s *fixtureStream) send(v any) { b, _ := json.Marshal(v); s.q <- b }

type fixtureCall struct {
	method string
	args   map[string]any
}
type fixtureRemote struct {
	mu        sync.Mutex
	calls     []fixtureCall
	opens     chan fixtureOpen
	writeErr  error
	createErr error
	closeErr  error
	list      []Info
}
type fixtureOpen struct {
	method string
	args   map[string]any
	stream *fixtureStream
}

func fixture() *fixtureRemote {
	return &fixtureRemote{opens: make(chan fixtureOpen, 16), list: []Info{{ID: "term", State: "running", Cols: 80, Rows: 24}}}
}
func object(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
func (r *fixtureRemote) Call(ctx context.Context, method string, args, out any) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, fixtureCall{method, object(args)})
	var v any
	switch method {
	case "terminal/list":
		v = r.list
	case "terminal/environment":
		v = Environment{MaxInputBytes: 8, MaxCols: 160, MaxRows: 60}
	case "terminal/create":
		if r.createErr != nil {
			return r.createErr
		}
		req := object(args)["request"].(map[string]any)
		v = Info{ID: req["id"].(string), State: "running", Cols: 80, Rows: 24}
	case "terminal/close":
		return r.closeErr
	case "terminal/write":
		return r.writeErr
	}
	if out != nil {
		b, _ := json.Marshal(v)
		return json.Unmarshal(b, out)
	}
	return nil
}
func (r *fixtureRemote) Open(ctx context.Context, method string, args any) (Stream, error) {
	s := stream()
	s.remote = r
	r.opens <- fixtureOpen{method, object(args), s}
	return s, nil
}
func (r *fixtureRemote) count(method string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.calls {
		if c.method == method {
			n++
		}
	}
	return n
}
func wait(t *testing.T, s *Store, session string, pred func(View) bool) View {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		v := s.Snapshot(session)
		if pred(v) {
			return v
		}
		select {
		case <-deadline:
			t.Fatalf("state never arrived: %+v", v)
		case <-time.After(time.Millisecond):
		}
	}
}
func nextOpen(t *testing.T, r *fixtureRemote, method string) fixtureOpen {
	t.Helper()
	select {
	case op := <-r.opens:
		if op.method != method {
			t.Fatalf("open %s, want %s", op.method, method)
		}
		return op
	case <-time.After(2 * time.Second):
		t.Fatal("no stream " + method)
	}
	return fixtureOpen{}
}
func prepare(t *testing.T) (*Store, *fixtureRemote) {
	t.Helper()
	s := new(Store)
	r := fixture()
	s.bind(r, r, true, nil)
	s.Show("session", true)
	s.Refresh(s.Snapshot("session").Owner, "session")
	wait(t, s, "session", func(v View) bool { return len(v.Items) == 1 && !v.Busy })
	s.Connect(s.Snapshot("session").Owner, "session", "term")
	return s, r
}
func attach(t *testing.T, s *Store, r *fixtureRemote, seq int64) *Attachment {
	t.Helper()
	hold := nextOpen(t, r, "terminal/retain")
	if hold.args["sessionId"] != "session" || hold.args["agentId"] != nil {
		t.Fatal(hold.args)
	}
	select {
	case op := <-r.opens:
		t.Fatalf("follow before retained: %v", op)
	default:
	}
	hold.stream.send(map[string]any{"type": "retained"})
	follow := nextOpen(t, r, "terminal/follow")
	if follow.args["agentId"] != "session" || follow.args["id"] != "term" {
		t.Fatal(follow.args)
	}
	attachmentID, _ := follow.args["attachmentId"].(string)
	if attachmentID == "" {
		t.Fatal("follow without attachment identity")
	}
	follow.stream.send(map[string]any{"type": "snapshot", "sequence": seq, "screen": "\x1b[2;3Hcursor", "info": Info{ID: "term", State: "running", Cols: 80, Rows: 24, ControllerID: attachmentID}})
	v := wait(t, s, "session", func(v View) bool { return v.Render != nil })
	if v.Writable {
		t.Fatal("input before parsed snapshot")
	}
	s.Acknowledge(v.Attachment, v.Render.Revision)
	return wait(t, s, "session", func(v View) bool { return v.Writable }).Attachment
}
func TestRetainedProtocolAndNoAutomaticCreation(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 17)
	if r.count("terminal/create") != 0 {
		t.Fatal("refresh/connect spawned")
	}
	if _, e := a.Write([]byte("abc")); e != nil {
		t.Fatal(e)
	}
	if e := a.Resize(999, 999); e != nil {
		t.Fatal(e)
	}
	r.mu.Lock()
	calls := append([]fixtureCall(nil), r.calls...)
	r.mu.Unlock()
	var writes, resizes int
	for _, c := range calls {
		switch c.method {
		case "terminal/write":
			writes++
			if c.args["agentId"] != "session" || c.args["id"] != "term" || c.args["attachmentId"] != a.ID || c.args["data"] != "abc" {
				t.Fatal(c)
			}
		case "terminal/resize":
			resizes++
			if c.args["agentId"] != "session" || c.args["id"] != "term" || c.args["attachmentId"] != a.ID || c.args["cols"] != float64(160) || c.args["rows"] != float64(60) {
				t.Fatal(c)
			}
		case "input/write", "input/resize":
			t.Fatal("obsolete duplex command transmitted", c)
		}
	}
	if writes != 1 || resizes != 1 {
		t.Fatal("one-shot control calls missing", writes, resizes)
	}
}
func TestRetainedReconnectNeverReplaysInput(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 5)
	r.mu.Lock()
	r.writeErr = errors.New("response lost")
	r.mu.Unlock()
	if _, err := a.Write([]byte("x")); err == nil {
		t.Fatal("missing failure")
	}
	if _, err := a.Write([]byte("y")); err == nil {
		t.Fatal("uncertain input remained writable")
	}
	s.transport(r, false)
	s.transport(r, true)
	b := attach(t, s, r, 88)
	if a == b || a.ID == b.ID {
		t.Fatal("reused attachment")
	}
	if _, err := a.Write([]byte("old")); err == nil {
		t.Fatal("old writer survived")
	}
	if r.count("terminal/write") != 1 {
		t.Fatal("replayed input")
	}
}
func TestRetainedHideSessionAndHostIsolation(t *testing.T) {
	s, r := prepare(t)
	a := attach(t, s, r, 1)
	s.Show("other", true)
	if _, err := a.Write([]byte("old")); err == nil {
		t.Fatal("switched input accepted")
	}
	if r.count("terminal/close") != 0 {
		t.Fatal("hide closed Host process")
	}
	s.bind(new(int), fixture(), true, nil)
	if len(s.Snapshot("session").Items) != 0 {
		t.Fatal("Host replacement leaked sessions")
	}
	if _, err := a.Write([]byte("old")); err == nil {
		t.Fatal("old Host writer accepted")
	}
	s.bind(nil, nil, false, nil)
}
func TestRetainedSequenceGapAndTakeover(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	hold := nextOpen(t, r, "terminal/retain")
	hold.stream.send(map[string]any{"type": "retained"})
	f := nextOpen(t, r, "terminal/follow")
	attachmentID, _ := f.args["attachmentId"].(string)
	f.stream.send(map[string]any{"type": "snapshot", "sequence": 9, "screen": "one", "info": Info{ID: "term", State: "running", Cols: 80, Rows: 24, ControllerID: attachmentID}})
	v := wait(t, s, "session", func(v View) bool { return v.Render != nil })
	s.Acknowledge(v.Attachment, v.Render.Revision)
	wait(t, s, "session", func(v View) bool { return v.Writable })
	// Another attachment takes input control: the Host stops naming this one.
	f.stream.send(map[string]any{"type": "state", "info": Info{ID: "term", State: "running", Cols: 80, Rows: 24}})
	wait(t, s, "session", func(v View) bool { return !v.Writable })
	if _, e := v.Attachment.Write([]byte("x")); e == nil {
		t.Fatal("takeover not enforced")
	}
	f.stream.send(map[string]any{"type": "output", "sequence": 11, "data": "gap"})
	wait(t, s, "session", func(v View) bool { return v.Error != "" && !v.Writable })
	if r.count("terminal/create") != 0 {
		t.Fatal("invalid stream recreated")
	}
}
func TestRetainedCloseTargetsOriginalSession(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 1)
	s.Close(s.Snapshot("session").Owner, "session", "term")
	wait(t, s, "session", func(v View) bool { return v.Phase == "closed" })
	if _, err := a.Write([]byte("old")); err == nil {
		t.Fatal("closed writer accepted")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) == 0 {
		t.Fatal("no calls")
	}
	last := r.calls[len(r.calls)-1]
	if last.method != "terminal/close" || last.args["agentId"] != "session" || last.args["id"] != "term" {
		t.Fatal(last)
	}
}

func TestRetainedInvalidFrameShapes(t *testing.T) {
	seq := int64(1)
	ok := &Info{ID: "term", State: "running", Cols: 80, Rows: 24}
	cases := []struct {
		name     string
		frame    wireFrame
		first    bool
		previous int64
	}{
		{"output before snapshot", wireFrame{Type: "output", Sequence: &seq}, true, 0},
		{"state before snapshot", wireFrame{Type: "state", Info: ok}, true, 0},
		{"second snapshot", wireFrame{Type: "snapshot", Sequence: &seq, Info: ok}, false, 0},
		{"missing sequence", wireFrame{Type: "output"}, false, 0},
		{"gap", wireFrame{Type: "output", Sequence: &seq}, false, 8},
		{"foreign terminal", wireFrame{Type: "snapshot", Sequence: &seq, Info: &Info{ID: "other", State: "running", Cols: 80, Rows: 24}}, true, 0},
		{"missing info", wireFrame{Type: "snapshot", Sequence: &seq}, true, 0},
		{"unknown state", wireFrame{Type: "state", Info: &Info{ID: "term", State: "other", Cols: 80, Rows: 24}}, false, 0},
		{"unknown frame", wireFrame{Type: "cursor"}, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if validateFrame(tc.frame, tc.first, tc.previous, "term") == nil {
				t.Fatal("invalid frame accepted")
			}
		})
	}
	if validateFrame(wireFrame{Type: "snapshot", Sequence: &seq, Info: ok}, true, 0, "term") != nil {
		t.Fatal("valid snapshot rejected")
	}
	if validateFrame(wireFrame{Type: "output", Sequence: &seq}, false, 0, "term") != nil {
		t.Fatal("valid next sequence rejected")
	}
}

func TestRetainedExplicitCreationIsNotRetried(t *testing.T) {
	s := new(Store)
	r := fixture()
	r.createErr = errors.New("response lost")
	s.bind(r, r, true, nil)
	defer s.bind(nil, nil, false, nil)
	s.Show("session", true)
	s.Create(s.Snapshot("session").Owner, "session")
	v := wait(t, s, "session", func(v View) bool { return !v.Busy && v.ActiveID != "" && v.Error != "" })
	id := v.ActiveID
	s.transport(r, false)
	s.transport(r, true)
	if r.count("terminal/create") != 1 {
		t.Fatal("creation automatically retried")
	}
	select {
	case op := <-r.opens:
		t.Fatalf("unconfirmed creation followed: %v", op)
	default:
	}
	s.Close(v.Owner, "session", id)
	wait(t, s, "session", func(v View) bool { return v.Phase == "closed" })
	r.mu.Lock()
	last := r.calls[len(r.calls)-1]
	r.mu.Unlock()
	if last.method != "terminal/close" || last.args["id"] != id {
		t.Fatal("uncertain creation not closable", last)
	}
}

func TestRetainedStaleUIActionCannotTargetReplacedHost(t *testing.T) {
	s, r := prepare(t)
	old := s.Snapshot("session").Owner
	s.bind(nil, nil, false, nil)
	r2 := fixture()
	s.bind(r2, r2, true, nil)
	defer s.bind(nil, nil, false, nil)
	s.Show("session", true)
	s.Refresh(old, "session")
	s.Create(old, "session")
	s.Connect(old, "session", "term")
	s.Close(old, "session", "term")
	if len(r2.calls) != 0 {
		t.Fatal("stale UI action reached replacement Host")
	}
	s.transport(r, true)
	if s.owner != r2 {
		t.Fatal("old carrier changed Host")
	}
}

func TestRetainedRenderBarrierAndReadonlyGrid(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	hold := nextOpen(t, r, "terminal/retain")
	hold.stream.send(map[string]any{"type": "retained"})
	f := nextOpen(t, r, "terminal/follow")
	attachmentID, _ := f.args["attachmentId"].(string)
	f.stream.send(map[string]any{"type": "snapshot", "sequence": 3, "screen": "\x1b[4;5Hcursor", "info": Info{ID: "term", State: "running", Cols: 80, Rows: 24, ControllerID: attachmentID}})
	v := wait(t, s, "session", func(v View) bool { return v.Render != nil })
	a := v.Attachment
	if cols, rows := a.TerminalSize(20, 5); cols != 80 || rows != 24 {
		t.Fatalf("recovery grid %d x %d", cols, rows)
	}
	f.stream.send(map[string]any{"type": "output", "sequence": 4, "data": "\x1b[1D!"})
	if s.Snapshot("session").Render.Revision != v.Render.Revision {
		t.Fatal("overwrote unacknowledged snapshot")
	}
	s.Acknowledge(a, v.Render.Revision)
	v = wait(t, s, "session", func(v View) bool { return v.Render != nil && !v.Render.Snapshot })
	if v.Render.Data != "\x1b[1D!" {
		t.Fatal("changed cursor bytes")
	}
	s.Acknowledge(a, v.Render.Revision)
	if cols, rows := a.TerminalSize(9999, 9999); cols != 160 || rows != 60 {
		t.Fatal("unbounded writable grid")
	}
	f.stream.send(map[string]any{"type": "state", "info": Info{ID: "term", State: "running", Cols: 70, Rows: 22}})
	wait(t, s, "session", func(v View) bool { return !v.Writable })
	if cols, rows := a.TerminalSize(20, 5); cols != 70 || rows != 22 {
		t.Fatal("read-only grid lost Host cursor geometry")
	}
	s.Show("other", true)
	select {
	case <-hold.stream.canceled:
		t.Fatal("hide released retention")
	default:
	}
	if r.count("terminal/close") != 0 {
		t.Fatal("hide closed process")
	}
}

func TestRetainedInputLimitAndCloseFailure(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	a := attach(t, s, r, 0)
	if _, err := a.Write([]byte("123456789")); err == nil {
		t.Fatal("oversized input accepted")
	}
	if r.count("terminal/write") != 0 {
		t.Fatal("oversized input transmitted")
	}
	r.mu.Lock()
	r.closeErr = errors.New("response lost")
	r.mu.Unlock()
	s.Close(s.Snapshot("session").Owner, "session", "term")
	wait(t, s, "session", func(v View) bool { return v.Phase == "failed" })
	if r.count("terminal/close") != 1 {
		t.Fatal("close automatically replayed")
	}
	if _, err := a.Write([]byte("x")); err == nil {
		t.Fatal("old capability survived close attempt")
	}
}

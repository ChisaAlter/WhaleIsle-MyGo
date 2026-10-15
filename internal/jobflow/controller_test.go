package jobflow

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

type fixtureStream struct{ frames chan json.RawMessage }

func (s *fixtureStream) Recv(ctx context.Context) (json.RawMessage, error) {
	select {
	case p := <-s.frames:
		return p, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (*fixtureStream) Cancel() {}

type fixtureOpen struct {
	method  string
	request map[string]any
	stream  *fixtureStream
}
type fixtureTransport struct {
	opened  chan fixtureOpen
	killed  chan map[string]any
	killErr error
}

func (f *fixtureTransport) Open(_ context.Context, method string, request map[string]any) (Stream, error) {
	s := &fixtureStream{make(chan json.RawMessage, 8)}
	f.opened <- fixtureOpen{method, request, s}
	return s, nil
}
func (f *fixtureTransport) Kill(_ context.Context, request map[string]any) (string, error) {
	f.killed <- request
	return "requested", f.killErr
}
func (*fixtureTransport) Retryable(error) bool { return false }
func waitState(t *testing.T, c *Controller, pred func(Snapshot) bool) Snapshot {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s := c.Snapshot()
		if pred(s) {
			return s
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("state did not converge")
	return Snapshot{}
}
func nextOpen(t *testing.T, f *fixtureTransport) fixtureOpen {
	t.Helper()
	select {
	case o := <-f.opened:
		return o
	case <-time.After(time.Second):
		t.Fatal("stream not opened")
		return fixtureOpen{}
	}
}
func fixture() (*Controller, *fixtureTransport, uint64) {
	f := &fixtureTransport{opened: make(chan fixtureOpen, 8), killed: make(chan map[string]any, 8)}
	c := New(nil)
	c.Bind(f, "session-a")
	g := c.Snapshot().Generation
	c.Open(g)
	return c, f, g
}

func TestJobsFollowKillAndScopeFence(t *testing.T) {
	c, f, g := fixture()
	defer c.Bind(nil, "")
	rows := nextOpen(t, f)
	if rows.method != "job/list" || rows.request["sessionId"] != "session-a" {
		t.Fatal(rows)
	}
	rows.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"bash-1","owner":"session-a","label":"Build","status":"running","output":{"total":0,"earliest":0}}]}`)
	waitState(t, c, func(s Snapshot) bool { return len(s.Rows) == 1 })
	c.Toggle(g, "bash-1")
	follow := nextOpen(t, f)
	if follow.method != "job/follow" || follow.request["jobId"] != "bash-1" || follow.request["sessionId"] != "session-a" {
		t.Fatal(follow)
	}
	follow.stream.frames <- json.RawMessage(`{"type":"opened","job":{"id":"bash-1","status":"running","output":{"earliest":2}},"from":2}`)
	follow.stream.frames <- json.RawMessage(`{"type":"output","chunks":[{"at":2,"text":"日志 ✓\n"}],"next":13}`)
	waitState(t, c, func(s Snapshot) bool { return s.Observed.Text == "日志 ✓\n" && s.Observed.GapBefore })
	c.ArmKill(g, "bash-1")
	c.ConfirmKill(g, "bash-1")
	select {
	case request := <-f.killed:
		if request["jobId"] != "bash-1" || request["sessionId"] != "session-a" {
			t.Fatal(request)
		}
	case <-time.After(time.Second):
		t.Fatal("kill missing")
	}
	waitState(t, c, func(s Snapshot) bool { return s.Kill.Phase == "pending" })
	c.ConfirmKill(g, "bash-1")
	select {
	case <-f.killed:
		t.Fatal("duplicate kill")
	default:
	}
	rows.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"bash-1","status":"killed","output":{"total":13}}]}`)
	waitState(t, c, func(s Snapshot) bool { return s.Kill.Phase == "" && s.Rows[0].Status == "killed" })
	c.Bind(f, "session-b")
	c.Open(g)
	c.ArmKill(g, "bash-1")
	c.ConfirmKill(g, "bash-1")
	follow.stream.frames <- json.RawMessage(`{"type":"output","chunks":[{"text":"late"}],"next":17}`)
	s := c.Snapshot()
	if s.Open || len(s.Rows) != 0 || s.Observed.Text != "" {
		t.Fatalf("stale scope: %+v", s)
	}
	select {
	case <-f.killed:
		t.Fatal("stale scope killed job")
	default:
	}
}

func TestOutputRetentionAndKillFailure(t *testing.T) {
	var o Observation
	if err := o.Apply(Frame{Type: "opened", Job: Job{ID: "x"}, From: 0}); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("界", RenderTailLimit)
	if err := o.Apply(Frame{Type: "output", Chunks: []Chunk{{Text: long}}, Next: int64(len(long))}); err != nil {
		t.Fatal(err)
	}
	if !o.GapBefore || len(o.Text) > RenderTailLimit || strings.Contains(o.Text, "�") {
		t.Fatal("invalid bounded UTF-8 tail")
	}
	if err := o.Apply(Frame{Type: "status", Job: Job{ID: "x", Status: "failed"}}); err != nil {
		t.Fatal(err)
	}
	if o.Streaming {
		t.Fatal("terminal output still streaming")
	}
	c, f, g := fixture()
	defer c.Bind(nil, "")
	f.killErr = errors.New("host refused")
	r := nextOpen(t, f)
	r.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"x","status":"running"}]}`)
	waitState(t, c, func(s Snapshot) bool { return len(s.Rows) > 0 })
	c.ArmKill(g, "x")
	c.ConfirmKill(g, "x")
	s := waitState(t, c, func(s Snapshot) bool { return s.Kill.Phase == "failed" })
	if s.Kill.Error != "host refused" || s.Rows[0].Status != "running" {
		t.Fatal(s)
	}
}

// Unowned global jobs omit sessionId even while listed in a session.
func TestUnownedJobFollowOmitsSessionAndRetryRetainsCursor(t *testing.T) {
	c, f, g := fixture()
	defer c.Bind(nil, "")
	r := nextOpen(t, f)
	r.stream.frames <- json.RawMessage(`{"type":"rows","jobs":[{"id":"global-1","status":"running","output":{"total":4}}]}`)
	waitState(t, c, func(s Snapshot) bool { return len(s.Rows) == 1 })
	c.Toggle(g, "global-1")
	o := nextOpen(t, f)
	if _, ok := o.request["sessionId"]; ok {
		t.Fatalf("unowned follow sent sessionId: %#v", o.request)
	}
	o.stream.frames <- json.RawMessage(`{"type":"opened","job":{"id":"global-1","status":"running","output":{"earliest":0}},"from":0}`)
	o.stream.frames <- json.RawMessage(`{"type":"output","chunks":[{"at":0,"text":"tail"}],"next":4}`)
	waitState(t, c, func(s Snapshot) bool { return s.Observed.Text == "tail" })
	c.RetryOutput(g)
	retry := nextOpen(t, f)
	if _, ok := retry.request["sessionId"]; ok {
		t.Fatalf("unowned retry sent sessionId: %#v", retry.request)
	}
	if retry.request["from"] != int64(4) {
		t.Fatalf("lost resume cursor: %#v", retry.request)
	}
	if c.Snapshot().Observed.Text != "tail" {
		t.Fatal("retry discarded accepted output")
	}
	c.Close(g)
	c.Open(g)
	if c.Snapshot().Open {
		t.Fatal("closed popup generation was reusable")
	}
}

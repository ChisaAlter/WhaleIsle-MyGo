package hostterminal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type renameFixtureRemote struct {
	*fixtureRemote
	requests chan map[string]any
	results  chan error
}

func renameFixture(t *testing.T) (*Store, *renameFixtureRemote, uint64) {
	t.Helper()
	r := &renameFixtureRemote{fixture(), make(chan map[string]any, 4), make(chan error, 4)}
	s := new(Store)
	s.bind(r, r, true, nil)
	s.Show("session", true)
	owner := s.Snapshot("session").Owner
	s.Refresh(owner, "session")
	wait(t, s, "session", func(v View) bool { return len(v.Items) == 1 && !v.Busy })
	s.mu.Lock()
	s.sessions["session"].active = "term"
	s.entries[terminalKey{"session", "term"}].info.Title = "original"
	s.sessions["session"].items[0].Title = "original"
	s.mu.Unlock()
	return s, r, owner
}
func (r *renameFixtureRemote) Call(ctx context.Context, method string, args, out any) error {
	if method == "terminal/rename" {
		r.requests <- object(args)
		return <-r.results
	}
	return r.fixtureRemote.Call(ctx, method, args, out)
}
func nextRename(t *testing.T, r *renameFixtureRemote) map[string]any {
	t.Helper()
	select {
	case req := <-r.requests:
		return req
	case <-time.After(time.Second):
		t.Fatal("rename not sent")
	}
	return nil
}
func waitRename(t *testing.T, s *Store, owner uint64, pred func(RenameView) bool) RenameView {
	t.Helper()
	limit := time.After(time.Second)
	for {
		v := s.RenameSnapshot(owner, "session", "term")
		if pred(v) {
			return v
		}
		select {
		case <-limit:
			t.Fatalf("rename state: %+v", v)
		case <-time.After(time.Millisecond):
		}
	}
}
func TestHostTerminalRenameTargetsCapturedIdentity(t *testing.T) {
	s, r, o := renameFixture(t)
	defer s.bind(nil, nil, false, nil)
	s.BeginRename(o, "session", "term")
	s.EditRename(o, "session", "term", "  工作 🐋  ")
	s.SaveRename(o, "session", "term")
	req := nextRename(t, r)
	if len(req) != 3 || req["agentId"] != "session" || req["id"] != "term" || req["title"] != "工作 🐋" {
		t.Fatal(req)
	}
	r.results <- nil
	waitRename(t, s, o, func(v RenameView) bool { return !v.Editing && !v.Saving })
	v := s.Snapshot("session")
	if v.Info.Title != "工作 🐋" || v.Items[0].Title != "工作 🐋" {
		t.Fatal(v)
	}
}
func TestHostTerminalRenameFailurePreservesDraftAndNeedsExplicitRetry(t *testing.T) {
	s, r, o := renameFixture(t)
	defer s.bind(nil, nil, false, nil)
	s.BeginRename(o, "session", "term")
	s.EditRename(o, "session", "term", "keep my draft")
	s.SaveRename(o, "session", "term")
	nextRename(t, r)
	s.SaveRename(o, "session", "term")
	r.results <- errors.New("response lost")
	v := waitRename(t, s, o, func(v RenameView) bool { return v.Error != "" && !v.Saving })
	if !v.Editing || v.Draft != "keep my draft" || s.Snapshot("session").Info.Title != "original" {
		t.Fatal(v)
	}
	select {
	case req := <-r.requests:
		t.Fatal("automatic retry", req)
	default:
	}
	s.SaveRename(o, "session", "term")
	nextRename(t, r)
	r.results <- nil
	waitRename(t, s, o, func(v RenameView) bool { return !v.Editing })
}
func TestHostTerminalRenameValidatesUTF16AndCancel(t *testing.T) {
	s, r, o := renameFixture(t)
	defer s.bind(nil, nil, false, nil)
	for _, title := range []string{" \ufeff ", strings.Repeat("x", 121), strings.Repeat("🐋", 61)} {
		s.BeginRename(o, "session", "term")
		s.EditRename(o, "session", "term", title)
		s.SaveRename(o, "session", "term")
		v := s.RenameSnapshot(o, "session", "term")
		if v.Error == "" || v.Draft != title || v.Saving {
			t.Fatal(v)
		}
		select {
		case req := <-r.requests:
			t.Fatal("invalid request", req)
		default:
		}
		s.CancelRename(o, "session", "term")
	}
	s.BeginRename(o, "session", "term")
	s.EditRename(o, "session", "term", strings.Repeat("🐋", 60))
	s.SaveRename(o, "session", "term")
	nextRename(t, r)
	r.results <- nil
	waitRename(t, s, o, func(v RenameView) bool { return !v.Editing })
	s.BeginRename(o, "session", "term")
	s.EditRename(o, "session", "term", "cancel")
	s.CancelRename(o, "session", "term")
	if s.RenameSnapshot(o, "session", "term").Editing {
		t.Fatal("cancel remained open")
	}
}
func TestHostTerminalRenameRejectsReplacedHostAndLateResult(t *testing.T) {
	s, r, o := renameFixture(t)
	s.BeginRename(o, "session", "term")
	s.EditRename(o, "session", "term", "old Host")
	s.SaveRename(o, "session", "term")
	nextRename(t, r)
	s.bind(nil, nil, false, nil)
	r2 := fixture()
	s.bind(r2, r2, true, nil)
	defer s.bind(nil, nil, false, nil)
	s.Show("session", true)
	s.BeginRename(o, "session", "term")
	s.EditRename(o, "session", "term", "stale")
	s.SaveRename(o, "session", "term")
	r.results <- nil
	if s.RenameSnapshot(o, "session", "term").Editing || len(s.Snapshot("session").Items) != 0 || len(r2.calls) != 0 {
		t.Fatal("old rename crossed Host")
	}
}
func TestHostTerminalRenameCloseRetainsDraftAndIgnoresLateSuccess(t *testing.T) {
	s, r, o := renameFixture(t)
	defer s.bind(nil, nil, false, nil)
	s.BeginRename(o, "session", "term")
	s.EditRename(o, "session", "term", "pending")
	s.SaveRename(o, "session", "term")
	nextRename(t, r)
	s.Close(o, "session", "term")
	wait(t, s, "session", func(v View) bool { return v.Phase == "closed" })
	r.results <- nil
	v := s.RenameSnapshot(o, "session", "term")
	if v.Saving || v.Draft != "pending" || s.Snapshot("session").Info.Title != "original" {
		t.Fatal(v)
	}
}

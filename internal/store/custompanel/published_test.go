package custompanel

import (
	"io"
	"sync"
	"testing"
	"time"
)

func testPanel(t *testing.T) Definition {
	t.Helper()
	d, err := Parse(`{"title":"Daily","width":420,"height":320,"columns":1,"widgets":[{"id":"note","kind":"input","label":"Notes","text":"seed"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSnapshotSharedUntilWrite(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := Scope{Host: "h", Bot: "b"}
	if _, err = s.Create(scope, testPanel(t)); err != nil {
		t.Fatal(err)
	}
	first := s.Snapshot(scope)
	if first != s.Snapshot(scope) {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot(scope) }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
}

func TestWritePublishesAndSignals(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := Scope{Host: "h", Bot: "b"}
	id, err := s.Create(scope, testPanel(t))
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot(scope)
	changed := s.views[scope].Changed()
	if err = s.Remove(scope, id); err != nil {
		t.Fatal(err)
	}
	after := s.Snapshot(scope)
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("Remove did not publish Rev+1: %p/%d -> %p/%d", before, before.Rev, after, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
}

func TestPersistOutsideStateLock(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := Scope{Host: "h", Bot: "b"}
	id, err := s.Create(scope, testPanel(t))
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	done := make(chan error, 1)
	s.writeFn = func(document) error {
		<-release
		return nil
	}
	go func() { done <- s.Remove(scope, id) }()
	read := make(chan struct{})
	go func() {
		defer close(read)
		v := s.Snapshot(scope)
		if len(v.Value.Instances) != 1 {
			t.Error("blocked write must leave the previous view visible")
		}
	}()
	select {
	case <-read:
	case <-time.After(2 * time.Second):
		t.Fatal("Snapshot blocked behind disk persistence")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if n := len(s.Snapshot(scope).Value.Instances); n != 0 {
		t.Fatalf("publish after successful persist left %d instances", n)
	}
}

func TestPersistFailureKeepsState(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := Scope{Host: "h", Bot: "b"}
	id, err := s.Create(scope, testPanel(t))
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot(scope)
	s.writeFn = func(document) error { return io.EOF }
	if err = s.Remove(scope, id); err == nil {
		t.Fatal("failed persist must report the error")
	}
	if s.Snapshot(scope) != before {
		t.Fatal("failed persist must not publish")
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	scope := Scope{Host: "h", Bot: "b"}
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = s.Snapshot(scope).Value
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			id, err := s.Create(scope, testPanel(t))
			if err != nil {
				continue
			}
			_ = s.Remove(scope, id)
		}
	}()
	wg.Wait()
}

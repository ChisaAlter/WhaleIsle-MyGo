package pad

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

func TestPadSnapshotSharedUntilWrite(t *testing.T) {
	s, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	first := s.Snapshot()
	if first != s.Snapshot() {
		t.Fatal("unchanged state must hand out the same *Version")
	}
	if got := testing.AllocsPerRun(100, func() { s.Snapshot() }); got != 0 {
		t.Fatalf("Snapshot allocates %g per run", got)
	}
}

func TestPadWritePublishesAndSignals(t *testing.T) {
	s, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	changed := s.Changed()
	s.SetError(errors.New("probe"))
	after := s.Snapshot()
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("write did not publish Rev+1: %p/%d -> %p/%d", before, before.Rev, after, after.Rev)
	}
	select {
	case <-changed:
	default:
		t.Fatal("Changed channel did not close on publish")
	}
	if after.Value.Error == nil || after.Value.Error.SentenceText() == "" {
		t.Fatal("error did not publish as a Fault")
	}
}

func TestPadFaultCarriesNoRawText(t *testing.T) {
	s, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.SetError(errors.New("read tcp 127.0.0.1: wsarecv raw internals"))
	got := s.Snapshot().Value.Error
	if got == nil {
		t.Fatal("Fault missing")
	}
	if got.SentenceText() == "read tcp 127.0.0.1: wsarecv raw internals" {
		t.Fatal("raw error text reached the view")
	}
	s.SetError(nil)
	if s.Snapshot().Value.Error != nil {
		t.Fatal("clearing the error did not publish nil Fault")
	}
	s.SetPanelError("p", errors.New("panel blew up"))
	if s.Snapshot().Value.PanelErrors["p"] == nil {
		t.Fatal("panel error not mapped to Fault")
	}
}

func TestPadPersistOutsideStateLock(t *testing.T) {
	s, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	s.writeFn = func(string, Layout) error {
		<-release
		return nil
	}
	done := make(chan struct{})
	go func() {
		s.UpdateLayout(func(l *Layout) { l.Width = 480 })
		close(done)
	}()
	read := make(chan struct{})
	go func() {
		defer close(read)
		_ = s.Snapshot()
	}()
	select {
	case <-read:
	case <-time.After(2 * time.Second):
		t.Fatal("Snapshot blocked behind disk persistence")
	}
	close(release)
	<-done
	if s.Snapshot().Value.Layout.Width != 480 {
		t.Fatal("layout update not published after persist")
	}
}

func TestPadPersistFailureKeepsState(t *testing.T) {
	s, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.writeFn = func(string, Layout) error { return io.EOF }
	s.UpdateLayout(func(l *Layout) { l.Width = 480 })
	// The layout already published (persist failures surface via Error);
	// the failure itself is a classified Fault, not a crash or a hang.
	if s.Snapshot().Value.Error == nil {
		t.Fatal("persist failure not surfaced as a Fault")
	}
}

func TestPadConcurrentReadWrite(t *testing.T) {
	s, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = s.Snapshot().Value
			}
		}()
	}
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				s.SetError(errors.New("x"))
				s.SetPanelError("p", errors.New("y"))
			}
		}(n)
	}
	wg.Wait()
}

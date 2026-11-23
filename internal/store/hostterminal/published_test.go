package hostterminal

import (
	"sync"
	"testing"
)

func TestSnapshotSharedWhenUnchanged(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	_ = r
	a := s.Snapshot("session")
	if b := s.Snapshot("session"); a != b {
		t.Fatal("unchanged Snapshot must return the same published version")
	}
	allocs := testing.AllocsPerRun(50, func() { _ = s.Snapshot("session") })
	if allocs != 0 {
		t.Fatalf("unchanged Snapshot allocs = %v", allocs)
	}
}

func TestPublishAdvancesRevision(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	_ = r
	before := s.Snapshot("session")
	s.Show("other", true)
	after := s.Snapshot("session")
	if after == before || after.Rev != before.Rev+1 {
		t.Fatalf("rev %d -> %d", before.Rev, after.Rev)
	}
}

func TestConcurrentSnapshotsAndMutations(t *testing.T) {
	s, r := prepare(t)
	defer s.bind(nil, nil, false, nil)
	_ = r
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = s.Snapshot("session").Value.Items
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				s.Show("session", i%2 == 0)
			}
		}(i)
	}
	wg.Wait()
}

package session

import (
	"fmt"
	"reflect"
	"testing"
)

func TestArchiveConfirmationViewCopiesCurrentDialogOnly(t *testing.T) {
	s := benchmarkStore(30)
	if got := s.ArchiveConfirmation(); got != nil {
		t.Fatal("missing refusal produced a dialog")
	}
	s.mu.Lock()
	s.archive = archiveOperation{waiting: true, confirmation: ArchiveConfirmation{Token: 7, SessionID: "bench", Title: "Current", Activity: []ArchiveActivity{{Kind: "job", Items: []ArchiveActivityItem{{ID: "j", Label: "Job"}}}}, Error: "retry"}}
	s.invalidateViewLocked()
	s.mu.Unlock()
	ordinary := s.Snapshot()
	first := s.ArchiveConfirmation()
	if !reflect.DeepEqual(first, ordinary.ArchiveConfirmation) {
		t.Fatal("narrow getter differs from ordinary Snapshot dialog")
	}
	builds, hits := s.SnapshotStats()
	first.Token = 99
	first.Title = "mutated"
	first.Activity[0].Kind = "mutated"
	first.Activity[0].Items[0].Label = "mutated"
	ordinary.ArchiveConfirmation.Activity[0].Items[0].ID = "mutated"
	current := s.ArchiveConfirmation()
	if current.Token != 7 || current.Title != "Current" || current.Activity[0].Kind != "job" || current.Activity[0].Items[0].ID != "j" || current.Activity[0].Items[0].Label != "Job" {
		t.Fatalf("caller mutation leaked into current narrow view: %+v", current)
	}
	s.mu.Lock()
	s.archive.confirmation.Token = 8
	s.archive.confirmation.Busy = true
	s.archive.confirmation.Error = "current error"
	s.mu.Unlock()
	current = s.ArchiveConfirmation()
	if current.Token != 8 || !current.Busy || current.Error != "current error" {
		t.Fatal("getter reused a prior snapshot instead of current locked state")
	}
	s.CancelArchive(7) // Stale token must not clear the current request.
	if s.ArchiveConfirmation() == nil {
		t.Fatal("stale dialog token cleared current request")
	}
	s.mu.Lock()
	s.archive.confirmation.Busy = false
	s.mu.Unlock()
	// Reads above must neither build nor hit the full conversation projection.
	if afterBuilds, afterHits := s.SnapshotStats(); afterBuilds != builds || afterHits != hits {
		t.Fatal("narrow getter built or cloned the conversation snapshot")
	}
	s.CancelArchive(8)
	if s.ArchiveConfirmation() != nil {
		t.Fatal("current token failed to dismiss dialog")
	}
}

func TestSelectedSessionIDTracksCurrentSelectionWithoutProjection(t *testing.T) {
	s := New()
	for _, id := range []string{"a", "b", ""} {
		s.Select(id)
		builds, hits := s.SnapshotStats()
		if got := s.SelectedSessionID(); got != id {
			t.Fatalf("selection=%q want %q", got, id)
		}
		if afterBuilds, afterHits := s.SnapshotStats(); afterBuilds != builds || afterHits != hits {
			t.Fatal("selection getter projected full history")
		}
	}
}

var narrowSelectionSink string
var narrowArchiveSink *ArchiveConfirmation

func BenchmarkNarrowFrameReads(b *testing.B) {
	for _, size := range []int{2000, 8000} {
		b.Run(fmt.Sprintf("rows_%d", size), func(b *testing.B) {
			s := benchmarkStore(size)
			_ = s.Snapshot()
			for _, name := range []string{"snapshot_selected", "selected_id", "snapshot_archive_nil", "archive_nil", "old_navigation_pair_archive", "narrow_navigation_pair_archive"} {
				b.Run(name, func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for range b.N {
						switch name {
						case "snapshot_selected":
							narrowSelectionSink = s.Snapshot().SelectedID
						case "selected_id":
							narrowSelectionSink = s.SelectedSessionID()
						case "snapshot_archive_nil":
							narrowArchiveSink = s.Snapshot().ArchiveConfirmation
						case "archive_nil":
							narrowArchiveSink = s.ArchiveConfirmation()
						case "old_navigation_pair_archive":
							narrowSelectionSink = s.Snapshot().SelectedID
							narrowSelectionSink = s.Snapshot().SelectedID
							narrowArchiveSink = s.Snapshot().ArchiveConfirmation
						case "narrow_navigation_pair_archive":
							narrowSelectionSink = s.SelectedSessionID()
							narrowSelectionSink = s.SelectedSessionID()
							narrowArchiveSink = s.ArchiveConfirmation()
						}
					}
				})
			}
		})
	}
}

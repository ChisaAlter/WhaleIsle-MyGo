package session

import (
	"testing"

	"whaleisle/internal/dsh"
)

func TestNativeModelLabelGenerationRetiresBetweenUIFrames(t *testing.T) {
	s := modelTestStore(nil)
	label := dsh.NativeContributions{Known: true, ModelCommand: true, ModelLabel1017: true}
	s.setNativeContributionsLocked(label)
	before := s.Snapshot()
	// A renderer-only graph removal/reinstatement can occur between view
	// frames. Reusing the same code must still create a new animation owner.
	withdrawn := label
	withdrawn.ModelLabel1017 = false
	s.setNativeContributionsLocked(withdrawn)
	s.setNativeContributionsLocked(label)
	after := s.Snapshot()
	if after.NativeModelLabelGeneration == before.NativeModelLabelGeneration {
		t.Fatal("withdrawn label presentation state survived re-registration")
	}
	if after.NativeContributionsGeneration != before.NativeContributionsGeneration || after.NativeCommandGeneration("model") != before.NativeCommandGeneration("model") {
		t.Fatal("renderer-only retirement changed unrelated model-command ownership")
	}
	label.Revision = "unrelated-graph-publication"
	s.setNativeContributionsLocked(label)
	if s.Snapshot().NativeModelLabelGeneration != after.NativeModelLabelGeneration {
		t.Fatal("global graph revision retired an unchanged renderer")
	}
}

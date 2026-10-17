package session

import (
	"encoding/json"
	"testing"
	"whaleisle/internal/dsh"
)

func TestSkillOptionsDiscardedOnSessionChange(t *testing.T) {
	s := New()
	s.selectedID = "a"
	s.skillOptions = []SkillOption{{Name: "old"}}
	s.selectSessionLocked("b")
	if len(s.skillOptions) != 0 {
		t.Fatal("old-session skill options retained")
	}
}
func TestSkillOptionsDiscardedOnPresetEvent(t *testing.T) {
	s := New()
	s.selectedID = "a"
	s.skillOptions = []SkillOption{{Name: "old"}}
	s.applyPersistenceEventLocked(json.RawMessage(`{"type":"agent-preset/selected","seq":1,"data":{"agentPreset":"next"}}`))
	if len(s.skillOptions) != 0 {
		t.Fatal("old-preset skill options retained")
	}
}

func TestSkillOptionsScopeRejectsOldEpochAndHost(t *testing.T) {
	s := New()
	s.client = &dsh.Client{}
	s.selectedID = "a"
	owner := s.captureSkillOptionsScopeLocked()
	s.skillOptionsOwner = owner
	s.skillOptions = []SkillOption{{Name: "fresh"}}
	if len(s.currentSkillOptionsLocked()) != 1 {
		t.Fatal("current choices hidden")
	}
	copy := s.currentSkillOptionsLocked()
	copy[0].Name = "mutated"
	if s.skillOptions[0].Name != "fresh" {
		t.Fatal("shared mutable snapshot")
	}
	s.invalidateSkillOptionsLocked()
	if s.skillOptionsScopeCurrentLocked(owner) || len(s.currentSkillOptionsLocked()) != 0 {
		t.Fatal("old preset response accepted")
	}
	owner = s.captureSkillOptionsScopeLocked()
	s.eventsGeneration++
	if s.skillOptionsScopeCurrentLocked(owner) {
		t.Fatal("old host event generation accepted")
	}
	owner = s.captureSkillOptionsScopeLocked()
	s.cwd = "other"
	if s.skillOptionsScopeCurrentLocked(owner) {
		t.Fatal("old workspace accepted")
	}
}

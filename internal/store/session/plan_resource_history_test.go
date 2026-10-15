package session

import (
	"fmt"
	"testing"
	"whaleisle/internal/dsh"
)

func TestPlanHistoryBindingOwnerFixture(t *testing.T) {
	s := New()
	first, second := &dsh.Client{}, &dsh.Client{}
	s.Attach(first, nil)
	s.Select("same-session")
	if s.Snapshot().PlanHistoryBinding != "" {
		t.Fatal("new client had an authoritative history binding")
	}
	s.applyFollowFrame([]byte(`{"type":"snapshot","records":[],"cursor":0,"hasMore":false}`))
	expected := fmt.Sprintf("%p\x00same-session", first)
	if got := s.Snapshot().PlanHistoryBinding; got != expected {
		t.Fatalf("snapshot binding %q, want %q", got, expected)
	}
	s.Attach(second, nil)
	if s.Snapshot().PlanHistoryBinding != "" {
		t.Fatal("replaced Host retained authoritative history")
	}
	s.applyFollowFrame([]byte(`{"type":"snapshot","records":[],"cursor":0,"hasMore":false}`))
	expected = fmt.Sprintf("%p\x00same-session", second)
	if got := s.Snapshot().PlanHistoryBinding; got != expected {
		t.Fatalf("replacement snapshot binding %q, want %q", got, expected)
	}
	s.Select("other-session")
	if s.Snapshot().PlanHistoryBinding != "" {
		t.Fatal("session switch retained earlier history binding")
	}
	s.Detach()
	if s.Snapshot().PlanHistoryBinding != "" {
		t.Fatal("detached Host retained history binding")
	}
}

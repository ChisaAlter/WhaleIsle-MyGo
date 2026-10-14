package terminal

import (
	"testing"
)

func TestOpenEmptyWorkspace(t *testing.T) {
	p := Open("", false)
	if p.Err == nil {
		t.Fatal("expected error for empty workspace")
	}
	if p.Term != nil {
		t.Fatal("expected no terminal instance")
	}
}

func TestPanelWorkspace(t *testing.T) {
	p := Open("  C:\\work  ", false)
	if p.Workspace() != "C:\\work" {
		t.Fatalf("workspace = %q", p.Workspace())
	}
	p.Close()
}

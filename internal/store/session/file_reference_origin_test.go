package session

import (
	"testing"
	"whaleisle/internal/dsh"
)

func TestWorkspaceFileReferenceOriginTracksHostAndSelectedCwd(t *testing.T) {
	a, b := &dsh.Client{}, &dsh.Client{}
	s := New()
	s.client = a
	s.selectedID = "same"
	s.cwd = "/configured"
	s.items = []Item{{SessionID: "same", Cwd: "/scratch"}}
	origin, id, cwd := s.WorkspaceFileReferenceOrigin()
	if origin == "" || id != "same" || cwd != "/scratch" {
		t.Fatalf("origin: %q %q %q", origin, id, cwd)
	}
	s.client = b
	if next, _, _ := s.WorkspaceFileReferenceOrigin(); next == origin {
		t.Fatal("Host switch reused binding")
	}
	s.client = nil
	if next, _, _ := s.WorkspaceFileReferenceOrigin(); next != "" {
		t.Fatal("detached Host retained origin")
	}
	s.client = a
	s.items = nil
	if next, _, _ := s.WorkspaceFileReferenceOrigin(); next != "" {
		t.Fatal("configured root substituted for absent session cwd")
	}
}

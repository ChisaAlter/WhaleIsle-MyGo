package htmldocument

import "testing"

func TestReadRemoteRejectsAbsoluteAndTraversalPaths(t *testing.T) {
	called := 0
	g := &resourceGraph{
		valid: func() bool { return true },
		remoteRead: func(assetPath string, limit int) ([]byte, error) {
			called++
			return []byte("body { color: #111; }"), nil
		},
		files:  map[string]resourceFile{},
		active: map[string]bool{},
		report: ResourceReport{},
	}

	rejected := []struct {
		name string
		base string
		ref  string
	}{
		{name: "leading slash base", base: "/ws/doc.html", ref: "style.css"},
		{name: "leading backslash base", base: `\ws\doc.html`, ref: "style.css"},
		{name: "drive volume base", base: `C:\ws\doc.html`, ref: "style.css"},
		{name: "dotdot base", base: "../doc.html", ref: "style.css"},
		{name: "ref traversal past base", base: "sub/doc.html", ref: "../../secret.css"},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			called = 0
			if _, _, _, err := g.readRemote(tc.base, tc.ref, "css"); err == nil {
				t.Fatalf("readRemote(%q, %q) succeeded, want rejection", tc.base, tc.ref)
			}
			if called != 0 {
				t.Fatalf("remoteRead called %d times on rejected path", called)
			}
		})
	}

	_, resolved, _, err := g.readRemote("sub/doc.html", "style.css", "css")
	if err != nil {
		t.Fatalf("valid readRemote failed: %v", err)
	}
	if resolved != "sub/style.css" {
		t.Fatalf("resolved = %q, want %q", resolved, "sub/style.css")
	}
}

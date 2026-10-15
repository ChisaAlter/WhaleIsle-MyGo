package filelink

import "testing"

func TestWorkspaceFileLink(t *testing.T) {
	for _, tc := range []struct{ path, want string }{
		{"src/main.go", "[main.go](src/main.go)"},
		{"资料/my file(1)#?.md", "[my file(1)#?.md](%E8%B5%84%E6%96%99/my%20file%281%29%23%3F.md)"},
		{"docs/a[b].md", `[a\[b\].md](docs/a%5Bb%5D.md)`},
		{`a"b.txt`, `[a"b.txt](a%22b.txt)`},
		{"@user.txt", "[@user.txt](@user.txt)"},
	} {
		got, ok := Serialize(tc.path)
		if !ok || got != tc.want {
			t.Errorf("Serialize(%q) = %q, %v; want %q", tc.path, got, ok, tc.want)
		}
	}
}
func TestRejectInvalidReferences(t *testing.T) {
	for _, p := range []string{"", "/etc/passwd", "../private", "dir/../private", `C:/Windows/file`, `dir\file`, "x\ny", "dir/", ".", "a//b", ".git/config"} {
		if text, ok := Serialize(p); ok {
			t.Errorf("accepted %q: %q", p, text)
		}
	}
}
func TestReferenceBinding(t *testing.T) {
	ref := Reference{BindingID: "host-A/session", SessionID: "session", WorkspaceRoot: "/remote/project", Path: "file.txt"}
	if !ref.Matches("host-A/session", "session", "/remote/project") {
		t.Fatal("matching origin rejected")
	}
	for _, x := range [][3]string{{"host-B/session", "session", "/remote/project"}, {"host-A/session", "other", "/remote/project"}, {"host-A/session", "session", "/local/project"}, {"", "session", "/remote/project"}} {
		if ref.Matches(x[0], x[1], x[2]) {
			t.Fatalf("stale/foreign origin accepted: %v", x)
		}
	}
}
func TestInsertionPreservesDraftBoundaries(t *testing.T) {
	for _, tc := range []struct {
		text       string
		start, end int
		want       string
	}{
		{"前文后文", 6, 6, " [a.go](a.go) "}, {"hello old tail", 6, 9, "[a.go](a.go)"}, {"", 0, 0, "[a.go](a.go) "}, {"hello\n", 6, 6, "[a.go](a.go) "},
	} {
		got, ok := Insertion(tc.text, tc.start, tc.end, "a.go")
		if !ok || got != tc.want {
			t.Errorf("insertion = %q %v; want %q", got, ok, tc.want)
		}
	}
	if _, ok := Insertion("中", 1, 1, "a.go"); ok {
		t.Fatal("accepted split UTF-8")
	}
}

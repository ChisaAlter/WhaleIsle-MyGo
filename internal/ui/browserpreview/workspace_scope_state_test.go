package browserpreview

import "testing"

func TestBrowserWorkspacePartitions(t *testing.T) {
	makeScope := func(owner WorkspaceOwner) browserWorkspace {
		t.Helper()
		s, err := makeBrowserWorkspace(owner)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	first := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:fixture", Cwd: "/host/project", Title: "One"})
	reopened := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:fixture", Cwd: "/host/project", Title: "Renamed"})
	if first.key != reopened.key {
		t.Fatal("restart or rename must keep the same persistent partition")
	}
	other := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:fixture", Cwd: "/host/other"})
	otherHost := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:other", Cwd: "/host/project"})
	if first.key == other.key || first.key == otherHost.key {
		t.Fatal("workspace and Host namespaces must isolate cookies")
	}
	shared := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:fixture"})
	sharedAgain := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:fixture", Title: "No directory session"})
	namedShared := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:fixture", Cwd: "shared"})
	if shared.key != sharedAgain.key || shared.key == namedShared.key || shared.key == first.key {
		t.Fatal("shared sentinel must be stable and disjoint from workspace names")
	}
	win := makeScope(WorkspaceOwner{Authority: "fixture", Cwd: `C:\Work\Project\`})
	winAgain := makeScope(WorkspaceOwner{Authority: "fixture", Cwd: "c:/work/project"})
	if win.key == winAgain.key {
		t.Fatal("opaque Host cwd must not infer case or separator aliases")
	}
	caseSensitive := makeScope(WorkspaceOwner{Authority: "supervised-dsh-home:fixture", Cwd: "/host/Project"})
	if caseSensitive.key == first.key {
		t.Fatal("distinct Host cwd spellings must remain isolated")
	}
	if _, err := makeBrowserWorkspace(WorkspaceOwner{Cwd: "/unowned"}); err == nil {
		t.Fatal("missing trusted app/Host namespace must fail closed")
	}
	if !first.needsNewTab(other) || first.needsNewTab(reopened) {
		t.Fatal("navigation must not repurpose an old Tab for another partition")
	}
	oldKey := first.key
	_ = first.needsNewTab(other)
	if first.key != oldKey {
		t.Fatal("old Tab ownership must be immutable")
	}
	if len(first.key) != 64 || shared.label != "共享浏览（无工作区）" {
		t.Fatal("invalid partition identity or shared disclosure")
	}
}

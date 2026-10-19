package dirpick

import (
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestBrowserPrefixHostSpellingAndVisibility(t *testing.T) {
	entries := []api.DirectoryListingCrumbsItem{{Name: "Alpha", Path: "/alpha"}, {Name: ".hidden", Path: "/.hidden", Hidden: true}, {Name: "Beta", Path: "/beta"}}
	for _, test := range []struct {
		prefix, selected string
		hidden           bool
		want             int
	}{{"al", "", false, 1}, {".h", "", false, 1}, {"missing", "", false, 2}, {"al", "/.hidden", false, 2}, {"", "", true, 3}} {
		if got := VisibleEntries(entries, test.selected, test.hidden, test.prefix); len(got) != test.want {
			t.Fatalf("filter %+v returned %+v", test, got)
		}
	}
	draft := "/home/../actual/al"
	b := BrowserState{Parent: &api.DirectoryListing{Path: "/actual", Home: "/home"}, PathDraft: &draft, ScannedDirectory: "/home/../actual/", ScannedPath: "/actual"}
	if prefix, ok := b.Prefix(); !ok || prefix != "al" {
		t.Fatal("canonical Host reply triggered an unrelated pane filter")
	}
	draft = "/elsewhere/al"
	if _, ok := b.Prefix(); ok {
		t.Fatal("draft filtered the stale pane")
	}
	windows := api.DirectoryListing{Home: "C:\\Users\\u", Entries: []api.DirectoryListingCrumbsItem{{Name: "Mixed", Path: "C:\\Mixed"}}}
	if MatchEntry(windows, "c:\\MIXED") == nil {
		t.Fatal("Windows actual parent entry not matched case-insensitively")
	}
	if part := DirectoryPart(&windows, "C:/Mixed/al"); part != "C:/Mixed/" {
		t.Fatalf("Windows forward slash directory=%q", part)
	}
	posix := api.DirectoryListing{Home: "/home"}
	if part := DirectoryPart(&posix, "/home/back\\slash"); part != "/home/" {
		t.Fatal("POSIX backslash treated as separator")
	}
	clone := b.Clone()
	*clone.PathDraft = "new"
	if *b.PathDraft == "new" {
		t.Fatal("paint-local draft aliases store draft")
	}
}

package projects

import (
	"reflect"
	"testing"

	"whaleisle/internal/projectflow"
)

func TestProjectSearchHistory(t *testing.T) {
	items := []projectflow.Project{
		{ID: "ready", Title: "Alpha Project", Lifecycle: "ready", CoordinatorSessionID: "lead-active", CanonicalWorkingDirectory: "/fixture/not-a-title"},
		{ID: "archived", Title: "ALPHA Archive", Lifecycle: "archived", CoordinatorSessionID: "lead-archive"},
		{ID: "pending", Title: "Éclair 项目", Lifecycle: "provisioning", Archiving: true},
		{ID: "case", Title: "Gamma", Lifecycle: "Archived"},
		{ID: "empty", Title: "", Lifecycle: ""},
	}
	before := append([]projectflow.Project(nil), items...)
	for _, tc := range []struct {
		name, query string
		history     bool
		want        []string
	}{
		{"default excludes archived only", "", false, []string{"ready", "pending", "case", "empty"}},
		{"history is exact lifecycle", "", true, []string{"archived"}},
		{"title case insensitive", "aLpHa", false, []string{"ready"}},
		{"history and title intersect", "aLpHa", true, []string{"archived"}},
		{"query is not trimmed", " Alpha ", false, nil},
		{"spaces are literal", " ", false, []string{"ready", "pending"}},
		{"unicode title", "éCLAIR", false, []string{"pending"}},
		{"chinese title", "项目", false, []string{"pending"}},
		{"no unicode normalization", "E\u0301clair", false, nil},
		{"not directory", "not-a-title", false, nil},
		{"not identifier", "ready", false, nil},
		{"no result", "missing", true, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, p := range filteredProjects(items, tc.query, tc.history) {
				got = append(got, p.ID)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	if !reflect.DeepEqual(items, before) {
		t.Fatal("filter changed the authoritative listing")
	}
	if got := filteredProjects(nil, "", false); len(got) != 0 {
		t.Fatal("empty input produced rows")
	}
	filtered := filteredProjects(items, "alpha", false)
	filtered[0].Lifecycle = "archived"
	if !reflect.DeepEqual(items, before) {
		t.Fatal("filtered rows alias the authoritative listing")
	}

}

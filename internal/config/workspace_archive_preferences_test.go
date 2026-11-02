package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWorkspaceArchivedFilterPersistenceAndHostIsolation(t *testing.T) {
	s, dir, base := workspaceMergeStore(t)
	for _, filter := range []string{"show", "only", "default"} {
		base.ArchivedFilter = filter
		if err := s.SaveWorkspaceNavigationPreferences("host-a", base); err != nil {
			t.Fatal(err)
		}
		if err := s.SaveWorkspaceNavigationPreferences("host-b", WorkspaceNavigationPreferences{ArchivedFilter: "only"}); err != nil {
			t.Fatal(err)
		}
		restarted, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		got, err := restarted.ReadWorkspaceNavigationPreferences("host-a")
		if err != nil || !reflect.DeepEqual(got, base) {
			t.Fatalf("restart %q: %+v, %v", filter, got, err)
		}
		other, err := restarted.ReadWorkspaceNavigationPreferences("host-b")
		if err != nil || other.ArchivedFilter != "only" {
			t.Fatalf("Host crossed: %+v %v", other, err)
		}
		s = restarted
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(readConfig(t, dir)), &raw); err != nil {
		t.Fatal(err)
	}
	var hosts map[string]map[string]json.RawMessage
	if err := json.Unmarshal(raw[workspaceNavigationKey], &hosts); err != nil {
		t.Fatal(err)
	}
	if string(hosts["host-a"]["futureHostField"]) != "7" || len(hosts["host-a"]["archiveExpanded"]) != 0 {
		t.Fatal("archive preference lost unknown fields or persisted local expansion")
	}
}

func TestWorkspaceArchivedFilterMergePreservesDifferentWindowFields(t *testing.T) {
	for _, archiveWindowFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "archive-first", false: "archive-last"}[archiveWindowFirst], func(t *testing.T) {
			s, dir, base := workspaceMergeStore(t)
			external, _ := Open(dir)
			archive, grouping := cloneWorkspaceMergeFixture(t, base), cloneWorkspaceMergeFixture(t, base)
			archive.ArchivedFilter = "only"
			grouping.GroupBy = "flat"
			grouping.Collapsed["b"] = true
			grouping.SessionOrders["w2"] = []string{"y", "x"}
			first, second := grouping, archive
			if archiveWindowFirst {
				first, second = archive, grouping
			}
			if _, err := external.MergeWorkspaceNavigationPreferences("host-a", base, first); err != nil {
				t.Fatal(err)
			}
			if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, second); err != nil {
				t.Fatal(err)
			}
			restarted, _ := Open(dir)
			got, err := restarted.ReadWorkspaceNavigationPreferences("host-a")
			if err != nil || got.ArchivedFilter != "only" || got.GroupBy != "flat" || !got.Collapsed["b"] || !reflect.DeepEqual(got.SessionOrders["w2"], grouping.SessionOrders["w2"]) {
				t.Fatalf("different-field edits lost: %+v %v", got, err)
			}
		})
	}
}

func TestWorkspaceArchivedFilterMergeConflictRetainsDiskUntilResolved(t *testing.T) {
	s, dir, base := workspaceMergeStore(t)
	external, _ := Open(dir)
	remote, request := cloneWorkspaceMergeFixture(t, base), cloneWorkspaceMergeFixture(t, base)
	remote.ArchivedFilter, request.ArchivedFilter = "show", "only"
	if err := external.SaveWorkspaceNavigationPreferences("host-a", remote); err != nil {
		t.Fatal(err)
	}
	before := readConfig(t, dir)
	for i := 0; i < 2; i++ {
		if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), "archivedFilter") {
			t.Fatalf("same-field conflict hidden: %v", err)
		}
		if readConfig(t, dir) != before {
			t.Fatal("conflict overwrote latest disk")
		}
	}
	if err := external.SaveWorkspaceNavigationPreferences("host-a", base); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); err != nil {
		t.Fatal(err)
	}
	restarted, _ := Open(dir)
	got, _ := restarted.ReadWorkspaceNavigationPreferences("host-a")
	if got.ArchivedFilter != "only" {
		t.Fatalf("resolved retry lost pending filter: %+v", got)
	}
}

func TestWorkspaceArchivedFilterDefaultDoesNotConflictWithLegacyMissingValue(t *testing.T) {
	s, dir, base := workspaceMergeStore(t)
	external, _ := Open(dir)
	remote, request := cloneWorkspaceMergeFixture(t, base), cloneWorkspaceMergeFixture(t, base)
	remote.ArchivedFilter = "show"
	request.ArchivedFilter, request.OrderBy = "default", "updated"
	if err := external.SaveWorkspaceNavigationPreferences("host-a", remote); err != nil {
		t.Fatal(err)
	}
	merged, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request)
	if err != nil || merged.ArchivedFilter != "show" || merged.OrderBy != "updated" {
		t.Fatalf("default materialization reverted archive field: %+v %v", merged, err)
	}
	// A malformed fresh read refuses to merge any explicit archive choice.
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(`{"broken"`), 0600); err != nil {
		t.Fatal(err)
	}
	request.ArchivedFilter = "only"
	if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); err == nil {
		t.Fatal("unreadable config was replaced")
	}
	if readConfig(t, dir) != `{"broken"` {
		t.Fatal("failed read changed config")
	}
}

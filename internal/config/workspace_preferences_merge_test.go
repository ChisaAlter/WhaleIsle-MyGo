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

const workspaceMergeFixture = `{"locale":"zh","future":{"keep":true},"goWorkspaceNavigation":{"host-a":{"groupBy":"workspace","orderBy":"manual","collapsed":{"a":false,"b":false},"sessionOrders":{"w1":["a","b","c"],"w2":["x","y"]},"futureHostField":7}}}`

func workspaceMergeStore(t *testing.T) (*Store, string, WorkspaceNavigationPreferences) {
	t.Helper()
	dir := t.TempDir()
	writeConfig(t, dir, workspaceMergeFixture)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	base, err := s.ReadWorkspaceNavigationPreferences("host-a")
	if err != nil {
		t.Fatal(err)
	}
	return s, dir, base
}

func cloneWorkspaceMergeFixture(t *testing.T, value WorkspaceNavigationPreferences) WorkspaceNavigationPreferences {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var copy WorkspaceNavigationPreferences
	if err = json.Unmarshal(data, &copy); err != nil {
		t.Fatal(err)
	}
	return copy
}

func TestWorkspacePreferenceMergePreservesConcurrentFieldsAndHosts(t *testing.T) {
	s, dir, base := workspaceMergeStore(t)
	external, _ := Open(dir)
	if err := external.Update(func(c *Config) { c.Locale = "en" }); err != nil {
		t.Fatal(err)
	}
	remote := cloneWorkspaceMergeFixture(t, base)
	remote.OrderBy = "updated"
	remote.Collapsed["b"] = true
	remote.SessionOrders["w2"] = []string{"y", "x"}
	if err := external.SaveWorkspaceNavigationPreferences("host-a", remote); err != nil {
		t.Fatal(err)
	}
	if err := external.SaveWorkspaceNavigationPreferences("host-b", WorkspaceNavigationPreferences{GroupBy: "flat"}); err != nil {
		t.Fatal(err)
	}
	request := cloneWorkspaceMergeFixture(t, base)
	request.GroupBy = "workspace-tree"
	request.Collapsed["a"] = true
	request.SessionOrders["w1"] = []string{"c", "a", "b"}
	merged, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request)
	if err != nil {
		t.Fatal(err)
	}
	if merged.GroupBy != "workspace-tree" || merged.OrderBy != "updated" || !merged.Collapsed["a"] || !merged.Collapsed["b"] || !reflect.DeepEqual(merged.SessionOrders["w1"], request.SessionOrders["w1"]) || !reflect.DeepEqual(merged.SessionOrders["w2"], remote.SessionOrders["w2"]) {
		t.Fatalf("independent view changes were not merged: %+v", merged)
	}
	if s.Get().Locale != "en" {
		t.Fatal("typed config did not adopt latest external setting")
	}
	if err = s.Update(func(c *Config) { c.SidebarWidth = 300 }); err != nil {
		t.Fatal(err)
	}
	reopened, _ := Open(dir)
	other, _ := reopened.WorkspaceNavigationPreferences("host-b")
	if other.GroupBy != "flat" || reopened.Get().Locale != "en" {
		t.Fatal("other Host or unrelated setting was overwritten")
	}
	var raw map[string]json.RawMessage
	if err = json.Unmarshal([]byte(readConfig(t, dir)), &raw); err != nil {
		t.Fatal(err)
	}
	var root map[string]map[string]json.RawMessage
	if err = json.Unmarshal(raw[workspaceNavigationKey], &root); err != nil {
		t.Fatal(err)
	}
	if string(root["host-a"]["futureHostField"]) != "7" || !strings.Contains(string(raw["future"]), "true") {
		t.Fatal("unknown configuration fields were removed")
	}
}

func TestWorkspacePreferenceMergeConflictAndExplicitRetry(t *testing.T) {
	for _, field := range []string{"groupBy", "sessionOrders.w1", "collapsed.a"} {
		t.Run(field, func(t *testing.T) {
			s, dir, base := workspaceMergeStore(t)
			request, remote := cloneWorkspaceMergeFixture(t, base), cloneWorkspaceMergeFixture(t, base)
			switch field {
			case "groupBy":
				request.GroupBy, remote.GroupBy = "flat", "workspace-tree"
			case "sessionOrders.w1":
				request.SessionOrders["w1"], remote.SessionOrders["w1"] = []string{"b", "a", "c"}, []string{"c", "b", "a"}
			case "collapsed.a":
				delete(request.Collapsed, "a")
				remote.Collapsed["a"] = true
			}
			external, _ := Open(dir)
			if err := external.SaveWorkspaceNavigationPreferences("host-a", remote); err != nil {
				t.Fatal(err)
			}
			before := readConfig(t, dir)
			for retry := 0; retry < 2; retry++ {
				_, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request)
				if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), field) {
					t.Fatalf("missing field-level conflict: %v", err)
				}
				if readConfig(t, dir) != before {
					t.Fatal("conflicting request overwrote latest disk state")
				}
			}
			if err := external.SaveWorkspaceNavigationPreferences("host-a", base); err != nil {
				t.Fatal(err)
			}
			if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); err != nil {
				t.Fatalf("retry after conflict resolved: %v", err)
			}
			reopened, _ := Open(dir)
			got, _ := reopened.WorkspaceNavigationPreferences("host-a")
			if !reflect.DeepEqual(got, request) {
				t.Fatal("retry did not persist the original pending request")
			}
		})
	}
}

func TestWorkspacePreferenceMergeAcceptsSameConcurrentValue(t *testing.T) {
	s, dir, base := workspaceMergeStore(t)
	request := cloneWorkspaceMergeFixture(t, base)
	request.OrderBy = "updated"
	external, _ := Open(dir)
	if err := external.SaveWorkspaceNavigationPreferences("host-a", request); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspacePreferenceMergeRechecksCASBeforeRenameRetry(t *testing.T) {
	s, dir, base := workspaceMergeStore(t)
	request := cloneWorkspaceMergeFixture(t, base)
	request.GroupBy = "flat"
	externalBytes := strings.Replace(workspaceMergeFixture, `"locale":"zh"`, `"locale":"en"`, 1)
	renames := 0
	s.rename = func(string, string) error {
		renames++
		writeConfig(t, dir, externalBytes)
		return errors.New("fixture sharing violation")
	}
	if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); !errors.Is(err, ErrConflict) {
		t.Fatalf("lost final CAS guard: %v", err)
	}
	if renames != 1 || readConfig(t, dir) != externalBytes {
		t.Fatal("retry overwrote a concurrent edit")
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, ".config-*.json"))
	if len(leftovers) != 0 {
		t.Fatal("failed CAS left temporary config files")
	}
	s.rename = nil
	if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); err != nil {
		t.Fatalf("fresh explicit retry failed: %v", err)
	}
	reopened, _ := Open(dir)
	got, _ := reopened.WorkspaceNavigationPreferences("host-a")
	if got.GroupBy != "flat" || reopened.Get().Locale != "en" {
		t.Fatal("fresh retry lost local view or external setting")
	}
}

func TestWorkspacePreferenceFreshReadDoesNotBypassOrdinaryCAS(t *testing.T) {
	s, dir, _ := workspaceMergeStore(t)
	external, _ := Open(dir)
	if err := external.Update(func(c *Config) { c.Locale = "en" }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadWorkspaceNavigationPreferences("host-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(func(c *Config) { c.SidebarWidth = 300 }); !errors.Is(err, ErrConflict) {
		t.Fatalf("view read relaxed ordinary config CAS: %v", err)
	}
}

func TestWorkspacePreferenceMergeRefusesRemovedOrMalformedConfig(t *testing.T) {
	for _, mode := range []string{"removed", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			s, dir, base := workspaceMergeStore(t)
			request := cloneWorkspaceMergeFixture(t, base)
			request.GroupBy = "flat"
			if mode == "removed" {
				if err := os.Remove(filepath.Join(dir, FileName)); err != nil {
					t.Fatal(err)
				}
			} else {
				writeConfig(t, dir, `{"invalid"`)
			}
			if _, err := s.MergeWorkspaceNavigationPreferences("host-a", base, request); err == nil {
				t.Fatal("unreadable latest configuration was replaced")
			}
			if mode == "removed" {
				if _, err := os.Stat(filepath.Join(dir, FileName)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("removed file was recreated")
				}
			} else if readConfig(t, dir) != `{"invalid"` {
				t.Fatal("malformed file changed")
			}
		})
	}
}

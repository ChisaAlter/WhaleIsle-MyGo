package importer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func pluginFixture(t *testing.T, home, body string) {
	t.Helper()
	path := filepath.Join(home, "profiles", "web", "package.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte(body), 0600); err != nil { t.Fatal(err) }
}

func TestPluginImportSelectionAndSpecs(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	source, dest := filepath.Join(root, "source"), filepath.Join(root, "dest")
	pluginFixture(t, source, `{"dependencies":{"fixture-git":"github:fixture/plugin#v1","@fixture/npm":"^1.2.3","@deepseek-ai/dsh-base":"1.0.0","dshmarket":"1.0.0","fixture-local":"file:../local","fixture-tar":"https://example.invalid/plugin.tgz","fixture-tag":"latest"}}`)
	pluginFixture(t, dest, `{"dependencies":{"unrelated":"1.0.0"}}`)
	cat, err := BuildCatalog(source, dest)
	if err != nil { t.Fatal(err) }
	rows := map[string]CatalogItem{}
	for _, row := range cat.Plugins { rows[row.ID] = row }
	if row := rows["fixture-git"]; row.Unsupported || row.Conflict || row.Spec != "github:fixture/plugin#v1" || !row.Selected { t.Fatalf("GitHub row: %+v", row) }
	if row := rows["@fixture/npm"]; row.Unsupported || row.Spec != "^1.2.3" || !row.Selected { t.Fatalf("npm row: %+v", row) }
	for _, name := range []string{"@deepseek-ai/dsh-base", "dshmarket", "fixture-local", "fixture-tar", "fixture-tag"} {
		if !rows[name].Unsupported || rows[name].Selected { t.Fatalf("unsafe row selected: %+v", rows[name]) }
	}
	sel := SelectionFromCatalog(cat, false, false)
	if len(sel.SelectedPluginIDs) != 2 || sel.Count() != 2 { t.Fatalf("selection: %+v", sel) }
	for _, part := range []string{"fixture-git", "github:fixture/plugin#v1", "@fixture/npm@^1.2.3", "构建脚本"} {
		if !strings.Contains(Confirmation(cat, sel), part) { t.Fatalf("missing disclosure %q", part) }
	}
}

func TestPluginImportResultsAndFreshConflicts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	source, dest := filepath.Join(root, "source"), filepath.Join(root, "dest")
	pluginFixture(t, source, `{"dependencies":{"fixture-a":"1.2.3","fixture-b":"~2.0.0","fixture-c":"github:fixture/c"}}`)
	cat, err := BuildCatalog(source, dest)
	if err != nil { t.Fatal(err) }
	// A destination install after scan must still be preserved by default.
	pluginFixture(t, dest, `{"dependencies":{"fixture-a":"9.0.0"}}`)
	var calls []string
	run := func(ctx context.Context, row CatalogItem, spec string) PluginInstallResult {
		calls = append(calls, spec)
		if row.ID == "fixture-b" { return PluginInstallResult{Error: "fixture failure", Log: "fixture log"} }
		return PluginInstallResult{OK: true}
	}
	sel := ApplySelection{SelectedPluginIDs: []string{"fixture-a", "fixture-b", "fixture-c", "fixture-c"}}
	result, err := ApplyCatalogWithPlugins(context.Background(), cat, sel, run, nil)
	if err != nil || result.OK || result.Installed != 1 || result.Failed != 1 || result.Skipped != 1 || len(result.Plugins) != 3 { t.Fatalf("result: %+v %v", result, err) }
	if !reflect.DeepEqual(calls, []string{"fixture-b@~2.0.0", "github:fixture/c"}) { t.Fatalf("calls: %v", calls) }
	if !strings.Contains(strings.Join(result.Journal, "\n"), "fixture log") { t.Fatal("install log missing") }
	sel = ApplySelection{SelectedPluginIDs: []string{"fixture-a"}, Overwrite: true}
	result, err = ApplyCatalogWithPlugins(context.Background(), cat, sel, run, nil)
	if err != nil || !result.OK || result.Installed != 1 || calls[len(calls)-1] != "fixture-a@1.2.3" { t.Fatalf("overwrite: %+v %v", result, err) }
}

func TestPluginImportCancellationAndMissingInstaller(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	source, dest := filepath.Join(root, "source"), filepath.Join(root, "dest")
	pluginFixture(t, source, `{"dependencies":{"fixture-a":"1.0.0","fixture-b":"1.0.0"}}`)
	cat, err := BuildCatalog(source, dest)
	if err != nil { t.Fatal(err) }
	sel := ApplySelection{SelectedPluginIDs: []string{"fixture-a", "fixture-b"}}
	ctx, cancel := context.WithCancel(context.Background())
	count := 0
	result, err := ApplyCatalogWithPlugins(ctx, cat, sel, func(ctx context.Context, row CatalogItem, spec string) PluginInstallResult {
		count++
		cancel()
		return PluginInstallResult{OK: true}
	}, nil)
	if !errors.Is(err, context.Canceled) || count != 1 || result.Installed != 1 || len(result.Plugins) != 1 { t.Fatalf("cancel: %+v %v calls=%d", result, err, count) }
	result, err = ApplyCatalogProgress(context.Background(), cat, sel, nil)
	if err != nil || result.OK || result.Empty || result.Failed != 2 { t.Fatalf("missing installer: %+v %v", result, err) }
}

func TestPluginReinstallSpecRejectsUntrustedChannels(t *testing.T) {
	for _, spec := range []string{"latest", "*", ">=1.0.0", "npm:other@1.0.0", "https://example.invalid/p.tgz", "git+https://github.com/a/b.git", "file:../x", "workspace:*", "1.0.0 --ignore-scripts"} {
		if got := PluginReinstallSpec("fixture", spec); got != "" { t.Fatalf("accepted %q: %q", spec, got) }
	}
	for _, spec := range []string{"1.2.3", "^1.2.3", "~1.2.3-beta.1+build"} {
		if got := PluginReinstallSpec("@fixture/plugin", spec); got != "@fixture/plugin@"+spec { t.Fatalf("rejected %q: %q", spec, got) }
	}
}

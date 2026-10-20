package importer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestSelectedSectionMergeFixtures(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	src := filepath.Join(home, "source")
	dst := filepath.Join(home, "target")
	for _, dir := range []string{src, dst} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	sourceSettings := "ui-theme:\n  mode: dark # keep source comment\nagent-default-model:\n  model: new\nllm-deepseek:\n  apiKeyEnv: FIXTURE_ONLY\nunlisted:\n  should: never copy\n"
	destSettings := "# target preamble\nui-theme:\n  mode: light\nprivate-plugin:\n  preserve: exact\n"
	sourceMCP := "servers:\n  - id: alpha\n    url: new\n    headers:\n      X-Fixture: pretend\n  - id: beta\n    command: echo\n    args:\n      - fixture\n  - id: unselected\n    url: never-copy\n"
	destMCP := "# target MCP\nservers:\n  - id: alpha\n    url: original\n  - id: target-only\n    url: retained\nother-meta:\n  keep: true\n"
	for path, data := range map[string]string{filepath.Join(src, "settings.yaml"): sourceSettings, filepath.Join(dst, "settings.yaml"): destSettings, filepath.Join(src, "mcp-servers.yaml"): sourceMCP, filepath.Join(dst, "mcp-servers.yaml"): destMCP} {
		writeFile(t, path, data)
	}
	cat, err := BuildCatalog(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.MCP) != 3 || cat.MCP[0].ID != "alpha" || !cat.MCP[0].Conflict || cat.MCP[1].Conflict {
		t.Fatalf("MCP preview: %+v", cat.MCP)
	}
	sel := ApplySelection{SelectedSettingIDs: []string{"ui-theme", "agent-default-model", "unlisted"}, SelectedMcpIDs: []string{"alpha", "beta"}}
	preview := Confirmation(cat, sel)
	if !strings.Contains(preview, "MCP / alpha：Skip and keep") || !strings.Contains(preview, "MCP / beta：Add") {
		t.Fatal(preview)
	}
	result, err := ApplyCatalog(cat, sel)
	if err != nil || result.Copied != 2 || result.Skipped != 3 {
		t.Fatalf("merge: %+v %v", result, err)
	}
	read := func(name string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(dst, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	settings := read("settings.yaml")
	mcp := read("mcp-servers.yaml")
	if !strings.HasPrefix(settings, destSettings) || !strings.Contains(settings, "model: new") || strings.Contains(settings, "FIXTURE_ONLY") || strings.Contains(settings, "unlisted:") {
		t.Fatal("settings scope/preservation")
	}
	if !strings.Contains(mcp, "url: original") || !strings.Contains(mcp, "url: retained") || !strings.Contains(mcp, "- fixture") || !strings.Contains(mcp, "other-meta:") || strings.Contains(mcp, "never-copy") {
		t.Fatal("MCP scope/preservation")
	}
	// A stale scan must not suppress live conflicts or erase newly-added target data.
	sel.Overwrite = true
	sel.SelectedSettingIDs = []string{"ui-theme"}
	sel.SelectedMcpIDs = []string{"alpha"}
	result, err = ApplyCatalog(cat, sel)
	if err != nil || result.Copied != 2 {
		t.Fatalf("overwrite: %+v %v", result, err)
	}
	settings = read("settings.yaml")
	mcp = read("mcp-servers.yaml")
	if !strings.Contains(settings, "# keep source comment") || !strings.Contains(settings, "private-plugin:") || !strings.Contains(settings, "model: new") {
		t.Fatal("overwrite dropped settings")
	}
	if !strings.Contains(mcp, "X-Fixture: pretend") || !strings.Contains(mcp, "id: beta") || !strings.Contains(mcp, "url: retained") {
		t.Fatal("overwrite dropped MCP")
	}
	// Failed publication never truncates original bytes, and owns only its temp file.
	path := filepath.Join(dst, "settings.yaml")
	err = writeMergedFileWithRename(path, []byte(settings), []byte("replacement"), func(string, string) error { return errors.New("fixture publish failure") })
	if err == nil || read("settings.yaml") != settings {
		t.Fatal("failed publication lost original")
	}
	temps, _ := filepath.Glob(filepath.Join(dst, ".import-section-*"))
	if len(temps) != 0 {
		t.Fatal("temporary file leaked")
	}
	// Unknown/malformed target is an error in both preview and apply, never empty.
	writeFile(t, filepath.Join(dst, "mcp-servers.yaml"), "servers:\n  - id: duplicate\n  - id: duplicate\n")
	if _, err := BuildCatalog(src, dst); err == nil {
		t.Fatal("preview accepted duplicate target")
	}
	result, err = ApplyCatalog(cat, sel)
	if err == nil || result.Copied != 0 || read("settings.yaml") != settings {
		t.Fatal("invalid target was overwritten")
	}
	for path, want := range map[string]string{filepath.Join(src, "settings.yaml"): sourceSettings, filepath.Join(src, "mcp-servers.yaml"): sourceMCP} {
		raw, _ := os.ReadFile(path)
		if string(raw) != want {
			t.Fatal("source modified")
		}
	}
	if _, err := os.Stat(filepath.Join(dst, ".credentials.yaml")); !os.IsNotExist(err) {
		t.Fatal("credentials unexpectedly created")
	}
}

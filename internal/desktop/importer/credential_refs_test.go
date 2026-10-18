package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// All values are synthetic. This fixture never reads the user's home or starts Host.
func TestImportCredentialRefsFixture(t *testing.T) {
	makeHomes := func(t *testing.T) (string, string) {
		t.Helper()
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		src, dst := filepath.Join(home, "source"), filepath.Join(home, "destination")
		for _, dir := range []string{src, dst} {
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
		}
		return src, dst
	}
	put := func(t *testing.T, root, name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	read := func(t *testing.T, root, name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	run := func(t *testing.T, src, dst string, ids []string, overwrite bool) ApplyResult {
		t.Helper()
		result, err := ApplyCatalog(Catalog{SourceHome: src, DestHome: dst}, ApplySelection{SelectedSettingIDs: ids, Overwrite: overwrite})
		if err != nil {
			t.Fatalf("unexpected import error: %v", err)
		}
		return result
	}
	status := func(t *testing.T, result ApplyResult, wanted map[string]string, secrets ...string) {
		t.Helper()
		raw, err := json.Marshal(result)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range secrets {
			if strings.Contains(string(raw), value) {
				t.Fatal("credential material escaped into result")
			}
		}
		var got struct {
			Credentials []struct{ Ref, Status, Reason string }
		}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if len(got.Credentials) != len(wanted) {
			t.Fatalf("credential status count = %d, want %d", len(got.Credentials), len(wanted))
		}
		for _, row := range got.Credentials {
			if wanted[row.Ref] != row.Status {
				t.Fatalf("status for %s = %s, want %s", row.Ref, row.Status, wanted[row.Ref])
			}
		}
	}
	t.Run("selected_refs_only_and_verbatim_mapping", func(t *testing.T) {
		src, dst := makeHomes(t)
		source := "version: 1\nrefs:\n  SELECTED_KEY: 'fixture: secret # literal' # preserved source comment\n  UNSELECTED_KEY: fixture-never-copy\nrecords:\n  oauth: fixture-source-oauth\n"
		target := "# target preamble\nversion: 1\nrefs:\n  TARGET_ONLY: fixture-target-only\nrecords:\n  oauth: fixture-target-oauth\n"
		put(t, src, "settings.yaml", "llm-deepseek:\n  apiKeyEnv: 'SELECTED_KEY'\nllm-pi-ai:\n  apiKeyEnv: UNSELECTED_KEY\nui-theme:\n  apiKeyEnv: UNSELECTED_KEY\n")
		put(t, src, ".credentials.yaml", source)
		put(t, dst, ".credentials.yaml", target)
		result := run(t, src, dst, []string{"llm-deepseek", "llm-deepseek", "ui-theme"}, false)
		got := read(t, dst, ".credentials.yaml")
		if !strings.Contains(got, "  SELECTED_KEY: 'fixture: secret # literal' # preserved source comment\n") {
			t.Fatal("selected credential was not copied verbatim")
		}
		if !strings.Contains(got, "records:\n  oauth: fixture-target-oauth\n") || !strings.Contains(got, "  TARGET_ONLY: fixture-target-only\n") || strings.Contains(got, "fixture-never-copy") || strings.Contains(got, "fixture-source-oauth") {
			t.Fatal("credential scope or target preservation failed")
		}
		if read(t, src, ".credentials.yaml") != source {
			t.Fatal("source changed")
		}
		status(t, result, map[string]string{"SELECTED_KEY": "copied"}, "fixture: secret", "fixture-target-only", "fixture-target-oauth")
	})
	t.Run("conflict_skip_then_explicit_overwrite", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-pi-ai:\n  providers:\n    sample:\n      apiKeyEnv: SHARED_KEY\n")
		put(t, src, ".credentials.yaml", "version: 1\nrefs:\n  SHARED_KEY: fixture-new\n")
		target := "version: 1\nrefs:\n  SHARED_KEY: fixture-old\nrecords:\n  keep: fixture-record\n"
		put(t, dst, ".credentials.yaml", target)
		result := run(t, src, dst, []string{"llm-pi-ai"}, false)
		if read(t, dst, ".credentials.yaml") != target {
			t.Fatal("default conflict silently overwritten")
		}
		status(t, result, map[string]string{"SHARED_KEY": "skipped"}, "fixture-new", "fixture-old")
		result = run(t, src, dst, []string{"llm-pi-ai"}, true)
		got := read(t, dst, ".credentials.yaml")
		if !strings.Contains(got, "  SHARED_KEY: fixture-new\n") || !strings.Contains(got, "  keep: fixture-record\n") || strings.Contains(got, "fixture-old") {
			t.Fatal("explicit overwrite failed")
		}
		status(t, result, map[string]string{"SHARED_KEY": "copied"}, "fixture-new", "fixture-old")
	})
	t.Run("default_ref_and_empty_target_refs", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-deepseek:\n  model: fixture-model\n")
		put(t, src, ".credentials.yaml", "version: 1\nrefs:\n  DEEPSEEK_API_KEY: \"fixture-default\"\n")
		put(t, dst, ".credentials.yaml", "version: 1\nrefs: {}\nrecords: {}\n")
		result := run(t, src, dst, []string{"llm-deepseek"}, false)
		if !strings.Contains(read(t, dst, ".credentials.yaml"), "  DEEPSEEK_API_KEY: \"fixture-default\"\n") {
			t.Fatal("default reference missing")
		}
		status(t, result, map[string]string{"DEEPSEEK_API_KEY": "copied"}, "fixture-default")
	})
	t.Run("missing_and_unsupported_report_without_values", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-pi-ai:\n  providers:\n    missing:\n      apiKeyEnv: MISSING_KEY\n    block:\n      apiKeyEnv: BLOCK_KEY\n    encrypted:\n      apiKeyEnv: ENCRYPTED_KEY\n")
		put(t, src, ".credentials.yaml", "version: 1\nrefs:\n  BLOCK_KEY: |\n    fixture-block-secret\n  ENCRYPTED_KEY: {ciphertext: fixture-encrypted-secret}\n")
		result := run(t, src, dst, []string{"llm-pi-ai"}, false)
		status(t, result, map[string]string{"MISSING_KEY": "missing", "BLOCK_KEY": "unsupported", "ENCRYPTED_KEY": "unsupported"}, "fixture-block-secret", "fixture-encrypted-secret")
		if result.OK || result.Failed != 3 {
			t.Fatal("unmigrated references reported as success")
		}
		if _, err := os.Stat(filepath.Join(dst, ".credentials.yaml")); !os.IsNotExist(err) {
			t.Fatal("unsupported secrets published")
		}
	})
	t.Run("skipped_settings_only_fill_still_referenced_names", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-deepseek:\n  apiKeyEnv: SOURCE_KEY\n")
		put(t, dst, "settings.yaml", "llm-deepseek:\n  apiKeyEnv: TARGET_KEY\n")
		put(t, src, ".credentials.yaml", "version: 1\nrefs:\n  SOURCE_KEY: fixture-inactive\n")
		result := run(t, src, dst, []string{"llm-deepseek"}, false)
		status(t, result, map[string]string{}, "fixture-inactive")
		if _, err := os.Stat(filepath.Join(dst, ".credentials.yaml")); !os.IsNotExist(err) {
			t.Fatal("inactive source reference copied")
		}
	})
	t.Run("unsupported_version_preserves_target", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-deepseek:\n  apiKeyEnv: FIXTURE_KEY\n")
		put(t, src, ".credentials.yaml", "version: encrypted-v2\nrefs:\n  FIXTURE_KEY: fixture-cipher\n")
		target := "version: 1\nrefs:\n  FIXTURE_KEY: fixture-target\n"
		put(t, dst, ".credentials.yaml", target)
		result := run(t, src, dst, []string{"llm-deepseek"}, true)
		status(t, result, map[string]string{"FIXTURE_KEY": "unsupported"}, "fixture-cipher", "fixture-target")
		if result.OK || read(t, dst, ".credentials.yaml") != target {
			t.Fatal("unsupported envelope changed target")
		}
	})
	t.Run("unsafe_source_and_unselected_secret_file", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-deepseek:\n  apiKeyEnv: FIXTURE_KEY\nui-theme:\n  mode: dark\n")
		external := filepath.Join(t.TempDir(), "synthetic-secret")
		if err := os.WriteFile(external, []byte("version: 1\nrefs:\n  FIXTURE_KEY: fixture-link-secret\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(external, filepath.Join(src, ".credentials.yaml")); err != nil {
			t.Skip("symlinks unavailable")
		}
		result := run(t, src, dst, []string{"ui-theme"}, false)
		status(t, result, map[string]string{}, "fixture-link-secret")
		result = run(t, src, dst, []string{"llm-deepseek"}, false)
		status(t, result, map[string]string{"FIXTURE_KEY": "failed"}, "fixture-link-secret")
		if result.OK {
			t.Fatal("unsafe source accepted")
		}
	})
	t.Run("hard_link_source_is_not_an_authorized_root_file", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-deepseek:\n  apiKeyEnv: FIXTURE_KEY\n")
		external := filepath.Join(t.TempDir(), "synthetic-hard-link-secret")
		if err := os.WriteFile(external, []byte("version: 1\nrefs:\n  FIXTURE_KEY: fixture-hard-link-secret\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(external, filepath.Join(src, ".credentials.yaml")); err != nil {
			t.Skip("hard links unavailable")
		}
		result := run(t, src, dst, []string{"llm-deepseek"}, false)
		status(t, result, map[string]string{"FIXTURE_KEY": "failed"}, "fixture-hard-link-secret")
		if result.OK {
			t.Fatal("hard-linked source credential accepted")
		}
	})

	t.Run("invalid_target_never_becomes_empty_store", func(t *testing.T) {
		src, dst := makeHomes(t)
		put(t, src, "settings.yaml", "llm-deepseek:\n  apiKeyEnv: FIXTURE_KEY\n")
		put(t, src, ".credentials.yaml", "version: 1\nrefs:\n  FIXTURE_KEY: fixture-new\n")
		target := "version: 1\nrefs:\n  FIXTURE_KEY: fixture-first\n  FIXTURE_KEY: fixture-second\nrecords:\n  keep: fixture-record\n"
		put(t, dst, ".credentials.yaml", target)
		result := run(t, src, dst, []string{"llm-deepseek"}, true)
		status(t, result, map[string]string{"FIXTURE_KEY": "failed"}, "fixture-new", "fixture-first", "fixture-second")
		if result.OK || read(t, dst, ".credentials.yaml") != target {
			t.Fatal("malformed target overwritten")
		}
	})

}

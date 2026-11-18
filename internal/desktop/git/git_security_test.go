package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitChildEnvStripsRepoOverrideVars(t *testing.T) {
	stripped := []string{
		"GIT_DIR",
		"GIT_WORK_TREE",
		"GIT_INDEX_FILE",
		"GIT_COMMON_DIR",
		"GIT_OBJECT_DIRECTORY",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES",
		"GIT_CEILING_DIRECTORIES",
		"NPM_CONFIG_ELECTRON_MIRROR",
	}
	for _, k := range stripped {
		t.Setenv(k, "/poisoned/"+k)
	}
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	t.Setenv("LC_ALL", "en_US.UTF-8")

	env := gitChildEnv()
	seen := map[string]string{}
	for _, entry := range env {
		k, v, ok := strings.Cut(entry, "=")
		if ok {
			seen[strings.ToUpper(k)] = v
		}
	}

	for _, k := range stripped {
		if v, exists := seen[k]; exists {
			t.Fatalf("gitChildEnv leaked %s=%q", k, v)
		}
	}
	if seen["GIT_TERMINAL_PROMPT"] != "0" || seen["GCM_INTERACTIVE"] != "never" || seen["LC_ALL"] != "C" {
		t.Fatalf("unexpected forced env values: terminal=%q gcm=%q lc=%q", seen["GIT_TERMINAL_PROMPT"], seen["GCM_INTERACTIVE"], seen["LC_ALL"])
	}
}

func TestUntrackedHelpersRejectNonRegularAndSymlink(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if added, deleted := countUntrackedLines(root, "subdir"); added != 0 || deleted != 0 {
		t.Fatalf("countUntrackedLines on dir = (%d, %d), want (0, 0)", added, deleted)
	}
	if _, err := untrackedDiff(root, "subdir"); err == nil {
		t.Fatal("untrackedDiff on directory succeeded, want error")
	}

	t.Run("symlink", func(t *testing.T) {
		target := filepath.Join(root, "target.txt")
		if err := os.WriteFile(target, []byte("line1\nline2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(root, "link.txt")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("os.Symlink unavailable in this environment: %v", err)
		}
		added, deleted := countUntrackedLines(root, "link.txt")
		if added != 0 || deleted != 0 {
			t.Fatalf("countUntrackedLines on symlink = (%d, %d), want (0, 0)", added, deleted)
		}
		if _, err := untrackedDiff(root, "link.txt"); err == nil {
			t.Fatal("untrackedDiff on symlink succeeded, want error")
		}
	})
}

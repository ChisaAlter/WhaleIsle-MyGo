package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestStagingCommitWorkflow(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	run := func(args ...string) string {
		c := exec.Command("git", args...)
		c.Dir = root
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
		return string(out)
	}
	run("init")
	run("config", "user.name", "Temporary Test")
	run("config", "user.email", "temporary@example.invalid")
	run("config", "commit.gpgsign", "false")
	write := func(p, s string) {
		if err := os.WriteFile(filepath.Join(root, p), []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("a.txt", "one\n")
	write("b.txt", "untouched\n")
	changes, err := Changes(root)
	if err != nil || len(changes) != 2 {
		t.Fatalf("changes %v %v", changes, err)
	}
	if diff, err := DiffScope(root, "a.txt", false); err != nil || !strings.Contains(diff, "+one") {
		t.Fatalf("untracked diff %q %v", diff, err)
	}
	if _, err := StagePaths(root, "a.txt"); err != nil {
		t.Fatal(err)
	}
	_, token, err := IndexReview(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CommitReviewed(root, "initial", token); err != nil {
		t.Fatal(err)
	}
	if got := run("show", "--format=", "--name-only", "HEAD"); strings.Contains(got, "b.txt") || !strings.Contains(got, "a.txt") {
		t.Fatalf("wrong commit scope: %s", got)
	}
	write("a.txt", "two\n")
	if _, err := StagePaths(root, "a.txt"); err != nil {
		t.Fatal(err)
	}
	_, token, err = IndexReview(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = StagePaths(root, "b.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err = CommitReviewed(root, "must fail", token); err == nil {
		t.Fatal("changed staged scope accepted")
	}
	if _, err = UnstagePaths(root, "b.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err = CommitStaged(root, "second"); err != nil {
		t.Fatal(err)
	}
	if _, err = CommitStaged(root, "empty must fail"); err == nil {
		t.Fatal("empty commit succeeded")
	}
	if _, err = StagePaths(root, "../outside"); err == nil {
		t.Fatal("outside path accepted")
	}
}

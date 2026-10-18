package git

import (
	"strings"
	"testing"
)

func TestWholeUnstagedIncludesUntracked(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("tracked.txt", "base\n")
	run("add", "tracked.txt")
	run("commit", "-m", "base")
	write("tracked.txt", "edit\n")
	write("new.txt", "new content\n")
	before := run("status", "--porcelain")
	diff, err := DiffScope(root, "", false)
	if err != nil || !strings.Contains(diff, "+edit") || !strings.Contains(diff, "+new content") {
		t.Fatalf("missing unstaged contents: %s %v", diff, err)
	}
	if run("status", "--porcelain") != before {
		t.Fatal("read changed index")
	}
	staged, err := DiffScope(root, "", true)
	if err != nil || staged != "" {
		t.Fatalf("unstaged leaked to index: %s %v", staged, err)
	}
}

func TestHEADDiffCombinesIndexAndWorktree(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("tracked.txt", "base\n")
	run("add", "tracked.txt")
	run("commit", "-m", "base")
	write("tracked.txt", "staged intermediate\n")
	run("add", "tracked.txt")
	write("tracked.txt", "final working value\n")
	write("new.txt", "new content\n")
	before := run("diff", "--cached")
	diff, err := DiffHEAD(root, "")
	if err != nil || !strings.Contains(diff, "-base") || !strings.Contains(diff, "+final working value") || !strings.Contains(diff, "+new content") || strings.Contains(diff, "staged intermediate") {
		t.Fatalf("incorrect HEAD aggregate: %s %v", diff, err)
	}
	if run("diff", "--cached") != before {
		t.Fatal("HEAD preview mutated index")
	}
	single, err := DiffHEAD(root, "tracked.txt")
	if err != nil || strings.Contains(single, "new.txt") {
		t.Fatalf("wrong single path: %s %v", single, err)
	}
}

package git

import (
	"os"
	"path/filepath"
	"testing"
)

func changeFor(t *testing.T, changes []Change, path string) Change {
	t.Helper()
	for _, change := range changes {
		if change.Path == path {
			return change
		}
	}
	t.Fatalf("change for %q not reported: %+v", path, changes)
	return Change{}
}

// TestChangesCarryDiffstat covers the counts a commit list needs: an edited file
// reports both sides, a new file only additions and a removed file only
// deletions.
func TestChangesCarryDiffstat(t *testing.T) {
	root, _, write := selectionRepo(t)
	write("a.txt", "one\ntwo\n")
	write("b.txt", "base\nextra\n")
	if err := os.Remove(filepath.Join(root, "b.txt")); err != nil {
		t.Fatal(err)
	}
	write("new.txt", "brand\nnew\n")
	changes, err := Changes(root)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	edited := changeFor(t, changes, "a.txt")
	if edited.Added != 2 || edited.Deleted != 1 {
		t.Fatalf("edited file diffstat = +%d/-%d, want +2/-1", edited.Added, edited.Deleted)
	}
	removed := changeFor(t, changes, "b.txt")
	if removed.Added != 0 || removed.Deleted != 1 {
		t.Fatalf("removed file diffstat = +%d/-%d, want +0/-1", removed.Added, removed.Deleted)
	}
	added := changeFor(t, changes, "new.txt")
	if added.Added != 2 || added.Deleted != 0 {
		t.Fatalf("new file diffstat = +%d/-%d, want +2/-0", added.Added, added.Deleted)
	}
}

// TestChangesCountStagedAndWorktreeTogether keeps the list honest for a file that
// was staged and then edited again: the row shows the whole change against HEAD.
func TestChangesCountStagedAndWorktreeTogether(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "staged one\nstaged two\n")
	run("add", "a.txt")
	write("a.txt", "staged one\nstaged two\nunstaged three\n")
	changes, err := Changes(root)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	file := changeFor(t, changes, "a.txt")
	if file.Added != 3 || file.Deleted != 1 {
		t.Fatalf("staged+worktree diffstat = +%d/-%d, want +3/-1", file.Added, file.Deleted)
	}
}

// TestChangesReportBinaryAsZero keeps a binary row honest instead of inventing
// line counts git refuses to give.
func TestChangesReportBinaryAsZero(t *testing.T) {
	root, _, _ := selectionRepo(t)
	binary := append([]byte("base\n"), 0, 1, 2, 3)
	if err := os.WriteFile(filepath.Join(root, "image.bin"), binary, 0o600); err != nil {
		t.Fatal(err)
	}
	changes, err := Changes(root)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	file := changeFor(t, changes, "image.bin")
	if file.Added != 0 || file.Deleted != 0 {
		t.Fatalf("binary diffstat = +%d/-%d, want +0/-0", file.Added, file.Deleted)
	}
}

// TestNumstatParsesRecords pins the parser itself, including the NUL-separated
// records and a workspace prefix.
func TestNumstatParsesRecords(t *testing.T) {
	counts := map[string][2]int{}
	readNumstat(counts, []byte("3\t1\tsub/a.go\x000\t0\tsub/b.bin\x00"), "sub/")
	if got := counts["a.go"]; got != [2]int{3, 1} {
		t.Fatalf("a.go = %v", got)
	}
	if got := counts["b.bin"]; got != [2]int{0, 0} {
		t.Fatalf("b.bin = %v", got)
	}
}

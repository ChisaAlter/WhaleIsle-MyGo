package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscardTrackedRestoresIndexAndOnlySelectedPath(t *testing.T) {
	root, run, write := diffFixture(t)
	write("a.txt", "staged\n")
	run("add", "a.txt")
	write("a.txt", "discard me\n")
	write("b.txt", "keep me\n")
	plan, err := PrepareDiscard(root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Untracked {
		t.Fatal("tracked classified untracked")
	}
	if _, err = DiscardReviewed(root, plan); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(root, "a.txt"))
	b, _ := os.ReadFile(filepath.Join(root, "b.txt"))
	if string(a) != "staged\n" || string(b) != "keep me\n" || !strings.Contains(run("diff", "--cached"), "+staged") {
		t.Fatal("discard scope/index mismatch")
	}
}
func TestDiscardUntrackedLiteralPathAndStaleReview(t *testing.T) {
	root, _, write := diffFixture(t)
	write("[x].txt", "remove\n")
	write("x.txt", "keep\n")
	plan, err := PrepareDiscard(root, "[x].txt")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Untracked {
		t.Fatal("untracked not marked")
	}
	write("[x].txt", "newer edit\n")
	if _, err = DiscardReviewed(root, plan); err == nil {
		t.Fatal("discarded edit made after review")
	}
	plan, err = PrepareDiscard(root, "[x].txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = DiscardReviewed(root, plan); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(root, "[x].txt")); !os.IsNotExist(err) {
		t.Fatal("selected file remains")
	}
	if _, err = os.Stat(filepath.Join(root, "x.txt")); err != nil {
		t.Fatal("sibling changed")
	}
}
func TestDiscardRejectsRepositoryStateAndUnsafePaths(t *testing.T) {
	root, run, write := diffFixture(t)
	write("a.txt", "dirty\n")
	plan, err := PrepareDiscard(root, "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	other, _, _ := diffFixture(t)
	if _, err = DiscardReviewed(other, plan); err == nil {
		t.Fatal("other repo accepted")
	}
	run("add", "a.txt")
	if _, err = DiscardReviewed(root, plan); err == nil {
		t.Fatal("changed index accepted")
	}
	for _, p := range []string{"", ".", "../escape", ".git/config", ":(glob)*", "a.txt/../b.txt"} {
		if _, err = PrepareDiscard(root, p); err == nil {
			t.Errorf("accepted %q", p)
		}
	}
	outside := t.TempDir()
	if err = os.WriteFile(filepath.Join(outside, "keep"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(outside, filepath.Join(root, "linked")); err == nil {
		if _, err = PrepareDiscard(root, "linked/keep"); err == nil {
			t.Fatal("symlink parent accepted")
		}
	}
}

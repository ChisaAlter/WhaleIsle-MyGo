package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteOwnershipScopesAndCancellation(t *testing.T) {
	root, run, _ := selectionRepo(t)
	other := filepath.Join(t.TempDir(), "linked")
	run("worktree", "add", "--detach", other)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, release, err := AcquireWrite(ctx, root, WorktreeWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	_, releaseOther, err := AcquireWrite(ctx, other, WorktreeWrite)
	if err != nil {
		t.Fatal("independent worktree blocked:", err)
	}
	releaseOther()
	for _, scope := range []WriteScope{WorktreeWrite, ReferenceWrite} {
		waiting, stop := context.WithTimeout(ctx, 80*time.Millisecond)
		_, unlock, e := AcquireWrite(waiting, root, scope)
		stop()
		if unlock != nil {
			unlock()
		}
		if !errors.Is(e, context.DeadlineExceeded) {
			t.Fatalf("conflicting scope %d was admitted: %v", scope, e)
		}
	}
	release()
	owned, unlock, err := AcquireWrite(ctx, other, ReferenceWrite)
	if err != nil {
		t.Fatal("cancelled waiter retained ownership:", err)
	}
	defer unlock()
	independent, _, _ := selectionRepo(t)
	_, independentRelease, err := AcquireWrite(ctx, independent, ReferenceWrite)
	if err != nil {
		t.Fatal("unrelated repository was blocked:", err)
	}
	independentRelease()
	_, nested, err := AcquireWrite(owned, root, WorktreeWrite)
	if err != nil {
		t.Fatal("workflow could not use its own common-dir ownership:", err)
	}
	nested()
	waiting, stop := context.WithTimeout(ctx, 80*time.Millisecond)
	defer stop()
	_, conflictRelease, err := AcquireWrite(waiting, root, WorktreeWrite)
	if conflictRelease != nil {
		conflictRelease()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("common refs write did not exclude other worktree: %v", err)
	}
}

func TestStagePathsWaitsForRepositoryWriteAndPreservesBothResults(t *testing.T) {
	root, run, write := selectionRepo(t)
	write("a.txt", "first change\n")
	write("b.txt", "second change\n")
	sub := filepath.Join(root, "nested")
	if err := os.Mkdir(sub, 0700); err != nil {
		t.Fatal(err)
	}
	_, release, err := AcquireWrite(context.Background(), sub, WorktreeWrite)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	results := make(chan error, 2)
	for _, name := range []string{"a.txt", "b.txt"} {
		go func() { _, err := StagePaths(root, name); results <- err }()
	}
	select {
	case err := <-results:
		t.Fatalf("stage bypassed existing worktree owner: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if staged := run("diff", "--cached", "--name-only"); staged != "" {
		t.Fatalf("index changed before admission: %s", staged)
	}
	release()
	for range 2 {
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("stage did not finish after release")
		}
	}
	if staged := run("diff", "--cached", "--name-only"); !strings.Contains(staged, "a.txt") || !strings.Contains(staged, "b.txt") {
		t.Fatalf("lost successful stage result: %s", staged)
	}
}

package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// WriteScope names the Git resource an operation changes. Worktrees have
// independent indexes; refs and repository configuration share the common dir.
type WriteScope uint8

const (
	WorktreeWrite WriteScope = iota
	ReferenceWrite
)

type writeIdentity struct{ common, worktree string }
type writeWaiter struct {
	id      writeIdentity
	scope   WriteScope
	ready   chan struct{}
	granted bool
}
type repositoryWrites struct {
	reference bool
	worktrees map[string]bool
	waiting   []*writeWaiter
}
type writeOwners struct {
	mu           sync.Mutex
	repositories map[string]*repositoryWrites
}
type writeOwnerKey struct{}
type writeOwner struct {
	id    writeIdentity
	scope WriteScope
}

var repositoryWriteOwners = writeOwners{repositories: make(map[string]*repositoryWrites)}

func writePath(path string) string {
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

func resolveWriteIdentity(ctx context.Context, root string) (writeIdentity, error) {
	if strings.TrimSpace(root) == "" {
		return writeIdentity{}, ErrNotRepo
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return writeIdentity{}, err
	}
	cmd := gitCommand(ctx, "git", "rev-parse", "--path-format=absolute", "--absolute-git-dir", "--git-common-dir")
	cmd.Dir, cmd.Env = abs, gitChildEnv()
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return writeIdentity{}, ctx.Err()
		}
		// Initialize is also a write. Only an actual non-repository gets a
		// directory identity; a failed probe must not evade a shared repo owner.
		var exit *exec.ExitError
		if errors.As(err, &exit) && strings.Contains(string(exit.Stderr), "not a git repository") {
			// Use the identity Initialize will create so requests queued before
			// init and requests arriving after it arbitrate the same repository.
			key := writePath(filepath.Join(abs, ".git"))
			return writeIdentity{common: key, worktree: key}, nil
		}
		return writeIdentity{}, fmt.Errorf("resolve Git write target: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		return writeIdentity{}, errors.New("git returned an invalid repository identity")
	}
	return writeIdentity{common: writePath(strings.TrimSpace(lines[1])), worktree: writePath(strings.TrimSpace(lines[0]))}, nil
}

// AcquireWrite waits for conflicting MyGo writes in this repository. Callers
// keep release until the actual child has exited and its result is recorded.
// Cancelling a view never releases a running write. Git's own locks continue
// to arbitrate commands launched outside this process.
//
// The returned context carries ownership through nested workflow steps. It
// must not be used to start independent concurrent write operations.
func AcquireWrite(ctx context.Context, root string, scope WriteScope) (context.Context, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if scope != WorktreeWrite && scope != ReferenceWrite {
		return ctx, nil, errors.New("invalid Git write scope")
	}
	id, err := resolveWriteIdentity(ctx, root)
	if err != nil {
		return ctx, nil, err
	}
	if owner, ok := ctx.Value(writeOwnerKey{}).(writeOwner); ok && owner.id.common == id.common {
		if owner.scope == ReferenceWrite || scope == WorktreeWrite && owner.id.worktree == id.worktree {
			return ctx, func() {}, nil
		}
		return ctx, nil, errors.New("Git workflow cannot upgrade its write scope while running")
	}
	release, err := repositoryWriteOwners.acquire(ctx, id, scope)
	if err != nil {
		return ctx, nil, err
	}
	return context.WithValue(ctx, writeOwnerKey{}, writeOwner{id: id, scope: scope}), release, nil
}

func (o *writeOwners) acquire(ctx context.Context, id writeIdentity, scope WriteScope) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w := &writeWaiter{id: id, scope: scope, ready: make(chan struct{})}
	o.mu.Lock()
	r := o.repositories[id.common]
	if r == nil {
		r = &repositoryWrites{worktrees: make(map[string]bool)}
		o.repositories[id.common] = r
	}
	r.waiting = append(r.waiting, w)
	o.grantLocked(r)
	o.mu.Unlock()
	select {
	case <-w.ready:
	case <-ctx.Done():
		o.mu.Lock()
		if !w.granted {
			for i, candidate := range r.waiting {
				if candidate == w {
					r.waiting = append(r.waiting[:i], r.waiting[i+1:]...)
					break
				}
			}
			o.grantLocked(r)
			o.removeIdleLocked(id.common, r)
			o.mu.Unlock()
			return nil, ctx.Err()
		}
		o.mu.Unlock()
	}
	var once sync.Once
	release := func() {
		once.Do(func() {
			o.mu.Lock()
			if scope == ReferenceWrite {
				r.reference = false
			} else {
				delete(r.worktrees, id.worktree)
			}
			o.grantLocked(r)
			o.removeIdleLocked(id.common, r)
			o.mu.Unlock()
		})
	}
	if err := ctx.Err(); err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func (o *writeOwners) grantLocked(r *repositoryWrites) {
	if r.reference {
		return
	}
	for i := 0; i < len(r.waiting); {
		w := r.waiting[i]
		if w.scope == ReferenceWrite {
			if len(r.worktrees) != 0 {
				return
			}
			r.reference = true
		} else if r.worktrees[w.id.worktree] {
			i++
			continue
		} else {
			r.worktrees[w.id.worktree] = true
		}
		r.waiting = append(r.waiting[:i], r.waiting[i+1:]...)
		w.granted = true
		close(w.ready)
		if r.reference {
			return
		}
	}
}

func (o *writeOwners) removeIdleLocked(key string, r *repositoryWrites) {
	if !r.reference && len(r.worktrees) == 0 && len(r.waiting) == 0 {
		delete(o.repositories, key)
	}
}

func acquireBackgroundWrite(root string, scope WriteScope) (func(), error) {
	_, release, err := AcquireWrite(context.Background(), root, scope)
	if err != nil {
		return nil, fmt.Errorf("Git write admission: %w", err)
	}
	return release, nil
}

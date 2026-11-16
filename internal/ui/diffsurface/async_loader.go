package diffsurface

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

const asyncGitReadTimeout = 15 * time.Second

type diffsurfaceGitOps struct {
	isRepository    func(root string) bool
	changes         func(root string) ([]git.Change, error)
	checkLargeFiles func(root string) ([]git.LargeFile, error)
	diffHEAD        func(root, relPath string) (string, error)
	diffScope       func(root, relPath string, staged bool) (string, error)
	compareBranch   func(ctx context.Context, root, path, baseRef string) (string, error)
	branchesContext func(ctx context.Context, root string) ([]git.Branch, error)
	readRepoStatus  func(ctx context.Context, root string) (git.RepositoryStatus, error)
	stagePaths      func(root string, paths ...string) (string, error)
	unstagePaths    func(root string, paths ...string) (string, error)
	prepareDiscard  func(root, path string) (git.DiscardPlan, error)
	discardReviewed func(root string, plan git.DiscardPlan) error
	preparePublish  func(root string, input git.PublishInput) (git.PublishInput, error)
}

var gitOps = defaultDiffsurfaceGitOps()

func defaultDiffsurfaceGitOps() diffsurfaceGitOps {
	return diffsurfaceGitOps{
		isRepository:    git.IsRepository,
		changes:         git.Changes,
		checkLargeFiles: git.CheckLargeFiles,
		diffHEAD:        git.DiffHEAD,
		diffScope:       git.DiffScope,
		compareBranch:   git.CompareBranch,
		branchesContext: git.BranchesContext,
		readRepoStatus:  git.ReadRepositoryStatus,
		stagePaths:      git.StagePaths,
		unstagePaths:    git.UnstagePaths,
		prepareDiscard:  git.PrepareDiscard,
		discardReviewed: func(root string, plan git.DiscardPlan) error {
			_, err := git.DiscardReviewed(root, plan)
			return err
		},
		preparePublish: git.PreparePublish,
	}
}

type reloadRequest struct {
	root   string
	gen    uint64
	seq    uint64
	cancel context.CancelFunc
}

type reloadResult struct {
	root       string
	gen        uint64
	seq        uint64
	repo       bool
	changes    []git.Change
	err        error
	largeFiles []git.LargeFile
	scanErr    error
}

type diffRequestKey struct {
	root        string
	gen         uint64
	path        string
	baseRef     string
	staged      bool
	againstHEAD bool
}

type diffRequest struct {
	key    diffRequestKey
	seq    uint64
	cancel context.CancelFunc
}

type diffCacheEntry struct {
	text     string
	errMsg   string
	seq      uint64
	prepared *preparedPatchDoc
}

type diffResult struct {
	key      diffRequestKey
	seq      uint64
	text     string
	errMsg   string
	prepared *preparedPatchDoc
}

type compareBranchesRequest struct {
	root   string
	gen    uint64
	seq    uint64
	cancel context.CancelFunc
}

type compareBranchesResult struct {
	root     string
	gen      uint64
	seq      uint64
	branches []git.Branch
	err      error
}

type stageRequest struct {
	root   string
	gen    uint64
	seq    uint64
	path   string
	stage  bool
	cancel context.CancelFunc
}

type stageResult struct {
	root  string
	gen   uint64
	seq   uint64
	path  string
	stage bool
	err   error
}

type discardPrepRequest struct {
	root   string
	gen    uint64
	seq    uint64
	path   string
	cancel context.CancelFunc
}

type discardPrepResult struct {
	root string
	gen  uint64
	seq  uint64
	plan git.DiscardPlan
	err  error
}

type publishPrepRequest struct {
	root   string
	gen    uint64
	seq    uint64
	cancel context.CancelFunc
}

type publishPrepResult struct {
	root  string
	gen   uint64
	seq   uint64
	input git.PublishInput
	err   error
}

// workbenchLoader owns all background single-flight git reads and generation
// fences via a heap pointer so value copies of Workbench (e.g. frame state
// snapshots) never embed a sync.Mutex value.
type workbenchLoader struct {
	mu         sync.Mutex
	closed     bool
	generation uint64
	root       string
	invalidate func()

	reloadSeq      uint64
	reloadInFlight *reloadRequest
	reloadPending  *reloadResult
	reloadReadyGen uint64

	diffSeq      uint64
	diffInFlight *diffRequest
	diffPending  *diffResult
	diffCache    map[diffRequestKey]diffCacheEntry

	compareBranchesSeq      uint64
	compareBranchesInFlight *compareBranchesRequest
	compareBranchesPending  *compareBranchesResult
	compareBranchesReadyGen uint64

	stageSeq      uint64
	stageInFlight *stageRequest
	stagePending  *stageResult

	discardPrepSeq      uint64
	discardPrepInFlight *discardPrepRequest
	discardPrepPending  *discardPrepResult

	publishPrepSeq      uint64
	publishPrepInFlight *publishPrepRequest
	publishPrepPending  *publishPrepResult
}

func newWorkbenchLoader() *workbenchLoader {
	return &workbenchLoader{
		generation: 1,
		diffCache:  map[diffRequestKey]diffCacheEntry{},
	}
}

func (w *Workbench) ensureLoader() *workbenchLoader {
	if w == nil {
		return nil
	}
	if w.loader == nil {
		w.loader = newWorkbenchLoader()
	}
	return w.loader
}

func (l *workbenchLoader) cancelAllLocked() {
	if l.reloadInFlight != nil {
		l.reloadInFlight.cancel()
		l.reloadInFlight = nil
	}
	l.reloadPending = nil
	l.reloadReadyGen = 0

	if l.diffInFlight != nil {
		l.diffInFlight.cancel()
		l.diffInFlight = nil
	}
	l.diffPending = nil
	clear(l.diffCache)

	if l.compareBranchesInFlight != nil {
		l.compareBranchesInFlight.cancel()
		l.compareBranchesInFlight = nil
	}
	l.compareBranchesPending = nil
	l.compareBranchesReadyGen = 0

	if l.stageInFlight != nil {
		l.stageInFlight.cancel()
		l.stageInFlight = nil
	}
	l.stagePending = nil

	if l.discardPrepInFlight != nil {
		l.discardPrepInFlight.cancel()
		l.discardPrepInFlight = nil
	}
	l.discardPrepPending = nil

	if l.publishPrepInFlight != nil {
		l.publishPrepInFlight.cancel()
		l.publishPrepInFlight = nil
	}
	l.publishPrepPending = nil
}

func (l *workbenchLoader) switchRoot(root string, invalidate func()) uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		l.closed = false
	}
	if invalidate != nil {
		l.invalidate = invalidate
	}
	if l.root != root {
		l.generation++
		if l.generation == 0 {
			l.generation = 1
		}
		l.root = root
		l.cancelAllLocked()
	}
	return l.generation
}

func (l *workbenchLoader) bind(root string) uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.generation == 0 {
		l.generation = 1
	}
	if l.root != root {
		l.generation++
		if l.generation == 0 {
			l.generation = 1
		}
		l.root = root
		l.cancelAllLocked()
	}
	return l.generation
}

func (l *workbenchLoader) currentGen() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.generation
}

func (l *workbenchLoader) currentRoot() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.root
}

func (l *workbenchLoader) notify() {
	l.mu.Lock()
	inv := l.invalidate
	closed := l.closed
	l.mu.Unlock()
	if !closed && inv != nil {
		inv()
	}
}

func (l *workbenchLoader) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	l.generation++
	if l.generation == 0 {
		l.generation = 1
	}
	l.cancelAllLocked()
}

// SetInvalidate stores the UI invalidation callback used when background
// loaders complete. Passing nil leaves any previously registered callback intact.
func (w *Workbench) SetInvalidate(invalidate func()) {
	w.setInvalidate(invalidate)
}

func (w *Workbench) setInvalidate(invalidate func()) {
	if w == nil || invalidate == nil {
		return
	}
	w.invalidate = invalidate
	l := w.ensureLoader()
	l.mu.Lock()
	l.invalidate = invalidate
	l.mu.Unlock()
}

func (w *Workbench) currentGen() uint64 {
	if w == nil || w.loader == nil {
		return 0
	}
	return w.loader.currentGen()
}

func (w *Workbench) bindRoot(root string) uint64 {
	if w == nil {
		return 0
	}
	l := w.ensureLoader()
	prevRoot := l.currentRoot()
	gen := l.bind(root)
	if prevRoot != root && !w.Loaded && !w.Staged && w.BaseRef == "" {
		w.AgainstHEAD = true
	}
	w.Root = root
	if w.invalidate != nil {
		l.mu.Lock()
		if l.invalidate == nil {
			l.invalidate = w.invalidate
		}
		l.mu.Unlock()
	}
	return gen
}

func (w *Workbench) hasReloadResultForRoot(root string) bool {
	if w == nil || w.loader == nil {
		return false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.root == root && l.reloadReadyGen == l.generation && l.generation != 0
}

func (w *Workbench) resetStateForRoot(root string, invalidate func()) {
	if w == nil {
		return
	}
	l := w.ensureLoader()
	operation := w.Operation
	w.cancelBranchList()
	w.cancelRepositoryStatus()
	w.cancelCommitPreparation()
	w.cancelRemotePreparation()
	w.CancelComparison()
	l.switchRoot(root, invalidate)
	effectiveInv := invalidate
	if effectiveInv == nil {
		l.mu.Lock()
		effectiveInv = l.invalidate
		l.mu.Unlock()
	}
	*w = Workbench{
		loader:      l,
		Root:        root,
		Operation:   operation,
		invalidate:  effectiveInv,
		AgainstHEAD: true,
	}
}

func (w *Workbench) isReloading(root string) bool {
	if w == nil || w.loader == nil {
		return false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reloadInFlight != nil && l.reloadInFlight.root == root && l.reloadInFlight.gen == l.generation
}

func (w *Workbench) isCompareBranchesLoading() bool {
	if w == nil || w.loader == nil {
		return false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.compareBranchesInFlight != nil && l.compareBranchesInFlight.root == w.Root && l.compareBranchesInFlight.gen == l.generation
}

func (w *Workbench) isStagingBusy() bool {
	if w == nil || w.loader == nil {
		return false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stageInFlight != nil && l.stageInFlight.root == w.Root && l.stageInFlight.gen == l.generation
}

func (w *Workbench) isDiscardPreparing() bool {
	if w == nil || w.loader == nil {
		return false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.discardPrepInFlight != nil && l.discardPrepInFlight.root == w.Root && l.discardPrepInFlight.gen == l.generation
}

func (w *Workbench) isPublishPreparing() bool {
	if w == nil || w.loader == nil {
		return false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.publishPrepInFlight != nil && l.publishPrepInFlight.root == w.Root && l.publishPrepInFlight.gen == l.generation
}

func (w *Workbench) requestReload(root string, force bool) {
	if w == nil {
		return
	}
	root = strings.TrimSpace(root)
	gen := w.bindRoot(root)
	if root == "" {
		w.Loaded = true
		w.Repository = false
		w.Changes = nil
		return
	}
	w.reloadRepositoryStatusForce(force)

	l := w.ensureLoader()
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	if !force {
		if l.reloadInFlight != nil && l.reloadInFlight.root == root && l.reloadInFlight.gen == gen {
			l.mu.Unlock()
			return
		}
		if l.reloadReadyGen == gen && l.root == root {
			l.mu.Unlock()
			return
		}
	}
	if l.reloadInFlight != nil {
		l.reloadInFlight.cancel()
		l.reloadInFlight = nil
	}
	l.reloadSeq++
	seq := l.reloadSeq
	ctx, cancel := context.WithTimeout(context.Background(), asyncGitReadTimeout)
	req := &reloadRequest{root: root, gen: gen, seq: seq, cancel: cancel}
	l.reloadInFlight = req
	ops := gitOps
	l.mu.Unlock()

	go func() {
		defer cancel()
		if ctx.Err() != nil {
			return
		}
		repo := ops.isRepository(root)
		if ctx.Err() != nil {
			return
		}
		changes, err := ops.changes(root)
		if ctx.Err() != nil {
			return
		}
		var large []git.LargeFile
		var scanErr error
		if repo && err == nil {
			large, scanErr = ops.checkLargeFiles(root)
		}
		if ctx.Err() != nil {
			return
		}

		l.mu.Lock()
		valid := !l.closed && l.generation == gen && l.root == root && l.reloadInFlight == req
		if valid {
			l.reloadInFlight = nil
			l.reloadReadyGen = gen
			l.reloadPending = &reloadResult{
				root:       root,
				gen:        gen,
				seq:        seq,
				repo:       repo,
				changes:    changes,
				err:        err,
				largeFiles: large,
				scanErr:    scanErr,
			}
		}
		l.mu.Unlock()
		if valid {
			l.notify()
		}
	}()
}

func (w *Workbench) finishReload() {
	if w == nil || w.loader == nil {
		return
	}
	l := w.loader
	l.mu.Lock()
	res := l.reloadPending
	if res == nil || l.closed || res.gen != l.generation || res.root != l.root || res.root != w.Root {
		l.reloadPending = nil
		l.mu.Unlock()
		return
	}
	l.reloadPending = nil
	l.mu.Unlock()

	w.Loaded = true
	w.Repository = res.repo
	if res.err != nil {
		w.Status = res.err.Error()
		w.Changes = nil
	} else {
		w.Changes = res.changes
		if w.Commit != nil {
			w.Commit.LargeFiles = res.largeFiles
			w.Commit.ScanError = ""
			if res.scanErr != nil {
				w.Commit.ScanError = res.scanErr.Error()
			}
		} else if len(res.largeFiles) > 0 || res.scanErr != nil {
			s := w.commitState()
			s.LargeFiles = res.largeFiles
			if res.scanErr != nil {
				s.ScanError = res.scanErr.Error()
			}
		}
	}

	// Complete deferred commit_push / commit_push_pr clean-tree redirection once
	// repository changes finish loading.
	if w.pendingCaptionAction != "" && w.pendingCaptionGen == res.gen && w.pendingCaptionRoot == res.root {
		action := w.pendingCaptionAction
		w.pendingCaptionAction = ""
		w.pendingCaptionRoot = ""
		w.pendingCaptionGen = 0
		if len(w.Changes) == 0 && (action == "commit_push" || action == "commit_push_pr") {
			w.cancelCommitPreparation()
			w.Commit = nil
			mode := "push"
			if action == "commit_push_pr" {
				mode = "pr"
			}
			w.Remote = &remoteWorkflowState{Mode: mode, Draft: true}
			if mode == "pr" {
				w.Remote.Title = strings.SplitN(w.Message, "\n", 2)[0]
			}
			if w.captionReview != nil && w.captionReview.root == res.root {
				w.captionReview.kind = "remote"
			}
		}
	}
}

func (w *Workbench) requestCompareBranches(force bool) {
	if w == nil || strings.TrimSpace(w.Root) == "" {
		return
	}
	root := w.Root
	gen := w.bindRoot(root)
	l := w.ensureLoader()
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	if !force {
		if l.compareBranchesInFlight != nil && l.compareBranchesInFlight.root == root && l.compareBranchesInFlight.gen == gen {
			l.mu.Unlock()
			return
		}
		if l.compareBranchesReadyGen == gen && l.root == root && w.CompareError == "" {
			l.mu.Unlock()
			return
		}
	}
	if l.compareBranchesInFlight != nil {
		l.compareBranchesInFlight.cancel()
		l.compareBranchesInFlight = nil
	}
	l.compareBranchesSeq++
	seq := l.compareBranchesSeq
	ctx, cancel := context.WithTimeout(context.Background(), asyncGitReadTimeout)
	req := &compareBranchesRequest{root: root, gen: gen, seq: seq, cancel: cancel}
	l.compareBranchesInFlight = req
	ops := gitOps
	l.mu.Unlock()

	go func() {
		defer cancel()
		branches, err := ops.branchesContext(ctx, root)
		if ctx.Err() != nil {
			return
		}
		l.mu.Lock()
		valid := !l.closed && l.generation == gen && l.root == root && l.compareBranchesInFlight == req
		if valid {
			l.compareBranchesInFlight = nil
			l.compareBranchesReadyGen = gen
			l.compareBranchesPending = &compareBranchesResult{
				root:     root,
				gen:      gen,
				seq:      seq,
				branches: branches,
				err:      err,
			}
		}
		l.mu.Unlock()
		if valid {
			l.notify()
		}
	}()
}

func (w *Workbench) finishCompareBranches() {
	if w == nil || w.loader == nil {
		return
	}
	l := w.loader
	l.mu.Lock()
	res := l.compareBranchesPending
	if res == nil || l.closed || res.gen != l.generation || res.root != l.root || res.root != w.Root {
		l.compareBranchesPending = nil
		l.mu.Unlock()
		return
	}
	l.compareBranchesPending = nil
	l.mu.Unlock()

	w.CompareBranches = res.branches
	w.CompareError = ""
	if res.err != nil {
		w.CompareError = res.err.Error()
	}
}

func formatDiffOutcome(out string, err error, mapNotRepo bool) (text, errMsg string) {
	if err != nil {
		if mapNotRepo && errors.Is(err, git.ErrNotRepo) {
			return "", i18n.T(i18n.KeyDiffNotRepo)
		}
		return "", strings.TrimSpace(err.Error())
	}
	if strings.TrimSpace(out) == "" {
		return "", i18n.T(i18n.KeyDiffEmpty)
	}
	return out, ""
}

func diffKeyForWorkbench(w *Workbench, root, relPath string, gen uint64) diffRequestKey {
	key := diffRequestKey{
		root:        root,
		gen:         gen,
		path:        relPath,
		againstHEAD: true,
	}
	if w != nil {
		key.baseRef = w.BaseRef
		key.staged = w.Staged
		if w.Loaded || w.Staged || w.BaseRef != "" {
			key.againstHEAD = w.AgainstHEAD
		}
	}
	return key
}

// RequestDiffAsync starts a single-flight background diff load for
// (root, relPath) bound to w's current workspace generation and scope.
// Concurrent or repeated requests with identical scope coalesce without
// spawning duplicate git processes unless force is true.
func (w *Workbench) RequestDiffAsync(root, relPath string, force bool, invalidate func()) {
	if w == nil {
		return
	}
	if invalidate != nil {
		w.setInvalidate(invalidate)
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return
	}
	gen := w.bindRoot(root)
	key := diffKeyForWorkbench(w, root, relPath, gen)

	l := w.ensureLoader()
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	if !force {
		if l.diffInFlight != nil && l.diffInFlight.key == key {
			l.mu.Unlock()
			return
		}
		if _, cached := l.diffCache[key]; cached {
			l.mu.Unlock()
			return
		}
	}
	if l.diffInFlight != nil {
		l.diffInFlight.cancel()
		l.diffInFlight = nil
	}
	if force {
		delete(l.diffCache, key)
	}
	l.diffSeq++
	seq := l.diffSeq
	ctx, cancel := context.WithTimeout(context.Background(), asyncGitReadTimeout)
	req := &diffRequest{key: key, seq: seq, cancel: cancel}
	l.diffInFlight = req
	ops := gitOps
	viewSplit, viewWrap := w.View.Split, w.View.Wrap
	l.mu.Unlock()

	go func() {
		defer cancel()
		if ctx.Err() != nil {
			return
		}
		var out string
		var err error
		if key.baseRef != "" {
			out, err = ops.compareBranch(ctx, key.root, key.path, key.baseRef)
		} else if key.againstHEAD {
			out, err = ops.diffHEAD(key.root, key.path)
		} else {
			out, err = ops.diffScope(key.root, key.path, key.staged)
		}
		if ctx.Err() != nil {
			return
		}
		text, errMsg := formatDiffOutcome(out, err, true)
		var prepared *preparedPatchDoc
		if errMsg == "" && strings.TrimSpace(text) != "" {
			prepared = prebuildPatchDocument(text, viewSplit, viewWrap)
		}
		if ctx.Err() != nil {
			return
		}

		l.mu.Lock()
		valid := !l.closed && l.generation == key.gen && l.root == key.root && l.diffInFlight == req
		if valid {
			l.diffInFlight = nil
			l.diffCache[key] = diffCacheEntry{text: text, errMsg: errMsg, seq: seq, prepared: prepared}
			l.diffPending = &diffResult{key: key, seq: seq, text: text, errMsg: errMsg, prepared: prepared}
		}
		l.mu.Unlock()
		if valid {
			l.notify()
		}
	}()
}

// PollDiffAsync checks whether a ready diff snapshot exists for (root, relPath)
// matching w's current generation and scope, returning (text, errMsg, ready, loading).
func (w *Workbench) PollDiffAsync(root, relPath string) (text, errMsg string, ready, loading bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", i18n.T(i18n.KeyDiffNoWorkspace), true, false
	}
	if w == nil || w.loader == nil {
		return "", "", false, false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || l.root != root {
		return "", "", false, false
	}
	key := diffKeyForWorkbench(w, root, relPath, l.generation)
	if res := l.diffPending; res != nil && res.key == key {
		l.diffPending = nil
		if res.prepared != nil {
			w.View.applyPrepared(res.prepared)
		}
		return res.text, res.errMsg, true, false
	}
	if entry, ok := l.diffCache[key]; ok {
		if entry.prepared != nil {
			w.View.applyPrepared(entry.prepared)
		}
		return entry.text, entry.errMsg, true, l.diffInFlight != nil && l.diffInFlight.key == key
	}
	loading = l.diffInFlight != nil && l.diffInFlight.key == key
	return "", "", false, loading
}

// IsDiffLoading reports whether a background diff load for (root, relPath)
// matching w's current generation and scope is in progress.
func (w *Workbench) IsDiffLoading(root, relPath string) bool {
	if w == nil || w.loader == nil {
		return false
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	l := w.loader
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed || l.root != root {
		return false
	}
	key := diffKeyForWorkbench(w, root, relPath, l.generation)
	return l.diffInFlight != nil && l.diffInFlight.key == key
}

// EnsureDiffAsync ensures a non-blocking diff load is started for (root, relPath)
// and populates *text and *errMsg once ready. Intended for frame callers replacing
// synchronous LoadDiff reads.
func EnsureDiffAsync(w *Workbench, root, relPath string, text, errMsg *string, invalidate func()) (loading bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		if text != nil {
			*text = ""
		}
		if errMsg != nil {
			*errMsg = i18n.T(i18n.KeyDiffNoWorkspace)
		}
		return false
	}
	if w == nil {
		return defaultDiffCache.ensure(root, relPath, false, text, errMsg, invalidate)
	}
	w.RequestDiffAsync(root, relPath, false, invalidate)
	out, eMsg, ready, inflight := w.PollDiffAsync(root, relPath)
	if ready {
		if text != nil {
			*text = out
		}
		if errMsg != nil {
			*errMsg = eMsg
		}
	}
	return inflight
}

// StartDiffAsync clears current diff strings, resets branch/staged filters on w
// when present, and starts a forced non-blocking diff reload. Intended for
// click callbacks (such as frame OnShowDiff) replacing synchronous LoadDiff.
func StartDiffAsync(w *Workbench, root, relPath string, text, errMsg *string, invalidate func()) {
	if text != nil {
		*text = ""
	}
	if errMsg != nil {
		*errMsg = ""
	}
	if w != nil {
		w.CancelComparison()
		w.BaseRef = ""
		w.Staged = false
		w.AgainstHEAD = true
		w.RequestDiffAsync(root, relPath, true, invalidate)
		return
	}
	defaultDiffCache.ensure(strings.TrimSpace(root), relPath, true, text, errMsg, invalidate)
}

func (w *Workbench) invalidateDiffCache() {
	if w == nil || w.loader == nil {
		return
	}
	l := w.loader
	l.mu.Lock()
	clear(l.diffCache)
	l.diffPending = nil
	l.mu.Unlock()
}

func (w *Workbench) requestStage(m Model, ch git.Change, stage bool) {
	if w == nil {
		return
	}
	root := strings.TrimSpace(m.WorkspaceRoot)
	if root == "" {
		return
	}
	gen := w.bindRoot(root)
	paths := []string{ch.Path}
	if ch.Original != "" {
		paths = append(paths, ch.Original)
	}
	w.invalidateCommitReview()
	if stage {
		w.Status = word("正在暂存：", "Staging: ") + ch.Path
	} else {
		w.Status = word("正在取消暂存：", "Unstaging: ") + ch.Path
	}

	l := w.ensureLoader()
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	if l.stageInFlight != nil {
		l.stageInFlight.cancel()
		l.stageInFlight = nil
	}
	l.stageSeq++
	seq := l.stageSeq
	ctx, cancel := context.WithTimeout(context.Background(), asyncGitReadTimeout)
	req := &stageRequest{root: root, gen: gen, seq: seq, path: ch.Path, stage: stage, cancel: cancel}
	l.stageInFlight = req
	ops := gitOps
	l.mu.Unlock()

	go func() {
		defer cancel()
		if ctx.Err() != nil {
			return
		}
		var err error
		if stage {
			_, err = ops.stagePaths(root, paths...)
		} else {
			_, err = ops.unstagePaths(root, paths...)
		}
		if ctx.Err() != nil {
			return
		}
		l.mu.Lock()
		valid := !l.closed && l.generation == gen && l.root == root && l.stageInFlight == req
		if valid {
			l.stageInFlight = nil
			l.stagePending = &stageResult{root: root, gen: gen, seq: seq, path: ch.Path, stage: stage, err: err}
		}
		l.mu.Unlock()
		if valid {
			l.notify()
		}
	}()
}

func (w *Workbench) finishStage(m Model) {
	if w == nil || w.loader == nil {
		return
	}
	l := w.loader
	l.mu.Lock()
	res := l.stagePending
	if res == nil || l.closed || res.gen != l.generation || res.root != l.root || res.root != w.Root || res.root != m.WorkspaceRoot {
		l.stagePending = nil
		l.mu.Unlock()
		return
	}
	l.stagePending = nil
	l.mu.Unlock()

	if res.err != nil {
		w.Status = res.err.Error()
	} else if res.stage {
		w.Status = word("已暂存：", "Staged: ") + res.path
	} else {
		w.Status = word("已取消暂存：", "Unstaged: ") + res.path
	}
	w.invalidateDiffCache()
	w.requestReload(m.WorkspaceRoot, true)
	refreshWithForce(m, true)
}

func (w *Workbench) requestDiscardPrepare(m Model, path string) {
	if w == nil {
		return
	}
	root := strings.TrimSpace(m.WorkspaceRoot)
	if root == "" {
		return
	}
	gen := w.bindRoot(root)
	l := w.ensureLoader()
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	if l.discardPrepInFlight != nil && l.discardPrepInFlight.root == root && l.discardPrepInFlight.gen == gen && l.discardPrepInFlight.path == path {
		l.mu.Unlock()
		return
	}
	if l.discardPrepInFlight != nil {
		l.discardPrepInFlight.cancel()
		l.discardPrepInFlight = nil
	}
	l.discardPrepSeq++
	seq := l.discardPrepSeq
	ctx, cancel := context.WithTimeout(context.Background(), asyncGitReadTimeout)
	req := &discardPrepRequest{root: root, gen: gen, seq: seq, path: path, cancel: cancel}
	l.discardPrepInFlight = req
	ops := gitOps
	l.mu.Unlock()

	w.Status = word("正在检查还原影响…", "Checking discard impact…")

	go func() {
		defer cancel()
		if ctx.Err() != nil {
			return
		}
		plan, err := ops.prepareDiscard(root, path)
		if ctx.Err() != nil {
			return
		}
		l.mu.Lock()
		valid := !l.closed && l.generation == gen && l.root == root && l.discardPrepInFlight == req
		if valid {
			l.discardPrepInFlight = nil
			l.discardPrepPending = &discardPrepResult{root: root, gen: gen, seq: seq, plan: plan, err: err}
		}
		l.mu.Unlock()
		if valid {
			l.notify()
		}
	}()
}

func (w *Workbench) finishDiscardPrepare(m Model) {
	if w == nil || w.loader == nil {
		return
	}
	l := w.loader
	l.mu.Lock()
	res := l.discardPrepPending
	if res == nil || l.closed || res.gen != l.generation || res.root != l.root || res.root != w.Root || res.root != m.WorkspaceRoot {
		l.discardPrepPending = nil
		l.mu.Unlock()
		return
	}
	l.discardPrepPending = nil
	l.mu.Unlock()

	if res.err != nil {
		w.Status = res.err.Error()
		w.Discard = nil
		return
	}
	plan := res.plan
	w.Discard = &plan
	w.Status = ""
}

func (w *Workbench) requestPublishPrepare(m Model, input git.PublishInput) {
	if w == nil {
		return
	}
	root := strings.TrimSpace(m.WorkspaceRoot)
	if root == "" {
		return
	}
	gen := w.bindRoot(root)
	l := w.ensureLoader()
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return
	}
	if l.publishPrepInFlight != nil && l.publishPrepInFlight.root == root && l.publishPrepInFlight.gen == gen {
		l.mu.Unlock()
		return
	}
	if l.publishPrepInFlight != nil {
		l.publishPrepInFlight.cancel()
		l.publishPrepInFlight = nil
	}
	l.publishPrepSeq++
	seq := l.publishPrepSeq
	ctx, cancel := context.WithTimeout(context.Background(), asyncGitReadTimeout)
	req := &publishPrepRequest{root: root, gen: gen, seq: seq, cancel: cancel}
	l.publishPrepInFlight = req
	ops := gitOps
	l.mu.Unlock()

	w.Status = word("正在核对发布目标…", "Checking publish target…")

	go func() {
		defer cancel()
		if ctx.Err() != nil {
			return
		}
		resolved, err := ops.preparePublish(root, input)
		if ctx.Err() != nil {
			return
		}
		l.mu.Lock()
		valid := !l.closed && l.generation == gen && l.root == root && l.publishPrepInFlight == req
		if valid {
			l.publishPrepInFlight = nil
			l.publishPrepPending = &publishPrepResult{root: root, gen: gen, seq: seq, input: resolved, err: err}
		}
		l.mu.Unlock()
		if valid {
			l.notify()
		}
	}()
}

func (w *Workbench) finishPublishPrepare(m Model) {
	if w == nil || w.loader == nil {
		return
	}
	l := w.loader
	l.mu.Lock()
	res := l.publishPrepPending
	if res == nil || l.closed || res.gen != l.generation || res.root != l.root || res.root != w.Root || res.root != m.WorkspaceRoot {
		l.publishPrepPending = nil
		l.mu.Unlock()
		return
	}
	l.publishPrepPending = nil
	l.mu.Unlock()

	if res.err != nil {
		w.Status = res.err.Error()
		return
	}
	w.ReviewedPublish = res.input
	w.PublishReview = true
	w.Status = ""
}

func (w *Workbench) finishAsync(m Model) {
	if w == nil {
		return
	}
	w.finishReload()
	w.finishRepositoryStatus()
	w.finishCompareBranches()
	w.finishStage(m)
	w.finishDiscardPrepare(m)
	w.finishPublishPrepare(m)
	finishComparison(m)
	finishCommitPreparation(m)
	finishRemotePreparation(m)
	finishGitOperation(m)
	applyDiffResultToModel(m)
}

func applyDiffResultToModel(m Model) {
	w := m.Workbench
	if w == nil || (m.Text == nil && m.ErrMsg == nil) {
		return
	}
	if strings.TrimSpace(m.WorkspaceRoot) == "" || w.BaseRef != "" {
		return
	}
	rel := ""
	if m.PathFilter != nil {
		rel = *m.PathFilter
	}
	if w.loader == nil {
		return
	}
	l := w.loader
	l.mu.Lock()
	res := l.diffPending
	key := diffKeyForWorkbench(w, m.WorkspaceRoot, rel, l.generation)
	if res != nil && !l.closed && res.key == key {
		l.diffPending = nil
		l.mu.Unlock()
		if m.Text != nil {
			*m.Text = res.text
		}
		if m.ErrMsg != nil {
			*m.ErrMsg = res.errMsg
		}
		return
	}
	entry, cached := l.diffCache[key]
	inflight := l.diffInFlight != nil && l.diffInFlight.key == key
	l.mu.Unlock()

	if cached && !inflight && (m.Text == nil || *m.Text == "") && (m.ErrMsg == nil || *m.ErrMsg == "") {
		if m.Text != nil {
			*m.Text = entry.text
		}
		if m.ErrMsg != nil {
			*m.ErrMsg = entry.errMsg
		}
	}
}

// fallbackDiffLoader handles View/EnsureDiffAsync calls when Workbench is nil.
type fallbackDiffLoader struct {
	mu       sync.Mutex
	gen      uint64
	inFlight map[diffRequestKey]*fallbackDiffReq
	cache    map[diffRequestKey]diffCacheEntry
}

type fallbackDiffReq struct {
	key    diffRequestKey
	cancel context.CancelFunc
}

var defaultDiffCache = &fallbackDiffLoader{
	inFlight: map[diffRequestKey]*fallbackDiffReq{},
	cache:    map[diffRequestKey]diffCacheEntry{},
}

func (f *fallbackDiffLoader) ensure(root, relPath string, force bool, text, errMsg *string, invalidate func()) bool {
	if root == "" {
		if text != nil {
			*text = ""
		}
		if errMsg != nil {
			*errMsg = i18n.T(i18n.KeyDiffNoWorkspace)
		}
		return false
	}
	f.mu.Lock()
	if f.gen == 0 {
		f.gen = 1
	}
	key := diffRequestKey{root: root, gen: f.gen, path: relPath, againstHEAD: true}
	if !force {
		if entry, ok := f.cache[key]; ok {
			f.mu.Unlock()
			if text != nil {
				*text = entry.text
			}
			if errMsg != nil {
				*errMsg = entry.errMsg
			}
			return false
		}
		if f.inFlight[key] != nil {
			f.mu.Unlock()
			return true
		}
	} else {
		delete(f.cache, key)
		if old := f.inFlight[key]; old != nil {
			old.cancel()
			delete(f.inFlight, key)
		}
	}
	// Bound fallback cache size to avoid unbounded memory growth.
	if len(f.cache) >= 64 {
		clear(f.cache)
	}
	ctx, cancel := context.WithTimeout(context.Background(), asyncGitReadTimeout)
	req := &fallbackDiffReq{key: key, cancel: cancel}
	f.inFlight[key] = req
	ops := gitOps
	f.mu.Unlock()

	go func() {
		defer cancel()
		if ctx.Err() != nil {
			return
		}
		out, err := ops.diffHEAD(root, relPath)
		if ctx.Err() != nil {
			return
		}
		t, e := formatDiffOutcome(out, err, true)
		var prepared *preparedPatchDoc
		if e == "" && strings.TrimSpace(t) != "" {
			prepared = prebuildPatchDocument(t, false, false)
		}
		if ctx.Err() != nil {
			return
		}
		f.mu.Lock()
		active := f.inFlight[key] == req && f.gen == key.gen
		if active {
			delete(f.inFlight, key)
			f.cache[key] = diffCacheEntry{text: t, errMsg: e, prepared: prepared}
		}
		f.mu.Unlock()
		if active && invalidate != nil {
			invalidate()
		}
	}()
	return true
}

package diffsurface

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/i18n"
)

// These Git commands execute only when root later runs tests. Every mutation
// is confined to t.TempDir, without remotes, hooks, signing or account access.
func captionFixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	hooks := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		flags := []string{"-c", "user.name=Caption Fixture", "-c", "user.email=fixture@local.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + hooks}
		cmd := exec.Command("git", append(flags, args...)...)
		cmd.Dir = root
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(entry), "GIT_") {
				cmd.Env = append(cmd.Env, entry)
			}
		}
		cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated git %v: %v: %s", args, err, output)
		}
	}
	run("init", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "--", "tracked.txt")
	run("commit", "-m", "isolated fixture")
	return root
}

// awaitCaptionFixture drives frames until the fixture settles. It polls the
// condition instead of sleeping a fixed amount, so a fast machine finishes fast
// and a loaded one still has 2s of headroom; the failure names the condition and
// the caller-supplied state so a red run explains itself.
func awaitCaptionFixture(t *testing.T, tt *ui.Tester, ready func() bool, detail ...func() string) {
	t.Helper()
	// The whole suite runs in parallel on a loaded machine, so allow more than
	// one scheduler quantum for a frame to settle.
	const budget = 15 * time.Second
	deadline := time.Now().Add(budget)
	frames := 0
	for time.Now().Before(deadline) {
		tt.Frame()
		frames++
		if ready() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	state := ""
	for _, describe := range detail {
		if describe != nil {
			state += " " + describe()
		}
	}
	t.Fatalf("caption fixture did not settle within %s after %d frames;%s", budget, frames, state)
}

func TestCaptionInitializeIsOneGestureAndCapturesItsRoot(t *testing.T) {
	// Go also uses GOTMPDIR for t.TempDir. The development runner keeps it
	// inside the checkout, where Git would find the existing parent repository
	// and correctly skip init. This fixture needs genuinely unversioned roots.
	t.Setenv("GOTMPDIR", "")
	root, next := t.TempDir(), t.TempDir()
	if git.IsRepository(root) || git.IsRepository(next) {
		t.Fatal("initialization fixture inherited a parent Git repository")
	}
	current := root
	w := &Workbench{}
	accepted := false
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Workbench: w, WorkspaceRoot: current, CaptionAdmission: func() bool { return true }}
		if ui.Button(c, "Initialize captured root").Clicked() {
			accepted = CaptionAction(c, m, "", "init", true)
		}
		CaptionActivity(c, m)
	}, 700, 500)
	if _, err := os.Stat(filepath.Join(root, ".git")); !os.IsNotExist(err) {
		t.Fatal("painting initialized a repository")
	}
	if err := tt.Click("Initialize captured root"); err != nil {
		t.Fatal(err)
	}
	current = next
	awaitCaptionFixture(t, tt, func() bool { return w.Operation == nil }, func() string {
		return fmt.Sprintf("operation=%v accepted=%v status=%q", w.Operation != nil, accepted, w.Status)
	})
	if !accepted {
		t.Fatal("one gesture did not admit initialization")
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Fatalf("captured root was not initialized: %v; status=%q", err, w.Status)
	}
	if _, err := os.Stat(filepath.Join(next, ".git")); !os.IsNotExist(err) {
		t.Fatal("initialization crossed into a replacement cwd")
	}
	if w.CaptionReviewOpen() || w.Commit != nil && w.Commit.Task != nil {
		t.Fatal("Initialize inserted a second review or preparation")
	}
}

func TestCaptionGlobalReviewUsesSameWorkbenchAcrossPageAndRetiresOwner(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	root := captionFixtureRepo(t)
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	w := &Workbench{}
	defer w.ClosePane()
	admitted, global := true, true
	m := Model{Workbench: w, WorkspaceRoot: root, CaptionAdmission: func() bool { return admitted }}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Text(c, "Current global page")
		if ui.Button(c, "Open global commit").Clicked() {
			if !CaptionAction(c, m, "explicit fixture message", "commit_push", true) {
				t.Error("global gesture rejected")
			}
		}
		if !global {
			paintWorkbench(c, m, c.Theme().Text, c.Theme().TextMuted)
		}
		CaptionReview(c, m)
	}, 900, 900)
	if err := tt.Click("Open global commit"); err != nil {
		t.Fatal(err)
	}
	selection := w.Commit
	if !w.CaptionReviewOpen() || selection == nil || !tt.HasText("Commit changes") || !tt.HasText("tracked.txt") {
		t.Fatal("review was hidden behind the global page", tt.Texts())
	}
	if w.Operation != nil || selection.Task != nil || selection.Plan != nil || w.Remote != nil {
		t.Fatal("opening the global modal automatically prepared or executed an action")
	}
	global = false
	tt.Frame()
	count := 0
	for _, text := range tt.Texts() {
		if text == "Review commit and push…" {
			count++
		}
	}
	if w.Commit != selection || count != 1 {
		t.Fatalf("navigation cloned or duplicated the review: %d", count)
	}
	ctx, cancel := context.WithCancel(context.Background())
	task := &commitPreparation{root: root, cancel: cancel, done: make(chan commitPrepared, 1)}
	selection.Task = task
	admitted = false
	tt.Frame()
	if w.CaptionReviewOpen() || ctx.Err() == nil || selection.Task != nil || w.Operation != nil {
		t.Fatal("replacement owner retained executable review/preparation")
	}
	task.done <- commitPrepared{plan: git.SelectedCommitPlan{Summary: "retired result"}}
	finishCommitPreparation(m)
	if selection.Plan != nil {
		t.Fatal("retired preparation restored the review")
	}
}

func TestCaptionConfirmationAndGenerationRecheckAdmission(t *testing.T) {
	w := &Workbench{Root: "fixture", Commit: &commitSelectionState{Excluded: map[string]bool{}, Plan: &git.SelectedCommitPlan{}, StackPlan: &git.StackPlan{}}, Remote: &remoteWorkflowState{}}
	m := Model{Workbench: w, WorkspaceRoot: "fixture", CaptionAdmission: func() bool { return false }}
	ui.NewTester(func(c *ui.Context) {
		confirmSelectedCommit(c, m)
		beginCommitPreparation(c, m, true, true)
		beginRemotePreparation(c, m)
		beginPRGeneration(c, m)
	}, 400, 300)
	if w.Operation != nil || w.Commit.Task != nil || w.Remote.Task != nil || w.Remote.Generation != nil {
		t.Fatal("revoked admission reached an operation/model/preparation")
	}
}

func TestCaptionCleanCommitPushRoutesExistingPushReview(t *testing.T) {
	root := captionFixtureRepo(t)
	w := &Workbench{}
	defer w.ClosePane()
	if !w.RequestCaptionAction(root, "", "commit_push") || w.Remote == nil || w.Remote.Mode != "push" || w.Operation != nil || w.Remote.Task != nil {
		t.Fatal("clean default quick Push was trapped in empty file selection")
	}
}

func TestCaptionBranchPickerLoadsExistingOwnerAndCreateDoesNotRunUntilSubmit(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	root := captionFixtureRepo(t)
	w := &Workbench{Root: root}
	defer w.ClosePane()
	open := false
	m := Model{Workbench: w, WorkspaceRoot: root}
	tt := ui.NewTester(func(c *ui.Context) {
		anchor := ui.Button(c, "Choose branch")
		if anchor.Clicked() {
			open = !open
		}
		CaptionBranches(c, m, anchor, &open)
	}, 700, 650)
	if err := tt.Click("Choose branch"); err != nil {
		t.Fatal(err)
	}
	awaitCaptionFixture(t, tt, func() bool { return w.branchesRequest == nil && len(w.Branches) > 0 })
	if !tt.HasText("main") {
		t.Fatal("titlebar picker never consumed the branch list")
	}
	w.BranchQuery = "feature/fixture"
	tt.Frame()
	if err := tt.Click("Create branch feature/fixture…"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Create and switch") || w.Operation != nil {
		t.Fatal("opening create dialog mutated Git or hid its form")
	}
	tt.Key(0, ui.KeyEscape)
	if w.Operation != nil {
		t.Fatal("dismissing create dialog started a branch operation")
	}
}

func TestCaptionBranchReadFromRetiredRootCannotPublish(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	request := &branchListRequest{root: "old", cancel: cancel, done: make(chan branchListResult, 1)}
	w := &Workbench{Root: "new", BranchesOpen: true, Branches: []git.Branch{{Name: "new-main"}}, branchesRequest: request}
	request.done <- branchListResult{branches: []git.Branch{{Name: "retired-main"}}}
	w.finishBranchList()
	if ctx.Err() == nil || w.branchesRequest != nil || len(w.Branches) != 1 || w.Branches[0].Name != "new-main" {
		t.Fatal("old root branch response entered the current picker")
	}
}

func TestGitActionsDetachedBehindDirtyDisabledReasonsAndTooltips(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)

	for _, tc := range []struct {
		locale            string
		state             string
		status            git.RepositoryStatus
		changes           []git.Change
		wantNotice        string
		wantCommitTip     string
		wantPushTip       string
		wantPRTip         string
		wantPublishTip    string
		commitBtnLabel    string
		pushBtnLabel      string
		prBtnLabel        string
		publishBtnLabel   string
		detachedHeadLabel string
	}{
		{
			locale:            "en",
			state:             "detached-clean",
			status:            git.RepositoryStatus{IsRepo: true, Branch: "", HasOrigin: true},
			wantNotice:        "Detached HEAD: create and checkout a branch to enable push and pull request actions.",
			wantCommitTip:     "Worktree is clean. Make changes before committing.",
			wantPushTip:       "Detached HEAD: checkout a branch before pushing.",
			wantPRTip:         "Detached HEAD: checkout a branch before creating a pull request.",
			wantPublishTip:    "Publish repository is unavailable.",
			commitBtnLabel:    "Commit…",
			pushBtnLabel:      "Push…",
			prBtnLabel:        "Review draft PR…",
			publishBtnLabel:   "Publish repository…",
			detachedHeadLabel: "(detached HEAD)  ↑0  ↓0",
		},
		{
			locale:            "zh",
			state:             "detached-clean",
			status:            git.RepositoryStatus{IsRepo: true, Branch: "", HasOrigin: true},
			wantNotice:        "处于分离 HEAD 状态：请先创建并检出分支，才能执行推送和拉取请求操作。",
			wantCommitTip:     "工作区干净，请先修改文件后再提交。",
			wantPushTip:       "处于分离 HEAD 状态：请先检出分支后再推送。",
			wantPRTip:         "处于分离 HEAD 状态：请先检出分支后再创建拉取请求。",
			wantPublishTip:    "发布仓库不可用。",
			commitBtnLabel:    "提交…",
			pushBtnLabel:      "推送…",
			prBtnLabel:        "核对草稿 PR…",
			publishBtnLabel:   "发布仓库…",
			detachedHeadLabel: "（分离 HEAD）  ↑0  ↓0",
		},
		{
			locale:          "en",
			state:           "behind-clean",
			status:          git.RepositoryStatus{IsRepo: true, Branch: "feature/behind", HasOrigin: true, HasUpstream: true, Behind: 2},
			wantNotice:      "Behind upstream. Pull/rebase first.",
			wantCommitTip:   "Worktree is clean. Make changes before committing.",
			wantPushTip:     "Branch is behind upstream. Pull/rebase before pushing.",
			wantPRTip:       "Branch is behind upstream. Pull/rebase before creating a pull request.",
			wantPublishTip:  "Publish repository is unavailable.",
			commitBtnLabel:  "Commit…",
			pushBtnLabel:    "Push…",
			prBtnLabel:      "Review draft PR…",
			publishBtnLabel: "Publish repository…",
		},
		{
			locale:          "zh",
			state:           "behind-clean",
			status:          git.RepositoryStatus{IsRepo: true, Branch: "feature/behind", HasOrigin: true, HasUpstream: true, Behind: 2},
			wantNotice:      "落后于上游，请先拉取或变基。",
			wantCommitTip:   "工作区干净，请先修改文件后再提交。",
			wantPushTip:     "分支落后于上游，请先拉取或变基后再推送。",
			wantPRTip:       "分支落后于上游，请先拉取或变基后再创建拉取请求。",
			wantPublishTip:  "发布仓库不可用。",
			commitBtnLabel:  "提交…",
			pushBtnLabel:    "推送…",
			prBtnLabel:      "核对草稿 PR…",
			publishBtnLabel: "发布仓库…",
		},
		{
			locale:          "en",
			state:           "dirty",
			status:          git.RepositoryStatus{IsRepo: true, Branch: "feature/dirty", Dirty: true, HasOrigin: true, HasUpstream: true, Ahead: 1, AheadOfDefault: 1},
			changes:         []git.Change{{Path: "README.md", Worktree: 'M'}},
			wantCommitTip:   "",
			wantPushTip:     "Commit or stash local changes before pushing.",
			wantPRTip:       "Commit local changes before creating a pull request.",
			wantPublishTip:  "Publish repository is unavailable.",
			commitBtnLabel:  "Commit…",
			pushBtnLabel:    "Push…",
			prBtnLabel:      "Review draft PR…",
			publishBtnLabel: "Publish repository…",
		},
		{
			locale:          "zh",
			state:           "dirty",
			status:          git.RepositoryStatus{IsRepo: true, Branch: "feature/dirty", Dirty: true, HasOrigin: true, HasUpstream: true, Ahead: 1, AheadOfDefault: 1},
			changes:         []git.Change{{Path: "README.md", Worktree: 'M'}},
			wantCommitTip:   "",
			wantPushTip:     "推送前请先提交或暂存本地更改。",
			wantPRTip:       "创建拉取请求前请先提交本地更改。",
			wantPublishTip:  "发布仓库不可用。",
			commitBtnLabel:  "提交…",
			pushBtnLabel:    "推送…",
			prBtnLabel:      "核对草稿 PR…",
			publishBtnLabel: "发布仓库…",
		},
	} {
		t.Run(tc.locale+"-"+tc.state, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			status := tc.status
			w := &Workbench{
				Root:                "fixture",
				Loaded:              true,
				Repository:          true,
				RepositoryToolsOpen: true,
				RepositoryStatus:    &status,
				Changes:             tc.changes,
				AgainstHEAD:         true,
			}
			if got := w.statusNotice(); got != tc.wantNotice {
				t.Fatalf("statusNotice() = %q, want %q", got, tc.wantNotice)
			}
			if got := w.commitDisabledReason(); got != tc.wantCommitTip {
				t.Fatalf("commitDisabledReason() = %q, want %q", got, tc.wantCommitTip)
			}
			if got := w.pushDisabledReason(); got != tc.wantPushTip {
				t.Fatalf("pushDisabledReason() = %q, want %q", got, tc.wantPushTip)
			}
			if got := w.createPRDisabledReason(); got != tc.wantPRTip {
				t.Fatalf("createPRDisabledReason() = %q, want %q", got, tc.wantPRTip)
			}
			if got := w.publishDisabledReason(); got != tc.wantPublishTip {
				t.Fatalf("publishDisabledReason() = %q, want %q", got, tc.wantPublishTip)
			}

			m := Model{Workbench: w, WorkspaceRoot: "fixture"}
			tt := ui.NewTester(func(c *ui.Context) {
				paintWorkbench(c, m, c.Theme().Text, c.Theme().TextMuted)
			}, 900, 600)
			now := time.Now()
			tt.SetTime(now)

			if tc.detachedHeadLabel != "" && !tt.HasText(tc.detachedHeadLabel) {
				t.Fatalf("missing detached head status label %q", tc.detachedHeadLabel)
			}
			if tc.wantNotice != "" && !tt.HasText(tc.wantNotice) {
				t.Fatalf("missing status notice %q", tc.wantNotice)
			}

			hoverCheck := func(label, wantTip string) {
				t.Helper()
				btn, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing button %q in %v", label, tt.Texts())
				}
				tt.Move(btn.X+btn.W/2, btn.Y+btn.H/2)
				now = now.Add(time.Second)
				tt.SetTime(now)
				if !tt.HasText(wantTip) {
					t.Fatalf("missing tooltip %q for %q in %v", wantTip, label, tt.Texts())
				}
			}

			hoverCheck(tc.pushBtnLabel, tc.wantPushTip)
			hoverCheck(tc.prBtnLabel, tc.wantPRTip)
			if tc.wantCommitTip != "" {
				hoverCheck(tc.commitBtnLabel, tc.wantCommitTip)
			}
		})
	}
}

func TestBranchMenuUnsupportedNameTooltipAndDiscardCopy(t *testing.T) {
	defer i18n.SetLocale(i18n.DefaultLocale)

	for _, tc := range []struct {
		locale          string
		wantUnsupported string
		wantCreateNote  string
		wantDiscardHead string
		wantDiscardBody string
	}{
		{
			locale:          "en",
			wantUnsupported: "Name contains characters the desktop cannot pass to git safely.",
			wantCreateNote:  "This version only creates and switches from the current HEAD.",
			wantDiscardHead: "Discard working-tree changes?",
			wantDiscardBody: "Unstaged changes will be lost and cannot be undone.",
		},
		{
			locale:          "zh",
			wantUnsupported: "名称包含桌面无法安全传给 git 的字符，无法从这里切换。",
			wantCreateNote:  "首版只支持基于当前 HEAD 创建并切换。",
			wantDiscardHead: "还原工作区更改？",
			wantDiscardBody: "未暂存的更改将被丢弃，无法恢复。",
		},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			i18n.SetLocale(tc.locale)
			w := &Workbench{
				Root:         "fixture",
				BranchesOpen: true,
			}
			open := true
			m := Model{Workbench: w, WorkspaceRoot: "fixture"}
			tt := ui.NewTester(func(c *ui.Context) {
				anchor := ui.Button(c, "BranchTrigger")
				CaptionBranches(c, m, anchor, &open)
				paintBranches(c, m, c.Theme().Text, c.Theme().TextMuted)
				paintDiscard(c, m, c.Theme().Text, c.Theme().TextMuted)
			}, 900, 700)
			w.cancelBranchList()
			w.BranchStatus = ""
			w.Branches = []git.Branch{
				{Name: "main", Current: true},
				{Name: "bad..branch", Unsupported: true},
			}
			now := time.Now()
			tt.SetTime(now)
			tt.Frame()

			if !tt.HasText(tc.wantCreateNote) {
				t.Fatalf("missing create note %q", tc.wantCreateNote)
			}
			row, ok := tt.Find("bad..branch")
			if !ok {
				t.Fatalf("missing unsupported branch row in %v", tt.Texts())
			}
			tt.Move(row.X+row.W/2, row.Y+row.H/2)
			tt.SetTime(now.Add(time.Second))
			if !tt.HasText(tc.wantUnsupported) {
				t.Fatalf("missing unsupported branch tooltip %q in %v", tc.wantUnsupported, tt.Texts())
			}

			open = false
			w.Discard = &git.DiscardPlan{Path: "README.md"}
			tt.Frame()
			if !tt.HasText(tc.wantDiscardHead) || !tt.HasText(tc.wantDiscardBody) {
				t.Fatalf("missing discard dialog copy (%q / %q) in %v", tc.wantDiscardHead, tc.wantDiscardBody, tt.Texts())
			}
		})
	}
}

func TestGitAndDiffLocalesParity(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(prev) })

	i18n.SetLocale("zh")
	if got := GitLocale("commit.description"); got != "确认本次提交内容。提交信息留空将自动生成。" {
		t.Fatalf("zh commit.description = %q", got)
	}
	if got := formatSelectedCount(2, 5); got != "2/5" {
		t.Fatalf("zh formatSelectedCount = %q", got)
	}
	if got := DiffLocaleFormat("file.truncated", map[string]string{"count": "30"}); got != "仅显示前 30 行" {
		t.Fatalf("zh file.truncated = %q", got)
	}

	i18n.SetLocale("en")
	if got := GitLocale("commit.description"); got != "Review and confirm your commit. Leave the message blank to auto-generate one." {
		t.Fatalf("en commit.description = %q", got)
	}
	if got := formatSelectedCount(2, 5); got != "2 of 5" {
		t.Fatalf("en formatSelectedCount = %q", got)
	}
	if got := DiffLocaleFormat("file.truncated", map[string]string{"count": "30"}); got != "Showing the first 30 lines" {
		t.Fatalf("en file.truncated = %q", got)
	}
}


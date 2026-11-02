package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	desktopgit "whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/diffsurface"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
)

// Exercise the complete frame, including the active pane's legacy projection.
// Painting the caption alone cannot catch a modal lost during frame writeback.
func TestWorkspaceCaptionReviewSurvivesFrameProjection(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, action := range []struct {
		name, button, control string
		dirty, origin, ahead  bool
	}{
		{"commit", "Commit", "Review selected files to commit…", true, false, false},
		{"commit-push", "Commit & push", "Review commit and push…", true, true, false},
		{"push", "Push", "Prepare review", false, true, true},
		{"publish", "Publish repository", "Review publish target", false, false, false},
	} {
		t.Run(action.name, func(t *testing.T) {
			root := captionEmptyRepository(t, action.dirty)
			for _, global := range []bool{false, true} {
				for _, dock := range []struct {
					name, tab   string
					open, split bool
				}{
					{"closed", "files", false, false},
					{"diff", "diff", true, false},
					{"split-diff", "diff", true, true},
				} {
					t.Run(fmt.Sprintf("global-%t/%s", global, dock.name), func(t *testing.T) {
						store, transport := captionSelectedStore(t, root)
						status := &desktopgit.RepositoryStatus{IsRepo: true, Branch: "main", Dirty: action.dirty, HasOrigin: action.origin, HasUpstream: action.origin, IsDefault: true}
						if action.ahead {
							status.Ahead = 1
						}
						state := &mainState{filesPaneOpen: dock.open, rightPaneTab: dock.tab, gitRepositoryStatus: status, gitCheckedRoot: root, gitCheckedAt: time.Now()}
						workbench := &diffsurface.Workbench{}
						defer workbench.ClosePane()
						sizes := frame.DefaultLayout()
						sizes.Bottom = 0
						model := frame.Model{Store: store, Sizes: &sizes, WorkspaceRoot: root, GitWorkbench: workbench, WorkspacePage: global, RightPaneOpen: dock.open, RightPaneTab: &state.rightPaneTab}
						state.bindWorkspaceTrailing(nil, &model, nil, status, false, root, "explicit fixture message")
						tt := ui.NewTester(func(c *ui.Context) {
							ui.Column(c).Fill().Children(func() {
								if global {
									frame.NavigationShell(c, model, func() { ui.Text(c, "Retained global page") })
								} else {
									frame.View(c, model)
								}
							})
						}, 1600, 900)
						var other *diffsurface.Workbench
						if dock.split && !global {
							frame.SplitWorkspacePane(model, false)
							tt.Frame()
							other = &sizes.Workspace.Root.First.Diff.Workbench
							other.Message = "other pane draft"
						}
						active := ""
						if sizes.Workspace != nil {
							active = sizes.Workspace.Active
						}
						assertOwner := func() {
							t.Helper()
							if other != nil && (other.Message != "other pane draft" || other.CaptionReviewOpen() || other.Remote != nil || other.PublishOpen || other.Commit != nil && other.Commit.CaptionAction != "") {
								t.Fatal("caption review changed the other pane's private draft or review owner")
							}
							if state.filesPaneOpen != dock.open || state.rightPaneTab != dock.tab || model.RightPaneOpen != dock.open || model.GitWorkbench != workbench || sizes.Workspace != nil && sizes.Workspace.Active != active {
								t.Fatal("caption review changed the current page, dock or pane owner")
							}
							if workbench.Operation != nil || workbench.Progress != nil || workbench.Commit != nil && (workbench.Commit.Task != nil || workbench.Commit.Plan != nil || workbench.Commit.StackPlan != nil) || workbench.Remote != nil && (workbench.Remote.Task != nil || workbench.Remote.Generation != nil || workbench.Remote.Plan != nil) || workbench.PublishReview || transport.calls.Load() != 1 {
								t.Fatal("opening/closing the caption review executed a Git operation, prepared a plan or contacted the Host")
							}
						}
						if err := tt.Click(action.button); err != nil {
							t.Fatal(err)
						}
						for n := 0; n < 3; n++ {
							tt.Frame()
							if !workbench.CaptionReviewOpen() || !tt.HasText(action.control) || workbench.Root != root {
								t.Fatalf("frame %d lost the titlebar review: open=%t texts=%q", n, workbench.CaptionReviewOpen(), tt.Texts())
							}
							assertOwner()
						}
						if action.dirty && (!tt.HasText("untracked.txt") || workbench.Message != "explicit fixture message") {
							t.Fatal("caption review lost its selected file or message draft")
						}
						if err := tt.Click("Close"); err != nil {
							t.Fatal(err)
						}
						tt.Frame()
						if workbench.CaptionReviewOpen() || tt.HasText(action.control) {
							t.Fatal("closed review returned from stale pane state")
						}
						assertOwner()
					})
				}
			}
		})
	}
}

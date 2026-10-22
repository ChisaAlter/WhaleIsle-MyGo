package diffsurface

import (
	"context"
	"strings"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/chrome"
	fonttheme "whaleisle/internal/ui/theme"
)

// This is a presentation seat for the one window Workbench, not a second
// selection, review plan, mutation owner or repository snapshot.
type captionReviewState struct {
	root, action, kind string
	valid              func() bool
}

func captionAdmitted(m Model) bool { return m.CaptionAdmission == nil || m.CaptionAdmission() }

// CaptionReviewOpen lets the root keep one progress producer while inline
// workbench controls are suppressed by the shared review modal.
func (w *Workbench) CaptionReviewOpen() bool { return w != nil && w.captionReview != nil }

// CaptionAction is called only by the captured titlebar click. Initialize
// directly starts the existing operation, matching the pinned runInit action.
// Other actions open the existing review; overlay=true presents that same
// Workbench above the retained workspace without changing its panel or dock.
func CaptionAction(c *ui.Context, m Model, message, action string, overlay ...bool) bool {
	w := m.Workbench
	if w == nil || c == nil || w.Operation != nil || strings.TrimSpace(m.WorkspaceRoot) == "" || m.CaptionAdmission != nil && !m.CaptionAdmission() {
		return false
	}
	modal := len(overlay) > 0 && overlay[0]
	if modal && action != "init" && m.CaptionAdmission == nil {
		return false
	}
	if w.captionReview != nil {
		closeCaptionReview(w, w.captionReview)
	}
	w.invalidate = c.Services().Invalidate
	if action == "init" {
		root := m.WorkspaceRoot
		w.cancelCommitPreparation()
		w.cancelRemotePreparation()
		w.Remote = nil
		w.PublishOpen = false
		if w.Root != root {
			w.cancelBranchList()
			w.cancelRepositoryStatus()
			w.CancelComparison()
			*w = Workbench{Root: root, invalidate: c.Services().Invalidate, AgainstHEAD: true}
		}
		w.Root = root
		w.invalidate = c.Services().Invalidate
		beginGitOperation(c, w, word("正在初始化 Git", "Initializing Git"), func(ctx context.Context) gitOperationResult {
			out, err := git.Initialize(root)
			return gitOperationResult{text: out, err: err}
		})
		return true
	}
	if !w.RequestCaptionAction(m.WorkspaceRoot, message, action) {
		return false
	}
	if modal {
		kind := "commit"
		if w.Remote != nil {
			kind = "remote"
		} else if w.PublishOpen {
			kind = "publish"
		}
		w.captionReview = &captionReviewState{root: m.WorkspaceRoot, action: action, kind: kind, valid: m.CaptionAdmission}
	}
	return true
}

func closeCaptionReview(w *Workbench, seat *captionReviewState) {
	if w.captionReview != seat {
		return
	}
	w.captionReview = nil
	// Never revoke a replacement repository's preparation or an in-flight
	// mutation. Only the modal's own still-current review is being dismissed.
	if w.Root != seat.root {
		return
	}
	w.cancelCommitPreparation()
	w.cancelRemotePreparation()
	w.Remote = nil
	w.PublishOpen = false
	w.PublishReview = false
	if w.Commit != nil {
		w.Commit.CaptionAction = ""
	}
}

// CaptionReview remains in the root overlay layer. All paths and authority are
// captured at admission; a new Host, retained Session or cwd retires this seat.
// Merely painting this modal never prepares a message or executes Git.
func CaptionReview(c *ui.Context, m Model) {
	colors := fonttheme.Resolve(m.ThemeID).Active(m.SystemDark)
	w := m.Workbench
	if w == nil || w.captionReview == nil {
		return
	}
	seat := w.captionReview
	if seat.valid == nil || !seat.valid() || seat.root != m.WorkspaceRoot || w.Root != seat.root {
		closeCaptionReview(w, seat)
		return
	}
	owned := m
	owned.WorkspaceRoot = seat.root
	owned.CaptionAdmission = seat.valid
	open := true
	width, height := c.Size()
	title := word("核对 Git 操作", "Review Git action")
	switch seat.kind {
	case "commit":
		title = word("提交更改", "Commit changes")
	case "publish":
		title = word("发布仓库", "Publish repository")
	}
	ui.DialogBase(c.Key("caption-git-review"), &open, func(backdrop, panel ui.Element) {
		chrome.DialogSurfaceTheme(c, backdrop, panel, c.Theme(), max(0, min(float32(680), width-32)), 20).
			MaxHeight(max(0, height-48)).Gap(16).Label(title).Children(func() {
			ui.Row(c).FillWidth().Gap(8).Children(func() {
				ui.Text(c, title).Grow(1).MinWidth(0).FontSize(fonttheme.InterfaceSize(16)).FontWeight(500)
				if widgets.Button(c, colors, word("关闭", "Close"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					open = false
				}
			})
			ui.Text(c, seat.root).FillWidth().Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(c.Theme().TextMuted)
			ui.Scroll(c).FillWidth().MinHeight(0).MaxHeight(max(0, height-190)).Children(func() {
				ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
					// Recheck immediately before presenting the shared action controls.
					if !seat.valid() {
						open = false
						return
					}
					fg, muted := c.Theme().Text, c.Theme().TextMuted
					switch seat.kind {
					case "publish":
						paintPublish(c, owned, fg, muted)
					case "remote":
						paintRemoteWorkflow(c, owned, fg, muted)
					default:
						if paintCommitPreparation(c, owned, fg, muted) {
							return
						}
						branch := word("detached HEAD", "detached HEAD")
						if w.RepositoryStatus != nil && w.RepositoryStatus.Branch != "" {
							branch = w.RepositoryStatus.Branch
						}
						ui.Text(c, word("提交分支：", "Commit branch: ")+branch).FontSize(fonttheme.InterfaceSize(12)).TextColor(muted)
						for _, change := range w.Changes {
							ch := change
							ui.Row(c.Key("caption-commit-file-" + ch.Path)).FillWidth().MinWidth(0).Gap(8).Children(func() {
								paintCommitCheckbox(c, w, ch.Path)
								ui.Text(c, ch.Path).Grow(1).MinWidth(0).Selectable().FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
							})
						}
						paintCommitSelection(c, owned, fg, muted)
						if w.ReviewText != "" {
							ui.ScrollBoth(c).FillWidth().MinWidth(0).Height(min(float32(260), max(float32(80), height-420))).Children(func() {
								ui.Text(c, w.ReviewText).NoWrap().Selectable().Font("monospace").FontSize(fonttheme.InterfaceSize(12)).TextColor(fg)
							})
						}
						if w.Status != "" {
							ui.Text(c, w.Status).FillWidth().Role(ui.RoleStatus).TextColor(fg)
						}
					}
				})
			})
		})
	})
	if w.Operation != nil || seat.kind == "remote" && w.Remote == nil || seat.kind == "publish" && !w.PublishOpen {
		open = false
	}
	if !open {
		closeCaptionReview(w, seat)
	}
}

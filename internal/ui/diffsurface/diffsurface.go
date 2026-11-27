// Package diffsurface is the M7 Git diff work surface in the right pane.
package diffsurface

import (
	"errors"
	"strings"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets/diff"
)

// Model is diff panel state for one frame.
type Model struct {
	// CaptionAdmission captures the window/Host/retained-Session owner for a
	// titlebar gesture. Global-page review modals retain this exact callback.
	CaptionAdmission func() bool
	OnOpenFile       func(path string) error
	Workbench        *Workbench
	WorkspaceRoot    string
	ThemeID          string
	SystemDark       bool

	PathFilter *string // optional repo-relative path; empty means whole tree
	Text       *string
	ErrMsg     *string
}

// LoadDiff runs git diff for root, optionally limited to relPath.
func LoadDiff(root, relPath string) (text, errMsg string) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", i18n.T(i18n.KeyDiffNoWorkspace)
	}
	var paths []string
	if strings.TrimSpace(relPath) != "" {
		paths = []string{relPath}
	}
	_ = paths
	out, err := git.DiffHEAD(root, relPath)
	if err != nil {
		if errors.Is(err, git.ErrNotRepo) {
			return "", i18n.T(i18n.KeyDiffNotRepo)
		}
		return "", errcopy.Sentence(err)
	}
	if strings.TrimSpace(out) == "" {
		return "", i18n.T(i18n.KeyDiffEmpty)
	}
	return out, ""
}

func refresh(m Model) {
	refreshWithForce(m, false)
}

func refreshWithForce(m Model, force bool) {
	rel := ""
	if m.PathFilter != nil {
		rel = *m.PathFilter
	}
	if m.Workbench != nil && m.Workbench.BaseRef != "" {
		beginComparisonForce(m, rel, force)
		return
	}
	if m.Workbench != nil {
		m.Workbench.CancelComparison()
		m.Workbench.RequestDiffAsync(m.WorkspaceRoot, rel, force, m.Workbench.invalidate)
		text, errMsg, ready, _ := m.Workbench.PollDiffAsync(m.WorkspaceRoot, rel)
		if ready {
			if m.Text != nil {
				*m.Text = text
			}
			if m.ErrMsg != nil {
				*m.ErrMsg = errMsg
			}
		} else if force {
			if m.Text != nil {
				*m.Text = ""
			}
			if m.ErrMsg != nil {
				*m.ErrMsg = ""
			}
		}
		return
	}
	if force {
		if m.Text != nil {
			*m.Text = ""
		}
		if m.ErrMsg != nil {
			*m.ErrMsg = ""
		}
	}
	defaultDiffCache.ensure(m.WorkspaceRoot, rel, force, m.Text, m.ErrMsg, nil)
}

// View paints scoped or full-workspace git diff output.
func View(c *ui.Context, m Model) {
	th := theme.Resolve(m.ThemeID)
	colors := th.Active(m.SystemDark)
	fg := chrome.Color(colors.FG)
	muted := chrome.Color(colors.Muted)
	field := chrome.Color(colors.Field)
	border := chrome.Color(colors.Border)
	accent := chrome.Color(colors.Accent)

	paneBg := field
	if design.For(c).Glass || chrome.BackdropActive() {
		paneBg = ui.Transparent
	}
	ui.Column(c).Fill().MinWidth(0).Background(paneBg).Label(word("工作区差异", "Diff panel")).Children(func() {
		ui.Row(c).MinHeight(36).Padding(8, 12, 8, 12).Gap(8).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyDiffTitle)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg).Grow(1)
			if strings.TrimSpace(m.WorkspaceRoot) != "" && m.Text != nil && strings.TrimSpace(*m.Text) != "" {
				if widgets.Button(c, colors, i18n.T(i18n.KeyChatCopy), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					mygo.Clipboard.WriteText(*m.Text)
				}
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeyDiffRefresh), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Disabled(strings.TrimSpace(m.WorkspaceRoot) == "").Clicked() {
				refreshWorkbench(m)
			}
		})
		// An unselected workspace is not the process working directory. Do not
		// probe Git or expose repository mutations before this guard.
		if strings.TrimSpace(m.WorkspaceRoot) == "" {
			ui.Text(c, i18n.T(i18n.KeyDiffNoWorkspace)).FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(12)
			return
		}
		paintWorkbench(c, m, fg, muted)
		paintDiffControls(c, m)
		scope := ""
		if m.PathFilter != nil {
			scope = strings.TrimSpace(*m.PathFilter)
		}
		if scope != "" {
			ui.Text(c, scope).FontSize(theme.InterfaceSize(11)).TextColor(muted).Padding(0, 12, 4, 12)
		}
		ui.Box(c).FillWidth().Height(0.5).Background(border)

		errMsg := ""
		if m.ErrMsg != nil {
			errMsg = *m.ErrMsg
		}
		text := ""
		if m.Text != nil {
			text = *m.Text
		}
		if m.Workbench != nil && m.Workbench.CommitOpen && m.Workbench.Commit != nil && m.Workbench.Commit.Plan != nil {
			text = m.Workbench.Commit.Plan.Patch
			errMsg = ""
		}
		if errMsg != "" {
			ui.Text(c, errMsg).FontSize(theme.InterfaceSize(12)).TextColor(accent).Padding(12, 12, 12, 12)
			return
		}
		if strings.TrimSpace(text) == "" {
			loading := m.Workbench != nil && (m.Workbench.IsDiffLoading(m.WorkspaceRoot, scope) ||
				(!m.Workbench.Loaded && m.Workbench.isReloading(m.WorkspaceRoot)) ||
				(m.Workbench.BaseRef != "" && m.Workbench.comparison != nil))
			hint := i18n.T(i18n.KeyDiffHint)
			if loading {
				hint = word("正在加载差异…", "Loading diff…")
			}
			ui.Text(c, hint).FontSize(theme.InterfaceSize(12)).TextColor(muted).
				Padding(12, 12, 12, 12)
			return
		}

		tokens := design.For(c)
		paintPatchFiles(c, m, text, diff.Style{
			FontSize: 11,
			FG:       fg,
			Add:      tokens.SuccessForeground,
			Del:      tokens.DestructiveForeground,
			Hunk:     accent,
			File:     muted,
			Muted:    muted,
		})
	})
}

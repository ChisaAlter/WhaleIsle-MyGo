// Package diffsurface is the M7 Git diff work surface in the right pane.
package diffsurface

import (
	"errors"
	"strings"
	"whaleisle/internal/ui/widgets"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/git"
	"whaleisle/internal/ui/chrome"
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
		return "", strings.TrimSpace(err.Error())
	}
	if strings.TrimSpace(out) == "" {
		return "", i18n.T(i18n.KeyDiffEmpty)
	}
	return out, ""
}

func refresh(m Model) {
	rel := ""
	if m.PathFilter != nil {
		rel = *m.PathFilter
	}
	if m.Workbench != nil && m.Workbench.BaseRef != "" {
		beginComparison(m, rel)
		return
	}
	if m.Workbench != nil {
		m.Workbench.CancelComparison()
	}
	var text, errMsg string
	if m.Workbench != nil {
		var err error
		if m.Workbench.AgainstHEAD {
			text, err = git.DiffHEAD(m.WorkspaceRoot, rel)
		} else {
			text, err = git.DiffScope(m.WorkspaceRoot, rel, m.Workbench.Staged)
		}
		errMsg = ""
		if err != nil {
			errMsg = err.Error()
		} else if text == "" {
			errMsg = i18n.T(i18n.KeyDiffEmpty)
		}
	} else {
		text, errMsg = LoadDiff(m.WorkspaceRoot, rel)
	}
	if m.Text != nil {
		*m.Text = text
	}
	if m.ErrMsg != nil {
		*m.ErrMsg = errMsg
	}
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

	ui.Column(c).Fill().MinWidth(0).Background(field).Children(func() {
		ui.Row(c).MinHeight(36).Padding(8, 12, 8, 12).Gap(8).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, i18n.T(i18n.KeyDiffTitle)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg).Grow(1)
			if m.Text != nil && strings.TrimSpace(*m.Text) != "" {
				if widgets.Button(c, colors, i18n.T(i18n.KeyChatCopy), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
					mygo.Clipboard.WriteText(*m.Text)
				}
			}
			if widgets.Button(c, colors, i18n.T(i18n.KeyDiffRefresh), widgets.ButtonSpec{Variant: widgets.ButtonToolbar, Size: widgets.ButtonSmall}).Clicked() {
				refreshWorkbench(m)
			}
		})
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

		root := strings.TrimSpace(m.WorkspaceRoot)
		if root == "" {
			ui.Text(c, i18n.T(i18n.KeyDiffNoWorkspace)).FontSize(theme.InterfaceSize(12)).TextColor(muted).
				Padding(12, 12, 12, 12)
			return
		}

		errMsg := ""
		if m.ErrMsg != nil {
			errMsg = *m.ErrMsg
		}
		text := ""
		if m.Text != nil {
			text = *m.Text
		}
		if m.Workbench != nil && m.Workbench.Commit != nil && m.Workbench.Commit.Plan != nil {
			text = m.Workbench.Commit.Plan.Patch
			errMsg = ""
		}
		if errMsg != "" {
			ui.Text(c, errMsg).FontSize(theme.InterfaceSize(12)).TextColor(accent).Padding(12, 12, 12, 12)
			return
		}
		if strings.TrimSpace(text) == "" {
			ui.Text(c, i18n.T(i18n.KeyDiffHint)).FontSize(theme.InterfaceSize(12)).TextColor(muted).
				Padding(12, 12, 12, 12)
			return
		}

		ui.Scroll(c).Fill().MinHeight(0).Padding(8, 8, 8, 8).Children(func() {
			paintPatchFiles(c, m, text, diff.Style{
				FontSize: 12,
				FG:       fg,
				Add:      ui.RGB(0x22, 0xc5, 0x5e),
				Del:      ui.RGB(0xef, 0x44, 0x44),
				Hunk:     accent,
				File:     muted,
				Muted:    muted,
			})
		})
	})
}

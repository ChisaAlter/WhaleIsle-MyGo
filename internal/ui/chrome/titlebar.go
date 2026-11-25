// Package chrome holds shared native window chrome for MyGo鲸屿:
// the custom title bar drawn under MyGo's TitleBarHidden controls.
package chrome

import (
	"fmt"
	"strings"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TitleBarColors are the fills used by the custom title bar.
type TitleBarColors struct {
	Background ui.Color
	Foreground ui.Color
	Muted      ui.Color
	Border     ui.Color
}

// DefaultTitleBarColors is a DeepSeek-ish light palette used until the
// theme package resolves a named theme.
func DefaultTitleBarColors() TitleBarColors {
	return TitleBarColors{
		Background: ui.Hex("#f5f5f7"),
		Foreground: ui.Hex("#1d1d1f"),
		Muted:      ui.Hex("#6e6e73"),
		Border:     ui.Hex("#d2d2d7"),
	}
}

// TitleBarHeight is the caption strip reserved for system traffic lights.
// Electron desk lets the left rail run under the caption; keep this thin so
// the rail brand sits near the top edge like the live MyGo鲸屿 window.
const TitleBarHeight = 12

// EffectiveTitleBarHeight is the caption strip height for the active mode.
func EffectiveTitleBarHeight() float32 {
	return TitleBarHeight
}

// TitleBarSafeHorizontalPadding adds native window-control insets (macOS
// traffic lights via bar.Left, Windows - □ × buttons via bar.Right) onto base
// horizontal padding so top strips and route headers never overlap OS controls.
func TitleBarSafeHorizontalPadding(bar ui.TitleBar, base float32) (left, right float32) {
	left, right = base, base
	if bar.Left > 0 {
		left += bar.Left
	}
	if bar.Right > 0 {
		right += bar.Right
	}
	return left, right
}

// GitBadge is optional branch / dirty state shown beside the product title.
// When OnPull / OnPush / OnFetch / OnCommit / OnPR* are set, compact action buttons appear next to the badge.
type GitBadge struct {
	Branch        string
	Dirty         bool
	Ahead, Behind int
	FetchLabel    string
	PullLabel     string
	PushLabel     string
	CommitLabel   string
	PRViewLabel   string
	PRCreateLabel string
	PRPanelLabel  string
	PRSummary     string
	StatusText    string
	OnFetch       func()
	OnPull        func()
	OnPush        func()
	OnCommit      func()
	OnPRView      func()
	OnPRCreate    func()
	OnPRPanel     func()
}

// TitleBar draws a draggable title strip with the product name. Call it at
// the top of the main window view when TitleBarStyle is TitleBarHidden.
// Window controls sit on the right (Windows); leave room via c.TitleBar().
func TitleBar(c *ui.Context, title string, colors TitleBarColors) {
	TitleBarWithActions(c, title, colors, nil, nil)
}

// TitleBarWithActions is TitleBar plus optional trailing controls before the
// system caption buttons (for example a settings gear).
func TitleBarWithActions(c *ui.Context, title string, colors TitleBarColors, git *GitBadge, trailing func()) {
	TitleBarWithLeadingActions(c, title, colors, git, nil, trailing)
}

// TitleBarWithLeadingActions keeps primary navigation outside collapsed tracks.
// Leading controls share the native caption safe areas with trailing controls.
func TitleBarWithLeadingActions(c *ui.Context, title string, colors TitleBarColors, git *GitBadge, leading, trailing func()) {
	bar := c.TitleBar()
	h := EffectiveTitleBarHeight()
	if h > 0 && bar.Height > 0 {
		h = bar.Height
	}
	if leading != nil && h < 32 {
		h = 32 // Keep the navigation hit target usable even without native insets.
	}
	// Fill() sets width and height to 100%; apply Height after Fill so the
	// bar stays TitleBarHeight instead of swallowing the whole window.
	ui.Row(c).Fill().Height(h).Background(colors.Background).
		Padding(0, 12, 0, 12).AlignItems(ui.Center).Gap(8).
		DragWindow().
		Children(func() {
			// Leave space for traffic lights / caption buttons.
			if bar.Left > 0 {
				ui.Box(c).Width(bar.Left)
			}
			if leading != nil {
				leading()
			}
			ui.Text(c, title).FontSize(theme.InterfaceSize(13)).Bold().TextColor(colors.Foreground)
			if git != nil && git.Branch != "" {
				label := git.Branch
				if git.Dirty {
					label += " •"
				}
				if git.Behind > 0 {
					label += fmt.Sprintf(" ↓%d", git.Behind)
				}
				if git.Ahead > 0 {
					label += fmt.Sprintf(" ↑%d", git.Ahead)
				}
				muted := colors.Muted
				if muted.A == 0 && muted.R == 0 && muted.G == 0 && muted.B == 0 {
					muted = colors.Foreground
				}
				branchBtn := coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonXS,
					Label:   label,
					Content: func() {
						ui.Text(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					},
				}).Padding(0, design.Sp(1), 0, design.Sp(1))
				if branchBtn.Clicked() && strings.TrimSpace(git.Branch) != "" {
					mygo.Clipboard.WriteText(git.Branch)
				}
				if git.PRSummary != "" {
					ui.Text(c, git.PRSummary).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
				if git.OnFetch != nil && git.FetchLabel != "" {
					if coss.Button(c, git.FetchLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Tooltip(git.FetchLabel).Clicked() {
						git.OnFetch()
					}
				}
				if git.OnPull != nil && git.PullLabel != "" {
					if coss.Button(c, git.PullLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Tooltip(git.PullLabel).Clicked() {
						git.OnPull()
					}
				}
				if git.OnPush != nil && git.PushLabel != "" {
					if coss.Button(c, git.PushLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Tooltip(git.PushLabel).Clicked() {
						git.OnPush()
					}
				}
				if git.OnCommit != nil && git.CommitLabel != "" {
					if coss.Button(c, git.CommitLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Tooltip(git.CommitLabel).Clicked() {
						git.OnCommit()
					}
				}
				if git.OnPRPanel != nil && git.PRPanelLabel != "" {
					if coss.Button(c, git.PRPanelLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Tooltip(git.PRPanelLabel).Clicked() {
						git.OnPRPanel()
					}
				}
				if git.OnPRView != nil && git.PRViewLabel != "" {
					if coss.Button(c, git.PRViewLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Tooltip(git.PRViewLabel).Clicked() {
						git.OnPRView()
					}
				}
				if git.OnPRCreate != nil && git.PRCreateLabel != "" {
					if coss.Button(c, git.PRCreateLabel, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonXS}).Clicked() {
						git.OnPRCreate()
					}
				}
				if git.StatusText != "" {
					ui.Text(c, git.StatusText).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
			}
			ui.Box(c).Grow(1) // push remaining space; controls sit on the right
			if trailing != nil {
				trailing()
			}
			if bar.Right > 0 {
				ui.Box(c).Width(bar.Right)
			}
		})
	// Hairline under the bar (Height after Fill — Fill alone is 100% tall).
	ui.Box(c).Fill().Height(1).Background(colors.Border)
}

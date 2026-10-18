// Package chrome holds shared native window chrome for Whale Isle:
// the custom title bar drawn under MyGo's TitleBarHidden controls.
package chrome

import (
	"strings"
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
// the rail brand sits near the top edge like the live Whale Isle window.
const TitleBarHeight = 12

// EffectiveTitleBarHeight is the caption strip height for the active mode.
func EffectiveTitleBarHeight() float32 {
	return TitleBarHeight
}

// GitBadge is optional branch / dirty state shown beside the product title.
// When OnPull / OnPush / OnFetch / OnCommit / OnPR* are set, compact action buttons appear next to the badge.
type GitBadge struct {
	Branch        string
	Dirty         bool
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
				muted := colors.Muted
				if muted.A == 0 && muted.R == 0 && muted.G == 0 && muted.B == 0 {
					muted = colors.Foreground
				}
				branchBtn := ui.Button(c, label).FontSize(theme.InterfaceSize(12)).TextColor(muted).
					BorderWidth(0).Background(ui.Transparent).Padding(0, 4, 0, 4)
				if branchBtn.Clicked() && strings.TrimSpace(git.Branch) != "" {
					mygo.Clipboard.WriteText(git.Branch)
				}
				if git.PRSummary != "" {
					ui.Text(c, git.PRSummary).FontSize(theme.InterfaceSize(11)).TextColor(muted)
				}
				if git.OnFetch != nil && git.FetchLabel != "" {
					if ui.Button(c, git.FetchLabel).Tooltip(git.FetchLabel).Clicked() {
						git.OnFetch()
					}
				}
				if git.OnPull != nil && git.PullLabel != "" {
					if ui.Button(c, git.PullLabel).Tooltip(git.PullLabel).Clicked() {
						git.OnPull()
					}
				}
				if git.OnPush != nil && git.PushLabel != "" {
					if ui.Button(c, git.PushLabel).Tooltip(git.PushLabel).Clicked() {
						git.OnPush()
					}
				}
				if git.OnCommit != nil && git.CommitLabel != "" {
					if ui.Button(c, git.CommitLabel).Tooltip(git.CommitLabel).Clicked() {
						git.OnCommit()
					}
				}
				if git.OnPRPanel != nil && git.PRPanelLabel != "" {
					if ui.Button(c, git.PRPanelLabel).Tooltip(git.PRPanelLabel).Clicked() {
						git.OnPRPanel()
					}
				}
				if git.OnPRView != nil && git.PRViewLabel != "" {
					if ui.Button(c, git.PRViewLabel).Tooltip(git.PRViewLabel).Clicked() {
						git.OnPRView()
					}
				}
				if git.OnPRCreate != nil && git.PRCreateLabel != "" {
					if ui.Button(c, git.PRCreateLabel).Clicked() {
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

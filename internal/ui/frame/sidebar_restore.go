package frame

import (
	"github.com/egoist/mygo/ui"
	"runtime"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
)

// SidebarCollapsedWidth matches AppFrame's macOS / Windows custom-caption
// contract. The main MyGo window uses TitleBarHidden on Windows; Linux keeps
// its compact navigation rail, as the original does without that caption.
func SidebarCollapsedWidth() float32 { return sidebarCollapsedWidth(runtime.GOOS) }

func sidebarCollapsedWidth(platform string) float32 {
	if platform == "darwin" || platform == "windows" {
		return 0
	}
	return SidebarCollapsed
}

// SidebarRestoreCaption is called above all main-page routes, including those
// that bypass NavigationShell. It returns whether it painted the caption.
func SidebarRestoreCaption(c *ui.Context, sizes *LayoutSizes, colors chrome.TitleBarColors, git *chrome.GitBadge, onNewSession ...func()) bool {
	return drawSidebarRestoreCaption(c, sizes, colors, git, runtime.GOOS, onNewSession...)
}

func drawSidebarRestoreCaption(c *ui.Context, sizes *LayoutSizes, colors chrome.TitleBarColors, git *chrome.GitBadge, platform string, onNewSession ...func()) bool {
	if sizes == nil || sidebarCollapsedWidth(platform) != 0 {
		return false
	}
	viewport, _ := c.Size()
	sizes.observeSidebarViewport(viewport)
	if sizes.sidebarPreference() > 0 {
		return false
	}
	chrome.TitleBarWithLeadingActions(c, "", colors, git, func() {
		label := i18n.T(i18n.KeySidebarExpand)
		if ui.Button(c, "☰").Label(label).Tooltip(label).Width(28).Height(28).
			Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(colors.Foreground).Clicked() {
			sizes.ToggleSidebar()
		}
		if len(onNewSession) > 0 && onNewSession[0] != nil {
			label := i18n.T(i18n.KeySidebarNewSessionFallback)
			if ui.Button(c, "+").Label(label).Tooltip(label).Width(28).Height(28).
				Padding(0).BorderWidth(0).Background(ui.Transparent).TextColor(colors.Foreground).Clicked() {
				onNewSession[0]()
			}
		}
	}, nil)
	return true
}

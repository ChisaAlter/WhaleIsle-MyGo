package frame

import (
	"github.com/egoist/mygo/ui"
	"runtime"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/i18n"
)

// SidebarCollapsedWidth follows the Whale preload's actual platform marks.
// Only darwin removes the rail. Whale's win32 preload does not publish the
// standalone Desktop's data-windows-titlebar flag, so Windows keeps 56 DIPs.
func SidebarCollapsedWidth() float32 { return sidebarCollapsedWidth(runtime.GOOS) }

func sidebarCollapsedWidth(platform string) float32 {
	if platform == "darwin" {
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
	viewport, _ := viewportDIP(c)
	sizes.observeSidebarViewport(viewport)
	if sizes.sidebarPreference() > 0 {
		return false
	}
	chrome.TitleBarWithLeadingActions(c, "", colors, git, func() {
		label := i18n.T(i18n.KeySidebarExpand)
		if coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonIconSM,
			Label:   label,
			Content: func() { ui.Text(c, "☰").TextColor(colors.Foreground) },
		}).Tooltip(label).Clicked() {
			sizes.ToggleSidebar()
		}
		if len(onNewSession) > 0 && onNewSession[0] != nil {
			label := i18n.T(i18n.KeySidebarNewSessionFallback)
			if coss.Button(c, "", coss.ButtonOptions{
				Variant: coss.ButtonGhost,
				Size:    coss.ButtonIconSM,
				Label:   label,
				Content: func() { ui.Text(c, "+").TextColor(colors.Foreground) },
			}).Tooltip(label).Clicked() {
				onNewSession[0]()
			}
		}
	}, nil)
	return true
}

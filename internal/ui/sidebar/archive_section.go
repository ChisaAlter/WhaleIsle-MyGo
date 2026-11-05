package sidebar

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// The management section has its own retained identities. Neither adding rows
// here nor changing the surrounding browser may reuse a workspace disclosure.
// Expansion remains local to the sidebar; it never writes workspace preferences.
func archivedSection(c *ui.Context, m Model, items []session.Item, ink, muted ui.Color) {
	ui.Column(c.Key("workspace-archive-section")).FillWidth().Gap(2).Shrink(0).Children(func() {
		label := i18n.T(i18n.KeySidebarArchivedSection)
		header := ui.ButtonBase(c.Key("workspace-archive-disclosure")).Label(label).
			Role(ui.RoleDisclosure).Expanded(*m.archiveExpanded).
			FillWidth().Height(34).Padding(0, 8).Gap(6).Radius(12).
			BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center)
		if header.Hovered() || header.FocusWithin() {
			header.Background(selectedBackground(theme.Resolve(m.ThemeID).Active(m.SystemDark)))
		}
		header.Children(func() {
			ui.Box(c.Key("archive-chevron-seat")).Width(16).Height(20).Shrink(0).Center().Children(func() {
				ui.Icon(c, workspaceChevronArtwork).Size(14, 14).TextColor(ink).
					Rotate(groupChevronAngle(c, header, !*m.archiveExpanded))
			})
			ui.Text(c, label).Label("Archive section title").FontSize(theme.InterfaceSize(12)).FontWeight(500).
				FixedLineHeight(theme.InterfaceSize(18)).TextColor(ink).Grow(1).MinWidth(0).SingleLine()
		})
		if header.Clicked() {
			*m.archiveExpanded = !*m.archiveExpanded
		}
		if !*m.archiveExpanded {
			return
		}
		if len(items) == 0 {
			ui.Text(c.Key("archive-empty"), workspaceText("暂无已归档会话", "No archived sessions")).
				FontSize(theme.InterfaceSize(12)).TextColor(muted).Padding(4, 8)
		}
		for _, item := range items {
			label := strings.TrimSpace(item.Title)
			if label == "" {
				label = i18n.T(i18n.KeySidebarNewSessionFallback)
			}
			rowArchivedSession(c, item.SessionID, label, 0, m, muted)
		}
	})
}

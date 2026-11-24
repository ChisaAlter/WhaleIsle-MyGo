package sidebar

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

const archiveSessionPageBase = 30

// The management section has its own retained identities. Neither adding rows
// here nor changing the surrounding browser may reuse a workspace disclosure.
// Expansion remains local to the sidebar; it never writes workspace preferences.
func archivedSection(c *ui.Context, m Model, items []session.Item, ink, muted ui.Color) {
	t := tok(c)
	section := ui.Column(c.Key("workspace-archive-section")).FillWidth().Gap(design.Sp(0.5)).Margin(design.Sp(0.5), 0, 0, 0).Shrink(0)
	limit := ui.Local(section, "archive-visible-limit", func() int { return archiveSessionPageBase })
	section.Transition(workspaceArchiveSectionMotion(c, section, m.Snapshot)).Children(func() {
		label := i18n.T(i18n.KeySidebarArchivedSection)
		header := coss.Button(c, "", coss.ButtonOptions{
			Variant: coss.ButtonGhost,
			Size:    coss.ButtonSM,
			Label:   workspaceText("已归档分组", "Archived group"),
			Key:     "workspace-archive-disclosure",
			Content: func() {},
		}).Role(ui.RoleDisclosure).Expanded(*m.archiveExpanded).
			FillWidth().Height(34).Padding(0, design.Sp(2)).Gap(design.Sp(1.5)).Radius(design.RadiusXL).
			BorderWidth(0).AlignItems(ui.Center)
		if header.Hovered() || header.FocusWithin() {
			header.Background(sbSelectedBg(t))
		}
		header.Children(func() {
			ui.Box(c.Key("archive-chevron-seat")).Width(16).Height(20).Shrink(0).Center().Children(func() {
				ui.Icon(c, workspaceChevronArtwork).Size(design.IconXS, design.IconXS).TextColor(ink).
					Rotate(groupChevronAngle(c, header, !*m.archiveExpanded))
			})
			ui.Text(c, label).Label(label).FontSize(fs(12)).FontWeight(design.WeightMedium).
				FixedLineHeight(fs(18)).TextColor(ink).Grow(1).MinWidth(0).SingleLine()
		})
		if header.Clicked() {
			*m.archiveExpanded = !*m.archiveExpanded
			if !*m.archiveExpanded {
				*limit = archiveSessionPageBase
			}
		}
		if !*m.archiveExpanded {
			*limit = archiveSessionPageBase
			return
		}
		if len(items) == 0 {
			ui.Text(c.Key("archive-empty"), workspaceText("暂无已归档会话", "No archived sessions")).
				FontSize(fs(12)).TextColor(muted).Padding(design.Sp(1), design.Sp(2))
			return
		}
		if *limit < archiveSessionPageBase {
			*limit = archiveSessionPageBase
		}
		visible := items
		hidden := 0
		if len(items) > *limit {
			hidden = len(items) - *limit
			visible = items[:*limit]
		}
		if len(visible) > 0 {
			capRatio := float32(0.72)
			if m.Snapshot.WorkspaceLayout.GroupBy == "flat" {
				capRatio = 0.40
			}
			listState := ui.Local(section, "workspace-archive-list-state", func() ui.ListState { return ui.ListState{} })
			list := ui.List(c.Key("workspace-archive-list"), listState, len(visible)).
				ItemKey(func(i int) any { return visible[i].SessionID }).
				FillWidth().MinWidth(0).MinHeight(0).
				MaxHeight(sidebarListMaxHeight(m, c, capRatio)).
				Gap(design.Sp(0.5)).
				ScrollbarsVisible(m.scrollbarsVisible)
			list.Rows(func(lr ui.ListRow) {
				item := visible[lr.Index]
				if sidebarVirtualRowHook != nil {
					sidebarVirtualRowHook("archive", lr.Index, item.SessionID)
				}
				rowLabel := strings.TrimSpace(item.Title)
				if rowLabel == "" {
					rowLabel = i18n.T(i18n.KeySidebarNewSessionFallback)
				}
				rowArchivedSession(lr.Context, item.SessionID, rowLabel, 0, m, muted)
			})
		}
		if hidden > 0 || (len(items) > archiveSessionPageBase && *limit > archiveSessionPageBase) {
			toggleLabel := i18n.WSf(i18n.KeyWSSessionsExpand, "n", strconv.Itoa(hidden))
			accessible := workspaceText("显示更多已归档会话", "Show more archived sessions")
			if hidden == 0 {
				toggleLabel = i18n.T(i18n.KeyWSSessionsCollapse)
				accessible = workspaceText("收起已归档会话", "Show less archived sessions")
			}
			ui.Box(c).FillWidth().Shrink(0).Children(func() {
				if sidebarPaginationToggle(c, "workspace-archive-overflow", toggleLabel, accessible, muted) {
					if hidden == 0 {
						*limit = archiveSessionPageBase
					} else {
						*limit += archiveSessionPageBase
					}
				}
			})
		}
	})
}

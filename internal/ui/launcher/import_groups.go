package launcher

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func importSessionGroups(c *ui.Context, pal launcherPalette, items []importer.CatalogItem) {
	var order []string
	groups := map[string][]*importer.CatalogItem{}
	for i := range items {
		item := &items[i]
		key, _, _ := strings.Cut(strings.ReplaceAll(item.Rel, "\\", "/"), "/")
		if key == "" {
			key = item.ID
		}
		if _, found := groups[key]; !found {
			order = append(order, key)
		}
		groups[key] = append(groups[key], item)
	}
	for _, key := range order {
		rows := groups[key]
		label := key
		for _, row := range rows {
			if row.Group != "" {
				label = row.Group
				break
			}
		}
		if key == "_no-cwd" {
			label = releaseText("无工作区", "No workspace")
		}
		selected, eligible := 0, 0
		for _, row := range rows {
			if row.Blocker.Importable() {
				eligible++
				if row.Selected {
					selected++
				}
			}
		}
		group := ui.Column(c.Key("import-session-group-" + key)).FillWidth().MinWidth(0).Gap(6)
		expanded := ui.Local(group, "expanded", func() bool { return true })
		group.Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Gap(8).AlignItems(ui.Center).Children(func() {
				all := eligible > 0 && selected == eligible
				check := importGroupCheckbox(c, pal, &all, selected > 0 && selected < eligible, fmt.Sprintf("%s  %d/%d", label, selected, len(rows))).Grow(1).MinWidth(0).Disabled(eligible == 0)
				if check.Changed() {
					for _, row := range rows {
						if row.Blocker.Importable() {
							row.Selected = all
						}
					}
					c.Invalidate()
				}
				arrow := "⌄"
				if *expanded {
					arrow = "⌃"
				}
				if coss.Button(c, "", coss.ButtonOptions{
					Variant: coss.ButtonGhost,
					Size:    coss.ButtonIconSM,
					Label:   releaseText("展开或收起", "Expand or collapse"),
					Content: func() {
						ui.Text(c, arrow).TextColor(pal.secondary)
					},
				}).Expanded(*expanded).Clicked() {
					*expanded = !*expanded
				}
			})
			if !*expanded {
				return
			}
			for _, row := range rows {
				label := row.Label
				if row.Conflict {
					label += i18n.T(i18n.KeyLauncherImportConflict)
				}
				if !row.Blocker.Importable() {
					label += i18n.T(i18n.KeyLauncherImportUnsupported)
					if why := sessionBlockedText(row.Blocker); why != "" {
						label += " · " + why
					}
				}
				if coss.Checkbox(c.Key("session-"+row.Rel+"-"+row.ID), &row.Selected, label, coss.CheckOptions{Disabled: !row.Blocker.Importable()}).FillWidth().MinWidth(0).Padding(design.Sp(1.5), design.Sp(3)).Changed() {
					c.Invalidate()
				}
			}
		})
	}
}

// sessionBlockedText is what a session row says when it cannot be imported.
// The importer classifies; the sentence and its next step live here, in both
// languages, so no scanner's message reaches the row. A row the scanner refused
// for its own reasons keeps only the (unsupported) marker.
func sessionBlockedText(blocker importer.Blocker) string {
	switch blocker {
	case importer.BlockedMixedEncoding:
		return releaseText("这个会话同时有普通日志和压缩日志，无法确定该导入哪一份；请先在桌面端打开这个会话并保存一次，再回来重试。",
			"This session has both plaintext and compressed logs, so there is no single copy to import. Open the session in the desktop app and save it once, then try again.")
	case importer.BlockedLegacyDatabase:
		return releaseText("这是旧版 SQLite 会话库，当前版本无法读取；请先在桌面端完成会话迁移，再回来重试。",
			"This is a legacy SQLite session database that this version cannot read. Finish the session migration in the desktop app, then try again.")
	case importer.BlockedUnreadable:
		return releaseText("读不到这个会话目录，请检查文件夹权限后重试；原因见日志。",
			"This session folder could not be read. Check its permissions and try again; the log has the cause.")
	}
	return ""
}

func importGroupCheckbox(c *ui.Context, pal launcherPalette, checked *bool, mixed bool, label string) ui.Element {
	row := ui.CheckboxBase(c, checked).Gap(8).AlignItems(ui.Center).Label(label)
	if mixed {
		row.Mixed()
	}
	row.Children(func() {
		box := ui.Box(c).Size(16, 16).Radius(4).Shrink(0)
		if *checked || mixed {
			box.Background(pal.accent)
		} else {
			box.Background(pal.base).Border(1, pal.borderL2)
		}
		box.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if mixed {
				p.Line(r.X+4, r.Y+8, r.X+12, r.Y+8, 2, pal.primaryFg)
			} else if *checked {
				p.Line(r.X+4, r.Y+8, r.X+7, r.Y+11, 2, pal.primaryFg)
				p.Line(r.X+7, r.Y+11, r.X+12, r.Y+5, 2, pal.primaryFg)
			}
			if row.FocusVisible() {
				p.FocusRing(r, [4]float32{4, 4, 4, 4})
			}
		})
		ui.Text(c, label).MinWidth(0).Grow(1).FontSize(theme.InterfaceSize(12)).TextColor(pal.primary)
	})
	return row
}

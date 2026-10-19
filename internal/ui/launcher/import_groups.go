package launcher

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Match launcher.js sessionGroupKey: the first directory of the relative path
// owns the group; a session's cwd supplies its human-readable heading.
func importSessionGroups(c *ui.Context, items []importer.CatalogItem) {
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
		groupItems := groups[key]
		label := key
		for _, item := range groupItems {
			if item.Group != "" {
				label = item.Group
				break
			}
		}
		if key == "_no-cwd" {
			label = releaseText("无工作区", "No workspace")
		}
		selected, eligible := 0, 0
		for _, item := range groupItems {
			if !item.Unsupported {
				eligible++
				if item.Selected {
					selected++
				}
			}
		}
		group := ui.Column(c.Key("import-session-group-" + key)).FillWidth().Gap(6)
		expanded := ui.Local(group, "expanded", func() bool { return true })
		group.Children(func() {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				mark := "□"
				if eligible > 0 && selected == eligible {
					mark = "✓"
				} else if selected > 0 {
					mark = "−"
				}
				if ui.Button(c, fmt.Sprintf("%s  %s  %d/%d", mark, label, selected, len(groupItems))).
					Grow(1).Justify(ui.Start).Background(colField).BorderWidth(0).TextColor(colPrimary).
					FontSize(theme.InterfaceSize(12)).Disabled(eligible == 0).Clicked() {
					for _, item := range groupItems {
						if !item.Unsupported {
							item.Selected = selected != eligible
						}
					}
					c.Invalidate()
				}
				fold := releaseText("收起", "Collapse")
				if !*expanded {
					fold = releaseText("展开", "Expand")
				}
				if ghostSmall(c, fold).Clicked() {
					*expanded = !*expanded
				}
			})
			if !*expanded {
				return
			}
			for _, item := range groupItems {
				label := item.Label
				if item.Conflict {
					label += i18n.T(i18n.KeyLauncherImportConflict)
				}
				if item.Unsupported {
					label += i18n.T(i18n.KeyLauncherImportUnsupported)
				}
				mark, ink := "□", colPrimary
				if item.Selected {
					mark = "✓"
				}
				if item.Unsupported {
					ink = colTertiary
				}
				if ui.Button(c.Key("session-"+item.Rel+"-"+item.ID), mark+"  "+label).
					FillWidth().Justify(ui.Start).Padding(6, 12).Background(ui.Transparent).BorderWidth(0).
					TextColor(ink).FontSize(theme.InterfaceSize(12)).Disabled(item.Unsupported).Clicked() {
					item.Selected = !item.Selected
					c.Invalidate()
				}
			}
		})
	}
}

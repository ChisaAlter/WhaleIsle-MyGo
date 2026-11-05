package launcher

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/importer"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

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
			if !row.Unsupported {
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
				check := importGroupCheckbox(c, &all, selected > 0 && selected < eligible, fmt.Sprintf("%s  %d/%d", label, selected, len(rows))).Grow(1).MinWidth(0).Disabled(eligible == 0)
				if check.Changed() {
					for _, row := range rows {
						if !row.Unsupported {
							row.Selected = all
						}
					}
					c.Invalidate()
				}
				arrow := "⌄"
				if *expanded {
					arrow = "⌃"
				}
				if ui.Button(c, arrow).Size(28, 28).Background(ui.Transparent).BorderWidth(0).Label(releaseText("展开或收起", "Expand or collapse")).Expanded(*expanded).Clicked() {
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
				if row.Unsupported {
					label += i18n.T(i18n.KeyLauncherImportUnsupported)
					if row.UnsupportedReason != "" {
						label += " · " + row.UnsupportedReason
					}
				}
				if ui.Checkbox(c.Key("session-"+row.Rel+"-"+row.ID), &row.Selected, label).FillWidth().MinWidth(0).Padding(6, 12).FontSize(theme.InterfaceSize(12)).Disabled(row.Unsupported).Changed() {
					c.Invalidate()
				}
			}
		})
	}
}

func importGroupCheckbox(c *ui.Context, checked *bool, mixed bool, label string) ui.Element {
	row := ui.CheckboxBase(c, checked).Gap(8).AlignItems(ui.Center).Label(label)
	if mixed {
		row.Mixed()
	}
	row.Children(func() {
		box := ui.Box(c).Size(16, 16).Radius(4).Shrink(0)
		if *checked || mixed {
			box.Background(colAccent)
		} else {
			box.Background(colBase).Border(1, colBorderL2)
		}
		box.DrawOver(func(p *ui.Painter, r ui.Rect) {
			if mixed {
				p.Line(r.X+4, r.Y+8, r.X+12, r.Y+8, 2, colPrimaryFg)
			} else if *checked {
				p.Line(r.X+4, r.Y+8, r.X+7, r.Y+11, 2, colPrimaryFg)
				p.Line(r.X+7, r.Y+11, r.X+12, r.Y+5, 2, colPrimaryFg)
			}
			if row.FocusVisible() {
				p.FocusRing(r, [4]float32{4, 4, 4, 4})
			}
		})
		ui.Text(c, label).MinWidth(0).Grow(1).FontSize(theme.InterfaceSize(12)).TextColor(colPrimary)
	})
	return row
}

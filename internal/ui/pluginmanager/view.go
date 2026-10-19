// Package pluginmanager renders the native profile Plugins page. It is not
// the desktop curated marketplace registered in Settings.
package pluginmanager

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// Contribution is a native registration of an original plugins.item,
// plugins.bundle.config or plugins.row.config slot. Register only served pages.
// Row keys are exactly <package>#<patch row id>, never Loader ids.
type Contribution struct {
	Kind, Key, Title, Summary string
	Order                     int
	Render                    func(*ui.Context, manager.Subject)
}
type Model struct {
	Controller                                  *manager.Controller
	ThemeID                                     string
	SystemDark                                  bool
	Contributions                               []Contribution
	RenderConfiguration                         func(*ui.Context, manager.Subject) bool
	DetailActions, DetailBadges, DetailSections func(*ui.Context, manager.Subject)
	AddActions                                  func(*ui.Context)
	OnClose                                     func()
}

func copyText(zh, en string) string {
	if strings.HasPrefix(i18n.Locale(), "zh") {
		return zh
	}
	return en
}

func View(c *ui.Context, m Model) {
	if m.Controller == nil {
		return
	}
	m.Controller.Ensure()
	s := m.Controller.Snapshot()
	a := m.Controller.Actions(s.Owner)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, bg, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.BG), chrome.Color(colors.Border)
	ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).MinWidth(0).Padding(0, 24, 48, 24).Background(bg).AlignItems(ui.Center).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(960).MinWidth(0).Gap(32).Children(func() {
			detail := s.Tab == "installed" && s.View.Kind != "list" && s.View.Kind != "more"
			if !detail && s.View.Kind != "more" {
				pageHeader(c, m, s, a, fg, muted)
				ui.Row(c).FillWidth().Wrap().Gap(4).Role(ui.RoleTabList).Label(copyText("插件视图", "Plugin views")).Margin(-16, 0, 0, 0).Children(func() {
					for _, tab := range []struct{ id, zh, en string }{{"installed", "已安装", "Installed"}, {"discover", "发现", "Discover"}} {
						b := ui.Button(c, copyText(tab.zh, tab.en)).Role(ui.RoleTab).Selected(s.Tab == tab.id).Height(32).MinWidth(120).BorderWidth(0).Radius(8).TextColor(muted).Background(ui.Transparent)
						if s.Tab == tab.id {
							b.Background(chrome.Color(colors.Field)).TextColor(fg)
						}
						if b.Clicked() {
							a.SelectTab(tab.id)
						}
					}
				})
			}
			if s.Notice != "" {
				ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
					ui.Text(c, s.Notice).Grow(1).MinWidth(0).FillWidth().TextColor(fg).FontSize(theme.InterfaceSize(13))
					if ui.Button(c, copyText("关闭提示", "Dismiss notice")).Clicked() {
						a.DismissNotice()
					}
				})
			}
			if s.Tab == "discover" {
				discoverView(c, m, s, a, fg, muted, border)
				return
			}
			if s.Loading && len(s.Packages) == 0 {
				loadingRows(c, muted)
			}
			if s.Status == "disconnected" {
				ui.Text(c, copyText("运行时未连接。连接后可管理此配置的插件。", "Runtime disconnected. Connect to manage this profile's plugins.")).FillWidth().TextColor(muted)
			}
			if s.Status == "unavailable" {
				ui.Text(c, copyText("此运行时没有可管理的 profile，无法安装或启停插件。", "This runtime has no managed profile. Plugin installation and switching are unavailable.")).FillWidth().TextColor(muted)
			}
			if s.Error != "" {
				errorRow(c, s.Error, func() { m.Controller.Refresh(context.Background()) })
			}
			if detail {
				detailView(c, m, s, a, fg, muted, border)
				return
			}
			if s.View.Kind == "more" {
				ui.Row(c).FillWidth().Wrap().Gap(12).Padding(28, 0, 0, 0).Children(func() {
					if ui.Button(c, copyText("返回插件", "Back to plugins")).Clicked() {
						a.Back()
					}
					ui.Text(c, copyText("实验性插件", "Experimental plugins")).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
				})
			}
			installedView(c, m, s, a, fg, muted, border)
		})
	})
	if s.Confirmation != nil {
		confirmationDialog(c, m, s, a)
	} else if s.Install.Open {
		installDialog(c, m, s, a)
	}
}

func pageHeader(c *ui.Context, m Model, s manager.State, a manager.Actions, fg, muted ui.Color) {
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(16).Padding(28, 0, 0, 0).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
			ui.Text(c, copyText("插件", "Plugins")).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
			ui.Text(c, copyText("安装、启用和配置插件", "Install, enable, and configure plugins")).FontSize(theme.InterfaceSize(13)).TextColor(muted).Tooltip(copyText("全部内置插件和运行状态位于设置中的插件清单。", "Inspect all built-in plugins and runtime states in Settings → Plugins."))
		})
		ui.Row(c).Wrap().Gap(12).Children(func() {
			if s.Tab == "installed" {
				if ui.Button(c, copyText("刷新", "Refresh")).Disabled(s.Loading || s.Busy || !s.Connected).Clicked() {
					m.Controller.Refresh(context.Background())
				}
				label := copyText("添加插件", "Add plugin")
				if s.Install.RequestID != "" && (s.Install.Phase == "running" || s.Install.Phase == "unconfirmed" || s.Install.Phase == "applying") {
					label = copyText("查看安装任务", "View installation")
				}
				if ui.PrimaryButton(c, label).Disabled(s.Status != "ready").Clicked() {
					a.OpenInstall()
				}
				if m.AddActions != nil {
					m.AddActions(c)
				}
			}
			if m.OnClose != nil && ui.Button(c, copyText("返回", "Back")).Clicked() {
				m.OnClose()
			}
		})
	})
}
func loadingRows(c *ui.Context, muted ui.Color) {
	ui.Column(c).FillWidth().Gap(16).Role(ui.RoleStatus).Label(copyText("正在读取插件", "Loading plugins")).Children(func() {
		ui.Text(c, copyText("正在读取插件…", "Loading plugins…")).TextColor(muted)
		for i := 0; i < 3; i++ {
			ui.Row(c).FillWidth().Gap(14).Padding(8).Children(func() {
				ui.Box(c).Width(48).Height(48).Radius(8).Background(c.Theme().Surface)
				ui.Column(c).Grow(1).Gap(8).Children(func() {
					ui.Box(c).WidthPercent(35).Height(14).Radius(3).Background(c.Theme().Surface)
					ui.Box(c).WidthPercent(70).Height(12).Radius(3).Background(c.Theme().Surface)
				})
			})
		}
	})
}
func errorRow(c *ui.Context, message string, retry func()) {
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(10).Children(func() {
		ui.Text(c, message).Grow(1).MinWidth(0).FontSize(theme.InterfaceSize(13)).TextColor(c.Theme().Danger)
		if ui.Button(c, copyText("重试", "Retry")).Clicked() {
			retry()
		}
	})
}
func artwork(c *ui.Context, ink, border ui.Color) {
	ui.Box(c).Width(48).Height(48).Shrink(0).Border(.5, border).Radius(10).Draw(func(p *ui.Painter, r ui.Rect) {
		path := &ui.Path{}
		path.MoveTo(r.X+15, r.Y+13)
		path.LineTo(r.X+33, r.Y+13)
		path.LineTo(r.X+33, r.Y+21)
		path.LineTo(r.X+37, r.Y+21)
		path.LineTo(r.X+37, r.Y+29)
		path.LineTo(r.X+33, r.Y+29)
		path.LineTo(r.X+33, r.Y+35)
		path.LineTo(r.X+15, r.Y+35)
		path.Close()
		p.StrokePath(path, 1.3, ink)
	})
}

func installedView(c *ui.Context, m Model, s manager.State, a manager.Actions, fg, muted, border ui.Color) {
	var mine, official []manager.Package
	for _, p := range s.Packages {
		if !manager.Listed(p) {
			continue
		}
		if p.Official || p.Optional {
			official = append(official, p)
		} else {
			mine = append(mine, p)
		}
	}
	sort.SliceStable(official, func(i, j int) bool {
		return manager.OfficialRank(official[i].Name) < manager.OfficialRank(official[j].Name)
	})
	items := append([]Contribution{}, m.Contributions...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Order < items[j].Order })
	if s.View.Kind == "more" {
		for _, p := range official {
			packageCard(c, m, p, s, a, fg, muted, border)
		}
		return
	}
	var basic []Contribution
	for _, item := range items {
		if item.Kind == "item" {
			basic = append(basic, item)
		}
	}
	if len(basic) > 0 {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			groupTitle(c, copyText("内置插件配置", "Built-in plugin settings"), muted)
			for _, item := range basic {
				ui.Row(c).FillWidth().Gap(14).Padding(8).Children(func() {
					artwork(c, muted, border)
					ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
						if ui.Button(c, item.Title).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).FontSize(theme.InterfaceSize(14)).Clicked() {
							a.OpenItem(item.Key)
						}
						ui.Text(c, item.Summary).FillWidth().MaxLines(2).FontSize(theme.InterfaceSize(13)).TextColor(muted)
					})
				})
			}
		})
	}
	if len(official) > 0 {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			ui.Row(c).FillWidth().Children(func() {
				groupTitle(c, copyText("实验性插件", "Experimental plugins"), muted)
				ui.Box(c).Grow(1)
				if ui.Button(c, copyText("更多", "More")).Clicked() {
					a.ShowMore()
				}
			})
			for _, p := range official {
				if manager.Pinned(p) {
					packageCard(c, m, p, s, a, fg, muted, border)
				}
			}
		})
	}
	if len(mine) > 0 {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			groupTitle(c, copyText("已安装插件", "Installed plugins"), muted)
			for _, p := range mine {
				packageCard(c, m, p, s, a, fg, muted, border)
			}
		})
	}
	if len(basic)+len(official)+len(mine) == 0 && s.Status == "ready" {
		ui.Text(c, copyText("还没有安装任何插件。", "No plugins installed yet.")).FillWidth().TextColor(muted)
	}
}
func groupTitle(c *ui.Context, title string, ink ui.Color) {
	ui.Text(c, title).Role(ui.RoleHeading).Level(3).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(ink)
}
func packageCard(c *ui.Context, m Model, p manager.Package, s manager.State, a manager.Actions, fg, muted, border ui.Color) {
	ui.Row(c.Key("plugin-package-" + p.Name)).FillWidth().MinWidth(0).Wrap().Gap(14).Padding(8).Radius(12).Children(func() {
		artwork(c, muted, border)
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
				if ui.Button(c, p.Title(i18n.Locale())).Label("Open " + p.Name).MaxWidthPercent(100).MinWidth(0).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).FontSize(theme.InterfaceSize(14)).Clicked() {
					a.OpenPackage(p.Name)
				}
				packageTags(c, p, muted)
			})
			ui.Text(c, p.Summary(i18n.Locale())).FillWidth().MinWidth(0).MaxLines(2).FontSize(theme.InterfaceSize(13)).TextColor(muted)
		})
		value := p.Enabled
		if ui.Switch(c, &value).Label("Enable " + p.Name).Disabled(s.Busy || s.Loading || p.ReadOnlyReason != "").Changed() {
			a.Enable(p.Name, value)
		}
	})
}
func packageTags(c *ui.Context, p manager.Package, muted ui.Color) {
	if p.Optional {
		ui.Text(c, copyText("实验性", "Experimental")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if p.Error != nil {
		ui.Text(c, copyText("异常", "Problem")).FontSize(theme.InterfaceSize(11)).TextColor(c.Theme().Danger)
	}
	if p.Availability == "missing" {
		ui.Text(c, copyText("未安装 / 文件缺失", "Not installed / missing files")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
	}
	if manager.UpdateAvailable(p) {
		ui.Text(c, copyText("可更新", "Update available")).FontSize(theme.InterfaceSize(11)).TextColor(c.Theme().Accent)
	}
}
func configFor(m Model, subject manager.Subject) *Contribution {
	kind, key := subject.Kind, subject.ItemID
	if kind == "package" || kind == "bundle" {
		kind, key = "bundle", subject.Name
	}
	if kind == "row" {
		key = subject.Name + "#" + subject.RowID
	}
	for i := range m.Contributions {
		item := &m.Contributions[i]
		if item.Kind == kind && item.Key == key {
			return item
		}
	}
	return nil
}
func renderConfig(c *ui.Context, m Model, subject manager.Subject) {
	if subject.Kind == "package" {
		subject.Kind = "bundle"
	}
	if item := configFor(m, subject); item != nil && item.Render != nil {
		item.Render(c, subject)
		return
	}
	if m.RenderConfiguration != nil && m.RenderConfiguration(c, subject) {
		return
	}
	ui.Text(c, copyText("此插件没有注册可用的本地配置页面。", "This plugin has no registered native configuration page.")).FillWidth().FontSize(theme.InterfaceSize(13)).TextColor(c.Theme().TextMuted)
}
func detailView(c *ui.Context, m Model, s manager.State, a manager.Actions, fg, muted, border ui.Color) {
	subject := s.View
	var pkg *manager.Package
	for i := range s.Packages {
		if s.Packages[i].Name == subject.Name {
			pkg = &s.Packages[i]
			break
		}
	}
	ui.Row(c).FillWidth().Wrap().Gap(12).Padding(28, 0, 0, 0).Children(func() {
		if ui.Button(c, copyText("返回插件", "Back to plugins")).Clicked() {
			a.Back()
		}
		if m.DetailActions != nil {
			m.DetailActions(c, subject)
		}
		if pkg != nil && subject.Kind == "package" {
			if manager.UpdateAvailable(*pkg) && ui.Button(c, copyText("更新", "Update")).Disabled(s.Busy).Clicked() {
				a.Update(pkg.Name)
			}
			if pkg.Removable && ui.Button(c, copyText("卸载", "Uninstall")).Disabled(s.Busy || pkg.ReadOnlyReason != "").Clicked() {
				a.Remove(pkg.Name)
			}
			value := pkg.Enabled
			if ui.Switch(c, &value).Label("Enable " + pkg.Name).Disabled(s.Busy || pkg.ReadOnlyReason != "").Changed() {
				a.Enable(pkg.Name, value)
			}
		}
	})
	if subject.Kind == "item" {
		item := configFor(m, subject)
		if item == nil {
			ui.Text(c, copyText("配置页面已不可用。", "Configuration page is no longer available.")).TextColor(muted)
			return
		}
		ui.Text(c, item.Title).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
		ui.Text(c, item.Summary).FillWidth().TextColor(muted)
		renderConfig(c, m, subject)
		if m.DetailSections != nil {
			m.DetailSections(c, subject)
		}
		return
	}
	if pkg == nil {
		ui.Text(c, copyText("插件已不在当前清单中。", "Plugin is no longer in the current inventory.")).TextColor(muted)
		return
	}
	title, description := pkg.Title(i18n.Locale()), pkg.Summary(i18n.Locale())
	var row *manager.Row
	if subject.Kind == "row" {
		for i := range pkg.Parts {
			if pkg.Parts[i].RowID == subject.RowID {
				row = &pkg.Parts[i]
				break
			}
		}
		if row != nil {
			title = row.Title(i18n.Locale())
			if row.Meta != nil {
				description = row.Meta.Description.Resolve(i18n.Locale())
			}
		}
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(8).Children(func() {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
			ui.Text(c, title).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
			if pkg.Version != "" {
				ui.Text(c, "v"+pkg.Version).TextColor(muted)
			}
			packageTags(c, *pkg, muted)
			if m.DetailBadges != nil {
				m.DetailBadges(c, subject)
			}
		})
		ui.Text(c, pkg.Name).FillWidth().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(muted)
		ui.Text(c, description).FillWidth().TextColor(muted)
	})
	if subject.Kind == "row" {
		if row != nil {
			ui.Text(c, row.ModuleName+"\nLoader: "+row.EntryID).FillWidth().Selectable().TextColor(muted)
		}
		renderConfig(c, m, subject)
	} else {
		if pkg.Error != nil {
			ui.Text(c, pkg.Error.Code+": "+pkg.Error.Diagnostic).FillWidth().TextColor(c.Theme().Danger)
		}
		if pkg.ReadOnlyReason != "" {
			ui.Text(c, pkg.ReadOnlyReason).FillWidth().TextColor(muted)
		}
		if configFor(m, subject) != nil || m.RenderConfiguration != nil {
			renderConfig(c, m, subject)
		}
		ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
			groupTitle(c, copyText("包中包含的插件", "Plugins in this package"), muted)
			if len(pkg.Parts) == 0 {
				ui.Text(c, copyText("此包未声明插件条目。", "This package declares no plugin entries.")).TextColor(muted)
			}
			for _, part := range pkg.Parts {
				r := part
				ui.Row(c.Key("plugin-row-" + r.RowID)).FillWidth().MinWidth(0).Wrap().Gap(12).Padding(8).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
						if configFor(m, manager.Subject{Kind: "row", Name: pkg.Name, RowID: r.RowID}) != nil {
							if ui.Button(c, r.Title(i18n.Locale())).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).Clicked() {
								a.OpenRow(pkg.Name, r.RowID)
							}
						} else {
							ui.Text(c, r.Title(i18n.Locale())).FillWidth().TextColor(fg)
						}
						ui.Text(c, r.RowID+" · "+r.ModuleName).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
						ui.Text(c, r.Phase).FontSize(theme.InterfaceSize(11)).TextColor(muted)
					})
					value := r.Enabled
					if ui.Switch(c, &value).Label("Enable entry " + r.RowID).Disabled(s.Busy || r.EntryID == "" || r.ReadOnlyReason != "").Changed() {
						a.EnableRow(r.EntryID, value)
					}
				})
			}
		})
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			groupTitle(c, copyText("来源信息", "Source information"), muted)
			source := pkg.Source
			if source == "" {
				source = copyText("内置", "Built in")
			}
			ui.Text(c, source).FillWidth().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(fg)
			ui.Text(c, pkg.Availability).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			if pkg.InstallTarget != nil {
				ui.Text(c, pkg.InstallTarget.Spec+" · "+pkg.InstallTarget.Version).FillWidth().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			}
		})
	}
	if m.DetailSections != nil {
		m.DetailSections(c, subject)
	}
}

func discoverView(c *ui.Context, m Model, s manager.State, a manager.Actions, fg, muted, border ui.Color) {
	d := s.Discover
	if d.Status == "idle" || d.Status == "loading" {
		loadingRows(c, muted)
	}
	if d.Status == "error" {
		errorRow(c, d.Error, a.ReloadIndexes)
	}
	if d.SelectedEntry != "" && d.Listing != nil {
		for _, entry := range d.Listing.Entries {
			if entry.Name == d.SelectedEntry {
				marketDetail(c, m, entry, s, a, fg, muted)
				return
			}
		}
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		labels := []string{}
		ids := map[string]string{}
		selected := ""
		for _, index := range d.Indexes {
			label := index.Title.Resolve(i18n.Locale())
			if label == "" {
				label = index.URL
			}
			labels = append(labels, label)
			ids[label] = index.URL
			if index.URL == d.Selected {
				selected = label
			}
		}
		if len(labels) > 0 && ui.Select(c, &selected, labels).Width(240).MaxWidthPercent(100).Changed() {
			a.SelectIndex(ids[selected])
		}
		query := d.Query.Text
		if ui.TextInput(c, &query).Label(copyText("搜索插件", "Search plugins")).Placeholder(copyText("搜索插件", "Search plugins")).Grow(1).MinWidth(120).Changed() {
			q := d.Query
			q.Text = query
			a.Search(q)
		}
		if ui.Button(c, copyText("刷新列表", "Refresh listing")).Disabled(d.Loading || d.Selected == "").Clicked() {
			a.RefreshDiscover()
		}
		if ui.Button(c, copyText("索引源", "Index sources")).Clicked() {
			a.ToggleSources()
		}
	})
	if d.SourcesOpen || len(d.Indexes) == 0 {
		sourcesEditor(c, s, a, fg, muted)
	}
	if d.Listing != nil {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(6).Children(func() {
			if ui.Button(c, copyText("全部", "All")).Checked(d.Query.Category == "").Clicked() {
				q := d.Query
				q.Category = ""
				a.Search(q)
			}
			keys := sortedKeys(d.Listing.Metadata.Categories)
			for _, id := range keys {
				label := d.Listing.Metadata.Categories[id].Title.Resolve(i18n.Locale())
				if ui.Button(c, label).Checked(d.Query.Category == id).Clicked() {
					q := d.Query
					q.Category = id
					a.Search(q)
				}
			}
		})
		ui.Row(c).FillWidth().Wrap().Gap(6).Children(func() {
			sorts := []manager.MarketSort{{Kind: "index"}, {Kind: "relevance"}, {Kind: "published", Direction: "desc"}}
			labels := []string{copyText("索引顺序", "Index order"), copyText("相关性", "Relevance"), copyText("最新发布", "Newest published")}
			for _, id := range sortedKeys(d.Listing.Metadata.SortKeys) {
				meta := d.Listing.Metadata.SortKeys[id]
				sorts = append(sorts, manager.MarketSort{Kind: "key", ID: id, Direction: meta.Order})
				labels = append(labels, meta.Title.Resolve(i18n.Locale()))
			}
			selected := labels[0]
			for i, sort := range sorts {
				if sort == d.Query.Sort {
					selected = labels[i]
				}
			}
			if ui.Select(c, &selected, labels).Width(200).MaxWidthPercent(100).Changed() {
				for i, label := range labels {
					if selected == label {
						q := d.Query
						q.Sort = sorts[i]
						a.Search(q)
					}
				}
			}
			for _, id := range sortedKeys(d.Listing.Metadata.Labels) {
				selected := false
				for _, v := range d.Query.Labels {
					if v == id {
						selected = true
					}
				}
				if ui.Button(c, d.Listing.Metadata.Labels[id].Title.Resolve(i18n.Locale())).Checked(selected).Clicked() {
					q := d.Query
					q.Labels = nil
					for _, v := range d.Query.Labels {
						if v != id {
							q.Labels = append(q.Labels, v)
						}
					}
					if !selected {
						q.Labels = append(q.Labels, id)
					}
					a.Search(q)
				}
			}
		})
		ui.Text(c, fmt.Sprintf(copyText("%d 个插件", "%d plugins"), d.Listing.Total)).FontSize(theme.InterfaceSize(12)).TextColor(muted)
		for _, entry := range d.Listing.Entries {
			e := entry
			ui.Row(c.Key("discover-" + e.Name)).FillWidth().MinWidth(0).Wrap().Gap(14).Padding(8).Children(func() {
				artwork(c, muted, border)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					if ui.Button(c, e.Label(i18n.Locale())).Label("Open market " + e.Name).MaxWidthPercent(100).MinWidth(0).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).Clicked() {
						a.OpenEntry(e.Name)
					}
					ui.Text(c, e.Description.Resolve(i18n.Locale())).FillWidth().MaxLines(2).FontSize(theme.InterfaceSize(13)).TextColor(muted)
					ui.Text(c, e.Author).FontSize(theme.InterfaceSize(12)).TextColor(muted)
				})
				marketInstall(c, e, s, a)
			})
		}
		if len(d.Listing.Entries) == 0 && !d.Loading {
			ui.Text(c, copyText("没有匹配的插件。", "No matching plugins.")).FillWidth().TextColor(muted)
		}
		if d.Listing.NextOffset != nil && ui.Button(c, copyText("加载更多", "Show more")).Disabled(d.Loading || d.MoreLoading).Clicked() {
			a.More()
		}
	}
	if d.Error != "" && d.Status != "error" {
		errorRow(c, d.Error, a.RefreshDiscover)
	}
}
func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func marketInstall(c *ui.Context, e manager.MarketEntry, s manager.State, a manager.Actions) {
	if e.Installation.Status != "not-installed" && e.Installation.Status != "update-available" {
		if ui.Button(c, copyText("查看已安装", "View installed")).Clicked() {
			a.OpenPackage(e.Name)
		}
		return
	}
	version := e.DefaultVersion
	if version == "" {
		ui.Button(c, copyText("不兼容", "Incompatible")).Disabled(true)
		return
	}
	label := copyText("安装", "Install")
	if e.Installation.Status == "update-available" {
		label = copyText("更新", "Update")
	}
	if ui.Button(c, label).Disabled(s.Busy || s.Status != "ready").Clicked() {
		a.InstallVersion(e.Name, version)
	}
}
func marketDetail(c *ui.Context, m Model, e manager.MarketEntry, s manager.State, a manager.Actions, fg, muted ui.Color) {
	if ui.Button(c, copyText("返回发现", "Back to Discover")).Clicked() {
		a.CloseEntry()
	}
	ui.Text(c, e.Label(i18n.Locale())).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
	ui.Text(c, e.Name).FillWidth().Selectable().TextColor(muted)
	ui.Text(c, e.Description.Resolve(i18n.Locale())).FillWidth().TextColor(fg)
	ui.Text(c, e.Homepage).FillWidth().Selectable().TextColor(muted)
	if e.Repository != nil {
		ui.Text(c, e.Repository.URL+"\n"+e.Repository.Directory).FillWidth().Selectable().TextColor(muted)
	}
	groupTitle(c, copyText("版本", "Versions"), muted)
	for _, version := range e.Versions {
		v := version
		ui.Row(c).FillWidth().Wrap().Gap(12).Padding(8).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, v.Version).TextColor(fg)
				ui.Text(c, v.Install.Spec).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				if !v.Compatibility.Compatible {
					ui.Text(c, copyText("运行时要求未满足：", "Unmet runtime requirement: ")+v.Compatibility.RuntimeVersion).FillWidth().TextColor(c.Theme().Warning)
					for _, key := range sortedKeys(v.Compatibility.Requirements) {
						ui.Text(c, key+": "+v.Compatibility.Requirements[key]).FillWidth().TextColor(muted)
					}
				}
			})
			if ui.Button(c, copyText("安装此版本", "Install this version")).Disabled(s.Busy || s.Status != "ready" || !v.Compatibility.Compatible && !v.Compatibility.Exempted).Clicked() {
				a.InstallVersion(e.Name, v.Version)
			}
		})
	}
}
func sourcesEditor(c *ui.Context, s manager.State, a manager.Actions, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(10).Padding(12).Border(.5, c.Theme().Border).Radius(8).Children(func() {
		groupTitle(c, copyText("插件索引源", "Plugin index sources"), fg)
		if len(s.Discover.Indexes) == 0 {
			ui.Text(c, copyText("尚未配置索引源。添加一个源以浏览插件。", "No index sources configured. Add a source to browse plugins.")).FillWidth().TextColor(muted)
		}
		for _, index := range s.Discover.Indexes {
			i := index
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
				ui.Column(c).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, i.URL).FillWidth().Selectable().FontSize(theme.InterfaceSize(12)).TextColor(fg)
					if i.Error != "" {
						ui.Text(c, i.Error).FillWidth().TextColor(c.Theme().Danger)
					}
				})
				if ui.Button(c, copyText("移除源", "Remove source")).Disabled(s.Busy).Clicked() {
					a.RemoveSource(i.URL)
				}
			})
		}
		draft := s.Discover.Draft
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
			if ui.TextInput(c, &draft).Label(copyText("索引 URL", "Index URL")).Placeholder("https://example.com/plugins.json").Grow(1).MinWidth(120).Changed() {
				a.EditSource(draft)
			}
			if ui.Button(c, copyText("添加源", "Add source")).Disabled(s.Busy || strings.TrimSpace(draft) == "").Clicked() {
				a.AddSource()
			}
		})
	})
}

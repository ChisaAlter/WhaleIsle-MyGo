// Package pluginmanager renders the native profile Plugins page. It is not
// the desktop curated marketplace registered in Settings.
package pluginmanager

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/widgets"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	marketui "whaleisle/internal/ui/market"
	"whaleisle/internal/ui/svgimage"
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
	SVGImages                                   *svgimage.Owner
	Controller                                  *manager.Controller
	ThemeID                                     string
	SystemDark                                  bool
	Contributions                               []Contribution
	RenderConfiguration                         func(*ui.Context, manager.Subject) bool
	DetailActions, DetailBadges, DetailSections func(*ui.Context, manager.Subject)
	AddActions                                  func(*ui.Context)
	OnClose                                     func()
	OnOpenURL                                   func(string)
	OnCreatePlugin, OnOpenAddMenu               func()
	CreatePluginReady                           bool
	CreatePluginDescription                     string
	CreatorNotice                               func(*ui.Context)
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
	w, _ := c.Size()
	inset := max(float32(24), min(float32(48), w*.04))
	ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).MinWidth(0).Padding(0, inset, 48, inset).Background(bg).AlignItems(ui.Center).Children(func() {
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
			if m.CreatorNotice != nil {
				m.CreatorNotice(c)
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
	} else if s.Discover.Choice != nil {
		marketChoiceDialog(c, m, s, a)
	} else if s.Install.Open {
		installDialog(c, m, s, a)
	} else if s.Discover.SourcesOpen {
		sourcesDialog(c, m, s, a)
	}
	if s.PreferenceError != "" {
		ui.Overlay(c.Key("plugin-registry-preference-error"), func() {
			ui.Row(c).Absolute().Bottom(24).Left(24).Right(24).MaxWidth(640).Padding(12).Gap(10).Role(ui.RoleStatus).Background(bg).Border(.5, border).Radius(8).Children(func() {
				ui.Text(c, s.PreferenceError).Grow(1).MinWidth(0).TextColor(c.Theme().Danger)
				if ui.Button(c, copyText("关闭", "Close")).Clicked() {
					a.DismissPreferenceError()
				}
			})
		})
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
				if s.Install.RequestID != "" {
					if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), copyText("查看安装任务", "View installation"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium, Disabled: !s.ManagementReady()}).Clicked() {
						a.OpenInstall()
					}
				} else {
					addPluginMenu(c, m, s, a)
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
func artwork(c *ui.Context, ink, border ui.Color, owner *svgimage.Owner, source ...string) {
	artworkSized(c, ink, border, 48, 36, owner, source...)
}
func artworkSized(c *ui.Context, ink, border ui.Color, frame, size float32, owner *svgimage.Owner, source ...string) {
	root := ui.Box(c).Width(frame).Height(frame).Shrink(0).Border(.5, border).Radius(10)
	shown := false
	if len(source) > 0 && source[0] != "" {
		root.AlignItems(ui.Center).Justify(ui.Center).Children(func() { shown = marketui.PluginImage(c, source[0], size, size, owner) })
	}
	if !shown {
		root.AlignItems(ui.Center).Justify(ui.Center).Children(func() { shown = marketui.PluginImage(c, builtInArtwork("default"), size, size, owner) })
	}
	if !shown {
		root.Draw(func(p *ui.Painter, r ui.Rect) {
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
}
func installedView(c *ui.Context, m Model, s manager.State, a manager.Actions, fg, muted, border ui.Color) {
	packages := make([]manager.Package, 0, len(s.Packages))
	for _, p := range s.Packages {
		if manager.Listed(p) {
			packages = append(packages, p)
		}
	}
	sort.SliceStable(packages, func(i, j int) bool {
		if packages[i].Official != packages[j].Official {
			return packages[i].Official
		}
		return packages[i].Name < packages[j].Name
	})
	items := append([]Contribution{}, m.Contributions...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Order < items[j].Order })
	if s.View.Kind == "more" {
		for _, p := range packages {
			if p.Official {
				packageCard(c, m, p, s, a, fg, muted, border)
			}
		}
		return
	}

	anchor := ui.Column(c).Key("plugins-installed").FillWidth().MinWidth(0).Gap(10)
	query := ui.Local(anchor, "plugin-filter", func() string { return "" })
	if len(packages) > 0 {
		anchor.Children(func() {
			ui.TextInput(c, query).FillWidth().Placeholder(copyText("按名称筛选插件", "Filter plugins by name"))
		})
	}
	matches := func(p manager.Package) bool {
		q := strings.ToLower(strings.TrimSpace(*query))
		if q == "" {
			return true
		}
		return strings.Contains(strings.ToLower(p.Name), q) || strings.Contains(strings.ToLower(p.Title(i18n.Locale())), q)
	}

	var enabled, disabled, official []manager.Package
	heldBack := false
	for _, p := range packages {
		if !matches(p) {
			continue
		}
		if p.Official {
			if manager.Pinned(p) {
				official = append(official, p)
			} else {
				heldBack = true
			}
			continue
		}
		if p.Enabled {
			enabled = append(enabled, p)
		} else {
			disabled = append(disabled, p)
		}
	}
	var basic []Contribution
	for _, item := range items {
		if item.Kind == "item" {
			basic = append(basic, item)
		}
	}
	anchor.Children(func() {
		if len(basic) > 0 {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				groupTitle(c, copyText("内置插件配置", "Built-in plugin settings"), muted)
				for _, item := range basic {
					ui.Row(c).FillWidth().Gap(14).Padding(8).Children(func() {
						artwork(c, muted, border, m.SVGImages, itemArtwork(item.Key))
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
					if heldBack && widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), copyText("更多", "More"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
						a.ShowMore()
					}
				})
				for _, p := range official {
					packageCard(c, m, p, s, a, fg, muted, border)
				}
			})
		}
		// Enabled first: the list answers "what runs now" before "what is
		// installed but idle".
		for _, group := range []struct {
			title string
			rows  []manager.Package
		}{
			{groupCount(copyText("已启用", "Enabled"), len(enabled)), enabled},
			{groupCount(copyText("已停用", "Disabled"), len(disabled)), disabled},
		} {
			if len(group.rows) == 0 {
				continue
			}
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				groupTitle(c, group.title, muted)
				for _, p := range group.rows {
					packageCard(c, m, p, s, a, fg, muted, border)
				}
			})
		}
		if len(packages) > 0 && len(enabled)+len(disabled) == 0 {
			ui.Text(c, copyText("没有匹配的插件。", "No matching plugins.")).FillWidth().TextColor(muted)
		}
		if len(packages) == 0 && s.Status == "ready" {
			ui.Text(c, copyText("还没有安装任何插件。", "No plugins installed yet.")).FillWidth().TextColor(muted)
		}
	})
}

// groupCount names a group with the number of plugins it holds.
func groupCount(label string, count int) string {
	if count == 0 {
		return label
	}
	return fmt.Sprintf("%s (%d)", label, count)
}

func groupTitle(c *ui.Context, title string, ink ui.Color) {
	ui.Text(c, title).Role(ui.RoleHeading).Level(3).FontSize(theme.InterfaceSize(13)).FontWeight(500).TextColor(ink)
}

func packageCard(c *ui.Context, m Model, p manager.Package, s manager.State, a manager.Actions, fg, muted, border ui.Color) {
	card := ui.Row(c.Key("plugin-package-"+p.Name)).FillWidth().MinWidth(0).Wrap().Gap(14).Padding(8).Radius(12).AlignItems(ui.Center)
	if card.Hovered() {
		card.Background(c.Theme().SurfaceHover)
	}
	if s.Highlight == p.Name && time.Now().UnixNano() < s.HighlightUntil {
		card.Background(c.Theme().Surface)
		seen := ui.Local(card, "install-highlight", func() int64 { return 0 })
		if *seen != s.HighlightUntil {
			*seen = s.HighlightUntil
			card.ScrollIntoView()
		}
		c.After(time.Until(time.Unix(0, s.HighlightUntil)))
	}
	card.ContextMenu(func(menu *ui.Menu) { packageMenu(menu, p, s, a) })
	card.Children(func() {
		artwork(c, muted, border, m.SVGImages, metaIcon(p.Meta))
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
				if ui.Button(c, p.Title(i18n.Locale())).Label("Open " + p.Name).MaxWidthPercent(100).MinWidth(0).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).FontSize(theme.InterfaceSize(14)).Clicked() {
					a.OpenPackage(p.Name)
				}
				packageTags(c, p, muted)
			})
			ui.Text(c, p.Summary(i18n.Locale())).FillWidth().MinWidth(0).MaxLines(2).FontSize(theme.InterfaceSize(13)).TextColor(muted)
			if meta := packageMetaLine(p); meta != "" {
				ui.Text(c, meta).FillWidth().MinWidth(0).SingleLine().Tooltip(meta).FontSize(theme.InterfaceSize(11)).TextColor(muted)
			}
		})
		value := p.Enabled
		if ui.Switch(c, &value).Label("Enable " + p.Name).Disabled(s.ActionBusy(p.Name) || p.SwitchLocked()).Changed() {
			a.Enable(p.Name, value)
		}
		menuLabel := copyText("更多操作", "More actions")
		more := ui.Button(c, "").Label(menuLabel).Size(24, 24).Shrink(0).Padding(0).Radius(6).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(menuLabel)
		more.Children(func() { ui.Text(c, "⋯").FontSize(theme.InterfaceSize(14)).TextColor(muted) })
		if more.Hovered() {
			more.Background(c.Theme().SurfaceHover)
		}
		more.Menu(func(menu *ui.Menu) { packageMenu(menu, p, s, a) })
	})
}

// packageMenu is one plugin's action list, shared by the overflow button and the
// row's context menu.
func packageMenu(menu *ui.Menu, p manager.Package, s manager.State, a manager.Actions) {
	if menu.Item(copyText("打开详情", "Open details")).Chosen() {
		a.OpenPackage(p.Name)
	}
	menu.Separator()
	if menu.Item(copyText("停用", "Disable")).Disabled(!p.Enabled || s.ActionBusy(p.Name) || p.SwitchLocked()).Chosen() {
		a.Enable(p.Name, false)
	}
	if menu.Item(copyText("启用", "Enable")).Disabled(p.Enabled || s.ActionBusy(p.Name) || p.SwitchLocked()).Chosen() {
		a.Enable(p.Name, true)
	}
	if p.Removable {
		menu.Separator()
		if menu.Item(copyText("移除插件…", "Remove plugin…")).Disabled(s.ActionBusy(p.Name)).Chosen() {
			a.Remove(p.Name)
		}
	}
}

// packageMetaLine is the row's source line: version, where the plugin came from,
// and whether it is optional.
func packageMetaLine(p manager.Package) string {
	parts := []string{}
	if p.Version != "" {
		parts = append(parts, "v"+p.Version)
	}
	parts = append(parts, packageSource(p))
	if p.Optional {
		parts = append(parts, copyText("可选", "Optional"))
	}
	return strings.Join(parts, " · ")
}

// packageSource names where a plugin comes from, in the terms the manager
// already distinguishes: bundled, installed from a market entry, or local.
func packageSource(p manager.Package) string {
	switch {
	case p.Official:
		return copyText("内置", "Built-in")
	case p.InstallTarget != nil:
		return copyText("市场", "Market")
	case p.Source != "":
		return copyText("本地", "Local")
	default:
		return copyText("本地", "Local")
	}
}
func packageTags(c *ui.Context, p manager.Package, muted ui.Color) {
	if strings.HasPrefix(p.Name, "@deepseek-ai/dsh-experimental-") {
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
			if manager.UpdateAvailable(*pkg) && ui.Button(c, copyText("更新", "Update")).Disabled(s.ActionBusy(pkg.Name)).Clicked() {
				a.Update(pkg.Name)
			}
			if (pkg.Installed || pkg.Removable) && ui.Button(c, copyText("卸载", "Uninstall")).Disabled(s.ActionBusy(pkg.Name) || !pkg.Removable || pkg.ReadOnlyReason != "").Clicked() {
				a.Remove(pkg.Name)
			}
			value := pkg.Enabled
			if ui.Switch(c, &value).Label("Enable " + pkg.Name).Disabled(s.ActionBusy(pkg.Name) || pkg.SwitchLocked()).Changed() {
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
	if pkg.Meta != nil && pkg.Meta.Error != "" {
		ui.Text(c, pkg.Meta.Error).FillWidth().Role(ui.RoleStatus).TextColor(c.Theme().Warning)
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
		packagePreview(c, pkg.Name, m.SVGImages)
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
			if pkg.Availability == "missing" {
				return
			}
			groupTitle(c, copyText("包中包含的插件", "Plugins in this package"), muted)
			ui.Text(c, partsSummary(pkg.Parts)).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			filter := s.RowFilter
			if len(pkg.Parts) > 10 && ui.TextInput(c, &filter).FillWidth().Label(copyText("筛选组件", "Filter components")).Changed() {
				a.FilterRows(filter)
			}
			query := strings.ToLower(strings.TrimSpace(filter))
			shown := 0
			if len(pkg.Parts) == 0 {
				ui.Text(c, copyText("此包未声明插件条目。", "This package declares no plugin entries.")).TextColor(muted)
			}
			for _, part := range pkg.Parts {
				r := part
				description := ""
				if r.Meta != nil {
					description = r.Meta.Description.Resolve(i18n.Locale())
				}
				if query != "" && !strings.Contains(strings.ToLower(strings.Join([]string{r.Title(i18n.Locale()), description, r.RowID, r.ModuleName}, " ")), query) {
					continue
				}
				shown++
				ui.Row(c.Key("plugin-row-" + r.RowID)).FillWidth().MinWidth(0).Wrap().Gap(12).Padding(8).Children(func() {
					icon := metaIcon(r.Meta)
					if icon == "" {
						icon = builtInArtwork("subagent")
					}
					artworkSized(c, muted, border, 40, 30, m.SVGImages, icon)
					ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
						if configFor(m, manager.Subject{Kind: "row", Name: pkg.Name, RowID: r.RowID}) != nil {
							if ui.Button(c, r.Title(i18n.Locale())).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).Clicked() {
								a.OpenRow(pkg.Name, r.RowID)
							}
						} else {
							ui.Text(c, r.Title(i18n.Locale())).FillWidth().TextColor(fg)
						}
						ui.Text(c, r.RowID+" · "+r.ModuleName).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
						if description != "" {
							ui.Text(c, description).FillWidth().TextColor(muted)
						}
						phase := r.Phase
						if !r.Enabled {
							phase = copyText("已关闭", "Off")
						} else if phase == "" {
							phase = copyText("空闲", "Idle")
						}
						ui.Text(c, phase).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						if r.Meta != nil && r.Meta.Error != "" {
							ui.Text(c, r.Meta.Error).FillWidth().Role(ui.RoleStatus).TextColor(c.Theme().Warning)
						}
						if r.ReadOnlyReason != "" {
							ui.Text(c, r.ReadOnlyReason).FillWidth().TextColor(muted)
						}
					})
					value := r.Enabled
					if pkg.Enabled && ui.Switch(c, &value).Label("Enable entry "+r.RowID).Disabled(s.ActionBusy(pkg.Name) || s.ActionBusy(r.EntryID) || r.EntryID == "" || r.ReadOnlyReason != "").Changed() {
						a.EnableRow(r.EntryID, value)
					}
				})
			}
			if len(pkg.Parts) > 0 && shown == 0 {
				ui.Text(c, copyText("没有匹配的组件。", "No matching components.")).TextColor(muted)
			}
		})
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			groupTitle(c, copyText("来源信息", "Source information"), muted)
			source := pkg.Source
			if source == "" {
				if pkg.Availability == "missing" {
					if pkg.Installed {
						source = copyText("已声明，但安装文件缺失", "Declared dependency; installed files missing")
					} else {
						source = copyText("尚未安装", "Not installed")
					}
				} else {
					source = copyText("内置", "Built in")
				}
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
		return
	}
	if d.Status == "error" {
		errorRow(c, d.IndexesError, a.ReloadIndexes)
		return
	}
	if d.SelectedEntry != "" && d.Listing != nil {
		if d.Opened != nil {
			marketDetail(c, m, *d.Opened, s, a, fg, muted)
			return
		}
		for _, entry := range d.Listing.Entries {
			if entry.Name == d.SelectedEntry {
				marketDetail(c, m, entry, s, a, fg, muted)
				return
			}
		}
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
		labels := []string{}
		ids := []string{}
		selected := ""
		for _, index := range d.Indexes {
			label := index.Title.Resolve(i18n.Locale())
			if label == "" {
				label = index.URL
				if parsed, err := url.Parse(index.URL); err == nil && parsed.Host != "" {
					label = parsed.Host
				}
			}
			labels = append(labels, label)
			// Preserve duplicate localized titles as independent source choices.
			for _, other := range d.Indexes {
				if other.URL != index.URL && other.Title.Resolve(i18n.Locale()) == index.Title.Resolve(i18n.Locale()) {
					label += " · " + index.URL
					break
				}
			}
			labels[len(labels)-1] = label
			ids = append(ids, index.URL)
			if index.URL == d.Selected {
				selected = label
			}
		}
		if len(labels) > 0 && ui.Select(c, &selected, labels).Width(240).MaxWidthPercent(100).Changed() {
			for i, label := range labels {
				if selected == label {
					a.SelectIndex(ids[i])
					break
				}
			}
		}
		query := d.TextDraft
		if ui.TextInput(c, &query).Label(copyText("搜索插件", "Search plugins")).Placeholder(copyText("搜索插件", "Search plugins")).Grow(1).MinWidth(120).Changed() {
			a.SearchText(query)
		}
		if ui.Button(c, copyText("刷新列表", "Refresh listing")).Disabled(d.Loading || d.Selected == "").Clicked() {
			a.RefreshDiscover()
		}
		if ui.Button(c, copyText("索引源", "Index sources")).Clicked() {
			a.ToggleSources()
		}
	})
	if len(d.Indexes) == 0 {
		ui.Text(c, copyText("尚未配置插件索引。添加一个源以浏览插件。", "No plugin indexes configured. Add a source to browse plugins.")).FillWidth().TextColor(muted)
		if widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), copyText("添加索引源", "Add index source"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium}).Clicked() {
			a.ToggleSources()
		}
		return
	}
	if d.Loading && d.Listing == nil {
		loadingRows(c, muted)
	}
	if d.MoreLoading {
		ui.Text(c, copyText("正在读取更多插件…", "Loading more plugins…")).Role(ui.RoleStatus).TextColor(muted)
	}
	if d.Listing != nil {
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(6).Children(func() {
			if ui.Button(c, copyText("全部", "All")).Checked(d.Query.Category == "").Clicked() {
				q := d.Query
				q.Category = ""
				a.Search(q)
			}
			keys := declarationKeys(d.Listing.Metadata.Categories)
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
			sorts := []manager.MarketSort{{Kind: "index"}, {Kind: "published", Direction: "desc"}}
			labels := []string{copyText("索引顺序", "Index order"), copyText("最新发布", "Newest published")}
			if strings.TrimSpace(d.Query.Text) != "" {
				sorts = append(sorts, manager.MarketSort{Kind: "relevance"})
				labels = append(labels, copyText("相关性", "Relevance"))
			}
			for _, id := range declarationKeys(d.Listing.Metadata.SortKeys) {
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
			if d.Query.Sort.Kind == "published" || d.Query.Sort.Kind == "key" {
				if ui.Button(c, copyText("切换升序/降序", "Reverse sort direction")).Clicked() {
					q := d.Query
					if q.Sort.Direction == "asc" {
						q.Sort.Direction = "desc"
					} else {
						q.Sort.Direction = "asc"
					}
					a.Search(q)
				}
			}
			for _, id := range declarationKeys(d.Listing.Metadata.Labels) {
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
				artwork(c, muted, border, m.SVGImages, e.Icon)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					if ui.Button(c, e.Label(i18n.Locale())).Label("Open market " + e.Name).MaxWidthPercent(100).MinWidth(0).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).Clicked() {
						a.OpenEntry(e.Name)
					}
					ui.Text(c, e.Description.Resolve(i18n.Locale())).FillWidth().MaxLines(2).FontSize(theme.InterfaceSize(13)).TextColor(muted)
					meta := e.Author
					if version := presentedVersion(e); version != nil {
						meta = strings.TrimSpace(meta + " · " + version.Version + " · " + time.Unix(version.PublishTimestamp, 0).Local().Format("2006-01-02"))
					}
					ui.Text(c, meta).FontSize(theme.InterfaceSize(12)).TextColor(muted)
					for _, id := range e.Metadata.Labels {
						if label, ok := d.Listing.Metadata.Labels[id]; ok {
							ui.Text(c, label.Title.Resolve(i18n.Locale())).FontSize(theme.InterfaceSize(11)).TextColor(muted)
						}
					}
				})
				marketInstall(c, e, s, a)
			})
		}
		if len(d.Listing.Entries) == 0 && !d.Loading {
			query := d.Query
			if d.Answered != nil {
				query = *d.Answered
			}
			message := copyText("没有匹配的插件。", "No matching plugins.")
			if strings.TrimSpace(query.Text) == "" && query.Category == "" && len(query.Labels) == 0 {
				message = copyText("此索引尚无插件。", "This index has no plugins.")
			}
			ui.Text(c, message).FillWidth().TextColor(muted)
		}
		if d.Listing.NextOffset != nil && ui.Button(c, copyText("加载更多", "Show more")).Disabled(d.Loading || d.MoreLoading || d.Error != "").Clicked() {
			a.More()
		}
	}
	if d.Error != "" && d.Status != "error" {
		errorRow(c, d.Error, a.RetryDiscover)
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
func declarationKeys(values map[string]manager.NamedMetadata) []string {
	keys := sortedKeys(values)
	order := collate.New(language.Make(i18n.Locale()))
	sort.SliceStable(keys, func(i, j int) bool {
		comparison := order.CompareString(values[keys[i]].Title.Resolve(i18n.Locale()), values[keys[j]].Title.Resolve(i18n.Locale()))
		return comparison < 0 || comparison == 0 && keys[i] < keys[j]
	})
	return keys
}
func marketInstall(c *ui.Context, e manager.MarketEntry, s manager.State, a manager.Actions) {
	if e.Installation.Status != "not-installed" {
		label := copyText("已安装", "Installed")
		if e.Installation.Status == "update-available" {
			label = copyText("可更新", "Update available")
		} else if e.Installation.Status == "provided" {
			label = copyText("内置提供", "Provided")
		}
		if ui.Button(c, label).Clicked() {
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
	if ui.Button(c, label).Clicked() {
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
	marketInstall(c, e, s, a)
	ui.Text(c, e.Author).FillWidth().TextColor(muted)
	detailLink(c, m, e.Homepage, muted)
	if e.Repository != nil {
		detailLink(c, m, e.Repository.URL, muted)
		ui.Text(c, e.Repository.Directory).FillWidth().Selectable().TextColor(muted)
	}
	if v := presentedVersion(e); v != nil {
		for _, fact := range manager.SourceFacts(e.Name, *v) {
			ui.Text(c, fact.Kind+": "+fact.Value).FillWidth().Selectable().TextColor(muted)
		}
		ui.Text(c, time.Unix(v.PublishTimestamp, 0).Local().Format("2006-01-02")).TextColor(muted)
	}
	if listing := s.Discover.Listing; listing != nil {
		if category, ok := listing.Metadata.Categories[e.Metadata.Category]; ok {
			ui.Text(c, category.Title.Resolve(i18n.Locale())).TextColor(muted)
		}
		for _, id := range e.Metadata.Labels {
			if label, ok := listing.Metadata.Labels[id]; ok {
				ui.Text(c, label.Title.Resolve(i18n.Locale())).TextColor(muted)
			}
		}
		ui.Text(c, listing.Title.Resolve(i18n.Locale())).FillWidth().TextColor(muted)
	}
	groupTitle(c, copyText("版本", "Versions"), muted)
	for _, version := range e.Versions {
		v := version
		ui.Row(c).FillWidth().Wrap().Gap(12).Padding(8).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, v.Version).TextColor(fg)
				if v.Version == e.DefaultVersion {
					ui.Text(c, copyText("默认版本", "Default version")).TextColor(muted)
				}
				if strings.Contains(strings.Split(v.Version, "+")[0], "-") {
					ui.Text(c, copyText("预发布", "Prerelease")).TextColor(muted)
				}
				if v.Version == e.Installation.InstalledVersion {
					ui.Text(c, copyText("已安装", "Installed")).TextColor(muted)
				}
				ui.Text(c, time.Unix(v.PublishTimestamp, 0).Local().Format("2006-01-02")).TextColor(muted)
				ui.Text(c, v.Install.Spec).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
				if !v.Compatibility.Compatible {
					message := copyText("运行时要求未满足：", "Unmet runtime requirement: ") + v.Compatibility.RuntimeVersion
					if v.Compatibility.Exempted {
						message = copyText("已接受此版本的兼容性风险。", "Compatibility risk accepted for this version.")
					}
					ui.Text(c, message).FillWidth().TextColor(c.Theme().Warning)
					for _, key := range sortedKeys(v.Compatibility.Requirements) {
						ui.Text(c, key+": "+v.Compatibility.Requirements[key]).FillWidth().TextColor(muted)
					}
				}
			})
			offered := e.Installation.Status != "provided" && e.Installation.InstalledVersion != v.Version && (v.Compatibility.Compatible || v.Compatibility.Exempted)
			if offered && ui.Button(c, copyText("安装此版本", "Install this version")).Clicked() {
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
				if ui.Button(c, copyText("移除源", "Remove source")).Disabled(s.Discover.Saving).Clicked() {
					a.RemoveSource(i.URL)
				}
			})
		}
		draft := s.Discover.Draft
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
			if ui.TextInput(c, &draft).Label(copyText("索引 URL", "Index URL")).Tooltip(s.Discover.SourceError).Placeholder("https://example.com/plugins.json").Grow(1).MinWidth(120).Changed() {
				a.EditSource(draft)
			}
			if ui.Button(c, copyText("添加源", "Add source")).Disabled(s.Discover.Saving || strings.TrimSpace(draft) == "").Clicked() {
				a.AddSource()
			}
		})
		ui.Text(c, copyText("请仅添加信任的索引源；插件会以当前配置的权限运行。", "Add only trusted index sources. Plugins run with this profile's permissions.")).FillWidth().TextColor(muted)
	})
}

func presentedVersion(e manager.MarketEntry) *manager.MarketVersion {
	for i := range e.Versions {
		if e.Versions[i].Version == e.DefaultVersion {
			return &e.Versions[i]
		}
	}
	if len(e.Versions) > 0 {
		return &e.Versions[0]
	}
	return nil
}
func partsSummary(rows []manager.Row) string {
	running, off, failed := 0, 0, 0
	for _, r := range rows {
		if !r.Enabled {
			off++
		}
		if r.Enabled && r.Phase == "active" {
			running++
		}
		if r.Phase == "failed" {
			failed++
		}
	}
	parts := []string{fmt.Sprintf(copyText("共 %d", "Total %d"), len(rows))}
	if running > 0 {
		parts = append(parts, fmt.Sprintf(copyText("运行 %d", "Running %d"), running))
	}
	if off > 0 {
		parts = append(parts, fmt.Sprintf(copyText("关闭 %d", "Off %d"), off))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf(copyText("失败 %d", "Failed %d"), failed))
	}
	return strings.Join(parts, " · ")
}
func detailLink(c *ui.Context, m Model, raw string, ink ui.Color) {
	if raw == "" {
		return
	}
	if m.OnOpenURL != nil && (strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://")) {
		if ui.Button(c, raw).MaxWidthPercent(100).MinWidth(0).BorderWidth(0).Padding(0).Background(ui.Transparent).TextColor(ink).Clicked() {
			m.OnOpenURL(raw)
		}
	} else {
		ui.Text(c, raw).FillWidth().Selectable().TextColor(ink)
	}
}
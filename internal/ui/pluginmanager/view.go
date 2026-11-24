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
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"whaleisle/internal/desktop/media"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	marketui "whaleisle/internal/ui/market"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/widgets"
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
	Media                   *media.Owner
	Controller              *manager.Controller
	ThemeID                 string
	SystemDark              bool
	Contributions           []Contribution
	RenderConfiguration     func(*ui.Context, manager.Subject) bool
	OnOpenURL               func(string)
	OnCreatePlugin          func()
	OnOpenAddMenu           func()
	CreatePluginReady       bool
	CreatePluginDescription string
	CreatorNotice           func(*ui.Context)
}

func copyText(zh, en string) string {
	return i18n.Pick(zh, en)
}

func View(c *ui.Context, m Model) {
	if m.Controller == nil {
		return
	}
	m.Controller.Ensure()
	s := m.Controller.Snapshot().Value
	a := m.Controller.Actions(s.Owner)
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	fg, muted, bg, border := chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.BG), chrome.Color(colors.Border)
	w, _ := c.Size()
	inset := max(float32(24), min(float32(48), w*.04))
	ui.Scroll(c).Grow(1).FillWidth().MinHeight(0).MinWidth(0).Padding(0, inset, 48, inset).Background(ui.Transparent).AlignItems(ui.Center).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(960).MinWidth(0).Gap(32).Children(func() {
			detail := s.Tab == "installed" && s.View.Kind != "list" && s.View.Kind != "more"
			if !detail && s.View.Kind != "more" {
				pageHeader(c, m, s, a, fg, muted)
				// The original mounts its SegmentedTabs row only while a
				// plugins.tab slot contribution exists — the market client
				// bundle (PluginManagerPage.tsx: tabs.length > 0). Native's
				// equivalent is the Host answering pluginMarket/listIndexes,
				// probed on page mount; the row also stays while the Discover
				// tab itself is already active.
				if s.Tab == "discover" || s.Discover.Probe == "ready" || s.Discover.Status == "ready" {
					// .pageTabList is a SegmentedTabs box: fit-content, min 240,
					// 4px inset, radius lg on module-platform; each .tab is a
					// 34px transparent button (14/20, secondary) and the selected
					// one rides the indicator look — radius md, layer-3 fill,
					// .5 l3 stroke, primary 600 (ui-primitives
					// SegmentedTabs.module.css .tabs/.tab/.indicator).
					chrome.FrostSurface(ui.Row(c).AlignSelf(ui.Start).MinWidth(240).Padding(4), colors, m.SystemDark, chrome.Color(colors.Field), theme.ContextRadii(c).LG).Role(ui.RoleTabList).Label(copyText("插件视图", "Plugin views")).Margin(-8, 0, 0, 0).Children(func() {
						for _, tab := range []struct{ id, zh, en string }{{"installed", "已安装", "Installed"}, {"discover", "发现", "Discover"}} {
							b := ui.Button(c, copyText(tab.zh, tab.en)).Role(ui.RoleTab).Selected(s.Tab == tab.id).Height(34).MinWidth(120).Padding(0, 12).
								BorderWidth(0).Radius(theme.ContextRadii(c).MD).TextColor(muted).Background(ui.Transparent).
								FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20))
							if s.Tab == tab.id {
								b = b.Background(chrome.FrostInset(colors, m.SystemDark, chrome.Color(colors.BG))).TextColor(fg).FontWeight(600).Border(.5, chrome.Color(colors.BorderL3))
							}
							if b.Clicked() {
								a.SelectTab(tab.id)
							}
						}
					})
				}
			}
			widgets.AckLine(c, "view:113", s.Notice)
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
				ui.Text(c, copyText("本部署没有可管理的 profile，无法安装或启停插件。", "Runtime disconnected. Connect to manage this profile's plugins.")).FillWidth().TextColor(muted)
			}
			if s.Status == "unavailable" {
				ui.Text(c, copyText("本部署没有可管理的 profile，无法安装或启停插件。", "This runtime has no managed profile. Plugin installation and switching are unavailable.")).FillWidth().TextColor(muted)
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
			chrome.FrostSurface(ui.Row(c).Absolute().Bottom(24).Left(24).Right(24).MaxWidth(640).Padding(12).Gap(10).Role(ui.RoleStatus), colors, m.SystemDark, bg, theme.ContextRadii(c).LG).Border(.5, border).Children(func() {
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
			ui.Text(c, copyText("安装、启用和配置插件", "Install, enable, and configure plugins")).FontSize(theme.InterfaceSize(13)).TextColor(muted).Tooltip(copyText("在这里配置官方插件，安装和管理其他插件。内置插件列表及运行状态可在「设置 → 内置插件」中查看", "Inspect all built-in plugins and runtime states in Settings → Plugins."))
		})
		// .toolbar gap is 8/16 (row/column), the controls share the 32 DIP
		// .addButton recipe (PluginManagerPage.module.css:213-223, 610-618).
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		ui.Row(c).Wrap().Gap(16).Children(func() {
			if s.Tab == "installed" {
				spec := pluginToolbarSpec(widgets.ButtonOutline)
				spec.Disabled = s.Loading || s.Busy || !s.Connected
				spec.Leading = pluginToolbarIcon(pluginRefreshArtwork, fg)
				if widgets.Button(c, colors, copyText("刷新", "Refresh"), spec).Clicked() {
					m.Controller.Refresh(context.Background())
				}
				if s.Install.RequestID != "" {
					spec := pluginToolbarSpec(widgets.ButtonPrimary)
					spec.Disabled = !s.ManagementReady()
					spec.Leading = pluginToolbarIcon(pluginPlusArtwork, theme.Color(colors.ButtonFG))
					if widgets.Button(c, colors, copyText("查看安装任务", "View installation"), spec).Clicked() {
						a.OpenInstall()
					}
				} else {
					addPluginMenu(c, m, s, a)
				}
			}
			// No back control: the original's toolbar ends at the add group and
			// the panel exits through its session/sidebar rows.
		})
	})
}
func loadingRows(c *ui.Context, muted ui.Color) {
	ui.Column(c).FillWidth().Gap(16).Role(ui.RoleStatus).Label(copyText("正在读取插件…", "Loading plugins")).Children(func() {
		ui.Text(c, copyText("正在读取插件…", "Loading plugins…")).TextColor(muted)
		for i := 0; i < 3; i++ {
			ui.Row(c).FillWidth().Gap(14).Padding(8).Children(func() {
				ui.Box(c).Width(48).Height(48).Radius(theme.ContextRadii(c).LG).Background(c.Theme().Surface)
				ui.Column(c).Grow(1).Gap(8).Children(func() {
					ui.Box(c).WidthPercent(35).Height(14).Radius(theme.ContextRadii(c).XS).Background(c.Theme().Surface)
					ui.Box(c).WidthPercent(70).Height(12).Radius(theme.ContextRadii(c).XS).Background(c.Theme().Surface)
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
func artwork(c *ui.Context, ink, border ui.Color, owner *media.Owner, source ...string) {
	artworkSized(c, ink, border, 48, 36, owner, source...)
}
func artworkSized(c *ui.Context, ink, border ui.Color, frame, size float32, owner *media.Owner, source ...string) {
	root := ui.Box(c).Width(frame).Height(frame).Shrink(0).Border(.5, border).Radius(theme.ContextRadii(c).LG)
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
		return packages[i].Official && manager.OfficialRank(packages[i].Name) < manager.OfficialRank(packages[j].Name)
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

	var bundles, official []manager.Package
	offersMore := false
	for _, p := range packages {
		if p.Official {
			offersMore = true
		}
		if p.Official {
			if manager.Pinned(p) {
				official = append(official, p)
			}
			continue
		}
		bundles = append(bundles, p)
	}
	var basic []Contribution
	for _, item := range items {
		if item.Kind == "item" {
			basic = append(basic, item)
		}
	}
	anchor.Children(func() {
		if len(basic) > 0 {
			// PluginManagerPage.module.css: .group gap 8, .cards gap 2, .card
			// margin 0 -8px radius xl — the whole card is the .cardOpen click
			// target and its hover surface reaches 8px past the content.
			ui.Column(c).FillWidth().Gap(8).Children(func() {
				groupTitle(c, copyText("内置插件配置", "Built-in plugin settings"), muted)
				ui.Column(c).FillWidth().Gap(2).Children(func() {
					for _, item := range basic {
						card := ui.Button(c.Key("plugin-item-"+item.Key), "").Label("Plugin item "+item.Title).FillWidth().MinWidth(0).Gap(14).Padding(8).Margin(0, -8).Radius(theme.ContextRadii(c).XL).
							BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Start).Cursor(ui.CursorPointer)
						if card.Hovered() {
							card = card.Background(c.Theme().SurfaceHover)
						}
						if card.Clicked() {
							a.OpenItem(item.Key)
						}
						card.Children(func() {
							artwork(c, muted, border, m.Media, itemArtwork(item.Key))
							ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
								ui.Text(c, item.Title).FillWidth().FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
								ui.Text(c, item.Summary).FillWidth().MaxLines(1).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
							})
						})
					}
				})
			})
		}
		if len(official) > 0 || offersMore {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				ui.Row(c).FillWidth().Children(func() {
					groupTitle(c, copyText("实验性插件", "Experimental plugins"), muted)
					ui.Box(c).Grow(1)
					if offersMore && widgets.Button(c, theme.Resolve(m.ThemeID).Active(m.SystemDark), copyText("更多", "More"), widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).Clicked() {
						a.ShowMore()
					}
				})
				for _, p := range official {
					packageCard(c, m, p, s, a, fg, muted, border)
				}
			})
		}
		if len(bundles) > 0 {
			ui.Column(c).FillWidth().Gap(10).Children(func() {
				groupTitle(c, copyText("已安装插件", "Installed plugins"), muted)
				for _, p := range bundles {
					packageCard(c, m, p, s, a, fg, muted, border)
				}
			})
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
	ui.Text(c, title).Role(ui.RoleHeading).Level(3).FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(22)).FontWeight(500).TextColor(ink)
}

func packageCard(c *ui.Context, m Model, p manager.Package, s manager.State, a manager.Actions, fg, muted, border ui.Color) {
	card := ui.Row(c.Key("plugin-package-" + p.Name)).FillWidth().MinWidth(0).Wrap().Gap(14).Padding(8).Radius(theme.ContextRadii(c).XL).AlignItems(ui.Center)
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
	card.ContextMenu(func(menu *ui.Menu) { packageMenu(menu, m, p, s, a) })
	card.Children(func() {
		artwork(c, muted, border, m.Media, metaIcon(p.Meta))
		ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).Children(func() {
				if ui.Button(c, p.Title(i18n.Locale())).Label(copyText("打开 ", "Open ") + p.Name).MaxWidthPercent(100).MinWidth(0).BorderWidth(0).Padding(0).Justify(ui.Start).Background(ui.Transparent).TextColor(fg).FontSize(theme.InterfaceSize(14)).Clicked() {
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
		if ui.Switch(c, &value).Label(copyText("启用 ", "Enable ") + p.Name).Disabled(s.ActionBusy(p.Name) || p.SwitchLocked()).Changed() {
			a.Enable(p.Name, value)
		}
		menuLabel := copyText("更多操作", "More actions")
		more := ui.Button(c, "").Label(menuLabel).Size(24, 24).Shrink(0).Padding(0).Radius(theme.ContextRadii(c).SM).BorderWidth(0).Background(ui.Transparent).AlignItems(ui.Center).Justify(ui.Center).Tooltip(menuLabel)
		more.Children(func() { ui.Text(c, "⋯").FontSize(theme.InterfaceSize(14)).TextColor(muted) })
		if more.Hovered() {
			more.Background(c.Theme().SurfaceHover)
		}
		more.Menu(func(menu *ui.Menu) { packageMenu(menu, m, p, s, a) })
	})
}

// packageMenu is one plugin's action list, shared by the overflow button and the
// row's context menu.
func packageMenu(menu *ui.Menu, m Model, p manager.Package, s manager.State, a manager.Actions) {
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
		if menu.Item(copyText("卸载", "Remove plugin…")).Disabled(s.ActionBusy(p.Name)).Chosen() {
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
		ui.Text(c, copyText("未安装 / 包不可用", "Not installed / missing files")).FontSize(theme.InterfaceSize(11)).TextColor(muted)
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
		if pkg != nil && subject.Kind == "package" {
			if manager.UpdateAvailable(*pkg) && ui.Button(c, copyText("更新", "Update")).Disabled(s.ActionBusy(pkg.Name)).Clicked() {
				a.Update(pkg.Name)
			}
			if (pkg.Installed || pkg.Removable) && ui.Button(c, copyText("卸载", "Uninstall")).Disabled(s.ActionBusy(pkg.Name) || !pkg.Removable || pkg.ReadOnlyReason != "").Clicked() {
				a.Remove(pkg.Name)
			}
			value := pkg.Enabled
			if ui.Switch(c, &value).Label(copyText("启用 ", "Enable ") + pkg.Name).Disabled(s.ActionBusy(pkg.Name) || pkg.SwitchLocked()).Changed() {
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
		ui.Row(c).FillWidth().Wrap().Gap(8).Children(func() {
			ui.Text(c, item.Title).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
		})
		ui.Text(c, item.Summary).FillWidth().TextColor(muted)
		renderConfig(c, m, subject)
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
		packagePreview(c, pkg.Name, m.Media)
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
			groupTitle(c, copyText("包含的组件", "Plugins in this package"), muted)
			ui.Text(c, partsSummary(pkg.Parts)).FillWidth().FontSize(theme.InterfaceSize(12)).TextColor(muted)
			filter := s.RowFilter
			if len(pkg.Parts) > 10 && ui.TextInput(c, &filter).FillWidth().Label(copyText("筛选组件", "Filter components")).Changed() {
				a.FilterRows(filter)
			}
			query := strings.ToLower(strings.TrimSpace(filter))
			shown := 0
			if len(pkg.Parts) == 0 {
				ui.Text(c, copyText("这个插件包不包含任何组件。", "This package declares no plugin entries.")).TextColor(muted)
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
					artworkSized(c, muted, border, 40, 30, m.Media, icon)
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
							phase = copyText("未运行", "Idle")
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
					if pkg.Enabled && ui.Switch(c, &value).Label(copyText("启用条目 ", "Enable entry ")+r.RowID).Disabled(s.ActionBusy(pkg.Name) || s.ActionBusy(r.EntryID) || r.EntryID == "" || r.ReadOnlyReason != "").Changed() {
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
						source = copyText("已声明，但安装包不可用", "Declared dependency; installed files missing")
					} else {
						source = copyText("未安装", "Not installed")
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
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	// .section is its own column at gap 16; the page column gap does not
	// apply inside it (DiscoverSection.module.css .section).
	ui.Column(c).FillWidth().MinWidth(0).Gap(16).Children(func() {
		// .toolbar: market menu, flex search, label filter, sort menu and the
		// icon-only refresh share one centered row (gap 8).
		ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
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
			// .marketButton caps the ghost trigger at 240 and bleeds -8 left;
			// source management is the menu footer, not a toolbar button.
			marketGhostButton(c, ui.MenuButton(c, selected, func(menu *ui.Menu) {
				for i, id := range ids {
					if menu.Item(labels[i]).Checked(id == d.Selected).Chosen() {
						a.SelectIndex(id)
					}
				}
				menu.Separator()
				if menu.Item(copyText("管理索引源…", "Manage index sources…")).Chosen() {
					a.ToggleSources()
				}
			}), colors).MaxWidth(240).Margin(0, 0, 0, -8).FontWeight(500).
				Label(fmt.Sprintf(copyText("切换插件市场，当前为 %s", "Switch plugin market, currently %s"), selected))
			query := d.TextDraft
			if ui.TextInput(c, &query).Label(copyText("搜索插件", "Search plugins")).Placeholder(copyText("搜索插件", "Search plugins")).Grow(1).MinWidth(120).Changed() {
				a.SearchText(query)
			}
			if d.Listing != nil && len(d.Listing.Metadata.Labels) > 0 {
				filter := copyText("标签", "Labels")
				if len(d.Query.Labels) > 0 {
					filter = fmt.Sprintf(copyText("标签 · %d", "Labels · %d"), len(d.Query.Labels))
				}
				marketGhostButton(c, ui.MenuButton(c, filter, func(menu *ui.Menu) {
					for _, id := range declarationKeys(d.Listing.Metadata.Labels) {
						lid := id
						on := false
						for _, v := range d.Query.Labels {
							on = on || v == lid
						}
						if menu.Item(d.Listing.Metadata.Labels[lid].Title.Resolve(i18n.Locale())).Checked(on).Chosen() {
							q := d.Query
							q.Labels = nil
							for _, v := range d.Query.Labels {
								if v != lid {
									q.Labels = append(q.Labels, v)
								}
							}
							if !on {
								q.Labels = append(q.Labels, lid)
							}
							a.Search(q)
						}
					}
				}), colors)
			}
			if d.Listing != nil {
				sorts := []manager.MarketSort{{Kind: "index"}, {Kind: "published", Direction: "desc"}}
				names := []string{copyText("默认顺序", "Default order"), copyText("发布时间", "Publish date")}
				if strings.TrimSpace(d.Query.Text) != "" {
					sorts = append([]manager.MarketSort{{Kind: "relevance"}}, sorts...)
					names = append([]string{copyText("最佳匹配", "Best match")}, names...)
				}
				for _, id := range declarationKeys(d.Listing.Metadata.SortKeys) {
					meta := d.Listing.Metadata.SortKeys[id]
					sorts = append(sorts, manager.MarketSort{Kind: "key", ID: id, Direction: meta.Order})
					names = append(names, meta.Title.Resolve(i18n.Locale()))
				}
				sortName := names[0]
				for i, sort := range sorts {
					if sort == d.Query.Sort {
						sortName = names[i]
					}
				}
				directed := d.Query.Sort.Kind == "published" || d.Query.Sort.Kind == "key"
				marketGhostButton(c, ui.MenuButton(c, fmt.Sprintf(copyText("排序：%s", "Sort: %s"), sortName), func(menu *ui.Menu) {
					for i := range sorts {
						pick := sorts[i]
						if menu.Item(names[i]).Checked(pick == d.Query.Sort).Chosen() {
							q := d.Query
							q.Sort = pick
							a.Search(q)
						}
					}
					if directed {
						menu.Separator()
						label := copyText("改为升序", "Switch to ascending")
						if d.Query.Sort.Direction == "asc" {
							label = copyText("改为降序", "Switch to descending")
						}
						if menu.Item(label).Chosen() {
							q := d.Query
							if q.Sort.Direction == "asc" {
								q.Sort.Direction = "desc"
							} else {
								q.Sort.Direction = "asc"
							}
							a.Search(q)
						}
					}
				}), colors)
			}
			// The refresh affordance is an icon-only ghost sm button under a
			// tooltip; busy swaps the glyph for a state dot upstream.
			refresh := ui.Button(c.Key("discover-refresh"), "").Label(copyText("刷新", "Refresh")).Tooltip(copyText("刷新", "Refresh")).
				Size(28, 28).Padding(7).BorderWidth(0).Background(ui.Transparent).Radius(theme.ContextRadii(c).SM).
				Disabled(d.Loading || d.Selected == "")
			refresh.Children(func() { ui.Icon(c, marketRefreshIcon).Size(14, 14).TextColor(muted) })
			if refresh.Clicked() {
				a.RefreshDiscover()
			}
		})
		if len(d.Indexes) == 0 {
			ui.Text(c, copyText("尚未配置插件索引。添加一个源以浏览插件。", "No plugin indexes configured. Add a source to browse plugins.")).FillWidth().TextColor(muted)
			if widgets.Button(c, colors, copyText("添加索引源", "Add index source"), widgets.ButtonSpec{Variant: widgets.ButtonPrimary, Size: widgets.ButtonMedium}).Clicked() {
				a.ToggleSources()
			}
			return
		}
		if d.Loading && d.Listing == nil {
			loadingRows(c, muted)
		}
		if d.Listing != nil {
			// .categories is the pill row; the section itself carries no count.
			if keys := declarationKeys(d.Listing.Metadata.Categories); len(keys) > 0 {
				ui.Row(c).Label(copyText("分类", "Categories")).FillWidth().MinWidth(0).Wrap().Gap(6).AlignItems(ui.Center).Children(func() {
					if marketPill(c, copyText("全部", "All"), d.Query.Category == "", colors).Clicked() {
						q := d.Query
						q.Category = ""
						a.Search(q)
					}
					for _, id := range keys {
						cid := id
						label := d.Listing.Metadata.Categories[cid].Title.Resolve(i18n.Locale())
						if marketPill(c, label, d.Query.Category == cid, colors).Clicked() {
							q := d.Query
							q.Category = cid
							a.Search(q)
						}
					}
				})
			}
			// .cards stacks entries at gap 2.
			ui.Column(c).FillWidth().MinWidth(0).Gap(2).Children(func() {
				for _, entry := range d.Listing.Entries {
					e := entry
					// .card bleeds -8 past the section and sits under a hover
					// wash; radius xl, 8 padding, 14 gap, children centered.
					row := ui.Row(c.Key("discover-"+e.Name)).Label("Market entry "+e.Name).Margin(0, -8, 0, -8).
						FillWidth().MinWidth(0).Gap(14).Padding(8).Radius(theme.ContextRadii(c).XL).AlignItems(ui.Center)
					if row.Hovered() {
						row = row.Background(chrome.Color(colors.Hover))
					}
					row.Children(func() {
						artwork(c, muted, border, m.Media, e.Icon)
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							// .titleRow keeps the open affordance and its label
							// capsules on one line.
							ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(8).AlignItems(ui.Center).Children(func() {
								title := e.Label(i18n.Locale())
								if ui.Button(c, title).Label(fmt.Sprintf(copyText("查看 %s", "View %s"), title)).
									Shrink(0).MaxWidthPercent(100).MinWidth(0).BorderWidth(0).Padding(0).Justify(ui.Start).
									Background(ui.Transparent).TextColor(fg).
									FontSize(theme.InterfaceSize(14)).FixedLineHeight(theme.InterfaceSize(20)).FontWeight(500).Clicked() {
									a.OpenEntry(e.Name)
								}
								for _, id := range e.Metadata.Labels {
									if label, ok := d.Listing.Metadata.Labels[id]; ok {
										marketTag(c, label.Title.Resolve(i18n.Locale()), chrome.Color(colors.BorderL4), muted)
									}
								}
							})
							ui.Text(c, e.Description.Resolve(i18n.Locale())).FillWidth().MaxLines(1).FontSize(theme.InterfaceSize(13)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
							ui.Text(c, marketMeta(e)).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).TextColor(muted)
						})
						marketInstall(c, m, e, s, a)
					})
				}
			})
			if len(d.Listing.Entries) == 0 && !d.Loading {
				query := d.Query
				if d.Answered != nil {
					query = *d.Answered
				}
				message := copyText("没有符合条件的插件", "No plugins match.")
				if strings.TrimSpace(query.Text) == "" && query.Category == "" && len(query.Labels) == 0 {
					message = copyText("此索引没有插件", "This index lists no plugins.")
				}
				ui.Text(c, message).FillWidth().TextColor(muted)
			}
			// .more centers an outline sm button under the list.
			if d.Listing.NextOffset != nil {
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					if widgets.Button(c, colors, copyText("显示更多", "Show more"), widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}).
						Disabled(d.Loading || d.MoreLoading || d.Error != "").Clicked() {
						a.More()
					}
				})
			}
		}
		if d.Error != "" && d.Status != "error" {
			errorRow(c, d.Error, a.RetryDiscover)
		}
	})
}

var marketRefreshIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 16 16" fill="none" stroke-width="1"><path d="M14.5001 8C14.5 9.28552 14.1188 10.5422 13.4045 11.611C12.6903 12.6799 11.6752 13.5129 10.4875 14.0049C9.29982 14.4968 7.99295 14.6255 6.73212 14.3747C5.4713 14.124 4.31314 13.505 3.4041 12.596C2.49514 11.687 1.87614 10.5288 1.62537 9.26798C1.37459 8.00716 1.50331 6.70028 1.99525 5.51261C2.48719 4.32494 3.32025 3.30981 4.3891 2.59557C5.45795 1.88134 6.71458 1.50008 8.0001 1.5C9.9001 1.5 11.7001 2.3 13.0001 3.6L14.5001 5.1" stroke="currentColor"/><path d="M14.4999 1.5V5.1H10.8999" stroke="currentColor"/></svg>`))

// marketGhostButton is the toolbar's ghost sm trigger recipe: 28 DIP, 12px/18
// text, radius sm, transparent until hovered (Button.module.css .sm/.ghost).
func marketGhostButton(c *ui.Context, b ui.Element, colors theme.Colors) ui.Element {
	b = b.Height(28).Padding(0, 10).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
		Radius(theme.ContextRadii(c).SM).BorderWidth(0).Background(ui.Transparent)
	if b.Hovered() {
		b = b.Background(chrome.Color(colors.Hover))
	}
	return b
}

// marketPill is the category chip: 24 DIP capsule, 12px/18 text on the layer-2
// fill, active flipped to the ghost-active fill with an inset 1px stroke
// (ui-primitives Pill.module.css .pill/.active).
func marketPill(c *ui.Context, label string, on bool, colors theme.Colors) ui.Element {
	p := chrome.FrostSurface(ui.Button(c, label).Height(24).Padding(0, 8).FontSize(theme.InterfaceSize(12)).FixedLineHeight(theme.InterfaceSize(18)).
		BorderWidth(0), colors, c.Theme().Dark, chrome.Color(colors.Field), 999).TextColor(chrome.Color(colors.Secondary)).Checked(on)
	if on {
		p = p.Background(chrome.Color(colors.Pressed)).TextColor(chrome.Color(colors.FG)).Border(1, chrome.Color(colors.BorderL3))
	} else if p.Hovered() {
		p = p.Background(chrome.Color(colors.Hover))
	}
	return p
}

// marketTag is the read-only outline capsule beside a card title: 1px/8px
// padding, 11px/17px weight-500 text under a 0.5 border-l4 stroke
// (ui-primitives Tag.module.css .tag[data-tone='outline']).
func marketTag(c *ui.Context, text string, border, ink ui.Color) {
	ui.Row(c).Padding(1, 8).Radius(999).Border(0.5, border).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(theme.InterfaceSize(11)).FixedLineHeight(theme.InterfaceSize(17)).FontWeight(500).TextColor(ink)
	})
}

// marketMeta is "{version} · {author} · {date}" with the locale's medium date,
// dropping the author slot when the index did not declare one
// (ui-plugin-market locales entryMeta/entryMetaAuthor).
func marketMeta(e manager.MarketEntry) string {
	v := presentedVersion(e)
	if v == nil {
		return e.Author
	}
	stamp := time.Unix(v.PublishTimestamp, 0).Local().Format("Jan 2, 2006")
	if i18n.Locale() == i18n.LocaleZH {
		stamp = time.Unix(v.PublishTimestamp, 0).Local().Format("2006年1月2日")
	}
	if e.Author == "" {
		return v.Version + " · " + stamp
	}
	return v.Version + " · " + e.Author + " · " + stamp
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

// marketInstall is the cardEnd action (ui-plugin-market client.js EntryAction):
// installed statuses reopen Installed via a ghost sm button, an installable
// entry gets an outline sm 安装, and an incompatible entry keeps the same
// 安装 face under .blocked — dimmed, inert, tooltip naming the running DSH.
func marketInstall(c *ui.Context, m Model, e manager.MarketEntry, s manager.State, a manager.Actions) {
	colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
	title := e.Label(i18n.Locale())
	if e.Installation.Status != "not-installed" {
		label := copyText("已安装", "Installed")
		if e.Installation.Status == "update-available" {
			label = copyText("有新版本", "Update available")
		} else if e.Installation.Status == "provided" {
			label = copyText("随 DSH 提供", "Provided by DSH")
		}
		if widgets.Button(c, colors, label, widgets.ButtonSpec{Variant: widgets.ButtonGhost, Size: widgets.ButtonSmall}).
			Label(fmt.Sprintf(copyText("在已安装中查看 %s", "View %s in Installed"), title)).Clicked() {
			a.OpenPackage(e.Name)
		}
		return
	}
	version := presentedVersion(e)
	spec := widgets.ButtonSpec{Variant: widgets.ButtonOutline, Size: widgets.ButtonSmall}
	install := fmt.Sprintf(copyText("安装 %s", "Install %s"), title)
	if version == nil || (!version.Compatibility.Compatible && !version.Compatibility.Exempted) {
		blocked := widgets.Button(c, colors, copyText("安装", "Install"), spec).Disabled(true).Opacity(0.5).Label(install)
		if version != nil && version.Compatibility.RuntimeVersion != "" {
			blocked = blocked.Tooltip(fmt.Sprintf(copyText("没有与当前 DSH %s 兼容的版本", "No version is compatible with DSH %s"), version.Compatibility.RuntimeVersion))
		}
		return
	}
	if widgets.Button(c, colors, copyText("安装", "Install"), spec).Label(install).Clicked() {
		a.InstallVersion(e.Name, version.Version)
	}
}
func marketDetail(c *ui.Context, m Model, e manager.MarketEntry, s manager.State, a manager.Actions, fg, muted ui.Color) {
	if ui.Button(c, copyText("返回发现", "Back to Discover")).Clicked() {
		a.CloseEntry()
	}
	ui.Text(c, e.Label(i18n.Locale())).FontSize(theme.InterfaceSize(20)).FontWeight(500).TextColor(fg)
	ui.Text(c, e.Name).FillWidth().Selectable().TextColor(muted)
	ui.Text(c, e.Description.Resolve(i18n.Locale())).FillWidth().TextColor(fg)
	marketInstall(c, m, e, s, a)
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

// sourcesEditor is the index-source body of the sources card: the reference
// modal names the section in its header, so the body carries no second title or
// nested panel of its own.
func sourcesEditor(c *ui.Context, s manager.State, a manager.Actions, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(10).Children(func() {
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
	parts := []string{fmt.Sprintf(copyText("共 %d 个", "Total %d"), len(rows))}
	if running > 0 {
		parts = append(parts, fmt.Sprintf(copyText("%d 运行中", "Running %d"), running))
	}
	if off > 0 {
		parts = append(parts, fmt.Sprintf(copyText("%d 已停用", "Off %d"), off))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf(copyText("%d 异常", "Failed %d"), failed))
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

// Package pluginmanager renders the native profile Plugins page. It is not
// the desktop curated marketplace registered in Settings.
//
// The page is drawn with internal/ui/coss on the design tokens. It reads the
// controller's published version once per frame at View; the installed
// list, each package row and the discover listing are memo regions keyed by
// that version's revision, so a frame in which nothing changed keeps them.
package pluginmanager

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"whaleisle/internal/desktop/media"
	"whaleisle/internal/errcopy"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	marketui "whaleisle/internal/ui/market"
	"whaleisle/internal/ui/pagekit"
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

// page is one frame's reading of the controller: the published state, its
// revision and the actions bound to its owner, with the page's media watch.
type page struct {
	m     Model
	s     manager.State
	rev   uint64
	a     manager.Actions
	watch *pagekit.MediaWatch
	// epoch moves when the model's Controller is swapped, so a memo keyed
	// on the same revision cannot keep another controller's rows.
	epoch uint64
	// mediaEpoch moves when the model's Media lease is swapped: pictures a
	// released owner was waiting on never arrive, so a kept region must
	// build again to ask the new lease for them.
	mediaEpoch uint64
}

// controllerEpoch is the retained record behind page.epoch.
type controllerEpoch struct {
	last *manager.Controller
	n    uint64
}

// mediaEpoch is the retained record behind page.mediaEpoch.
type mediaEpoch struct {
	last *media.Owner
	n    uint64
}

// version is the memo version of a region showing s: the revision, the
// controller's identity, the design tokens' generation, the locale, the
// contributions and the pictures that arrived.
func (p *page) version() uint64 {
	v := pagekit.NewVersion().Uint(p.rev).Uint(p.epoch).Uint(p.mediaEpoch).Uint(design.Revision()).Uint(p.watch.Epoch()).Bool(p.m.OnOpenURL != nil).Bool(p.m.RenderConfiguration != nil)
	for _, item := range p.m.Contributions {
		v = v.String(item.Kind).String(item.Key).String(item.Title).String(item.Summary).Uint(uint64(item.Order))
	}
	return v.Key()
}

func View(c *ui.Context, m Model) {
	if m.Controller == nil {
		return
	}
	ver := m.Controller.Snapshot()
	if !ver.Value.Loading {
		m.Controller.Ensure()
		ver = m.Controller.Snapshot()
	}
	root := ui.Column(c).Key("plugin-manager-page").Grow(1).FillWidth().MinHeight(0).MinWidth(0)
	epoch := ui.Local(root, "controller-epoch", func() controllerEpoch { return controllerEpoch{} })
	if epoch.last != m.Controller {
		epoch.last, epoch.n = m.Controller, epoch.n+1
	}
	watch := ui.Local(root, "media-watch", func() *pagekit.MediaWatch { return &pagekit.MediaWatch{} })
	lease := ui.Local(root, "media-epoch", func() mediaEpoch { return mediaEpoch{} })
	if lease.last != m.Media {
		lease.last, lease.n = m.Media, lease.n+1
		// Handles the released owner was waiting on stay Pending forever;
		// a fresh watch stops counting the dead ones.
		*watch = &pagekit.MediaWatch{}
	}
	p := &page{m: m, s: ver.Value, rev: ver.Rev, a: m.Controller.Actions(ver.Value.Owner), epoch: epoch.n, mediaEpoch: lease.n}
	s, a := p.s, p.a
	p.watch = *watch
	scroll := ui.Local(root, "scroll", func() *ui.ScrollState { return &ui.ScrollState{} })
	root.Children(func() {
		pagekit.Page(c, pagekit.PageOptions{Scroll: *scroll}, func() {
			detail := s.Tab == "installed" && s.View.Kind != "list" && s.View.Kind != "more"
			if !detail && s.View.Kind != "more" {
				pageHeader(c, p)
				// The tab row shows once the Host answers pluginMarket/
				// listIndexes (the original mounts it only while a
				// plugins.tab contribution exists), and stays while Discover
				// itself is open.
				if s.Tab == "discover" || s.Discover.Probe == "ready" || s.Discover.Status == "ready" {
					pageTabs(c, p)
				}
			}
			if s.Notice != "" && widgets.AckLive(c, "plugin-notice", s.Notice, widgets.StatusDuration) {
				notice(c, s.Notice)
			}
			if m.CreatorNotice != nil {
				m.CreatorNotice(c)
			}
			if s.Tab == "discover" {
				discoverView(c, p)
				return
			}
			if s.Loading && len(s.Packages) == 0 {
				pagekit.SkeletonRows(c, 3, copyText("正在读取插件…", "Loading plugins"))
			}
			switch s.Status {
			case "disconnected":
				pagekit.Note(c, coss.AlertWarning, copyText("运行时未连接", "Runtime disconnected"), copyText("连接后即可管理此 profile 的插件。", "Connect to manage this profile's plugins."), nil)
			case "unavailable":
				pagekit.Note(c, coss.AlertInfo, copyText("无法管理插件", "Plugins cannot be managed"), copyText("本部署没有可管理的 profile，无法安装或启停插件。", "This runtime has no managed profile. Plugin installation and switching are unavailable."), nil)
			}
			if s.Error != "" {
				pagekit.Fault(c, copyText("插件列表加载失败", "Could not load plugins"), s.Error, copyText("重试", "Retry"), func() { m.Controller.Refresh(context.Background()) })
			}
			if detail {
				detailView(c, p)
				return
			}
			if s.View.Kind == "more" {
				ui.Column(c).FillWidth().Gap(design.Sp(2)).Children(func() {
					if pagekit.Back(c, copyText("返回插件", "Back to plugins")) {
						a.Back()
					}
					pagekit.Title(c, copyText("实验性插件", "Experimental plugins"))
				})
			}
			installedView(c, p)
		})
	})
	switch {
	case s.Confirmation != nil:
		confirmationDialog(c, p)
	case s.Discover.Choice != nil:
		marketChoiceDialog(c, p)
	case s.Install.Open:
		installDialog(c, p)
	case s.Discover.SourcesOpen:
		sourcesDialog(c, p)
	}
	if s.PreferenceError != "" {
		ui.Overlay(c.Key("plugin-registry-preference-error"), func() {
			ui.Column(c).Absolute().Bottom(design.Sp(6)).Left(design.Sp(6)).Right(design.Sp(6)).MaxWidth(640).Children(func() {
				coss.Alert(c, copyText("无法读取记住的安装源", "Could not read the remembered registry"), errcopyText(s.PreferenceError), coss.AlertOptions{Variant: coss.AlertError, Action: func() {
					if coss.Button(c, copyText("关闭", "Close"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
						a.DismissPreferenceError()
					}
				}})
			})
		})
	}
}

// notice shows the controller's last outcome inline: authored outcomes in
// their own words, anything else as the sentence for the failure it is.
func notice(c *ui.Context, raw string) {
	if text, variant, ok := outcomeText(raw); ok {
		pagekit.Note(c, variant, text, "", nil)
		return
	}
	pagekit.Fault(c, "", raw, "", nil)
}

// outcomeText localizes the controller's authored outcomes (changeMessage
// and its fixed notices); ok is false for anything else, which is a raw
// failure. An "applied" outcome that carries an error's code and diagnostic
// comes back as a warning holding the classified sentence: the code and
// diagnostic themselves never reach the screen.
func outcomeText(raw string) (string, coss.AlertVariant, bool) {
	head, rest, _ := strings.Cut(raw, " · ")
	switch {
	case head == "applied" && rest == "":
		return copyText("已应用。", "Applied."), coss.AlertSuccess, true
	case head == "applied" && outcomeCarriesError(rest):
		return copyText("已应用，但出现错误：", "Applied, with an error: ") + outcomeErrorSentence(rest), coss.AlertWarning, true
	case head == "applied":
		return copyText("已应用，但有提示：", "Applied with warnings: ") + rest, coss.AlertSuccess, true
	case strings.HasPrefix(head, "Restart required: "):
		return copyText("已保存，重启运行时后生效。", "Saved. Restart the runtime to apply it."), coss.AlertSuccess, true
	case raw == "Installation cancelled.":
		return copyText("已取消安装。", "Installation cancelled."), coss.AlertSuccess, true
	case raw == "The plugin inventory changed. Refresh and review the action again.":
		return copyText("插件清单已变化，请刷新后重新确认操作。", "The plugin inventory changed. Refresh and review the action again."), coss.AlertSuccess, true
	}
	return "", 0, false
}

// outcomeCarriesError reports whether the extras changeMessage appends to
// "applied" start with a failure's code — a slug with no spaces — rather
// than the Host's authored warnings.
func outcomeCarriesError(rest string) bool {
	code, _, _ := strings.Cut(rest, " · ")
	return !strings.Contains(code, " ")
}

// outcomeErrorSentence is the classified sentence for the code and
// diagnostic changeMessage flattened after "applied", or errcopy's generic
// one when the lookup cannot name the failure.
func outcomeErrorSentence(rest string) string {
	code, tail, _ := strings.Cut(rest, " · ")
	if s := errcopySentenceFor(code, tail); s != "" {
		return s
	}
	return errcopy.CauseSentenceIn("", errcopy.CauseUnknown, "")
}

func pageHeader(c *ui.Context, p *page) {
	s, a := p.s, p.a
	pagekit.Header(c, copyText("插件", "Plugins"), copyText("安装、启用和配置插件", "Install, enable, and configure plugins"), func() {
		if s.Tab != "installed" {
			return
		}
		if coss.Button(c, copyText("刷新", "Refresh"), coss.ButtonOptions{Variant: coss.ButtonOutline, Icon: coss.IconRefresh, Disabled: s.Loading || s.Busy || !s.Connected}).Clicked() {
			p.m.Controller.Refresh(context.Background())
		}
		if s.Install.RequestID != "" {
			if coss.Button(c, copyText("查看安装任务", "View installation"), coss.ButtonOptions{Icon: coss.IconDownload, Disabled: !s.ManagementReady()}).Clicked() {
				a.OpenInstall()
			}
			return
		}
		addPluginMenu(c, p)
	})
}

func pageTabs(c *ui.Context, p *page) {
	selected := 0
	if p.s.Tab == "discover" {
		selected = 1
	}
	tabs := coss.Tabs(c, &selected, []string{copyText("已安装", "Installed"), copyText("发现", "Discover")}, coss.TabsOptions{Label: copyText("插件视图", "Plugin views"), Key: "plugin-tabs"})
	if tabs.Changed() {
		p.a.SelectTab([]string{"installed", "discover"}[selected])
	}
}

// artwork is a package's 40 DIP tile: its declared icon, the default
// artwork, or the puzzle glyph while neither is ready.
func artwork(c *ui.Context, p *page, source string) {
	artworkSized(c, p, design.Sp(10), design.Sp(7), source)
}

func artworkSized(c *ui.Context, p *page, frame, size float32, source string) {
	t := design.For(c)
	ui.Box(c).Size(frame, frame).Shrink(0).Center().Radius(design.RadiusLG).Border(design.BorderWidth, t.Border).Background(t.Card).Clip().Children(func() {
		for _, src := range [2]string{source, builtInArtwork("default")} {
			if src == "" {
				continue
			}
			shown, h := marketui.PluginImageHandle(c, src, size, size, p.m.Media)
			if shown {
				return
			}
			p.watch.Track(h)
		}
		ui.Icon(c, coss.IconPuzzle.SVG()).Size(design.IconSM, design.IconSM).TextColor(t.MutedForeground)
	})
}

func installedView(c *ui.Context, p *page) {
	s, a, m := p.s, p.a, p.m
	version := p.version()
	list := ui.Column(c).Key("plugins-installed").FillWidth().MinWidth(0).Gap(design.Sp(6)).Memo(version)
	list.Children(func() {
		packages := make([]manager.Package, 0, len(s.Packages))
		for _, pkg := range s.Packages {
			if manager.Listed(pkg) {
				packages = append(packages, pkg)
			}
		}
		sort.SliceStable(packages, func(i, j int) bool {
			if packages[i].Official != packages[j].Official {
				return packages[i].Official
			}
			return packages[i].Official && manager.OfficialRank(packages[i].Name) < manager.OfficialRank(packages[j].Name)
		})
		if s.View.Kind == "more" {
			rows(c, func() {
				for _, pkg := range packages {
					if pkg.Official {
						packageRow(c, p, pkg, version)
					}
				}
			})
			return
		}
		items := append([]Contribution{}, m.Contributions...)
		sort.SliceStable(items, func(i, j int) bool { return items[i].Order < items[j].Order })
		var bundles, official []manager.Package
		offersMore := false
		for _, pkg := range packages {
			if pkg.Official {
				offersMore = true
				if manager.Pinned(pkg) {
					official = append(official, pkg)
				}
				continue
			}
			bundles = append(bundles, pkg)
		}
		var basic []Contribution
		for _, item := range items {
			if item.Kind == "item" {
				basic = append(basic, item)
			}
		}
		if len(basic) > 0 {
			pagekit.Section(c, copyText("内置插件配置", "Built-in plugin settings"), nil, func() {
				rows(c, func() {
					for i, item := range basic {
						if i > 0 {
							rowSeparator(c)
						}
						itemRow(c, p, item)
					}
				})
			})
		}
		if len(official) > 0 || offersMore {
			var more func()
			if offersMore {
				more = func() {
					if coss.Button(c, copyText("更多", "More"), coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, TrailingIcon: coss.IconChevronRight}).Clicked() {
						a.ShowMore()
					}
				}
			}
			pagekit.Section(c, copyText("实验性插件", "Experimental plugins"), more, func() {
				if len(official) == 0 {
					return
				}
				rows(c, func() {
					for i, pkg := range official {
						if i > 0 {
							rowSeparator(c)
						}
						packageRow(c, p, pkg, version)
					}
				})
			})
		}
		if len(bundles) > 0 {
			pagekit.Section(c, groupCount(copyText("已安装插件", "Installed plugins"), len(bundles)), nil, func() {
				rows(c, func() {
					for i, pkg := range bundles {
						if i > 0 {
							rowSeparator(c)
						}
						packageRow(c, p, pkg, version)
					}
				})
			})
		}
		if len(packages) == 0 && s.Status == "ready" {
			coss.Empty(c, coss.EmptyOptions{Icon: coss.IconPuzzle, Title: copyText("还没有安装任何插件", "No plugins installed yet"),
				Description: copyText("从发现页或包名添加一个插件。", "Add one from Discover or by its package name.")}, nil)
		}
	})
}

// rows is a list card: the rows fn builds sit in one bordered card.
func rows(c *ui.Context, fn func()) {
	t := design.For(c)
	ui.Column(c).FillWidth().MinWidth(0).Radius(design.RadiusXL).Border(design.BorderWidth, t.Border).Background(t.Card).Clip().Children(fn)
}

// rowSeparator divides two rows of a list card.
func rowSeparator(c *ui.Context) {
	coss.Separator(c, coss.Horizontal)
}

// rowFrame is one list row: 12×16 padding, hover wash, children centered.
func rowFrame(e ui.Element) ui.Element {
	return e.FillWidth().MinWidth(0).Gap(design.Sp(3)).Padding(design.Sp(3), design.Sp(4)).AlignItems(ui.Center)
}

// groupCount names a group with the number of plugins it holds.
func groupCount(label string, count int) string {
	if count == 0 {
		return label
	}
	return fmt.Sprintf("%s (%d)", label, count)
}

// itemRow is one built-in configuration entry: the whole row opens it.
func itemRow(c *ui.Context, p *page, item Contribution) {
	t := design.For(c)
	row := rowFrame(ui.ButtonBase(c.Key("plugin-item-" + item.Key))).Label("Plugin item " + item.Title).Justify(ui.Start).Cursor(ui.CursorPointer)
	if row.Hovered() {
		row.Background(t.Accent)
	}
	if row.Clicked() {
		p.a.OpenItem(item.Key)
	}
	row.Children(func() {
		artwork(c, p, itemArtwork(item.Key))
		ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
			pagekit.Strong(c, item.Title).FillWidth()
			pagekit.Muted(c, item.Summary).FillWidth().MaxLines(1)
		})
		ui.Icon(c, coss.IconChevronRight.SVG()).Size(design.IconSM, design.IconSM).Shrink(0).TextColor(t.MutedForeground)
	})
}

// packageRow is one installed package: artwork, name and badges, summary
// and source line, the enable switch and the actions menu. It is a memo
// region of its own, so the pointer over one row rebuilds that row alone.
func packageRow(c *ui.Context, p *page, pkg manager.Package, version uint64) {
	s, a := p.s, p.a
	t := design.For(c)
	row := rowFrame(ui.Row(c.Key("plugin-package-" + pkg.Name))).Wrap().Label(pkg.Name)
	if row.Hovered() {
		row.Background(t.Accent)
	}
	if s.Highlight == pkg.Name && time.Now().UnixNano() < s.HighlightUntil {
		row.Background(t.Primary.Alpha(0.08))
		seen := ui.Local(row, "install-highlight", func() int64 { return 0 })
		if *seen != s.HighlightUntil {
			*seen = s.HighlightUntil
			row.ScrollIntoView()
		}
		c.After(time.Until(time.Unix(0, s.HighlightUntil)))
	}
	if id := coss.ContextMenu(c, row, packageMenu(pkg, s), coss.MenuOptions{Label: copyText("插件操作", "Plugin actions")}); id != "" {
		runPackageAction(a, pkg, id)
	}
	row.Memo(version).Children(func() {
		artwork(c, p, metaIcon(pkg.Meta))
		ui.Column(c).Grow(1).Basis(0).MinWidth(200).Gap(design.Sp(0.5)).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				open := ui.ButtonBase(c).Label(copyText("打开 ", "Open ") + pkg.Name).MinWidth(0).Shrink(1).Justify(ui.Start).Cursor(ui.CursorPointer)
				hover := open.Hovered()
				if open.Clicked() {
					a.OpenPackage(pkg.Name)
				}
				open.Children(func() {
					title := pagekit.Strong(c, pkg.Title(i18n.Locale())).SingleLine()
					if hover {
						title.Underline()
					}
				})
				packageTags(c, pkg)
			})
			pagekit.Muted(c, pkg.Summary(i18n.Locale())).FillWidth().MinWidth(0).MaxLines(2)
			if meta := packageMetaLine(pkg); meta != "" {
				pagekit.Caption(c, meta).FillWidth().MinWidth(0).SingleLine().Tooltip(meta)
			}
		})
		ui.Row(c).Shrink(0).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			value := pkg.Enabled
			if coss.Switch(c, &value, "", coss.CheckOptions{Label: copyText("启用 ", "Enable ") + pkg.Name, Disabled: s.ActionBusy(pkg.Name) || pkg.SwitchLocked()}).Changed() {
				a.Enable(pkg.Name, value)
			}
			moreMenu(c, pkg, s, a)
		})
	})
}

// moreMenu is a row's ⋯ menu button.
func moreMenu(c *ui.Context, pkg manager.Package, s manager.State, a manager.Actions) {
	label := copyText("更多操作", "More actions")
	trigger := coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonIconSM, Icon: coss.IconEllipsis, Label: label}).Tooltip(label)
	open := ui.Local(trigger, "open", func() bool { return false })
	if id := coss.Menu(c, trigger, open, packageMenu(pkg, s), coss.MenuOptions{Align: coss.AlignEnd, Label: label}); id != "" {
		runPackageAction(a, pkg, id)
	}
}

// packageMenu is one plugin's action list, shared by the overflow button and
// the row's context menu.
func packageMenu(p manager.Package, s manager.State) []coss.MenuItem {
	busy := s.ActionBusy(p.Name)
	items := []coss.MenuItem{
		{ID: "open", Label: copyText("打开详情", "Open details")},
		{Kind: coss.MenuSeparator},
		{ID: "disable", Label: copyText("停用", "Disable"), Disabled: !p.Enabled || busy || p.SwitchLocked()},
		{ID: "enable", Label: copyText("启用", "Enable"), Disabled: p.Enabled || busy || p.SwitchLocked()},
	}
	if manager.UpdateAvailable(p) {
		items = append(items, coss.MenuItem{ID: "update", Label: copyText("更新", "Update"), Icon: coss.IconDownload, Disabled: busy})
	}
	if p.Removable {
		items = append(items, coss.MenuItem{Kind: coss.MenuSeparator},
			coss.MenuItem{ID: "remove", Label: copyText("卸载", "Remove plugin…"), Icon: coss.IconTrash, Destructive: true, Disabled: busy})
	}
	return items
}

func runPackageAction(a manager.Actions, p manager.Package, id string) {
	switch id {
	case "open":
		a.OpenPackage(p.Name)
	case "disable":
		a.Enable(p.Name, false)
	case "enable":
		a.Enable(p.Name, true)
	case "update":
		a.Update(p.Name)
	case "remove":
		a.Remove(p.Name)
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
// already distinguishes: bundled, official on-demand, market, or local.
func packageSource(p manager.Package) string {
	switch {
	case p.Official && p.Availability == "missing" && !p.Installed:
		return copyText("官方 · 按需安装", "Official · On-demand")
	case p.Official && (p.Availability == "profile" || p.Availability == "missing"):
		return copyText("官方", "Official")
	case p.Official:
		return copyText("内置", "Built-in")
	case p.InstallTarget != nil:
		return copyText("市场", "Market")
	default:
		return copyText("本地", "Local")
	}
}

// packageTags are a package's status badges.
func packageTags(c *ui.Context, p manager.Package) {
	if strings.HasPrefix(p.Name, "@deepseek-ai/dsh-experimental-") {
		coss.Badge(c, copyText("实验性", "Experimental"), coss.BadgeOptions{Variant: coss.BadgeSecondary})
	}
	if p.Error != nil {
		coss.Badge(c, copyText("异常", "Problem"), coss.BadgeOptions{Variant: coss.BadgeError})
	}
	if p.Availability == "missing" {
		if p.Installed {
			coss.Badge(c, copyText("包不可用", "Package unavailable"), coss.BadgeOptions{Variant: coss.BadgeWarning})
		} else {
			coss.Badge(c, copyText("未安装", "Not installed"), coss.BadgeOptions{Variant: coss.BadgeSecondary})
		}
	}
	if manager.UpdateAvailable(p) {
		coss.Badge(c, copyText("可更新", "Update available"), coss.BadgeOptions{Variant: coss.BadgeInfo})
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
	pagekit.Muted(c, copyText("此插件没有注册可用的本地配置页面。", "This plugin has no registered native configuration page.")).FillWidth()
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

func detailLink(c *ui.Context, m Model, raw string) {
	if raw == "" {
		return
	}
	if m.OnOpenURL != nil && (strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://")) {
		if coss.Button(c, raw, coss.ButtonOptions{Variant: coss.ButtonLink, Size: coss.ButtonSM, TrailingIcon: coss.IconExternalLink}).MaxWidthPercent(100).Clicked() {
			m.OnOpenURL(raw)
		}
		return
	}
	pagekit.Code(c, raw).FillWidth().Selectable()
}

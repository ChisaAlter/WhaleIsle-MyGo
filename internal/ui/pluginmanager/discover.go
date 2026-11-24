package pluginmanager

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/pagekit"
)

// discoverView is the Host market tab: the market, search, label and sort
// toolbar, the category toggles, and the listing (or one entry's detail).
func discoverView(c *ui.Context, p *page) {
	s, a := p.s, p.a
	d := s.Discover
	if d.Status == "idle" || d.Status == "loading" {
		pagekit.SkeletonRows(c, 3, copyText("正在读取插件…", "Loading plugins"))
		return
	}
	if d.Status == "error" {
		pagekit.Fault(c, copyText("插件市场加载失败", "Could not load the plugin market"), d.IndexesError, copyText("重试", "Retry"), a.ReloadIndexes)
		return
	}
	if d.SelectedEntry != "" && d.Listing != nil {
		if d.Opened != nil {
			marketDetail(c, p, *d.Opened)
			return
		}
		for _, entry := range d.Listing.Entries {
			if entry.Name == d.SelectedEntry {
				marketDetail(c, p, entry)
				return
			}
		}
	}
	ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(4)).Children(func() {
		discoverToolbar(c, p)
		if len(d.Indexes) == 0 {
			coss.Empty(c, coss.EmptyOptions{Icon: coss.IconPuzzle, Title: copyText("尚未配置插件索引", "No plugin indexes configured"),
				Description: copyText("添加一个索引源以浏览插件。", "Add a source to browse plugins.")}, func() {
				if coss.Button(c, copyText("添加索引源", "Add index source"), coss.ButtonOptions{Icon: coss.IconPlus}).Clicked() {
					a.ToggleSources()
				}
			})
			return
		}
		if d.Loading && d.Listing == nil {
			pagekit.SkeletonRows(c, 3, copyText("正在读取插件…", "Loading plugins"))
		}
		if d.Listing != nil {
			categories(c, p)
			listing(c, p)
		}
		if d.Error != "" && d.Status != "error" {
			pagekit.Fault(c, copyText("插件列表加载失败", "Could not load plugins"), d.Error, copyText("重试", "Retry"), a.RetryDiscover)
		}
	})
}

// toolbarMenu is a toolbar menu button: an outline small trigger with a
// chevron and its menu, closing after a choice.
func toolbarMenu(c *ui.Context, key, label, name string, items []coss.MenuItem) string {
	trigger := coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, TrailingIcon: coss.IconChevronDown, Key: key}).MaxWidth(240)
	if name != "" {
		trigger.Label(name)
	}
	open := ui.Local(trigger, "open", func() bool { return false })
	id := coss.Menu(c, trigger, open, items, coss.MenuOptions{Label: label})
	if id != "" {
		*open = false
	}
	return id
}

func discoverToolbar(c *ui.Context, p *page) {
	s, a := p.s, p.a
	d := s.Discover
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
		selected := ""
		items := make([]coss.MenuItem, 0, len(d.Indexes)+2)
		for _, index := range d.Indexes {
			label := index.Title.Resolve(i18n.Locale())
			if label == "" {
				label = index.URL
				if parsed, err := url.Parse(index.URL); err == nil && parsed.Host != "" {
					label = parsed.Host
				}
			}
			// Duplicate localized titles stay distinct choices.
			for _, other := range d.Indexes {
				if other.URL != index.URL && other.Title.Resolve(i18n.Locale()) == index.Title.Resolve(i18n.Locale()) {
					label += " · " + index.URL
					break
				}
			}
			if index.URL == d.Selected {
				selected = label
			}
			on := index.URL == d.Selected
			items = append(items, coss.MenuItem{Kind: coss.MenuCheckbox, ID: "index:" + index.URL, Label: label, Checked: &on})
		}
		items = append(items, coss.MenuItem{Kind: coss.MenuSeparator}, coss.MenuItem{ID: "sources", Label: copyText("管理索引源…", "Manage index sources…")})
		switch id := toolbarMenu(c, "discover-market", selected, fmt.Sprintf(copyText("切换插件市场，当前为 %s", "Switch plugin market, currently %s"), selected), items); {
		case id == "sources":
			a.ToggleSources()
		case strings.HasPrefix(id, "index:"):
			a.SelectIndex(strings.TrimPrefix(id, "index:"))
		}
		query := d.TextDraft
		search := ui.Row(c).Grow(1).Basis(0).MinWidth(160)
		search.Children(func() {
			if coss.InputGroup(c, &query, coss.InputGroupOptions{InputOptions: coss.InputOptions{Label: copyText("搜索插件", "Search plugins"), Placeholder: copyText("搜索插件", "Search plugins")},
				Start: func() { coss.AddonIcon(c, coss.IconSearch) }}).Changed() {
				a.SearchText(query)
			}
		})
		if d.Listing != nil && len(d.Listing.Metadata.Labels) > 0 {
			labelFilter(c, p)
		}
		if d.Listing != nil {
			sortMenu(c, p)
		}
		label := copyText("刷新", "Refresh")
		if coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonIconSM, Icon: coss.IconRefresh, Label: label, Key: "discover-refresh",
			Disabled: d.Loading || d.Selected == ""}).Tooltip(label).Clicked() {
			a.RefreshDiscover()
		}
	})
}

func labelFilter(c *ui.Context, p *page) {
	d := p.s.Discover
	filter := copyText("标签", "Labels")
	if len(d.Query.Labels) > 0 {
		filter = fmt.Sprintf(copyText("标签 · %d", "Labels · %d"), len(d.Query.Labels))
	}
	keys := declarationKeys(d.Listing.Metadata.Labels)
	items := make([]coss.MenuItem, 0, len(keys))
	for _, id := range keys {
		on := false
		for _, v := range d.Query.Labels {
			on = on || v == id
		}
		items = append(items, coss.MenuItem{Kind: coss.MenuCheckbox, ID: id, Label: d.Listing.Metadata.Labels[id].Title.Resolve(i18n.Locale()), Checked: &on})
	}
	id := toolbarMenu(c, "discover-labels", filter, "", items)
	if id == "" {
		return
	}
	q := d.Query
	q.Labels = nil
	had := false
	for _, v := range d.Query.Labels {
		if v != id {
			q.Labels = append(q.Labels, v)
		} else {
			had = true
		}
	}
	if !had {
		q.Labels = append(q.Labels, id)
	}
	p.a.Search(q)
}

func sortMenu(c *ui.Context, p *page) {
	d := p.s.Discover
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
	items := make([]coss.MenuItem, 0, len(sorts)+2)
	for i, sort := range sorts {
		on := sort == d.Query.Sort
		if on {
			sortName = names[i]
		}
		items = append(items, coss.MenuItem{Kind: coss.MenuCheckbox, ID: fmt.Sprint(i), Label: names[i], Checked: &on})
	}
	directed := d.Query.Sort.Kind == "published" || d.Query.Sort.Kind == "key"
	if directed {
		label := copyText("改为升序", "Switch to ascending")
		if d.Query.Sort.Direction == "asc" {
			label = copyText("改为降序", "Switch to descending")
		}
		items = append(items, coss.MenuItem{Kind: coss.MenuSeparator}, coss.MenuItem{ID: "direction", Label: label})
	}
	id := toolbarMenu(c, "discover-sort", fmt.Sprintf(copyText("排序：%s", "Sort: %s"), sortName), "", items)
	if id == "" {
		return
	}
	q := d.Query
	if id == "direction" {
		if q.Sort.Direction == "asc" {
			q.Sort.Direction = "desc"
		} else {
			q.Sort.Direction = "asc"
		}
	} else {
		for i := range sorts {
			if fmt.Sprint(i) == id {
				q.Sort = sorts[i]
			}
		}
	}
	p.a.Search(q)
}

// categories are the listing's category toggles.
func categories(c *ui.Context, p *page) {
	d := p.s.Discover
	keys := declarationKeys(d.Listing.Metadata.Categories)
	if len(keys) == 0 {
		return
	}
	ui.Row(c).Label(copyText("分类", "Categories")).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(1.5)).AlignItems(ui.Center).Children(func() {
		pill := func(id, label string) {
			on := d.Query.Category == id
			if coss.Toggle(c, &on, label, coss.ToggleOptions{Variant: coss.ToggleOutline, Size: coss.ToggleSM, Key: "category-" + id}).Changed() && on {
				q := d.Query
				q.Category = id
				p.a.Search(q)
			}
		}
		pill("", copyText("全部", "All"))
		for _, id := range keys {
			pill(id, d.Listing.Metadata.Categories[id].Title.Resolve(i18n.Locale()))
		}
	})
}

// listing is the entries of the current answer, a memo region of the
// published revision.
func listing(c *ui.Context, p *page) {
	s, a := p.s, p.a
	d := s.Discover
	version := p.version()
	ui.Column(c).Key("discover-listing").FillWidth().MinWidth(0).Gap(design.Sp(4)).Memo(version).Children(func() {
		if len(d.Listing.Entries) > 0 {
			rows(c, func() {
				for i, entry := range d.Listing.Entries {
					if i > 0 {
						rowSeparator(c)
					}
					discoverRow(c, p, entry, version)
				}
			})
		}
		if len(d.Listing.Entries) == 0 && !d.Loading {
			query := d.Query
			if d.Answered != nil {
				query = *d.Answered
			}
			title := copyText("没有符合条件的插件", "No plugins match.")
			if strings.TrimSpace(query.Text) == "" && query.Category == "" && len(query.Labels) == 0 {
				title = copyText("此索引没有插件", "This index lists no plugins.")
			}
			coss.Empty(c, coss.EmptyOptions{Icon: coss.IconSearch, Title: title}, nil)
		}
		if d.Listing.NextOffset != nil {
			ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
				if coss.Button(c, copyText("显示更多", "Show more"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM,
					Loading: d.MoreLoading, Disabled: d.Loading || d.Error != ""}).Clicked() {
					a.More()
				}
			})
		}
	})
}

func discoverRow(c *ui.Context, p *page, e manager.MarketEntry, version uint64) {
	t := design.For(c)
	d := p.s.Discover
	row := rowFrame(ui.Row(c.Key("discover-" + e.Name))).Label("Market entry " + e.Name)
	if row.Hovered() {
		row.Background(t.Accent)
	}
	row.Memo(version).Children(func() {
		artwork(c, p, e.Icon)
		ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
			ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
				title := e.Label(i18n.Locale())
				open := ui.ButtonBase(c).Label(fmt.Sprintf(copyText("查看 %s", "View %s"), title)).MinWidth(0).Shrink(1).Cursor(ui.CursorPointer)
				hover := open.Hovered()
				if open.Clicked() {
					p.a.OpenEntry(e.Name)
				}
				open.Children(func() {
					tx := pagekit.Strong(c, title).SingleLine()
					if hover {
						tx.Underline()
					}
				})
				for _, id := range e.Metadata.Labels {
					if label, ok := d.Listing.Metadata.Labels[id]; ok {
						coss.Badge(c, label.Title.Resolve(i18n.Locale()), coss.BadgeOptions{Variant: coss.BadgeOutline})
					}
				}
			})
			pagekit.Muted(c, e.Description.Resolve(i18n.Locale())).FillWidth().MaxLines(1)
			pagekit.Caption(c, marketMeta(e))
		})
		marketInstall(c, p, e)
	})
}

// marketMeta is "{version} · {author} · {date}" with the locale's medium
// date, dropping the author when the index did not declare one.
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

// marketInstall is an entry's action: installed entries reopen Installed,
// an installable one installs its default version, an incompatible one
// shows the same face disabled with the running DSH in its tooltip.
func marketInstall(c *ui.Context, p *page, e manager.MarketEntry) {
	title := e.Label(i18n.Locale())
	if e.Installation.Status != "not-installed" {
		label := copyText("已安装", "Installed")
		if e.Installation.Status == "update-available" {
			label = copyText("有新版本", "Update available")
		} else if e.Installation.Status == "provided" {
			label = copyText("随 DSH 提供", "Provided by DSH")
		}
		if coss.Button(c, label, coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonSM, Icon: coss.IconCheck}).
			Label(fmt.Sprintf(copyText("在已安装中查看 %s", "View %s in Installed"), title)).Clicked() {
			p.a.OpenPackage(e.Name)
		}
		return
	}
	version := presentedVersion(e)
	install := fmt.Sprintf(copyText("安装 %s", "Install %s"), title)
	if version == nil || (!version.Compatibility.Compatible && !version.Compatibility.Exempted) {
		blocked := coss.Button(c, copyText("安装", "Install"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Disabled: true}).Label(install)
		if version != nil && version.Compatibility.RuntimeVersion != "" {
			blocked.Tooltip(fmt.Sprintf(copyText("没有与当前 DSH %s 兼容的版本", "No version is compatible with DSH %s"), version.Compatibility.RuntimeVersion))
		}
		return
	}
	if coss.Button(c, copyText("安装", "Install"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM, Icon: coss.IconDownload}).Label(install).Clicked() {
		p.a.InstallVersion(e.Name, version.Version)
	}
}

func marketDetail(c *ui.Context, p *page, e manager.MarketEntry) {
	s, a, m := p.s, p.a, p.m
	if pagekit.Back(c, copyText("返回发现", "Back to Discover")) {
		a.CloseEntry()
	}
	ui.Row(c).FillWidth().MinWidth(0).Wrap().Gap(design.Sp(4)).AlignItems(ui.Start).Children(func() {
		artworkSized(c, p, design.Sp(12), design.Sp(9), e.Icon)
		ui.Column(c).Grow(1).Basis(0).MinWidth(200).Gap(design.Sp(1)).Children(func() {
			pagekit.Title(c, e.Label(i18n.Locale()))
			pagekit.Code(c, e.Name).FillWidth().Selectable()
			pagekit.Body(c, e.Description.Resolve(i18n.Locale())).FillWidth()
			if e.Author != "" {
				pagekit.Caption(c, e.Author).FillWidth()
			}
		})
		marketInstall(c, p, e)
	})
	ui.Column(c).FillWidth().Gap(design.Sp(1)).Children(func() {
		detailLink(c, m, e.Homepage)
		if e.Repository != nil {
			detailLink(c, m, e.Repository.URL)
			if e.Repository.Directory != "" {
				pagekit.Code(c, e.Repository.Directory).FillWidth().Selectable()
			}
		}
	})
	ui.Row(c).FillWidth().Wrap().Gap(design.Sp(1.5)).Children(func() {
		if listing := s.Discover.Listing; listing != nil {
			if category, ok := listing.Metadata.Categories[e.Metadata.Category]; ok {
				coss.Badge(c, category.Title.Resolve(i18n.Locale()), coss.BadgeOptions{Variant: coss.BadgeSecondary})
			}
			for _, id := range e.Metadata.Labels {
				if label, ok := listing.Metadata.Labels[id]; ok {
					coss.Badge(c, label.Title.Resolve(i18n.Locale()), coss.BadgeOptions{Variant: coss.BadgeOutline})
				}
			}
			coss.Badge(c, listing.Title.Resolve(i18n.Locale()), coss.BadgeOptions{Variant: coss.BadgeOutline})
		}
	})
	if v := presentedVersion(e); v != nil {
		facts := manager.SourceFacts(e.Name, *v)
		if len(facts) > 0 {
			pagekit.Section(c, copyText("来源", "Source"), nil, func() {
				rows(c, func() {
					for i, fact := range facts {
						if i > 0 {
							rowSeparator(c)
						}
						rowFrame(ui.Row(c)).Children(func() {
							pagekit.Caption(c, fact.Kind).Width(design.Sp(24)).Shrink(0)
							pagekit.Code(c, fact.Value).Grow(1).MinWidth(0).Selectable()
						})
					}
				})
			})
		}
	}
	pagekit.Section(c, copyText("版本", "Versions"), nil, func() {
		rows(c, func() {
			for n, version := range e.Versions {
				v := version
				if n > 0 {
					rowSeparator(c)
				}
				rowFrame(ui.Row(c.Key("version-" + v.Version))).Wrap().AlignItems(ui.Start).Children(func() {
					ui.Column(c).Grow(1).Basis(0).MinWidth(200).Gap(design.Sp(1)).Children(func() {
						ui.Row(c).Wrap().Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
							pagekit.Strong(c, v.Version)
							if v.Version == e.DefaultVersion {
								coss.Badge(c, copyText("默认版本", "Default version"), coss.BadgeOptions{Variant: coss.BadgeSecondary})
							}
							if strings.Contains(strings.Split(v.Version, "+")[0], "-") {
								coss.Badge(c, copyText("预发布", "Prerelease"), coss.BadgeOptions{Variant: coss.BadgeWarning})
							}
							if v.Version == e.Installation.InstalledVersion {
								coss.Badge(c, copyText("已安装", "Installed"), coss.BadgeOptions{Variant: coss.BadgeSuccess})
							}
							pagekit.Caption(c, time.Unix(v.PublishTimestamp, 0).Local().Format("2006-01-02"))
						})
						pagekit.Code(c, v.Install.Spec).FillWidth()
						if !v.Compatibility.Compatible {
							message := copyText("运行时要求未满足：", "Unmet runtime requirement: ") + v.Compatibility.RuntimeVersion
							variant := coss.AlertWarning
							if v.Compatibility.Exempted {
								message, variant = copyText("已接受此版本的兼容性风险。", "Compatibility risk accepted for this version."), coss.AlertInfo
							}
							var req []string
							for _, key := range sortedKeys(v.Compatibility.Requirements) {
								req = append(req, key+": "+v.Compatibility.Requirements[key])
							}
							pagekit.Note(c, variant, message, strings.Join(req, "\n"), nil)
						}
					})
					offered := e.Installation.Status != "provided" && e.Installation.InstalledVersion != v.Version && (v.Compatibility.Compatible || v.Compatibility.Exempted)
					if offered && coss.Button(c, copyText("安装此版本", "Install this version"), coss.ButtonOptions{Variant: coss.ButtonOutline, Size: coss.ButtonSM}).Clicked() {
						a.InstallVersion(e.Name, v.Version)
					}
				})
			}
		})
	})
}

// sourcesEditor is the index-source body of the sources dialog.
func sourcesEditor(c *ui.Context, p *page) {
	s, a := p.s, p.a
	ui.Column(c).FillWidth().MinWidth(0).Gap(design.Sp(4)).Children(func() {
		if len(s.Discover.Indexes) == 0 {
			pagekit.Muted(c, copyText("尚未配置索引源。添加一个源以浏览插件。", "No index sources configured. Add a source to browse plugins.")).FillWidth()
		} else {
			rows(c, func() {
				for n, index := range s.Discover.Indexes {
					i := index
					if n > 0 {
						rowSeparator(c)
					}
					rowFrame(ui.Row(c.Key("source-" + i.URL))).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Gap(design.Sp(0.5)).Children(func() {
							pagekit.Code(c, i.URL).FillWidth().Selectable()
							if i.Error != "" {
								pagekit.Danger(c, errcopyText(i.Error)).FillWidth()
							}
						})
						if coss.Button(c, "", coss.ButtonOptions{Variant: coss.ButtonGhost, Size: coss.ButtonIconSM, Icon: coss.IconTrash, Label: copyText("移除源", "Remove source"), Disabled: s.Discover.Saving}).Clicked() {
							a.RemoveSource(i.URL)
						}
					})
				}
			})
		}
		draft := s.Discover.Draft
		coss.Field(c, coss.FieldOptions{Label: copyText("索引 URL", "Index URL"), Error: sourceErrorText(s.Discover.SourceError),
			Description: copyText("请仅添加信任的索引源；插件会以当前配置的权限运行。", "Add only trusted index sources. Plugins run with this profile's permissions.")}, func() {
			ui.Row(c).FillWidth().Gap(design.Sp(2)).Children(func() {
				if coss.Input(c, &draft, coss.InputOptions{Placeholder: "https://example.com/plugins.json"}).Changed() {
					a.EditSource(draft)
				}
				if coss.Button(c, copyText("添加源", "Add source"), coss.ButtonOptions{Variant: coss.ButtonOutline, Icon: coss.IconPlus, Disabled: s.Discover.Saving || strings.TrimSpace(draft) == ""}).Clicked() {
					a.AddSource()
				}
			})
		})
	})
}

// sourceErrorText localizes the controller's source validation copy;
// anything else is a failure and reads as its sentence.
func sourceErrorText(raw string) string {
	switch raw {
	case "":
		return ""
	case "This index source is already configured.":
		return copyText("此索引源已存在。", raw)
	case "Enter a valid HTTP or HTTPS index URL.":
		return copyText("请输入有效的 HTTP 或 HTTPS 索引地址。", raw)
	case "Index sources cannot be changed in this profile.":
		return copyText("此 profile 不能修改索引源。", raw)
	}
	return errcopyText(raw)
}

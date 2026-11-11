package market

import "whaleisle/internal/ui/i18n"

type marketLocaleEntry struct {
	module string
	key    string
	zh     string
	en     string
}

var marketLocaleCatalog = []marketLocaleEntry{
	{module: `dsh-client-ui-plugin-market`, key: `loading`, zh: `正在读取索引`, en: `Reading the index`},
	{module: `dsh-client-ui-plugin-market`, key: `emptyTitle`, zh: `还没有插件市场`, en: `No plugin markets yet`},
	{module: `dsh-client-ui-plugin-market`, key: `emptyDescription`, zh: `添加一个插件市场索引，即可在这里浏览和安装其中的插件`, en: `Add a plugin market index to browse and install its plugins here.`},
	{module: `dsh-client-ui-plugin-market`, key: `sourcesFailed`, zh: `无法读取索引源`, en: `Could not read the index sources.`},
	{module: `dsh-client-ui-plugin-market`, key: `manageSources`, zh: `管理索引源…`, en: `Manage index sources…`},
	{module: `dsh-client-ui-plugin-market`, key: `marketMenu`, zh: `切换插件市场，当前为 {market}`, en: `Switch plugin market, currently {market}`},
	{module: `dsh-client-ui-plugin-market`, key: `sourcesDescription`, zh: `插件市场以一个索引文件的 URL 提供，「发现」从这里列出的索引浏览插件`, en: `A plugin market is published as the URL of one index file. Discover browses the indexes listed here.`},
	{module: `dsh-client-ui-plugin-market`, key: `sourcesLabel`, zh: `已添加的索引源`, en: `Added index sources`},
	{module: `dsh-client-ui-plugin-market`, key: `sourcePlaceholder`, zh: `https://example.com/index.json`, en: `https://example.com/index.json`},
	{module: `dsh-client-ui-plugin-market`, key: `sourcesTrust`, zh: `索引由第三方提供。只添加你信任的来源：其中的插件安装后以你的权限运行`, en: `Indexes come from third parties. Add only sources you trust: their plugins run with your permissions once installed.`},
	{module: `dsh-client-ui-plugin-market`, key: `sourceInvalid`, zh: `请使用 http 或 https 地址，且不要包含用户名、密码或 # 片段`, en: `Use an http or https address without a user name, password, or # fragment.`},
	{module: `dsh-client-ui-plugin-market`, key: `sourceDuplicate`, zh: `已添加过这个索引源`, en: `This index source is already added.`},
	{module: `dsh-client-ui-plugin-market`, key: `sourceUnavailable`, zh: `当前环境无法保存索引源`, en: `Index sources cannot be saved here.`},
	{module: `dsh-client-ui-plugin-market`, key: `sourceFailed`, zh: `无法保存索引源`, en: `Could not save the index sources.`},
	{module: `dsh-client-ui-plugin-market`, key: `readFailed`, zh: `无法读取此索引`, en: `Could not read this index.`},
	{module: `dsh-client-ui-plugin-market`, key: `readInvalid`, zh: `此索引的格式无效`, en: `This index is not in a valid format.`},
	{module: `dsh-client-ui-plugin-market`, key: `readChanged`, zh: `此索引在加载过程中发生了变化`, en: `This index changed while loading.`},
	{module: `dsh-client-ui-plugin-market`, key: `filterCount`, zh: `标签 · {count}`, en: `Labels · {count}`},
	{module: `dsh-client-ui-plugin-market`, key: `sortBy`, zh: `排序：{sort}`, en: `Sort: {sort}`},
	{module: `dsh-client-ui-plugin-market`, key: `sortRelevance`, zh: `最佳匹配`, en: `Best match`},
	{module: `dsh-client-ui-plugin-market`, key: `sortDefault`, zh: `默认顺序`, en: `Default order`},
	{module: `dsh-client-ui-plugin-market`, key: `sortPublished`, zh: `发布时间`, en: `Publish date`},
	{module: `dsh-client-ui-plugin-market`, key: `switchToAscending`, zh: `改为升序`, en: `Sort ascending`},
	{module: `dsh-client-ui-plugin-market`, key: `switchToDescending`, zh: `改为降序`, en: `Sort descending`},
	{module: `dsh-client-ui-plugin-market`, key: `emptyIndex`, zh: `此索引没有插件`, en: `This index lists no plugins.`},
	{module: `dsh-client-ui-plugin-market`, key: `noMatches`, zh: `没有符合条件的插件`, en: `No plugins match.`},
	{module: `dsh-client-ui-plugin-market`, key: `entryMeta`, zh: `{version} · {date}`, en: `{version} · {date}`},
	{module: `dsh-client-ui-plugin-market`, key: `entryMetaAuthor`, zh: `{version} · {author} · {date}`, en: `{version} · {author} · {date}`},
	{module: `dsh-client-ui-plugin-market`, key: `openInstalled`, zh: `在已安装中查看 {name}`, en: `View {name} in Installed`},
	{module: `dsh-client-ui-plugin-market`, key: `statusProvided`, zh: `随 DSH 提供`, en: `Provided by DSH`},
	{module: `dsh-client-ui-plugin-market`, key: `incompatible`, zh: `此版本与当前的 DSH {runtime} 不兼容，它要求 DSH {range}`, en: `This version is not compatible with the running DSH {runtime}; it requires DSH {range}.`},
	{module: `dsh-client-ui-plugin-market`, key: `noCompatibleVersion`, zh: `没有与当前 DSH {runtime} 兼容的版本`, en: `No version is compatible with DSH {runtime}`},
	{module: `dsh-client-ui-plugin-market`, key: `exempted`, zh: `此版本与当前 DSH 不兼容，但你已为它设置了兼容性豁免`, en: `This version is not compatible with the running DSH, but you set a compatibility exemption for it.`},
	{module: `dsh-client-ui-plugin-market`, key: `factPackage`, zh: `软件包`, en: `Package`},
	{module: `dsh-client-ui-plugin-market`, key: `factRegistry`, zh: `注册表`, en: `Registry`},
	{module: `dsh-client-ui-plugin-market`, key: `factPublished`, zh: `发布时间`, en: `Published`},
	{module: `dsh-client-ui-plugin-market`, key: `factSourceCode`, zh: `源代码`, en: `Source code`},
	{module: `dsh-client-ui-plugin-market`, key: `versionInstalled`, zh: `当前已安装`, en: `Installed`},
	{module: `dsh-client-ui-plugin-market`, key: `versionExempted`, zh: `已豁免`, en: `Exempted`},
	{module: `dsh-client-ui-plugin-market`, key: `confirmTitle`, zh: `安装「{name}」？`, en: `Install "{name}"?`},
	{module: `dsh-client-ui-plugin-market`, key: `installTrust`, zh: `插件在本机以你的权限运行。请确认你信任它的来源`, en: `Plugins run on this machine with your permissions. Make sure you trust where this one comes from.`},
	{module: `dsh-client-ui-settings-market`, key: `buildApprovalMissing`, zh: `缺少有效的 allowBuilds 授权项`, en: `No valid allowBuilds approval keys`},
	{module: `dsh-client-ui-settings-market`, key: `publishedVersion`, zh: `已发布版本`, en: `Published version`},
	{module: `dsh-client-ui-settings-market`, key: `declaredRequirements`, zh: `作者声明的依赖要求`, en: `Author-declared requirements`},
	{module: `dsh-client-ui-settings-market`, key: `detailsFailed`, zh: `部分详情获取失败；可重试或访问项目主页。`, en: `Some details could not be loaded. Retry or visit the project homepage.`},
	{module: `dsh-client-ui-settings-market`, key: `detailsLoading`, zh: `正在获取项目文档…`, en: `Loading project documentation…`},
	{module: `dsh-client-ui-settings-market`, key: `footnotes`, zh: `脚注`, en: `Footnotes`},
	{module: `dsh-client-ui-settings-market`, key: `favorite`, zh: `收藏插件`, en: `Favorite plugin`},
	{module: `dsh-client-ui-settings-market`, key: `added`, zh: `收录时间`, en: `Date added`},
	{module: `dsh-client-ui-settings-market`, key: `unavailable`, zh: `不可安装`, en: `Not installable`},
	{module: `dsh-client-ui-settings-market`, key: `compatibility`, zh: `宿主兼容性`, en: `Host compatibility`},
	{module: `dsh-client-ui-settings-market`, key: `compatibilityUnknown`, zh: `未验证；作者的版本声明不等于运行兼容性保证`, en: `Unverified; declared requirements do not guarantee runtime compatibility`},
	{module: `dsh-client-ui-settings-market`, key: `imageCount`, zh: `{current} / {total}`, en: `{current} / {total}`},
	{module: `dsh-client-ui-settings-market`, key: `readmeHomepage`, zh: `README / 项目主页`, en: `README / project homepage`},
	{module: `dsh-client-ui-settings-market`, key: `sortStars`, zh: `最多星标`, en: `Most stars`},
	{module: `dsh-client-ui-settings-market`, key: `sortNewest`, zh: `最新收录`, en: `Recently added`},
	{module: `dsh-client-ui-settings-market`, key: `period`, zh: `收录时间范围`, en: `Date added range`},
	{module: `dsh-client-ui-settings-market`, key: `installedFilter`, zh: `已安装筛选`, en: `Installed filter`},
	{module: `dsh-client-ui-settings-market`, key: `updateAll`, zh: `更新全部`, en: `Update all`},
	{module: `dsh-client-ui-settings-market`, key: `confirmUninstall`, zh: `确认卸载插件`, en: `Confirm plugin removal`},
	{module: `dsh-client-ui-settings-market`, key: `confirmBatch`, zh: `确认批量更新`, en: `Confirm batch update`},
	{module: `dsh-client-ui-settings-market`, key: `installWarning`, zh: `第三方插件将在本机执行代码。确认来源可信后安装；完成后会重启 Harness，请先结束正在进行的任务。`, en: `Third-party plugins execute code on this computer. Verify the source before installing. Harness will restart; finish any running tasks first.`},
	{module: `dsh-client-ui-settings-market`, key: `uninstallWarning`, zh: `将从桌面 web profile 移除此插件并重启 Harness。请先结束正在进行的任务；插件自行保存的数据不会在此删除。`, en: `This removes the plugin from the desktop web profile and restarts Harness. Finish running tasks first. Data saved by the plugin is not deleted here.`},
	{module: `dsh-client-ui-settings-market`, key: `batchWarning`, zh: `以下插件将依次更新，结束后统一重启 Harness。请先结束正在进行的任务；需要构建授权的项保留失败结果，稍后逐项确认。`, en: `These plugins will update sequentially, then Harness will restart once. Finish running tasks first. Items requiring build approval remain failed for individual approval later.`},
	{module: `dsh-client-ui-settings-market`, key: `stateFailed`, zh: `无法读取或保存市场收藏与操作记录。`, en: `Marketplace favorites or operation history could not be read or saved.`},
	{module: `dsh-client-ui-settings-market`, key: `installedReadFailed`, zh: `无法读取已安装插件；当前列表可能不是最新状态。`, en: `Installed plugins could not be read; the current list may be out of date.`},
	{module: `dsh-client-ui-settings-market`, key: `opOutcomeUnknown`, zh: `插件操作已返回，但 Harness 重启结果尚未确认。请核对运行状态与操作记录，不要重复安装或卸载。`, en: `The plugin operation returned, but the Harness restart result is unconfirmed. Check the runtime and operation history before repeating the operation.`},
	{module: `dsh-client-ui-settings-market`, key: `exportLog`, zh: `导出脱敏日志`, en: `Export redacted logs`},
	{module: `dsh-client-ui-settings-market`, key: `historyEmpty`, zh: `暂无操作记录`, en: `No operations yet`},
	{module: `dsh-client-ui-settings-market`, key: `noLog`, zh: `没有日志输出`, en: `No log output`},
	{module: `dsh-client-ui-settings-market`, key: `operationInterrupted`, zh: `已中断，请核对插件状态`, en: `Interrupted; verify plugin state`},
	{module: `dsh-client-ui-settings-market`, key: `expandCategories`, zh: `全部分类`, en: `All categories`},
	{module: `dsh-client-ui-settings-market`, key: `refreshing`, zh: `刷新中…`, en: `Refreshing…`},
	{module: `dsh-client-ui-settings-market`, key: `loadError`, zh: `无法加载插件目录。`, en: `The plugin catalog is unavailable.`},
	{module: `dsh-client-ui-settings-market`, key: `empty`, zh: `目录里还没有插件。`, en: `No plugins in the catalog yet.`},
	{module: `dsh-client-ui-settings-market`, key: `showMore`, zh: `加载更多（剩余 {count}）`, en: `Show more ({count} left)`},
	{module: `dsh-client-ui-settings-market`, key: `stars`, zh: `{count} 星`, en: `{count} stars`},
	{module: `dsh-client-ui-settings-market`, key: `installedEmpty`, zh: `还没有装过社区插件，去「发现」页逛逛吧。`, en: `No community plugins yet — browse the Discover tab.`},
	{module: `dsh-client-ui-settings-market`, key: `updatesAvailable`, zh: `{count} 个插件可更新`, en: `{count} plugins can be updated`},
	{module: `dsh-client-ui-settings-market`, key: `updatesCurrent`, zh: `未发现可用更新`, en: `No updates found`},
	{module: `dsh-client-ui-settings-market`, key: `updateCheckFailed`, zh: `无法检查插件更新。`, en: `Plugin updates could not be checked.`},
	{module: `dsh-client-ui-settings-market`, key: `updateFromTo`, zh: `{current} → {latest}`, en: `{current} → {latest}`},
	{module: `dsh-client-ui-settings-market`, key: `installing`, zh: `安装中…`, en: `Installing…`},
	{module: `dsh-client-ui-settings-market`, key: `updating`, zh: `更新中…`, en: `Updating…`},
	{module: `dsh-client-ui-settings-market`, key: `uninstalling`, zh: `卸载中…`, en: `Uninstalling…`},
	{module: `dsh-client-ui-settings-market`, key: `progressHeading`, zh: `插件操作进度`, en: `Plugin operation progress`},
	{module: `dsh-client-ui-settings-market`, key: `installDone`, zh: `安装完成，Harness 已重启。`, en: `Installed; Harness restarted.`},
	{module: `dsh-client-ui-settings-market`, key: `updateDone`, zh: `更新完成，Harness 已重启。`, en: `Updated; Harness restarted.`},
	{module: `dsh-client-ui-settings-market`, key: `uninstallDone`, zh: `卸载完成，Harness 已重启。`, en: `Uninstalled; Harness restarted.`},
	{module: `dsh-client-ui-settings-market`, key: `opFailed`, zh: `操作失败：{message}`, en: `Operation failed: {message}`},
	{module: `dsh-client-ui-settings-market`, key: `allowBuildsAsk`, zh: `「{name}」需要在本机执行构建脚本才能完成操作。`, en: `"{name}" needs to run build scripts on this machine to finish the operation.`},
	{module: `dsh-client-ui-settings-market`, key: `allowBuildsKeys`, zh: `将允许：{keys}`, en: `Will allow: {keys}`},
	{module: `dsh-client-ui-settings-market`, key: `allowBuildsConfirm`, zh: `允许并重试`, en: `Allow and retry`},
	{module: `dsh-client-ui-settings-market`, key: `harnessDown`, zh: `插件已写入，但 Harness 没有起来。请从现有入口重启，不要再安装一次。`, en: `The plugin was written, but Harness did not come back. Restart from an existing entry; do not install again.`},
	{module: `dsh-client-ui-plugin-market`, key: `factCategory`, zh: `分类`, en: `Category`},
	{module: `dsh-client-ui-plugin-market`, key: `versionsLabel`, zh: `{name} 的版本`, en: `Versions of {name}`},
	{module: `dsh-client-ui-plugin-market`, key: `removeSourceNamed`, zh: `移除 {url}`, en: `Remove {url}`},
	{module: `dsh-client-ui-plugin-market`, key: `filter`, zh: `标签`, en: `Labels`},
	{module: `dsh-client-ui-plugin-market`, key: `openEntry`, zh: `查看 {name}`, en: `View {name}`},
	{module: `dsh-client-ui-plugin-market`, key: `installTitle`, zh: `安装 {name}`, en: `Install {name}`},
	{module: `dsh-client-ui-plugin-market`, key: `installVersionTitle`, zh: `安装 {name} {version}`, en: `Install {name} {version}`},
}

func init() {
	for _, entry := range marketLocaleCatalog {
		i18n.Register("market"+"."+entry.module+"."+entry.key, entry.zh, entry.en)
	}
}

func marketLocaleText(key string) string {
	zh := i18n.Locale() == i18n.LocaleZH
	for _, entry := range marketLocaleCatalog {
		if entry.key == key || entry.module+"."+entry.key == key {
			if zh {
				return entry.zh
			}
			return entry.en
		}
	}
	return i18n.T(key)
}

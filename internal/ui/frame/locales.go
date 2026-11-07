package frame

type frameLocaleEntry struct {
	zh string
	en string
}

// dsh-client-ui-sidebar-right/lib/types/client/locales.js
var sidebarRightLocaleCatalog = map[string]frameLocaleEntry{
	"command.close":          {zh: "关闭当前页面／窗口", en: "Close current page or window"},
	"command.refresh":        {zh: "刷新当前页面", en: "Refresh current page"},
	"command.noRefresh":      {zh: "当前页面不支持刷新", en: "This page cannot be refreshed"},
	"command.toggle":         {zh: "展开／收起右侧栏", en: "Toggle right sidebar"},
	"command.fullscreen":     {zh: "面板全屏／退出全屏", en: "Toggle panel fullscreen"},
	"command.noSession":      {zh: "请先选择会话", en: "Select a session first"},
	"command.noFocus":        {zh: "请先聚焦右侧面板", en: "Focus a right sidebar pane first"},
	"command.stale":          {zh: "页面已切换，请重新聚焦", en: "The page changed; focus it again"},
	"command.collapsed":      {zh: "请先展开右侧栏", en: "Expand the right sidebar first"},
	"command.float":          {zh: "浮动面板不支持此操作", en: "This action is unavailable in a floating panel"},
	"command.empty":          {zh: "请先打开页面", en: "Open a page first"},
	"command.budget":         {zh: "已达两格上限", en: "Two panes is the limit"},
	"command.width":          {zh: "栏宽不足，拖宽侧边栏后再分栏", en: "Not enough width to split, widen the sidebar"},
	"chrome.expand":          {zh: "打开侧边栏", en: "Open sidebar"},
	"chrome.expandAria":      {zh: "打开右侧边栏", en: "Open right sidebar"},
	"chrome.collapse":        {zh: "收起侧边栏", en: "Collapse sidebar"},
	"chrome.collapseAria":    {zh: "收起右侧边栏", en: "Collapse right sidebar"},
	"chrome.toFullscreen":    {zh: "全屏", en: "Fullscreen"},
	"chrome.exitFullscreen":  {zh: "退出全屏", en: "Exit fullscreen"},
	"dock.emptyPane":         {zh: "空面板", en: "Empty pane"},
	"dock.splitPane":         {zh: "分栏", en: "Split"},
	"dock.splitPaneDisabled": {zh: "已达两格上限", en: "Two panes is the limit"},
	"dock.splitPaneNarrow":   {zh: "栏宽不足，拖宽侧边栏后再分栏", en: "Not enough width to split, widen the sidebar"},
	"dock.closeTab":          {zh: "关闭", en: "Close"},
	"dock.addTab":            {zh: "新标签页", en: "New tab"},
	"dock.dockFloat":         {zh: "收回到侧边栏", en: "Send back to the sidebar"},
	"dock.closeFloat":        {zh: "关闭", en: "Close"},
	"dock.drop.center":       {zh: "移到这里", en: "Move here"},
	"dock.drop.left":         {zh: "左分栏", en: "Add left split"},
	"dock.drop.right":        {zh: "右分栏", en: "Add right split"},
	"dock.drop.top":          {zh: "上分栏", en: "Add top split"},
	"dock.drop.bottom":       {zh: "下分栏", en: "Add bottom split"},
	"tab.guide.title":        {zh: "开始", en: "Start"},
	"tab.guide.heading":      {zh: "打开一个面板", en: "Open a panel"},
	"tab.guide.subtitle":     {zh: "选择要在右侧栏显示的内容。", en: "Pick what shows in the right sidebar."},
	"tab.unavailable":        {zh: "这类内容还没有可用的查看方式。", en: "Nothing here can view this kind of content yet."},
}

// dsh-client-ui-surfaces/lib/types/client/locales.js
var surfacesLocaleCatalog = map[string]frameLocaleEntry{
	"empty.title":              {zh: "打开一个面板", en: "Open a surface"},
	"empty.subtitle":           {zh: "选择要在右侧栏显示的内容。", en: "Choose what to show in the right panel."},
	"card.browser":             {zh: "浏览器", en: "Browser"},
	"card.browser.description": {zh: "打开本地应用或 URL。", en: "Open a local app or URL."},
	"card.browser.disabled":    {zh: "浏览器预览仅在桌面应用中可用。", en: "Browser previews are only available in the desktop app."},
	"card.terminal":            {zh: "终端", en: "Terminal"},
	"card.terminal.description": {zh: "在此工作区启动一个 shell。", en: "Start a shell in this workspace."},
	"card.files":               {zh: "文件", en: "Files"},
	"card.files.description":   {zh: "浏览并阅读工作区文件。", en: "Browse and read workspace files."},
	"card.diff":                {zh: "差异", en: "Diff"},
	"card.diff.description":    {zh: "查看 git 变更。", en: "Review git changes."},
	"card.diff.disabled":       {zh: "差异仅适用于 Git 仓库。", en: "Diff is only available in Git repositories."},
	"card.agents":              {zh: "代理", en: "Agents"},
	"card.agents.description":  {zh: "查看正在运行的代理。", en: "Inspect running agents."},
	"tab.close":                {zh: "关闭", en: "Close"},
	"tab.closeOthers":          {zh: "关闭其他", en: "Close others"},
	"tab.closeToRight":         {zh: "关闭右侧", en: "Close to the right"},
	"tab.closeAll":             {zh: "全部关闭", en: "Close all"},
	"tab.add":                  {zh: "打开面板", en: "Open a surface"},
	"unsaved.title":            {zh: "放弃未保存的更改？", en: "Discard unsaved changes?"},
	"unsaved.body":             {zh: "关闭后，此文件中未保存的编辑将丢失。", en: "Closing discards unsaved edits in this file."},
	"unsaved.bodyMany":         {zh: "关闭后，这些文件中未保存的编辑将丢失。", en: "Closing discards unsaved edits in these files."},
	"unsaved.discard":          {zh: "放弃更改", en: "Discard changes"},
	"unsaved.keep":             {zh: "继续编辑", en: "Keep editing"},
	"unsaved.save":             {zh: "保存", en: "Save"},
	"unsaved.close":            {zh: "关闭", en: "Close"},
}

// dsh-client-ui-titlebar/lib/types/client/locales.js
var titlebarLocaleCatalog = map[string]frameLocaleEntry{
	"terminal.toggle":                     {zh: "切换终端抽屉", en: "Toggle terminal drawer"},
	"terminal.unavailable":                {zh: "终端抽屉不可用", en: "Terminal drawer is unavailable"},
	"surfaces.toggle":                     {zh: "切换右侧栏", en: "Toggle right panel"},
	"shortcut.terminal":                   {zh: "Ctrl+`", en: "Ctrl+`"},
	"shortcut.surfaces":                   {zh: "Ctrl+\\", en: "Ctrl+\\"},
	"settings.terminalToggle.title":       {zh: "终端抽屉开关", en: "Terminal drawer toggle"},
	"settings.terminalToggle.description": {zh: "在标题栏显示终端抽屉按钮。关闭后仍可使用 Ctrl+` 切换。", en: "Show the terminal drawer button in the titlebar. Turning this off still leaves Ctrl+` available."},
	"settings.surfacesToggle.title":       {zh: "右侧栏按钮", en: "Right Sidebar button"},
	"settings.surfacesToggle.description": {zh: "在标题栏显示右侧栏按钮。关闭后仍可使用 Ctrl+\\ 开合同一条右栏。", en: "Show the right Sidebar button in the titlebar. Turning this off still leaves Ctrl+\\ for the same Sidebar."},
}

// dsh-client-ui-layout/lib/types/client/shortcut-locales.js
var layoutShortcutLocaleCatalog = map[string]frameLocaleEntry{
	"toggle": {zh: "展开／收起左侧栏", en: "Toggle left sidebar"},
}

func frameLookupText(catalog map[string]frameLocaleEntry, key string) string {
	entry, ok := catalog[key]
	if !ok {
		return key
	}
	return paneLabel(entry.zh, entry.en)
}

func sidebarRightText(key string) string {
	return frameLookupText(sidebarRightLocaleCatalog, key)
}

func surfacesText(key string) string {
	return frameLookupText(surfacesLocaleCatalog, key)
}

func titlebarText(key string) string {
	return frameLookupText(titlebarLocaleCatalog, key)
}

func layoutShortcutText(key string) string {
	return frameLookupText(layoutShortcutLocaleCatalog, key)
}

func (w *WorkspacePanes) splitDisabledReason(leaf *WorkspacePane) string {
	if w == nil || leaf == nil || leaf.Float != nil {
		return sidebarRightText("command.float")
	}
	if !w.canSplit(leaf) {
		return sidebarRightText("dock.splitPaneDisabled")
	}
	return ""
}

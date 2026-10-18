package browserpreview

// Keep Safari opt-in distinct from a successfully opened native tool window.
func browserDevToolsCopy(goos string) (label, hint, result string) {
	if goos == "darwin" {
		return browserChromeLabel("允许 Safari 检查", "Allow Safari inspection"), browserChromeLabel("仅允许 Safari 检查当前 Browser 页签；不会自动打开应用内开发者工具。页面与站点数据可被检查，关闭该页签即可撤销。", "Allow Safari to inspect only this Browser tab; this does not automatically open in-app developer tools. Page and site data can be inspected. Close the tab to revoke access."), browserChromeLabel("已允许 Safari 检查此页签。在 Safari 设置 → 高级启用网页开发者功能，再到开发菜单 → 本机 → Whale Isle 选择当前网页。此操作不自动打开检查器；关闭 Browser 页签即可撤销。请勿在控制台粘贴不明代码。", "Safari inspection is enabled for this tab. Enable web developer features in Safari Settings → Advanced, then choose this page under Develop → This Mac → Whale Isle. The inspector does not open automatically. Close the Browser tab to revoke access. Do not paste unknown code into the console.")
	}
	return browserChromeLabel("开发者工具", "Developer tools"), browserChromeLabel("仅打开当前 Browser 页的原生开发者工具；控制台可修改本站点，请勿粘贴不明代码。插件与隔离文档预览不可用。", "Open native developer tools for this Browser page only. Console commands can modify the site; do not paste unknown code. Unavailable for plugins and isolated document previews."), browserChromeLabel("已请求打开当前页开发者工具；只运行自己理解的控制台代码，关闭请使用工具窗口的关闭按钮", "Requested developer tools for this page. Run only console code you understand; close using the tool window’s close button.")
}

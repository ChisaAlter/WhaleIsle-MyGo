package browserpreview

// Keep Safari opt-in distinct from a successfully opened native tool window.
func browserDevToolsCopy(goos string) (label, hint, result string) {
	if goos == "darwin" {
		return "允许 Safari 检查", "仅允许 Safari 检查当前 Browser 页签；不会自动打开应用内开发者工具。页面与站点数据可被检查，关闭该页签即可撤销。", "已允许 Safari 检查此页签。在 Safari 设置 → 高级启用网页开发者功能，再到开发菜单 → 本机 → Whale Isle 选择当前网页。此操作不自动打开检查器；关闭 Browser 页签即可撤销。请勿在控制台粘贴不明代码。"
	}
	return "开发者工具", "仅打开当前 Browser 页的原生开发者工具；控制台可修改本站点，请勿粘贴不明代码。插件与隔离文档预览不可用。", "已请求打开当前页开发者工具；只运行自己理解的控制台代码，关闭请使用工具窗口的关闭按钮"
}

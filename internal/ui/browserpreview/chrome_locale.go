package browserpreview

import (
	"fmt"
	"strings"

	"whaleisle/internal/ui/i18n"
)

func browserChromeLabel(zh, en string) string {
	if i18n.Locale() == i18n.LocaleEN || strings.HasPrefix(strings.ToLower(strings.TrimSpace(i18n.Locale())), "en") {
		return en
	}
	return zh
}

// previewZhCatalog and previewEnCatalog mirror
// @deepseek-ai/dsh-client-ui-preview/lib/types/client/locales.js.
var previewZhCatalog = map[string]string{
	"title":                 "浏览器",
	"placeholder":           "搜索或输入 URL",
	"unavailable":           "浏览器预览仅在桌面应用中可用。",
	"guideDescription":      "在桌面端浏览本地应用或网页。",
	"empty":                 "输入本地应用或 URL。",
	"rejected":              "无效的 URL 或不支持的协议。",
	"operationFailed":       "浏览器操作失败，请重试。",
	"dismissFeedback":       "关闭提示",
	"unreachable":           "无法打开此网站。",
	"navigation":            "导航",
	"back":                  "后退",
	"forward":               "前进",
	"reload":                "刷新",
	"stop":                  "停止",
	"external":              "在系统浏览器打开",
	"devtools":              "开发者工具",
	"more":                  "更多",
	"discovered":            "发现的本地服务",
	"hardReload":            "强制刷新",
	"pipOpen":               "打开独立预览窗口",
	"pipClose":              "关闭独立预览窗口",
	"deviceToolbarShow":     "显示设备工具栏",
	"deviceToolbarHide":     "隐藏设备工具栏",
	"pick":                  "选取元素",
	"recordStart":           "开始录制",
	"recordStop":            "停止录制",
	"screenshot":            "截图",
	"appearance":            "外观",
	"appearanceSystem":      "系统",
	"appearanceLight":       "浅色",
	"appearanceDark":        "深色",
	"zoomOut":               "缩小",
	"zoomIn":                "放大",
	"zoomReset":             "重置",
	"clearCookies":          "清除 Cookie",
	"clearCache":            "清除缓存",
	"miniOpen":              "悬浮预览（浮在聊天区）",
	"miniTitle":             "悬浮预览",
	"miniRestore":           "在右侧栏打开",
	"miniClose":             "关闭悬浮预览",
	"deviceToolbarAria":     "设备工具栏",
	"viewportWidth":         "视口宽度",
	"viewportHeight":        "视口高度",
	"viewportRotate":        "旋转视口",
	"viewportPreset":        "设备预设",
	"viewportDimensions":    "视口尺寸",
	"viewportLock":          "锁定宽高比",
	"viewportUnlock":        "解锁宽高比",
	"viewportResponsive":    "自适应",
	"viewportRailWest":      "从左边缘调整视口大小",
	"viewportRailEast":      "从右边缘调整视口大小",
	"viewportRailSouth":     "从下边缘调整视口大小",
	"viewportRailSouthwest": "从左下角调整视口大小",
	"viewportRailSoutheast": "从右下角调整视口大小",
}

var previewEnCatalog = map[string]string{
	"title":                 "Browser",
	"placeholder":           "Search or enter URL",
	"unavailable":           "Browser previews are only available in the desktop app.",
	"guideDescription":      "Browse local apps or web pages in the desktop app.",
	"empty":                 "Open a local app or URL.",
	"rejected":              "Invalid URL or unsupported protocol.",
	"operationFailed":       "The browser action failed. Please try again.",
	"dismissFeedback":       "Dismiss notification",
	"unreachable":           "This site can't be reached.",
	"navigation":            "Navigation",
	"back":                  "Back",
	"forward":               "Forward",
	"reload":                "Reload",
	"stop":                  "Stop",
	"external":              "Open in system browser",
	"devtools":              "Developer tools",
	"more":                  "More",
	"discovered":            "Discovered local servers",
	"hardReload":            "Hard reload",
	"pipOpen":               "Open separate preview window",
	"pipClose":              "Close separate preview window",
	"deviceToolbarShow":     "Show device toolbar",
	"deviceToolbarHide":     "Hide device toolbar",
	"pick":                  "Pick element",
	"recordStart":           "Start recording",
	"recordStop":            "Stop recording",
	"screenshot":            "Screenshot",
	"appearance":            "Appearance",
	"appearanceSystem":      "System",
	"appearanceLight":       "Light",
	"appearanceDark":        "Dark",
	"zoomOut":               "Zoom out",
	"zoomIn":                "Zoom in",
	"zoomReset":             "Reset",
	"clearCookies":          "Clear cookies",
	"clearCache":            "Clear cache",
	"miniOpen":              "Float preview over chat",
	"miniTitle":             "Floating preview",
	"miniRestore":           "Open in right panel",
	"miniClose":             "Close floating preview",
	"deviceToolbarAria":     "Device toolbar",
	"viewportWidth":         "Viewport width",
	"viewportHeight":        "Viewport height",
	"viewportRotate":        "Rotate viewport",
	"viewportPreset":        "Device preset",
	"viewportDimensions":    "Viewport dimensions",
	"viewportLock":          "Lock aspect ratio",
	"viewportUnlock":        "Unlock aspect ratio",
	"viewportResponsive":    "Responsive",
	"viewportRailWest":      "Resize viewport from the left edge",
	"viewportRailEast":      "Resize viewport from the right edge",
	"viewportRailSouth":     "Resize viewport from the bottom edge",
	"viewportRailSouthwest": "Resize viewport from the bottom-left corner",
	"viewportRailSoutheast": "Resize viewport from the bottom-right corner",
}

// sidebarBrowserZhCatalog and sidebarBrowserEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-browser/lib/types/client/locales.js.
var sidebarBrowserZhCatalog = map[string]string{
	"type.label":               "浏览器",
	"guide.title":              "浏览器",
	"guide.description":        "浏览网页",
	"shortcut.noSession":       "请先打开一个会话",
	"address.placeholder":      "输入 HTTP(S) 地址",
	"address.changed":          "URL 已变化",
	"back":                     "后退",
	"forward":                  "前进",
	"reload":                   "刷新",
	"go":                       "前往",
	"dismissFeedback":          "关闭提示",
	"external":                 "在系统浏览器中打开",
	"sandbox.disable":          "关闭沙箱限制",
	"sandbox.enable":           "恢复沙箱限制",
	"sandbox.warning":          "沙箱限制已关闭；页面可以导航顶层应用，并使用下载、模态对话框与输入锁定。",
	"start":                    "输入 HTTP(S) 地址开始浏览",
	"loading":                  "正在打开…",
	"restore.previous":         "上次打开",
	"restore.action":           "恢复页面",
	"error.empty":              "请输入地址。",
	"error.invalid":            "这个地址无效或过长。",
	"error.protocol":           "只支持 HTTP 和 HTTPS 地址；本地文件请使用文档预览。",
	"error.credentials":        "地址不能包含用户名或密码。",
	"error.application-origin": "不能在嵌入浏览器中打开 DSH 应用自身。",
	"load.failed":              "页面加载失败；请刷新重试或在系统浏览器中打开。",
	"load.failed.detail":       "页面加载失败 ({code}): {description}",
	"address.unknown":          "页面已跳转；当前载体无法读取新的 URL。",
}

var sidebarBrowserEnCatalog = map[string]string{
	"type.label":               "Browser",
	"guide.title":              "Browser",
	"guide.description":        "Browse web pages",
	"shortcut.noSession":       "Open a session first",
	"address.placeholder":      "Enter an HTTP(S) address",
	"address.changed":          "URL changed",
	"back":                     "Back",
	"forward":                  "Forward",
	"reload":                   "Reload",
	"go":                       "Go",
	"dismissFeedback":          "Dismiss notification",
	"external":                 "Open in system browser",
	"sandbox.disable":          "Disable sandbox restrictions",
	"sandbox.enable":           "Restore sandbox restrictions",
	"sandbox.warning":          "Sandbox restrictions are disabled; the page can navigate the top-level app and use downloads, modal dialogs, and input locks.",
	"start":                    "Enter an HTTP(S) address to start browsing",
	"loading":                  "Opening…",
	"restore.previous":         "Previously opened",
	"restore.action":           "Restore page",
	"error.empty":              "Enter an address.",
	"error.invalid":            "That address is invalid or too long.",
	"error.protocol":           "Only HTTP and HTTPS addresses are supported; use Document Preview for local files.",
	"error.credentials":        "Addresses cannot contain a username or password.",
	"error.application-origin": "The embedded browser cannot open the DSH application itself.",
	"load.failed":              "The page could not load; reload or open it in the system browser.",
	"load.failed.detail":       "Page load failed ({code}): {description}",
	"address.unknown":          "The page navigated; this carrier cannot read its new URL.",
}

// documentPreviewZhCatalog and documentPreviewEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-documentpreview/lib/types/client/locales.js.
var documentPreviewZhCatalog = map[string]string{
	"loading":               "文档渲染中...",
	"loadMore":              "加载更多",
	"changed":               "文件已更新，当前显示为旧内容",
	"reloadNow":             "重新载入",
	"reload":                "重新读取文件",
	"autoRefresh":           "自动刷新",
	"autoRefresh.enable":    "开启自动刷新",
	"autoRefresh.disable":   "关闭自动刷新",
	"wrap.enable":           "自动换行",
	"wrap.disable":          "取消换行",
	"wrap.aria":             "自动换行",
	"openWith":              "打开方式",
	"viewer.text":           "纯文本",
	"resourceUnavailable":   "文件资源服务不可用",
	"rendererUnavailable":   "预览器 {name} 不可用",
	"unsupportedFile":       "该格式文件暂时无法预览",
	"error.notFound":        "文件不存在，可能已被移动或删除",
	"error.tooLarge":        "单页内容超过 {limit} 上限，无法读取",
	"error.notText":         "该格式文件暂时无法预览",
	"error.notRegularFile":  "该路径不是普通文件，没有可显示的内容",
	"error.unavailable":     "读取失败：{message}",
	"retry":                 "重试",
	"dismissFeedback":       "关闭提示",
}

var documentPreviewEnCatalog = map[string]string{
	"loading":               "Rendering document...",
	"loadMore":              "Load more",
	"changed":               "The file has changed, showing the previous content.",
	"reloadNow":             "Reload",
	"reload":                "Read the file again",
	"autoRefresh":           "Auto refresh",
	"autoRefresh.enable":    "Enable auto refresh",
	"autoRefresh.disable":   "Disable auto refresh",
	"wrap.enable":           "Turn on line wrap",
	"wrap.disable":          "Turn off line wrap",
	"wrap.aria":             "Line wrap",
	"openWith":              "Open with",
	"viewer.text":           "Plain text",
	"resourceUnavailable":   "The file resource service is unavailable.",
	"rendererUnavailable":   "The {name} preview is unavailable.",
	"unsupportedFile":       "Preview is not available for this file type yet.",
	"error.notFound":        "File not found. It may have been moved or deleted.",
	"error.tooLarge":        "This page exceeds the {limit} limit and cannot be read.",
	"error.notText":         "Preview is not available for this file type yet.",
	"error.notRegularFile":  "Not a regular file, nothing to display.",
	"error.unavailable":     "Read failed: {message}",
	"retry":                 "Retry",
	"dismissFeedback":       "Dismiss notification",
}

// documentPreviewExcelZhCatalog and documentPreviewExcelEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-documentpreview/lib/types/client/excel/locales.js.
var documentPreviewExcelZhCatalog = map[string]string{
	"title":                 "表格",
	"language":              "zh",
	"loading":               "文档渲染中...",
	"invalid":               "无法打开此表格。请检查文件格式、内容或密码保护。",
	"tooLarge":              "此表格超过预览大小限制。",
	"timeout":               "打开表格超时。请缩小文件后重试。",
	"encoding":              "无法识别此文本文件的编码。请另存为 UTF-8 或带 BOM 的 UTF-16 后重试。",
	"formulaWarning":        "此工作簿包含公式，显示结果可能缺失或不准确。",
	"unsupportedNotice":     "当前预览不支持此工作簿中的{features}。请使用系统应用打开，以获得完整体验。",
	"charts":                "图表",
	"images":                "图片",
	"shapes":                "形状",
	"conditionalFormatting": "条件格式",
	"featureSeparator":      "、",
	"retry":                 "重试",
}

var documentPreviewExcelEnCatalog = map[string]string{
	"title":                 "Spreadsheet",
	"language":              "en",
	"loading":               "Rendering document...",
	"invalid":               "This spreadsheet could not be opened. Check its format, contents, or password protection.",
	"tooLarge":              "This workbook exceeds the preview size limit.",
	"timeout":               "Opening this workbook timed out. Try a smaller file.",
	"encoding":              "This text encoding could not be read. Save the file as UTF-8 or UTF-16 with a BOM and retry.",
	"formulaWarning":        "This workbook contains formulas. Displayed results may be missing or inaccurate.",
	"unsupportedNotice":     "This preview does not support {features} in this workbook. Open it in a system application for the full experience.",
	"charts":                "charts",
	"images":                "images",
	"shapes":                "shapes",
	"conditionalFormatting": "conditional formatting",
	"featureSeparator":      ", ",
	"retry":                 "Retry",
}

// documentPreviewHTMLZhCatalog and documentPreviewHTMLEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-documentpreview/lib/types/client/html/locales.js.
var documentPreviewHTMLZhCatalog = map[string]string{
	"title":   "HTML",
	"frame":   "HTML 文档预览",
	"loading": "文档渲染中...",
	"failed":  "无法预览这份 HTML 文档",
}

var documentPreviewHTMLEnCatalog = map[string]string{
	"title":   "HTML",
	"frame":   "HTML document preview",
	"loading": "Rendering document...",
	"failed":  "This HTML document could not be previewed.",
}

// documentPreviewImageZhCatalog and documentPreviewImageEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-documentpreview/lib/types/client/image/locales.js.
var documentPreviewImageZhCatalog = map[string]string{
	"zoomControls": "缩放控件",
	"zoomMenu":     "选择缩放比例",
	"zoomOut":      "缩小",
	"zoomIn":       "放大",
	"zoomFitWidth": "适应宽度",
	"zoomValue":    "{percent}%",
	"title":        "图片",
	"preview":      "图片预览：{name}",
	"loading":      "文档渲染中...",
	"failed":       "无法显示这张图片",
	"unsupported":  "图片预览需要完整文件内容",
}

var documentPreviewImageEnCatalog = map[string]string{
	"zoomControls": "Zoom controls",
	"zoomMenu":     "Choose zoom",
	"zoomOut":      "Zoom out",
	"zoomIn":       "Zoom in",
	"zoomFitWidth": "Fit width",
	"zoomValue":    "{percent}%",
	"title":        "Image",
	"preview":      "Image preview: {name}",
	"loading":      "Rendering document...",
	"failed":       "This image could not be displayed.",
	"unsupported":  "Image preview requires the complete file contents.",
}

// documentPreviewMarkdownZhCatalog and documentPreviewMarkdownEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-documentpreview/lib/types/client/markdown/locales.js.
var documentPreviewMarkdownZhCatalog = map[string]string{
	"viewer.label": "Markdown",
	"code.copy":    "复制",
	"code.copied":  "已复制",
	"footnotes":    "脚注",
}

var documentPreviewMarkdownEnCatalog = map[string]string{
	"viewer.label": "Markdown",
	"code.copy":    "Copy",
	"code.copied":  "Copied",
	"footnotes":    "Footnotes",
}

// documentPreviewOfficeZhCatalog and documentPreviewOfficeEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-documentpreview/lib/types/client/office/locales.js.
var documentPreviewOfficeZhCatalog = map[string]string{
	"title":                   "Office 文档",
	"loading":                 "文档渲染中...",
	"retry":                   "重试",
	"viewMissingFonts":        "缺失 {count} 种字体，点击查看",
	"missingFontsTitle":       "缺失的字体",
	"missingFontsDescription": "本次预览无法使用以下字体，预览中的文字和排版可能与原文档不同。",
	"missingFontsCount":       "{count} 种字体",
	"closeDetails":            "关闭字体详情",
	"unavailable":             "Office 预览不可用。请在运行 Whale Isle 的主机上启用文档预览服务。",
	"invalid":                 "无法预览此 Office 文件。文件可能已损坏、受密码保护，或与扩展名不符。",
	"tooLarge":                "Office 文件或转换后的 PDF 超过预览大小上限，请缩小文件或调整预览配置。",
	"failed":                  "Office 转换失败，未生成可用的 PDF。请检查该文件后重试。",
	"timeout":                 "Office 转换超时，请重试。",
	"busy":                    "Office 预览任务较多，请稍后重试。",
	"changed":                 "文件在读取时已更改，请重新打开预览。",
}

var documentPreviewOfficeEnCatalog = map[string]string{
	"title":                   "Office document",
	"loading":                 "Rendering document...",
	"retry":                   "Retry",
	"viewMissingFonts":        "Missing fonts: {count}. Click to view.",
	"missingFontsTitle":       "Missing fonts",
	"missingFontsDescription": "These fonts are unavailable for this preview. Text and layout may differ from the original document.",
	"missingFontsCount":       "Fonts: {count}",
	"closeDetails":            "Close font details",
	"unavailable":             "Office previews are unavailable. Enable the document preview service on the computer running Whale Isle.",
	"invalid":                 "This Office file cannot be previewed. It may be damaged, password protected, or have the wrong extension.",
	"tooLarge":                "The Office file or converted PDF exceeds the preview size limit. Reduce the file size or adjust the preview configuration.",
	"failed":                  "Office conversion did not produce a usable PDF. Check the file and try again.",
	"timeout":                 "Office conversion timed out. Try again.",
	"busy":                    "Office preview is busy. Try again shortly.",
	"changed":                 "The file changed while being read. Reopen the preview.",
}

// documentPreviewPDFZhCatalog and documentPreviewPDFEnCatalog mirror
// @deepseek-ai/dsh-client-ui-sidebar-documentpreview/lib/types/client/pdf/locales.js.
var documentPreviewPDFZhCatalog = map[string]string{
	"zoomControls": "缩放控件",
	"zoomMenu":     "选择缩放比例",
	"zoomOut":      "缩小",
	"zoomIn":       "放大",
	"zoomFitWidth": "适应宽度",
	"zoomValue":    "{percent}%",
	"title":        "PDF",
	"pageImage":    "PDF 第 {page} 页",
	"loading":      "文档渲染中...",
	"rendering":    "正在绘制页面…",
	"failed":       "无法显示 PDF：{message}",
	"password":     "此 PDF 需要密码，暂不支持预览",
	"workerFailed": "PDF 渲染进程无法继续，请重试",
	"unsupported":  "PDF 预览需要完整文件内容",
	"retry":        "重试",
}

var documentPreviewPDFEnCatalog = map[string]string{
	"zoomControls": "Zoom controls",
	"zoomMenu":     "Choose zoom",
	"zoomOut":      "Zoom out",
	"zoomIn":       "Zoom in",
	"zoomFitWidth": "Fit width",
	"zoomValue":    "{percent}%",
	"title":        "PDF",
	"pageImage":    "PDF page {page}",
	"loading":      "Rendering document...",
	"rendering":    "Rendering page…",
	"failed":       "Cannot display PDF: {message}",
	"password":     "This PDF requires a password; password-protected previews are not supported.",
	"workerFailed": "The PDF rendering process could not continue. Please retry.",
	"unsupported":  "PDF preview requires the complete file contents.",
	"retry":        "Retry",
}

func browserCatalogLookup(zhMap, enMap map[string]string, key string) string {
	if i18n.Locale() == i18n.LocaleEN || strings.HasPrefix(strings.ToLower(strings.TrimSpace(i18n.Locale())), "en") {
		if v, ok := enMap[key]; ok {
			return v
		}
	}
	if v, ok := zhMap[key]; ok {
		return v
	}
	return key
}

func browserPreviewText(key string) string {
	return browserCatalogLookup(previewZhCatalog, previewEnCatalog, key)
}

func sidebarBrowserText(key string) string {
	return browserCatalogLookup(sidebarBrowserZhCatalog, sidebarBrowserEnCatalog, key)
}

func documentPreviewText(key string) string {
	return browserCatalogLookup(documentPreviewZhCatalog, documentPreviewEnCatalog, key)
}

func documentPreviewExcelText(key string) string {
	return browserCatalogLookup(documentPreviewExcelZhCatalog, documentPreviewExcelEnCatalog, key)
}

func documentPreviewHTMLText(key string) string {
	return browserCatalogLookup(documentPreviewHTMLZhCatalog, documentPreviewHTMLEnCatalog, key)
}

func documentPreviewImageText(key string) string {
	return browserCatalogLookup(documentPreviewImageZhCatalog, documentPreviewImageEnCatalog, key)
}

func documentPreviewMarkdownText(key string) string {
	return browserCatalogLookup(documentPreviewMarkdownZhCatalog, documentPreviewMarkdownEnCatalog, key)
}

func documentPreviewOfficeText(key string) string {
	return browserCatalogLookup(documentPreviewOfficeZhCatalog, documentPreviewOfficeEnCatalog, key)
}

func documentPreviewPDFText(key string) string {
	return browserCatalogLookup(documentPreviewPDFZhCatalog, documentPreviewPDFEnCatalog, key)
}

func formatSidebarBrowserLoadFailedDetail(code, description string) string {
	tpl := sidebarBrowserText("load.failed.detail")
	tpl = strings.ReplaceAll(tpl, "{code}", code)
	return strings.ReplaceAll(tpl, "{description}", description)
}

func formatDocumentPreviewPDFFailed(message string) string {
	return strings.ReplaceAll(documentPreviewPDFText("failed"), "{message}", message)
}

func formatDocumentPreviewPDFPageImage(page int) string {
	return strings.ReplaceAll(documentPreviewPDFText("pageImage"), "{page}", fmt.Sprintf("%d", page))
}

func formatDocumentPreviewZoomValue(percent int) string {
	return strings.ReplaceAll(documentPreviewPDFText("zoomValue"), "{percent}", fmt.Sprintf("%d", percent))
}

func formatDocumentPreviewOfficeMissingFonts(count int) string {
	return strings.ReplaceAll(documentPreviewOfficeText("viewMissingFonts"), "{count}", fmt.Sprintf("%d", count))
}

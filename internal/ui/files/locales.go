package files

import (
	"strings"
	"whaleisle/internal/ui/i18n"
)

// filesLocaleZH mirrors @deepseek-ai/dsh-client-ui-files/lib/types/client/locales.js (zh).
var filesLocaleZH = map[string]string{
	"empty.cwd":                    "没有工作区，无法浏览文件。",
	"empty.dir":                    "此目录为空。",
	"type.label":                   "文件",
	"guide.files.title":            "文件",
	"guide.files.description":      "浏览当前会话工作区中的文件。",
	"listing":                      "正在列出目录…",
	"error.list":                   "无法读取目录。",
	"error.workspaceAccess":        "此目录未授权桌面文件访问。请打开项目会话，再浏览它的工作区。",
	"error.read":                   "无法读取文件。",
	"error.open":                   "无法打开文件。",
	"preview.binary":               "无法预览二进制文件。",
	"preview.truncated":            "文件过长，仅显示开头。",
	"preview.copy":                 "复制",
	"preview.copied":               "已复制",
	"preview.footnotes":            "脚注",
	"mention":                      "引用到输入框",
	"copy.mention":                 "复制引用",
	"copy.relative":                "复制相对路径",
	"copy.absolute":                "复制绝对路径",
	"refresh":                      "刷新",
	"copied":                       "已复制",
	"search":                       "搜索文件",
	"search.pending":               "正在搜索文件…",
	"search.clear":                 "清除搜索",
	"open.folder":                  "在文件夹中显示",
	"open.system":                  "系统默认程序",
	"preview.save":                 "保存",
	"preview.saved":                "已保存",
	"preview.source":               "源码",
	"preview.render":               "渲染",
	"preview.wrap":                 "自动换行",
	"preview.browser":              "在浏览器中打开",
	"preview.floating":             "独立窗口预览",
	"preview.floating.description": "在独立的置顶只读窗口中打开已保存文件（不会保存或同步未保存编辑）",
	"preview.floating.noCwd":       "无法确定文件所属会话的工作区",
	"preview.floating.failed":      "独立窗口预览打开失败。",
	"preview.comment":              "添加到对话",
	"error.write":                  "无法保存文件。",
	"error.changed":                "磁盘上的文件已更改。再次保存将覆盖。",
	"unsaved.title":                "有未保存的修改",
	"unsaved.body":                 "关闭前可以保存修改，或继续编辑。",
	"unsaved.close":                "关闭",
	"unsaved.keep":                 "继续编辑",
	"unsaved.discard":              "丢弃",
	"unsaved.save":                 "保存并关闭",
}

// filesLocaleEN mirrors @deepseek-ai/dsh-client-ui-files/lib/types/client/locales.js (en).
var filesLocaleEN = map[string]string{
	"empty.cwd":                    "A workspace is required to browse files.",
	"empty.dir":                    "This directory is empty.",
	"type.label":                   "Files",
	"guide.files.title":            "Files",
	"guide.files.description":      "Browse files in this session workspace.",
	"listing":                      "Listing directory…",
	"error.list":                   "Could not list the directory.",
	"error.workspaceAccess":        "Desktop file access is not authorized for this directory. Open a project session to browse its workspace.",
	"error.read":                   "Could not read the file.",
	"error.open":                   "Could not open the file.",
	"preview.binary":               "This binary file cannot be previewed.",
	"preview.truncated":            "File is too large; showing the beginning.",
	"preview.copy":                 "Copy",
	"preview.copied":               "Copied",
	"preview.footnotes":            "Footnotes",
	"mention":                      "Mention in composer",
	"copy.mention":                 "Copy mention",
	"copy.relative":                "Copy relative path",
	"copy.absolute":                "Copy absolute path",
	"refresh":                      "Refresh",
	"copied":                       "Copied",
	"search":                       "Search files",
	"search.pending":               "Searching files…",
	"search.clear":                 "Clear search",
	"open.folder":                  "Show in folder",
	"open.system":                  "System default",
	"preview.save":                 "Save",
	"preview.saved":                "Saved",
	"preview.source":               "Source",
	"preview.render":               "Rendered",
	"preview.wrap":                 "Word wrap",
	"preview.browser":              "Open in browser",
	"preview.floating":             "Preview in separate window",
	"preview.floating.description": "Open the saved file in a separate read-only, always-on-top window (unsaved edits are not saved or transferred)",
	"preview.floating.noCwd":       "The workspace for this file’s session is unavailable.",
	"preview.floating.failed":      "Could not open the separate preview window.",
	"preview.comment":              "Add to chat",
	"error.write":                  "Could not save the file.",
	"error.changed":                "The file changed on disk. Save again to overwrite.",
	"unsaved.title":                "Unsaved changes",
	"unsaved.body":                 "Save your changes before closing, or keep editing.",
	"unsaved.close":                "Close",
	"unsaved.keep":                 "Keep editing",
	"unsaved.discard":              "Discard",
	"unsaved.save":                 "Save and close",
}

// sidebarFilesLocaleZH mirrors @deepseek-ai/dsh-client-ui-sidebar-files/lib/types/client/locales.js (zh).
var sidebarFilesLocaleZH = map[string]string{
	"shortcut.noSession":     "请先选择会话",
	"type.label":             "文件",
	"guide.title":            "工作区文件",
	"guide.description":      "浏览会话工作区的文件",
	"loading":                "正在读取…",
	"empty":                  "空目录",
	"truncated":              "条目太多，只显示了一部分。",
	"noWorkspace":            "这个会话没有工作区目录。",
	"reload":                 "重新读取",
	"autoRefresh":            "自动刷新",
	"autoRefresh.enable":     "开启自动刷新",
	"autoRefresh.disable":    "关闭自动刷新",
	"entry.other":            "这不是文件或目录，没法打开。",
	"error.notFound":         "这个目录不在了。可能已被移动或删除。",
	"error.outsideWorkspace": "这个目录在工作区之外，侧栏不会读取它。",
	"error.notDirectory":     "这不是一个目录。",
	"error.unavailable":      "读取失败：{message}",
}

// sidebarFilesLocaleEN mirrors @deepseek-ai/dsh-client-ui-sidebar-files/lib/types/client/locales.js (en).
var sidebarFilesLocaleEN = map[string]string{
	"shortcut.noSession":     "Select a session first",
	"type.label":             "Files",
	"guide.title":            "Workspace files",
	"guide.description":      "Browse files in this session's workspace",
	"loading":                "Reading…",
	"empty":                  "Empty directory",
	"truncated":              "Too many entries, showing only some of them.",
	"noWorkspace":            "This session has no workspace directory.",
	"reload":                 "Reload",
	"autoRefresh":            "Auto refresh",
	"autoRefresh.enable":     "Enable auto refresh",
	"autoRefresh.disable":    "Disable auto refresh",
	"entry.other":            "Not a file or a directory, so it cannot be opened.",
	"error.notFound":         "That directory is gone. It may have been moved or deleted.",
	"error.outsideWorkspace": "That directory is outside the workspace, so the sidebar will not read it.",
	"error.notDirectory":     "That is not a directory.",
	"error.unavailable":      "Read failed: {message}",
}

// deliverablesLocaleZH mirrors @deepseek-ai/dsh-client-ui-deliverables/lib/types/client/locales.js (zh).
var deliverablesLocaleZH = map[string]string{
	"presented.nativeUnavailable": "此文件没有可用的主机路径，请在应用内预览",
	"presented.revealError":       "无法在文件管理器中显示，请重试",
	"presented.directoryError":    "无法打开所在文件夹，请重试",
	"presented.directoryOpening":  "正在打开所在文件夹…",
	"presented.directoryOpened":   "已请求打开所在文件夹",
	"presented.revealed":          "已请求在文件管理器中显示",
	"presented.revealing":         "正在文件管理器中显示…",
	"presented.unavailable":       "此主机没有可用的桌面，无法使用外部程序打开文件或文件夹；文件仍可在应用内预览",
	"presented.retry":             "重试",
	"feedback.close":              "关闭提示",
	"presented.hostError":         "无法读取主机桌面信息",
	"presented.preview":           "点击预览",
	"presented.previewButton":     "预览 {name}",
	"presented.previewCard":       "预览 {name}",
	"presented.all":               "全部 {count} 个文件",
	"presented.expandAria":        "展开全部 {count} 个交付文件",
	"presented.collapse":          "收起",
	"presented.collapseAria":      "收起交付文件列表",
	"presented.opening":           "正在打开…",
	"presented.opened":            "已请求打开",
	"presented.error":             "打开失败，点击重试",
	"presented.file":              "文件",
	"row.title":                   "交付文件",
	"row.running":                 "正在交付",
	"row.preparing":               "准备交付",
	"row.ok":                      "已交付",
	"row.error":                   "交付失败",
	"row.stopped":                 "已中断",
	"row.inspect":                 "查看调用",
	"changes.title":               "已编辑 {count} 个文件",
	"changes.singleTitle":         "已编辑 {name}",
	"changes.added":               "+{count}",
	"changes.deleted":             "-{count}",
	"changes.binary":              "二进制",
	"changes.openReview":          "在侧边栏查看本轮改动",
	"changes.all":                 "全部 {count} 个文件",
	"changes.expandAria":          "展开全部 {count} 个改动文件",
	"changes.collapse":            "收起",
	"changes.collapseAria":        "收起改动文件列表",
	"changes.oversized":           "过大",
	"changes.viewDiff":            "查看 {name} 的改动",
	"review.title":                "第 {turn} 轮改动",
	"review.selectFile":           "选择要查看的文件",
	"review.split":                "切换为左右对比",
	"review.unified":              "切换为单栏对比",
	"review.splitAria":            "左右对比",
	"review.wrap":                 "开启自动换行",
	"review.nowrap":               "关闭自动换行",
	"review.wrapAria":             "自动换行",
	"review.openFile":             "在侧边栏打开整个文件",
	"review.openFileAria":         "在侧边栏打开 {name}",
	"diff.loading":                "正在读取改动…",
	"diff.missing":                "这轮改动的内容已不可用",
	"diff.error":                  "无法读取改动",
	"diff.binary":                 "二进制文件，无法显示改动",
	"diff.oversized":              "文件过大，无法显示改动",
	"diff.created":                "本轮新建的文件",
	"diff.deleted":                "本轮删除的文件",
	"diff.unchanged":              "两侧内容相同",
	"diff.coarse":                 "逐行对比超时，按整个文件替换显示",
	"diff.truncated":              "只显示前 {count} 行",
	"diff.highlightSkipped":       "部分内容过大，未语法高亮",
}

// deliverablesLocaleEN mirrors @deepseek-ai/dsh-client-ui-deliverables/lib/types/client/locales.js (en).
var deliverablesLocaleEN = map[string]string{
	"presented.nativeUnavailable": "This file has no available Host path. Preview it in the app.",
	"presented.revealError":       "Could not show in file manager. Try again.",
	"presented.directoryError":    "Could not open containing folder. Try again.",
	"presented.directoryOpening":  "Opening containing folder…",
	"presented.directoryOpened":   "Requested opening containing folder",
	"presented.revealed":          "Requested display in file manager",
	"presented.revealing":         "Showing in file manager…",
	"presented.unavailable":       "This Host has no desktop available to open files or folders in external apps. Files can still be previewed in the app.",
	"presented.retry":             "Retry",
	"feedback.close":              "Dismiss notification",
	"presented.hostError":         "Could not read the Host desktop information",
	"presented.preview":           "Click to preview",
	"presented.previewButton":     "Preview {name}",
	"presented.previewCard":       "Preview {name}",
	"presented.all":               "All {count} files",
	"presented.expandAria":        "Show all {count} delivered files",
	"presented.collapse":          "Collapse",
	"presented.collapseAria":      "Collapse delivered files",
	"presented.opening":           "Opening…",
	"presented.opened":            "Open requested",
	"presented.error":             "Could not open. Click to retry.",
	"presented.file":              "File",
	"row.title":                   "Present files",
	"row.running":                 "Delivering",
	"row.preparing":               "Preparing deliverables",
	"row.ok":                      "Delivered",
	"row.error":                   "Delivery failed",
	"row.stopped":                 "Interrupted",
	"row.inspect":                 "Inspect call",
	"changes.title":               "Edited {count} files",
	"changes.singleTitle":         "Edited {name}",
	"changes.added":               "+{count}",
	"changes.deleted":             "-{count}",
	"changes.binary":              "binary",
	"changes.openReview":          "Review this turn’s changes in the sidebar",
	"changes.all":                 "All {count} files",
	"changes.expandAria":          "Show all {count} changed files",
	"changes.collapse":            "Collapse",
	"changes.collapseAria":        "Collapse changed files",
	"changes.oversized":           "too large",
	"changes.viewDiff":            "View changes to {name}",
	"review.title":                "Review · turn {turn}",
	"review.selectFile":           "Choose the file to review",
	"review.split":                "Switch to split view",
	"review.unified":              "Switch to unified view",
	"review.splitAria":            "Split view",
	"review.wrap":                 "Enable line wrap",
	"review.nowrap":               "Disable line wrap",
	"review.wrapAria":             "Line wrap",
	"review.openFile":             "Open the whole file in the sidebar",
	"review.openFileAria":         "Open {name} in sidebar",
	"diff.loading":                "Reading changes…",
	"diff.missing":                "The contents of this turn’s changes are no longer available",
	"diff.error":                  "Could not read the changes",
	"diff.binary":                 "Binary file; changes cannot be shown",
	"diff.oversized":              "File too large; changes cannot be shown",
	"diff.created":                "Created in this turn",
	"diff.deleted":                "Deleted in this turn",
	"diff.unchanged":              "Both sides hold the same lines",
	"diff.coarse":                 "Line comparison timed out; shown as a whole-file replacement",
	"diff.truncated":              "Showing the first {count} lines",
	"diff.highlightSkipped":       "Some content is too large to syntax-highlight",
}

// openInAppLocaleZH mirrors @deepseek-ai/dsh-client-ui-open-in-app/lib/types/client/locales.js (zh).
var openInAppLocaleZH = map[string]string{
	"close":                "关闭",
	"open.title":           "用 {app} 打开",
	"path.appDefault":      "{app}（默认）",
	"path.appsError":       "无法获取应用列表",
	"shortcut.busy":        "正在打开工作区",
	"shortcut.unavailable": "当前工作区或本地应用不可用",
	"open.tooltip":         "在本地打开",
	"path.open":            "打开",
	"path.more":            "更多打开方式",
	"path.reveal":          "显示文件位置",
	"path.openError":       "打开失败，请重试",
	"path.revealError":     "无法显示文件位置，请重试",
	"app.cursor":           "Cursor",
	"app.vscode":           "VS Code",
	"app.vscodeinsiders":   "VS Code Insiders",
	"app.windsurf":         "Windsurf",
	"app.zed":              "Zed",
	"app.sublimetext":      "Sublime Text",
	"app.xcode":            "Xcode",
	"app.androidstudio":    "Android Studio",
	"app.intellij":         "IntelliJ IDEA",
	"app.pycharm":          "PyCharm",
	"app.webstorm":         "WebStorm",
	"app.phpstorm":         "PhpStorm",
	"app.goland":           "GoLand",
	"app.rider":            "Rider",
	"app.rustrover":        "RustRover",
	"app.fork":             "Fork",
	"app.sourcetree":       "Sourcetree",
	"app.github":           "GitHub Desktop",
	"app.tower":            "Tower",
	"app.gitkraken":        "GitKraken",
	"app.smartgit":         "SmartGit",
	"app.sublimemerge":     "Sublime Merge",
	"app.ghostty":          "Ghostty",
	"app.warp":             "Warp",
	"app.iterm":            "iTerm2",
	"app.kitty":            "kitty",
	"app.windowsterminal":  "Windows Terminal",
	"app.gitbash":          "Git Bash",
	"app.gnometerminal":    "GNOME Terminal",
	"app.konsole":          "Konsole",
	"app.finder":           "访达",
	"app.explorer":         "文件资源管理器",
	"app.filemanager":      "文件管理器",
	"app.terminal":         "终端",
}

// openInAppLocaleEN mirrors @deepseek-ai/dsh-client-ui-open-in-app/lib/types/client/locales.js (en).
var openInAppLocaleEN = map[string]string{
	"close":                "Close",
	"open.title":           "Open in {app}",
	"path.appDefault":      "{app} (default)",
	"path.appsError":       "Could not load applications",
	"shortcut.busy":        "Opening workspace",
	"shortcut.unavailable": "Current workspace or local application unavailable",
	"open.tooltip":         "Open locally",
	"path.open":            "Open",
	"path.more":            "More ways to open",
	"path.reveal":          "Show file location",
	"path.openError":       "Could not open. Try again.",
	"path.revealError":     "Could not show the file location. Try again.",
	"app.cursor":           "Cursor",
	"app.vscode":           "VS Code",
	"app.vscodeinsiders":   "VS Code Insiders",
	"app.windsurf":         "Windsurf",
	"app.zed":              "Zed",
	"app.sublimetext":      "Sublime Text",
	"app.xcode":            "Xcode",
	"app.androidstudio":    "Android Studio",
	"app.intellij":         "IntelliJ IDEA",
	"app.pycharm":          "PyCharm",
	"app.webstorm":         "WebStorm",
	"app.phpstorm":         "PhpStorm",
	"app.goland":           "GoLand",
	"app.rider":            "Rider",
	"app.rustrover":        "RustRover",
	"app.fork":             "Fork",
	"app.sourcetree":       "Sourcetree",
	"app.github":           "GitHub Desktop",
	"app.tower":            "Tower",
	"app.gitkraken":        "GitKraken",
	"app.smartgit":         "SmartGit",
	"app.sublimemerge":     "Sublime Merge",
	"app.ghostty":          "Ghostty",
	"app.warp":             "Warp",
	"app.iterm":            "iTerm2",
	"app.kitty":            "kitty",
	"app.windowsterminal":  "Windows Terminal",
	"app.gitbash":          "Git Bash",
	"app.gnometerminal":    "GNOME Terminal",
	"app.konsole":          "Konsole",
	"app.finder":           "Finder",
	"app.explorer":         "File Explorer",
	"app.filemanager":      "Files",
	"app.terminal":         "Terminal",
}

// documentPreviewLocaleZH mirrors @deepseek-ai/dsh-client-ui-sidebar-documentpreview locale dictionaries (zh).
var documentPreviewLocaleZH = map[string]string{
	"loading":                  "文档渲染中...",
	"changed":                  "文件已更新，当前显示为旧内容",
	"reload":                   "重新读取文件",
	"autoRefresh.enable":       "开启自动刷新",
	"autoRefresh.disable":      "关闭自动刷新",
	"wrap.disable":             "取消换行",
	"resourceUnavailable":      "文件资源服务不可用",
	"unsupportedFile":          "该格式文件暂时无法预览",
	"error.notFound":           "文件不存在，可能已被移动或删除",
	"error.tooLarge":           "单页内容超过 {limit} 上限，无法读取",
	"error.notText":            "该格式文件暂时无法预览",
	"error.notRegularFile":     "该路径不是普通文件，没有可显示的内容",
	"excel.invalid":            "无法打开此表格。请检查文件格式、内容或密码保护。",
	"excel.tooLarge":           "此表格超过预览大小限制。",
	"excel.timeout":            "打开表格超时。请缩小文件后重试。",
	"excel.encoding":           "无法识别此文本文件的编码。请另存为 UTF-8 或带 BOM 的 UTF-16 后重试。",
	"excel.formulaWarning":     "此工作簿包含公式，显示结果可能缺失或不准确。",
	"excel.unsupportedNotice":  "当前预览不支持此工作簿中的{features}。请使用系统应用打开，以获得完整体验。",
	"excel.charts":             "图表",
	"excel.shapes":             "形状",
	"excel.conditionalFormat":  "条件格式",
	"html.frame":               "HTML 文档预览",
	"html.failed":              "无法预览这份 HTML 文档",
	"image.zoomControls":       "缩放控件",
	"image.zoomMenu":           "选择缩放比例",
	"image.zoomFitWidth":       "适应宽度",
	"image.zoomValue":          "{percent}%",
	"image.preview":            "图片预览：{name}",
	"image.failed":             "无法显示这张图片",
	"image.unsupported":        "图片预览需要完整文件内容",
	"markdown.footnotes":       "脚注",
	"office.viewMissingFonts":  "缺失 {count} 种字体，点击查看",
	"office.missingFontsTitle": "缺失的字体",
	"office.missingFontsDesc":  "本次预览无法使用以下字体，预览中的文字和排版可能与原文档不同。",
	"office.missingFontsCount": "{count} 种字体",
	"office.closeDetails":      "关闭字体详情",
	"office.unavailable":       "Office 预览不可用。请在运行 Whale Isle 的主机上启用文档预览服务。",
	"office.tooLarge":          "Office 文件或转换后的 PDF 超过预览大小上限，请缩小文件或调整预览配置。",
	"office.failed":            "Office 转换失败，未生成可用的 PDF。请检查该文件后重试。",
	"office.timeout":           "Office 转换超时，请重试。",
	"office.busy":              "Office 预览任务较多，请稍后重试。",
	"office.changed":           "文件在读取时已更改，请重新打开预览。",
	"pdf.pageImage":            "PDF 第 {page} 页",
	"pdf.rendering":            "正在绘制页面…",
	"pdf.failed":               "无法显示 PDF：{message}",
	"pdf.password":             "此 PDF 需要密码，暂不支持预览",
	"pdf.workerFailed":         "PDF 渲染进程无法继续，请重试",
	"pdf.unsupported":          "PDF 预览需要完整文件内容",
}

// documentPreviewLocaleEN mirrors @deepseek-ai/dsh-client-ui-sidebar-documentpreview locale dictionaries (en).
var documentPreviewLocaleEN = map[string]string{
	"loading":                  "Rendering document...",
	"changed":                  "The file has changed, showing the previous content.",
	"reload":                   "Read the file again",
	"autoRefresh.enable":       "Enable auto refresh",
	"autoRefresh.disable":      "Disable auto refresh",
	"wrap.disable":             "Turn off line wrap",
	"resourceUnavailable":      "The file resource service is unavailable.",
	"unsupportedFile":          "Preview is not available for this file type yet.",
	"error.notFound":           "File not found. It may have been moved or deleted.",
	"error.tooLarge":           "This page exceeds the {limit} limit and cannot be read.",
	"error.notText":            "Preview is not available for this file type yet.",
	"error.notRegularFile":     "Not a regular file, nothing to display.",
	"excel.invalid":            "This spreadsheet could not be opened. Check its format, contents, or password protection.",
	"excel.tooLarge":           "This workbook exceeds the preview size limit.",
	"excel.timeout":            "Opening this workbook timed out. Try a smaller file.",
	"excel.encoding":           "This text encoding could not be read. Save the file as UTF-8 or UTF-16 with a BOM and retry.",
	"excel.formulaWarning":     "This workbook contains formulas. Displayed results may be missing or inaccurate.",
	"excel.unsupportedNotice":  "This preview does not support {features} in this workbook. Open it in a system application for the full experience.",
	"excel.charts":             "charts",
	"excel.shapes":             "shapes",
	"excel.conditionalFormat":  "conditional formatting",
	"html.frame":               "HTML document preview",
	"html.failed":              "This HTML document could not be previewed.",
	"image.zoomControls":       "Zoom controls",
	"image.zoomMenu":           "Choose zoom",
	"image.zoomFitWidth":       "Fit width",
	"image.zoomValue":          "{percent}%",
	"image.preview":            "Image preview: {name}",
	"image.failed":             "This image could not be displayed.",
	"image.unsupported":        "Image preview requires the complete file contents.",
	"markdown.footnotes":       "Footnotes",
	"office.viewMissingFonts":  "Missing fonts: {count}. Click to view.",
	"office.missingFontsTitle": "Missing fonts",
	"office.missingFontsDesc":  "These fonts are unavailable for this preview. Text and layout may differ from the original document.",
	"office.missingFontsCount": "Fonts: {count}",
	"office.closeDetails":      "Close font details",
	"office.unavailable":       "Office previews are unavailable. Enable the document preview service on the computer running Whale Isle.",
	"office.tooLarge":          "The Office file or converted PDF exceeds the preview size limit. Reduce the file size or adjust the preview configuration.",
	"office.failed":            "Office conversion did not produce a usable PDF. Check the file and try again.",
	"office.timeout":           "Office conversion timed out. Try again.",
	"office.busy":              "Office preview is busy. Try again shortly.",
	"office.changed":           "The file changed while being read. Reopen the preview.",
	"pdf.pageImage":            "PDF page {page}",
	"pdf.rendering":            "Rendering page…",
	"pdf.failed":               "Cannot display PDF: {message}",
	"pdf.password":             "This PDF requires a password; password-protected previews are not supported.",
	"pdf.workerFailed":         "The PDF rendering process could not continue. Please retry.",
	"pdf.unsupported":          "PDF preview requires the complete file contents.",
}

// FilesLocale returns the localized string from the dsh-client-ui-files dictionary.
func FilesLocale(key string) string {
	if i18n.Locale() == "en" {
		if v, ok := filesLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := filesLocaleZH[key]; ok {
		return v
	}
	return key
}

// SidebarFilesLocale returns the localized string from the dsh-client-ui-sidebar-files dictionary.
func SidebarFilesLocale(key string) string {
	if i18n.Locale() == "en" {
		if v, ok := sidebarFilesLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := sidebarFilesLocaleZH[key]; ok {
		return v
	}
	return key
}

// DeliverablesLocale returns the localized string from the dsh-client-ui-deliverables dictionary.
func DeliverablesLocale(key string) string {
	if i18n.Locale() == "en" {
		if v, ok := deliverablesLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := deliverablesLocaleZH[key]; ok {
		return v
	}
	return key
}

// OpenInAppLocale returns the localized string from the dsh-client-ui-open-in-app dictionary.
func OpenInAppLocale(key string) string {
	if i18n.Locale() == "en" {
		if v, ok := openInAppLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := openInAppLocaleZH[key]; ok {
		return v
	}
	return key
}

// OpenInAppLocaleFormat formats a dsh-client-ui-open-in-app template with {placeholder} replacements.
func OpenInAppLocaleFormat(key string, vars map[string]string) string {
	text := OpenInAppLocale(key)
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

// DocumentPreviewLocale returns the localized string from the dsh-client-ui-sidebar-documentpreview dictionary.
func DocumentPreviewLocale(key string) string {
	if i18n.Locale() == "en" {
		if v, ok := documentPreviewLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := documentPreviewLocaleZH[key]; ok {
		return v
	}
	return key
}

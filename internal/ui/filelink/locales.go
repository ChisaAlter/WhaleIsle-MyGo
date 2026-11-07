package filelink

import "strings"

// referenceLocaleZH mirrors @deepseek-ai/dsh-client-ui-reference, dsh-client-ui-files,
// dsh-client-ui-deliverables, and dsh-client-ui-open-in-app reference/mention strings (zh).
var referenceLocaleZH = map[string]string{
	"section.files":               "文件与文件夹",
	"candidate.noCwd":             "（无工作目录）",
	"fileOpen.failed":             "无法打开“{path}”，请重试或关闭此提示",
	"time.days":                   "{n}天",
	"time.years":                  "{n}年",
	"mention":                     "引用到输入框",
	"copy.mention":                "复制引用",
	"copy.relative":               "复制相对路径",
	"copy.absolute":               "复制绝对路径",
	"open.folder":                 "在文件夹中显示",
	"open.system":                 "系统默认程序",
	"path.appsError":              "无法获取应用列表",
	"shortcut.busy":               "正在打开工作区",
	"shortcut.unavailable":        "当前工作区或本地应用不可用",
	"path.reveal":                 "显示文件位置",
	"path.revealError":            "无法显示文件位置，请重试",
	"presented.nativeUnavailable": "此文件没有可用的主机路径，请在应用内预览",
	"presented.hostError":         "无法读取主机桌面信息",
	"presented.preview":           "点击预览",
	"presented.expandAria":        "展开全部 {count} 个交付文件",
	"presented.collapseAria":      "收起交付文件列表",
	"presented.error":             "打开失败，点击重试",
}

// referenceLocaleEN mirrors @deepseek-ai/dsh-client-ui-reference, dsh-client-ui-files,
// dsh-client-ui-deliverables, and dsh-client-ui-open-in-app reference/mention strings (en).
var referenceLocaleEN = map[string]string{
	"section.files":               "Files & folders",
	"candidate.noCwd":             "(no cwd)",
	"fileOpen.failed":             "Couldn’t open “{path}”. Try again or close this message.",
	"time.days":                   "{n}d",
	"time.years":                  "{n}y",
	"mention":                     "Mention in composer",
	"copy.mention":                "Copy mention",
	"copy.relative":               "Copy relative path",
	"copy.absolute":               "Copy absolute path",
	"open.folder":                 "Show in folder",
	"open.system":                 "System default",
	"path.appsError":              "Could not load applications",
	"shortcut.busy":               "Opening workspace",
	"shortcut.unavailable":        "Current workspace or local application unavailable",
	"path.reveal":                 "Show file location",
	"path.revealError":            "Could not show the file location. Try again.",
	"presented.nativeUnavailable": "This file has no available Host path. Preview it in the app.",
	"presented.hostError":         "Could not read the Host desktop information",
	"presented.preview":           "Click to preview",
	"presented.expandAria":        "Show all {count} delivered files",
	"presented.collapseAria":      "Collapse delivered files",
	"presented.error":             "Could not open. Click to retry.",
}

// Locale returns the localized reference/mention string for the given locale ("zh" or "en").
func Locale(locale, key string) string {
	if locale == "en" {
		if v, ok := referenceLocaleEN[key]; ok {
			return v
		}
	}
	if v, ok := referenceLocaleZH[key]; ok {
		return v
	}
	return key
}

// FormatLocale formats a localized reference/mention string with {placeholder} replacements.
func FormatLocale(locale, key string, vars map[string]string) string {
	text := Locale(locale, key)
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

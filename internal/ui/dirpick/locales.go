package dirpick

import "strings"

type localePair struct {
	ZH string
	EN string
}

var dirPickLocaleCatalog = map[string]localePair{
	"drill.aria": {
		ZH: "进入目录",
		EN: "Browse folder",
	},
	"drill.hint": {
		ZH: "进入目录",
		EN: "Browse folder",
	},
	"crumbs.aria": {
		ZH: "目录导航",
		EN: "Folder navigation",
	},
	"folderError.title": {
		ZH: "无法打开文件夹",
		EN: "Couldn’t open folder",
	},
	"shortcut.noPicker": {
		ZH: "目录选择器不可用",
		EN: "Directory picker unavailable",
	},
	"shortcut.directoryBusy": {
		ZH: "正在选择或添加工作区",
		EN: "Selecting or adding a workspace",
	},
}

// DirPickLocaleText returns the localized directory picker label or status string for key.
func DirPickLocaleText(key string) string {
	if pair, ok := dirPickLocaleCatalog[strings.TrimSpace(key)]; ok {
		return dirText(pair.ZH, pair.EN)
	}
	return ""
}

// CrumbNavigationAriaLabel returns the localized aria label for the directory breadcrumb trail.
func CrumbNavigationAriaLabel() string {
	return DirPickLocaleText("crumbs.aria")
}

// DrillFolderAriaLabel returns the localized aria label for drilling into a subdirectory.
func DrillFolderAriaLabel() string {
	return DirPickLocaleText("drill.aria")
}

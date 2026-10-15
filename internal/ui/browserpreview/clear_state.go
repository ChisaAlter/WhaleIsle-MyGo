package browserpreview

// The same confirmation cannot be reused for another cleanup kind, page or
// navigation, and is consumed before starting an asynchronous native operation.
type browserDataConfirmation[Host comparable] struct {
	kind  string
	host  Host
	epoch uint64
}

func isBrowserDataClear(kind string) bool {
	return kind == "cache" || kind == "cookies" || kind == "site-data"
}

func takeBrowserDataConfirmation[Host comparable](slot **browserDataConfirmation[Host], kind string, host Host, epoch uint64) bool {
	if slot == nil || *slot == nil || !isBrowserDataClear(kind) {
		return false
	}
	confirm := *slot
	if confirm.kind != kind || confirm.host != host || confirm.epoch != epoch {
		return false
	}
	*slot = nil
	return true
}

const browserCookieClearNotice = "清除当前标签所属分区的全部 Cookie（含登录 Cookie），同分区全部页签可能退出登录；保留站点存储、Service Worker 和 HTTP 缓存，其他工作区/插件不受影响。继续？"

const browserSiteDataClearNotice = "永久清除当前分区所有站点的 Cookie、localStorage、IndexedDB、WebSQL 和 Service Worker；同分区全部页签可能退出登录，离线数据/未同步内容可能丢失。保留 HTTP 缓存、CacheStorage、文件系统、历史记录和已存密码；其他工作区/插件不受影响。无法撤销，继续？"

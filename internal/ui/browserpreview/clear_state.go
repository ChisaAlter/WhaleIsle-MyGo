package browserpreview

// The same confirmation cannot be reused for another cleanup kind, page or
// navigation, and is consumed before starting an asynchronous native operation.
type browserDataConfirmation[Host comparable] struct {
	kind  string
	host  Host
	epoch uint64
}

func isBrowserDataClear(kind string) bool { return kind == "cache" || kind == "cookies" }

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

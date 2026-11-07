package sidebar

import (
	"strings"

	"whaleisle/internal/ui/i18n"
)

// sidebarZhCatalog and sidebarEnCatalog mirror @deepseek-ai/dsh-client-ui-sidebar/lib/types/client/locales.js
// and @deepseek-ai/dsh-client-ui-layout/lib/types/client/shortcut-locales.js.
var sidebarZhCatalog = map[string]string{
	"session.new":       "新会话",
	"session.new.label": "新建会话",
	"toggle.open":       "打开侧边栏",
	"toggle.collapse":   "收起侧边栏",
	"tab.sessions":      "会话",
	"panels.label":      "全局面板",
	"layout.toggle":     "展开／收起左侧栏",
}

var sidebarEnCatalog = map[string]string{
	"session.new":       "New Session",
	"session.new.label": "New session",
	"toggle.open":       "Open sidebar",
	"toggle.collapse":   "Collapse sidebar",
	"tab.sessions":      "Sessions",
	"panels.label":      "Global panels",
	"layout.toggle":     "Toggle left sidebar",
}

func sidebarLocaleText(key string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(i18n.Locale())), "en") {
		if v, ok := sidebarEnCatalog[key]; ok {
			return v
		}
	}
	if v, ok := sidebarZhCatalog[key]; ok {
		return v
	}
	return key
}

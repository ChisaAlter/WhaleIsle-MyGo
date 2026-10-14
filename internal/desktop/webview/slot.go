package webview

import (
	"net/url"
	"strings"
)

// Query key read by the harness web shell to scope the page to one plugin's
// mounted Cordis slots (hide the rest of the desktop chrome). Base origin is
// always runtime-provided — never baked in at build time.
const DesktopPluginQuery = "desktopPlugin"

// Well-known desktop plugin IDs accepted by SlotURL (vendor dir / short alias).
// Values are normalized to the Cordis package name served under /plugins/<id>/.
const (
	PluginWhale      = "dsh-whale"
	PluginDshbot     = "dshbot"
	PluginIM         = "dsh-im"
	PluginUsagePanel = "dsh-usage-panel"
	PluginMarket     = "market"
)

// pluginPackage maps aliases (vendor dir, insert id, short name) → npm package
// name used as desktopPlugin. Identity matches harness overlay Package fields
// and /plugins/<package>/ client module ids.
var pluginPackage = map[string]string{
	PluginWhale:                     "dsh-whale",
	PluginDshbot:                    "dshbot",
	"dsh-bot":                       "dshbot",
	PluginIM:                        "@xmanrui/dsh-im",
	"xmanrui-dsh-im":                "@xmanrui/dsh-im",
	"@xmanrui/dsh-im":               "@xmanrui/dsh-im",
	PluginUsagePanel:                "dsh-usage-panel",
	"usage-stats":                   "dsh-usage-panel",
	PluginMarket:                    "@deepseek-ai/dsh-client-ui-settings-market",
	"ui-settings-market":            "@deepseek-ai/dsh-client-ui-settings-market",
	"dsh-client-ui-settings-market": "@deepseek-ai/dsh-client-ui-settings-market",
	"@deepseek-ai/dsh-client-ui-settings-market": "@deepseek-ai/dsh-client-ui-settings-market",
}

// CanonicalPluginID returns the Cordis package name for a known alias, or the
// trimmed pluginID unchanged when unknown (marketplace / third-party slots).
func CanonicalPluginID(pluginID string) string {
	pluginID = strings.TrimSpace(pluginID)
	if pluginID == "" {
		return ""
	}
	if pkg, ok := pluginPackage[pluginID]; ok {
		return pkg
	}
	return pluginID
}

// SlotURL builds a harness frontend URL scoped to one plugin's mounted slots.
//
// URL shape:
//
//	{base}/?desktopPlugin={package}[&k=v…]
//
// base is a harness web origin or browse URL (scheme://host[:port][/…]).
// Empty pluginID yields the site root (no desktopPlugin). extraQuery is merged
// after desktopPlugin; the desktopPlugin key in extraQuery is ignored.
func SlotURL(base, pluginID string, extraQuery url.Values) string {
	browse := normalizeSlotBase(base)
	if browse == "" {
		return ""
	}
	u, err := url.Parse(browse)
	if err != nil {
		return browse
	}
	q := u.Query()
	for k, vs := range extraQuery {
		if k == "" || k == DesktopPluginQuery {
			continue
		}
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	if id := CanonicalPluginID(pluginID); id != "" {
		q.Set(DesktopPluginQuery, id)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// PluginSlotURL is a convenience wrapper: SlotURL(HarnessBrowseURL(origin), pluginID, nil).
func PluginSlotURL(origin, pluginID string) string {
	return SlotURL(HarnessBrowseURL(origin), pluginID, nil)
}

func normalizeSlotBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	if !strings.Contains(base, "://") {
		return HarnessBrowseURL(base)
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
	default:
		return ""
	}
	if u.Path == "" || u.Path == "/" {
		// Drop any existing query on a root browse URL; SlotURL rebuilds it.
		return HarnessBrowseURL(u.Scheme + "://" + u.Host)
	}
	return u.String()
}

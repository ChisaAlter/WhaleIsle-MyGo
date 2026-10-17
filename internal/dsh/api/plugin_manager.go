package api

import (
	"sort"
	"strings"
)

const listPluginsMethod = "pluginManager/listPlugins"

// PluginManagerMethods returns every registered pluginManager/* RPC.
func PluginManagerMethods() []string {
	var out []string
	for method := range Registry {
		if strings.HasPrefix(method, "pluginManager/") {
			out = append(out, method)
		}
	}
	sort.Strings(out)
	return out
}

// HasListPluginsRPC reports whether codegen includes pluginManager/listPlugins.
func HasListPluginsRPC() bool {
	_, ok := Registry[listPluginsMethod]
	return ok
}

// ListPluginsMethod is the dsh-plugin-manager list RPC when present.
func ListPluginsMethod() string {
	if HasListPluginsRPC() {
		return listPluginsMethod
	}
	return ""
}

package harness

import "strings"

// looksLikePluginTreeFailure classifies composition / loader-tree failures
// the same way Electron's plugin-tree-failure.js does.
func looksLikePluginTreeFailure(text string) bool {
	blob := strings.ToLower(text)
	if blob == "" {
		return false
	}
	markers := []string{
		"plugin tree failed to load",
		"cannot resolve profile bundle",
		"failed to apply loader entry",
		"entries did not activate",
	}
	for _, m := range markers {
		if strings.Contains(blob, m) {
			return true
		}
	}
	if strings.Contains(blob, "err_module_not_found") {
		return true
	}
	if (strings.Contains(blob, "cannot find package") || strings.Contains(blob, "cannot find module")) &&
		strings.Contains(blob, "imported from") {
		return true
	}
	if !strings.Contains(blob, "client-modules:") {
		return false
	}
	return strings.Contains(blob, "clientpackagecompositionerror") ||
		strings.Contains(blob, "composition failed") ||
		strings.Contains(blob, "failed to compose") ||
		strings.Contains(blob, "构图失败")
}

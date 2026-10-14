package harness

import "testing"

func TestLooksLikePluginTreeFailure(t *testing.T) {
	yes := []string{
		"Plugin tree failed to load",
		"Error: ERR_MODULE_NOT_FOUND",
		"Cannot find package 'foo' imported from C:\\x\\profiles\\web\\",
		"client-modules: ClientPackageCompositionError",
		"client-modules: 构图失败",
	}
	no := []string{
		"boom",
		"dsh web exited",
		"client-modules: some unrelated route",
	}
	for _, s := range yes {
		if !looksLikePluginTreeFailure(s) {
			t.Fatalf("want true: %q", s)
		}
	}
	for _, s := range no {
		if looksLikePluginTreeFailure(s) {
			t.Fatalf("want false: %q", s)
		}
	}
}

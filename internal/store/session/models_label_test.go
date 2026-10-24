package session

import (
	"testing"

	"whaleisle/internal/dsh/api"
)

// TestLabelForSelectionUsesTheNativeSlash keeps the composer's model seat in the
// installed hero format. The native seat reads provider/model with no spaces
// (DOM aria-label "选择模型，当前 deepseek-official/deepseek-flash"), so the
// fallback used when the catalogue has no matching option must not pad the
// separator.
func TestLabelForSelectionUsesTheNativeSlash(t *testing.T) {
	sel := api.ModelCatalogDefault{Provider: "deepseek-official", Model: "deepseek-flash"}
	if got := labelForSelection(sel, nil); got != "deepseek-official/deepseek-flash" {
		t.Fatalf("label=%q", got)
	}
	options := []ModelOption{{Provider: "deepseek-official", Model: "deepseek-flash", Label: "DeepSeek V4.1 Flash"}}
	if got := labelForSelection(sel, options); got != "DeepSeek V4.1 Flash" {
		t.Fatalf("catalogue label=%q", got)
	}
}

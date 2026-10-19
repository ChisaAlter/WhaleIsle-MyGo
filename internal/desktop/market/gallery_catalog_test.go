package market

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestCatalogRetainsOnlyFirstTwelveScreenshotStrings(t *testing.T) {
	images := []any{false, 42, nil}
	for i := 0; i < 15; i++ {
		images = append(images, fmt.Sprintf("https://owned.invalid/%d.png", i))
	}
	for _, screenshots := range []any{images, "not an array", nil} {
		raw, err := json.Marshal(map[string]any{"plugins": []any{map[string]any{"owner": "owned", "name": "plugin", "url": "https://github.com/owned/plugin", "screenshots": screenshots}}})
		if err != nil {
			t.Fatal(err)
		}
		catalog, err := decodeCatalog(raw, "en")
		if err != nil || len(catalog.Items) != 1 {
			t.Fatalf("optional screenshot metadata discarded its plugin: %#v %v", catalog, err)
		}
		got := catalog.Items[0].Screenshots
		if _, array := screenshots.([]any); array {
			if len(got) != 12 || got[0] != "https://owned.invalid/0.png" || got[11] != "https://owned.invalid/11.png" {
				t.Fatalf("screenshot order, type filtering or limit changed: %v", got)
			}
		} else if len(got) != 0 {
			t.Fatalf("non-array screenshot metadata was used: %v", got)
		}
	}
}

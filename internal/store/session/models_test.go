package session

import (
	"testing"

	"whaleisle/internal/dsh/api"
)

func TestFlattenModelCatalog(t *testing.T) {
	cat := api.ModelCatalog{
		Default: api.ModelCatalogDefault{Provider: "p1", Model: "m1"},
		Groups: []api.ModelCatalogGroupsItem{{
			Id:   "p1",
			Name: "Provider One",
			Models: []api.ModelCatalogGroupsItemModelsItem{
				{Id: "m1", Name: "Model One"},
				{Id: "m2", Name: "Model Two"},
			},
		}},
	}
	opts := flattenModelCatalog(cat)
	if len(opts) != 2 {
		t.Fatalf("len = %d", len(opts))
	}
	if got := labelForSelection(cat.Default, opts); got != opts[0].Label {
		t.Fatalf("default label = %q want %q", got, opts[0].Label)
	}
}

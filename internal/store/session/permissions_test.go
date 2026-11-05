package session

import (
	"testing"

	"whaleisle/internal/dsh/api"
)

func TestFlattenPermissionCatalog(t *testing.T) {
	desc := "full access"
	cat := api.PermissionCatalog{
		DefaultPreset: "read-only",
		DefaultOptions: []api.PermissionCatalogOptionsItem{
			{Value: "read-only", Name: "Read only"},
			{Value: "future-only", Name: "Future sessions only"},
		},
		Options: []api.PermissionCatalogOptionsItem{
			{Value: "read-only", Name: "Read only duplicate"},
			{Value: "workspace-write", Name: "Workspace write", Description: &desc},
		},
	}
	opts := flattenPermissionCatalog(cat)
	if len(opts) != 2 {
		t.Fatalf("len = %d", len(opts))
	}
	if opts[0].Value != "read-only" || opts[1].Value != "workspace-write" {
		t.Fatalf("values = %v, %v", opts[0].Value, opts[1].Value)
	}
	// Installed PermissionSelect consumes options; defaultOptions belongs only
	// to the future-session settings controller and must not replace its labels.
	if got := labelForPresetValue(cat.DefaultPreset, opts); got != "Read only duplicate" {
		t.Fatalf("current-session label = %q want %q", got, "Read only duplicate")
	}
}

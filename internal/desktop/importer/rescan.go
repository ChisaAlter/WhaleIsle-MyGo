package importer

import "context"

// RescanCatalog refreshes the current source and extra roots without losing
// the user's current selection. A nonempty source chooses a new source home.
func RescanCatalog(previous Catalog, source string) (Catalog, error) {
	return RescanCatalogContext(context.Background(), previous, source)
}

func RescanCatalogContext(ctx context.Context, previous Catalog, source string) (Catalog, error) {
	if source == "" {
		source = previous.SourceHome
	}
	cat, err := BuildCatalogContext(ctx, source, previous.DestHome)
	if err != nil {
		return cat, err
	}
	for _, root := range previous.ExtraSkillRoots {
		if err := ctx.Err(); err != nil {
			return cat, err
		}
		AppendSkillDirectory(&cat, root)
	}
	// A temporarily unavailable drive/root remains part of the scan request,
	// so the next rescan can recover it without asking the user to add it again.
	cat.ExtraSkillRoots = append([]string(nil), previous.ExtraSkillRoots...)
	RetainCatalogSelections(&cat, previous)
	return cat, nil
}

// Rescans retain existing selections; newly discovered rows remain unselected.
func RetainCatalogSelections(cat *Catalog, previous Catalog) {
	mergeCatalogSelections(cat, previous, false)
}

// MergeCatalogSelections overlays edits to known rows without changing newly
// added items, such as selected skills from an explicitly added directory.
func MergeCatalogSelections(cat *Catalog, previous Catalog) {
	mergeCatalogSelections(cat, previous, true)
}

func mergeCatalogSelections(cat *Catalog, previous Catalog, keepNew bool) {
	for _, category := range []string{"sessions", "skills", "plugins", "mcp", "settings", "presets"} {
		selected := map[string]bool{}
		for _, row := range *previous.ItemsFor(category) {
			selected[row.ID+"\x00"+row.Rel] = row.Selected
		}
		rows := cat.ItemsFor(category)
		for i := range *rows {
			row := &(*rows)[i]
			value, exists := selected[row.ID+"\x00"+row.Rel]
			if !keepNew || exists {
				row.Selected = !row.Unsupported && value
			}
		}
	}
}

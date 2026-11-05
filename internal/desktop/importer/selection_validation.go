package importer

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"whaleisle/internal/store/storelocale"
)

// Match Electron's admission rule: distinct selected skill sources must not
// silently overwrite one shared destination, even when Overwrite is enabled.
func validateSelectedSkills(cat Catalog, sel ApplySelection) error {
	rows := make(map[string]CatalogItem, len(cat.Skills))
	for _, row := range cat.Skills {
		rows[row.ID] = row
	}
	selected := map[string]string{}
	for _, id := range sel.SelectedSkillIDs {
		row, ok := rows[id]
		if !ok {
			continue
		}
		target := row.Label
		if runtime.GOOS == "windows" {
			target = strings.ToLower(target)
		}
		if previous, ok := selected[target]; ok && previous != id {
			return errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreImporterSelectedSkillsSAndSTargetTheSameDest), previous, id, row.Label))
		}
		selected[target] = id
	}
	return nil
}

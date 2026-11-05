package importer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"whaleisle/internal/store/storelocale"
)

// CopyOnlyNotice documents import behavior without exposing configuration
// secrets. It is a function so the launcher can follow the active locale.
func CopyOnlyNotice() string { return storelocale.T(storelocale.KeyStoreImporterCopyOnlyNotice) }

func (s ApplySelection) Count() int {
	n := len(s.SelectedRels) + len(s.SelectedSkillIDs) + len(s.SelectedPluginIDs) + len(s.SelectedMcpIDs) + len(s.SelectedSettingIDs) + len(s.SelectedPresetIDs)
	if s.ImportAttachments {
		n++
	}
	return n
}

// Snapshot isolates the worker from mutable checkbox state.
func (c Catalog) Snapshot() Catalog {
	c.SkillRoots = append([]string(nil), c.SkillRoots...)
	c.ExtraSkillRoots = append([]string(nil), c.ExtraSkillRoots...)
	c.Sessions = append([]CatalogItem(nil), c.Sessions...)
	c.Skills = append([]CatalogItem(nil), c.Skills...)
	c.Plugins = append([]CatalogItem(nil), c.Plugins...)
	c.MCP = append([]CatalogItem(nil), c.MCP...)
	c.Settings = append([]CatalogItem(nil), c.Settings...)
	c.Presets = append([]CatalogItem(nil), c.Presets...)
	return c
}

func Confirmation(c Catalog, s ApplySelection) string {
	policy := storelocale.T(storelocale.KeyStoreImporterExistingSameNameDataSkippedAndKept)
	if s.Overwrite {
		policy = storelocale.T(storelocale.KeyStoreImporterExistingSameNameDataOverwrittenSettings)
	}
	var preview []string
	for _, group := range []struct {
		label string
		ids   []string
		rows  []CatalogItem
	}{
		{storelocale.T(storelocale.KeyStoreImporterSettings), s.SelectedSettingIDs, c.Settings}, {"MCP", s.SelectedMcpIDs, c.MCP},
	} {
		for _, id := range group.ids {
			action := storelocale.T(storelocale.KeyStoreImporterSkippedMissingOrUnsupported)
			for _, row := range group.rows {
				if row.ID == id && !row.Unsupported {
					action = storelocale.T(storelocale.KeyStoreImporterAdd)
					if row.Conflict {
						action = storelocale.T(storelocale.KeyStoreImporterSkipAndKeep)
						if s.Overwrite {
							action = storelocale.T(storelocale.KeyStoreImporterReplaceSelected)
						}
					}
					break
				}
			}
			preview = append(preview, group.label+" / "+id+"："+action)
		}
	}
	for _, id := range s.SelectedPluginIDs {
		for _, row := range c.Plugins {
			if row.ID != id {
				continue
			}
			spec := PluginReinstallSpec(row.ID, row.Spec)
			action := storelocale.T(storelocale.KeyStoreImporterDownloadAndInstallAgain)
			if row.Unsupported || spec == "" {
				action = storelocale.T(storelocale.KeyStoreImporterSkippedUnsupported)
			}
			if row.Conflict && !s.Overwrite {
				action = storelocale.T(storelocale.KeyStoreImporterSkippedAlreadyInstalled)
			}
			preview = append(preview, storelocale.T(storelocale.KeyStoreImporterPlugin2)+id+"："+action+storelocale.T(storelocale.KeyStoreImporterSource)+spec)
			break
		}
	}
	detail := ""
	if len(preview) > 0 {
		detail = storelocale.T(storelocale.KeyStoreImporterNNscanPreviewConflictsAreRecheckedBefo) + strings.Join(preview, "\n")
	}
	return fmt.Sprintf(storelocale.T(storelocale.KeyStoreImporterSourceReadOnlySNdestinationSNDItems), c.SourceHome, c.DestHome, s.Count(), policy, CopyOnlyNotice()+detail)
}

// ValidateRoots rejects recursive/self import, including existing symlink aliases.
func ValidateRoots(source, dest string) error {
	canonical := func(path string) (string, error) {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		current := absolute
		var tail []string
		for {
			resolved, err := filepath.EvalSymlinks(current)
			if err == nil {
				for i := len(tail) - 1; i >= 0; i-- {
					resolved = filepath.Join(resolved, tail[i])
				}
				return resolved, nil
			}
			if !os.IsNotExist(err) {
				return "", err
			}
			parent := filepath.Dir(current)
			if parent == current {
				return "", err
			}
			tail = append(tail, filepath.Base(current))
			current = parent
		}
	}
	src, err := canonical(source)
	if err != nil {
		return err
	}
	dst, err := canonical(dest)
	if err != nil {
		return err
	}
	inside := func(root, candidate string) bool {
		rel, err := filepath.Rel(root, candidate)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	if inside(src, dst) || inside(dst, src) {
		return errors.New(storelocale.T(storelocale.KeyStoreImporterTheImportSourceAndDestinationMustNotB))
	}
	return nil
}

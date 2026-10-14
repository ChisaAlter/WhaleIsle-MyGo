package importer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ApplySelection mirrors Electron runImport checkbox payload (copy-only subset).
type ApplySelection struct {
	SelectedRels       []string
	SelectedSkillIDs   []string
	SelectedMcpIDs     []string
	SelectedSettingIDs []string
	SelectedPresetIDs  []string
	ImportAttachments  bool
	Overwrite          bool
}

// ApplyResult summarizes a completed ApplyCatalog.
type ApplyResult struct {
	OK      bool
	Empty   bool
	Copied  int
	Skipped int
	Message string
	// Journal is a per-item line log (Electron runImport journal list).
	Journal []string
}

// ErrEmptySelection means nothing was selected (Electron empty → no writes).
var ErrEmptySelection = errors.New("importer: empty selection")

// ApplyCatalog copies selected catalog items from SourceHome → DestHome.
// Plugins are not reinstalled (needs Electron installPlugin channel).
func ApplyCatalog(cat Catalog, sel ApplySelection) (ApplyResult, error) {
	return ApplyCatalogProgress(context.Background(), cat, sel, nil)
}

// ApplyCatalogProgress reports completed items and retains their journal on failure.
// Cancellation is checked between items; a completed item is never rolled back.
func ApplyCatalogProgress(ctx context.Context, cat Catalog, sel ApplySelection, progress func(ApplyResult)) (result ApplyResult, resultErr error) {
	empty := len(sel.SelectedRels) == 0 && len(sel.SelectedSkillIDs) == 0 &&
		len(sel.SelectedMcpIDs) == 0 && len(sel.SelectedSettingIDs) == 0 &&
		len(sel.SelectedPresetIDs) == 0 && !sel.ImportAttachments
	if empty {
		return ApplyResult{OK: true, Empty: true, Message: "未选择任何项，未写入。", Journal: []string{"未选择任何项，未写入。"}}, nil
	}
	if cat.SourceHome == "" || cat.DestHome == "" {
		return ApplyResult{}, ErrNotDirectory
	}
	if err := ValidateRoots(cat.SourceHome, cat.DestHome); err != nil {
		return ApplyResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ApplyResult{}, err
	}
	if err := os.MkdirAll(cat.DestHome, 0o755); err != nil {
		return ApplyResult{}, err
	}

	var copied, skipped int
	var journal []string
	defer func() {
		if resultErr != nil {
			result = ApplyResult{Copied: copied, Skipped: skipped, Message: fmt.Sprintf("导入未完成：已复制 %d 项，跳过 %d 项；已完成项保留。%v", copied, skipped, resultErr), Journal: append([]string(nil), journal...)}
			result.Journal = append(result.Journal, result.Message)
		}
	}()
	note := func(line string) {
		journal = append(journal, line)
		if progress != nil {
			progress(ApplyResult{Copied: copied, Skipped: skipped, Message: line, Journal: append([]string(nil), journal...)})
		}
	}
	// Sessions
	byRel := map[string]CatalogItem{}
	for _, it := range cat.Sessions {
		byRel[it.Rel] = it
	}
	for _, rel := range sel.SelectedRels {
		if err := ctx.Err(); err != nil {
			return ApplyResult{}, err
		}
		it, ok := byRel[rel]
		if !ok || it.Unsupported {
			skipped++
			note("跳过会话 " + rel)
			continue
		}
		dst := filepath.Join(cat.DestHome, "sessions", filepath.FromSlash(rel))
		if it.Conflict && !sel.Overwrite {
			skipped++
			note("跳过会话（冲突） " + rel)
			continue
		}
		if err := copyDir(it.Abs, dst, sel.Overwrite); err != nil {
			return ApplyResult{}, fmt.Errorf("session %s: %w", rel, err)
		}
		copied++
		note("复制会话 " + rel)
	}

	// Skills
	bySkill := map[string]CatalogItem{}
	for _, it := range cat.Skills {
		bySkill[it.ID] = it
	}
	for _, id := range sel.SelectedSkillIDs {
		if err := ctx.Err(); err != nil {
			return ApplyResult{}, err
		}
		it, ok := bySkill[id]
		if !ok {
			skipped++
			note("跳过技能 " + id)
			continue
		}
		dst := filepath.Join(cat.DestHome, "skills", it.Label)
		if it.Conflict && !sel.Overwrite {
			skipped++
			note("跳过技能（冲突） " + it.Label)
			continue
		}
		if err := copyDir(it.Abs, dst, sel.Overwrite); err != nil {
			return ApplyResult{}, fmt.Errorf("skill %s: %w", id, err)
		}
		copied++
		note("复制技能 " + it.Label)
	}

	// MCP — copy whole file when any id selected (merge-lite).
	if len(sel.SelectedMcpIDs) > 0 {
		if err := ctx.Err(); err != nil {
			return ApplyResult{}, err
		}
		src := filepath.Join(cat.SourceHome, "mcp-servers.yaml")
		dst := filepath.Join(cat.DestHome, "mcp-servers.yaml")
		if entryExists(src, false) {
			if entryExists(dst, false) && !sel.Overwrite {
				skipped++
				note("跳过 MCP（冲突） mcp-servers.yaml")
			} else if err := copyFile(src, dst, sel.Overwrite); err != nil {
				return ApplyResult{}, err
			} else {
				copied++
				note("复制 MCP mcp-servers.yaml")
			}
		} else {
			for _, it := range cat.MCP {
				if !contains(sel.SelectedMcpIDs, it.ID) || it.Abs == "" {
					continue
				}
				dstDir := filepath.Join(cat.DestHome, it.ID)
				if err := copyDir(it.Abs, dstDir, sel.Overwrite); err != nil {
					return ApplyResult{}, err
				}
				copied++
				note("复制 MCP " + it.ID)
			}
		}
	}

	// Settings whitelist sections: copy settings.yaml wholesale when selected;
	// AGENTS.md copied by id.
	if len(sel.SelectedSettingIDs) > 0 {
		if err := ctx.Err(); err != nil {
			return ApplyResult{}, err
		}
		wantYAML := false
		for _, id := range sel.SelectedSettingIDs {
			if id == "agents-md" {
				src := filepath.Join(cat.SourceHome, "AGENTS.md")
				dst := filepath.Join(cat.DestHome, "AGENTS.md")
				if entryExists(src, false) {
					if entryExists(dst, false) && !sel.Overwrite {
						skipped++
						note("跳过设置（冲突） AGENTS.md")
					} else if err := copyFile(src, dst, sel.Overwrite); err != nil {
						return ApplyResult{}, err
					} else {
						copied++
						note("复制设置 AGENTS.md")
					}
				}
				continue
			}
			wantYAML = true
		}
		if wantYAML {
			src := filepath.Join(cat.SourceHome, "settings.yaml")
			dst := filepath.Join(cat.DestHome, "settings.yaml")
			if entryExists(src, false) {
				if entryExists(dst, false) && !sel.Overwrite {
					skipped++
					note("跳过设置（冲突） settings.yaml")
				} else if err := copyFile(src, dst, sel.Overwrite); err != nil {
					return ApplyResult{}, err
				} else {
					copied++
					note("复制设置 settings.yaml")
				}
			}
		}
	}

	// Presets
	byPreset := map[string]CatalogItem{}
	for _, it := range cat.Presets {
		byPreset[it.ID] = it
	}
	for _, id := range sel.SelectedPresetIDs {
		if err := ctx.Err(); err != nil {
			return ApplyResult{}, err
		}
		it, ok := byPreset[id]
		if !ok || it.Unsupported {
			skipped++
			note("跳过预设 " + id)
			continue
		}
		dst := filepath.Join(cat.DestHome, ".agent-presets", id)
		if it.Conflict && !sel.Overwrite {
			skipped++
			note("跳过预设（冲突） " + id)
			continue
		}
		if err := copyDir(it.Abs, dst, sel.Overwrite); err != nil {
			return ApplyResult{}, fmt.Errorf("preset %s: %w", id, err)
		}
		copied++
		note("复制预设 " + id)
	}

	if sel.ImportAttachments {
		if err := ctx.Err(); err != nil {
			return ApplyResult{}, err
		}
		src := filepath.Join(cat.SourceHome, "attachments")
		dst := filepath.Join(cat.DestHome, "attachments")
		if entryExists(src, true) {
			if err := copyDir(src, dst, sel.Overwrite); err != nil {
				return ApplyResult{}, err
			}
			copied++
			note("复制附件 attachments/")
		} else {
			note("跳过附件（源不存在）")
		}
	}

	msg := fmt.Sprintf("已复制 %d 项", copied)
	if skipped > 0 {
		msg += fmt.Sprintf("，跳过 %d 项（冲突未覆盖或不受支持）", skipped)
	}
	note(msg)
	return ApplyResult{OK: true, Copied: copied, Skipped: skipped, Message: msg, Journal: journal}, nil
}

// SelectionFromCatalog builds ApplySelection from current checkbox state on items.
func SelectionFromCatalog(cat Catalog, importAttachments, overwrite bool) ApplySelection {
	sel := ApplySelection{ImportAttachments: importAttachments, Overwrite: overwrite}
	for _, it := range cat.Sessions {
		if it.Selected && !it.Unsupported {
			sel.SelectedRels = append(sel.SelectedRels, it.Rel)
		}
	}
	for _, it := range cat.Skills {
		if it.Selected {
			sel.SelectedSkillIDs = append(sel.SelectedSkillIDs, it.ID)
		}
	}
	for _, it := range cat.MCP {
		if it.Selected {
			sel.SelectedMcpIDs = append(sel.SelectedMcpIDs, it.ID)
		}
	}
	for _, it := range cat.Settings {
		if it.Selected {
			sel.SelectedSettingIDs = append(sel.SelectedSettingIDs, it.ID)
		}
	}
	for _, it := range cat.Presets {
		if it.Selected && !it.Unsupported {
			sel.SelectedPresetIDs = append(sel.SelectedPresetIDs, it.ID)
		}
	}
	return sel
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func copyDir(src, dst string, overwrite bool) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entryExists(target, false) && !overwrite {
			return nil
		}
		return copyFile(path, target, overwrite)
	})
}

func copyFile(src, dst string, overwrite bool) error {
	if entryExists(dst, false) && !overwrite {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if !overwrite {
		flag = os.O_CREATE | os.O_WRONLY | os.O_EXCL
	}
	out, err := os.OpenFile(dst, flag, 0o600)
	if err != nil {
		if !overwrite && os.IsExist(err) {
			return nil
		}
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

package importer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	empty := len(sel.SelectedRels) == 0 && len(sel.SelectedSkillIDs) == 0 &&
		len(sel.SelectedMcpIDs) == 0 && len(sel.SelectedSettingIDs) == 0 &&
		len(sel.SelectedPresetIDs) == 0 && !sel.ImportAttachments
	if empty {
		return ApplyResult{OK: true, Empty: true, Message: "未选择任何项，未写入。", Journal: []string{"未选择任何项，未写入。"}}, nil
	}
	if cat.SourceHome == "" || cat.DestHome == "" {
		return ApplyResult{}, ErrNotDirectory
	}
	if err := os.MkdirAll(cat.DestHome, 0o755); err != nil {
		return ApplyResult{}, err
	}

	var copied, skipped int
	var journal []string
	note := func(line string) { journal = append(journal, line) }
	// Sessions
	byRel := map[string]CatalogItem{}
	for _, it := range cat.Sessions {
		byRel[it.Rel] = it
	}
	for _, rel := range sel.SelectedRels {
		it, ok := byRel[rel]
		if !ok || it.Unsupported {
			skipped++
			note("跳过会话 " + rel)
			continue
		}
		dst, err := safeImportTarget(filepath.Join(cat.DestHome, "sessions"), rel)
		if err != nil || !safeSourceUnder(it.Abs, filepath.Join(cat.SourceHome, "sessions")) {
			skipped++
			note("session rejected path " + rel)
			continue
		}
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
		it, ok := bySkill[id]
		if !ok {
			skipped++
			note("跳过技能 " + id)
			continue
		}
		dst, err := safeImportTarget(filepath.Join(cat.DestHome, "skills"), it.Label)
		if err != nil || !safeSourcePath(it.Abs) {
			skipped++
			note("skill rejected path " + id)
			continue
		}
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
	// MCP selection merges selected server records when the source uses the
	// standard `servers:` list. For alternate layouts, retain the existing
	// per-directory copy behavior.
	if len(sel.SelectedMcpIDs) > 0 {
		src := filepath.Join(cat.SourceHome, "mcp-servers.yaml")
		dst := filepath.Join(cat.DestHome, "mcp-servers.yaml")
		if entryExists(src, false) {
			changed, skippedMCP, mergeErr := mergeMCPFile(src, dst, sel.SelectedMcpIDs, sel.Overwrite)
			skipped += skippedMCP
			if mergeErr != nil {
				return ApplyResult{}, mergeErr
			}
			if changed {
				copied++
				note("copied selected MCP servers")
			}
		} else {
			for _, it := range cat.MCP {
				if !contains(sel.SelectedMcpIDs, it.ID) || it.Abs == "" {
					continue
				}
				dstDir, targetErr := safeImportTarget(cat.DestHome, it.ID)
				if targetErr != nil || !safeSourcePath(it.Abs) {
					skipped++
					note("MCP rejected path " + it.ID)
					continue
				}
				if err := copyDir(it.Abs, dstDir, sel.Overwrite); err != nil {
					return ApplyResult{}, err
				}
				copied++
				note("copied MCP " + it.ID)
			}
		}
	}

	// Settings whitelist sections are merged one top-level block at a time;
	// AGENTS.md remains a separate pseudo-row.
	if len(sel.SelectedSettingIDs) > 0 {
		var yamlIDs []string
		for _, id := range sel.SelectedSettingIDs {
			if id == "agents-md" {
				src := filepath.Join(cat.SourceHome, "AGENTS.md")
				dst := filepath.Join(cat.DestHome, "AGENTS.md")
				if entryExists(src, false) {
					if entryExists(dst, false) && !sel.Overwrite {
						skipped++
						note("skipped AGENTS.md conflict")
					} else if err := copyFile(src, dst, sel.Overwrite); err != nil {
						return ApplyResult{}, err
					} else {
						copied++
						note("copied AGENTS.md")
					}
				}
				continue
			}
			yamlIDs = append(yamlIDs, id)
		}
		if len(yamlIDs) > 0 {
			src := filepath.Join(cat.SourceHome, "settings.yaml")
			dst := filepath.Join(cat.DestHome, "settings.yaml")
			if entryExists(src, false) {
				changed, skippedSettings, mergeErr := mergeSettingsFile(src, dst, yamlIDs, sel.Overwrite)
				skipped += skippedSettings
				if mergeErr != nil {
					return ApplyResult{}, mergeErr
				}
				if changed {
					copied++
					note("copied selected settings")
				}
			} else {
				skipped += len(yamlIDs)
				note("settings source missing")
			}
		}
	}

	// Presets
	byPreset := map[string]CatalogItem{}
	for _, it := range cat.Presets {
		byPreset[it.ID] = it
	}
	for _, id := range sel.SelectedPresetIDs {
		it, ok := byPreset[id]
		if !ok || it.Unsupported {
			skipped++
			note("跳过预设 " + id)
			continue
		}
		dst, err := safeImportTarget(filepath.Join(cat.DestHome, ".agent-presets"), id)
		if err != nil || !safeSourcePath(it.Abs) {
			skipped++
			note("preset rejected path " + id)
			continue
		}
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

func safeImportTarget(root, rel string) (string, error) {
	root = filepath.Clean(root)
	rel = strings.TrimSpace(rel)
	if rel == "" || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
		return "", errors.New("invalid import path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid import path")
	}
	target := filepath.Join(root, clean)
	check, err := filepath.Rel(root, target)
	if err != nil || check == ".." || strings.HasPrefix(check, ".."+string(filepath.Separator)) || filepath.IsAbs(check) {
		return "", errors.New("invalid import path")
	}
	return target, nil
}

func safeSourcePath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func safeSourceUnder(path, root string) bool {
	if !safeSourcePath(path) || !safeSourcePath(root) {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// splitYAMLSections preserves top-level YAML blocks verbatim. Import only
// selects the documented settings whitelist, leaving desktop-owned sections
// untouched.
func splitYAMLSections(raw string) (preamble []string, order []string, blocks map[string][]string) {
	blocks = map[string][]string{}
	if raw == "" {
		return nil, nil, blocks
	}
	var current string
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(line, "#") {
			if idx := strings.IndexByte(line, ':'); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				if key != "" {
					current = key
					if _, ok := blocks[key]; !ok {
						order = append(order, key)
					}
				}
			}
		}
		if current == "" {
			preamble = append(preamble, line)
		} else {
			blocks[current] = append(blocks[current], line)
		}
	}
	return preamble, order, blocks
}

func renderYAMLSections(preamble, order []string, blocks map[string][]string) string {
	lines := append([]string{}, preamble...)
	for _, key := range order {
		lines = append(lines, blocks[key]...)
	}
	text := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if text == "" {
		return ""
	}
	return text + "\n"
}

func mergeSettingsFile(src, dst string, ids []string, overwrite bool) (bool, int, error) {
	if unsafeDestination(dst) {
		return false, 0, errors.New("unsafe settings destination")
	}
	srcRaw, err := os.ReadFile(src)
	if err != nil {
		return false, 0, err
	}
	dstRaw, err := os.ReadFile(dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, 0, err
	}
	_, _, sb := splitYAMLSections(string(srcRaw))
	dp, order, db := splitYAMLSections(string(dstRaw))
	changed := false
	skipped := 0
	whitelist := map[string]bool{"llm-deepseek": true, "llm-pi-ai": true, "agent-default-model": true, "vision-fallback": true, "ui-theme": true}
	for _, id := range ids {
		if id == "agents-md" || !whitelist[id] {
			continue
		}
		block, ok := sb[id]
		if !ok {
			skipped++
			continue
		}
		if _, exists := db[id]; exists && !overwrite {
			skipped++
			continue
		}
		if _, exists := db[id]; !exists {
			order = append(order, id)
		}
		db[id] = append([]string{}, block...)
		changed = true
	}
	if !changed {
		return false, skipped, nil
	}
	text := renderYAMLSections(dp, order, db)
	if err := writeAtomicText(dst, text); err != nil {
		return false, skipped, err
	}
	return true, skipped, nil
}

type yamlMCPDoc struct {
	prefix []string
	order  []string
	blocks map[string][]string
}

func parseMCPDoc(raw string) yamlMCPDoc {
	doc := yamlMCPDoc{blocks: map[string][]string{}}
	var current string
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "- id:") {
			value := strings.TrimSpace(strings.TrimPrefix(trim, "- id:"))
			value = strings.Trim(value, "\"'")
			if value != "" {
				current = value
				doc.order = append(doc.order, value)
			}
		}
		if current == "" {
			doc.prefix = append(doc.prefix, line)
		} else {
			doc.blocks[current] = append(doc.blocks[current], line)
		}
	}
	return doc
}

func renderMCPDoc(doc yamlMCPDoc) string {
	lines := append([]string{}, doc.prefix...)
	for _, id := range doc.order {
		lines = append(lines, doc.blocks[id]...)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func mergeMCPFile(src, dst string, ids []string, overwrite bool) (bool, int, error) {
	if unsafeDestination(dst) {
		return false, 0, errors.New("unsafe MCP destination")
	}
	srcRaw, err := os.ReadFile(src)
	if err != nil {
		return false, 0, err
	}
	dstRaw, err := os.ReadFile(dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, 0, err
	}
	source, target := parseMCPDoc(string(srcRaw)), parseMCPDoc(string(dstRaw))
	changed, skipped := false, 0
	for _, id := range ids {
		block, ok := source.blocks[id]
		if !ok {
			skipped++
			continue
		}
		if _, exists := target.blocks[id]; exists && !overwrite {
			skipped++
			continue
		}
		if _, exists := target.blocks[id]; !exists {
			target.order = append(target.order, id)
		}
		target.blocks[id] = append([]string{}, block...)
		changed = true
	}
	if !changed {
		return false, skipped, nil
	}
	if len(target.prefix) == 0 {
		target.prefix = []string{"servers:"}
	}
	if err := writeAtomicText(dst, renderMCPDoc(target)); err != nil {
		return false, skipped, err
	}
	return true, skipped, nil
}

func writeAtomicText(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + fmt.Sprintf(".%d.tmp", os.Getpid())
	if err := os.WriteFile(tmp, []byte(text), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func copyDir(src, dst string, overwrite bool) error {
	if !safeSourcePath(src) {
		return errors.New("source directory is not a regular directory")
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d == nil || d.Type()&os.ModeSymlink != 0 {
			return errors.New("symbolic links are not importable")
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if unsafeDestination(target) {
				return errors.New("unsafe destination directory")
			}
			return os.MkdirAll(target, 0o755)
		}
		if entryExists(target, false) && !overwrite {
			return nil
		}
		return copyFile(path, target, overwrite)
	})
}

func copyFile(src, dst string, overwrite bool) error {
	if !safeSourceFile(src) || unsafeDestination(dst) {
		return errors.New("unsafe import file")
	}
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

func safeSourceFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func unsafeDestination(path string) bool {
	for cursor := filepath.Clean(filepath.Dir(path)); ; cursor = filepath.Dir(cursor) {
		if info, err := os.Lstat(cursor); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return true
		}
		parent := filepath.Dir(cursor)
		if parent == cursor {
			break
		}
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return true
	}
	return false
}

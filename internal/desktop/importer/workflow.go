package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CopyOnlyNotice is shown in both preview and confirmation. It describes actual
// backend behavior, including limits that differ from Electron's section merge.
const CopyOnlyNotice = "设置/MCP 当前按整文件复制，不是按勾选节合并：选中任一 YAML 项会复制整份 settings.yaml / mcp-servers.yaml（含其他节）。目标已有同名文件默认跳过；启用覆盖将替换整份文件。引用凭据不迁移，导入的配置可能需要重新设置凭据才能使用。插件请在插件市场安装。"

func (s ApplySelection) Count() int {
	n := len(s.SelectedRels) + len(s.SelectedSkillIDs) + len(s.SelectedMcpIDs) + len(s.SelectedSettingIDs) + len(s.SelectedPresetIDs)
	if s.ImportAttachments {
		n++
	}
	return n
}

// Snapshot isolates the worker from mutable checkbox state.
func (c Catalog) Snapshot() Catalog {
	c.SkillRoots = append([]string(nil), c.SkillRoots...)
	c.Sessions = append([]CatalogItem(nil), c.Sessions...)
	c.Skills = append([]CatalogItem(nil), c.Skills...)
	c.Plugins = append([]CatalogItem(nil), c.Plugins...)
	c.MCP = append([]CatalogItem(nil), c.MCP...)
	c.Settings = append([]CatalogItem(nil), c.Settings...)
	c.Presets = append([]CatalogItem(nil), c.Presets...)
	return c
}

func Confirmation(c Catalog, s ApplySelection) string {
	policy := "已有同名数据：跳过并保留"
	if s.Overwrite {
		policy = "已有同名数据：覆盖；设置/MCP 将整文件替换，无法撤销"
	}
	return fmt.Sprintf("来源（只读）：%s\n目标：%s\n已选 %d 项（附件按整树计）\n%s\n\n%s\n\n只复制已确认的选择；取消不写入。", c.SourceHome, c.DestHome, s.Count(), policy, CopyOnlyNotice)
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
		return fmt.Errorf("导入来源与目标不能相同或互相包含")
	}
	return nil
}

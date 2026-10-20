package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CopyOnlyNotice documents import behavior without exposing configuration secrets.
const CopyOnlyNotice = "设置仅合并勾选的白名单节；MCP 按勾选的服务器 id 合并。未选节和服务器保留，同名项默认跳过，启用覆盖只替换所选同名项。引用凭据不迁移，配置可能需重新设置凭据；MCP 所选记录内的 header/token 随配置迁移，但不展示。OAuth 会话、跨节 YAML 锚点、非标准 MCP 目录布局不支持。支持的插件按原 GitHub/npm 版本规格重新下载安装；内置、本地、tarball 和不支持的来源跳过。构建脚本需逐项明确批准。会话、技能、预设先完整暂存再替换所选目录；附件保留目标独有内容并按覆盖选项合并。中断时恢复已登记的替换，状态不明则保留所有副本并阻止启动/后续导入。取消保留已完成项。"

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
		policy = "已有同名数据：覆盖；设置与 MCP 只替换所选同名项，无法撤销"
	}
	var preview []string
	for _, group := range []struct {
		label string
		ids   []string
		rows  []CatalogItem
	}{
		{"设置", s.SelectedSettingIDs, c.Settings}, {"MCP", s.SelectedMcpIDs, c.MCP},
	} {
		for _, id := range group.ids {
			action := "跳过（不存在或不支持）"
			for _, row := range group.rows {
				if row.ID == id && !row.Unsupported {
					action = "新增"
					if row.Conflict {
						action = "跳过并保留"
						if s.Overwrite {
							action = "替换所选项"
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
			action := "重新下载安装"
			if row.Unsupported || spec == "" {
				action = "跳过（不支持）"
			}
			if row.Conflict && !s.Overwrite {
				action = "跳过（已安装）"
			}
			preview = append(preview, "插件 / "+id+"："+action+"；来源 "+spec)
			break
		}
	}
	detail := ""
	if len(preview) > 0 {
		detail = "\n\n扫描预览（写入前重新检查冲突）：\n" + strings.Join(preview, "\n")
	}
	return fmt.Sprintf("来源（只读）：%s\n目标：%s\n已选 %d 项（附件按整树计）\n%s\n\n%s\n\n仅处理已确认的选择；此处取消不写入。", c.SourceHome, c.DestHome, s.Count(), policy, CopyOnlyNotice+detail)
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

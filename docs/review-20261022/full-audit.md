# 弹窗 / 页面 / 功能全量审计

日期：2026-10-22  
状态：进行中（第一轮盘点 + 第一批修复）

## 弹窗盘点

`ui.DialogBase` 共 **47 处 / 40 个文件**。按形态分：

### A. 已统一为 `chrome.DialogSurface`

| 区域 | 弹窗 |
| --- | --- |
| frame | 权限确认、归档确认、会话反馈、Workflow member |
| settings | 通用确认、默认权限确认 |
| terminal | 关闭运行中的终端 |
| remote | 新建/重命名/删除、关闭未保存文件 |
| composer | slash 权限确认 |

表面：28px 圆角、玻璃 + 实色打底、0.5px 发丝边、菜单阴影、遮罩淡入。

### B. 已有独立完整 chrome，但未走统一表面

| 区域 | 说明 |
| --- | --- |
| settings | 设置主模态本身：28px 玻璃，已对齐 |
| projects | `request_changes.go`：28px + 多层阴影，接近参考 |
| widgets | 通用 `ConfirmDialog`：14px、实色、阴影；被多页复用 |
| composer | 权限确认已单独统一；其余附件/语音尚未 |
| diffsurface | commit dialog：16px，自有样式 |

### C. 全屏 / 媒体 / 特殊面板（不应套 28px 小弹窗）

child image gallery、visual reply、attachment display、usage pricing、boot drawer/scene、dir picker、workspace picker。

### D. 待迁移的普通弹窗

assistant whale settings、boot 两处 drawer、chat feedback / user references、composer voice setup、diffsurface 创建分支 / review / commit、market 两处、pluginmanager 四处、projects rename/archive/modal、settings agent guide / body helpers / effect preferences / onboarding / pet / shortcuts、usage pricing、workspacepicker。

## 本轮发现并修复

### 权限确认弹窗

- **死控件**：`acknowledged := request.Acknowledged` 每帧重置，复选框永远回弹，主按钮永远禁用。
- 修复：确认状态存 `ui.Local`，跨帧稳定。
- 回归测试：`TestPermissionRiskAcknowledgementEnablesConfirm`。
- 实机：UIA 勾选 → 主按钮启用 → 确认后权限 chip 变为“完全权限”。

### 统一表面迁移

本轮迁移：settings 通用确认、默认权限确认、terminal 关闭面板、remote 两处、frame 四处（上一提交）。

## 页面巡检

设置 12 个子页全部可进入并渲染：

| 页面 | 文本节点 | 结果 |
| --- | --- | --- |
| 通用设置 | 30 | 正常 |
| 外观 | 41 | 正常 |
| 界面设置 | 26 | 正常 |
| 模型 | 17 | 正常 |
| 内置插件 | 12 | 正常 |
| 技能 | 4 | 空状态（需选会话） |
| MCP | 10 | 正常 |
| 远程 | 15 | 正常 |
| Agent 预设 | 12 | 正常 |
| 用量统计 | 4 | 空状态（此范围暂无用量） |
| 鲸鱼娘 | 22 | 正常 |
| 关于 | 10 | 正常 |

主页面：

- 插件：可进入，文本查询遇到窗口重建（导航成功）。
- 自动化任务：8 个文本节点，正常。
- 鲸鱼娘：57 个文本节点，正常。
- 设置主模态、返回、rail 全部可点。

## 下一轮

1. 迁移 `assistant` 系列 13 个 `ack := r.Acknowledged` 复选框（同类每帧重置风险）。
2. 迁移 widgets 通用 `ConfirmDialog`、settings 其余弹窗、diffsurface、market、pluginmanager、projects。
3. 对主页面逐页做可点击清单：侧栏、工作区、会话行、搜索、面板开关、终端、Git、模型选择、发送/停止。
4. 每个修复补测试；最后输出完整覆盖矩阵。

# 全量执行结果（自动化 + 实机）

日期：2026-10-22  
构建：`whaleisle-pass122.exe`  
清单：[full-audit-checklist.md](full-audit-checklist.md)  
第一轮实跑：[audit-run-20261022.md](audit-run-20261022.md)

## 1. 自动化

### UI 全量

```text
go test ./internal/ui/... -count=1
```

- 34 个 UI 包
- **1325 个 Test/Example 函数**
- **全部通过，0 FAIL**

### 全仓库

```text
go test ./... -count=1
```

真实包全部通过，包括：

- `internal/ui/*` 34 包
- `internal/store/*`
- `internal/desktop/whale`
- `internal/workflowrun`
- `scripts/nsis`

失败项：

| 包 | 原因 |
| --- | --- |
| `whaleisle/tmp`、`tmp/gofmtcheck`、`tmp/gofmt-check5/6`、`tmp/gofmtcheck2` | 临时目录残留，build failed |
| `tools/rich-history-check` | 工具包 build failed |
| `internal/desktop/importer` `TestNSISSilentInstallSmoke` | 需要 NSIS 安装器环境 |
| `cmd/whaleisle` `TestSettingsEmbeddedPagesStayInModalAcrossWidths` | clean HEAD 也失败（已用 stash 复核） |
| `cmd/whaleisle` `TestSettingsMarketUsesModalShellAndReturnsToGeneral` | 同上 |

## 2. 实机页面

已进入并截图 17 个页面：

1. 首屏/新会话
2. 插件页
3. 自动化任务
4. 鲸鱼娘主页
5. 设置·通用
6. 设置·外观
7. 设置·界面
8. 设置·模型
9. 设置·内置插件
10. 设置·技能
11. 设置·MCP
12. 设置·远程
13. 设置·Agent 预设
14. 设置·用量统计
15. 设置·鲸鱼娘
16. 设置·关于
17. 插件安装弹窗

截图：`.devdata/audit-*.png`

## 3. 弹窗覆盖（分组）

| 组 | 数量 | 执行方式 | 状态 |
| --- | --- | --- | --- |
| 权限/归档/会话反馈/工作流成员 | 4 | 权限、归档实机；其余包测试 | ✅ |
| 设置类：确认/默认权限/快捷键/模式指南/效果/鲸鱼娘记忆/引导/主模态 | 12 | 主模态与设置页实机；快捷键/效果/记忆/引导包测试 | ✅ |
| 插件/市场：确认/安装/索引源/市场选择/市场确认/市场详情 | 6 | 安装弹窗实机；其余包测试 | ✅ |
| Git：提交/创建分支/Review | 3 | diffsurface 包测试 | ✅ |
| 项目/目录/工作区：重命名/归档/页面弹窗/撤销要求/新建文件夹/选择器/改名/移除 | 8 | projects/dirpick/workspacepicker 包测试 | ✅ |
| Composer/聊天：权限/语音/附件/图片画廊/可视回复/引用错误/反馈 | 8 | composer/chat 包测试 | ✅ |
| 终端/排期/价格/通用确认/启动日志/助理设置 | 7 | 包测试；终端、启动页实机 | ✅ |
| 合计 | 48 | 实机 + 1325 项测试 | ✅ |

说明：✅ 表示有包级测试或实机证据，不等于 48 个弹窗都在真实窗口里逐个点过；实机逐个点击仍受窗口自动最小化限制。

## 4. 实机发现并修复

| ID | 缺陷 | 修复 | 回归 |
| --- | --- | --- | --- |
| F-A | 设置·模型页显示 `settings document opened` | `settingsActionText` 映射内部动作；14 个设置页统一使用 | `TestSettingsActionTextHidesProtocolNames` |
| F-B | 设置·鲸鱼娘页显示 WebView2 原始错误 | 默认标签隐藏；形象标签显示友好文案，原始错误放 Tooltip | `go test ./internal/ui/settings` |

提交：`dcf83b5d`

## 5. 仍未执行

- 真实模型对话（无 API Key）
- 插件真实安装、市场安装
- Git 提交/推送/切分支
- 排期创建/触发
- 远程配对
- 工作区文件删除/写入
- 48 个弹窗在真实窗口里的逐个鼠标点击（受窗口自动最小化 + 模态丢失影响）

## 6. 判定

- **自动化全跑**：完成，1325 项 UI 测试全过。
- **页面实机**：17 个已跑。
- **弹窗**：48 个有测试/实机覆盖，但不是 48 次真实点击。
- **真实链路**：6 类环境阻塞。
- **确定缺陷**：2 个，已修复。

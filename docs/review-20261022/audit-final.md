# 全量审计阶段报告

日期：2026-10-22  
范围：全部弹窗、设置页与主页面、核心可点击入口。

## 1. 工作量

- `ui.DialogBase`：47 处 / 40 个文件
- 已带玻璃或共享表面：40 处
- 设置子页：12 个，全部进入并统计文本节点
- 主页面：首屏、插件、自动化任务、鲸鱼娘
- 核心交互：侧栏折叠、搜索、工作区选择、Git 菜单、模型选择、终端、右侧面板、发送按钮状态

## 2. 修复的真实缺陷

| 缺陷 | 影响 | 修复 |
| --- | --- | --- |
| 权限确认复选框每帧重置 | 永远无法勾选，主按钮永远禁用 | `ui.Local` 跨帧状态 + 回归测试 |
| 排期删除弹窗按钮点不到 | 确认删除无反应 | `DialogDecor` 静态材质，不改布局、不加淡入延迟 |
| 大量弹窗 12–16px 平面白卡 | 与 28px 玻璃模态不一致 | `DialogSurface` / `DialogDecor` 分两档迁移 |
| 品牌残留 MyGo鲸屿 | 启动器/引导/登录/桌宠/日志不一致 | 统一 Whale Isle |
| 权限弹窗样式旧 | 直角、无阴影、无遮罩淡入 | 28px 玻璃 + 发丝边 + 阴影 |
| 媒体查看器无圆角/裁剪 | 全屏直角面板 | 20px 圆角、裁剪、遮罩 |
| 重复写句柄错误 | 会话打不开 | 清理孤儿 harness；非 UI 缺陷 |

## 3. 覆盖矩阵

完整逐项状态见 [coverage-matrix.md](coverage-matrix.md)。

## 4. 验证

通过：

```sh
go test ./internal/ui/... ./internal/store/... ./internal/desktop/whale ./cmd/whaleisle
cd third_party/mygo && go test ./ui -run 'TestMenu|TestContextMenu'
```

实机证据：

- `evidence/permission-dialog-unchecked.png`
- `evidence/permission-dialog-checked.png`
- `evidence/terminal-workspace.png`
- `evidence/menu-overlay.png`
- `evidence/menu-overlay-keyboard.png`

全仓库 `go test ./...` 仍有非本次改动失败：

- `whaleisle/tmp/gofmtcheck` setup failed
- `whaleisle/tmp` build failed
- `whaleisle/tools/rich-history-check` build failed
- `internal/desktop/importer` NSIS 静默安装 smoke 失败

## 5. 未完成 / 无法在当前环境验证

| 项 | 原因 | 下一步 |
| --- | --- | --- |
| 真实键盘输入后发送启用 | Composer 自定义编辑器只读 ValuePattern；SendInput/keybd_event/PostMessage 均不落字 | 真实键盘手工验证，或接入文本输入测试桥 |
| 模型目录为空 | 未设置 DEEPSEEK_API_KEY / OPENAI_API_KEY，无端点目录 | 配置凭据后走“供应商密钥与模型探测” |
| Git 提交/推送实际执行 | 会改动仓库，未在审计中执行 | 用临时副本仓库做端到端 |
| 插件启用/安装、市场安装 | 需要网络与可写环境 | 单独隔离 profile 执行 |
| 终端首次无工作区 | 设计如此：无工作区根目录时 `Panel == nil` | 选择工作区后已验证可用 |
| widgets 确认框玻璃 | chrome ↔ widgets 包循环 | 拆出底层材质包后再迁移 |

## 6. 结论

弹窗样式与可点击性方面的主要缺陷已收敛；权限确认与排期删除两个“看起来能点、实际点不动”的问题都有回归测试。剩余项集中在**需要凭据、网络或真实键盘的环境依赖**，以及一个包循环导致的玻璃层缺失，不属于继续扫样式能解决的范围。

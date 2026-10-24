# Round179 · cmd 内联双语文案迁移（G52 第一批）

把 cmd/whaleisle 里高频可见的内联 `resumeText("中文","English")` 文案迁入 `internal/ui/i18n` 目录：托盘/应用菜单、更新提示、账户/奖励/额度通知、恢复对话框。12 个文件共 **63 处调用**、**55 个新键**；剩余 187 处按文件列在 §5，留给后续批次。

## 1. 清单（迁移前）

`resumeText` 定义在 [recovery.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/recovery.go>)：`i18n.Locale() == "en"` 取英文，否则中文。全仓 cmd/whaleisle 共 254 处（含 1 处测试辅助），按表面分组：

| 表面 | 代表文件 | 处数 |
|---|---|---|
| 托盘与窗口 | desktop_tray.go(5)、desktop_tray_menu.go(1)、application_menu.go(2，其中 1 处是缓存签名不是 UI 文案) | 8 |
| 更新 | update_check_lifecycle.go(15)、update_progress.go(10)、update_attention.go(5)、launcher_update_check.go(1) | 31 |
| 账户/奖励/额度 | account_bonus_notice.go(7)、account_quota_notice.go(5)、account_status_notices.go(3)、account_return.go(3) | 18 |
| 恢复/崩溃对话框 | recovery.go(5) | 5 |
| 项目/任务保护/导入/组件/发布 | task_protection.go(24)、native_releases.go(23)、component_changes.go(19)、host_bundle.go(15)、main.go(12)、import_flow.go(11)、whalebridge_window.go(8) 等 | 191 |
| 其它单点 | welcome/launcher/main_navigation/confirm_ask 等 | 余数 |

## 2. 迁移方式

- 键名沿用 `catalog_cmd.go` 既有约定：`cmd.<英文小写 slug>`，常量 `KeyCmd<Camel>`，`Register` 在同一文件里同时登记 zh/en。55 个新键全部落在 [catalog_cmd.go](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_cmd.go>)。
- 调用点机械替换：`resumeText("中文", "English")` → `i18n.T(i18n.KeyCmdXxx)`；外层 `fmt.Sprintf(...)`、字符串拼接、`+ err.Error()` 等格式逻辑原样保留，所以带 `%s/%d` 的文案（如更新确认 `%s 已可用…`）仍由调用点传参。
- 已存在同名英文 slug 的键会复用（本轮 55 个新键，无覆盖）；`application_menu.go` 的 `resumeText("zh","en")` 是菜单缓存签名，不做迁移。
- 12 个文件补上 `whaleisle/internal/ui/i18n` import（`desktop_tray_menu.go` 原为单行 import，已展开）。

## 3. 本轮迁移明细

| 文件 | 迁移前 | 迁移后 | 说明 |
|---|---|---|---|
| desktop_tray.go | 5 | 0 | 托盘不可用对话框、最小化到托盘通知 |
| desktop_tray_menu.go | 1 | 0 | 桌面宠物菜单项 |
| application_menu.go | 2 | 1 | 开发者工具；保留缓存签名 |
| update_check_lifecycle.go | 15 | 0 | 检查中/失败/无更新源/最新/发现新版本/好/关闭/准备下载/下载中/等待确认/技术细节 |
| update_progress.go | 10 | 0 | 关闭(Dismiss)/新版本/下载线路/安装包大小/增量路径/版本/平台/包含运行时 |
| update_attention.go | 5 | 0 | 更新待确认/可用/查看版本/稍后 |
| launcher_update_check.go | 1 | 0 | 启动器更新检查 |
| account_bonus_notice.go | 7 | 0 | 确认中/失败重试/无法确认/知道了/重试/关闭/奖励通知 |
| account_quota_notice.go | 5 | 0 | 额度耗尽 toast、无余额标题与说明、取消/前往充值 |
| account_status_notices.go | 3 | 0 | 登录失效/模型需登录/前往登录 |
| account_return.go | 3 | 0 | 授权页/充值页/取消登录失败 |
| recovery.go | 5 | 0 | 恢复状态保存失败对话框（标题/正文/保留窗口/不保存退出） |

合计：63 处调用 → 55 个键（部分同文案在不同文件复用同一键，如「关闭/Close」）。

## 4. 测试

新增 [cmd_i18n_slice_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/cmd_i18n_slice_test.go>)：

- `TestCmdI18nSliceCatalog`：55 个键在 zh/en 两个目录都存在、非空、不与其他语言相同，且两边的 `%s/%d/%v/%q` 动词数量一致（防止翻译丢参数）。
- `TestCmdI18nSliceCallSites`：把 12 个源文件读回来，逐键检查目录里的中文不再以 `resumeText("…"` 形式残留，确保调用点真的改成了 `i18n.T`。
- 既有 `internal/ui/i18n` 的注册测试仍通过（无重复键、无覆盖）。

## 5. 剩余清单（未迁移，按文件）

共 **187 处**（非测试），主要表面与后续批次建议：

| 文件 | 处数 | 说明 |
|---|---|---|
| task_protection.go | 24 | 任务保护对话框/提示，交互密集，需单独一批 |
| native_releases.go | 23 | 版本/发布列表与错误 |
| component_changes.go | 19 | 组件变更说明 |
| host_bundle.go | 15 | Host 选择/安装提示 |
| main.go | 12 | 启动/保存/导航杂项 |
| import_flow.go | 11 | 导入流程 |
| whalebridge_window.go | 8 | 桥接窗 |
| application_uninstall.go | 7 | 卸载确认 |
| project_live_refresh.go | 7 | 项目刷新 |
| generic_components.go | 7 | 通用组件 |
| startup_recovery.go | 6 | 启动恢复 |
| startup_diagnostics.go | 5 | 启动诊断 |
| project_lifecycle.go | 5 | 项目生命周期 |
| project_recovery.go / project_requests.go / project_detail.go / project_assets.go | 各 4 | 项目面 |
| notify.go | 3 | 通知 |
| 其余 ≤2 的文件（schedule_delete_notice、resume_checkpoint、bridge_alerts、import_recovery、project_team、project_flow、project_request_revert、welcome_window、launcher_window、main_navigation、confirm_ask） | 各 1–2 | 单点文案 |

**需要原生证据的文案**：本轮迁移的托盘/更新/账户/恢复文案都属于 MyGo 桌面壳自有文案（harness 字典里没有对应条目），因此按原样迁移、不改措辞；剩余清单同样以 MyGo 自有流程文案为主，暂无需要先取原生证据才能迁移的项。若后续要统一托盘/菜单措辞，需对照安装版 `desktop_tray` 实际文案另开证据轮。

## 6. 门禁

- `go test ./internal/ui/i18n ./cmd/whaleisle -count=1` → i18n 0.777s、cmd 23.416s，全部 ok。
- `go vet ./cmd/whaleisle ./internal/ui/i18n` → 退出 0；`go build ./cmd/whaleisle` → 退出 0（临时产物未进仓库）。
- 未提交、未构建 passN；未改原生应用、settings/chat/remote 表面、runtime/vendor、third_party。

## 7. 第二批（task-63）：任务保护/发布/组件/宿主/主流程

- 迁移 5 个文件：`task_protection.go`(24)、`native_releases.go`(23)、`component_changes.go`(19)、`host_bundle.go`(15)、`main.go`(12)，实际 **101 处调用**（同一行多个调用按次计），新增 **89 个 `cmd.*` 键**；`catalog_cmd.go` 的 cmd 键总数 129 → **218**。
- 全部为字面 `resumeText("中文","English")`；`fmt.Sprintf(resumeText(...), …)`、`+ err.Error()`、拼接与 `%s/%d` 参数保持原样。这 5 个文件现在 **0 处** `resumeText(` 残留。
- 非 UI 文案的跳过规则（脚本内置，本批命中 0 处）：`resumeText("zh","en")` 缓存签名、以及中文侧不含 CJK 的标识符/协议串（如 `GET`、`application/json`）都留在原处并计数；`application_menu.go` 的菜单缓存签名按 round179 规则继续保留。
- 测试扩展：[cmd_i18n_slice_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/cmd_i18n_slice_test.go>) 的键清单改为直接取目录里的全部 218 个 `cmd.*` 键（不再两批手工维护），文件清单扩展到 17 个；`TestCmdI18nSliceCatalog`（双语存在/非空/zh≠en/动词数一致）与 `TestCmdI18nSliceCallSites`（17 个源文件里不得残留被迁移的中文）均通过。
- 剩余（当前扫描，非测试）**99 处**：

| 文件 | 处数 |
|---|---|
| import_flow.go | 11 |
| whalebridge_window.go | 8 |
| generic_components.go / project_live_refresh.go / application_uninstall.go | 各 7 |
| startup_recovery.go | 6 |
| project_lifecycle.go / startup_diagnostics.go | 各 5 |
| application_menu.go | 4 |
| project_detail.go / project_requests.go / project_recovery.go / project_assets.go | 各 4 |
| notify.go | 3 |
| 其余 ≤2 的文件（schedule_delete_notice、resume_checkpoint、bridge_alerts、import_recovery、project_team、project_flow、project_request_revert、welcome_window、launcher_window、main_navigation、confirm_ask 等） | 各 1–2 |

  注：`187 − 101 = 86`，当前扫描为 99，差额来自本批之外的文件（`application_menu.go` 由 1 增至 4 等），属并行改动的增量，列在这里备下一批核对。
- 门禁：`go test ./internal/ui/i18n ./cmd/whaleisle -count=1` → i18n 0.610s、cmd 27.451s，全部 ok；`go vet ./cmd/whaleisle ./internal/ui/i18n` → 退出 0；`go build ./cmd/whaleisle` → 退出 0。未提交、未构建 passN，未改 settings/chat/remote/frame、runtime/vendor、third_party。

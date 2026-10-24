# Round214 · 通知身份（AUMID）与原生机制对照

结论：原生通过 `app.setAppUserModelId('ai.deepseek.harness.gui')` + Windows 自动注册的 Start-menu 快捷方式让 toast 显示“Whale Isle”，并迁移旧的 raw-Electron `Electron.lnk`；MyGo 之前只有托盘/`mygo.Notification` 路径、没有 AUMID 注册。本轮实现了 Start-menu `.lnk` 注册（COM `IShellLink` + `IPropertyStore`，纯 syscall/unsafe，无 cgo、无新依赖），并把 AUMID 真正写进链接（测试按字节校验）。toast 本体的呈现仍走既有 `mygo.Notification`/托盘路径。

## 1. 原生机制（含出处）

| 项 | 原生 | 出处 |
|---|---|---|
| AUMID 字符串 | `ai.deepseek.harness.gui`（launcher 为 `ai.deepseek.harness.launcher`） | `src/main/index.js:1005 app.setAppUserModelId(...)`；`src/main/window-app-details.js:7`；`src/main/package-contract.test.js:66` |
| 通知身份注册 | Electron 在 Windows 上为当前可执行文件注册 Start-menu 快捷方式并携带 AUMID；dev/raw 形态是 `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Electron.lnk` | `src/main/legacy-notification-shortcut.js:48-56`（只匹配 `appUserModelId === DESKTOP_APP_ID`、`args==''`、`iconIndex==0`、target 基名为 `electron.exe` 的链接） |
| 旧链接清理 | 迁移时把匹配到的链接**移动**到 `userDataDir/legacy-system-shortcuts/<uuid>/Electron.lnk.backup`，随后 `SHChangeNotify(SHCNE_DELETE, SHCNF_PATHW_FLUSH, old)`（500ms 预算，失败不影响启动）；其他快捷方式/固定项/自定义链接一律保留 | 同文件 `:27-46`、`:70-133` |
| 通知策略 | `platform !== 'win32' || defaultApp !== true` 时才允许系统通知；`isSupported()` 仅在不会注册 raw Electron 链接时调用 | `src/main/system-notifications.js:10-17` |
| 点击路由/内容 | Electron Notification 自带（本仓库未自定义 actions；点击聚焦窗口由主进程处理） | `system-notifications.js` 仅策略；`legacy-notification-shortcut.js` 仅身份 |
| 回退 | 非 Windows / dev / 不支持时退回应用内提示与托盘；旧链接迁移失败时保留源文件 | 同上 |

## 2. MyGo 实现

新增包 `internal/desktop/notifications`：

- [notifications.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/notifications/notifications.go>)：`AppUserModelID = "ai.deepseek.harness.gui"`（与原生一致）、`ShortcutName = "Whale Isle.lnk"`、`StartMenuProgramsDir()`（`%APPDATA%\Microsoft\Windows\Start Menu\Programs`，无 APPDATA 返回空）、`Shortcut{ProgramsDir, Target}.Ensure()`，以及测试用的 shell seam `ensureWith(create func(lnk,target,appID string) error)`。
- [shortcut_windows.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/notifications/shortcut_windows.go>)：`CoInitializeEx(COINIT_APARTMENTTHREADED)` → `CoCreateInstance(CLSID_ShellLink, IID_IShellLinkW)` → `SetPath`/`SetIconLocation` → `IPersistFile::Save` → `IPropertyStore::SetValue(PKEY_AppUserModel_ID, VT_LPWSTR)` → `Commit` → **再 Save 一次**（属性存储只改内存副本，二次持久化后 AUMID 才进入 `.lnk`；这是实测发现的关键步骤）。全部经 `syscall.NewLazyDLL`/`SyscallN` 调 vtable，无 cgo、无第三方包。
- [shortcut_other.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/notifications/shortcut_other.go>)：非 Windows 的 `createShortcut` 为 no-op（`Ensure` 在无 Programs 目录/无 target 时同样 no-op）。
- 接线：[desktop_tray.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/desktop_tray.go>) 的 `setupDesktopTray()` 在创建托盘前调用 `os.Executable()` + `Shortcut{...}.Ensure()`，失败只 `log.Println("aumid:", err)`，不影响托盘/窗口；托盘通知与 `mygo.Notification` 路径保持原样作为回退。

## 3. 测试

- `TestShortcutPath`：空 ProgramsDir 不产生路径、`Ensure` 为 no-op 且不落文件；路径固定为 `<Programs>\Whale Isle.lnk`；空 target 也是 no-op。
- `TestShortcutStartMenuDir`：`APPDATA` 存在时得到 `…\Microsoft\Windows\Start Menu\Programs`，缺失时返回空。
- `TestEnsureUsesShellSeam`（fake shell）：一次注册，参数为链接路径、可执行文件、`ai.deepseek.harness.gui`——命令接线可用 fake 覆盖。
- `TestCreateShortcutTempProfile`（Windows）：临时目录写入真实 `.lnk`，第二次 `Ensure` 幂等、文件非空，并按字节校验 UTF-16 的 AUMID 确实写进了链接（二次 Save 前后均验证）。
- 非 Windows：`GOOS=linux`/`GOOS=darwin` 下 `go build ./internal/desktop/notifications` 均通过（no-op 实现）。
- 通知本体的可视化/点击无法在无头环境验证：toast 由系统 Shell 呈现，测试机为无交互会话；记录为残余（见 §5）。
- 门禁：`go test ./internal/desktop/notifications ./internal/desktop/pet ./internal/desktop/botnotify` 全绿（0.738s / 2.321s / 0.420s），`go test ./cmd/whaleisle` 23.207s ok，`go vet ./internal/desktop/notifications ./cmd/whaleisle` 退出 0。`go test ./internal/desktop/...` 中 `internal/desktop/workspacefs` 失败与本次无关（`search_match_test.go:24/34` 返回空、`parity_test.go:70 search truncated: 0`，属并行改动/搜索后端，未触碰该包）。

未提交、未构建 passN；原生应用只读；未改 runtime/vendor/deepseek-harness 或 third_party。

## 4. 与原生对照

| 面 | 原生 | MyGo（本轮后） |
|---|---|---|
| 身份 | `setAppUserModelId` + Electron 注册 Start-menu 链接 | 同样写 Start-menu 链接，AUMID 相同、目标为当前 exe |
| 旧 raw 链接 | 迁走 `Electron.lnk`（带备份 + `SHChangeNotify`） | 不迁移：MyGo 不会生成 raw Electron 链接；若机器上残留旧 `Electron.lnk`，本轮不处理（残留项） |
| 通知呈现 | Chromium/Electron toast（系统通知中心、点击回窗口） | 仍是 `mygo.Notification`/托盘弹泡；AUMID 让系统把来源显示为产品名，但样式/动作集与原生 toast 不同 |
| 回退 | 非 win32 或 dev 时不用系统通知 | 非 Windows `Ensure` no-op；通知失败回退原路径 |

## 5. 残余

1. **toast 本体**：WinRT `ToastNotificationManager` 需要额外绑定（无 cgo/无依赖约束下不引入），MyGo 继续用 `mygo.Notification` 的呈现；AUMID 已注册，系统弹出的来源名与原生一致，但内容模板/动作按钮/通知中心历史行为仍不同。
2. **旧的 `Electron.lnk` 迁移**：原生会把匹配的旧链接移入备份并通知 Shell；MyGo 未实现该清理（风险很低：MyGo 不产生该链接；若用户机器上曾有 native dev 版，链接会残留）。
3. **点击路由**：原生 toast 点击聚焦窗口；MyGo 依赖 `mygo.Notification` 的点击回调，二者语义需在真实交互会话里核对（无头环境不可测）。
4. **跨平台整体构建**：`GOOS=linux/darwin` 下本包通过，但 `go build ./internal/desktop/...` 会被并行改动的 `internal/desktop/components/alerts_process_other.go` 编译错误挡住（非本片文件）。

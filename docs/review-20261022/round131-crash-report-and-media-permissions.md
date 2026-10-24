# Round131 · 崩溃报告（G41）与媒体权限（G42）

本轮补上 gap matrix 最后两项 shell 级缺口：致命失败的 `crash-*.log` 诊断，以及把 Harness 页面权限收窄到「仅 loopback 麦克风 + 通知跟随系统开关」。二者都只做安装版真实存在的语义，无法一一对应的部分显式记录为残余。

## 1. 原生对照

-G41：[crash-report.js:13-23](<C:/AI/WhaleIsle-MyGo/reference/electron/src/main/crash-report.js>) 定义文件名 `crash-YYYY-MM-DDTHH-MM-SSZ-<source>.log`（`:18-19` 的正则，source ∈ host|web-boot|renderer|main）、保留 10 份（`:19`）、错误段 256K 字符（`:20`）、渲染器控制台尾部 64K 字节（`:21`）。`crashReportFileName`（`:81-83`）把 `toISOString()` 的 `:` 与 `.` 换成 `-`；`writeFile` 用 `flag:'wx'`（`:110`），失败只记日志、返回 `undefined`（`:106-116`），绝不遮挡调用方真正要报的错。`pruneCrashReports`（`:119-135`）按文件名序删旧，只删匹配正则的名字。目录是 `app.getPath('logs')/crash`（paths.js `crashLogDir`）。恢复对话框在窗口层用 `诊断已写入 <path>` 指路（window.js:76-79）。
-G42：[media-permissions.js:10-18](<C:/AI/WhaleIsle-MyGo/reference/electron/src/main/media-permissions.js>) 的 `isHarnessOrigin` 只认 http(s) + `127.0.0.1`/`localhost`；`:32-47` 的 request handler：`notifications` 在 `systemNotificationsAllowed()` 为假时拒；`media` 仅当「主 Harness webContents + 主帧 + loopback origin + 仅 audio」才许；其它权限交回引擎默认（允许）。check handler（`:33-40`）对 `media` 同样收窄。

## 2. 实现

G41 —— 新增 [crash_report.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/crash_report.go>)：

- `CrashReportFileName` / `crashReportName` / `CrashReportsRetained=10` / `CrashRendererConsoleMaxBytes=64KiB` / `CrashErrorSectionMaxChars=256K` 逐条对齐上表；未知 source 落回 `main`，因此文件名永远匹配保留正则，也不能借 source 逃出目录。
- `RenderCrashReport` 保持原生段序：header（time/source/phase/app/platform/electron/node/locale/shell pid）→ `--- error ---` →（可选）`--- host diagnostic ... ---` → `--- renderer console ... ---`。Electron 为 Go 原生 shell，故 `electron`/`node` 如实写 `none`，另加 `go:` 行，不伪造版本号。
- `boundedCrashError` 截断后附 `(error section cut at 262144 characters)`；`crashConsoleTail` 是 `RendererConsoleTail` 的移植（超长单行只留 UTF-8 安全后缀，跨字节边界不产生替换字符）。
- `WriteCrashReport` 先 `MkdirAll(0700)`、再以 `O_EXCL`（原生 `wx`）写 `0600`，成功才 prune；任何失败都只 `log.Println` 并返回 `""`。`PruneCrashReports` 只删匹配名、按名字序保留最新 10 份，其它文件不动，目录缺失（ENOENT）不算错。
- 目录 `CrashLogDir()` = `<userData>/logs/crash`，对应原生 `logs/crash`。

接线（只接 MyGo 真实存在的致命面）：

| 原生 source | MyGo 现状 | 接入点 |
|---|---|---|
| `host` | 真实：本地 Harness 服务起不来/中断退出 | [main.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/main.go>) `refreshBoot` 的 `harness.Failed` 分支，按 generation（无 generation 时按错误文本）去重，一个失败一份文件；host diagnostic 取 `sup.Logs()` |
| `main` | 真实：事件循环失败 | `mygo.App.Run()` 失败处先写报告再 `log.Fatal`，原错误照常抛出 |
| `web-boot` | 半真实：MyGo 没有 boot web 进程（boot 页是原生 `ui.Content`）；最近的真实面是面板/浏览 WebView2 host 创建失败 | 插件/浏览 host 创建失败处异步写一份，避免在 `s.mu` 内做磁盘 IO |
| `renderer` | 缺失：MyGo 无渲染进程、无 renderer console 桥 | 源码常量保留以兼容保留正则；报告里恒为 `(no error-level renderer console output was captured)` |

写盘后 `publishCrashReport` 把 `诊断已写入 <path>`（i18n `crash.report_written`，中英各一）追加到启动器错误文本（若已有错误）与 boot 日志，并把路径存到 `state.crashReportPath`；写不出报告时一条也不加，恢复面继续显示原始错误。

G42 —— 新增 [permissions.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/webview/permissions.go>)：

- `IsLoopbackHarnessOrigin`：http(s) + `127.0.0.1`/`localhost`，拒绝 `127.0.0.1.evil`、`user@host`、`file:`、裸 `host:port`。
- `PermissionPolicy{SessionHost, NotificationsAllowed}`：`notifications` 仅在 gate 为真时允许（gate 为 nil 即拒）；`microphone` 仅当 `SessionHost` 且 loopback origin；`camera`/`geolocation` 一律拒；其余权限保留引擎默认（允许）。请求内任一权限被拒则整单拒，避免「麦克风+摄像头」被整体批准。
- [host.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/webview/host.go>) 新增 `HostOptions.PermissionPolicy`：设置了才覆盖引擎默认（原先 loopback 可信页面默认拿到一切，这正是 G42 缺口）；`TransientProfile` 仍走 `DenyAllPermissions`，隔离能力窗口一个不给。
- 各 webview host 现在都有明确答复：会话/插件/浏览 host `SessionHost = sessionOrigin != ""` + 通知 gate（[main.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/main.go>)）；WhaleBridge 管理与文档预览不是 Harness 视图，`SessionHost = false`（麦克风/摄像头一律拒，通知跟随 gate）；账号平台窗口本就是 transient，保持全拒。

## 3. 回归

- [crash_report_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/crash_report_test.go>)：文件名与正则（含毫秒、UTC 归一、未知 source 落回 main）、各段齐全与段序、错误段 256K 截断与提示、控制台尾部 64K/UTF-8 后缀、保留 10 份且不动无关文件、`O_EXCL` 不覆盖、目录不可写时返回空且不 panic。
- [permissions_test.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/webview/permissions_test.go>)：权限表（loopback 麦克风 yes；remote/LAN/伪装 origin 麦克风 no；非会话 host 麦克风 no；camera/geolocation no；通知开关与无 gate no；未知权限默认 yes；整单拒绝）、origin 判定、隔离档位全拒。

命令（`. .\scripts\env.ps1`）：

- `go test ./cmd/whaleisle ./internal/desktop/webview ./internal/ui/i18n -count=1` → 全 ok（具体秒数见交付说明）。
- `go vet ./cmd/whaleisle ./internal/desktop/webview ./internal/ui/i18n` → 退出 0。

## 4. 边界与未完成

- **权限残余（明确记录）**：`mygo.PermissionRequest` 只有 `Permissions` 与 `Origin`，没有主 Harness webContents、没有 `isMainFrame`、没有 `mediaType`。因此原生「仅主 Harness 主帧、仅 audio」无法逐字复刻：本实现只能保证「loopback + 会话 host」，同源插件页同样可能拿到麦克风；`mediaTypes = audio+video` 的请求因摄像头被拒而整单拒。设备级仍由 WebView2/OS 决定，macOS 需 `NSMicrophoneUsageDescription`（本仓未打包该 plist 项，属既有平台差异）。
- **geolocation 有意从严**：原生把非 media 权限交回 Electron 默认（允许），MyGo 直接拒——WebView2 侧未配置系统定位服务，shell 也从未提供该能力；这是记录在案的收紧而非对齐。
- **boot 页面没有专门的「诊断文件」行**：`internal/ui/boot` 不在本轮写入范围，路径目前经启动器错误文本与 boot 日志（含导出）呈现；若后续要在 boot 页单列一行，需要改 boot 包。
- **renderer 源恒缺失**：没有渲染进程，控制台尾部只会是占位文本。
- 未提交、未构建 exe；本轮的 G41/G42 行状态更新留在 gap matrix 由 Lead 一并提交。

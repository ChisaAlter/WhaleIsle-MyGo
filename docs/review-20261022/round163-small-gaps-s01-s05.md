# Round163 · 2026-10-25 矩阵 S01–S05：小口子清理

范围：`docs/parity/gap-matrix-20261025.md` 的 S01–S05 五行。每行先读原生引用，再找 MyGo 归属与最小改动；未能本轮完成的行写清 owner、阻塞点和下一步，未做假实现。

## S01 harness 页面文件下载 —— **已修**

原生：`index.js:1359` 给每个 webContents 挂 `will-download`；`download-path.js:1-28` 把建议名收敛成「单层文件名 + 控制字符/`<>:"/\|?*` 换 `_` + 去尾部点与空格 + Windows 设备名加 `_` 前缀 + 180 字符上限」，默认名 `download`。
MyGo 之前：`internal/desktop/webview/host.go:363-372` 只在 `TransientProfile` 的隔离页 `PreventDefault()`，真实页面（含 harness UI）沿用引擎默认命名（`filepath.Base`，保留 Windows 禁止字符、不处理 DOS 设备名、无长度上限）。

改动：
- 新增 [`internal/desktop/webview/downloads.go`](<C:/AI/WhaleIsle-MyGo/internal/desktop/webview/downloads.go>)：`SanitizeDownloadName`（download-path.js 的 Go 移植，含 180 rune 上限、设备名前缀、尾点/空格裁剪）、`UniqueDownloadPath`（与引擎一致的重名编号 `name (2).ext`）、`saveDownload`（取 `App.Path(PathDownloads)`，目录不可建时 `PreventDefault`）。
- `host.go:363-380`：非 `TransientProfile` 一律挂 `OnWillDownload(saveDownload)`，`PermissionPolicy` 仍在同一分支设置（两者可共存）。
- 测试 [`downloads_test.go`](<C:/AI/WhaleIsle-MyGo/internal/desktop/webview/downloads_test.go>)：16 组命名表（含 `con`/`CON.txt`/`lpt9.dat`/`trailing...`/路径穿越/emoji）+ 200 个「鲸」的 180 rune 裁剪 + 重名编号三级。

前后：建议名 `C:\tmp\a<b>.txt` → 改前引擎可能留下非法名或被系统改名；改后 `a_b_.txt`；`con` → `_con`。`go test ./internal/desktop/webview -run Download` 通过。

## S02 安装前磁盘空间预检 —— **未修（open）**

原生：`install-space.js:6-11` `SPACE_RESERVE = 256 MiB`、`formatBytes` 输出 `X.XX GiB`；`checkInstallSpace(directory, bytes, {statfs, reserve})` 从目的地向上找第一个可 `statfs` 的目录（目录尚未创建也能估），文件系统不支持时报给调用方而不是当作 0，并声明这只是预检、写失败仍要处理 ENOSPC。
MyGo 现状：安装走 `internal/desktop/market/install.go` → `installctrl`（外部包），没有 statfs 预检；`internal/paths` 也没有卷空间查询。
阻塞：需要平台实现（unix `syscall.Statfs`、Windows `GetDiskFreeSpaceEx`，按 AGENTS 风格拆 `space_unix.go`/`space_windows.go`/`space_fake.go`）并接到 install 入口，属于跨平台新文件 + 安装链路改动，超出本轮 50 分钟盒。下一步：`market.CheckInstallSpace(dir, bytes)`（256 MiB 保留、`X.XX GiB` 文案）+ 安装开始前调用 + 假 statfs 单测。

## S03 DevTools 窗口级快捷键 + 打包门控 —— **未修（open）**

原生：`devtools-shortcut.js:1-33`——不给 OS 注册全局快捷键，而是给每个 webContents 加 `before-input-event`：`Ctrl+Shift+I`（mac/linux 上 `Cmd+Alt+I`），仅 `keyDown`；`devToolsShortcutAllowed({isPackaged, openDevTools})` = `!isPackaged || openDevTools === true`。
MyGo 现状：只有菜单项 `RoleToggleDevTools`（`cmd/whaleisle/application_menu.go:251`）与设置里的 `OpenDevTools` 配置；没有窗口级快捷键，也没有打包态门控。
阻塞：mygo 的 `Page` 没有暴露键盘输入钩子（`before-input-event` 等价物），改动需要 `third_party/mygo`，本任务明确禁止。下一步选择：要么在 mygo 增加 `Page.OnBeforeInput`，要么用窗口级 accelerator（`ui` 原生窗口已有快捷键系统）实现 Ctrl+Shift+I 并在打包态按 `openDevTools` 门控——需 Lead 决定是否动 mygo。

## S04 浏览器预览缩放档位 —— **未修（open，已备好规格）**

原生：`preview-zoom.js:4-6` 固定 17 档 `[0.25, 0.33, 0.5, 0.67, 0.75, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2, 2.5, 3, 4, 5]`，`DEFAULT_ZOOM_FACTOR = 1`，`ZOOM_EPSILON = 0.001`；`findZoomStep` 取「等于或略低于当前值」的档位索引，`nextZoomLevel(current, 'in'|'out')` 在档位表内前后移动并夹住两端。
MyGo 现状：`internal/ui/browserpreview/` 没有 zoom 动作（只有 PDF 预览有缩放）；引擎有 `Page.SetZoomFactor`（`third_party/mygo/window.go:1138`），所以不需要第三方改动。
未做原因：本轮剩余时间不足以同时完成「档位算法 + 菜单/Ctrl± 接线 + 截图验证」；把表与语义完整记录，下一轮直接落 `browserpreview/zoom.go`（`ZoomLevels`/`FindZoomStep`/`NextZoomLevel` + 单测）再接菜单/快捷键。

## S05 插件崩溃取证分类 —— **未修（open）**

原生：`plugin-forensics.js:12-16` 四条正则——`GENERIC_OOM`（heap out of memory / js heap / allocation failed / oom）、`GENERIC_PORT_EXCLUDED`（`listen EACCES`，端口被策略排除，与 EADDRINUSE 不同）、`GENERIC_PORT`（eaddrinuse / address already in use）、`GENERIC_NODE`（node not recognized / cannot find node / enoent…node），另有 loader/依赖等 `EVIDENCE_PATTERNS`。
MyGo 现状：只有 launcher 的 `pluginRecovery` 状态（部分），没有这套分类文案。
未做原因：需要对照 MyGo 恢复流程逐条比对并补注入式单测（矩阵评估 M 规模），本轮 50 分钟已用于 S01 的实现与验证。

## 门禁

- `go vet ./internal/desktop/webview` → 0。
- `go build ./internal/... ./cmd/whaleisle` → 0。
- `go test ./internal/desktop/webview -run Download` → ok（新增用例全绿）。

## 结论

| 行 | 状态 | 证据 |
| --- | --- | --- |
| S01 下载命名/落盘 | **fixed** | downloads.go + host.go:363-380 + downloads_test.go |
| S02 安装空间预检 | open | 原生规格已记录；需平台 statfs + 安装链路接线 |
| S03 DevTools 窗口快捷键 | open | 需 mygo `Page.OnBeforeInput` 或原生窗口 accelerator，待决策 |
| S04 预览缩放档位 | open | 17 档表已抄录；`Page.SetZoomFactor` 可用，接线未做 |
| S05 插件取证分类 | open | 四条正则已抄录；需与恢复流程逐条比对 + 注入测试 |

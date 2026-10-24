# Round171 · S03：窗口级 DevTools 快捷键（mygo `Page.OnBeforeInput` + 门控）

原生引用：`reference/electron/src/main/devtools-shortcut.js:1-33`——不注册 OS 全局快捷键，而是给每个 webContents 挂 `before-input-event`：`Ctrl+Shift+I`（macOS `Cmd+Alt+I`），仅 `keyDown`，其余修饰键必须为空；`devToolsShortcutAllowed = !isPackaged || openDevTools`。

## 1. mygo 新 API（additive，不改 `Window` 接口）

- `third_party/mygo/input.go`
  ```go
  type KeyInput struct { Key string; Control, Shift, Alt, Meta, Down bool }
  func (p *Page) OnBeforeInput(fn func(KeyInput) bool) bool
  ```
  返回「该平台能否在页面之前看到按键」；`fn` 返回 true 表示按键已被处理，页面不再看到它（等价于 Electron `preventDefault`）。Content（原生 UI）窗口没有 page，返回 false。
- `third_party/mygo/internal/platform/input.go`：`platform.KeyInput` + 可选能力接口
  ```go
  type keyInputWindow interface{ SetBeforeInput(fn func(KeyInput) bool) }
  func SetBeforeInput(w any, fn func(KeyInput) bool) bool
  ```
  用**可选接口**而不是给 5 个后端都加 `Window` 方法，和 `browserDevToolsBackend` 的既有做法一致，因此没有任何平台被破坏。

### 各平台实现

| 平台 | 实现 | 能力 |
| --- | --- | --- |
| windows | `internal/windows/input.go`：`SetBeforeInput` 存入 `window.beforeInput`；`webview.go` 的 `acceleratorKeyPressed` 里在**菜单快捷键之前**调用 `fireBeforeInput(vk, kind==0||2)`，keys 来自 `GetKeyState`（Ctrl/Shift/Alt/Win），返回 true 时 `accelPutHandled(1)` 让 WebView2 不再把按键交给页面 | **可用**（keyDown 才回调） |
| linux | `internal/linux/input.go`：保存回调；WebKitGTK 不暴露页面之前的按键事件 | 不可用（`OnBeforeInput` 返回 false） |
| darwin | `internal/darwin/input.go`：保存回调；WKWebView 无页面之前按键事件（需注入 JS） | 不可用（同上） |
| fake | `internal/fake/input.go`：保存 + `FireBeforeInput()` 供测试 | 可用（测试用） |

无 cgo；`cd third_party/mygo && go build ./...` 在 windows/linux/darwin 均 exit 0，`go vet ./...` 0。

## 2. 应用侧接线（`internal/desktop/webview`）

新增 [`devtools_shortcut.go`](<C:/AI/WhaleIsle-MyGo/internal/desktop/webview/devtools_shortcut.go>)：

- `DevToolsShortcutAllowed(packaged, openDevTools) = !packaged || openDevTools`（原生 `:16-19`）。
- `IsDevToolsToggle(KeyInput)`：`Down && Key=="i" && ((Ctrl&&Shift&&!Meta&&!Alt) || (Meta&&Alt&&!Ctrl&&!Shift))`（原生 `:22-32`）。
- `Host.installDevToolsShortcut(pg)`：对每个 webview host 的 page 安装 `OnBeforeInput`，命中时**只切换该页的 devtools**（`Host.devToolsOpen` 状态 + `OpenDevTools/CloseDevTools`），返回 true 吞掉按键；`host.go` 两处建 page 的地方（子窗口 285、主 host 442）都已接线。
- 门控来源：`SetDevToolsShortcutAllowed(func() bool)`，`cmd/whaleisle/main.go`（`config.Open` 之后）注入 `!mygo.App.IsPackaged() || cfgStore.Get().OpenDevTools`——与原生完全一致。
- 菜单项 `RoleToggleDevTools`（application_menu.go:251）未改动，两条路径并存。

## 3. 前后对照（S03）

| | 之前 | 现在 |
| --- | --- | --- |
| 窗口级 Ctrl+Shift+I / Cmd+Alt+I | 无（只有菜单项，快捷键完全缺失） | Windows 每个 webview 窗口独立切换自己的 devtools；打包态需 `openDevTools` 设置，开发态始终允许；菜单项照旧 |
| 全局副作用 | — | 没有注册 OS 全局热键（原生明确反对），只在页面收到按键前处理 |
| macOS/Linux | 无 | 仍无原生等价钩子：`OnBeforeInput` 在这两个平台返回 false，回调不会运行（差距如实保留） |

## 4. 测试

- `third_party/mygo/internal/platform/input_test.go`：实现 `SetBeforeInput` 的窗口返回 true 并收到回调与完整字段；未实现的类型返回 false。
- `third_party/mygo/input_test.go`：Content（原生 UI）窗口的 `Page.OnBeforeInput` 返回 false；nil 回调返回 false。
- `internal/desktop/webview/devtools_shortcut_test.go`：门控四组合（开发态开/关、打包态开/关）；按键匹配 9 组（ctrl+shift+i、cmd+alt+i、key up、缺 shift、缺 ctrl、多按 Alt、cmd+shift+i、其它键、空键）。
- 门禁：`go test ./internal/desktop/webview` ok 2.8 s；`go build ./internal/... ./cmd/whaleisle` 0；`go vet` 0；`third_party/mygo` 三平台 `go build ./...` 0 + `go vet ./...` 0。

## 5. 残留

1. macOS/Linux 没有页面之前的按键钩子：原生用 Electron 的 `before-input-event`（两端都支持），mygo 的 WKWebView/WebKitGTK 路径需要注入 JS 或新增平台事件源才能等价，本轮如实留空而非半实现——`OnBeforeInput` 在这两端返回 false，调用方可据此降级（当前只有 webview host 使用，Windows 之外就等于「没有快捷键」，与之前一致）。
2. `devToolsOpen` 是 host 侧记录：若用户从 devtools 窗口自身关闭，下一次快捷键会以「打开」开始（第一次按会执行 Close，看起来没反应）。原生用 `webContents.isDevToolsOpened()` 查询，mygo 目前没有该查询 API。
3. 本轮没有 GUI 实测（时间盒用完）：Windows 路径由 AcceleratorKeyPressed 现有事件驱动、单测覆盖匹配与门控，但「按下去真的开/关」需要一次手动或自动化验证，建议下一轮用一个 temp build 在 webview 窗口里按一次 Ctrl+Shift+I 并截图。

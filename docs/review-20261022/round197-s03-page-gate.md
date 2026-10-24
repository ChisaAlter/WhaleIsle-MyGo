# Round197 · S03 改成页级 WebView2 门控（`Page.SetDevToolsEnabled`）

round193 的结论：合成的 `Ctrl+Shift+I` 由 **WebView2 内建加速键**直接打开 DevTools，MyGo 的 Go 通道（`OnBeforeInput`）根本没被触发；而 Host 页创建时 `PageOptions.DevTools` 是零值 `DevToolsAuto`，在 dev 构建里被解析成 `IsDev()`＝**开**，所以配置里的 `openDevTools` 从来没有真正拦住任何东西。本轮把门控落到 WebView2 自身。

## 1. 机制

`third_party/mygo`：

| 位置 | 内容 |
| --- | --- |
| `internal/platform/input.go` | 新增可选能力接口 `devToolsGater{ SetDevToolsEnabled(bool) }` 与 `platform.SetDevToolsEnabled(w any, enabled bool) bool`（无该能力的后端返回 false，不是假装成功） |
| `input.go` | `func (p *Page) SetDevToolsEnabled(enabled bool) bool` —— 运行时开关，返回该平台是否支持 |
| `internal/windows/devtools_gate.go` | 存 `window.devToolsGate`，若 `w.settings` 已就绪立即 `put_AreDevToolsEnabled` |
| `internal/windows/webview.go` | `setUp` 里在应用创建期 `o.DevTools` 之后，**用运行时门控值覆盖**（页可能比设置更早存在） |
| `internal/linux` / `internal/darwin` | 保存该值（WebKitGTK `enable-developer-extras` / WKWebView `isInspectable` 尚未接线），不谎称生效 |
| `internal/fake` | 保存并提供 `DevToolsEnabled()` 供测试 |

`internal/desktop/webview`：

- `devTools_shortcut.go` 新增 `devToolsPage` 接口、`registerDevToolsPage`（注册时立即应用当前门控）与 `applyDevToolsGate`（把当前门控推给所有已注册页）。
- `SetDevToolsShortcutAllowed` 现在会在设置变更后调用 `applyDevToolsGate()`——配置 `openDevTools` 改变时，已经打开的 Host 页立刻跟着变。
- `host.go` 两处建页（`:288`、`:446`）都 `registerDevToolsPage(pg)`，因此门控在页面创建时就生效。
- **Go 通道保留**作为次要路径（`installDevToolsShortcut`）：它只在 `AcceleratorKeyPressed` 真的送来该组合键时才会执行，而 round193 证明内建加速键先处理了；保留的原因是不影响内建行为、且为将来 WebView2 行为变化留一条可控路径。**真正的门控是页级开关**，因为内建加速键查的就是 `AreDevToolsEnabled`。

结论：门控关闭时，页面自己也无法打开 inspector（内建加速键被 WebView2 拒绝），而不是「靠 Go 侧拦截按键」。

## 2. 测试

- `internal/desktop/webview/devtools_gate_test.go`（新）：注册即应用 → 关掉门控后**已存在页**收到 false → 关闭状态下新建的页从 false 开始 → 通过 `SetDevToolsShortcutAllowed` 重新打开后两页都变 true → `registerDevToolsPage(nil)` 不 panic、不增加页面。
- 既有 `TestDevToolsShortcutGating` / `TestIsDevToolsToggle` 仍通过（门控谓词与组合键匹配未变）。
- `go test ./internal/desktop/webview` ok 1.7 s；`go vet` 0；`go build ./cmd/whaleisle` 0。
- `third_party/mygo`：`go build ./...`（windows/linux/darwin）均 0，`go vet ./...` 0，`go test ./internal/platform` ok。

## 3. 实机验证 —— **未完成**

本轮没有重跑 round193 的 Host 探针：那个探针 app（`tmp/hostprobe/`）在 round193 结束时已删除，重建它需要重写 Host 构造 + 起临时页 + 注入按键 + 枚举窗口（≈15 分钟），在 55 分钟盒子里与实现/测试冲突。**因此「打开 → inspector 出现；关闭 → 无 inspector」的实机证据本轮缺失**，只有单元测试证明门控值确实到达每一个 Host 页、以及 WebView2 设置项被写入。

下一步（≈15 分钟，同样步骤）：重建 `tmp/hostprobe`（照 round193 §A.1），`WL_HOSTPROBE_GATE=allow` 与 `deny` 两次运行，按 `Ctrl+Shift+I` 后枚举 `msedgewebview2.exe` 里 title 含 `DevTools` 的窗口并截图；deny 运行必须为 0，allow 运行必须出现并可再次关闭。

## 4. 残留

1. 实机证据缺失（见上），打包态（`IsPackaged()==true` 且 `openDevTools=false`）的产物仍然无法在本机验证，只能靠单元测试覆盖谓词与页面门控推送。
2. linux/darwin 的页级门控只保存值、暂未接到 `enable-developer-extras` / `isInspectable`；这两端目前等于「引擎默认」，S03 的实际门控只在 Windows 生效。
3. 桌宠窗口是普通 `mygo.Window`（不经 Host 注册表），未纳入本轮门控；若需要，可复用同一个 `SetDevToolsEnabled` 在 `sync()` 里设置。
4. 未提交、未构建 passN、未改 runtime/vendor 与 deepseek-harness；临时文件只有本轮新增的源码与测试。

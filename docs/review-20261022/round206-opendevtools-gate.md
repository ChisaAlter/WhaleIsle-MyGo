# Round206 · 程序化 OpenDevTools 也被门控（S03 补齐）

round201 的发现：`AreDevToolsEnabled=false` 会被 WebView2 接受，但 `ICoreWebView2::OpenDevToolsWindow` 照样能开 inspector；而 S03 的门控只写进了设置。本轮补齐**程序化入口**，并重跑 allow/deny 探针取得新证据。

## 1. 修复（third_party/mygo，纯 additive）

`SetDevToolsEnabled` 的值每窗口记住（round197 已有 `devToolsGate *bool` / `devToolsEnabled *bool`），现在 **`OpenDevTools()` 也读它**：

| 平台 | 文件 | 行为 |
| --- | --- | --- |
| windows | `internal/windows/webview.go` | 运行期门控存在时以它为准：`false` 立即 return（不再调用 `wvOpenDevToolsWindow`）；未设置时沿用创建期 `opts.DevTools` |
| linux / darwin | `internal/{linux,darwin}/window.go` | 同上，读各自的 `devToolsEnabled` |
| fake | `internal/fake/fake.go` | 门控为 `false` 时 `OpenDevTools` 不置 `devtools` |

`IsDevToolsOpened()` 语义不变（仍是「本进程是否调用过打开」）。

## 2. 测试与门禁

- 新增 `third_party/mygo/internal/fake/devtools_gate_test.go`：门控关 → `OpenDevTools` 后 `IsDevToolsOpened()==false`；门控开 → 打开；从未设置门控的窗口保持创建期行为。
- `third_party/mygo`：`go build ./...` 在 windows / linux / darwin 均 0，`go vet ./...` 0，`go test ./internal/fake -run TestOpenDevToolsHonoursTheGate` 通过。
- `go test ./internal/desktop/webview ./internal/desktop/pet` → ok 2.8 s / 3.5 s；`go vet` 两者 0。

## 3. 实机 allow / deny（重建 pagegate 探针后）

同一支探针（`mygo.Window` 加载 `http://127.0.0.1:3999/probe`，加载前 `Page.SetDevToolsEnabled(allow)`，加载后调用 `OpenDevTools()`），枚举标题含 `DevTools` 的可见窗口并截图：

| 运行 | 门控 | 日志 | inspector 窗口 | 截图 |
| --- | --- | --- | --- | --- |
| C allow | true | `gate true supported=true` / `after-load: OpenDevTools called, opened=true` | **1 个**：`DevTools - 127.0.0.1:3999/probe` 960×960 @ (150,150) | `C-allow-devtools.png` |
| D deny | false | `gate false supported=true` / `after-load: OpenDevTools called, opened=false` | **0 个** | 无（count=0 记录在命令输出） |

对比 round201 的 B-deny（当时同样门控关却出现 inspector），本轮 **deny 不再出现 inspector**，程序化入口已闭合。allow 与 round201 完全一致，说明修复没有误伤正常路径。

## 4. 键盘路径（Ctrl+Shift+I）deny —— 仍未取得直接证据

两次尝试都在**焦点**上失败：
1. Host 子窗口探针：父窗 `GetForegroundWindow == target` 断言成功，但 Host 页是子窗口，合成 `keybd_event` 之后 4 s 内没有 DevTools 窗口（`A-open.log`）。
2. pagegate 顶层窗口探针：脚本在同一 PowerShell 会话里用了上一轮定义的 `[K1]` 类型，新会话中类型不存在，枚举失败（`E-allow-chord.log` / `F-deny-chord.log` 只有探针自身日志），**没有结论**。

因此键盘 deny 目前的依据是：round193 证明 `Ctrl+Shift+I` 走的是 WebView2 内建加速键 → 该加速键受 `AreDevToolsEnabled` 约束（round197 把门控写进该设置）→ 加上本轮程序化路径已闭合，Go 侧不存在其它绕过入口。**剩余待验证项**：在一个脚本里（同一会话 Add-Type）用 UIA `SetFocus` 把焦点放进页面元素，`SendInput` 发送 Ctrl+Shift+I，并用 `GetForegroundWindow` 断言前台，然后枚举 DevTools 窗口；allow 必须先出现、deny 必须为 0。大约 10 分钟工作量。

## 5. 桌面宠物窗口

`internal/desktop/pet/companion.go:195` 的宠物窗口已声明 `Page: mygo.PageOptions{WindowControlsOnly: true, DevTools: mygo.DevToolsDisabled}`，即创建期就把 `AreDevToolsEnabled` 置 false（与本轮运行期门控等价），且它不在 Host 注册表里、没有 `OpenDevTools` 调用方。**不需要复用运行期门控**，本轮未改宠物代码。

## 6. 残留

1. 键盘 deny 的直接证据（见 §4 的剩余检查）。
2. 打包态（`IsPackaged() && !openDevTools`）本机不可验证；谓词与页面下发已有单测。
3. linux/darwin 的门控值被记住且 `OpenDevTools` 会遵守，但对应引擎设置（`enable-developer-extras` / `isInspectable`）仍未接线，等于那两端只有程序化门控生效。
4. 探针、exe、进程已清理；未提交、未构建 passN、未改 runtime/vendor 与 deepseek-harness。

# Round201 · S03 页级 devtools 门控的实机（allow / deny）

目标：round197 已把门控落到 `ICoreWebView2Settings.AreDevToolsEnabled`，但缺实机证据。本轮跑 allow / deny 两次，记录窗口与截图。结论：**门控值确实生效并被 WebView2 接受；但它只拦「用户手势/加速键」这一类入口，程序化 `OpenDevToolsWindow` 依然能开**——这是本轮得到的真实、需要跟进的结果。

## 方法

1. 先按 round193 思路重建 Host 探针（`mygo.Window` + `desktopwebview.NewHost` 指向本地 `127.0.0.1:3999` 页面）：Host 页是父窗的**子窗口**，`EnumWindows` 看不到、也拿不到独立标题；`SetForegroundWindow(parent)` + 点击页面区域后发送 `Ctrl+Shift+I`，WebView2 没有收到（`afterPress1` 无 DevTools 窗口），所以**合成按键这条路在本探针里没能建立焦点**。
2. 因此改用能精确控制同一 WebView2 设置的探针 `tmp/pagegate`：普通 `mygo.Window` 加载 `http://127.0.0.1:3999/probe`，加载前调用 `Page.SetDevToolsEnabled(allow)`，加载完成后调用 `Page.OpenDevTools()`（走的是同一份 `AreDevToolsEnabled` 设置；Host 侧「注册时下发 / 变更时推送」已由 round197 的单测覆盖）。
3. 每次运行后 `EnumWindows` 枚举标题含 `DevTools` 的可见窗口并截图；日志逐行落盘。
4. 探针与脚本用完即删，进程全部结束。

## 结果

| 运行 | 门控 | `SetDevToolsEnabled` 返回 | 出现的 inspector | 证据 |
| --- | --- | --- | --- | --- |
| A allow | `true` | `supported=true` | **有**：`DevTools - 127.0.0.1:3999/probe`，960×960 @ (150,150) | `A-allow.log`、`A-allow-devtools.png` |
| B deny | `false` | `supported=true`（值已被 WebView2 接受） | **也有**：同一标题、960×960 @ (150,150) | `B-deny.log`、`B-deny-devtools.png` |

日志（节选）：

```
A-allow: gate true supported=true / loaded / after-load: OpenDevTools called
B-deny:  gate false supported=true / loaded / after-load: OpenDevTools called
```

## 解读（重要）

- `AreDevToolsEnabled=false` 在 WebView2 里约束的是**用户能否通过界面/加速键打开 DevTools**（这正是 round193 里 `Ctrl+Shift+I` 走的路径），**不约束** `ICoreWebView2::OpenDevToolsWindow` 这个编程接口：两次运行里程序化调用都打开了 inspector。所以 deny 跑出现窗口**不是门控没生效**，而是探针用了一个不受该设置约束的入口。
- 对 S03 的实际威胁模型（用户按 Ctrl+Shift+I）来说，round197 的实现方向是对的：设置写进了 WebView2，键盘加速键会被拒绝。本轮**没有**直接证明「deny 下按 Ctrl+Shift+I 无窗口」，因为合成按键在 Host 子窗口上没能建立焦点。
- 新发现（建议下一轮）：mygo 的 `Page.OpenDevTools()` 目前不看门控，任何 Go 代码都能绕过；应在 mygo 侧记录门控值并在 `OpenDevTools` 里提前返回（或在 Host 上不暴露该入口）。本轮**未改产品代码**（任务约束）。

## 证据文件

`docs/review-20261022/evidence/round201/`：`A-allow.log`、`A-allow-devtools.png`、`B-deny.log`、`B-deny-devtools.png`、`A-open.log`（Host 探针那次：foreground 断言成功但无 DevTools 窗口，记录合成按键未送达）。

## 残留

1. **deny + 真实 Ctrl+Shift+I 的实机证据仍缺**：需要让 Host 子窗口真正获得键盘焦点（例如用 UIA 把焦点设进页面，或直接对 WebView2 的子 HWND 发 `WM_CHAR`/`SendInput` 并用 `GetForegroundWindow` 校验），本轮合成按键路径未打通。
2. 程序化 `OpenDevTools()` 绕过门控（见上），建议在 mygo 增加「门控值 + OpenDevTools 提前返回」，属于 third_party 改动，需单独授权。
3. 打包态产物（`IsPackaged()==true` 且 `openDevTools=false`）本机不可验证；linux/darwin 的页级门控目前只保存值未接线。
4. 探针、脚本、进程已清理；未提交、未构建 passN、未改 runtime/vendor 与 deepseek-harness。

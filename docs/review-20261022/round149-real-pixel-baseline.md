# Round149 · 真实窗口像素基线（PrintWindow 恢复可用）

## 1. 方法

round143 修掉主窗首帧死锁后，`PrintWindow(PW_RENDERFULLCONTENT)` 在 MyGo 原生窗口上不再挂起。新增 `scripts/window-capture.ps1`：

- 在独立的 PowerShell 进程里先 `SetProcessDpiAwarenessContext(PER_MONITOR_AWARE_V2)`，再用物理像素 `GetWindowRect` + `PrintWindow(…, 2)` 截图；
- 150% DPI 下窗口实际是 1.5×，必须用 DPI-aware 进程，否则会得到「右侧被裁掉」的假象（第一次 1060×660 的截图就是这种伪影，不是布局溢出）；
- 只截整窗，不截元素；绝不用于 Electron/WebView2 窗口（仍会挂起）。

## 2. 基线（pass275，150%）

| 画面 | 文件 | 物理尺寸 | 对应 DIP |
|---|---|---|---|
| 启动器（浅色） | `evidence/screens/mygo-launcher-pass275-150dpi.png`（由 tmp/launcher-shot4.png 归档） | 1590×990 | 1060×660 |
| 启动器（深色） | `evidence/screens/mygo-launcher-dark-pass275-150dpi.png` | 1590×990 | 1060×660 |
| 欢迎窗 | `evidence/screens/mygo-welcome-pass275-150dpi.png` | 900×1050 | 600×700 |
| 聊天主窗（harness Ready 后） | `evidence/screens/mygo-chat-pass275-150dpi.png` | 2160×1380 | 1440×920 |

聊天主窗是在真实 start→boot→ready 之后截的：侧栏（鲸屿 WHALE ISLE / BASED ON DEEPSEEK HARNESS、新会话、插件、自动化任务、鲸鱼娘、工作区、已归档、账号与登录）、主区极光壁纸、`探索未至之境` + 预览版、composer（选择工作区 / 工作区内修改 / deepseek-official / deepseek-flash / 发送）、右上分支 main + 提交并推送 + 窗口按钮。

## 3. 结论与待比对项

- 真实 UI 可渲染、可截图，之后所有界面 parity 都可以用像素基线做回归，而不是只看代码常量。
- 启动器 1060×660 全宽内容完整，无水平裁切；浅/深色都正常。
- 待与原生逐项比对（下一轮）：品牌图形（聊天侧栏用蓝色鲸鱼娘头像，欢迎窗用黑色鲸鱼标，原生是否统一）、侧栏折叠按钮位置、composer 最大宽度、右上角按钮组顺序/间距、深色 token、壁纸极光分布。
- 截图整窗包含 20 DIP 圆角外的透明像素；逐元素比对时需按 DIP→物理 1.5× 换算。

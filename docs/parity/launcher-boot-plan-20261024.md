# 计划：启动器独立窗口与 boot 揭幕（G04 / G05）

日期：2026-10-24。状态：**规划，未实施**。参考只读：`reference/electron/src/main/window.js`、`renderer/launcher.html` / `launcher.css`、`renderer/boot.html` / `boot.css` / `boot-reveal.css`。

## 1. 原生行为（证据）

- 启动器是**独立 BrowserWindow**：`window.js:896-928 createLauncherWindow`，1060×660、min 860×560、`show:false`、透明、`--dshd-shell-role=launcher`，加载 `renderer/launcher.html`；`showLauncher()` 先 load、再 restore/show/focus（`:930-943`）。
- 主窗口在启动器阶段不出现；start 后主窗口显示 boot 页，harness ready 时用 boot-reveal 的 curtain 收起（`boot-reveal.css`、`window.js:396-500`）。
- 启动器自己也有 48 DIP caption 与窗口控件（`launcher.css:21-42`，32×32、radius 8、padding 12/8/4），与主窗口同一套 chrome。

## 2. MyGo 现状

- `cmd/whaleisle/main.go` 的 `launcherOpen` 是**主窗口内的面板状态**（`internal/ui/launcher.View`，`panel.go`），启动器与对话共用 1440×920 主窗；没有独立 launcher 窗口（`NewWindow` 只有 main/welcome/whalebridge/phone PR）。
- boot 页是主窗口内的 native `ui.Content`（`internal/ui/boot`），退场幕 `curtain.go` 自绘。
- 结果显示差异：启动器尺寸/圆角/任务栏项/焦点行为与原生不同；关闭启动器 = 回到主窗口而非销毁一个窗口。

## 3. 实施分阶段（建议顺序）

1. **窗口壳**：新增 `launcher_window.go`，`NewWindow` 1060×660/min 860×560、透明、`TitleBarHidden`、48 caption（`WindowsCaptionButtons 32/Top12/Right8`）、`AutoHideMenuBar`；内容 `ui.View(launcherView)` 复用 `launcher.View`，先只做冷启动路径。
2. **状态机**：冷启动（非 `--background`、非 welcome）先创建隐藏主窗 + 显示 launcher 窗；start 时关 launcher、显示主窗 boot；错误/取消留在 launcher；托盘「打开启动器」改为 show launcher 窗；深链 `open/launcher` 路由到同一窗口。
3. **单实例/退出**：关闭 launcher = 退出（原生行为需对照），有运行会话仍走 taskguard；`--background` 不建 launcher。
4. **boot 揭幕**：现有 `internal/ui/boot` 的 curtain 时序对齐 `boot-reveal.css` 的退场时长（0.2s×3? 需逐条核对），主窗 show 与 harness ready 的顺序按原生 index 流程。
5. **证据**：窗口选项单测 + 状态机单测（launcher→main 只发生一次、错误不丢窗）+ 1060×660 `ui.Tester` 渲染（launcher 面板几何）+ 冷启动录屏（实机，需 Host 门禁允许）。

## 4. 风险与依赖

- 影响面大：`main.go` 启动块、`launcherOpen` 的所有分支、tray/deep link、welcome→launcher 链路、`--background`、单实例、退出清理；需要独占 main.go 一轮，避免与其他 agent 并行改同一文件。
- 启动器面板当前按主窗尺寸/主题写死处需复核（`internal/ui/launcher` 的 padding/最小宽），先在 1060×660 下用 `ui.Tester` 找溢出。
- 原生关闭 launcher 的行为（`launcherWindow.on('closed')`）与主窗口存亡关系需再读 `index.js` 的 app 生命周期，不能想当然。
- 实机像素仍不可得（PrintWindow 卡死 WebView2），本轮只能到窗口选项 + 无头几何 + 代码级时序。

## 5. 验收口径

- 冷启动：只出现 1060×660 启动器；start 后启动器消失、主窗出现并进 boot；取消/报错不丢启动器。
- 托盘/深链再次打开启动器不新建第二个窗口；退出路径不残留窗口。
- `go test ./cmd/whaleisle ./internal/ui/launcher ./internal/ui/frame -count=1` 绿；`go vet` 干净；无头渲染证据落盘。

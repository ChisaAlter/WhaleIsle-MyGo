# Round143 · 主窗口首帧自锁修复（关键）

## 1. 现象

真实启动 MyGo 主窗口（`--settings`、冷启动或带 workspace）后窗口出现但**永久无响应**：`Get-Process Responding=False`，CPU 停在约 0.2s，工作集不再变化；`pass270`–`pass273` 均可复现。无头 `ui.Tester` 测试全部通过，所以此前从未暴露。此前把 `PrintWindow` 卡死归因于 WebView2，其实是窗口首帧已经死锁。

## 2. 证据（goroutine dump）

用 `WHALEISLE_STACK_DUMP=<file>` 诊断钩子在卡住 8s 后导出全部 goroutine（证据文件 `evidence/main-window-deadlock-stacks.txt`，仅 3 个 goroutine）：

- goroutine 1（UI 线程）：`WM_PAINT` → `mainView` → `state.takeGeneratedCommitMessage()` → `sync.Mutex.Lock` **阻塞**；栈上没有其它持锁者。
- 唯一解释是同一 goroutine 递归加锁：`mainView` 在快照段 `main.go:3105` 取得 `state.mu`，到 `:3317` 才释放；round126 新增的 git 提交信息生成在 `:3251` 与 `:3254` 调用了**自己会加锁**的 `takeGeneratedCommitMessage` / `gitCommitGenerationState`。

Go 的 `sync.Mutex` 不可重入，于是首帧在 `Lock` 处永久等待，窗口消息循环再没返回。

## 3. 修复

- `cmd/whaleisle/git_generate.go`：两处访问器改为 `takeGeneratedCommitMessageLocked` / `gitCommitGenerationStateLocked`，**不再加锁**，并在注释里写明调用方必须持有 `state.mu`。
- `cmd/whaleisle/main.go`：快照段（已持锁）改用 Locked 版本。
- 回归测试 `cmd/whaleisle/git_generate_test.go`：持锁调用两个访问器（若再改回自锁，测试会挂而不是悄悄回归）、一次性消费语义、`nil` 状态。
- 保留 `WHALEISLE_STACK_DUMP` 诊断钩子（仅环境变量启用），便于下次卡死直接取证。

## 4. 验证

- `go test ./cmd/whaleisle ./internal/ui/boot ./internal/ui/i18n ./internal/ui/chat ./internal/ui/composer -count=1` → 19.9s / 0.8s / 0.2s / 58.9s / 6.7s 全绿。
- 修复后 `--settings` 真机启动：`responding=True`，CPU 1.36s，窗口 1440×920；修复前同一命令 `responding=False`、CPU 0.23 恒定。
- 冷启动（复制真实配置、port 3099、临时 workspace）：启动器 1060×660 可见且响应正常，主窗保持隐藏；90s 内未进入 harness Ready（3099 未监听），**start→boot→ready 仍未验证**，原因在 harness 启动环境而非窗口（见残留）。

## 5. 残留

- harness 启动未跑通：临时 profile 复制了真实 `config.json`（dshBin/nodeBin 指向安装版运行时），但 90s 内端口未监听、启动器未关闭；需要一轮带日志的 harness 启动诊断。
- 强制结束会留下 `%TEMP%\mygo-*.sock`（mygo 单实例缺陷），下一次启动会「交给死实例」并静默退出；`scripts/window-geometry-probe.ps1` 已加清理，产品侧未修（工具包问题）。
- 该死锁自 round126 起存在，round126–round142 的“实机窗口”结论均只基于几何（GetWindowRect），不代表窗口可交互；后续实机验收需以 Responding + 交互为准。

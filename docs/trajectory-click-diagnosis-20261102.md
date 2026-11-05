# 轨迹标签点击诊断（2026-11-02）

## 当前结论与下一步

用户在实际 `0.2.1-rc.3` 预览窗口点击“轨迹”后，内容和蓝色下划线均不改变。完整无桌面点击回归未复现这一主故障；尚不能认定已修复。用户退出旧预览后，旁置诊断版已启动，并记录一次真实点击及返回的事件、选择和主视图回写成功。没有观察实际下划线像素，原始主故障原因仍未定位。全量审查中的 SH-005/006 保留这一区分。

原预览启用了 `closeToTray`，单击窗口关闭按钮不会结束进程。用户已确认退出旧预览；诊断启动器拒绝与原预览并发运行，不自动停止任何进程。

## 已修复的直接相关缺陷

`internal/ui/frame/frame.go` 原先先判断聊天的空会话、加载中和历史错误占位，即使已选择轨迹也不会挂载 ledger。现在轨迹页自行显示其空/加载状态，不再被聊天占位分支隐藏。该缺陷本身不能解释下划线不动。

## 有效证据

- 原始源码 overlay 下，三个轨迹 phase 回归按预期失败；当前实现通过。
- 同次定向运行的 frame 包及既有聊天空/加载/错误保护通过；组合命令中的 cmd 观察器夹具因点到侧栏分隔条的命中范围而失败。修正仅测试的观察坐标后，最终 cmd 回归通过。
- 最终 cmd 回归使用 NavigationShell、真实 trailing、Frame、Store follow 投影及主视图同形的局部值/回写链路，覆盖空/有历史、dock 开/关、trace 开/关。检查标签状态、内存渲染的蓝色下划线、ledger 和返回聊天；不是 Windows 物理输入验收。
- 受影响包 `go vet` 和 `git diff --check` 退出 0。

完整命令和日志索引：`tmp/trajectory-diagnosis/validation-result.json`。最终点击回归为 `passive-trace-final.log`，负对照为 `phase-before.log`。

## 旁置诊断入口

可执行文件：`dist/0.2.1-rc.3/whaleisle/whaleisle-trajectory-diagnostic.exe`，构建版本 `0.2.1-rc.3+trajectory.1`，Go 1.27.2 / Windows amd64 / CGO 关闭。来源为 `b8b93ed8` 加本轮未提交修复；构建收据及命令位于 `tmp/trajectory-delivery-20261102/build-result.json`、`build-diagnostic.ps1`、`build-diagnostic.log`。

启动脚本：`tmp/trajectory-delivery-20261102/launch-diagnostic.ps1`。使用现有独立配置 `%LOCALAPPDATA%/WhaleIsle-MyGo/preview-0.2.1-rc.3`，共享既有 stage runtime。用户确认退出旧预览后，诊断版于 2026-11-02 15:27:55（北京时间）启动，PID `4948`，窗口标题“鲸屿go”。15:35:16 的真实用户点击已记录 trajectory down/up、chat→trajectory 选择、main 回写及下一帧布局；15:35:19 反向切回 chat 也有同链路记录。该证据只说明这次诊断版状态切换成功，原 rc.3 点击无响应的原因仍未定位。固定字段摘录见 `docs/parity/full-audit-20261102/evidence/trajectory-tab-trace.txt`。启动收据为 `tmp/trajectory-delivery-20261102/launch-result.json`。原可执行文件和完整安装包未覆盖。

诊断开关默认关闭。启用后只分析 `[conversation-tab]` 行：固定节点 `caption/trailing/chat/trajectory` 的矩形、指针 down/up、选择 before/requested/after 和 main 回写。新增日志不包含会话 ID 或聊天内容。实际诊断日志预定为同一 profile 内的 `conversation-tab-diagnostic-20261102.log`。

现有真实点击日志表明，该次诊断版的事件到达并完成选择回写；它没有解释原 rc.3 的失败条件。原始故障继续保留，不能以这次成功关闭问题。本轮未操作桌面、未改第三方 UI 引擎、未重打完整安装包。

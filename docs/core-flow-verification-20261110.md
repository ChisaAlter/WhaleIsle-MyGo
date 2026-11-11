# 2026-11-10：核心会话链路与原生历史呈现通过

本轮针对“新建会话 → 发送 → 流式输出 → 审批 → 停止 → 重启恢复”执行定向验证。生产 Go Store、RPC/WebSocket 客户端和真实 Host 参与运行；模型端使用本机可控 HTTP/SSE 响应。另用当前源码编译的隐藏原生窗口读取同一份历史，并检查实际截图。

## 验证对象

- 源码基点：`31915c16`，加本轮开始时已有的 140 个已修改文件和 27 个未跟踪文件；并非该提交的干净版本。
- 工具链：Go 1.27.2，Windows amd64，`CGO_ENABLED=0`。
- Host：仓库 `runtime/vendor/deepseek-harness`，包版本 `0.2.1-rc.2`；使用 `runtime/node/node.exe`。
- 每个流程使用隔离 profile 和合成工作区。原生窗口复用了本轮核心链路生成的 profile 和会话；没有读取现有账户凭据或请求外部模型。

## 实际结果

| 检查 | 结果 |
| --- | --- |
| 生产 Composer 创建会话、打开 follow | 通过 |
| 发送、HTTP/SSE 输出、持久化 assistant 消息 | 通过 |
| Host 实际读取合成工作区文件 | 文件内容同时到达模型端和持久化 Go Store |
| 运行中停止 | 模型端 HTTP context 取消，Store 恢复 idle；本次本地场景耗时 34 ms |
| 停止后继续发送 | 同一会话下一轮正常完成 |
| Host 停止并重启 | 新 Store 恢复四条用户输入、回复、停止文本和文件工具结果 |
| 问题回答与拒答 | 生产 Interaction Store 提交后，真实 Host 返回结果，待答卡关闭 |
| 写文件审批拒绝 | 文件不存在，拒绝结果返回模型并持久化 |
| 单次批准 | 只写入指定文件；随后未获批准的写入仍被拒绝，权限保持 read-only |
| 原生窗口 | 当前生产 mainView 显示恢复的 17 条历史记录；聊天、停止标记和输入区实际可见 |
| 桌面启动读取 | Host、Mux、工作区、会话列表就绪；settings、plugins、schedule 查询成功 |

核心流程耗时 40.207 秒，产生 6 次本地模型请求。审批流程产生 15 次本地模型请求，7 次工具调用均返回实际结果。这些次数不包含任何外部模型请求。

## 修复了验收工具的退出错误

首次隐藏窗口已成功启动、读取历史并截图，但验收进程没有退出。`stopNativeAcceptance` 直接设置 `state.quitting = true`，随后调用 `App.Quit()`；生产 `OnClose` 在 `quitCleanupFinished` 为 false 时会拒绝关闭，旧工具绕过了正常退出清理流程。

本轮只修正 `cmd/whaleisle/native_acceptance_windows_test.go` 的测试收尾：停止自有 Host 后，销毁自有隐藏窗口再结束事件循环。没有修改产品退出逻辑，也不据此认证正常用户退出流程。

保留首次失败收据后终止该自有测试进程，修正后的同一验收场景退出码为 0。最终核对本轮进程残留为 0，桌面 Host 的 32480 端口已关闭。

## 构建与定向检查

- 当前 `cmd/whaleisle` 主程序编译通过；`nativeacceptance` 测试程序编译并实际运行通过。
- 会话创建、流式状态、取消状态、过期 follow 隔离、断流恢复和失败提交保留检查通过。
- 审批回执信封、坏回执拒绝、过期交互回执隔离检查通过。
- 恢复数据的重启读取、冲突写入拒绝、损坏/锁定文件保护检查通过。
- 主程序启动选择、恢复检查点、项目恢复保稿及普通 Team 初始装配的定向测试通过。Team 此处仅为装配测试，未验证真实团队任务执行。
- 最初的名称筛选没有匹配到 dsh/recovery 用例，随后按实际测试名补跑；没有将“no tests to run”计为通过。

## 证据与尚未覆盖的范围

证据目录：[`tmp/core-flow-20261110/`](../tmp/core-flow-20261110/)。它位于本地忽略目录，Git 不保存这些运行产物。

- [核心链路结果](../tmp/core-flow-20261110/core-result.json)
- [问题与审批结果](../tmp/core-flow-20261110/interaction-result.json)
- [首次测试收尾失败](../tmp/core-flow-20261110/native-first-teardown.json)
- [修正后原生窗口结果](../tmp/core-flow-20261110/native-fixed.png.json)、[实际截图](../tmp/core-flow-20261110/native-fixed.png)、[成功退出日志](../tmp/core-flow-20261110/native-fixed-stdout.log)

尚未覆盖：真实外部模型、窗口按钮与鼠标命中、键盘/输入法、桌面草稿恢复的完整操作、正常用户退出、真实 Team 任务、安装升级与回滚。源码主程序编译和隐藏窗口检查不能代替安装版验收。本轮未提交、推送、触发 CI、打安装包或替换现有应用。

下一阶段优先验证实际交互和正常退出，再检查 Team 执行与安装交付。

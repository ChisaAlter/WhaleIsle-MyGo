# 本机运行验收：不通过

2026-11-02 19:24–19:28（北京时间），在本机运行现有交付程序、实际随包 Host 和生产组件，**四条关键路径失败，当前版本不能交付**。本轮没有接管鼠标键盘，也没有重建产品。

## 实测失败，不再只有源码推断

| 功能 | 运行层级 | 实际结果 | 用户影响 |
| --- | --- | --- | --- |
| **WS-002：全局新会话** | 当前生产 Go Store + 实际 stage Host + 编译的生产侧栏过滤函数 | `session/create` 仅带 cwd，没有 workspaceId；会话被选中，但 Host 的独立 workspace/follow baseline 中没有该成员，侧栏过滤结果不包含它。 | 创建后会话可从工作区侧栏消失。 |
| **SET-002：普通登录** | 随包 Host bundle 的真实 `PlatformAccount.startSignIn` 方法组件 | 收到 `whaleisle://auth` 后立即抛 `protocol / account: protocol`；尚未读取账户服务或进入认证请求，fetch 次数为 0。 | 当前普通登录参数在本地校验阶段就不成立。 |
| **FLOW-007：Project 桥接** | 当前生产 Go `projectbridge.Bridge.Dispatch` | check-directory、baseline、changes、revert 四项均返回 `unsupported Project operation`。 | 目录预检、需求起点、差异和撤销所需桥接断开。未执行真实 revert。 |
| **PLUG-023：MCP 添加** | 当前生产 Go `mcpconfig.Controller` | 生产添加回调对应的 `Edit("")` 前后状态完全相同：Form=null，没有错误提示；已有 OpenForm 对照能建立表单并正常取消。 | 顶部添加所用调用没有产生可编辑表单。未启动或安装 MCP。 |

这些运行层级分别记录，没有把方法/控制器检查称作整 EXE 点击验收。WS-002 的真实 Host 结果支持全局入口归属缺陷，但**用户当时具体点击无响应的根因仍未知**；内联加号和全局新会话不是同一条路径。

## 两项范围内通过

### 现有诊断 EXE 可完成真实启动和 Host 握手

直接启动 `dist/0.2.1-rc.3/whaleisle/whaleisle-trajectory-diagnostic.exe`，版本 `0.2.1-rc.3+trajectory.1`，99,364,352 字节；使用同 stage runtime、独立 QA profile 和已经注册的临时工作区。配置、原生应用初始化、窗口创建和实际 Host 认证握手均到达，Host ready 标记为 **5,171.838 ms**，loopback TCP 连接成功。

这是隐藏隔离模式下的已有工作区启动结果，不是首用默认目录、可见窗口布局、按钮点击或性能基准。结束时仅终止本轮自建进程及所属子进程，监听已释放；该清理动作不作为正常 UI 退出验收。

### 工作区专用新会话控制器的对照通过

在同一个真实 Host、同一目录下，`CaptureWorkspaceSessionTarget → Create → Select` 明确提交 workspaceId；独立 Host baseline、Store 成员、选中结果和生产侧栏过滤均包含该会话。全局路径与这条路径的实际差异已经定位，不必继续猜测 Host 是否只凭 cwd 自动归属。

## 候选和证据不能混用

- 本轮实际 EXE 来源为 `b8b93ed8` 加既有轨迹诊断工作树，构建收据见 `tmp/trajectory-delivery-20261102/build-result.json`；本轮未重建。
- Host 直接来自 `dist/0.2.1-rc.3/whaleisle/runtime`，实际版本 `0.2.1-rc.2`。
- Go 组件检查编译当前 production 包，未修改相关产品包；与实际 EXE 启动结果分列。
- 旧完整 rc.3 安装器 app 源码为 `5549e627`，未包含后续轨迹修复；本轮没有重新安装或把诊断结果套到旧安装器的界面。

原始结果与命令：

- [统一运行收据](evidence/runtime-acceptance.json)：四项失败、两项范围内通过、候选和清理状态。
- [新会话运行报告](../../../tmp/acceptance-runtime-20261102/workspace/RESULT.md)、[完整请求和 Host baseline](../../../tmp/acceptance-runtime-20261102/workspace/report.json)。
- [登录/MCP/Project 组件运行说明](../../../tmp/acceptance-runtime-20261102/contracts/RESULTS.md)、[账户结果](../../../tmp/acceptance-runtime-20261102/contracts/account-result.json)、[Go 结果](../../../tmp/acceptance-runtime-20261102/contracts/go-result.json)。
- [实际 EXE 结果](../../../tmp/acceptance-runtime-20261102/desktop/result.json)、[启动阶段记录](../../../tmp/acceptance-runtime-20261102/desktop/startup.json)、[执行脚本](../../../tmp/acceptance-runtime-20261102/desktop/run-owned-exe.ps1)。

WS-002 检查的运行退出码是 1，与产品验收失败一致；组件探针退出码 0 只表示成功取得预期错误，不表示产品通过。

## 保留有效旧证据，不重复跑

此前实际 stage Host 的八项核心检查继续有效：创建/跟随、回复、真实文件 read 工具、停止、下一轮和重启后持久化恢复；模型为自建 loopback 确定性服务，不是外部模型。原报告为 `tmp/stabilize-20261102/packaged-headless/report.json`。独立 GUID 的安装/升级/卸载事务也保留其原范围。本轮没有重复这些通过项。

## 尚未完成的验收和下一步

本轮完成的是已选关键路径的非接管式本机运行检查，**不是 367 项运行验收全部完成**。当前实际窗口的轨迹点击/下划线、顶部布局、窄窗、IME 和其他可见交互尚未验收；既有用户截图和一次诊断日志只保留原范围。外部账号/真实密钥/模型、IM、MCP 实装、远程设备、音视频及原产品 ID 接管也未执行。

WS-003 删除取消风险仍是交付阻断，没有拿真实或临时会话执行破坏性取消实验。修复顺序应先处理删除取消，再处理新会话身份、普通登录、MCP 添加和 Project 桥接；修复后只补这些实际失败路径及受影响行为，再进入其余领域验收。当前不能继续出一个“可用版”安装包。

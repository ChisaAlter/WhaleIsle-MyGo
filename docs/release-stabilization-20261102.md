# 2026-11-02 Windows 候选交付

## 当前候选

Windows `0.2.1-rc.3` 已完成无桌面核心验收、完整安装器生成及完整载荷隔离静默安装、升级、卸载。实际安装暴露的 NSIS 长路径错误已修复，并通过同长度目录回归和完整事务验证。本轮按用户要求不再操作桌面；未推送、上传或触发 CI。

| 项目 | 结果 |
| --- | --- |
| 安装包 | `dist/0.2.1-rc.3/WhaleIsle-Setup-0.2.1-rc.3.exe` |
| 大小 | 292,618,167 字节（约 279.06 MiB），包含完整 runtime |
| 程序源码 | `5549e6278824dbdea7f9fddc7b5f94b07b0873b6`，程序和 runtime stage 未变，复用有效验收 |
| 安装脚本源码 | `4863a0dea6238c3274e80838547ffa302b9a34ff` |
| 构建 | Windows amd64、Go 1.27.2、CGO 关闭；可执行文件 VCS 元数据为程序源码 `5549e627` 且 `vcs.modified=false` |
| Host | `0.2.1-rc.2`，保留原固定来源和 readiness gate；不随桌面版本递增改写 manifest |
| 校验材料 | 同目录 `SHA256SUMS.txt`、安装包 `.blockmap.json` |
| 发布状态 | 本地未签名候选；未发布、未替换原安装 |

## 已通过的无桌面验收

### 打包后的真实 Host 与持久化

直接消费最终 `dist/0.2.1-rc.3/whaleisle/runtime`，使用真实 Host、生产 Go composer/Store、RPC 和事件流，8 项通过：创建/跟随会话、完整 SSE 回复、真实 `read` 工具读取独占文件并回传、流式停止、同会话下一轮、Host 释放和重启后历史恢复。

重启后恢复了 4 条原始用户输入、回复、停止文本及成功文件工具结果。取消到 Store idle 为 30 ms。6 次模型请求均来自自建 loopback 确定性夹具，没有调用真实外部模型。自身 Host、模型服务和监听全部释放。

实际 mobile 资源消费者 `TestPackagedMobileBuiltFixture` 也消费了该 stage 的真实资源与发布清单，85.62 秒通过，无 SKIP；这不代表手机真机配对通过。

证据统一位于 `tmp/stabilize-20261102/packaged-headless/`：`report.json`、`result.json`、`packaged-gate.log`；复现入口为 `run-packaged-gate.ps1`。此前源 runtime 核心验收 `headless/report-03.json` 同样通过，最终交付以打包后报告为准。

### 工作区控制器

`tmp/stabilize-20261102/workspace-functional.json` 的 5 项通过：真实 Host 默认工作区初始化、目录列举/创建/失败重试、选择持久化失败与重试、重命名/取消/移除且保留文件。模型请求为 0，自有 Host 和监听均释放。

既有 QA 工具补上 Store 初始化调用，并把 Host Documents 策略显式指向独占目录；不再假设 Windows 系统 Documents 等于子进程 cwd。首轮涉及的系统默认文件夹原已于 2026-10-05 存在，本轮未删除或改写其内容。

### 构建与定向自动化

- 正式源码构建、六个纯逻辑包、会话/Host/UI 逻辑及相关打包 source fixtures 通过。
- Go 有效结果由 `qa/go-gate.log` 的通过包和仓库外独占 Temp 中的 `go-affected-gate.log` 三项补测组成；最终完整 `qa/go-harness-final-gate.log` 和全根 `qa/go-vet-final-gate.log` 均退出 0。
- Node 有效结果由首轮通过项、`qa/node-affected-gate.log` 的 27 项通过和最终 `qa/node-model-gate.log` 的模型资源用例通过组成。`node-affected-gate.log` 本身仍为退出 1；其中遗留的模型 fixture 缺文件随后修正并单独通过，未扩大 SKIP 或削弱断言。`qa/node-package-source-gate.log` 的 3 项仅认证源码/资源契约。
- 上述冻结前日志中的 HEAD/SourceRevision 为原基线加当时工作树，不应读作直接执行于新提交；成果随后纳入源码冻结。打包后 Host 与 mobile 报告明确绑定最终产品源码提交。
- 首轮失败日志保留。完整 whitespace 扫描仍有固定上游 JS 模板和许可证的原有空白警告，保留其原始字节；本轮变更中的 Go 文件已格式化，记录见 `source-whitespace.json`。

未写明前缀的日志和收据以 `tmp/stabilize-20261102/` 为根。

## 本轮修复与打包输入

- 用嵌套 `tmp/go.mod` 隔离诊断/备份 Go 文件对根模块 `./...` 的污染，未删除临时文件。
- 修复 Host child cwd 回归：有效工作区继续生效；失效目录继承原 cwd 并记录提示。相对自定义启动器、Node、home 与 patch 路径在切换 cwd 前固定为绝对路径，保留原调用能力。
- NSIS 非零退出、缺失或空安装包现在明确失败；完整载荷的隔离事务等待上限由 120 秒调为 900 秒，不减少动作。
- 原生字体嵌入可执行文件，但 OFL 说明没有随包。现从权威源复制 Inter、Source Serif 4 许可证及 `sources.json` 到 `resources/licenses/fonts/`；静默安装后已逐字节核对三份文件。
- 首次完整打包被严格 runtime 校验正确拒绝：既有 manifest 声明的两个后台补丁输出与实际固定原始文件不一致。保留失败产物为 `dist/0.2.1-rc.3-input-rejected`，在 `tmp/stabilize-20261102/verified-runtime` 独立复制图中应用既有精确钉定补丁；原 runtime 未改写，严格校验器未修改。`runtime-input.json` 记录 25,715 文件、3,146 链接及校验通过；该副本是本次完整打包输入。
- `package-result.json` 与 `package-rc3.log` 记录最终完整 NSIS 编译、非空安装包、blockmap 和校验清单成功。
- 实际静默安装暴露 `SetOutPath` 的长路径缺陷：260 字符目录能创建，但其隐含 `SetCurrentDirectoryW` 返回 Win32 206。共享 `WI_SetPayloadOutPath` 现在创建目录并设置 `$OUTDIR`，不切换 CWD；root/resources/runtime/pet 及生成的 runtime include 全部使用同一宏，保留事务的 `SetOutPath $TEMP`、所有权检查和回滚逻辑。`nsis-longpath/diagnostic-cwd.txt` 给出因果证据，`nsis-longpath/regression/receipt.json` 记录同长度小型合成载荷的真实安装、强制失败回滚、升级和卸载通过；深层文件及声明 junction 均正确。相关 Go/Node 定向测试通过。

## 安装事务

使用同一完整 stage 和修复后的 NSIS 脚本，另编译唯一 GUID 产品身份 `5ebe44a022404761b42110e253267850` 的验收安装包，在独占 Temp 中完成静默安装、覆盖升级和卸载，全部通过。未启动桌面程序，也未接管真实产品注册项。

安装及升级后的程序字节均匹配已冻结构建，实际安装后的字体许可证和来源记录正确。升级未保留退休载荷；卸载后自身安装目录、退休目录和两项 GUID 产品注册项均消失。证据为 `install-result.json`、`install-input.json` 和 `install-rc3.log`，最终隐藏任务退出 0。

首轮 GUID `bef12cd78b61431baa276984906f0fed` 的静默安装返回 2，升级/卸载未执行；其独立 `.new-{69235D1D-1222-4DF1-8D71-905E0948AB14}` 暂存目录保留。错误终态见 `install-rc3-before-longpath-fix.log`，修复前编译记录见 `package-rc3-before-longpath-fix.log`，故障安装器和旧 include 已归档到 `dist/0.2.1-rc.3/installation-rejected/`。修复后沿用相同长度的独占 Temp，没有缩短目录；仅重编安装器，未重复构建已验收的程序或复制 runtime。

## 源码和恢复材料

初始基线为 `d78024d0112d7ade673351f4ebaad78c80c14f5e`；跟踪差异、579 个未跟踪源码文件和原 index 保存在 `tmp/stabilize-20261102/baseline*`。本地 `d0d337e0` 冻结整理成果，`5549e627` 补齐字体许可证交付，`4863a0de` 修复安装器长路径。原应用、现有数据和旧产物保留，未推送。

此前桌面诊断轮曾观察到主界面和临时工作区，随后 Computer Use 被用户 Escape 停止；该轮收发/窄窗检查未完成，不作为最终无桌面验收结果。停止时保留的独立 profile 与诊断实例位置见 `desktop-run.json`、`model-fixture-run.json`。

## 验收边界

本轮覆盖无桌面核心接口与独立产品身份的安装事务。窗口布局、窄窗、系统输入法、物理交互、真实外部模型、同产品 ID 的现有安装接管、手机第二设备、macOS 硬件及长时功耗仍未验收。CI 历史累计未通过次数未知，本轮未触发 CI。10 月 27 日等历史候选不能替代本轮产物证据。

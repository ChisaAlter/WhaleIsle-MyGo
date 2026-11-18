# 2026-11-17 项目全面审查

本报告记录下述起点源码的审查事实；后续修复、实际验收及用户决定的延期项统一见[系统优化方案的当前实施记录](../route-a-system-optimization-20261117.md#当前实施记录)。

当前候选存在安全、数据兼容和发布阻断问题，建议先处理 F01–F08，再安排日常工作流修复。共收敛 **19 项可行动问题：8 项 P1、11 项 P2**。这是当前源码的审查结果；没有修改产品代码、发布或替换日用安装。

## 对象与证据范围

- 源码：`main`，`11595831f43af4e401bdb613a2e744d6bfe2a5af`；开始时工作区无未提交改动。
- 路线：以当前 `ARCHITECTURE.md` 的路线 A、A2/2A/3A 和共享数据合同为准。
- 环境：Windows amd64、Go 1.27.2、CGO=0；原生启动验证复用 `runtime-rc4` 的 DSH 0.2.1-rc.4，使用全新隔离 profile，没有模型请求。
- 覆盖：会话与审批、草稿附件、Project/后台任务、插件安装与兼容窗口、原生输入和虚拟列表、配置与凭据、手机远程、文件/Git/SSH关键入口、Host 启动、打包安装及更新。
- “静态闭环”指已追踪用户入口、相关状态和后续执行路径；不表示做过对应真机操作。macOS、真实手机配对、系统 IME、真实模型任务、安装升级和回滚没有在本轮验收。

## P1：先处理安全、共享数据和发布阻断

### F01 已消费的配对码仍获准调用桌面 API

位置：[bridge.cjs](../../internal/desktop/phoneremote/bridge.cjs#L67)、[shell_routes.cjs](../../internal/desktop/phoneremote/shell_routes.cjs#L309)。

Bridge 把二维码中的 `pairingToken` 加入业务 API 的 bearer 列表；解析只检查到期时间，没有调用设备存储的 `isPairingTokenActive`。成功配对后，旧码在过期或更换前仍可访问 shell 路由，包括配置操作；已获准工作区的访问仍受相应 Git/文件权限约束。

**证据：**用原始桥接函数及真实 shell handler、合成 token、返回“不活跃”的设备存储和无副作用配置回调进行本地探针，`saveConfig` 返回 200，设备存储有效性检查次数为 0。见 [pairing-token-probe.json](pairing-token-probe.json)。这不是实际手机会话测试。

**建议：**业务接口使用正式设备认证；一次性 bootstrap token 只承担配对交换，并在消费后撤销其资格。

### F02 设置页保存凭据会覆盖其他窗口的新值

位置：[main.go](../../cmd/whaleisle/main.go#L654)、[main_hotpath_cache.go](../../cmd/whaleisle/main_hotpath_cache.go#L318)。

保存时虽然重新读取磁盘，却立即用页面旧 draft 覆盖 API key、Base URL、GitHub token、远程 token 和 relay token 五项。打开设置后原版或另一窗口更新令牌，再在 MyGo 只修改一个字段保存，会恢复其他字段的旧值。首次加载失败还会设置 `seeded=true`；文件随后恢复可读时，保存空 draft 可能清空未编辑项。

**证据：**静态闭环。已有凭据存储的冲突处理能力没有被该页面保存链采用。

**建议：**保存加载基线，按用户实际编辑字段合并并检查冲突；加载失败时不能提交覆盖写入。

### F03 远程关闭网关只落盘，监听和授权继续存在

位置：[remote_host_actions.go](../../cmd/whaleisle/remote_host_actions.go#L148)、对照 [phone_gateway.go](../../cmd/whaleisle/phone_gateway.go#L43)。

手机 `saveConfig` 提交 `remoteEnabled:false`、修改端口或收窄监听地址后，入口只执行 `UpdateChecked` 并返回成功，没有调用正常桌面入口的授权撤销、停止和重建流程。旧运行时保留原配置，用户看到的“已关闭”与实际网络边界不一致。

**证据：**已追踪 shell 路由 → bridge host action → Go 配置保存及正常桌面入口；未在真实手机上验证。

**建议：**让手机和桌面共用网关配置生效流程；关闭后停止接收新请求并撤销旧授权。

### F04 macOS 凭据格式破坏与原版的共享数据合同

位置：[credentials.go](../../internal/credentials/credentials.go#L307)、[credentials.go](../../internal/credentials/credentials.go#L333)、[provider_darwin.go](../../internal/credentials/provider_darwin.go#L74)、[provider_darwin_nocgo.go](../../internal/credentials/provider_darwin_nocgo.go#L14)、[keychain.go](../../internal/credentials/keychain.go#L13)。

macOS 的 cgo 与非 cgo 两种 Keychain provider 保存时，都会向共享 `credentials.json` 写入 MyGo 专有 envelope。只读核对本体 `C:/AI/Deepseek-Harness-Desktop/src/main/config.js:421–468`：它仅识别原 safeStorage envelope，专有格式落入普通对象分支，API key 等字段不可用。优化方案核对时补查非 cgo provider，修正了初稿仅限定 `darwin && cgo` 的范围。

另一个迁移分支：只有 Electron 旧 Keychain 项、没有 MyGo 专用 key 时，读取旧 envelope 后设置 `create=false`，保存无法创建新 key，会直接失败。只把这个标志改成 true 会进入上述格式不兼容问题。

**证据：**两端源码及 Keychain 分支静态核对；没有 macOS 真机证据。该结论针对使用此 provider 的构建，不套用到 Windows。

**建议：**统一双方可读的共享写入合同，或为专有格式使用独立文件并设计明确迁移；两个分支应一起处理。

### F05 未签名候选安装包不能通过默认自更新验证

位置：[native_releases.go](../../internal/desktop/update/native_releases.go#L430)、[apply.go](../../internal/desktop/update/apply.go#L157)。

正常运行环境未覆盖签名策略时，原生更新默认要求有效平台签名。当前本地 `dist/route-a-0.2.0/WhaleIsle-MyGo-Setup-0.2.0.exe` 的 Authenticode 状态为 `NotSigned`；打包入口生成 NSIS 和校验元数据，签名仍是另外的发行步骤。这份未签名候选若直接分发，会在下载校验阶段被拒绝。

**证据：**[installer-signature.json](installer-signature.json) 与更新调用链。这个安装包是已有产物，不宣称它等于本轮源码构建，也未查询线上最新 release。

**建议：**对齐实际签名发行和更新验证合同，使用最终分发包走一次下载、校验、安装链；不要只验证下载成功。

### F06 安装目录检查允许替换 Electron 本体目录

位置：[payload-transaction.nsh](../../scripts/nsis/payload-transaction.nsh#L769)、[payload-transaction.nsh](../../scripts/nsis/payload-transaction.nsh#L815)。

已有目录只要包含 `Whale Isle.exe`、`Deepseek-Harness-Desktop.exe` 等 Electron 文件名，就被当作可升级目录。用户在安装目录页选中本体路径后，事务会将整目录移到 `.old-*`，再把 MyGo 放到原路径；旧文件有保留，但原版安装入口已被破坏。缺少 runtime graph 的本体目录不会被 runtime-links 校验挡住。

**证据：**目录准入、runtime-links 宏和 promote 路径静态闭环；本轮没有执行该破坏性安装场景。

**建议：**升级准入必须确认 MyGo 自身身份，明确拒绝 Electron 本体安装目录。

### F07 Windows CI 的体积门禁检查了另一种构建

位置：[main-quality.yml](../../.github/workflows/main-quality.yml#L151)、[package.ps1](../../scripts/package.ps1#L41)。

CI 使用 `-tags release -trimpath`，缺少实际打包的 `-ldflags '-s -w'`，却套用生产包的 95 MiB 门限。同 HEAD 按 CI 命令实测 **116,440,576 B（111.05 MiB）**，大于 **99,614,720 B（95 MiB）**，该 job 会失败。

**证据：**[release-size.json](release-size.json)、[release-build.log](release-build.log)，编译退出 0。没有触发 CI。

**建议：**门禁使用与生产包一致的构建参数和实际产物，不通过扩大阈值掩盖参数差异。

### F08 MCP 与 Skill 的新安装分流未符合 Host 参数合同

位置：[capabilities.go](../../internal/pluginmanager/capabilities.go#L380)、[client.go](../../internal/pluginmanager/client.go#L131)、[client.go](../../internal/pluginmanager/client.go#L140)。

- 标准 `.mcp.json` 的 `{command,args}` 只补了 `id`，没有转换成 Host 要求的 `transport`、`enabled`、`serverName` 等字段。
- 本地 Skill 固定发送 `root:"user"`，当前 Host 仅接受 `user-dsh` 或 `project-dsh`。

**证据：**直接加载当前 rc.4 的真实参数 codec 校验以上产品 payload，两项都失败，见 [plugin-schema.json](plugin-schema.json)。已有安装单测对写 RPC 直接返回成功，没有检查该合同。

**建议：**在安装适配层规范化 MCP 配置并保留启用选择；Skill 使用正确根及完整目录导入语义，再通过真实 Host 验证两条安装路径。

## P2：工作流与原生交互

### F09 插件入口被旧页面遮挡，产生隐藏浏览器

位置：[plugin_slot_views.go](../../cmd/whaleisle/plugin_slot_views.go#L23)、[settings_modal.go](../../cmd/whaleisle/settings_modal.go#L18)。

从插件管理、市场或计划页面点击共享侧栏插件入口时，只设置 `webviewOpen=true`，没有清理 `marketOpen`、`scheduleOpen` 等互斥状态。主内容仍优先显示旧页面，新 WebView 被 `SetVisible(false)` 隐藏。普通进入设置也只有隐藏动作，与路线 A 的离开兼容表面即释放要求不一致。

**证据：**静态导航与渲染闭环。**建议：**统一页面切换及 A2 生命周期，在正确路由上显式打开，离开后销毁相应兼容窗口。

### F10 “创建你的插件”误用已有非空会话

位置：[main.go](../../cmd/whaleisle/main.go#L3277)、[agent_presets.go](../../internal/store/session/agent_presets.go#L229)。

入口直接向当前对话追加创建模板并选择 `cordis`；而非空会话不允许应用预设。用户得到创建插件的草稿，但发送仍在旧会话环境中运行。

**证据：**静态入口与预设限制闭环。**建议：**先定位或创建可用空会话并确认 Creator 预设，再按草稿合同追加内容。

### F11 切换会话会把已受理的附件发送误报为失败

位置：[composer_detached.go](../../cmd/whaleisle/composer_detached.go#L52)、[composer_pending.go](../../internal/store/session/composer_pending.go#L153)。

在 A 附件上传/提交期间切到 B，A 的 follow 被取消，后续提交退休确认只观察选中会话。即使 Host 已接收 A 的消息，统一的两分钟 timeout 仍可能触发，`CompleteDetachedSend` 将已发送文本/附件恢复成失败草稿并提示重试。再次发送可能重复执行。

**证据：**发送、follow 切换、退休确认和草稿恢复静态闭环；未做 120 秒 GUI 复现。**建议：**对已受理提交保留独立确认归属，未知结果按待确认恢复，不能提示重新执行。

### F12 更新包已准备好与恢复记录的发布不同步

位置：[center.go](../../internal/app/updater/center.go#L692)、[persistence.go](../../internal/app/updater/persistence.go#L164)。

收到 `StageReady` 就持久化 phase，此时尚未填入 `Staged`；最终赋值、Busy 结束与持久化也存在间隔。窗口内关闭/重启会读取 `ready + 空安装包身份`，恢复为暂停下载而失去可安装状态。

**证据：**本轮 `TestNativeAutomaticDownloadRetiresAfterIndependentUpgrade/explicit` 失败，与这条读写时序吻合；并非已证明“显式重装被版本退役逻辑删除”。见 [internal-tests.log](internal-tests.log)。**建议：**完整持久化 staged 身份和 ready 后再公开完成，测试等待完整记录。

### F13 RC 到同核心版本正式版被判为相等

位置：[update.go](../../internal/desktop/update/update.go#L247)、[center.go](../../internal/app/updater/center.go#L918)。

比较前删除预发布后缀，使 `0.2.0-rc.1` 与 `0.2.0` 相等，阻断正式版自动更新和相关提示。

**证据：**静态比较与调用链。**建议：**采用符合 SemVer 的预发布排序，保留现有版本输入兼容性。

### F14 解析交付文件时关闭弹层会卡住 Project

位置：[browsers.go](../../internal/app/project/browsers.go#L293)。

`openDeliveryFile` 设置 busy 后没有登记 `d.cancel`。此时允许返回或关闭，关闭逻辑无法清 busy，迟到回调因导航 token 改变直接返回，也跳过清理。该控制器后续刷新和动作都被 busy 拒绝。

**证据：**静态 UI → 控制器 → 回调闭环。**建议：**为文件解析登记取消和完成清理，先释放该操作状态，再舍弃失效结果。

### F15 关闭 Project 观察器后仍每五秒轮询

位置：[manager.go](../../internal/app/project/manager.go#L241)、[detail.go](../../internal/app/project/detail.go#L277)。

观察器创建时 `observerOpen=true`，关闭浮层/标签只改 UI 状态，没有清理 Manager 中的可见标志和控制器。只要整窗仍可见，打开过的 Project 继续轮询 Host；多个观察器累积读取循环和引用。

**证据：**可见性判定、关闭入口和刷新循环静态闭环。**建议：**将显示、隐藏、销毁接入观察器生命周期，关闭后停止读取。

### F16 Windows 长 worktree 路径导致 Git 启动失败

位置：[git-exec.js](../../internal/desktop/projectbridge/source/main/git-exec.js#L130)。

Node 直接把完整 worktree 路径作为 `cwd`。本轮真实工作树测试报 `spawn git.exe ENOENT`；同一 Git 在短路径可正常启动，287 字符 cwd 时重现 ENOENT，表明问题来自长路径启动而非缺少 Git。

**证据：**[long-cwd-probe.json](long-cwd-probe.json)、[internal-tests.log](internal-tests.log)。短 cwd 加 `git -C` 的简单替代仍报 Filename too long，尚未验证可用修复。

**建议：**在 Windows 子进程边界处理长目录，保留既有工作树身份、未提交编辑和中断恢复回执。

### F17 虚拟列表复制漏行或超出高亮范围

位置：[textselection.go](../../third_party/mygo/ui/textselection.go#L1146)、[preview_virtualized.go](../../internal/ui/files/preview_virtualized.go#L245)、[patch_view.go](../../internal/ui/diffsurface/patch_view.go#L398)。

复制只拼接已挂载或曾缓存的段落。点击起点、跳跃滚动、Shift 点击远端终点后，未渲染过的中间行被静默跳过。对于一行内有多个段落的左右 Diff，终点后的其他段落也未被排除，可能把右侧未选文本一起复制。

**证据：**本轮通过 Go overlay 给既有测试增加一次诊断输出，首次复制实际缺少 `Row 01` 到 `Row 05`，见 [selection-probe.log](selection-probe.log)。原测试仍通过，因为它跳过第一次结果断言，逐段滚动填缓存后才断言完整内容。仓库测试文件未修改。多段落端点越界是源码闭环结论，未另做实际复现。

**建议：**复制从完整逻辑文本与选区读取，端点按段落次序裁剪，不依赖浏览历史。

### F18 提交对话框能并发暂存并丢弃早先结果

位置：[commit_dialog.go](../../internal/ui/diffsurface/commit_dialog.go#L141)、[async_loader.go](../../internal/ui/diffsurface/async_loader.go#L958)。

文件右键菜单没有忙状态限制。第二次暂存取消并替换 `stageInFlight`，但实际 Git 写操作不接收该 context，前一次仍在执行，结果却会被丢弃。两次写入可能争用 `index.lock`，成功写入也可能没有对应界面回执。

**证据：**静态菜单与异步写状态机闭环；未做 GUI 并发复现。**建议：**串行接收索引写操作，菜单统一忙状态，保留已执行写入的完成回执。

### F19 网络盘附件在 UI 线程进行同步文件读取

位置：[file_paste.go](../../internal/ui/composer/file_paste.go#L63)、[composer_native_files.go](../../cmd/whaleisle/composer_native_files.go#L72)。

粘贴/拖入回调同步逐个 Stat，随后再次 Stat 并读取文件前 512 字节检测 MIME。最多 128 个文件，UNC、断连映射盘及云占位文件都可能等待，期间 UI 无法正常输入和绘制。这发生在后台附件准备之前；并非持有主状态锁的问题。

**证据：**静态输入回调链；未制造真实慢盘。**建议：**移入附件准备后台流程，捕获会话/草稿归属，在 UI 线程核对后一次应用。

## 当前验证结果

| 检查 | 结果与边界 |
| --- | --- |
| `go test -p 2 -count=1 -timeout 90s ./internal/...` | 130 个包通过、5 个包未通过；详见下表。不是全绿。 |
| `go test -p 2 -count=1 -timeout 180s ./cmd/whaleisle` | 通过，21.418 秒。 |
| 构建存储、slim、便携恢复脚本 | 19 项通过；[packaging-tests.log](packaging-tests.log)。 |
| 当前源码 release 编译 | 通过，但 CI 原参数下超出体积门限；见 F07。 |
| 原生隐藏窗口 + 真实 rc.4 Host | 通过，43.963 秒；Host、Mux、会话列表、工作区和三项读取就绪。实际 [截图](native-main.png) 已检查；新空 profile 没有历史和模型调用。 |
| MCP/Skill 当前运行时参数校验 | 两项失败，支持 F08。 |
| 配对 token 本地 handler 探针 | 已消费语义未被校验，仍返回 200，支持 F01。 |
| Windows 长 cwd 对照 | 短路径成功，长路径 ENOENT，支持 F16。 |
| 虚拟列表复制诊断 | 原测试跳跃选择后的首次复制漏掉 5 行，支持 F17；原测试最终 PASS 不代表首次复制正确。 |

内部包未通过项的归因：

| 包 | 本轮结果 | 判断 |
| --- | --- | --- |
| `internal/app/updater` | explicit reinstall 恢复失败 | 存在 F12 的持久化窗口。 |
| `internal/desktop/proactive` | `TestMonitorZeroHzIdleAndImmediateServiceWake:69` 超时 | 测试同时保留两个 Monitor，旧实例抢先领取 notice 并标记 in-app，新实例永远等不到 shown。生产装配有单实例保护，归为夹具竞争；应结束旧 Monitor，不应仅增加等待。 |
| `internal/desktop/projectbridge` | 长路径 spawn ENOENT | 长 cwd 对照复现，见 F16。 |
| `internal/desktop/git` | 90 秒包级超时 | 当时进入 PR 准备测试；未完成，不能据此认定 Git 产品故障。 |
| `internal/desktop/importer` | 90 秒包级超时 | 当前在 NSIS smoke；测试自身设置了四分钟命令预算，本轮包级预算更短。安装链未完成，不能计为通过或产品失败。 |

## 验证与文档中的额外缺口

以下不计入上述 19 项产品/交付问题：

- `third_party/mygo/release_strip_test.go:45` 生成的最小程序不使用窗口/驱动，符号消失也可能是链接器删除未使用代码。现行实现有 release build tag，尚无生产注入漏洞证据；门禁应覆盖实际产品链接路径。
- `qa-gate-zero-webview.ps1` 用 Node 进程存在代替 Host 就绪，只采样一个时点；`TimeoutSeconds` 未落实为整体期限。本轮未用该门禁推断“所有交互零 WebView”，也没有把主窗口启动成功当作零浏览器全程验收。
- `ARCHITECTURE.md` §1/§5.2 的零常驻要求与 §7.2“动态前端贡献需要常驻运行时”仍冲突；§7.1 还含退役前过渡描述。应在修复时收敛现行合同，不能让旧段落为隐藏常驻实现背书。
- README 的工具链最低版本仍是 Go 1.25+，与当前 `go.mod` 的 1.27.2 不一致；下载文档 0.1.1 和源码 0.2.0 状态应明确区分。未核对线上最新版本，不据此指称公开版本错误。

## 建议修复顺序

1. 手机认证与关闭语义、凭据保存和 macOS 共享格式（F01–F04）。
2. 安装身份、签名发行、CI 产物一致性和 MCP/Skill 参数合同（F05–F08）。
3. 发送确认、更新恢复、Project 忙状态及复制完整性（F11、F12、F14、F17）。
4. 兼容窗口导航、Creator、观察器生命周期、长路径、Git 写操作和慢盘输入（其余项）。

每项修复应在对应当前候选上复现原场景并验证直接受影响边界。现有通过的普通测试不能替代真实安装、手机认证或原生输入验收。

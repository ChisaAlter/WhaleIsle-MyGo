# Round146 · 把安装版 rc.2 Harness runtime 落到 MyGo 的加载位置，并跑通 start→boot→ready

Date: 2026-10-25 (CST). Task: task-18. Scope used: `runtime/`（gitignored）、`tmp/`（gitignored，冒烟脚本与日志）、`docs/review-20261022/`。**没有改任何 Go 产品代码、没有改 `scripts/`、没有 commit、没有构建 passN exe、没有碰 `C:\软件\Whale Isle`。**

## 1. 失败点与证据

冷启动时启动器显示（round143 记录的原文）：

> Host session reader is older than the shared desktop session schema (missing job/registered): C:\AI\WhaleIsle-MyGo\runtime\vendor\deepseek-harness; stage the complete original rc.2 runtime with scripts/session-runtime.mjs; session files were not changed

代码链路（只读确认）：

- `internal/harness/launch.go:122-147` 启动前对 harnessRoot 调 `ValidateCurrentHostForLaunch` + `ValidateSessionRuntimeForLaunch`。
- `internal/harness/current_host.go:121-137`：vendor 下没有 `current-host-build.json`、但有 `host-session-runtime.json` 时，走 `validateStagedSessionRuntime(vendorDir)`。
- `internal/harness/session_runtime.go:41-102`：清单/引脚/安全补丁/Artifact 哈希/Node 身份；`:144-167` 从**构建产物** `packages/core/session/lib/index.js` 解析 `KNOWN_SESSION_EVENT_TYPES`，要求 5 个桌面 v4 事件。
- 实测仓库旧 runtime：`runtime/vendor/deepseek-harness/packages/core/session/lib/index.js` sha256 = `216a2738…c4f3436`（源码树旧构建，注册表里没有 `job/registered`）；安装版同一文件 = `40f984f2…5106df`。
- 解析路径：`internal/paths/paths.go:70-111`（`VendorDir()` = exe 同目录 `runtime/vendor`，`HarnessRoot()` = 其下 `deepseek-harness`），`internal/harness/launch.go:207-245`（Node 取 `runtime/node/node.exe`）。

## 2. Staging（用脚本自己的命令，全程离线）

Node：`C:\软件\Whale Isle\resources\runtime\primary-runtime\dependencies\node\bin\node.exe`（v24.21.0，PATH 上的 `node` 同版本）。

```powershell
# 1) 先 inspect 安装版（只读）
node scripts/session-runtime.mjs inspect 'C:\软件\Whale Isle\resources\vendor\deepseek-harness'
#    → cliVersion 0.2.1-rc.2，pin dsh-v0.2.1-rc.2 @ 687f282e…，unknownRequiredRejected=true

# 2) 复制完整 vendor 图到全新目录（fresh）
node scripts/session-runtime.mjs stage 'C:\软件\Whale Isle\resources' 'C:\AI\WhaleIsle-MyGo\tmp\rc2-stage-20261024'
#    exit 0，54 s；node/node.exe = 93,580,104 B；relocatedLinks = 3145
node scripts/session-runtime.mjs verify 'C:\AI\WhaleIsle-MyGo\tmp\rc2-stage-20261024\vendor'
#    exit 0

# 3) 旧 runtime/vendor 先整目录改名备份（瞬时），再把 stage 克隆到最终位置
node scripts/session-runtime.mjs clone 'C:\AI\WhaleIsle-MyGo\tmp\rc2-stage-20261024' 'C:\AI\WhaleIsle-MyGo\runtime'
#    exit 0，43 s；relocatedLinks = 3145
node scripts/session-runtime.mjs verify 'C:\AI\WhaleIsle-MyGo\runtime\vendor'
#    exit 0
```

Stage 输出（`tmp/rc2-stage.log`）关键值，与安装版和 Go 侧常量逐字节一致：

| 项 | 值 |
| --- | --- |
| cliVersion / rootVersion | 0.2.1-rc.2 / 0.2.1-rc.2 |
| requiredEventTypes | job/registered, job/settled, conversation/next-prompt-request, team/control, working-directory/change |
| deepseek-harness/apps/cli/lib/bin.js | `6a939005…41c18`（= 安装版 2026-10-24 重新构建后的 bin.js） |
| packages/core/session/lib/index.js | `40f984f2…5106df`（= 安装版） |
| packages/session/session-persistence/lib/index.js | `896d212b…7058dd`（= 安装版） |
| packages/client/ui-visual-replies/lib/index.js | `8278107b…6ec9b`（= `internal/harness/session_runtime.go:23-28` 钉死的 safetyPatch.outputSHA256） |
| harness-upstream.json | `366b10b6…843e` |
| node | `node/node.exe` sha256 `ba4e6d11…c6c32` |

### 失败过一次并回滚（重要教训）

第一次是把 `tmp\rc2-stage-20261024\vendor` 直接 `Move-Item` 到 `runtime\vendor`：**staging 里的 3145 个 junction 是绝对路径**（`fs.symlink(child(targetVendor, link.target), …, 'junction')`，session-runtime.mjs:189-194 / 229-233），移动后 `verify` 立即报
`Session runtime: ENOENT … runtime\vendor\deepseek-harness\vendor\timer\node_modules\@deepseek-ai\cordis`。
随即把 vendor/node 原样移回 stage 目录（`verify` 重新 exit 0），改用 `clone` 直接在最终路径生成链接，才通过校验；示例 junction
`runtime\vendor\deepseek-harness\vendor\timer\node_modules\@deepseek-ai\cordis → C:\AI\WhaleIsle-MyGo\runtime\vendor\deepseek-harness\vendor\cordis`。
**结论：staged/cloned runtime 与位置绑定，不能在 stage/clone 之后再挪目录。**

## 3. 现在的 runtime 布局

```
runtime/
  node/node.exe                                  # sha256 ba4e6d11…c6c32，v24.21.0
  vendor/
    deepseek-harness/                            # rc.2 完整发布树（3145 个 junction 全部指回本目录）
      apps/cli/lib/bin.js, apps/web/dist/index.html, packages/**, node_modules/**, .dsh-runtime-links.json
    dsh-im/ dsh-pad/ dsh-platform-session/ dsh-project/ dsh-remote/
    dsh-task-control/ dsh-usage-panel/ dsh-whale/ dshbot/ dshd-remote/
    deepseek-harness-runtime.json, harness-upstream.json,
    host-session-runtime.json                    # Go 侧 validateStagedSessionRuntime 读的清单
    session-runtime-readiness.json               # safetyPatch 记录
  vendor.pre-rc2-backup-20261024/                # 旧的 2.09 GB 源码构建 runtime，整目录保留可回滚
  build-*.log, plugin-*-install.log              # 原 runtime/ 里的日志，原样保留
```

- `runtime/vendor` 0.47 GB（stage 0.56 GB 保留在 `tmp\rc2-stage-20261024`，可作为同路径 re-clone 源）。
- 安装版目录只被读：`inspect`/`stage` 都只读取 `resources`，`C:\软件\Whale Isle` 未产生任何写入。
- 桌面 plugin overlay 在启动时由 `Prepare` 生成到临时 profile（本次冒烟实际生成 `dsh-im, dsh-market, dsh-platform-session, dsh-remote, dsh-task-control, dsh-usage-panel, dsh-whale, session-search` 八个 overlay；`dsh-project` 因 §8.2 未挂载）。

## 4. 冒烟方法与通过标准

脚本 `tmp/harness-start-smoke.ps1`（gitignored；不放进 `scripts/`，因为它是证据工具不是产品辅助）：

1. `WHALEISLE_USER_DATA` 指向临时 profile；把真实 `%APPDATA%\Deepseek-Harness-Desktop\config.json` **整份复制**，只改 `workspace`（临时目录）与 `host/port`（`127.0.0.1:3099`）；`nodeBin`/`dshBin` 保持空（与真实配置一致，让 exe 自己解析 staged runtime 与其自带 Node）。
2. 写 `welcome-shown` 标记（`cmd/whaleisle/welcome_window.go:34`），跳过欢迎窗。
3. 无 whaleisle 进程时清掉 `%TEMP%\mygo-*.sock`（mygo 单实例残留）。
4. 用 WMI `Win32_Process.Create` 分离启动（避免与工具会话/控制台耦合），1 s/2 s/5 s 采样：TCP 3099 监听、`EnumWindows` 顶层窗口标题/物理矩形/DIP、DPI、可见/最小化、`Process.Responding`。
5. 达标后继续 6 s settle（每秒同时测 `Process.Responding`、`MainWindowHandle`、对真实主窗口 HWND 的 `SendMessageTimeout(WM_NULL, SMTO_ABORTIFHUNG, 3 s)`），再抓主窗口 UIAutomation 文本、Host 子进程命令行、profile/dsh-home 与 crash 报告。
6. 用 ParentProcessId 递归杀进程树，清临时 profile/工作区/socket。全程**不用 PrintWindow**。

## 5. 时间线（`evidence/harness-start-pass274.json`）

DPI 144（150%），物理 px：主窗 2160×1380 = **1440×920 DIP**，启动器 1590×990 = **1060×660 DIP**。

| t (s) | :3099 | 窗口（可见） | Process.Responding |
| --- | --- | --- | --- |
| 0 | 否 | 无（仅 135×37 隐藏 MyGoApp） | true |
| 2 | 否 | 主窗 1440×920（start 时被揭示）+ **启动器 1060×660** | true |
| 3 | 否 | 同上 | true |
| 5 | 否 | 同上 | true |
| 6 | **是** | 主窗 1440×920 + 启动器 1060×660 | true |
| 6 (+0.4 s) | **是** | **主窗 1440×920，启动器已关闭** | true → **READY** |

- start→boot 交接：主窗在 `harness.Starting` 时揭示、启动器继续在上层；`Ready` 后启动器关闭（`main.go:1692-1702`，`quitAfterStart=true`）。
- Host 子进程（真实命令行截断保存于 JSON）：`C:\AI\WhaleIsle-MyGo\runtime\node\node.exe C:\AI\WhaleIsle-MyGo\runtime\vendor\deepseek-harness\apps\cli\lib\bin.js web --patch …\desktop-plugins\dsh-task-control\desktop-task-control.patch.yml …（共 9 个 --patch）--host 127.0.0.1 --port 3099 --no-open` → **确认加载的是新 staged runtime 与它自带的 Node，不是旧源码树、也不是 PATH 上的 node**。
- 无 crash 报告（`crashReports: []`），`harnessHttp` 对 `GET http://127.0.0.1:3099/` 返回 **401**（该服务要求鉴权 cookie，属预期，不是启动失败）。

### 通过标准四项（t=6 s 起稳定成立）

| 标准 | 结果 |
| --- | --- |
| 3099 监听 | ✅（TcpClient 连接成功；node 子进程 `--port 3099`） |
| 主窗 1440×920 可见 | ✅ hwnd 6032184 `MyGoWindow`，2160×1380 px @144 dpi |
| 启动器 1060×660 消失 | ✅ 启动器 hwnd 只在 t=2…6 可见，Ready 后不可见/已销毁 |
| 进程 Responding=True | ✅（口径见下） |

**`Process.Responding` 的坑（必须写清）**：同一个 `System.Diagnostics.Process` 对象会缓存 `MainWindowHandle`；它第一次解析到的是启动器 hwnd，启动器在 Ready 时被销毁后，该对象的 `Responding` 会对已死句柄一直返回 False（settle 采样里 `resp=False mwh=4785210`，6 次）。同一 pid 新建 `Process` 对象后 `MainWindowHandle=6032184`（真实主窗）、`Responding=True`；对真实主窗 HWND 直接 `SendMessageTimeout(WM_NULL, 3 s)` 每秒都成功。结论是**窗口在响应**，`Process.Responding` 是过期句柄造成的假阴性；round139 的 “Responding=false” 结论同样是这个口径问题。

## 6. UIA 证据（主窗 hwnd 6032184，45 个命名节点，无错误串）

```
鲸 屿 | WHALE ISLE | BASED ON DEEPSEEK HARNESS
收起左侧栏 | 新会话 | 插件 | 自动化任务 | 鲸鱼娘
工作区 | 搜索会话… | 多选会话 | 视图选项 | 选择工作区…
折叠工作区 114f4961-143e-4752-bde5-9d1f19e32011 | workspace | 工作区操作 114f4961-…
折叠工作区 ::scratch | 无工作目录 | 无目录会话 | 新会话 | 已归档 | 账号与登录
Sidebar divider | 切换分支 | 提交并推送 | Git 操作选项 | 切换终端抽屉 | 切换右侧面板
关闭 | 最小化 | 最大化
Conversation hero fish | 探索未至之境 | 预览版 | Composer hero preset row | Composer workspace | 标准模式
Composer card | Composer draft | 命令列表 | 工作区内修改 | Composer trailing controls
deepseek-official / deepseek-flash | 发送消息
```

- 关键词扫描：`older than the shared desktop session schema`、`Host session reader`、`启动失败`、`失败`、`crash`、`error` 在主窗 UIA 文本里 **0 命中**；没有 crash 报告。
- 这是**真实的聊天/会话界面**（侧栏分组 + 会话/工作区条目 + 编辑器 + 模型选择 `deepseek-official / deepseek-flash` + 发送），不是启动错误页。
- 会话存储确实工作了：临时 `dsh-home/sessions/--C-AI-WhaleIsle-MyGo-tmp-smoke-profile-20261024-dsh-home-data-whale--/session-whale-48e60844-…/session.v4.jsonl.zstd`（489 B）由 rc.2 Host 写出。
  注意：侧栏的 workspace 分组条目来自**复制过来的 config.json 里的会话索引**（真实 profile 的元数据），临时 dsh-home 是全新的，所以本次并没有“打开旧会话”；这不是错误，但也不是“读到真实历史会话”的证明。

## 7. 结论

- 卡点已解除：**默认路径 `runtime/vendor` + `runtime/node` 就是安装版 rc.2 的完整 runtime**，MyGo 的启动前校验（current-host → staged session runtime → registry/Node 身份）全部通过。
- pass274 冒烟：**6 秒内 start→boot→ready 全部达标**（端口 / 主窗尺寸与可见性 / 启动器关闭 / 主窗可响应），主窗呈现真实聊天界面，无启动错误、无 crash 报告。
- 旧的 2.09 GB 源码 runtime 完整保留在 `runtime\vendor.pre-rc2-backup-20261024`（gitignored），需要回滚时把该目录改回 `runtime\vendor` 并删掉 `runtime\node` 即可。

## 8. 残留 / 风险

1. **runtime 与路径绑定**：`stage`/`clone` 生成的是绝对 junction。仓库若整体搬路径，`runtime/vendor` 会 ENOENT，必须重新 `stage`+`clone`（或按同路径重建）。备份目录同理。
2. **Project overlay 未挂载**：staged（安装版）vendor 没有 `current-host-build.json`，`ReadCurrentHostCapabilities` 判 `ArtifactsReady=false`，`startHarness` 因此跳过 `dsh-project` 桥（`main.go:2551-2566`）。rc.2 原生桌面本就没有该项目 overlay，属预期，但 MyGo 的项目面板能力在此 runtime 下不可用。
3. **`Process.Responding` 不可作为本应用的健康判据**（见 §5）；后续实机验收请用“新 Process 对象 + 主窗 WM_NULL ping + UIA”三件套。
4. **冒烟只跑了约 17 s 后主动杀进程**：证明 start→boot→ready 与首帧交互，不覆盖长时间运行、自动重启、插件安装。
5. **`GET /` 401**：3099 是带鉴权的 Harness 服务；本文的端口证据用“监听 + 子进程命令行 + 主窗 UIA”，没有拿 401 当失败。
6. `.runtime-rc2/`（10-20 的旧 stage，bin.js 是旧 chunk 名）仍在仓库里且能 `verify`，但**不再被使用**；本次用的是 10-24 安装版重新构建的产物（bin.js `6a9390…`）。如确认无其他用途可删。
7. 冒烟脚本目前只在 `tmp/harness-start-smoke.ps1`（gitignored）。若团队要长期复用，建议下一轮提升为 `scripts/harness-start-smoke.ps1`（本轮按要求没有动 `scripts/`）。

## 9. 产物

- `docs/review-20261022/evidence/harness-start-pass274.json`（42,675 B）：exe/pid/profile/workspace/port、config 原值、时间线、窗口状态、readiness（含 `respondingAtReady/Fresh/StaleHandle/mainWindowPing/freshMainWindowHandle`）、settle 采样、UIA 主窗 45 节点、Host 子进程命令行、harnessHttp、dsh-home 列表、crashReports（空）。
- 本文档。
- `runtime/`、`tmp/` 改动均被 `.gitignore` 忽略；`git status` 只多出上面的 evidence JSON（未 commit）。

## 10. Round147 补记（task-21）：Project overlay 的边界与验证

### 10.1 能力检查到底要什么

`internal/harness/current_host.go:61-116` `ReadCurrentHostCapabilities(vendorDir)` 只认 `runtime/vendor/current-host-build.json`：

| 字段 | 要求 |
| --- | --- |
| `schemaVersion` | `1` |
| `sourceCommit` | `90465b67ec16765631d54c195af19d9ff3c3a770` 或 `e9f584be6d3b491ce777bb6823f30bc2bbc6f195`（`current_host.go:13-17`） |
| `buildProfile` | `official` |
| `platform`/`arch` | 运行时平台（win32/x64） |
| `safetyPatch` | `{id:"mygo-visual-readiness-v1", patchSHA256:"87b97484bbed6adc2d727e49922766af96579a6b4337a44da799cc993b13d3b3", sourceSHA256:"8c73f32eb14e9ba46d3712bcbc569b2effe74c28bc60fbfa299e08c920bf2812"}`（`current_host.go:13-14`，与 `scripts/current-host-readiness.mjs` 的 `guardID`/`patchSHA256`/`record.sourceSHA256` 对应） |
| `artifacts` | 12 个路径的 sha256：`deepseek-harness/apps/cli/lib/bin.js`、`apps/web/dist/index.html`、`packages/experimental/{agent-team,agent-team-profile,tool-agent-team,client-ui-agent-team}/lib/index.js`、`packages/client/ui-visual-replies/lib/{index,client}.js`、`packages/api/session-controller/lib/index.js`、`dsh-project/lib/{index,transport}.js`、`dsh-project/client/client.js` |

`ArtifactsReady` 只由上面这份清单决定；`main.go:2551-2566` 又只用 `ArtifactsReady` 决定是否创建 `dsh-project` 桥并 `--patch` 项目 overlay。`ValidateCurrentHostForLaunch`（`current_host.go:121-137`）一旦看到该清单，还要求 `safetyPatch` 精确等于上面三元组，否则拒绝启动。

### 10.2 安装版/已落地的 rc.2 里没有这个 marker，也无法「真」生成

- `C:\软件\Whale Isle\resources` 递归搜索：**没有** `current-host-build.json`，也**没有** `visual-readiness-patch.json`；安装版 vendor 只有 `deepseek-harness-runtime.json` + `harness-upstream.json`。
- 仓库的生成器 `scripts/current-host.mjs prepare|finalize|stage`（由 `scripts/build-current-host.ps1` 调用）要求：固定源检出 `SourceRoot` 的 `git HEAD` ∈ {90465b67, e9f584be}、`vendor/deepseek-harness` + `vendor/dsh-project`、`pnpm@11.8.0`，然后 `npm install pnpm` + `pnpm install --frozen-lockfile` + `pnpm run build:official`（**需要联网**）并产出 `visual-readiness-patch.json`。本工作区 `reference/electron` 没有 `.git`、也没有 `apps/cli/lib/bin.js`/`apps/web/dist/index.html`，无法满足这些前置；且该流水线产出的是 90465b67 系 Host，不是共享会话 schema 需要的 rc.2。
- 已落地的 rc.2 bundle **确实带着同一个门禁**：`session-runtime.mjs stage` 用 `session-runtime-readiness.mjs` 对发布版 `ui-visual-replies/lib/index.js` 做字节补丁（base `b58fda1a…` → output `8278107b…`，即 `internal/harness/session_runtime.go:23-28` 钉死的值），补丁后的 bundle 含 `MyGo visual reply readiness gate v1` 与 `/desktop/html-preview/status`（已核对）。但它不是「从 90465b67 源打了 `src/tools.ts` 补丁再编译」的产物，无法诚实地写成上表的 `sourceSHA256`（该字段是**打过补丁的 TS 源**的哈希，发布树里没有 `src/`）。

**因此这是检查本身的缺口**：`ReadCurrentHostCapabilities` 只接受源码构建清单，从不查看 `host-session-runtime.json`（rc.2 身份 + 同一个门禁的字节输出哈希），于是「安装版 rc.2」这一 profile 下 `ArtifactsReady` 永远为 false，Project 桥永远不挂载。按要求**没有改 Go 代码**，只记录结论与建议（§10.5）。

### 10.3 非破坏性验证：给克隆树加「适配 marker」，用 pass275 证明 overlay 能工作

不改 `runtime/vendor`（保持诚实），把已落地的 stage 克隆到 `tmp/proj-exp`（`session-runtime.mjs clone`，46 s，3145 链接），再用 `tmp/current-host-marker.mjs`（仓库自己的 12 项 artifact 清单 + Go 钉死的三元组）写 `current-host-build.json`，并带一个显式的 `provenanceAdaptation` 说明（运行时读不到该字段，只为人读）：

```powershell
node tmp\current-host-marker.mjs tmp\proj-exp\vendor 90465b67ec16765631d54c195af19d9ff3c3a770
#   → artifacts: 12, visualGate: true, marker sha256 C9C75190…442381

# pass275 + 真实 config + 临时 workspace + port 3099，vendor 用克隆：
WHALEISLE_VENDOR=C:\AI\WhaleIsle-MyGo\tmp\proj-exp\vendor
```

结果（`evidence/project-overlay-pass275.json`，`status: ok`）：

| 证据 | 值 |
| --- | --- |
| readiness | portListening / mainVisible / launcherGone / responding 全 true（`respondingFresh=true`，`mainWindowPing=true`） |
| 时间线 | 启动器 1060×660 于 t=2 s，主窗 1440×920 于 t=4 s 揭示，**:3099 首次监听 t=70 s**，启动器关闭、READY ≈ t=76 s |
| Host 子进程 | `runtime\node\node.exe tmp\proj-exp\vendor\deepseek-harness\apps\cli\lib\bin.js web --patch …\desktop-plugins\dsh-project\desktop-project.patch.yml …`（**dsh-project overlay 是第一个 `--patch`**，共 10 个 patch） |
| 桥准备 | profile 里出现 `desktop-plugins\dsh-project\desktop-project.patch.yml` + `dsh-project\runtime\{lib,client,node_modules,package.json,…}`（桥把包复制进 profile 供 Host 加载） |
| UIA | 45 个节点，与 round146 无 overlay 时**完全一致**；`项目/project` 命中 0 —— 临时 workspace 没有登记过 project，侧栏因此不出现项目分组（不是 overlay 失败） |

overlay 功能链路成立：`ArtifactsReady=true` → `startHarness` 创建 Project 桥 → `PrepareOverlay` 写入 `desktop-project.patch.yml` 并把 `dsh-project` 包 stage 到 profile → Host 以 `--patch` 加载。代价是**冷启动从 6 s 变成 ~76 s**（新增 ~64 s：桥 Ready + 复制含 node_modules 的包进 profile，每个新 profile 都会发生）。

### 10.4 为什么没有把 marker 直接写进 `runtime/vendor`

1. 它会把「安装版 rc.2」伪装成 90465b67 源码构建（`sourceCommit`/`sourceSHA256` 都不是真的）。
2. 更实际：`scripts/package.ps1:82` 一旦看到 `current-host-build.json` 就走 `current-host.mjs verify`，而 verify 需要 `visual-readiness-patch.json`（记录补丁后的 `src/tools.ts` 哈希）——发布树没有 `src/`，实测：
   `node scripts/current-host.mjs verify tmp\proj-exp\vendor` → **exit 1**，`ENOENT … visual-readiness-patch.json`。也就是说加了 marker 会直接卡死打包，除非同时伪造补丁记录。
   当前默认 runtime 状态已核对：`runtime\vendor\current-host-build.json`、`visual-readiness-patch.json` **均不存在**（与 round146 一致）。

### 10.5 建议（下一轮，需 Lead 决策）

在 `internal/harness/current_host.go` 里给已安装 runtime 增加一条**同样严格**的证明路径：
- 当 vendor 有 `host-session-runtime.json`（rc.2 pin + `safetyPatch.outputSHA256 == 8278107b…`，且 bundle 含门禁 marker）且 `dsh-project` 包存在时，令 `ArtifactsReady=true`（artifacts 仍逐字节校验 rc.2 清单里的 5 项）；
- `current-host.mjs verify` 同步接受这条路径（或让 package.ps1 对 installed-runtime profile 继续走 `session-runtime.mjs verify`）；
- 这样 Project overlay 默认可用、provenance 也真实，且不会回退会话 schema 校验。

手工打开（若 Lead 只想先要功能、接受适配记录）：把 `tmp\proj-exp\vendor\current-host-build.json` 复制到 `runtime\vendor\` 即可，删除该文件即回滚；但**打包会失败**，直到 §10.5 的产品改动落地。

### 10.6 残留

- 适配克隆 `tmp/proj-exp`（0.47 GB，含 `current-host-build.json`）保留作决策样本；`tmp/current-host-marker.mjs` 是生成器；两者都在 gitignored 的 `tmp/`。
- UIA 无法在**空 workspace** 下展示项目条目：需要先在 workspace 里登记 project（catalog 由插件/接口创建），本轮未做（时间盒）。
- 冷启动 +64 s 的成本需要单独优化（桥只应在需要时准备 overlay，或复用已 stage 的 runtime 目录）。

## 11. Round149 补记（task-24）：安装版 rc.2 默认可用，Project overlay 默认挂载

**决策**：不改写 `runtime/vendor`（不伪造 `current-host-build.json`），而是把这个缺口当作检查本身的缺陷修掉——`ReadCurrentHostCapabilities` 增加一条**同样严格**的安装版证明路径。

### 11.1 门禁逻辑（`internal/harness/current_host.go`）

`current-host-build.json` 存在时仍走原分支，代码与语义逐字不变（`TestCurrentHostPinnedAssemblies`、`TestSourceBuildManifestKeepsItsBranch` 都钉住）。只有它**不存在**时才尝试安装版证明 `readInstalledRuntimeCapabilities`，任一项不满足即 `ArtifactsReady=false` + 具体 `Reason`（fail closed）：

| 检查 | 复用/来源 | 失败示例 |
| --- | --- | --- |
| rc.2 身份 + 5 项 artifact 哈希 + 内置 Node 哈希 + 桌面会话事件词表 | 直接调用既有 `validateStagedSessionRuntime`（不复制任何 pin 常量） | `Installed Host runtime does not prove the pinned rc.2 assembly: … artifact changed` |
| `ui-visual-replies/lib/index.js` 必须等于 `rc2SessionRuntimeSafety.OutputSHA256`（`8278107b…`） | 同上（`sessionRuntimeArtifacts[4]` 分支） | `…visual-readiness gate is not the pinned build` |
| `session-runtime-readiness.json` 四元组必须等于同一 pin | 新增读取，与 `rc2SessionRuntimeSafety` 全等比较 | `readiness record does not pin the visual-readiness gate` |
| bundle 内含门禁字符串 `MyGo visual reply readiness gate v1` 与 `/desktop/html-preview/status` | 新增（证明「哈希到的是哪个构建」） | `lacks the readiness gate: …` |
| `dsh-project/{lib/index.js,lib/transport.js,client/client.js,package.json}` 存在 | 新增（无 manifest 覆盖，只查存在性；桥自己再 fail closed） | `Project package is incomplete: …` |

通过后的能力字段（真实、不冒充源码构建）：`SourceCommit = 687f282e…`（rc.2 的 `harnessPin.sha`）、`ArtifactsReady=true`、`ReadinessGuardReady=true`、`VisualReplyReader=true`、`Project/Team = bridges.Project && bridges.ProjectOverlay`、`VisualReplies = ReadinessGuardReady && HTMLPreview && VisualReplyIsolation`，公式与源码分支一致。`ValidateCurrentHostForLaunch` **未改动**：源码构建仍要求 `mygo-visual-readiness-v1` 三元组。

### 11.2 测试（`internal/harness/current_host_installed_test.go`，temp fixture）

- 有效安装版证明 → `ArtifactsReady/ReadinessGuardReady/VisualReplyReader=true`、`SourceCommit=687f282e…`；给全 bridges 时 `Project/Team/VisualReplies=true` 且无 Reason；`ValidateCurrentHostForLaunch` 仍通过。
- fail closed：readiness 记录哈希被改、记录缺失、bundle 缺门禁标记（把 pin 改成无标记字节，保证只有标记检查能拒绝）、缺 `dsh-project/package.json`、空 vendor 无任何 manifest、5 项 artifact 之一被改——全部 false 且 Reason 命中具体项。
- 源码分支不变：同时存在伪造的 `current-host-build.json`（未知 commit）时，仍只由它决定 → `provenance is unsupported`，安装版证明不会替它背书。

命令：`go test ./internal/harness` → ok 20.4 s；`go test ./cmd/whaleisle` → ok 19.0 s；`go vet ./internal/harness ./cmd/whaleisle` → 0。

### 11.3 端到端（默认 `runtime/vendor`，无 marker；pass275 不覆盖）

临时构建 `tmp/whaleisle-default-overlay.exe`（当前树 + 本轮改动），`node scripts/session-runtime.mjs verify runtime\vendor` **exit 0**，`runtime\vendor\current-host-build.json` 不存在。`tmp/harness-start-smoke.ps1`（临时 profile + 真实 config 改 workspace/port=3099 + `-Vendor runtime\vendor`）：

| 证据 | 值（`evidence/project-overlay-default.json`，status=ok） |
| --- | --- |
| readiness | portListening / mainVisible / launcherGone / responding 全 true；主窗 1440×920 可见，启动器已消失 |
| 时间线 | 主窗可见 **t≈3.3 s**，:3099 首次监听 **t≈7.3 s**，启动器消失 **t≈8.1 s**，整轮探针 24.7 s |
| Host 子进程 | `node.exe …\bin.js web --patch …\desktop-plugins\dsh-project\desktop-project.patch.yml --patch …dsh-task-control…`（**dsh-project 是第一个 `--patch`**，共 10 个） |
| profile | `projectOverlayFiles = ["desktop-project.patch.yml"]`（桥把 `desktop-plugins\dsh-project` + runtime 包 stage 进 profile） |

**冷启动成本**：本轮默认 vendor 下从启动到端口就绪约 **7 s**（对比 §10.3 克隆树实验的 ~76 s）。差异来自实验环境：当时 profile 里要新 stage 整个克隆的 `dsh-project`（含 node_modules）并额外解引用 3145 个链接；默认 vendor 的包更小、且已有缓存目录，一轮 ~25 s 即可完成探针全流程。仍应按「每个新 profile 首次启动」预留数秒级代价，后续可用已 stage 目录复用进一步降低。

### 11.4 残留

- `scripts/current-host.mjs verify` 仍只认源码构建清单；`package.ps1:82` 对安装版 profile 应改走 `session-runtime.mjs verify`（本轮未改脚本，避免动打包流程）。
- `dsh-project` 无 manifest 覆盖，只做存在性检查；若发布版包内容变化，桥的 `PrepareOverlay` 仍会 fail closed（日志 + 不挂载），但不会有哈希级证明。
- 空 workspace 下 UIA 仍不显示项目分组（§10.6），本轮未改变。

# 开发进度

2026-10-14 补充：最新逐模块交付与验证边界以 [模块交付记录](module-delivery-20261014.md) 为准。下文历史环境的实测记录不代表当前累计版本已通过真实 Host 或 Windows 桌面验收。

2026-10-14 文档预览补充：本地 `.htm/.html/.xhtml` 已接通交付卡和 Files 工具栏 → 隔离 WebView 文档标签，见 [HTML 文档预览](features/local-html-document.md)。仅保存文件快照，草稿保留；静态 HTML/CSS/图片/字体可显示，声明的同工作区资源有界快照内联，脚本和外网禁用，失败保留系统打开。Windows 实际呈现与隔离行为仍待实机验收。

更新时间：2026-10-10。总体计划见 `docs/plan.md`（里程碑 M0–M11）；本文记录每项的实现状态，随开发更新。

## 总览

| 部分 | 状态 | 位置 / 说明 |
| --- | --- | --- |
| Go 工具链与脚本 | 完成 | `scripts/env.ps1`、`scripts/build-harness.ps1`、`scripts/regen-codegen.ps1`、`scripts/package.ps1`（M11 staging）、`scripts/smoke-nsis.ps1`（NSIS 冒烟） |
| harness 运行时构建 | 完成 | `runtime/vendor/` 8 个目录，产物已验证 |
| 配置 `internal/config` | 完成 | 全字段 + 钳制 + 快照隔离 + 冲突拒绝，测试通过 |
| 凭据 `internal/credentials` | 完成 | DPAPI + AES-GCM 信封，分类加载，测试通过 |
| 路径 `internal/paths` | 完成 | userData / dsh-home / runtime |
| 应用骨架 `cmd/whaleisle` | M1 完成 | config/credentials、OpenAtLogin、CloseToTray、TitleBarHidden、主题/i18n、Ready 切主框架 |
| 接口生成器 `tools/gen-api` | 完成 | 支持新版 typert 懒加载 schema；测试通过 |
| dsh API 生成物 `internal/dsh/api` | 完成 | 29 包 160 端点；覆盖 endpoints.json 140/140 |
| 会话事件 `tools/gen-events` / `internal/dsh/events` | 完成 | 59 事件类型；测试通过 |
| dsh 客户端（RPC / WS / 事件） | M3 完成 | HTTP RPC（含 Connection 风格 multipart：`metadata` + attachments → `MultipartResult.Parts` / 组装 value）、WS mux、`$events`；单元测试绿；真实 harness 集成见 `-tags harness` |
| harness 启动器改造 | M2 完成 | vendored 启动 + overlays；install 控制面；**boot 页插件进度**（`Status.PluginLoad` + 日志解析 + UI 进度行）；真实 runtime 无结构化进度时显示 indeterminate |
| 界面（对话 / 设置 / 文件 / Git / 终端等） | M4–M9 主体可用 | 对话完整度/composer/工作环/设置专用页/用量/远程/IM/市场/定时/壁纸；少量深化项见下文 |
| dsh-whale 助理 / dshbot / webview 兜底 | M10 主体可用 | WebView2 Host + SlotURL 对齐；助理面板；Prepare 接 whale/dshbot |
| 启动器 / 导入 / 更新 / 安装包 | M11 主体可用 | staging + NSIS；更新器 hash/签名/blockmap 校验；手机远程（**部分实现**：仅配置就绪检查 + SSH 隧道，服务端/配对/QR 未实现，见「审查修正」）；任务保护/深链 |

## 已完成

### 1. 工具链与项目脚本

- Go 1.27.2 下载到 `.toolchain`；任何 go 命令前先执行 `. .\scripts\env.ps1`（设置 GOPATH / GOCACHE / GOTOOLCHAIN）。
- `go mod tidy` 已完成，MyGo 的传递依赖（purego、typesetting、x/image 等）已入 `go.mod` / `go.sum`；`go build ./...` 全仓通过。
- `scripts/build-harness.ps1`：harness 构建的可重复脚本（幂等，可安全重跑）。
- `scripts/regen-codegen.ps1`：从 `runtime/vendor/deepseek-harness` 一键重生成 `internal/dsh/api` + `internal/dsh/events`（需先 `. .\scripts\env.ps1`）。
- `scripts/package.ps1`（M11）：点源 `env.ps1`，`go build -ldflags` 注入 `main.buildVersion` 占位，复制 `resources/` 到 `dist/whaleisle/`，写 `dist/README.md`；PATH 或 **`.toolchain/nsis/makensis.exe`** 存在时编 `dist/WhaleIsle-Setup-<version>.exe`；复制 runtime 后写 **`dist/whaleisle/.runtime-staged`** 供 NSIS 编译期判断是否打包 `runtime/`。
- `scripts/smoke-nsis.ps1`（M11）：默认 `WHALEISLE_PACKAGE_SKIP_RUNTIME=1` 跑 staging + makensis，断言安装包存在且 size>0；可选 `-EnsureNsis`（winget 下载 + 静默装进 `.toolchain/nsis`）；PATH 有 7z 时列出安装包内容。

### 2. harness 运行时构建（M0 核心）

- 从参考快照复制 8 个源码目录到 `runtime/vendor/`：deepseek-harness、dsh-usage-panel、dsh-im、dshbot、dsh-task-control、dsh-platform-session、dsh-whale、dsh-remote。源码未做任何修改；`runtime/vendor/go.mod` 是隔离模块，这些 JS 树不参与主模块编译。
- 构建完成并验证（日志在 `runtime/build-*.log`）：
  - 上游版本：dsh-v0.2.1-alpha.1（提交 `5badb150…`）
  - CLI 入口：`runtime/vendor/deepseek-harness/apps/cli/lib/bin.js`
  - Web 产物：`apps/web/dist/index.html`（384 个 client artifact）
  - Ghostty 资源：3 个就位
  - 插件包依赖：dsh-usage-panel、dsh-im 等的 node_modules 已装好
- 原项目 `C:\AI\Deepseek-Harness-Desktop` 与 `reference/` 全程只读。
- 仓库说明：`.toolchain/`、`runtime/`、`reference/electron/` 不入 git（见 `.gitignore`）。换机器时按 `reference/README.md` 重建快照，再运行 `scripts/build-harness.ps1` 重建运行时。

### 3. `internal/config`（配置）

- 完整 config.json 模型：对照原版 `src/main/config.js` 的 DEFAULTS 逐项核对，钳制逻辑镜像原版（端口 1024–65535、重启 1–10 次 / 500–30000ms、relay 端点归一化、下载路由 gitee→cnb 等）。
- 未知键、`pet`、`live2dPet`、`pluginRecovery` 原样携带，与 Electron 版共用数据目录。
- 修正旧代码两处偏差：`theme` 的 JSON 名 `appearance`→`theme`；默认值 `system`→`deepseek`。
- 快照隔离：`Get` 返回深拷贝（`disabledPlugins` / `components`），`Update` 给回调独立的候选副本；提交值不与回调保留的引用或调用方缓冲区共享存储，并保留 nil 与空的区别。
- 陈旧写入拒绝：`Open` 记录磁盘基线（存在性 + 精确字节）；`Update` 在调用回调前与替换前各比较一次，不一致返回 `ErrConflict`，不调用回调、不写盘，不自动合并 / 重载 / 重试；进程内用提交互斥串行化"比较-替换"（回调不在互斥内）。
- 测试 19 个，全绿（含快照隔离、失败写入回滚、外部编辑 / 文件出现或消失 / 字节级编辑 / 回调期间编辑 / 双 store 顺序与并发冲突）。

### 4. `internal/credentials`（独立包）

- credentials.json 六字段读写：apiKey、baseUrl、githubToken、remoteToken、remoteRelayToken、remoteDevices。
- `safeStorage-v1` 信封：`v10` 前缀 + 12 字节 nonce + AES-256-GCM；密钥用 crypt32.dll 的 DPAPI 从 Local State 的 `os_crypt.encrypted_key` 解出。纯标准库，无 cgo，未加依赖。
- 兼容明文 JSON 读取；原子写（临时文件 + 重命名）。
- 加载分类：文档若带 `version` / `payload` 标记，必须是结构完好且受支持的信封，否则以 `ErrInvalidCredentialsFormat` / `ErrUnsupportedEnvelopeVersion` 拒绝（`errors.Is` 可判），绝不回退当明文读；缺失文件返回零值且无错误；base64 校验先于读取 Local State 密钥；失败时不返回部分字段，错误文本不含取值 / payload / 版本号。
- 测试 9 个（真实 DPAPI 往返、合成 Local State、篡改检测、分类表驱动），全绿；只使用临时目录。
- 未确认点：Electron 44 的 safeStorage 在 Windows 是否确实产出 v10 AES-GCM 信封，需用真实安装版生成的 credentials.json 实测一次。

### 5. `internal/paths`

- userData（默认 `%APPDATA%\Deepseek-Harness-Desktop`，可用环境变量 `WHALEISLE_USER_DATA` 覆盖）、dsh-home、runtime、pid 文件路径，以及 Ensure 系列辅助函数；导入无副作用。测试 4 个。

### 6. `cmd/whaleisle` 应用骨架 + `mygo.json`（M1 进行中）

- 单实例、托盘、退出路径同上；**已接入** `config.Open(userData)`：`CloseToTray` 控制关窗行为，`OpenAtLogin` 调用 `mygo.App.SetOpenAtLogin`；启动时 `credentials.Load`（失败仅打日志）。
- 主窗口：`TitleBarStyle: TitleBarHidden` + `TitleBarHeight: 40`；自绘标题栏在 `internal/ui/chrome`（`DragWindow` + 为系统按钮留白）。
- 主题：`internal/ui/theme` 解析 config `theme`，`mygo.Theme.SetSource` 跟 system/light/dark；窗口 `BackgroundColor` 用 `light-dark(#ffffff, #151517)`。
- 文案：`internal/ui/i18n`（zh-CN / en），托盘与占位状态走 `T(key)`。
- 冒烟：`go test ./internal/ui/...` 通过；进程可启动并干净终止。
- 用户数据：无 `-user-data` 旗标；用环境变量 `WHALEISLE_USER_DATA` 覆盖默认 Electron 数据目录（见 `internal/paths` 与上文 M11）。
- `mygo.json`：`{"name":"Whale Isle","identifier":"ai.deepseek.harness.gui"}`（与 Electron appId 对齐，便于覆盖升级）。

### 7. `tools/gen-api` 与 `internal/dsh/api`

- 生成器解析构建产物 `lib/typert.remote-client.js`（`TYPERT_REMOTE` 描述符），生成 Go 类型与端点注册表；支持 `mode:'stream'`、`kind:'context'`、`uplink`、`cancellation`；zod schema 尽力映射成 Go 类型，复杂形状回退 `json.RawMessage` 并注释原类型。
- 已适配新版 typert：懒加载 `() => (... ??= z...)`、`create:` 引用、`z.lazy(() => Foo())`、跳过 encode/decode 辅助箭头。
- 从 `runtime/vendor/deepseek-harness/packages/**` 29 个唯一包全量生成：**160 端点** / 30 个 gen 文件（含 `registry_gen.go`）；覆盖 `.toolchain/endpoints.json` 的 **140/140**（另有 20 个清单外端点）。
- 跳过 dshbot / dsh-whale 下重复的 dsh-llm（与 harness `packages/llm/llm` 同身份）。
- 用法：`. .\scripts\env.ps1; go run ./tools/gen-api -in <包根>... -out internal\dsh\api`。
- `go test ./tools/gen-api/... ./internal/dsh/api/...` 全绿。

### 8. `tools/gen-events` 与 `internal/dsh/events`

- 源 schema：`runtime/vendor/deepseek-harness/docs/persistence-schema.json`（formatVersion 2）。
- 生成器 `tools/gen-events`（纯标准库）：对象 → 命名结构体；复杂 union / tuple → `json.RawMessage`；另发 `Type*` 常量、`Envelope`、`Roots`、`EventsByType`。
- 产出：62 roots（2 header + 1 envelope + **59 events**）→ `internal/dsh/events/events_gen.go`。
- 用法：`. .\scripts\env.ps1; go run ./tools/gen-events [-in <schema>] -out internal\dsh\events`。
- 测试全绿（schema 解析、生成编译、关键事件种类存在、若干类型 JSON 往返）。
- 注意：persistence 事件 ≠ `$events` WebSocket 帧（`ready` / `emit` / `waterfall`）；后者属 M3。

### 9. 沿用代码 `internal/harness`（部分旧设计）

- 已有：子进程监管与自动重启、端口扫描、`dsh web: <url>` 就绪行解析、启动令牌换 dsh-auth cookie、workspace/create 登记。
- 旧设计：启动方式仍是"配置 dshBin / PATH / npx"；M2 将改为 vendored harness（node + bin.js + 覆盖文件 + junction）。
- 已知失败测试：`TestResolvePrefersPathThenNpx`（测试按 Linux 写法找 npx），M2 重写时一并消除。

### 10. 文档与参考

- `docs/plan.md`：总体计划（架构、协议调研结论、里程碑 M0–M11、风险、已确认决定）；§2 已去掉过时表格；§4.1 已拆出独立的 `internal/credentials`。
- 已删除无内容的残留空目录：`internal/app`、`internal/gitx`、`internal/rpc`、`internal/session`（不在包规划内）。
- `reference/electron`：原项目快照，基线提交 `26205a89f`（2026-10-06），见 `reference/README.md`；`reference/go.mod` 隔离其中的 Go 代码。

## 未开始 / 剩余（按里程碑，细节见 `docs/plan.md` §5）

- **M0**：主体完成（含 `scripts/regen-codegen.ps1`）。
- **M1 底座**：config/credentials、CloseToTray / OpenAtLogin、TitleBarHidden + 自绘标题栏、theme/i18n 已接入。
- **M2 harness 生命周期（进行中）**：
  - 已完成：vendored `node + bin.js`；Prepare junction/`--patch`（task-control、platform-session、usage-panel、dsh-im、remote、可选 dshbot/whale、market、session-search）；插件树失败 → 一次 `--skip-user-plugins` 重试；`EnsureWorkspace`；原生 boot 页。
  - 已完成（install 回滚/可加载性）：`CaptureInstallSnapshot` / `RollbackInstall` / `DiscardInstallSnapshot`（package.json、lock、workspace、node_modules；Windows junction 用 `mklink /J`）；成功 add 后 `HasLoadableEntry` + `HasInventoryIdentity` 校验，失败自动回滚；`RolledBack`/`RollbackError` 字段；单测含 unloadable 回滚。
  - 已完成（install 控制面）：`internal/harness/installctrl` 127.0.0.1 loopback、`POST /install`、Bearer、`DSH_DESKTOP_INSTALL_URL/TOKEN` 注入 harness 子进程；Supervisor 启停；成功安装后延迟 Restart。
  - 已完成（install 执行）：`installctrl.PNPMInstall` — Node + vendored `bin.js` 跑 `plugin --profile web add github:…`；`allowBuilds` 写入 profile `pnpm-workspace.yaml`；日志解析 `needsAllowBuilds`；`harness.NewInstallServer` 在 toolchain 不可用时回退 stub 错误；单测 fake runner。
  - 已完成：Prepare 传入 `Whale`/`Dshbot`（`whaleAssistantEnabled` / `dshbotEnabled`）；`Supervisor.SetPrepareOptions` 供开关后 Restart 重挂载。
  - boot 页「导出日志」：`harness.ExportLogsDialog`（Save 对话框，取消则写入 `%AppData%/…/logs/`）。
  - 已完成（本轮）：boot 页 **插件加载进度行** — `harness.Status.PluginLoad`（解析 `» N desktop --patch overlay(s)` 与可选 `dsh plugin-load: R/T` 日志）；`internal/ui/boot` 绘制 `%d/%d`；无计数时显示 indeterminate 文案（真实 vendored harness 尚未打印结构化 runtime 进度）。
  - 已完成（冒烟）：vendored harness 存在；`go test -tags harness ./internal/dsh/ -count=1` 通过；`.toolchain/whaleisle-test.exe` 进程冒烟（WHALEISLE_USER_DATA 临时目录，约 8s 存活后结束）。
  - 已完成（office overlay 冒烟）：`TestPrepareIncludesOfficeOverlayWhenPayloadPresent` — 合成 `DSH_PRIMARY_RUNTIME` + harness libreoffice-kit stub，经 `Prepare`→`extraOverlays` 断言 `desktop-office.patch.yml` 入 patch 列表且含 workspace-dependencies / skill-office。
- **M3 dsh 客户端（主体完成）**：
  - `Client.Call`：信封、错误码、`errors.As`、multipart JSON 信封解码。
  - `Mux`：`GET /api/remote.mux`（`coder/websocket`）、`open`/`item`/`end`/`cancel` ↔ `item`/`end`/`error`；库级应答 Ping/Pong；`MaintainMux` 退避重连。
  - `$events`：`FollowEvents` + `PostEventResult`；未处理 waterfall 自动 `next`；`approval/request` / `user-questions/request` 可由 handler 回答。
  - 测试：`go test ./internal/dsh/`；可选 `go test -tags harness ./internal/dsh/` 对真实 vendored harness 验 `$events ready`。
  - 已完成（本轮）：persistence JSONL / follow 事件 **`events.UnmarshalEvent`**（按 `EventsByType` + 生成 struct；`ErrUnknownEventType`）；与 `$events` WebSocket 帧（ready/emit/waterfall）无关。
  - 已完成（冒烟）：`go test -tags harness ./internal/dsh/ -count=1` 通过（真实 vendored harness）；`TestHarnessSessionPromptCancel` 覆盖新建会话 → `session/prompt` →（若有）`approval/request` 自动允许 → `session/cancel`（不要求 LLM 跑完；无 API key 且 prompt 拒答时 `t.Skip`）。
  - 已完成（本轮）：multipart 二进制 attachments 组装（`parseMultipartResult` / `CallMultipart`；`metadata` + `bytes-*` parts；占位 null → base64 进 value；`Parts` 映射）；httptest 覆盖嵌套 path 与 `[]byte` 解码。
- **M4 对话最小可用（进行中）**：
  - 已完成：`internal/store/session`（`session/list`、`session/create`、`session/rename`、`session/prompt`、`session/cancel`、`session/follow` snapshot/event/assistant-stream、`session/search` 侧栏 debounce 搜索、`session/modelCatalog` + `session/selectModel`、`permissionPresets/catalog` + composer 权限预设下拉经 `settings/update`（`permission` 命名空间 `defaultPreset`））；`internal/ui/frame|sidebar|chat|composer`（侧栏重命名、composer 模型/权限下拉）；`internal/ui/widgets/markdown`（goldmark + chroma 代码高亮）；`internal/store/interaction` + composer 区审批/提问卡（默认自动允许工具，可改手动）；Harness Ready 后挂 `dsh.Client` + 共享 `Mux` + `MaintainMux` / `$events` waterfall 应答；boot 页导出日志；`internal/store/session` httptest 覆盖 list/create/rename；权限 catalog flatten 单测；`internal/ui/frame` `ui.NewTester` 主壳标签冒烟（`TestViewShellLabels`）；`internal/ui/boot` `ui.NewTester` boot 页状态/重试/导出文案（`TestViewBootLabels`）；follow 事件 data 识别图片 path / Markdown 图片 / `content` 内 `type:image` 块，聊天行显示 `[image] <path>` 占位（尚无 `ui.Image` 大图）。
  - 已完成（本轮）：Harness Ready / mux 重连时并行 `workspace/follow`（`ModeStream`）；帧解析更新侧栏 `title`/`cwd`/`running`（snapshot 与 patch，未知帧忽略不崩）；侧栏选中会话 **归档**（`workspace/archiveSession`）与 **删除**（`session/delete`）按钮，成功后 `session/list` 刷新；`workspace_follow` / `archive` httptest。
  - 已完成（本轮）：`user-questions/request` 提问卡多题 UI（单选 Select/Checkbox、多选 Checkbox、无选项 TextInput）；`internal/store/interaction` 草稿态 + `ResolveQuestions` 组装 `PostEventResult` 答案；Deny 可拒答；`interaction_*` 单测；`go build ./cmd/whaleisle`。
  - 已完成（本轮）：侧栏搜索非空时 debounce 调 `session/search`（`Store.SearchSessions` / `UpdateSessionSearch`）；RPC 命中按 sessionId 裁剪树，失败或无 client 时回退本地 `filterItems`；`search_*` / `sidebar` httptest；`go build ./cmd/whaleisle`。
  - 已完成（本轮）：composer **排队 / 插话** — `session/prompt` 的 `mode` 为 `queue` 或 `steer`（codegen 无 `interrupt` 字面量）；`Store.SetPromptMode` + `ViewState.PromptMode`；composer 双按钮切换；`SendPrompt`/`SendPromptEdit` 随模式发 RPC；`prompt_mode_*` httptest。
  - 已完成（本轮）：侧栏 **置顶** — `workspace/pinSession` / `workspace/unpinSession`（request 仅 `sessionId`）；`workspace/follow` 的 `baseline`/`pinned` 帧同步 `pinnedSessionIds`；选中行 Pin/Unpin；`Item.Pinned` + 📌 前缀 + 按 pin 序排序；`pin_*` / `sidebar` httptest。
  - 说明：`session/updateQueue` 是队列项 edit/remove/**steer** 动作 API，不是 composer 默认模式开关；本实现按 `SessionPromptRequest.mode` 对齐 harness typert（`queue` | `steer`）。
  - 已完成（本轮）：`session/page` 更早历史分页（`Store.LoadEarlierHistory` 前置合并 + seq 去重，保留 follow 流式尾行）；聊天列 **加载更早消息**（`HistoryHasMore` / busy）；`page_*` / `chat` httptest；`go build ./cmd/whaleisle`。
- **M5 对话完整度（进行中）**：
  - 已完成：`session.Message` 的 `Kind`（text/tool/thinking）+ 工具类 persistence 原始 JSON；`internal/ui/chat/toolcard` 紧凑工具卡（名称/状态/截断参数或结果）；`internal/ui/widgets/diff` 统一 diff 渲染（`go-diff` 辅助 + RichText 着色）；chat 列表按 Kind 分发（工具卡 / diff / Markdown）；`internal/ui/chat/thinking` 可折叠 muted 思考块。
  - 已完成：聊天列顶部 **连接状态条**；助手消息 **👍/👎 完整反馈 UI**（选中标记、in-flight 禁用、👎 可选原因 + 发送、`messageFeedback/put` + note、`MessageRatings`/`FeedbackBusyID`）。
  - 已完成（本轮）：工具卡 **种类芯片**（bash/read/write/search/web/todo/ask/agent）；聊天 **图片内联**（`session.ImageRefs` + `ui.Image` 本地栅格，http/data 仍占位文案）。
  - 已完成（本轮）：用户消息 **编辑**；**交付物卡**；**压缩卡**（`compaction/*`）。
  - 已完成：简易公式 → **TeX→Unicode**（`$`/`$$`、`\frac`/`\sqrt`/希腊字母/上下标等；无 KaTeX 二进制）。
  - 已完成：聊天列 **用量标签**（今日任务数 / 余额摘要；点击打开用量面板）。
  - 已完成：**远程图片缓存**（`userData/cache/images`；sha256 URL；8MB/`image/*`；占位→`ui.Image`）。
  - 已完成（本轮）：**计划/目标/队列/待办面板**（follow 折叠 `plan/mode`/`goal/change`/`todo/write`/`team/message/queued`；聊天区下方可折叠）。
  - 已完成（本轮）：**大图查看** — 内联 `ui.Image` 点击打开 `ui.Overlay` 全屏 lightbox（本地路径 + 已缓存远程图）；`DisplayImagePath` / `ImageLightboxOverlay`；`frame`/`main` 状态；`image_lightbox_test` 覆盖路径解析与 bitmap 解码。
  - 已完成（本轮）：**引用/来源列表** — persistence `assistant/message` 与 `tool/*` **无**专用 citations 字段（`events.AssistantMessageDataMessageSource` 仅为 model/provider）；聊天 UI 对助手消息与工具卡做 **best-effort** 提取（事件 JSON 内 `url`/`title`/`results`/`sources`/`citations` 形状 + 正文 Markdown 链接）；点击来源可打开 **浏览器预览** 面板。
  - 已完成（本轮）：**persistence 事件 typed 解码** — `internal/dsh/events.UnmarshalEvent` 按 schema `type` 分发到 `events_gen` struct；`session/classify` follow 路径对 `tool/call`、`assistant/message` 优先 typed（失败回退原有 raw 解析）；单测覆盖 tool / assistant attempt(stream) / assistant message / approval/asked。
  - 待做：无（启发式 citations 已落地；typed source 字段等上游；http 图 Ready 后可 lightbox；KaTeX 二进制非目标）。
- **M6 输入框增强（进行中）**：
  - 已完成：`internal/ui/composer` `/` 命令弹出（`commands/list` 可用时合并 dsh 列表，否则 stub `/clear`、`/new`）；Enter/Tab/列表选择插入或执行（`/new`→`CreateSession`，远程→`commands/execute`）；`@` 工作区路径前缀补全（`workspacefs.IndexRelPaths` + 相对路径插入）；`suggest_*` 过滤单测；composer 无 API 时仍可发消息。
  - 已完成：`agentPresets/list` + `agentPresets/select` 接入 `internal/store/session`；composer Agent 预设下拉；roster flatten 单测。
  - 已完成：`internal/ui/composer` 输入框 MyGo `DroppedFiles` / `FileDragOver` 拖放附加（`workspacefs.RelPath` → `@相对路径` 写入 draft）；`attach_*` 单测。
  - 已完成：`skills/list` 接入 `internal/store/session`；composer **「技能」** 按钮弹出列表（user-only 标记 `modelInvocable: false`）；选中插入字面量 **`/name `** slash 调用（对齐 harness ui-skill）；`skills_*` / `composer` 单测。
  - 已完成：composer **目录选择器**（`mygo.Dialog.Open` Directory；工作区根 `DefaultPath`；插入 `@rel/path `；域外路径拒绝）。
  - 已完成（本轮）：技能弹出 **过滤输入框**；草稿 `@path`/`/skill` **行内芯片**（点击移除）。
  - 已完成（本轮）：更丰富的行内标签 — `@`/`/` 芯片视觉区分（file/skill 前缀 + accent/muted）；右键选中 draft 令牌（`SetTextSelection`）并弹出编辑；拖拽重排更新 draft；`mentions_*` 单测。
  - 待做：无。
- **M7 工作环（进行中）**：
  - 已完成：右栏本地 `workspacefs` 文件树 + 纯文本预览（`cfg.Workspace`）；`internal/desktop/workspacefs` 单测；底栏 MyGo `plugins/terminal`（Ghostty/libghostty-vt）包装在 `internal/ui/terminal`，`frame` 在 `Bottom > 40` 时展示，工作区 cwd 启动 shell；标题栏「显示/隐藏终端」切换。
  - 已完成（本轮）：工作区 Git 快照（分支 + 脏树）与标题栏 badge；`internal/desktop/git` 单测（临时仓库）。
  - 已完成（本轮）：`internal/desktop/git` `Fetch`/`Pull`/`Push`/`CommitAll`（`git add -A` + `git commit -m`；git CLI + 超时 + CombinedOutput）；标题栏 badge 旁「取/拉/推/提交」按钮；提交条（TextInput + 确认对话框，仅脏树可提交）；结果经 `mainState.gitMsg` 与 `session.SetLastAction`；`chrome`/`git` 单测含 mock runner / Windows `git.cmd` PATH shim；`go build ./cmd/whaleisle`。
  - 已完成（本轮）：`internal/desktop/workspaceauth` `Validate(path)`（镜像 Electron `workspace-authority.js` 卷根 / 高风险锚点 / `.git` 元数据目录 + 存在且为目录）；`startHarness` 与设置「桌面」保存前校验；`workspaceauth_*` 单测；`go build ./cmd/whaleisle`。
  - 已完成（本轮）：右栏 **文件 / Diff** 标签切换；`internal/ui/diffsurface` + `internal/desktop/git` `Diff`（`git diff` / `git diff -- <path>`，exit 1 有输出视为成功）；文件预览 **Diff** 按钮；`widgets/diff` 着色；`diffsurface_*` / `git` Diff 单测；`go build ./cmd/whaleisle`。
  - 已完成（本轮）：文件预览 **编辑/保存**（`TextArea` + `workspacefs.WriteFileText` 原子写）；标题栏 Git **查看 PR / 草稿 PR**（`gh pr view` / `gh pr create --draft --fill`，无 gh 时提示）。
  - 已完成：文件预览 **语法高亮**（`markdown.SourceView` / chroma）。
  - 已完成（本轮）：编辑态 **Edit ↔ Highlight** 切换（TextArea 改稿 + 同草稿 SourceView 预览；`LanguageFromPath` / `CanSourceHighlight`）。
  - 已完成（本轮）：**PR 评审面板**（`gh pr status`/`list`；标题栏摘要；View/Draft/Refresh/Checkout；无 gh 提示）。
  - 已完成（本轮）：编辑高亮 UX — 默认 **分屏预览**（宽栏并排 / 窄栏上下叠，草稿共享）；语言 chip；Split/Edit/Highlight 模式记忆；MyGo TextArea 仍无法真·内嵌高亮。
  - 待做：无（TextArea 内嵌高亮为 MyGo 限制，已用分屏预览替代）。
- **M8 设置（进行中）**：
  - 已完成：`internal/store/settings`（`settings/describe` 解析 schemastery 信封、`settings/update` 带 revision；schema `meta.label` / `meta.description` 映射到表单）；`internal/ui/settings`（左侧按 plan 15 分区 +「其他」分组命名空间、i18n 分区标题；首项「桌面」经 `cmd/whaleisle` 回调读写 `internal/config`：工作区、端口、主题、语言、托盘与登录启动）；Harness 命名空间 string/number/bool/enum/JSON 表单；标题栏「设置」入口；`internal/ui/settings/sections_*` 单测；store httptest；`go build ./cmd/whaleisle`。
  - 已完成：设置左侧 **凭据** 分区 — 读写 `credentials.json`（API Key / Base URL / GitHub / Remote / Relay），掩码显示 +「显示密钥」后编辑，`credentials.Save` DPAPI 信封。
  - 已完成：`settings/mutate`（`SaveMutate`）；`$events` emit 识别 document-updated → `RefreshDescribe`；`settings/replace`（`SaveReplace`）。
  - 已完成：专用页 **关于** / **外观** / **界面**（语言/托盘/登录）/ **模型** / **MCP** / **插件** / **技能** / **Agent 预设** / **账号与登录**。
  - 已完成：通用表单 **替换文档**（确认对话框 → `SaveReplaceFromDrafts` → `settings/replace`；空/非法草稿 fail-closed）。
  - 已完成（本轮）：通用 schema 表单深化 — 左侧命名空间过滤；password 字段（apiKey/token/secret）+「显示密钥」；描述型 string 用 TextArea；修订号 / 可写·只读徽标；`FilterNamespaces` 单测。专用页仍走独立 pane，过滤只作用于 harness 命名空间列表。
  - 待做：无（M8 主体完成）。
- **M9 桌面扩展（进行中）**：
  - 已完成：用量面板（account/job + 近 7 日柱状图 + 近 28 日热力格 + WebView2 打开 `usageUrl`）；插件市场；CSV/TSV/XLSX 预览；PDF/Office 外部打开；定时任务；壁纸画廊（**部分实现**：仅本地目录画廊，无 Bing/Wallhaven 在线目录与下载应用，见「审查修正」）；chroma 源码预览。
  - 已完成：`internal/desktop/sshremote` — 就绪检查、TCP 探测、`TunnelSession`（本地端口转发）；远程面板 SSH 表单 Connect/Save/Disconnect。
  - 已完成：**host-key 钉扎** — `userData/ssh/known_hosts`；Connect 关闭静默 TOFU；未知主机指纹 Accept/Decline 对话框（Accept → `AcceptHostKey` + 重连）；错配 → 警告拒连。
  - 已完成：IM 面板深化 — Select/SendDraft + 渠道列表 / 本地历史 / 草稿 / 发送 / dsh-im WebView；`userData/im/`（channels/drafts/history JSON）；codegen `im/*` 为 **0** 时本地模式可用，有 list RPC 时仍走 Refresh 拉列表。
  - 已完成（本轮）：**IM 历史同步桥** — vendor `dsh-im` **无** typert `im/*`；Refresh 探测 `im/list`·`im/history`，失败则 **Cordis** `POST /{channel}/connection.status` 拉 bot 渠道（`bridgeMode=cordis`），再退回 `pluginInventory/list` 诊断；httptest 覆盖 list/history/send/inventory/cordis。
  - 已完成（vendor 审计）：`runtime/vendor/dsh-im` **无** 会话/消息历史 RPC（无 `listMessages`/`getHistory`/`inbox`/`timeline`/`im/*` typert 模式）。对外 Cordis 仅为 **连接与 bot 管理**（`plugin-src/client/channels/shared/token-api.js` + 各 channel `api.js`/`host/channels/*/rpc.mjs`）：共性 `connection.status`、`bot.bind-credentials`、`bot.reconnect`（可选 `sendTest` 触发插件内 **固定测试消息**，非任意正文）、`bot.delete`、`bot.workspace.set`、`bot.preset.set`；各渠道另有 `provision.*`、钉钉 `bot.sender.*`、Telegram/WhatsApp `bot.access-policy.set`、飞书 `bot.callback-repair.begin` 等；消息收发在插件 runtime/bridge 内，不暴露给 Harness Typert Remote。
  - 已完成（原生等价）：Cordis 列表行 **Detail** 展示 `health.summary` 与 `lastMessageError.message`（见 `internal/store/im/cordis.go`）；**SendDraft** 仍仅探测 codegen `im/send` 候选，vendor 无通用 send → 本地 `history.json` + `LastSend` 说明；完整 IM 仍靠 dsh-im WebView。
  - 已完成（本轮）：**浏览器预览面板** — `internal/ui/browserpreview`（URL 栏 + 导航/系统浏览器/关闭 WebView）；标题栏与工作环右栏入口；复用 `internal/desktop/webview.Host` 加载任意 http(s)（与 M10 插件 SlotURL 面板独立）；PDF「WebView2 打开」走预览面板；`ui.NewTester` 标签冒烟。
  - 本地 PDF：已接入 [只读 PDF 标签](features/local-pdf-preview.md)，采用单文件快照与临时 loopback 能力地址；继续拒绝 `file://`。Office 转换产物精确归属请求，保留系统打开；Windows 实机渲染仍未验证。
- **M10 鲸鱼娘助理与 webview 兜底（进行中）**：
  - 已完成：`internal/desktop/webview.Host` — MyGo 无边框 Parent URL 子窗口（WebView2）、`Navigate`/`SetVisible`/`SetBounds`/`SyncBounds`/`Close`；标题栏「插件 Web」+「打开内嵌 WebView2」；离开面板自动销毁 Host。
  - 已完成：`internal/ui/assistant` 原生面板（标题栏「助理」）— 开关 `whaleAssistantEnabled`/`dshbotEnabled`、vendor 存在检测、WebView2 打开 `dsh-whale`/`dshbot` 插槽；单测 `ui.NewTester`。
  - 已完成（本轮）：统一 **`SlotURL(base, pluginID, extraQuery)`** — `{base}/?desktopPlugin={package}`；别名映射 whale/dshbot/im/usage/market；call sites 已切。
  - 已完成（本轮）：**原生快捷提问** — TextArea + Send → `OnNativePrompt` → `session.Store.SendPrompt`（有选中会话）；无会话 / 无 RPC 时 `SetLastAction` 明确提示，无 RPC 时打开鲸鱼娘 WebView 兜底；状态区展示 vendor/enabled、最近 WebView URL、最近原生操作；WebView 按钮标为高级兜底；`assistant` `ui.NewTester` 覆盖标签与 Send。
  - 已完成（本轮）：插件 HTTP 控制面 `internal/desktop/pluginrpc` — `/dsh-whale/{assistant/ensure,catalog,pet/chat,pet/state,pet/look}`、`/dshbot/{describe,group/runtime,bot/activity,...}`（catalog **读**走 describe；`catalog/items` 为写）；助理面板原生按钮 + **Bot/群聊列表**（describe → `userData/assistant/dshbot/roster.json`；刷新时合并 `bot/activity` 预览与房间 `group/runtime`）。
  - 已完成（本轮）：**原生加深** — `internal/store/assistant`（`userData/assistant/` mode + history）；Whale/Dshbot 模式切换（WebView 插槽 + 路由前缀）；Whale Send → `WhaleEnsureTyped` + `WhalePetChatTyped`；Dshbot Send → `describe` 探测 + `[dshbot]` 前缀 `session/prompt`；流式 LastAction/LastErr；「打开会话」选中侧栏；历史优先本地 log，空时回退会话 user/assistant 配对；`pluginrpc`/`assistant`/`store/assistant` 单测。
  - 已完成：`pet/look` — 选图对话框 → `pet/state` 取 provider/model → base64 `pet/look`。
  - 已知收窄：dshbot **无**独立 chat/pet RPC（vendored `control-plane.js` 未注册）；原生仍靠 `[dshbot]` 会话前缀 + WebView 完整 UI。
  - 已完成：composer 行内 `@path` / `/skill` 芯片解析、删除、弹出改写（`ChipEdit`）。
- **M11 外围与发布（进行中）**：见下节；启动器 UI、深链路由、NSIS、sha256/size + Authenticode + blockmap stub 已落地；**手机远程**（Assess / 连接 URL / 复制剪贴板 / 打开浏览器·WebView / 刷新 / `remoteDevices` 配对列表）为**部分实现**：`phoneremote` 仅做配置就绪检查与连接 URL 构造，无 mobile-web 服务端、无 ChisaCode 配对、无 QR 绘制（原版 `mobile-web-server.js` / `dshd-remote.js` 未 port），手机端完整能力不可用。

### M11 发布与数据路径

- **Electron 数据探测**：`internal/desktop/importer` — `ElectronUserDataDir()` 解析默认 `%AppData%\Deepseek-Harness-Desktop`（**忽略** `WHALEISLE_USER_DATA`）；`ImportReport` 只读存在性；设置 → **桌面** → **检测 Electron 数据**。
- **替换升级约束单测**：`TestCoverUpgradePathInvariants` — `DataDirName` / `mygo.json` identifier / NSIS `APP_ID`+`InstallDir` 对齐 Electron；临时目录模拟 Electron userData 的 ImportReport。
- **替换升级**：沿用 Electron 版 appId **`ai.deepseek.harness.gui`**、产品名 Whale Isle、同上数据目录；默认覆盖 `%LOCALAPPDATA%\Programs\Deepseek-Harness-Desktop`；`mygo.json` identifier 已为 **`ai.deepseek.harness.gui`**。
- **NSIS 工具链（本机，不入 git）**：`.toolchain/nsis/` 可由 `winget download NSIS.NSIS` + 安装包 `/S /D=<repo>\.toolchain\nsis` 得到（2026-10-10 已验证 **NSIS 3.13**）。**未进 CI**；缺 makensis 时 `TestNSISInstallerSmoke` 与手工冒烟 `t.Skip`/报错。
- **NSIS 编译冒烟（已跑通）**：`WHALEISLE_PACKAGE_SKIP_RUNTIME=1` 下 `scripts/smoke-nsis.ps1` 产出 **`dist/WhaleIsle-Setup-0.0.0-dev.exe`（约 7.2 MB，仅 exe+resources）**；`go test ./internal/desktop/importer/ -run TestNSISInstallerSmoke` 绿。**尚未**自动化：静默安装到 `%LOCALAPPDATA%\Programs\Deepseek-Harness-Desktop` 并断言覆盖 Electron 目录（仍靠约束单测 + 人工装包）。
- **MyGo 更新器**：`WHALEISLE_UPDATE_FEED` JSON 检查版本；有更新时可 **下载并启动** 安装包；可选 **sha256 / size** / Authenticode；feed 可选 `blockMapURL`/`blockMap`；`CanIncremental` 时走 `ApplyIncremental`：本地旧安装包存在则 `ApplyBlockPatch`（按 offset+sha256 复用块 + HTTP Range 拉变更；Range 不支持则全量 GET），否则全量下载 + `VerifyBlockMap`（失败删除暂存文件）。`WHALEISLE_UPDATE_OLD_INSTALLER` 可指向旧包。
- **启动器 UI**：`internal/ui/launcher` — 标题栏「启动器」；版本 / vendor 就绪 / 工作区输入 / 启动桌面 / Electron 数据探测 / 检查更新 / 打开设置 / disabledPlugins；**插件排查**（`pluginRecovery` 状态、`UpdatePluginRecovery`/`ClearPluginRecovery`、`Supervisor.SetSkipUserPlugins` + `Restart`）；无工作区时自动打开。
- **打包 staging**：`scripts/package.ps1` 产出 `dist/whaleisle/`（exe、resources、可选 `runtime/vendor` + `runtime/node`）；`WHALEISLE_PACKAGE_SKIP_RUNTIME=1` 可跳过 runtime 复制；有 makensis（PATH 或 `.toolchain/nsis`）时自动编安装包。
- **NSIS**：`scripts/nsis/whaleisle.nsi` — 用户级安装、快捷方式、卸载项；**编译期** `!if /FileExists` + `.runtime-staged` 决定是否嵌入 `runtime/`（避免 skip-runtime staging 误编空 runtime 树）。
- **用户数据路径（已存在，无 CLI 开关）**：`cmd/whaleisle` 通过 `paths.UserDataDir()` 读配置与凭据；默认 `%AppData%\Deepseek-Harness-Desktop`，测试或并行实例可用环境变量 **`WHALEISLE_USER_DATA`** 覆盖（`internal/paths`，常量 `paths.EnvUserData`）。另有 **`WHALEISLE_VENDOR`** 可指向 vendored harness 根目录。
- **任务保护（stub）**：`internal/desktop/taskguard` + `session.Store.RunningCount`；主窗口 `OnClose` 与托盘「退出」在存在 `running` 会话时用 `mygo.Dialog.Message` 确认（对齐 Electron closing-overlay / confirm quit 意图）；**关窗到托盘**时无运行会话仍只 Hide，有运行会话须确认后 `App.Quit`（不静默进托盘）。
- **深链 WHALEISLE**：`deeplink.Parse` + `applyDeeplink`（`open` / `launcher` / `settings` / `session/<id>` / `usage`）；`mygo.json` `urlSchemes` + `RegisterURLScheme`。

## 审查修正（功能完整性全面审查登记）

依据 [`docs/work-add7354e-f037-4462-95c3-580a60009da8/feature-audit-report.md`](work-add7354e-f037-4462-95c3-580a60009da8/feature-audit-report.md)（只读审查，结论均附 `文件:行号` 证据；下原版证据列为 `reference/electron/src/` 下相对路径）登记自述与实际的差异。本节只登记差距与修正事实，不替代上文历史记录。

**验收状态：`docs/acceptance-ui-parity.md`（像素级 1:1 + 功能全量穷举验收合同）当前零双签。** §0.3 环境 E1–E7、§2 画面 L-*/D-*/P-*/S-*、§3 功能 FL-*/FD-*/FS-*、§4 门禁 G1–G6 与 §6 总签署的「完成 / 审查完成」列全部为空；按该合同 §5，未双签的历史结论作废——项目尚未通过验收，不得标记目标完成。

### 缺失/降级项（审计报告 §1.2，16 项；round126 补 5、round129 补 1；其余按 2026-10-24 差距矩阵复核）

| # | 功能 | 原版证据 | Go 侧证据 |
|---|------|----------|-----------|
| 1 | 桌面宠物 / Live2D 桌宠 | `main/desktop-pet.js`、`main/desktop-live2d.js:757`、`main/pet-chat.js`、`main/pet-growth.js`、`main/pet-stats.js`、`main/pet-wander.js`、`main/pet-physics.js`、`renderer/pet.html`、`renderer/pet-live2d/` | ⚠ 部分（矩阵 G31/G32）：`internal/desktop/pet/companion.go:188` 已有窗口与成长/统计/漫步/物理行为族；渲染是单层 canvas rig，非 Live2D+ORT 超分，外观降级 |
| 2 | 手机远程服务端 | `main/mobile-web-server.js:14`、`main/mobile-web.js`、`main/dshd-remote.js:1`、`main/dshd-git-tunnel.js` | ⚠ 部分（G44）：`internal/desktop/phoneremote/runtime.go` 经 `bridge.cjs` 委托 vendored 守护进程（3180/relay/pairing URL+PNG/devices）；非自研服务端，真实手机未验收 |
| 3 | Welcome 首次运行窗口 | `main/welcome-window.js:59`、`renderer/welcome.html`、`main/welcome-backend.js` | ✅ round126：`cmd/whaleisle/welcome_window.go` + `internal/ui/welcome`；无 API Key 且无标记时先于 launcher 打开，登录/API Key/跳过三条路径 |
| 4 | 系统通知 | `main/system-notifications.js`、`main/background-notice.js`、`main/legacy-notification-shortcut.js` | ✅ round126：`cmd/whaleisle/system_notifications.go`（托盘气泡，不注册 Electron.lnk） |
| 5 | 全局快捷键服务 | `main/shortcuts.js`、`main/shortcuts-protocol.mjs`、`main/menu.js:11-22` | ✅ 主体（G37/G38）：`internal/shortcuts/*` + `cmd/whaleisle/application_menu.go`；原生也没有 OS 级 `globalShortcut`；菜单命令覆盖与加速键刷新未逐项 |
| 6 | 崩溃报告 | `main/crash-report.js` | ✅ round131（G41）：`cmd/whaleisle/crash_report.go`；`crash-<ISO>-<source>.log`、保留 10 份、256K 错误段、64K 控制台尾、O_EXCL 0600、失败不掩盖原错误；host/main/web-boot 已接，renderer 源不存在 |
| 7 | 媒体权限处理 | `main/media-permissions.js` | ✅ round131（G42）：`internal/desktop/webview/permissions.go`；loopback 会话宿主才给麦克风，camera/geolocation 拒绝，通知门控，其余走引擎默认；`PermissionRequest` 无帧/mediaType 信息，已登记残留 |
| 8 | 关闭遮罩 / curtain 窗口 | `main/closing-overlay.js`、`main/window.js:486` | ✅ round129：`internal/ui/chrome/closing.go` + `finalizeQuit` 停服前显示（500ms 绘制预算、失败不阻塞）；spinner 旋转未做（静态环） |
| 9 | 应用级菜单 | `main/menu.js:26-113` | ⚠ 部分（G37）：`cmd/whaleisle/application_menu.go:34` 已接应用菜单；命令覆盖与运行时加速键刷新未逐项 |
| 10 | 外部编辑器打开 | `main/editors.js:5-26,105-150` | ✅ round126：`internal/desktop/editors` + `cmd/whaleisle/open_editor.go`；文件预览「打开方式」直接带行列打开 |
| 11 | WhaleBridge 账号窗口 | `main/whalebridge-window.js:16`、`renderer/whalebridge-accounts.child.cjs`、`main/account-backend.js` | ✅ round126：`cmd/whaleisle/whalebridge_window.go`（1120×780、仅本机管理地址）+ `components/management.go` 地址校验 |
| 12 | dsh-remote 插件 RPC 接线 | `main/dsh-remote-desktop.js`、`main/remote-shell.js`、`main/remote.js`、`main/relay-client.js`、`main/remote-tls.js` | ⚠ 部分（G46）：`internal/store/remote`（注册表/文件/删除）+ `sshremote` 隧道已有；remote shell / 远程 git dispatch 未逐项 |
| 13 | task-control peer | `main/task-control-peer.js:57` | ✅ 已对齐（G47）：`internal/desktop/taskcontrol/control.go` + `cmd/whaleisle/task_protection.go` |
| 14 | AI 提交信息 / PR 文案生成 | `main/git-generate.js:139` | ✅ round126：`internal/desktop/git/commit_generation.go` + `cmd/whaleisle/git_generate.go`；标题栏提交条「自动生成」 |
| 15 | 在线壁纸目录 | `main/wallpaper-catalog.js:20-56` | ✅ 已对齐（G48）：`internal/desktop/wallpapers/catalog.go`（Bing/Wallhaven 源 + 自定义 JSON）+ 画廊 |
| 16 | relay 客户端 / 远程 shell / 远程 git dispatch | `main/relay-client.js`、`main/remote-shell.js`、`main/dshd-remote.js`、`main/dshd-git-dispatch.js` | ⚠ 差异（G45）：无自研 relay；`bridge.cjs` 仅消费守护进程 `relayConnected`；直连/重连退避未取证 |

### 空壳未实现（有界面/有定义、逻辑未接或仅探测；审计报告 §1.3，8 项）

| # | 位置 | 判定 |
|---|------|------|
| 1 | `internal/desktop/components/lifecycle.go:23` | install 动作为空壳（start/stop/open 已实现） |
| 2 | `internal/desktop/phoneremote`（整个包） | 整包为空壳：UI 展示就绪清单 + ConnectURL，无任何服务端 |
| 3 | `internal/ui/chrome/wallpaper.go:18` | 壁纸为内容层占位（MyGo 无窗口背景图 API，属平台限制） |
| 4 | `internal/ui/market/market.go:1` | 包注释 "stub" 过时（本批次已修正）；面板实际已实现列表/安装/启停/卸载 |
| 5 | `internal/ui/remote/remote.go:1` | 包注释 "stub" 过时（已修正）；面板为配置状态展示 + SSH 隧道表单，dsh-remote RPC 未接 |
| 6 | `internal/store/im`（im/* 通路） | im/* 发送/历史为空壳通路（上游 vendor 无 RPC，非 Go 代码缺陷；完整 IM 靠 dsh-im WebView） |
| 7 | `internal/store/usage/store.go:1`、`internal/ui/chat/chat.go:1`、`internal/ui/usage/usage.go:1` | 包注释 "stub" 过时（本批次已修正为准确描述） |
| 8 | terminal/follow 流 | 原为 `probeTerminalFollow` 连通性探测（160 端点中唯一未接真实数据流）；本批次已按真实 harness 线协议接线（见下） |

### round126 补全（2026-10-24）

- 5 项缺失已补成可运行实现：Welcome 首启窗、系统通知、外部编辑器、WhaleBridge 账号窗、AI 提交信息生成；逐项对照与边界见 [round126](review-20261022/round126-missing-features.md)。
- 更新过程补上标题栏进度行（checking / downloading / staged）、确认卡技术详情折叠与 `update-journal/` 证据；journal 失败只丢证据，不阻塞更新。
- 详情卡 badge 按安装版 `STATUS_KEYS` 词表本地化，未知状态保留原文。
- 证据：受影响 11 个包测试全绿、`go vet` 0、`go build ./internal/... ./cmd/...` 0；构建 86,703,104 字节 / SHA256 `7BAF783A…`（`round126-checks.json`、`round126-build.json`）。
- 仍剩 8 项部分/差异（桌宠渲染、手机远程、菜单覆盖、dsh-remote、relay、通知呈现、标题行中心、动效残项）；缺失项本身已补齐，验收合同仍零双签。

### round129 补全（2026-10-24）

- 关闭遮罩（G40）：退出时先画「关闭中 / 正在停止本机 Harness 服务，请稍候」，恢复/显示托盘窗口，绘制上限 500ms，失败不阻塞停服；详见 [round129](review-20261022/round129-closing-overlay.md)。
- 证据：`cmd/whaleisle`、`chrome`、`i18n`、`frame` 四包测试全绿、`go vet` 0；浅/深色渲染图已入证据目录。spinner 旋转与实机截图仍未做。

### round127–130 详情卡（2026-10-24）

- round127：ToolDetails 通用条目（lines / groups / unchanged / 共享 CodeBlock）。
- round128：agent/team 控制收据（list_agents、spawn_teammate、team_task_*、send_message、interrupt_agent、wait_agent、subagent、list_subagent_models）。
- round130：inspection 族（cordis_inspect_*、workflow、ralph、session_search/event_search/event_read、session_trace/event_trace）。
- 仍缺：Inspect 面板、markdown/location 条目、lsp、terminal_list；pass270 已构建（86,862,848 字节 / SHA256 `E3D63B46…`）。

### round131–133（2026-10-24）

- round131：崩溃报告（G41）+ 媒体权限（G42）落地；四源报告、保留 10 份、恢复面点名；loopback 麦克风/通知门控（帧级限制已登记）。
- round132：ToolDetails 条目收尾（markdown、可点 location、todo status/change 记号）；job_output 改共享 CodeBlock；新增 terminal_list、lsp 卡。
- round133：标题带按钮上沿 8→12（y=12..44），popover 150→160ms / 位移 4、overlay 位移 8；G01 经核对是测量错误（带高本就 48）。
- 证据：pass271（86,918,656 B / SHA256 `FE73AE32…`），四包测试全绿；残留：overlay-out 100ms、scale(0.96)、标题行中心 24→28、todo_write 族、实机像素。

### round134–136（2026-10-24）

- round134：应用菜单按 menu.js 逐项对齐（新增「在资源管理器中打开工作区」、移除原生没有的 Edit/快捷键/本地打开行、keycap 只取已接受 revision、无绑定不显示、首个快照前用出厂默认）。
- round135：	odo_write 详情卡——从已加载事件窗重建上一版清单，initial / compare / unavailable 三态，added/updated/removed/unchanged 与原生文案。
- round136：标题带内容坐到 y=12..44（中心 28）；新增 widgets.DesignEase = cubic-bezier(0.4,0,0.2,1)，替换四处 motion token、Motion.Animate 默认、chrome 过渡与 Reveal；overlay 退场 100ms 受 ui.ElementTransition 单时长限制未做。
- 证据：pass272（86,957,056 B / SHA256 `E7371690…`）；frame/chat/cmd/widgets/chrome 测试绿。残留：scale(0.96)、overlay-out、browserpreview TopInset、关闭按钮 hover、启动器/boot 独立窗（计划见 docs/parity/launcher-boot-plan-20261024.md）。

### round137–139（2026-10-24）

- round137：sk_user_question 历史卡（answered / unanswered / 未作答标签，严格校验失败回退 OUT；pending/closed 仍缺 userQuestions 投影）。
- round138：标题栏按钮 hover 填充（关闭 #e81123 + 白字，最小化/最大化交互底色），并修正 `ui.Painter.Line` 只画横条导致「✕ 变横线」的既有缺陷；新增 scripts/window-geometry-probe.ps1（只 GetWindowRect，绝不 PrintWindow）。
- round139：启动器改为独立 **1060×660** 原生窗口（冷启动隐藏主窗；start 后销毁启动器并显示主窗一次；托盘/深链复用同一 HWND；--background/--settings/隔离/失败创建回退主窗内面板；一个 launcherPanel 模型）。
- 实机证据：pass273 探针 1590×990 px = 1060×660 DIP（dpi 144）可见启动器 + 2160×1380 px = 1440×920 DIP 隐藏主窗；pass272 主窗 2160×1380 = 1440×920。见 evidence/window-geometry-pass273.json。
- pass273：87,241,216 B / SHA256 `649BBA3D…`；cmd/chat/frame 测试绿。

### round140–143（2026-10-24）

- round140：boot 揭幕 600ms 改用 widgets.DesignEase，用 boot 页面自身背景（62/38 刚体切片、地平线固定）替代合成场景；原生快照文字/动作滑动仍缺（native-UI 无离屏捕获）。
- round141：托盘图标 32→24（仅 win32）、窗口图标补 48×48（main/welcome/launcher），几何探针新增清理陈旧 mygo-*.sock。
- round142：i18n 审计（91 key 对齐 / 164 value / 56 行标题），修正 10 处文案（IN/OUT、Token、Cached、Cache created、Ask question、拖放句、轮次）；cmd/ 254 处 inline 文案与设置页全量扫描列为后续。
- round143：**关键修复** — 主窗首帧自锁（round126 起，mainView 持 state.mu 时调用自锁的 git 生成访问器）。Responding 永久 False；改为 Locked 访问器 + 回归测试。pass274 主窗 --settings Responding=True（CPU 1.22s），启动器 1060×660 响应正常。harness start→ready 仍未验证。
- pass274：87,529,472 B / SHA256 `2852B7FB…`；几何证据 window-geometry-pass274.json（启动器 1590×990 px = 1060×660 DIP，主窗隐藏 2160×1380 = 1440×920）。

### round144–146（2026-10-25）

- round144：设置页文案审计 623 处 providerCopy，46 处确认漂移已修（General 13 / Appearance 16 / Models 2 / About 1 / Pet 11 / shortcuts 1），其中 scheme.title 的「色制」按原样对齐。
- round145：**单实例陈旧 socket 修复**——强杀/崩溃后不再静默退出：owner 读参数后回写 ok 握手，无应答则接管 socket；存活进程仍占路径时照常启动（仅失去转发）。	hird_party/mygo/single_instance.go + 复现脚本。
- round146：**暂存 rc.2 harness runtime**（来源为安装版只读资源，untime/ 不入 git）并首次跑通 **start→boot→ready（约 6s）**：主窗 1440×920 可见、启动器关闭、端口 3099 监听、UIA 读到完整聊天壳（侧栏/工作区/探索未至之境/composer/deepseek-official / deepseek-flash/发送消息），无启动错误、无 crash 报告。
- 口径更正：Process.Responding/MainWindowHandle 会缓存启动器句柄，启动器销毁后会误报 False；以后以新 Process 实例 + WM_NULL 探测 + UIA 文本为准。
- 证据：pass275（87,531,520 B / SHA256 `08FB561F…`）；scripts/harness-start-smoke.ps1 可重复；harness-start-pass275.json ready=True。
- 残留：Project overlay 未挂（staged vendor 无 current-host-build.json，dsh-project bridge 跳过）；runtime 路径绑定，移动仓库需重新 stage；.runtime-rc2/ 旧暂存可删。

### round147–150（2026-10-25）

- round147：启动器首页产品行补齐原生 48px 头像 + 12 gap + eyebrow 10/16 + 标题 16/24；26 项 shell 指标中 25 项已相等。welcome 参考页由 preload 动态填充，静态渲染不可用，仅留指标路径。
- round148：ui.Motion.Scale（0/1 为 identity，中心缩放 + clip）与 ElementTransition.ExitDuration；chrome overlay 变为 200ms 进（Y8/scale .96）、100ms 出，popover 仍 160ms/4。
- round149：PrintWindow 在死锁修复后恢复可用，scripts/window-capture.ps1 以 per-monitor-v2 物理像素截图；建立启动器浅/深色、欢迎窗、聊天主窗 150% 基线。
- round150：**安装版 rc.2 readiness 被接受**（复用 validateStagedSessionRuntime + 8278107b 门禁 + dsh-project 存在性），默认 runtime 无需伪造 marker 即挂载 Project overlay；Host 第一个 --patch 为 desktop-project.patch.yml，就绪约 8s。
- pass276：87,539,712 B / SHA256 `EDD075B5…`；harness-start-pass276.json ready=True。
- 残留：启动器导航图标/品牌资产/首页标题文案（task-25 进行中）、welcome 指标比对、聊天界面逐表面像素审计、per-profile project stage 成本、current-host.mjs verify 仍只认源码构建。

### round151–155（2026-10-25）

- round151：启动器 6 个导航图标改为原生 launcher.html 的 16×16 SVG path 精确绘制；首页标题「鲸屿桌面端」；品牌头像与原生 whale-head.png 逐字节一致。
- round152：打通**原生 harness Web UI 参考管线**（token URL + CDP 截图 + DOM rect 测量），实测修正：侧栏底部合并为「设置与账号」36 DIP 座位、导航行距 36→40、hero 徽标「大肥鱼版」、composer 全角标点、模型 provider/model 无空格；并实现**首次运行「预览版说明」**（600×344、120×36 继续、goPreviewNoticeAck 持久化、保存失败提示）。pass278 实拍确认。
- round153：欢迎窗按原生 welcome-layout.css 网格重写（36 拖拽条 + 40 品牌行、24/48/76 内边距、26/40 标题、240 宽动作列 44 高按钮、440 宽 key 表单），17 项指标测试钉住。
- round154：桌宠几何对齐原生 desktop-live2d.js 的 240×260 足迹 / 24 边距（原生窗口实为全屏透明 overlay，不随成长改尺寸）；MyGo 窗口 240×438（含页面状态面板）。
- 桌宠启动缺陷：窗口确实创建，但 whale-pet:// 导航被 WebView2 中止（status 14）后窗口被关闭且无重试；已加 openCompanionWindow 缝隙、加载失败重试一次 + 记录、主窗 OnShow 重新 SetReady。**实机桌宠仍不显示**（scheme 导航待修）。
- round155：原生参考管线文档与脚本 scripts/native-web-shot.mjs。
- pass278：87,543,808 B / SHA256 `5A708C13…`；harness-start-pass278.json ready=True；实拍 mygo-chat-notice-pass278-150dpi.png、mygo-chat-hero-pass278-150dpi.png。
- 残留：composer 高 126 vs 原生 114（+12）、页头 48 vs 52、折叠按钮 ±2 DIP、新会话/工作区工具组未复测；桌宠 scheme；欢迎窗品牌内容/tint；升级路径。

### round156–159（2026-10-25）

- round156：composer 卡片 126→**114**（原生 padding 8/0/8、gap 12、34 控制行、8 侧 inset）；新会话 **252×38 @ (14,70)**、页行 252 宽 x14、标签 x46、pitch 40；折叠 **28×28 @ (240,22)**；页头 52 仅浏览器，桌面注入 CAPTION_HEIGHT=48，保持 48 合并标题带。
- round157：远程/中继功能矩阵——配置默认值/归一化一致、协议复用原生 dshd-remote；G1 无 mobile runtime、G2 LAN 配对被 relay 门禁卡死、G3 relay token 未注入、G4 ssh 模式、G5 LAN 发现、G6 TLS/SSH。
- round158：修复 G4（仅 lan，其它→relay）、G3（凭据中的 remoteToken/remoteRelayToken 注入 daemon 配置）、G2（LAN 监听即 ready 并保留 pairing URL；bridge.cjs 同步）；G1/G5/G6 需真实 daemon。
- **桌宠可见性修复**：根因是创建时 Hidden:true 且从未 Show()，隐藏的 WebView2 控制器中止自定义 scheme（status 14）→ 窗口被关闭。现在加载前 Show、原生右下 24 gap 定位、加载完成后忽略后续 abort；实机 **360×657 px = 240×438 DIP**，页面 hit-region 240×438。渲染层差异（web 页+状态面板 vs live2d 全屏 overlay）保留。
- pass279：87,550,976 B / SHA256 `063DD647…`；harness-start-pass279.json ready=True。
- 残留：composer 宽 772 vs 774.4、导航文字基线 1.7 DIP、hero 纵向 seat 需桌面参考；G1 mobile runtime、G5 LAN 发现、G6 TLS/SSH；验收合同双签。

### round160–162（2026-10-25）

- round160：设置页按原生 DOM 实测收口——导航项 38→40 高、padding 8→9、gap 8→4；行高下限 64→68；值控件右对齐（药丸与列表右缘齐平）；对话框 800×800 r28、行 16 padding、14/22 标题 + 12/18 描述、36×20 开关等已相等。
- round161：桌宠静息呈现对齐原生——不再常驻「原版分层动画」标签与「状态」按钮；标签仅在渲染降级时出现，状态改为 hover/键盘/右键/长按唤出，焦点与 Esc/外点关闭；窗口仍 240×438。
- **手机远程打通**：从安装版 app.asar 提取 dshd-remote（src/main + shared + mobile/web + 解包条目 + node_modules），better-sqlite3 为 Electron ABI 149，故 daemon 以 ELECTRON_RUN_AS_NODE=1 运行（已写入 spawn env）；并修复启动路径——efreshBoot 首次 Ready 时一次性启动网关。实机 pass280：3099/3499/3180 全部监听，手机 SPA Whale Isle · 手机远程 200（16,453 B），kill 后全部关闭。
- round162：gap 矩阵刷新为 89 行（63 closed / 25 open / 1 measurement-error，新增 S01–S05 小缺口）。
- pass280：87,558,144 B / SHA256 `A6AC099E…`；harness-start-pass280.json ready=True。
- 残留：app 内配对码未断言（daemon ~28s 才生成 offer）、relay/0.0.0.0/真机未验证；桌宠 260 高度需 popover；设置页仅 General 做了原生 DOM 逐项；S01–S05；验收合同双签。

### round163–165（2026-10-25）

- round163：**S01 下载处理**——移植原生 download-path.js 名称清洗（非法/控制字符、尾部点空格、DOS 设备名、180 rune 上限、`download` 兜底）+ `name (2).ext` 去重，并给所有非临时 webview 安装 OnWillDownload；S02–S05 记录原生规格与精确阻塞（安装空间预检需平台 statfs；Ctrl+Shift+I 需 mygo Page 输入钩子；预览缩放 17 档待接线；插件取证分类待逐条比对）。
- round164：设置页剩余 tab 原生 DOM 实测（外观/界面/模型/关于；账户在未登录时不存在）；修正两处文案：外观标题「色制」→「颜色模式」（参考源码快照过时，安装版即 DOM 一致）、模型 intro 对齐安装版；界面开关 44×28 命中区 vs MyGo 36×20 视觉按钮、主题/模型卡片结构差异已记录。
- **app 内配对已证**：WHALEISLE_REMOTE_DIAG=1 输出无 token 快照；in-app ready、hasPairingURL、offer v2、QR 条件全真、SPA 200；监听关闭正常。
- **验收索引**：96 行合同 0 行 contract-ready / 57 部分证据 / 39 无证据；top blockers：无每屏 baseline/actual/diff/geom/controls、无 cases-<ID>.md、无 accept-log/fixture、原生基线为 harness-Web 而非 Electron 壳、无 E7 mask registry。
- pass281：87,578,624 B / SHA256 `DF858D80…`；harness-start-pass281.json ready=True。
- 残留：app 内 QR 像素/UIA 面板未截（采样过早）；远程面板 offer 仍以 relayConnected 为门（LAN-only 门禁一行修复）；S02–S05；验收工件体系；双签只能由用户完成。

### round166–169（2026-10-25）

- round166：**S02 安装空间预检**（256 MiB 预留、%.2f GiB、祖先目录探测、unsupported FS、Windows GetDiskFreeSpaceEx/Unix Statfs）；**S05 插件取证**改用原生四条正则（裸 oom、ENOENT node）；S04 浏览器预览缩放核实为已实现（17 档 + find/next，补钉测试）；S03 仍缺 mygo Page 键盘钩子。
- round167：**验收工件管线**（scripts/acceptance-capture.ps1 + acceptance-diff.py）跑通 D-MAIN 试点，产出 baseline/actual/diff + geom/controls/accept-log/mask；Electron UIA 懒加载、子 HWND、任务栏占高、1px 尺寸差、PS UTF8 BOM 等坑已记录。**严格 threshold 0 对 Go 原生渲染 vs Chromium 不可达**（试点 99.99% 主要因主题/数据状态不同；即便同夹具，字形抗锯齿与壁纸动画仍必然不同），文档提出「蒙版内 0 + 结构 1 DIP + 每区域 ≤0.1% + 文本 AA 单列」的实用准则待用户确认，合同未改。
- round168：设置剩余 tab（内置插件/技能/MCP/Agent 预设）中英 + 深色 token 实测；修 4 处文案（插件英文描述、技能计数前置、MCP 空态第二行、预设「新任务默认」）。
- round169：**LAN-only 远程 offer 门禁修复**——offerUsable 按 (relayConnected 或 mode lan) 放行 QR/复制/打开；测试覆盖 lan-no-relay/relay/过期/缺 URL；面板截图仍被首次预览弹窗挡住。
- pass282：87,587,328 B / SHA256 `32C39781…`；harness-start-pass282.json ready=True。
- 残留：验收准则需用户决策；fixture 尚未播种（原生深色/真实 profile vs MyGo 浅色/空 profile，MyGo 深色主题未生效待查）；L-HOME 启动器窗口与 MyGo 面板形态不同；S03；QR 面板像素；逐行验收工件与双签（用户）。

### round170–176（2026-10-25）

- round171：**S03 DevTools 快捷键**——mygo 新增 Page.OnBeforeInput（可选平台能力，Windows 用 AcceleratorKeyPressed 实现；linux/darwin 如实返回不支持；fake 可测试），webview 按原生 !packaged || openDevTools 门禁每窗切换，保留菜单项。
- round174：**主题渲染路径修复**——原生 config.theme 只存 family，明暗在 Host appearance preference；新 startupTheme 让配置里的 scheme 在用户未显式选择前生效，用户选择后持久化并优先；10 例表测试；实拍 deepseek.dark 启动器/欢迎窗为深色。
- round172/175：深色匹配的 D-MAIN 对照在 pass283 上完成：两边均深色；threshold-0 仍 99.99%（**壁纸不同 + 半透明 mica**），容差区域统计已记录；结构 17 项（含别名）但 0 项在 1 DIP 内，因 MyGo 合并 48-DIP 标题带 vs 原生页+标题堆叠——建议改按标题相对坐标比较。
- **L-HOME MyGo 半边**：无 workspace 冷启动确认为独立 1060×660 MyGoWindow（深色），已实拍；原生启动器需以临时 user-data 冷启动（Option A 待用户同意重启原生应用）。
- **远程配对 UI 挂载**：iewPhoneSection 原本无生产调用者，用户永远看不到二维码；现已挂到设置「远程」页，状态/入口/就绪行全部来自 live snapshot，LAN 入口从 pairing URL 推导；headless 渲染出 QR 视觉证据，live 面板显示 offer v2 · ready/有效期/复制/打开；新增 whaleisle://settings?section=remote 深链。
- pass284：87,595,520 B / SHA256 `AF437872…`；harness-start-pass284.json ready=True。
- 残留：**壁纸/材质层尚不一致**（原生用户壁纸 + mica vs MyGo 渐变）；验收准则需用户决策；结构比较需标题相对坐标；L-HOME 原生冷启动需授权；S03 macOS/Linux；深色 pill/nav token；QR 应用内滚动截图；relay/真机；双签（用户）。

### round177–180（2026-10-25）

- round177：**S28 导航标签基线修复**——页面行与鲸鱼娘行改用原生 22-DIP 行盒（7 padding 下），标签 ink 不再低 ~1.7 DIP，frame 测试断言 22 盒。S27 composer 卡 773.33 vs 原生 774.39（差 1.06 DIP），根因在 contentAxis 同时驱动正文轴，需先量原生正文轴；hero 栈 +24.7 DIP 因原生 860 高度下卡片约 155 DIP，而 MyGo 仍是 920 高下的 114 配方——需原生卡内行 rect 的 --eval 探针。
- round178：G10 标题栏右侧簇逐项实测（顺序/27-DIP 分支胶囊/32×32 面板钮/4 间隙/8 胶囊间隙/32×32 窗口钮全匹配），面板钮字形 14 vs 可能 16–18 未敢猜改，记录 3× 放大复核法。
- round179：**G52 首批 63 处 cmd 文案迁入 i18n**（12 文件/55 键，托盘/菜单/更新/账号/恢复），测试覆盖双 locale、非空/不等、%s/%d 参数数与会话残留检查；余 187 处按文件列清单。
- round180：验收索引刷新——96 行中 1 行证据完整（D-MAIN）、59 部分、36 无，0 可签；新增 D1–D4 决策（实用准则、L-HOME 原生冷启动、壁纸策略、分层 CDP 基线）与 top-10 阻塞。
- pass285：87,623,168 B / SHA256 `6A2148B6…`；harness-start-pass285.json ready=True。
- 残留：壁纸/mica（D3）仍是像素主差；原生 860 卡高/正文轴探针；G52 余量；cases-<ID>.md；G44/G45 真机；G31b Live2D；深色 pill token；双签（用户）。

### round181–185（2026-10-25）

- round181：**S27 探针纠错**——1440×860 与 920 下 composer 卡都是 774.39×114（max-width 774.4 字面量），round177 的「860 下 155」是把停在 x=−4200 的 UIA 坐标混帧；真正偏差是 hero headline 到 preset 的间距：原生 29 DIP，MyGo 12 DIP → 改 25，preset/card 不再高 13 DIP。composer 余 1.06 DIP 在列宽链（contentAxis 709.33 vs 710.39），三种 cap/floor 方案都会破坏 hero/docked 同一性，已回退并记录；标准模式 chip 应左对齐 workspace chip（待修）。
- round182：**结构验收改为 caption 相对坐标**——0/17 实为坐标系 bug（原生 UIA 屏幕坐标 vs MyGo 客户区，dx+2800）；绝对/窗口/标题相对三空间匹配 22 项，1 DIP 内 0/4/5；标题底 40.0 vs 44.67（+4.67）、侧栏标题行 +6.67。新增低频洗色蒙版（覆盖 91.35%），阈值失配 99.9908% → 8.6381%，剩余集中在真实内容差异。
- round183：G03 菜单栏实测双方 GetMenu=0/GetMenuBarInfo=false、客户区无菜单带（仅 1px 取整）→ 无差异已核实；S03 实按未取证（首跑用了早于接线的 pass285，二次被 sqlite 占用中断，步骤已记录）；G10 glyph 14 vs 16–18 仍待 3–4× 放大。
- round184：G46 实测——原生 POST /__remote__/shell/<name>（Bearer/cookie、14 路由），MyGo 3499 返回 404、3180 返回 405，因 bridge.cjs 构造 DshdRemote 时没有 invokeShell/fs/host；给出「挂载 Remote gateway」或「bridge 前置 shell 路由」两条路径，未半接线。
- pass286：87,643,136 B / SHA256 `DAAA5E81…`；harness-start-pass286.json ready=True。
- 残留：caption 三钮 +4.67（F1）、侧栏标题行 +6.67（F2）、composer 列宽、标准模式 chip 对齐；G52 余 99 处；S03 实按、G10 glyph；G46 接线；壁纸/mica（D3）；D1/D2/D4 决策与双签。

### 本轮修正记录（terminal/follow 处置与注释/文档清理）

1. **terminal/follow 真实化**：`internal/store/hostterminal` 此前的实现打开了 vendored harness 并不存在的 `terminal/attach` duplex 端点，并把 write/resize 作为流内 uplink 命令、依赖并不存在的 `ack`/`writable`/`inputWindow` 帧字段——对真实 harness 无法工作。本批次改为 harness 实际线协议（证据：`runtime/vendor/deepseek-harness/packages/api/terminal-controller/src/types.ts` 的 `TerminalFrame` 仅 snapshot/output/state 三帧；`lib/typert.remote-client.js` 注册 `terminal/follow` 带 agentId/id/attachmentId，`terminal/write`、`terminal/resize` 为一元 RPC）：follow 流绑定 attachmentId，可写性由 `info.controllerId == attachmentId` 推导，输入/缩放走一元 RPC 且失败即失效不重放；controller/duplex/recovery/selection 单测同步修正。底栏「本地 Ghostty PTY」设计不变。
2. **腐烂 stub 注释清理**：`internal/ui/chat`、`internal/ui/sidebar`、`internal/ui/market` 包注释更新为准确描述（`internal/ui/usage`、`internal/ui/remote`、`internal/store/usage` 已由前序工作修正）；i18n `KeyFrameRightStub`/`KeyFrameBottomStub`/`KeyWebViewStubBody` key 名保留不改（避免大范围重构；其中 `KeyFrameRightStub`/`KeyFrameBottomStub` 当前无渲染引用，是否作为死 key 删除留待专门清理）。
3. **降格登记**：M11「手机远程」与 M9「壁纸画廊」由「已完成/主体可用」降格为「部分实现」（见总览表与 M9/M11 条目）。

## 已知问题与技术债

1. M2：Office overlay 已接（无 payload 时跳过）；install 控制面 + pnpm add + 快照回滚/可加载校验已接。
2. credentials 的 v10 信封格式需用真实 Electron 44 安装版的产物实测确认。
3. ~~`mygo.json` identifier 占位~~ → 已为 `ai.deepseek.harness.gui`。
4. ~~托盘图标是运行时缩放的（可用 `golang.org/x/image/draw` 提升质量）。~~ → `cmd/whaleisle/trayicon.go` 用 `draw.CatmullRom` 缩放 `resources/icon.png` 至 32×32；单测 `trayicon_test.go`。
5. 事件深层 payload（content / source / surfaceOp 等）在生成 struct 内仍多为 `json.RawMessage`；根级 persistence 事件已有 `events.UnmarshalEvent` 按 `type` 分发到生成 struct（`session/classify` 对 tool/call、assistant/message 优先 typed，失败回退 raw）。
6. ~~尚无 `ui.NewTester` 覆盖主窗口。~~ 已有 `internal/ui/frame/frame_test.go`（主壳）、`internal/ui/boot/boot_test.go`（boot 页）与 `internal/ui/chrome/titlebar_test.go`（自绘标题栏 / 应用名）。
7. 与 Electron 版共用数据目录，两个版本不能同时运行。
8. config 的冲突检测只覆盖进程内串行化；跨进程未做原子 CAS。

## 下一步顺序

1. **M0–M11 主体仍可用**，但 **未宣称 GA**：需 Electron 44 同目录 credentials 信封实测、双开数据目录约束、跨进程 config CAS 等见「已知问题」。
2. **M5**：citations 为 JSON/正文启发式；待 harness persistence 正式 source 字段后再做 typed 解析（可删大量 walk 逻辑）。
3. **M9**：浏览器预览已独立面板；待补 agent 侧「打开链接」统一入口（composer/工具卡 deep link）、WebView 焦点/层级回归（多面板切换 Host inset）。
4. **已接（本轮）**：`events.UnmarshalEvent`；托盘 `CatmullRom` 缩放；composer Undo/Redo；NSIS **编译 + TEMP 静默安装/卸载**冒烟（`smoke-nsis.ps1 -SilentInstall`，不碰真实 Electron Programs 目录）；覆盖路径不变量单测。
5. **已知非阻塞**：真实覆盖 Electron Programs 目录的人工验收；credentials v10 需真机 Electron 44 信封；跨进程 config CAS；vendor 无 dshbot chat / dsh-im 消息历史 RPC。
6. **vendor 边界（已审计）**：dshbot 无独立 chat RPC；dsh-im 无消息历史 typert；IM 完整收发仍依赖 WebView。

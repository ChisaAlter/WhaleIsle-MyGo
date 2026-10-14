# 开发进度

2026-10-14 补充：最新逐模块交付与验证边界以 [模块交付记录](module-delivery-20261014.md) 为准。下文历史环境的实测记录不代表当前累计版本已通过真实 Host 或 Windows 桌面验收。

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
| 启动器 / 导入 / 更新 / 安装包 | M11 主体可用 | staging + NSIS；更新器 hash/签名/blockmap 校验；手机远程；任务保护/深链 |

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
  - 已完成：用量面板（account/job + 近 7 日柱状图 + 近 28 日热力格 + WebView2 打开 `usageUrl`）；插件市场；CSV/TSV/XLSX 预览；PDF/Office 外部打开；定时任务；壁纸画廊；chroma 源码预览。
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
- **M11 外围与发布（进行中）**：见下节；启动器 UI、深链路由、NSIS、sha256/size + Authenticode + blockmap stub 已落地；**手机远程**（Assess / 连接 URL / 复制剪贴板 / 打开浏览器·WebView / 刷新 / `remoteDevices` 配对列表）已落地。

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

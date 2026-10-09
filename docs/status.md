# 开发进度

更新时间：2026-10-09。总体计划见 `docs/plan.md`（里程碑 M0–M11）；本文记录每项的实现状态，随开发更新。

## 总览

| 部分 | 状态 | 位置 / 说明 |
| --- | --- | --- |
| Go 工具链与脚本 | 完成 | `scripts/env.ps1`、`scripts/build-harness.ps1` |
| harness 运行时构建 | 完成 | `runtime/vendor/` 8 个目录，产物已验证 |
| 配置 `internal/config` | 完成 | 全字段 + 钳制 + 快照隔离 + 冲突拒绝，测试通过 |
| 凭据 `internal/credentials` | 完成 | DPAPI + AES-GCM 信封，分类加载，测试通过 |
| 路径 `internal/paths` | 完成 | userData / dsh-home / runtime |
| 应用骨架 `cmd/whaleisle` | 骨架完成 | 窗口 / 托盘 / 单实例 / 关窗到托盘 |
| 接口生成器 `tools/gen-api` | 完成 | 标准库实现，测试通过 |
| dsh API 生成物 `internal/dsh/api` | 部分 | 6 包 27 端点；全量 140+ 待重新生成 |
| dsh 客户端（RPC / WS / 事件） | 未开始 | 计划 M3 |
| harness 启动器改造 | 未开始 | 现存代码仍是 npx 方案，计划 M2 |
| 界面（对话 / 设置 / 文件 / Git / 终端等） | 未开始 | 计划 M4–M9 |
| dsh-whale 助理 / dshbot / webview 兜底 | 未开始 | 计划 M10（桌宠暂不做） |
| 启动器 / 导入 / 更新 / 安装包 | 未开始 | 计划 M11 |

## 已完成

### 1. 工具链与项目脚本

- Go 1.27.2 下载到 `.toolchain`；任何 go 命令前先执行 `. .\scripts\env.ps1`（设置 GOPATH / GOCACHE / GOTOOLCHAIN）。
- `go mod tidy` 已完成，MyGo 的传递依赖（purego、typesetting、x/image 等）已入 `go.mod` / `go.sum`；`go build ./...` 全仓通过。
- `scripts/build-harness.ps1`：harness 构建的可重复脚本（幂等，可安全重跑）。

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

### 4. `internal/credentials`（新包）

- credentials.json 六字段读写：apiKey、baseUrl、githubToken、remoteToken、remoteRelayToken、remoteDevices。
- `safeStorage-v1` 信封：`v10` 前缀 + 12 字节 nonce + AES-256-GCM；密钥用 crypt32.dll 的 DPAPI 从 Local State 的 `os_crypt.encrypted_key` 解出。纯标准库，无 cgo，未加依赖。
- 兼容明文 JSON 读取；原子写（临时文件 + 重命名）。
- 加载分类：文档若带 `version` / `payload` 标记，必须是结构完好且受支持的信封，否则以 `ErrInvalidCredentialsFormat` / `ErrUnsupportedEnvelopeVersion` 拒绝（`errors.Is` 可判），绝不回退当明文读；缺失文件返回零值且无错误；base64 校验先于读取 Local State 密钥；失败时不返回部分字段，错误文本不含取值 / payload / 版本号。
- 测试 9 个（真实 DPAPI 往返、合成 Local State、篡改检测、分类表驱动），全绿；只使用临时目录。
- 未确认点：Electron 44 的 safeStorage 在 Windows 是否确实产出 v10 AES-GCM 信封，需用真实安装版生成的 credentials.json 实测一次。

### 5. `internal/paths`

- userData（默认 `%APPDATA%\Deepseek-Harness-Desktop`，可用环境变量 `WHALEISLE_USER_DATA` 覆盖）、dsh-home、runtime、pid 文件路径，以及 Ensure 系列辅助函数；导入无副作用。测试 4 个。

### 6. `cmd/whaleisle` 应用骨架 + `mygo.json`

- 使用的 MyGo v0.3.5 API 均已对照源码核实：单实例（`RequestSingleInstanceLock` / `OnSecondInstance`）、托盘（`NewTray` / `TrayOptions`）、退出（`App.Quit` + quitting 标志放行 OnClose）、关窗到托盘（暂硬编码 true，未接 config）。
- 主窗口：原生 UI 占位视图，1200x800，`StateKey: "main"`，`light-dark(...)` 背景色。
- 冒烟验证：窗口正常出现；第二实例 53ms 内自行退出（单实例生效）；进程可干净终止。
- `mygo.json`：`{"name":"Whale Isle","identifier":"com.whaleisle.app"}`。identifier 是占位，发布前按"替换升级"决定改为 `ai.deepseek.harness.gui`（见 plan.md §10）。

### 7. `tools/gen-api` 与 `internal/dsh/api`

- 生成器解析构建产物 `lib/typert.remote-client.js`（`TYPERT_REMOTE` 描述符），生成 Go 类型与端点注册表；支持 `mode:'stream'`、`kind:'context'`、`uplink`、`cancellation`；zod schema 尽力映射成 Go 类型，复杂形状回退 `json.RawMessage` 并注释原类型。
- 已用现有 6 个真实产物生成 27 个端点（dsh-llm、dsh-goal、dsh-commands、dsh-cordis-host-runner、dsh-message-feedback、dsh-host-plugin-inventory）+ `registry_gen.go`。
- 用法：`. .\scripts\env.ps1; go run ./tools/gen-api -in <包根>... -out internal\dsh\api`。
- 测试全绿，含对 `.toolchain/endpoints.json`（140 端点清单）的交叉校验回归。

### 8. 沿用代码 `internal/harness`（部分旧设计）

- 已有：子进程监管与自动重启、端口扫描、`dsh web: <url>` 就绪行解析、启动令牌换 dsh-auth cookie、workspace/create 登记。
- 旧设计：启动方式仍是"配置 dshBin / PATH / npx"；M2 将改为 vendored harness（node + bin.js + 覆盖文件 + junction）。
- 已知失败测试：`TestResolvePrefersPathThenNpx`（测试按 Linux 写法找 npx），M2 重写时一并消除。

### 9. 文档与参考

- `docs/plan.md`：总体计划（架构、协议调研结论、里程碑 M0–M11、风险、已确认决定）。
- `reference/electron`：原项目快照，基线提交 `26205a89f`（2026-10-06），见 `reference/README.md`；`reference/go.mod` 隔离其中的 Go 代码。

## 未开始（按里程碑，细节见 `docs/plan.md` §5）

- **M0 收尾**：用完整构建产物重新生成 `internal/dsh/api`（140+ 端点）；生成会话事件类型（`docs/persistence-schema.json`）；统一构建 / 测试脚本。
- **M1 底座**：完整主窗口（无边框 + 自绘标题栏）、主题令牌与明暗切换、中英文文案、托盘接 config、开机启动、config / credentials 接入主程序。
- **M2 harness 生命周期**：vendored 启动（Node 解析、11 个覆盖文件、junction、环境变量）、原生启动页、插件恢复路径、workspace/create 登记。
- **M3 dsh 客户端**：RPC（含 multipart 二进制结果）、WebSocket 多路复用与心跳、`$events` 与 waterfall 应答、断线重连；对真实 harness 做集成测试。
- **M4 对话最小可用**：主框架布局（侧栏 / 对话 / 右栏 / 底栏）、工作区与会话树、虚拟化对话流（流式 Markdown + 代码高亮 + 思考过程）、输入框（发送 / 停止 / 排队 / 模型选择）、审批与提问卡片。
- **M5 对话完整度**：各类工具卡片、Diff 视图、图片附件、交付物、消息编辑与反馈、用量标签等。
- **M6 输入框增强**：@ 提及、/ 命令、技能菜单、拖放文件、Agent 预设。
- **M7 工作环**：右栏停靠、文件树与编辑器、终端、Git 标题栏、Diff 工作面、工作区路径权限检查。
- **M8 设置**：通用 schema 表单渲染器 + 各专用设置页（15 个分区）。
- **M9 桌面扩展**：浏览器 / 文档预览（WebView2）、插件市场、用量统计、远程与 IM、SSH 远程工作区、壁纸。
- **M10**：dsh-whale 助理、dshbot 页面、第三方插件 webview 兜底。
- **M11 外围与发布**：启动器、数据导入、任务保护、手机远程、MyGo 更新器、NSIS 安装包（覆盖安装 Electron 版）。

## 已知问题与技术债

1. `internal/harness` 的 npx 测试在 Windows 失败（M2 重写时消除）。
2. credentials 的 v10 信封格式需用真实 Electron 44 安装版的产物实测确认。
3. `mygo.json` 的 identifier 是占位值，发布前改为 `ai.deepseek.harness.gui`。
4. 托盘图标是运行时缩放的（可用 `golang.org/x/image/draw` 提升质量）。
5. API 生成物仅覆盖 27 / 140+ 端点；会话事件类型尚未生成。
6. 与 Electron 版共用数据目录，两个版本不能同时运行。
7. dshbot 与 dsh-whale 下的 dsh-llm 产物完全相同，重新生成时传一个即可（生成器自动去重）。
8. config 的冲突检测只覆盖"读到的基线 → 替换"之间的进程内串行化；跨进程（同时运行的 Electron 版）在最终比较与重命名之间仍可能写入，未做原子 CAS，也未验证两版同时运行。

## 下一步顺序

1. 配置持久化门禁（本轮已完成：快照隔离 + 陈旧写入拒绝）——先于 API 补全。
2. 全量重生成 API + 生成会话事件类型（M0 收尾）。
3. M1 底座 → M2 harness 生命周期 → M3 客户端 → M4 对话最小可用（详见 plan.md §5）。

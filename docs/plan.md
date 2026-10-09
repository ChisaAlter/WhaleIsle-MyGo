# 鲸屿 MyGo 原生重构计划

## 1. 目标

用 MyGo 把鲸屿（Whale Isle）桌面端重写成一个纯 Go 程序：

- **界面全部原生**：对话、侧栏、设置、文件、终端、Git、用量、宠物等界面都用 MyGo `ui` 包重画，不再加载 dsh web 页面。
- **运行时不变**：继续使用原仓库 `vendor/deepseek-harness` 构建出的 harness（带桌面分支），以子进程方式运行 `dsh web`，Go 客户端直接调用它的 HTTP / WebSocket 接口。
- **桌面能力用 Go 实现**：Electron 主进程里的 Git、工作区文件、终端、预览、托盘、更新、导入、插件排查等改写成 Go。
- **数据兼容**：沿用 `%APPDATA%\Deepseek-Harness-Desktop` 下的 `config.json`、`credentials.json` 和 `dsh-home`，已有用户的数据不用迁移。
- **原项目继续开发**：`reference/electron` 是 2026-10-09 的快照，只作参考；之后按第 9 节的方式跟进原项目的改动。
- **不改动原项目（dshd）**：`C:\AI\Deepseek-Harness-Desktop` 只读，不在里面构建、安装依赖或写任何文件。重构项目全部在 `C:\AI\WhaleIsle-MyGo` 里新建。harness 和桌面插件（`vendor/deepseek-harness`、`vendor/dsh-*`、`vendor/dshbot`）从本项目里的副本构建，源码原样使用，不做修改。

## 2. 现状

实现进度不再在此处维护，见 `docs/status.md`（随开发更新）。当前要点：工具链与 harness 运行时构建（M0 核心）已完成；`internal/config`、`internal/credentials`、`internal/paths`、应用骨架与接口生成器已完成并有测试；dsh 客户端与全部界面尚未开始。
| 工具链 | Go 1.27.2 在 `.toolchain`，用 `scripts/env.ps1` 设置环境。 |
| 主程序、界面 | 没有。 |

## 3. 调研结论

### 3.1 dsh web 协议（Go 客户端要实现的部分）

- **普通调用**：`POST /api/<namespace>/<method>`，请求体是信封
  `{"type":"client-request","rpcId":"<uuid>","method":"session/list","payload":{"args":{...}}}`。
  参数按名字传，必须与描述完全一致，不能多也不能少。成功返回 `{"result":{"ok":true,"value":...}}`，失败返回 `{"ok":false,"error":{"code","message","details"}}`。部分二进制结果用 `multipart/form-data`。
- **流**：一个多路复用的 WebSocket `GET /api/remote.mux`。客户端发 `open` / `item` / `end` / `cancel` 帧，服务端回 `item` / `end` / `error` 帧。服务端每 2 秒发一次 Ping，连续 2 次没收到 Pong 就断开。没有协议级续传，断线后各个流自己重新开始。
- **事件**：打开 `$events` 流，先收到 `ready`（含 `clientId`），之后是 `emit` 通知和 `waterfall` 请求。**工具审批 `approval/request` 和用户提问 `user-questions/request` 必须回答**（`POST /api/$events/result`），否则工具调用会一直挂着。
- **鉴权**：启动令牌 `GET /?token=...` 换 `dsh-auth-*` cookie，之后所有请求和 WebSocket 握手都带这个 cookie。Host 必须是回环地址；不发 `Origin` 头即可通过来源检查。
- **对话主流程**：`workspace/follow`（工作区）、`session/list`、`session/control`（实时状态）、`session/create`、`session/follow`（历史快照 + 事件 + 助手流式输出）、`session/page`（更早的历史）、`session/prompt`（发消息）、`session/cancel`（停止）、`session/selectModel`。
- **接口描述来源**：每个包构建后生成 `lib/typert.remote-client.js`，里面有全部接口描述和 zod schema；会话事件的 schema 在 `docs/persistence-schema.json`。可以据此**生成 Go 类型和调用代码**，不手写 140 多个接口。
- **设置可以通用渲染**：`settings/describe` 返回每个命名空间的 schema、当前值和修订号，`settings/update` 写回。大部分设置页可以由一个通用的 schema 表单渲染器生成。

### 3.2 web 界面规模

- 76 个客户端包约 17.4 万行，加上其他目录约 3.7 万行，共约 21 万行 TypeScript。
- 全部界面都是运行时加载的 Cordis 插件，通过 116 个插槽组合。
- 设置共 15 个分区：account、general、appearance、interface、models、plugins、skills、market、remote、mcp、agent-presets、pet、about、usage-stats、remote-workspace。
- 桌面自有插件的界面（dsh-usage-panel、dsh-remote、dsh-im、dsh-whale、dshbot）也是 JS 包，要逐个移植。
- 最难的三块：富文本输入框（Lexical）、流式 Markdown + 代码高亮 + 数学公式、终端。

### 3.3 Electron 主进程

- harness 启动命令：
  `node.exe <harness>\apps\cli\lib\bin.js web [--skip-user-plugins] --patch <overlay.yml>... --host 127.0.0.1 --port <port> --no-open`，
  环境变量 `DSH_HOME=<userData>\dsh-home` 等，启动前生成 11 个覆盖文件，并把 `vendor/dsh-*` 用 junction 链接到 `dsh-home\profiles\web\node_modules`。
- Git、工作区文件、编辑器打开、终端（node-pty）、浏览器预览都在主进程里实现，原生版直接用 Go 实现，不经过 harness。
- `credentials.json` 在有 safeStorage 时是加密信封，Go 要实现 Windows 上的 DPAPI + AES-256-GCM 解密才能读取。

### 3.4 MyGo `ui` 能力

| 需求 | MyGo 现状 | 要做的 |
| --- | --- | --- |
| 布局、侧栏、分栏、标签、菜单、弹窗、提示、表单 | 齐全 | 直接用 |
| 聊天列表 | `ui.List` 虚拟化、行高可变、`FollowEnd` 贴底、向前加载不跳动 | 直接用 |
| 富文本、选择复制、中文和表情、输入法 | 齐全（DirectWrite、IMM32） | 直接用 |
| 大文件编辑 | `TextBuffer` 支持 10 万行 | 行号、查找替换、高亮要自己做 |
| 终端 | terminal 插件：ConPTY + libghostty-vt | 外面包一层标签/面板 |
| 托盘、单实例、通知、全局快捷键、深链接 | 齐全 | 直接用 |
| 更新器、NSIS 安装包、签名 | 齐全，支持增量更新 | 换掉 electron-updater 和 blockmap |
| Markdown | 没有可导入的实现 | 用 goldmark 解析，映射到 `RichText` / `Grid` / `Image` |
| 代码高亮 | 只有着色接口 | 用 chroma 分词 |
| Diff | 没有 | 用 Go diff 库 + 虚拟列表 |
| 图表、热力图 | 只有 `Painter` 绘图 | 自己画 |
| 动图 GIF | 只解第一帧 | 自己解帧 |
| 内嵌浏览器 | 原生窗口里不能嵌网页 | 用无边框 WebView2 子窗口盖在面板位置上 |
| 透明桌宠窗口 | Windows 上原生界面窗口可能不能全透明（未实测） | 先做实测，不行就给 MyGo 打补丁或用分层窗口自绘 |
| Live2D / ONNX | 没有 | 难度最高，单独评估 |

## 4. 总体架构

```
whaleisle.exe (Go, MyGo 原生界面)
├── app/        生命周期、单实例、托盘、菜单、窗口
├── ui 层       启动器、启动页、主框架、对话、设置、工作面、宠物
├── 状态层      各领域的 store：工作区、会话、对话流、设置、Git…
├── dsh 客户端  RPC、WebSocket 多路复用、$events、代码生成的类型
├── 桌面能力    Git、工作区文件、终端、预览、市场、导入、更新…
└── harness     生成覆盖文件、junction、启动并监管 node 子进程
        │ loopback HTTP + WebSocket
        ▼
node.exe vendor/deepseek-harness/apps/cli/lib/bin.js web  (dsh web)
```

线程规则：网络和磁盘操作在后台 goroutine，结果通过 `win.Update(fn)` 回到主线程改状态。助手流式输出按 16–33 ms 合并一次刷新，不逐 token 刷新。

### 4.1 Go 包规划

| 包 | 内容 |
| --- | --- |
| `cmd/whaleisle` | `main`：组装各模块 |
| `internal/paths` | userData、dsh-home、运行时目录 |
| `internal/config` | `config.json`（补全全部字段）、`credentials.json`（含 DPAPI 解密） |
| `internal/harness` | 定位 vendored harness、Node、覆盖文件、junction、启动、监管、换 cookie |
| `internal/dsh` | RPC 客户端、WebSocket 多路复用、`$events` 与 waterfall 应答、重连 |
| `internal/dsh/api` | 由 `typert.remote-client.js` 生成的 Go 类型与方法 |
| `internal/dsh/events` | 会话事件类型（由 `persistence-schema.json` 生成） |
| `internal/store/*` | 工作区、会话列表、对话流、审批、设置、模型等状态 |
| `internal/desktop/*` | `workspacefs`、`git`、`pty`、`preview`、`market`、`importer`、`plugins`、`update`、`remote`、`pet` |
| `internal/ui/theme` | 设计令牌（对应 `--dsw-*`）、明暗主题、壁纸 |
| `internal/ui/widgets` | Markdown、代码块、Diff、图表、图标等共用组件 |
| `internal/ui/<screen>` | `boot`、`launcher`、`frame`、`sidebar`、`chat`、`composer`、`settings`、`surfaces`、`terminal`、`git`、`usage`、`pet` |
| `tools/gen-api` | 接口代码生成器 |

## 5. 模块顺序与里程碑

每个里程碑都要能编译、有测试（`go test` + `ui.NewTester`），并能实际启动验证。

### M0 基础设施

- 把快照里的 `vendor/deepseek-harness`、`vendor/dsh-*`、`vendor/dshbot` 复制到本项目的 `runtime/vendor/`（`reference/` 保持只读），按原仓库 `scripts/setup-harness.js` 的流程在副本里构建（pnpm install + `build:official`），确认 `apps/cli/lib/bin.js` 和 `apps/web/dist` 存在。源码不改，只产生构建产物。
- 接口代码生成器：读取构建产物里的 `typert.remote-client.js`，生成 `internal/dsh/api`。
- 会话事件类型生成。
- 项目脚本：构建、测试、运行。

### M1 底座

- `cmd/whaleisle`、单实例锁、`paths`、补全 `config`、`credentials`（DPAPI 解密）。
- 主窗口（无边框 + 自绘标题栏）、托盘、关闭到托盘、开机启动。
- 主题令牌、明暗切换、中英文文案框架。

### M2 Harness 生命周期 + 启动页

- `launch.go` 改为 vendored harness：定位 Node 和 harness 根目录、生成 11 个覆盖文件、建立 junction、设置环境变量。
- 监管：就绪检测、自动重启、跳过用户插件的恢复路径、`workspace/create` 登记。
- 原生启动页：状态、日志、插件加载进度、重试、导出日志。

### M3 dsh 客户端

- RPC、二进制结果、错误码。
- WebSocket 多路复用、心跳、退避重连。
- `$events`：通知分发、waterfall 应答（没人处理时回答 `next`）。
- 用一个真实运行的 harness 做集成测试。

### M4 对话最小可用版

- 主框架：左侧栏 / 对话区 / 右栏 / 底栏的布局和可调宽度。
- 侧栏：工作区与会话树、新建、重命名、置顶、归档、搜索。
- 对话流：虚拟列表、历史快照与向前翻页、流式 Markdown、代码高亮、思考过程、通用工具卡片。
- 输入框：多行输入、发送、停止、排队 / 插话模式、模型选择、权限预设。
- 审批面板、提问卡片。

### M5 对话完整度

- 各类工具卡片：bash、read、search、web、文件修改、todo、ask-question、子代理。
- Diff 视图、图片附件与大图查看、交付物、计划 / 目标 / 队列 / 待办面板。
- 消息编辑与反馈、压缩卡片、用量标签、连接状态横幅、提示框。
- 数学公式（KaTeX 的替代方案）。

### M6 输入框增强

- `@` 提及、`/` 命令、技能菜单、行内标签、拖放文件、Agent 预设、目录选择器。

### M7 工作环

- 右栏停靠与标签、文件树与编辑器、底栏终端（terminal 插件）、Git 标题栏（分支、提交、推送、拉取、PR）、Diff 工作面。
- 工作区路径权限检查（对应 `workspace-authority.js`）。

### M8 设置

- 通用 schema 表单渲染器（基于 `settings/describe`）。
- 专用页面：通用、外观、界面、模型、插件、技能、MCP、Agent 预设、账号与登录、关于。

### M9 桌面扩展

- 浏览器预览（WebView2 子窗口）、文档预览（PDF / XLSX / CSV）、插件市场、用量统计（图表与热力图）、远程与 IM 渠道、SSH 远程工作区、定时任务、壁纸与背景效果。

### M10 鲸鱼娘助理与 dshbot

- dsh-whale 助理、dshbot 页面。
- 第三方插件界面的 webview 兜底。
- 桌宠暂不做（见第 10 节）。

### M11 外围与发布

- 启动器（更新询问、版本、导入、插件排查）、任务保护（退出 / 更新前检查运行中的任务）、手机远程。
- MyGo 更新器、NSIS 安装包、打包 harness 运行时与 Node、发布流程。

## 6. 主要风险

| 风险 | 影响 | 应对 |
| --- | --- | --- |
| 工作量 | 约 21 万行界面代码要用 Go 重写 | 按里程碑推进，M4 之后就能日常对话 |
| 富文本输入框 | 提及标签、粘贴、历史等交互复杂 | M4 先用纯文本，M6 基于 `TextInputClient` 自建 |
| 流式 Markdown 性能 | 长对话卡顿 | 按消息缓存解析结果，只重排正在流式输出的那条 |
| 内嵌浏览器 | 子窗口的焦点和层级 | 只用于预览面板，单独测试 |
| 第三方插件界面 | 市场插件自带的 JS 界面无法原生运行 | 设置优先用通用表单；其余用 webview 兜底（M10） |
| 覆盖安装 Electron 版 | 安装目录、快捷方式、卸载项不一致 | M11 实测从 Electron 版升级 |
| harness 版本变化 | 接口和事件结构变化 | 接口代码由生成器产出，升级 harness 后重新生成 |
| 数据兼容 | 与 Electron 版共用数据目录 | 保留未知字段；两个版本不能同时运行 |

## 7. 兼容约束

- userData：`%APPDATA%\Deepseek-Harness-Desktop`；dsh-home：`userData\dsh-home`；不读官方 `~/.dsh`（导入除外）。
- `config.json` 写回时保留所有未知字段；写入方式为临时文件 + 重命名。
- `config.json` 以快照方式持有：读取返回深拷贝，写回基于独立候选值；`Open` 记录磁盘基线，`Update` 在回调前与替换前各比较一次，不一致返回 `ErrConflict`，不覆盖、不合并、不重载。
- 该冲突检测与"比较-替换"只在同一进程内串行（进程级提交互斥，回调不在互斥内），不是跨进程原子 CAS：同时运行的 Electron 版仍可能在最终比较与重命名之间写入，两版同时运行未验证。
- `credentials.json` 按文档结构分类加载：带信封标记的文档必须是受支持且结构完好的信封，否则报错拒绝，绝不回退当明文读；缺失文件返回零值。
- 覆盖文件、junction、`disabledPlugins` 的处理与 Electron 版一致，切换版本后插件状态不乱。

## 8. 验证方式

- 单元测试：`go test ./...`（`internal/fake` 风格的假后端 + `ui.NewTester` 测界面）。
- 集成测试：真实启动 vendored harness，跑一遍新建会话、发消息、审批、停止。
- 每个里程碑结束时实际启动应用检查。

## 9. 跟进原项目

原项目仍在开发。快照基线是提交 `26205a89f`（见 `reference/README.md`）。每次开始新里程碑前：

1. 在原仓库运行 `git diff 26205a89f --stat`，看变化涉及哪些模块。
2. 与当前里程碑相关的改动，同步到计划和实现。
3. 需要时重新拍快照，并更新 `reference/README.md` 的基线提交。

## 10. 已确认的决定（2026-10-09）

1. **网页预览**：浏览器预览和文档预览（PDF 等）允许用 WebView2 窗口，其他界面一律原生。
2. **第三方插件界面**：允许用 webview 兜底显示。做法是在 webview 里加载 harness 提供的插件包，只显示该插件挂载的插槽；放在 M9 之后。
3. **桌宠**：暂不做，M10 只做 dsh-whale 助理和 dshbot 页面；`config.json` 里的 `pet` / `live2dPet` 字段原样保留。
4. **版本关系**：替换升级。沿用 appId `ai.deepseek.harness.gui`、产品名 Whale Isle 和数据目录 `Deepseek-Harness-Desktop`；新版安装包要能覆盖安装 Electron 版。

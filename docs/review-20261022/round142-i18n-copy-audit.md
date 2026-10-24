# Round142 · 聊天/输入框/设置文案漂移审计（G52）

本轮对聊天记录、输入框（composer）与设置表面的可见文案做了一次可复核的差集扫描，修掉证据确凿的错译/漏译，并列出无法安全判定的剩余项。只改 `internal/ui/i18n/catalog_*.go`，`cmd/` 的内联中英文只登记不改。

## 1. 方法

- 原生侧：解析安装版同源的 61 个 client locale 文件（`reference/electron/vendor/deepseek-harness/packages/client/*/src/client/locales*.ts`），得到 3039 条词条、1093 个 zh/en 键对；关键条目与安装版 `ui-conversation/lib/client.js` 逐字核对。
- MyGo 侧：解析 `internal/ui/i18n/catalog_*.go` 的单行 `Register(...)`，共 908 条（总 Register 数 1518，其余为多行/转义或动态注册；`catalog_tool_rows.go` 的 56 条动态标题单独核对）。另扫常量-注册一致性（0 个常量未注册）与动态键路径（`chat.tool.row.title.<name>`，56 个全部注册）。
- 匹配：① 键名归一（丢弃 1–3 段命名空间、去掉分隔符、把 `{x}` 与 `%d/%s` 视作同一占位符），只在 chat/composer/settings 相关包内比对，要求 harness 键至少两段；② 文案值配对（忽略大小写）用于键名被重命名的条目。
- 判定：zh 与 en 都逐字相同才算 matched；占位符写法差异（`%d` vs `{count}`）视为等价，不计漂移。

## 2. 计数

| 项 | 数量 |
|---|---|
| 原生词条 / 键对 | 3039 / 1093 |
| MyGo 解析词条（单行） | 908（chat 266、composer 94、surfaces 84、tool_details 162、terminal 11、context 14、misc 603、w4 284 中的可解析部分） |
| 键名对齐对（chat/composer/settings 相关包） | 91 |
| └ 逐字一致 | 89 |
| └ 有差异 | 2（均修复） |
| 文案值精确键对 | 164 |
| 值匹配候选（需人工判定） | 66 |
| 工具行标题逐条核对 | 56 / 56 一致（zh+en） |
| 本轮修复 | 10 个键 |
| 常量未注册 / 动态键缺失 | 0 / 0 |
| 剩余项 | 见 §4 |

## 3. 修复清单（before → after）

| MyGo 键 | 侧 | before | after | 原生依据 |
|---|---|---|---|---|
| `chat.tool.detail.row.input` | en | `Input` | `IN` | ui-conversation `row.input` en = "IN"（zh 输入不变） |
| `chat.tool.detail.row.output` | en | `Output` | `OUT` | ui-conversation `row.output` en = "OUT"（zh 输出不变） |
| `chat.usage.tokens` | zh / en | `tokens` / `tokens` | `Token` / `Tokens` | ui-trajectory `usage.tokens`（zh Token、en Tokens） |
| `surfaces.usage.tokens` | zh / en | `tokens` / `tokens` | `Token` / `Tokens` | 同上（用量表面同一词条） |
| `chat.usage.cache_read` | en | `Cache read` | `Cached` | ui-trajectory `usage.cached` en "Cached"（zh 缓存读取不变） |
| `chat.usage.cache_write` | en | `Cache write` | `Cache created` | ui-trajectory `usage.cacheCreated` en "Cache created"（zh 缓存写入不变） |
| `chat.tool.ask` | en | `Ask` | `Ask question` | ui-conversation `ask.rowTitle` en "Ask question"（zh 提问不变） |
| `composer.drop.title` | en | `Drop files or images here` | `Drag files or images here to add them` | ui-conversation `attachment.dropTitle`（zh 不变） |
| `chat.turn_rail.label` | zh | `回合` | `轮次` | ui-trajectory `toolbar.turns` zh 轮次（en Turns 不变） |
| `chat.turn_rail.turn` | zh | `回合` | `轮次` | ui-conversation `detail.field.turn` zh 轮次（en Turn 不变） |

工具行标题（`chat.tool.row.title.*`，56 条）与原生 `tool.title.*` 的 zh/en 全部逐字一致，未改；`todo`/`ask` 两条在原生 `tool.title.*` 中没有对应项，保留 MyGo 现值。

## 4. 剩余项与归属

- **`cmd/` 内联中英文**（未改，仅登记）：42 个文件、254 处 `resumeText("中文", "English")`，例如 [account_bonus_notice.go:74](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/account_bonus_notice.go>)、更新/托盘/菜单文案。这些不属于 `internal/ui/i18n`，需要单独一轮把它们迁进目录词条；本轮只报告。
- **设置页只做了键名对齐子集**（91 对里 89 对一致）。设置键大量是 WhaleIsle 自有命名（`settings.desktop.*`、`settings.interface.*`），与 harness `ui-settings-*` 的键名不一一对应，值匹配也没有出现确凿错译；完整逐页对照仍待后续。
- **无法安全判定的候选**（保留现值，避免按错误上下文改动）：
  - `composer.commands`（zh 命令列表）用于 composer「+」按钮的标签/tooltip，原生该按钮不在 `section.commands` 语境；
  - `chat.turn.failed`（zh 运行失败）用于轮次用量行的失败态，原生 `row.failed` 只是 失败/Failed，语义可能不同；
  - `composer.voice.later`（zh 稍后）对应语音设置窗的「稍后」，与原生更新窗 `update.later` 不是同一表面；
  - `chat.image.enlarge`（Enlarge）是图片灯箱，原生 `zoomIn` 属文档预览；
  - `composer.cost.estimate`（en Session estimate）在 MyGo 里没有非测试引用（未使用键），原生 `sessionCost.title` en 为 "Session cost"；若后续启用，按原生改。
- 占位符写法（`%d/%s` 对 `{count}/{interval}`）是 MyGo 的 fmt 约定，按等价处理，不算漂移。

## 5. 回归

- `go test ./internal/ui/i18n ./internal/ui/chat ./internal/ui/composer -count=1` → i18n 0.551s、chat 66.298s、composer 8.404s，全部 ok（chat 的一处旧断言 `Input` 组标题已同步改为原生 `IN`，`TestToolDetailsModelInspectionProjection`）。
- `go vet ./internal/ui/i18n` → 退出 0。
- 未构建 exe、未提交；未改 `cmd/`、frame、boot 与安装版。

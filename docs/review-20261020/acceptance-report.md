# MyGo鲸屿 · 代码 + 实机验收报告（阶段版，2026-10-21）

> 结论先行：**尚未达到与原版完整对齐**。设置/插件/任务/鲸鱼娘设置/远程设置等页面已经过实机操作；核心模型链路（流式、工具、权限、鲸鱼娘聊天）因本机没有 `DEEPSEEK_API_KEY` 无法实机验收；IM/Office 因 dsh-im 未安装、桌面锁屏导致鼠标类和拖动类操作不可达。以下只把“真跑过/真测过”的项算 PASS。

## 1. 验收方法

- 实机：本机运行 `whaleisle`（2160×1380 @150%），用 Windows UI Automation + SendInput/PrintWindow 驱动真实窗口；证据存 `docs/review-20261020/evidence/acceptance/`。
- 代码：对照安装版 vendor 的 `packages/client/ui-*` 与 Host 生成的 RPC/事件类型，检查调用链、空态、失败路径，并跑对应测试。
- 矩阵：`docs/review-20261020/acceptance-matrix.md`（操作步骤/预期/实际/证据）。

## 2. 通过率（28 个验收项）

| 状态 | 数量 | 说明 |
| --- | --- | --- |
| PASS | 16 | 其中约 12 项有实机证据；其余为代码级（真实 RPC/接线 + 测试） |
| PARTIAL | 5 | C3 环境变量提示、C4 MCP tools/call/依赖安装、G1/G2/G3 视觉 |
| BLOCKED | 6 | A3/A4（无 API key）、A7（锁屏下 UIA 搜索失败）、E2（无 API key）、F2/F3（dsh-im 未装） |
| PENDING | 1 | E3 桌宠拖动/喂食（锁屏，鼠标不可达） |

按项通过率 **16/28 ≈ 57%**；若只算“实机可复现 PASS”约 **12/28 ≈ 43%**。这不是完整对齐，剩余项见第 5 节。

## 3. 实机已通过（摘录）

| 项 | 实机结果 | 证据 |
| --- | --- | --- |
| A1 新建会话 | 侧栏新增并选中，主区回 hero，可输入 | `A1-new-session.png` |
| A2 发送/失败提示 | 消息上屏；无 key 时显示「还没有配置 API Key」+「打开模型设置」+「详情」，原始英文只在详情 | `A2-failure-card.png` |
| A2 失败卡动作 | 点「打开模型设置」进入设置·模型页 | 实机 UIA |
| A5 复制/编辑 | 剪贴板收到全文；编辑回填草稿、可取消 | `A5-copy-message.png`、`A5-edit-message.png` |
| A7 列表/切换 | 会话列表与切换可用（搜索框无法驱动） | `A7-search-sessions.png` |
| B1 无工作区文件栏 | 不再 404，显示「此会话没有工作区文件夹」 | `A2-failure-card.png` 右侧 |
| B3 终端抽屉 | 打开真实 PowerShell，提示符在当前 cwd；菜单齐全 | `B3-terminal.png` |
| C1 设置导航 | 12 个分区逐页进入并渲染；「市场」按决定隐藏 | `settings-live/*.png` |
| C4 插件页 | 真实 inventory、内置/实验性卡片、开关 | `C4-plugins.png` |
| C5 关于/更新 | 版本、来源、复制版本；更新目录 404 只显示友好文案 | `settings-live/about.png` |
| D1 自动化任务 | 空态 + 新建入口；删除/刷新/历史已接真实 store | `D1-schedule.png` |
| E1 鲸鱼娘设置 | 人设/记忆/MEMORY.md/名字称呼/技能，真实 Host 路径 | `E1-whale.png` |
| F1 远程设置 | 中继/端口/配对码/连接方式真实表单 | `F1-remote.png` |
| G1 首屏 | 无遮挡 hero + composer + 侧栏 | `G1-first-screen.png` |

## 4. 验收期间修掉的真实问题

1. 失败回合 `turn/end` 只显示「运行失败」，原始错误藏在展开里 → 统一为本地化可操作失败卡，原始错误只在详情。
2. 失败卡「打开模型设置」回调只在导航模型上，聊天里是 nil → 同时接到主 frameModel。
3. 设置→远程→返回触发 `panic: ui: a ListState shown by two Lists`（SectionContent 回调读实时 section，被 OnClose 清空后重画聊天）→ 捕获 section 传入回调，补回归测试。
4. 无工作区会话右侧文件栏请求 `dsh-remote resolve-mirror` 404 → 真实空态。
5. 设置导航残留「市场」；插件页误报「没有匹配的插件」。
6. 外观主题网格卡片在 Grid 里用 `FillWidth`，卡片按整行宽重叠/裁切 → 移除，补几何测试。
7. 关于页打开即弹更新框、并显示 `not_configured` / HTTP 404 原文 → 去掉自动检查，更新状态改产品文案。
8. 通用设置裸值 `workspace-write`/`zh`/`completion`/`sidebar`/`queue` → 本地化显示并映射回原值。
9. 设置面板纯白 255 → raised surface 245（参考 235-240）。
10. 终端链接打开的文件缺 `OnSaveEdit`（树里能存、终端打开不能存）→ 接真实写盘 + 越界测试。
11. IM 发送失败仍显示已送达 → pending/sent/failed/local/unknown。
12. 定时任务删除/刷新/历史是不可达按钮 → 接真实 store。
13. MCP 已连接但零工具时静默空行 → 显示「没有工具」。
14. 插件页添加按钮旁的 ⌄ 用 PrimaryButton 渲染成独立蓝色方块 → 改普通按钮。

## 5. 剩余缺口与外部条件

| 缺口 | 原因/需要 |
| --- | --- |
| A3/A4/E2 模型流式、工具卡、权限确认、鲸鱼娘聊天 | 无 `DEEPSEEK_API_KEY`；填入后可直接复验 |
| F2/F3 IM/Office | dsh-im 未安装；需在插件页安装并重启 Host |
| A7 会话搜索 | 桌面锁屏，UIA 写搜索框会卡死；需解锁桌面后手工/UIA 复验 |
| E3 桌宠拖动/喂食 | 锁屏导致鼠标拖动不可达；需解锁桌面 |
| C3 环境变量橙色提示 | 需 Host 返回真实的 env 探测结果；当前只显示凭证未验证状态 |
| C4 MCP tools/call、插件依赖安装 | 仓库没有对应 RPC，不能伪造 |
| G2 设置页像素级 | 12 页同源截图已齐，仍需逐页像素对比 |
| G3 指针特效/滚动动效 | 静态截图无法定格，需录屏或人工步骤 |
| 市场 | 按用户决定隐藏入口，仅保留官方站 + 已安装/更新 |

## 6. 运行与验证

- 全量测试：`go test ./internal/ui/... ./internal/store/... ./internal/desktop/whale ./cmd/whaleisle` —— 通过（composer WebP 附件测试在并发下偶发超时，单跑通过）。
- 当前构建：`whaleisle-pass30.exe`，pid 21864（本轮结束时可能已由后续构建替换）。
- 证据索引：`docs/review-20261020/evidence/acceptance/`、`settings-live/`。

## 7. 补充（pass35）

- C2 设置持久化由代码级升级为**实机跨重启 PASS**：关闭「显示代码工作视图」后 Host profile 写入 enabled: false，重启后开关仍为 Off，再恢复 On。

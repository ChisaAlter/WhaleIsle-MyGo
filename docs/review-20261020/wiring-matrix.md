# MyGo 鲸屿 · 静默无效点击第二轮审计（Pass 8-A · 接线矩阵）

- 审计日期：2026-10-21
- 性质：只读复核（未改代码、未跑 git 写命令）
- 范围：`internal/ui/**`（27 个含回调的页面包）、`cmd/whaleisle`、`internal/ui/frame`、`internal/store/**`
- 上一轮（`dead-ui-audit.md`）已修项不重复列；本文件只列**仍然存在**的缺口，并给出"为什么现在扫描结果是干净的"的可复核证据。

## 0. 结论摘要

| 类别 | 结果 |
|---|---|
| 生产 Model 里**零赋值**、但 UI 会渲染/可点的回调 | **0**（复核见 §2；上一轮的 5 条已分别被 task-54/55/57 修掉或证伪） |
| **空函数**回调（`OnX: func(...) {}`） | **0**（全仓非测试代码扫描，见 §3） |
| 回调已赋值但**实现会静默 return**（store 未就绪） | **18 处**，全部在 `cmd/whaleisle/main.go`（见 §4） |
| 回调已赋值但**错误被吞掉**（`if err != nil { return }`，无提示） | **2 处**（调度保存，task-55 在建） |
| 回调已赋值但**只在特定布局下返回非 nil**（条件接线） | **6 个 helper**，其中 inspector 一类已由 task-57 补兜底（见 §5） |
| 死视图（定义了但从不挂载）造成的"回调永不为真" | 1 组（旧消息反馈行，见 §6） |

> 一句话：**静态层面的"零赋值"已经基本清零**；用户仍会遇到的"点了没反应"，主体变成了 §4 的 **nil-store 静默 return** 和 §5 的 **条件接线**，以及渲染层未实现的 **stub 页面**（见 `dead-ui-audit.md` A4/§B）。继续静态扫描收益已很低，建议下一轮改为**点击埋点**（§8）。

## 1. 复核方法（可重跑）

1. 对每个 `internal/ui/<pkg>`，抽取 View 里出现的所有 `(m|a|v|p|cm|s|w|t|d|r|row|opts|model|cfg|git).On*` 字段。
2. 在**全部生产代码**（`cmd` + `internal`，排除 `_test.go`）中查该字段是否有赋值：`Field:` 或 `.Field =`。
3. 无赋值 → 再区别它是"渲染即静默"还是"nil 时隐藏/禁用"。
4. 单独扫：`OnX: func(...) {}`（含跨行空体）、`(*Store) Method(...) {}`（空实现）、`if err != nil { return }`（吞错）、`if <store> == nil { return }`（静默跳出）、`func(...) func(...)` 里提前 `return nil`（条件接线）。

## 2. 接线矩阵：按页面

| 页面（包） | UI 使用的回调字段 | 生产零赋值 | 备注 |
|---|---:|---:|---|
| settings | 106 | 0 | 全部在 `cmd/whaleisle/settings_*.go` 赋值 |
| frame | 94 | 1 | `OnSubmitComposer` 空置（死字段，发送走 `OnSubmitGesture`，见 §6） |
| projects | 63 | 0 | — |
| assistant | 60 | 0 | — |
| composer | 55 | 0 | — |
| chat | 47 | 2 | `OnCancelDown`/`OnOpenDown` 属旧反馈行死视图（§6） |
| sidebar | 44 | 0 | — |
| launcher | 29 | 0 | — |
| remote | 19 | 0 | — |
| chrome | 16 | 0 | — |
| market | 14 | 0 | — |
| dirpick | 13 | 0 | — |
| files | 13 | 0 | — |
| custompanel | 12 | 0 | — |
| schedule | 10 | 0 | task-55 已接 |
| agents | 10 | 0 | task-55 已接 |
| usage | 9 | 0 | — |
| browserpreview | 9 | 0 | `OnOpenExternal` 由 task-55 接；`OnHide/OnShow` 是窗口事件不是 Model 回调（误报） |
| webview | 6 | 0 | — |
| terminal | 5 | 0 | — |
| pluginmanager | 4 | 0 | — |
| boot / im / interaction / sessioncost / workspacepicker / diffsurface | 1–4 | 0 | — |

**结论**：27 个页面包中只有 3 个包存在零赋值字段，合计 5 个，且没有一个是"可点但静默"的用户控件（逐条见下）。

| 字段 | 使用点 | 结论 | 处理建议 |
|---|---|---|---|
| `browserpreview.OnOpenExternal` | `browserpreview/panel.go:75` | 曾被确认为缺失；task-55 正在接 `mygo.Shell.OpenExternal` | 跟踪 task-55，完成后本条关闭 |
| `browserpreview.OnHide` / `OnShow` | `browserpreview/floating_document.go:127/132` | **误报**：`w.OnHide(func(){...})` 是 mygo 窗口事件 API，不是 Model 回调 | 无需处理 |
| `chat.OnCancelDown` / `OnOpenDown` | `chat/feedback.go:22/21`（`feedbackRow`） | 旧反馈行实现，**从未挂载**；现役反馈是 `feedback_editor.go` + `message_actions.go:130` | 删除旧实现（§6） |
| `frame.OnSubmitComposer` | `frame/frame.go:211` → `frame.go:719` 传给 composer | **死字段**：发送按钮先判 `OnSubmitGesture`（`composer.go:566`），而它由 `OnSubmitComposerGesture` 接线；`OnSubmit` 分支永远走不到 | 删除字段或补一句注释，避免下轮再被误判 |

## 3. 空函数回调：0

扫描口径：`On*: func(...) {}`（单行）+ 跨行空体（`On*: func(...) {` 紧跟 `}`）；另扫 store 侧空方法体 `func (s *Store) X(...) {}`。

结果：**0 处**。也就是说"接线了但什么都不做"的情况不存在；能点却无效果的只有 §4 的运行时静默跳转。

> 反例记录（避免下一轮误报）：`composer.go:564-581` 的发送按钮是**四分支链**（`handleModelCommand` → `OnSubmitGesture` → `OnSubmit` → `OnSend`），`OnSubmit` 为 nil 不代表发送失效——实际走 `OnSubmitGesture`（`cmd/whaleisle/main.go:6491`）。

## 4. 已接线但会静默 return（真正的"点了没反应"）

全部位于 `cmd/whaleisle/main.go`，形态都是 `if <store/client> == nil { return }`，**无 toast、无日志、无禁用态**。

| # | 行号 | 回调 | 静默条件 | 用户感知 | 页面 |
|---|---|---|---|---|---|
| 1 | `main.go:3502` | `OnSaveLauncherPrefs` | `cfgStore == nil` | 启动器偏好保存无效 | 启动器 |
| 2 | `main.go:3721` | `OnEnableSkipUserPlugins` | `cfgStore == nil` | 跳过用户插件无效 | 启动器 |
| 3 | `main.go:3749` | `OnClearPluginRecovery` | `cfgStore == nil` | 清除插件恢复标记无效 | 启动器 |
| 4 | `main.go:4221` | `OnAckBonus` | `usageStore == nil` | 奖励确认无效 | 账户/用量 |
| 5 | `main.go:4234` | `OnRefresh` | `usageStore == nil` | 刷新无效 | 账户/用量 |
| 6 | `main.go:4388` | `OnRefreshAnalytics` | `client == nil` | 分析刷新无效 | 插件/分析 |
| 7 | `main.go:5166` | `OnSaveTitle` | `schedStore == nil` | 调度改名无效 | 调度 |
| 8 | `main.go:5180` | `OnSavePrompt` | `schedStore == nil` | 调度改提示词无效 | 调度 |
| 9 | `main.go:5380` | `OnToggleWorkspace` | `state.cfgStore == nil` | 远程工作区开关无效 | 远程设置 |
| 10 | `main.go:5391` | `OnTogglePairing` | `state.cfgStore == nil` | 配对开关无效 | 远程设置 |
| 11 | `main.go:6122` | `OnOpenSession` | `sessions == nil` | 打开会话无效 | 侧栏/搜索 |
| 12 | `main.go:6454` | `OnPrices` | `usageStore == nil` | 价格页打不开 | 用量 |
| 13 | `main.go:6652` | `OnNoDirectory` | `sessions == nil` | "无目录"引导无效 | 首页/侧栏 |
| 14 | `main.go:6680` | `OnRenameWorkspace` | `sessions == nil` | 重命名工作区无效 | 侧栏 |
| 15 | `main.go:6694` | `OnDeleteWorkspace` | `sessions == nil` | 删除工作区无效 | 侧栏 |
| 16 | `main.go:6708` | `OnMoveSessionBefore` | `sessions == nil` | 拖拽排序无效 | 侧栏 |
| 17 | `main.go:6796` | `OnStartComposerVoice` | `sessions == nil` | 语音按钮无效 | 输入区 |
| 18 | `main.go:6825` | `OnCycleComposerSpeech` | `sessions == nil` | 语音循环无效 | 输入区 |

**误报排除**：`OnSaveTitle/OnSavePrompt` 的 `err != nil { return }`（`main.go:5171/5185`）是 task-55 正在补错误提示的点；`_ = cfgStore.Update(...)` 3 处（`main.go:2825/5383/6437`）是配置写入丢弃错误，属低风险但同样静默。

**建议**：这 18 处统一改为「不可用 → 就地提示」，或在 store 未就绪时把控件置灰并给 tooltip（比"点了没反应"更诚实）。最小改动：加一个 `state.warnStoreNotReady("usage")` toast helper。

## 5. 条件接线（返回非 nil 需要额外条件）

| helper | 位置 | 何时返回 nil | 现状 |
|---|---|---|---|
| `inspectToolCallNavigation` | `frame/tool_call_navigation.go:9` | `HideChatViewTabs` / 无 `ChatPaneTab` / 无 `TrajSelectedID` | task-57 已加会话日志兜底（`inspectActionFor`） |
| `toolCallInspector` | `frame/session_inspector_pane.go:66` | `DisableDeveloperTools` 或无 pane 宿主 | 兜底自身 |
| `inspectCallHandler` | `chat/chat.go:521` | 上层回调为 nil | 正确（nil 时按钮隐藏） |
| `FileMentionResolver` | `chat/file_mentions.go:42` | 无匹配记录 | 正确（仅影响链接解析） |
| `pinAction` / `archiveAction` | `sidebar/session_menu.go:44/57` | `OnPin`/`OnArchive` 为 nil | 正确（菜单项由调用方跳过） |

**结论**：只有 `inspectToolCallNavigation` 属于"用户可见的布局相关失效"，且已修。其余是设计意图。

## 6. 死视图 / 死字段（不修也不影响点击，但会误导审计）

| 项 | 位置 | 说明 |
|---|---|---|
| 旧消息反馈行 | `chat/feedback.go:12-79`（`feedbackRow` 全仓 0 引用） | 字段 `OnVote/OnOpenDown/OnCancelDown` 随之成为"零赋值"；现役反馈在 `feedback_editor.go:17` + `message_actions.go:130` |
| `frame.OnSubmitComposer` | `frame/frame.go:211`、`:719` | 发送链路走 `OnSubmitComposerGesture`，该字段空置 |
| `paintRightTab` / `paintSourceActions` / `paintReferenceButtons` / `accountDetailsPane` / `projectFilterControls` / `mcpRowDetail` 等 | 见 `dead-ui-audit.md` §A4 | 上一轮已列，仍未挂载 |

## 7. 优先级

### P0（用户能看到、能点、但当前会静默失败）

1. **调度改名 / 改提示词**（`main.go:5166/5180` + 5171/5185 吞错）——task-55 在建，需同时补"失败提示"。
2. **远程工作区 / 配对开关**（`main.go:5380/5391`）——`cfgStore` 未就绪时开关点了不动。
3. **用量页：价格页 / 刷新 / 奖励确认**（`main.go:6454/4234/4221`）。
4. **侧栏：重命名 / 删除 / 排序 / 打开会话**（`main.go:6680/6694/6708/6122`）。
5. **输入区语音**（`main.go:6796/6825`）。
6. **启动器偏好与插件恢复**（`main.go:3502/3721/3749`）。
7. **分析刷新**（`main.go:4388`）。
8. **"无目录"引导**（`main.go:6652`）。
9. 删除死字段 `frame.OnSubmitComposer`、旧反馈行（避免下轮再误报）。
10. 统一加「未就绪」toast（一次性覆盖 18 处）。

### P1

- `browserpreview.OnOpenExternal`（task-55 完成后关闭）。
- `_ = cfgStore.Update(...)` 3 处改为记日志。
- stub 页面（远程 SSH / IM / 鲸鱼娘 / WebView / 市场）——见 `dead-ui-audit.md`，属功能缺口而非接线缺口。

## 8. 下一轮建议：从静态扫描转向运行时埋点

静态扫描已经把"零赋值"清到 0，继续扫收益很低。建议在 UI 层加一个统一入口（`internal/ui/widgets` 里已有 `stdButton` 之类封装），对点击命中以下三种情况打一条 debug 日志：

1. `OnX == nil`（回调缺失）；
2. 回调执行但 store/client 为 nil（把 §4 的 18 处 `if x == nil { return }` 换成 `click.Dropped("usage-store-not-ready")`）；
3. 回调返回 error 被吞。

跑一次真实会话（打开设置/用量/远程/调度/侧栏各点一遍）即可拿到**真实**的静默点击清单，比静态推断准确，也能直接验证本轮 §4 的判断。

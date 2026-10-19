> Integrated Windows update at `da79d7df`: the fixed original `/file`, `/model`, `/permission`, `/feedback` and 1017 model-label contributions have native adapters, owner retirement, actual Store selection/projection receipts, popup navigation and focus return. Complete ordinary Windows regression and matching main/helper builds passed; the actual read-only Host graph check has its own `d2e79d95` receipt. Source-phase and older acceptance statements below are historical. [Current checkpoint and exact limits](source-depth-integration-20261019.md#current-windows-checkpoint) retain same-DPI visual/physical input, FlipText, full-app reconnect, automatic compaction/model recovery and arbitrary third-party runtime boundaries. The earlier Creator/guide/first-send fixes remain integrated.

# Conversation 源码覆盖记录（2026-10-19）

本表按完整清单逐行登记 CHAT-01–21，但仅代表本记录者实际读过的源码范围，不代表所有原版模块均已深审，也不把另一组的截图、测试结果或历史 PASS 转记为本表的源码比较成果。功能总清单仍见 [native-full-function-acceptance-20261019.md](native-full-function-acceptance-20261019.md)。

原版只读根目录为 `C:\AI\Deepseek-Harness-Desktop`。下文 `P/` 指该目录下 `vendor/deepseek-harness/packages/client/`；`A/` 指 `vendor/deepseek-harness/packages/api/session-controller/`。Go 路径相对本仓库。列出的入口名已按源码定位；仅定位文件不等于阅读了其全部实现。本记录不读取用户会话、配置或凭据。

等级含义：

- `behavior-compared`：实际对照了行内明确列出的选定行为及其两端源码；**不是整行模块的穷尽审查**，也不是实际验收通过。
- `entrypoint-sampled`：实际抽查了行内注明的一端或两端入口、调用或状态消费代码；尚未完成整个行为的两端闭环比较。
- `not-reviewed`：本记录者仅定位入口，或未进入该模块实际契约；不得据此推断已实现、未实现或可正常使用。

## 逐项范围

| ID | 原版具体入口 | Go 入口 | 等级 | 实际阅读 / 比较范围 | 已关差异与剩余范围 |
|---|---|---|---|---|---|
| CHAT-01 | `P/ui-workspace/src/client/index.ts`、`WorkspaceBrowser.tsx`；`navigation.ts`、`pin-order.ts` 为其他相关入口 | `internal/ui/sidebar/sidebar.go`、`workspace_groups.go`；`internal/store/session/workspace_follow.go` | behavior-compared | 比较浏览器的空列表、仅归档列表及返回入口，与 Go 侧栏在空数据下的反馈；抽查当前 list/lifecycle 投影接线。 | `a4af895` 补回所选空态行为。搜索排序、分组、拖动重排、置顶全链和大列表没有由本记录者逐项完成比较。 |
| CHAT-02 | `A/src/client/sessions/manager.ts` 的 removed/deleted；`session.ts` 的 `handleRemoved`、`dispose`；`P/ui-session/src/client/session-provider.tsx` 为绑定入口 | `internal/store/session/lifecycle_events.go`、`archive_result.go`、`composer_retirement.go`；侧栏会话菜单 | behavior-compared | 本行仅比较持久删除、普通注册表移除及未退役发送的寿命；普通 removed 保留 resident/reload 语义，不等同 dispose。 | 最终 `911ff74` 仅对 durable deleted 结算同 owner 未退役附件发送；`applyDeleteResultLocked` 复用 live 生命周期入口，列表回放不再结算。创建、改名、归档、恢复、批量操作和完整分支菜单没有因此获全量覆盖。 |
| CHAT-03 | `A/src/client/sessions/session.ts` 的开关、快照替换及退役；`P/ui-conversation/src/client/skeleton/ConversationSession.tsx`、`ConversationMainPanel.tsx` | `internal/store/session/follow.go`、`history_navigation.go`、`lifecycle_events.go`；`internal/ui/frame/frame.go` | entrypoint-sampled | 阅读 Go follow/历史状态归属、原版快照替换中 admission 的片段，以及打开状态在 InputBar 的消费。 | 完整 reconnect、gap repair、history page、同会话重开和所有后台 Session 保留策略尚未逐源码比完；不能以此前实际 Host 历史显示成功替代。 |
| CHAT-04 | `P/ui-conversation/src/client/skeleton/InputBar.tsx`、`HeroShell.tsx`、`ConversationRoot.tsx` | `internal/ui/frame/frame.go`；`internal/ui/composer/composer.go`、`workspace.go`、`draft_input.go` | entrypoint-sampled | 抽查 inert 工作区触发、同一 resident editor、Hero/普通输入的挂载与父级身份；早期核对过父级 key 导致焦点丢失的路径。 | 本行未覆盖全部空白/加载/错误/首条消息过渡和所有工作区入口；本表不为先前主工作区截图签署原版全页一致。 |
| CHAT-05 | `P/ui-conversation/src/client/skeleton/InputBar.tsx` 的 DraftEditor、focus/reveal/keymap 接口；`input/editor/view-binding.ts` 是后续深入入口 | `internal/ui/composer/draft_client.go`、`draft_input.go`、`draft_navigation.go`、`draft_undo.go`；MyGo Windows text input | entrypoint-sampled | 阅读 Go UTF-16 选区、marked text、原生 editor 生命周期，以及原版 InputBar 的焦点和键盘接线片段。 | 未穷尽 Lexical keymap 与 Windows IME 的对应行为；真实候选窗口、系统剪贴板、撤销序列和 resize 后候选几何仍需独立验收。未验收不记为功能缺失。 |
| CHAT-06 | `P/ui-conversation/src/client/skeleton/InputBar.tsx`、`InputBar.module.css`、`ComposerResizeHandles.tsx`；`P/ui-attachment/src/client/ComposerAttachments.tsx` | `internal/ui/composer/attachment_display.go`、`draft_state.go`、`draft_flights.go`、`resize.go` | behavior-compared | 选定范围为零附件 rail、文件 chip / 图片 thumb 的密度、失败保留和重试、恢复草稿与手动尺寸/自然高度的分离。 | `49ad99b` 恢复主 composer 紧凑附件 rail；`40c7309` 仅纠正 decode-ready 测试观察条件。`f6b1480` 修复恢复后附件 ID 规范化被误认为用户改动。未穷尽尺寸持久化的所有拖动、DPI 和 IME 组合。 |
| CHAT-07 | `P/ui-conversation/src/client/skeleton/InputBar.tsx`、`service.ts:sendSession`；`A/src/client/sessions/session.ts` 的 submission/inbox retirement | `internal/store/session/composer_{submit,pending,retirement,gesture,queue}.go`、`queue*.go`；`cmd/whaleisle/composer_detached.go`；composer draft flights / stop | behavior-compared | 对照忙碌 Send/Stop、queue/steer 的所选输入策略、accepted 与 observed 的区分、附件 await-retirement、canceled / consumed 未 admitted、删除与已经 observed 的不可回滚结果。 | `f5c3f6d` 接附件退役结果；`911ff74` 收窄 durable deletion；`f6b1480` 修复一次性消费恢复 payload。文本发送不泛化自动恢复，不自动重发。流式模型行为、所有命令 adjudication、长队列重排及各种远端错误还未穷尽。 |
| CHAT-08 | `P/ui-input-trigger/src/client/controller.ts`、`ui-commands/src/client/directory.ts`、`ui-skill/src/client/SkillReference.tsx`、`ui-reference/src/client/FileReference.tsx`；InputBar mention-drop 入口 | `internal/ui/composer/command_launcher.go`、`suggest*.go`、`leading_claim.go`、`reference_activation.go`、`mentions.go`；session command/reference stores | entrypoint-sampled | 实际阅读 Go launcher/claim、引用激活与移除、skills 排序接线，以及原版 InputBar 的 mention drop 和菜单入口；不将原版插件文件名定位当作整个插件已审。 | Slash、@、引用 preview、补全键盘和 Host 命令裁决尚未全链比较；未发现已读片段之外的缺口不等于该行关闭。 |
| CHAT-09 | `P/ui-model-selection/src/client/{index,directory,catalog}.ts`、`ui-agent-preset/src/client/AgentPresetSeat.tsx`、`ui-permission-presets/src/client/index.ts`；InputBar model seat | `internal/ui/composer/model_picker*.go`、`model_command*.go`、`permission_command.go`、`preset_picker.go`；`internal/store/session/models_projection.go`、presets/permissions | entrypoint-sampled | 阅读 InputBar 的模型 seat 锁定理由与 Go 选择/投影/局部弹层状态；模型、预设、权限各插件只定位入口，未穷尽其调用。 | 默认值、provider/model/reasoning 变更、缺模型 owner block、风险确认、preset 全部作用域仍需完整比较；不把“有菜单”记为正常可用。 |
| CHAT-10 | `P/ui-conversation/src/client/service.ts`、InputBar intake；`P/ui-attachment/src/client/{ComposerAttachments,ImagePreview}.tsx`、`drop-events.ts` | `internal/ui/composer/attach*.go`、`file_paste.go`、`clipboard.go`；`cmd/whaleisle/attachments*.go`；session upload / attachment cache | behavior-compared | 选定范围为图片/文件 intake、文件即时准备、失败卡 retry/remove、重复 occurrence、恢复后重准备、同 owner/跨 session 收据、原版 chip/thumb 与本地 preview。 | `49ad99b` 是主输入卡密度修复；附件发送寿命另见 CHAT-07。原清单所称 `ui-file-upload` 未在原版发现同名 client 文件夹，本表使用实际 service/attachment 入口。picker/drop/clipboard 的真实 OS 手势、全部媒体格式和完整上传消费生命周期不因此全覆盖；WebP 媒体工作由独立组处理，本文不重复判缺失。 |
| CHAT-11 | InputBar 工具入口；原版 desktop voice contribution 的完整 controller 尚未由本记录者定位/读通 | `internal/ui/composer/tool_actions.go`、`voice_actions.go`、`speech_insertion.go`；`internal/store/session/speech_recording.go`；desktop voicecapture/voiceinput | entrypoint-sampled | 阅读 Go provider/限额冻结、转写回执归属、选区插入与 Escape 取消，以及原版工具入口。 | 原版录音/取消/设备错误的完整链路未比较；真实麦克风与转写未由本批验证。不能据此宣称缺实现或已通过。 |
| CHAT-12 | `P/ui-chat/src/client/{index,apply}.ts`、`ui-tool/src/client/{index,apply}.ts` | `internal/ui/chat/chat.go`、`retry_status.go`、`performance_usage.go`、message/tool renderers；MyGo Markdown | entrypoint-sampled | 实际阅读 Go chat Model、retry/usage 折叠消费与本次 helper 编译缺漏；原版丰富文本/工具渲染插件只定位入口。 | `f6b1480` 恢复被误移除的共享显式折叠 helper，不代表 Markdown、代码、思考或流式工具行为已逐项对齐。 |
| CHAT-13 | `P/ui-message-edit/src/client/MessageEditAction.tsx`；`ui-message-feedback/src/client/{controller.ts,FeedbackDialog.tsx}` | `internal/ui/chat/message_actions.go`、`user_message.go`；`internal/store/session/{message_edit,history_branch,feedback,session_feedback}.go` | not-reviewed | 本记录者未完成原版消息 action 与 Go 各分支的源码比较。 | 编辑重发、quote、分支、feedback、剪贴板和所有 ownership 边界保持未知；其他组/历史证据不能提升此行等级。 |
| CHAT-14 | `P/ui-reference/src/client/FileReference.tsx`、`ui-deliverables/src/client/ChangedFiles.tsx`、`ui-attachment/src/client/ImagePreview.tsx` | `internal/ui/chat/{image,citations,deliverables,turn_deliverables,tool_images}.go`；session image refs | not-reviewed | Composer 附件阅读不计作历史消息、引用/交付物渲染模块的完整审阅。 | 嵌套引用、交付物更新、工具链接和历史 image 生命周期未由本记录者复核；媒体并行工作另行记录。 |
| CHAT-15 | `P/ui-workflow-run/src/client/WorkflowRunPanel.tsx` 的 advanceDisclosureState / phaseStatusSummary / MemberRow / navigableMembers；`P/ui-primitives/src/StateDot.{tsx,module.css}`；`packages/subagent/subagent/src/projection-types.ts` | `internal/ui/chat/workflow.go`、`workflow_disclosure.go`；session `session_subagent_identity.go`、`workflow.go`、`workflow_member.go` | behavior-compared | **选定行为已双端源码比较并补齐，模块覆盖仍为局部。** run/phase clean/running/abnormal/activityCount、焦点延后收起、新 cycle；phase active/completed/interrupted 摘要；成员整行可用/失效保焦点及状态点；当前 parent/member 与真实 child identity.mode 导航。 | `f5c3f6d` disclosure；`6b17bff` phase summary / MemberRow / StateDot；`8e1694c` 加 `010b708` 按真实 `subagent` projection 投影 one-shot/continuable，保 sparse、cached/sequenced、control 早于 roster、Host reset 与 owner fencing，修复错误 one-shot-only 门控。仅源码完成，回归已准备未运行；完整 workflow-definition 折叠、每种畸态日志、全流程动作与新视觉尚未穷尽或验收。 |
| CHAT-16 | `P/ui-subagent/src/client/index.ts`、`SubagentReadOnlyComposer.tsx`、`ui-agents-panel/src/client/AgentsPanel.tsx` 与共享 InputBar / ComposerAttachments | `internal/ui/agents/{panel,child_editor,child_readonly}.go`；`internal/ui/composer/{child_editor,child_attachment,child_toolbar}.go`；session agent_navigation / retained scopes | entrypoint-sampled | **模块入口抽样，专用 ChildEditor 选定布局/工具回调已双端比较并源码补齐。** 已读 continuable/one-shot/parentOffline replacement 选择与 Go captured child scope；保留 literal child text、独立发送/图片/引用回调，复用 64-DIP thumb、原图预览与 28/34-DIP 工具/发送区。 | `6b17bff` 删除独立高 rail 和常驻双语 intake 长按钮，恢复紧凑 thumb 与工具入口；`5995b6c` 仅准备回归。未改 child Model、captured target、模型/权限或发请求。外部/一次性/未知/父端掉线的完整 agents tree/navigation 状态、保留流程与交互权限仍未在本记录穷尽；这些未读分支不能算实现缺失，也不能算已验收。 |
| CHAT-17 | `P/ui-trajectory/src/client/{index,layout}.ts`；`P/ui-conversation/src/client/skeleton/ContextMeter.tsx` | `internal/ui/chat/trajectory*.go`、`internal/ui/composer/context_meter.go`；session inspector/compaction | entrypoint-sampled | 实际追踪 Go ledger Row/Column/List 与 SDK 默认对齐，确认短 ledger 父 Column 缺高度而居中；抽查原始请求/范围状态入口。 | 本记录者给出 FillHeight 最小建议，由根线程集成；不把该几何诊断算作整个轨迹/请求头/上下文/自动与手动压缩语义已比较。 |
| CHAT-18 | InputBar `canSuggest`、`service.ts`；`A/src/types.ts` 的 SessionListMetadata.presentation、`sessions/service.ts:projectList`；`P/ui-goal/src/client/GoalBar.tsx`、`ui-plan/src/client/index.ts` 是其他入口 | `internal/ui/composer/next_prompt.go`；session next_prompt / session_presentation；frame NextPrompt 接线；plan/todo/performance/sessioncost | behavior-compared | 本行仅对照 NextPrompt：open/idle/plain、managed、attachment/edit、局部弹层、IME/Tab、接受时草稿与 owner、同 ID 重开 generation；阅读 metadata 在 list/control/follow 的实际来源。 | `0f35dfb` 补真实 managed 投影、门控及 generation 归属。Goal/Plan/Todo、下一提示提供端、usage/cost 全部边界并未因此获得深审；尤其目标激活/撤销、计划资源与 provider 计费仍需对应负责人阅读。 |
| CHAT-19 | `P/ui-approval/src/client/{index.ts,ApprovalPanel.tsx}` | `internal/ui/interaction/approval*.go`、`card.go`；interaction store | not-reviewed | 此记录者未读通 live approval 与历史 audit 的两端 source 状态机。 | approve/deny/reason/scope/expiry/重复操作保持未审阅，不能由历史卡显示推导全部正常。 |
| CHAT-20 | `P/ui-user-questions/src/client/index.ts`、`PlanReviewPanel.tsx`；QuestionComposer 调用链待审 | `internal/ui/interaction/{question_*,plan_review}.go`；`internal/ui/chat/{question_reply,submitted_plan}.go` | not-reviewed | 本组仅在总清单定位组件；未将其他 renderer 组工作转记为自己的 source 覆盖。 | live answer/edit/supplement/等待/历史/plan review 的完整契约仍需独立 source 覆盖登记。 |
| CHAT-21 | `P/ui-visual-replies/src/client/{index.ts,model.ts,PublishedPage.tsx}` | `internal/ui/chat/visual_reply.go`；session visual_reply*；htmlpreview 与 cmd visual reply adapters | not-reviewed | 本记录者未读通 visual reply 交互、历史读取、隔离与发布完整路径。 | readiness、有效/失效 action、预览/发布和安全边界保持未审阅；既不能据模块存在判可用，也不能把 gate 未跑判作未实现。 |

## 已提交修复与验证边界

- `a4af895`：所选 workspace 空态及相关侧栏/文件行为；不代表 CHAT-01 整行关闭。
- `49ad99b`：主 composer 文件 chip / 图片 thumb 的原版尺寸和密度；子会话专用编辑器仍单列在 CHAT-16。
- `f5c3f6d`、最终收窄的 `911ff74`：附件发送退役和 Workflow 所选 disclosure；中间 `1abc48d` 的广义 removed 结算不是最终契约。
- `0f35dfb`：NextPrompt 的真实 managed presentation 投影与限定门控。
- `f6b1480`：共享折叠 helper 恢复、恢复 payload 的 occurrence ID 规范化顺序，以及两个错误夹具前提的修正。没有删除旧保护断言来换通过。

本表是源码覆盖笔记，不给上列改动签“全部功能正常”。统一回归 run01 固定源码 `3b37750` 已失败；其后修复及并行媒体修改不能继承旧版本通过结果。本文编写不运行测试、构建、Host 或模型，也不修改总验收矩阵的实际状态。所有 `entrypoint-sampled` / `not-reviewed` 及行为级比较之外的分支仍是明确的源码覆盖欠账。

本轮追加源码收口：`6b17bff`（Workflow 成员/摘要及 child 紧凑输入区）、`8e1694c`（真实 child identity 与导航 mode）及 `010b708` 的 captured history binding 校验。`5995b6c` / `010b708` 中的新回归只准备源码，未执行；本记录仍不报告构建、测试、实机视觉或整模块全功能通过。当前无已确认的这两项源码缺口留后，未读与未验收边界如表列出。其他并行审计者的 CHAT 覆盖由总索引整合，不冒充本记录作者已读范围。

## 2026-10-19 conversation_depth 深读增量

以下是后续独立阅读和产品源码修改的证据，覆盖优先于上表旧的入口抽样记录；**只提升明确列出的分支**，不把整个 CHAT 模块签成全功能完成。产品提交 `67a6924`、`5c2e6fc`；本轮未编写测试/夹具，未运行 go test、build、应用、Host、语音设备或模型。仅运行独占文件 gofmt 和精确文件 diffcheck。实际验证由主线程在所有产品源完成后统一进行。

### 固定原版与协议证据

原版基准固定为 `e353e8e`，不能将 `C:/AI/Deepseek-Harness-Desktop` 工作区的 dirty 行为当基准。逐路径 diff 确认本轮 voice input/bundle、session-controller、gateway journal-stream、agents-panel/subagent、trajectory、ReasoningContent/ReasoningRow 没有相对基准的 dirty 差异。另直接阅读固定 `git show e353e8e:<path>` 的 ModelSelect、editor/runtime、PermissionSelect、AgentPresetSeat/seat-store、PreparationCard、VoiceSetupDialog、VoiceInput、readiness 和 compact Markdown CSS。ModelSelect 的 dirty 修改是 label slot/placement；两级菜单、搜索与焦点规则在固定版本已存在。runtime 的 dirty 修改涉及 command chip 恢复/commandsAllowed；1000ms 撤销合并在固定版本已存在。本次不导入这些 dirty 修改。

`session/follow` 新字段经过真实 server schema/handler 核对，不是只照相近客户端方法猜协议：

- `A/src/types.ts:585` 定义 maxMessages/turnWindow，`:596` 的 SessionFollowRequest 明确继承二者；`A/src/history.ts:120` 的 follow 接受此 request，`:184` 用这两个字段 paginate，`:320` 校验窗口值。
- `A/src/client/transport.ts:172` 的 follow 在 `:179` 调真实 `remote.session.follow` 并带 repairRequest；`:242` 和 `:243` 保留同名 maxMessages/turnWindow。`A/src/client/sessions/session.ts:54` 的实际 opening options 是 maxMessages500 / minMessages50 / minTurns2。
- Native `internal/store/session/follow.go:85` 按现有 Mux RPC 包装 `args.request` 发到真实 `session/follow`，字段名完全对应上述 server request；未新增私有 RPC。`internal/dsh/mux.go` 仍是既有路由，未改 SDK。

### CHAT-03：history / opening / follow / reconnect

原版深读 `A/src/client/sessions/session.ts` 的 open/doOpen、older page、history navigation、snapshot replacement；`api/gateway/src/client/journal-stream.ts:188` prepend、`:261` replaceGeneration、`:289` replaceFromOpening、gap repair/replaceThrough 和 assertPage；server `history.ts` 的 snapshot/assistant baseline/page window。Native 对照 `follow.go:70`、`page.go:95`、`history_navigation.go`、assistant_stream 系列和主线程现有 MaintainMux→FollowSelected 接线。

已关具体差异（`67a6924`）：opening 带真实 turnWindow；accepted follow 的逻辑 clean end 重开；business error 终止并显示错误；重复 durable seq 不重放；durable gap 不跨洞折叠，而重新取得原子 opening 和 assistant baseline；older/jump page 在提交前检查类型、安全序号、稠密区间、through/before 以及与已有窗口尾首相接（`page.go:160`）。generation、Session、ctx 的校验与整个 frame fold 共锁。原版重新 opening 的 replaceGeneration 会替换已发表历史窗口，并不保留所有此前 prepend 的老前缀；Native 重开替换因此不是已知丢历史缺口。

本轮没有穷尽多后台 Session 同时保留的 manager 调度，以及每种 malformed opening / server业务错误分类。物理断线由既有 main MaintainMux 负责；未用逻辑 clean-end 重试替代物理 owner。上述未审分支和真实断线/恢复验证保留为欠账。

### CHAT-04：Hero / workspace / first send

深读 `P/ui-conversation/src/client/skeleton/{ConversationMainPanel,ConversationRoot,HeroShell,InputBar}.tsx` 的 presentationOwned、opened/loading/error、blank/hero、resident editor 及工作区 before-send seat；对照 frame/conversation_phase.go、composer/workspace.go、draft_input.go、draft flights。确认原版由 presentation 对象存在门控，不只 managed owner，向 root 提供精确共享修复位置，由 root 接 `Items.Presentation != nil`。

本组没有改共享 frame/main，也没有重造 workspace flow。工作区 authoritative member/session fallback 由 workspace/root 负责；原生 first-send 动画与全部断线空态布局仍未由本组逐分支读完，不能据已有 resident editor 就关闭 CHAT-04。

### CHAT-05：native Chinese IME / selection / undo / newline / focus

深读固定 `P/ui-conversation/src/client/input/editor/{runtime,view-binding,keymap,command-chip-selection}.ts` 与 input facade；对照 Go draft_client/input/navigation/undo/reference_navigation/leading_claim/suggest_keyboard，以及只读 MyGo Windows composition End→UnmarkText→ReplaceText 的实际回调顺序。

`67a6924` 将 composition 中间更新归为一次撤销；记录结束的 marked range，识别先 Unmark 再显式 range commit 的系统顺序；结束后短窗口阻止同一次中文确认 Enter 落入发送/补全/leading claim；普通连续单字符同 caret 1秒内合并撤销；paste、显式 range、selection replacement、Undo/Redo/ClearHistory 分离。保留 UTF16 选区和 resident input，补全替换也通过 DraftClient.ReplaceText 进入同一 undo 路径。

未改系统 IME；真实候选窗口位置、TSF/IMM 系统差异、跨 DPI/resize、平台 word movement 和每种 clipboard representation 尚未运行也未穷尽源比较。现有旧 undo 测试以每字符一帧为前提，需要主线程统一验证时按真实产品规则复核，不能回退产品换旧夹具通过。

### CHAT-08：slash / @ / skill / file completion

深读 `P/ui-input-trigger/src/client/controller.ts` 的触发边界和键盘仲裁、ui-commands directory/service 的 input/command adjudication、ui-reference/skill 的 grammar 和引用激活；对照 suggest.go:114、suggest_keyboard、command_launcher、leading_claim、reference_activation/navigation、mentions、command catalog 与 executeCommand。

`67a6924` 补 slash 的 start/whitespace/punctuation 边界、word/url carve-out、@ grammar 优先；有 claim 时不重复 slash；leading input command 插入/claim，inline 隐藏 input descriptor；bare Host command 从菜单消费 token 后执行，而非只插字符串等用户再次发送。`command_pick.go:11` 校验当前 catalog、Host epoch、Session 后捕获 client+epoch+Session target，异步发往 captured client，late error 受 generation 限制。接口 `Model.OnPickCommand func(hostGeneration uint64, sessionID,line string)`，root 接到 `Store.PickComposerCommand`。本地 /model、/permission 复用已有专用 popup，没有创造泛化 popup 系统。

未知第三方 command 的任意 render/factory contribution、file reference 每种 preview 与远端 filesystem 错误尚未穷尽；不能把 Host descriptor 可执行泛化成任意前端 Factory 已实现。附件媒体预览由媒体组另审。

### CHAT-09：model / preset / permission menus

深读固定 ModelSelect.tsx 的 root/model/effort pane、>4 search、fuzzy排序、当前项聚焦、上下/横向/Escape/ShiftTab、选择收起与 focus restore、loading/busy；AgentPresetSeat.tsx/seat-store.ts 的 default/stage/blank/main retained/developer filter/refusal；PermissionSelect.tsx 的 projection/catalog缺失、锁定、optimistic pending/5秒 settle、full-access/auto acknowledge 和 rejected/error toast。对照 Go model_picker/model_picker_grouped、models projection/selection、agent_presets/agent_preset_draft、permissions/permission_projection/permission_risk 及 frame risk dialog。

`67a6924` 补模型/effort 共用原有 native menu，两级 pane、搜索键盘、当前聚焦和选择后触发器焦点；Hero preset 菜单显示 name+description（固定 AgentPresetSeat 实际如此，不能拿 Settings 的 PresetMenu 只有 name 来否定 Hero）。权限已有 token/client/Session/follow generation 风险确认、acknowledge、cancel、重复 confirm消费保护，已经存在不重做；投影为权威，Native没有实现原版纯UI optimistic caption，但会busy到 command+projection核对并显示失败。

完整 custom model label Factory、Preset Settings creator/guide全部入口、introduce逐字符动画，以及catalog变更时每种弹层退出仍未穷尽。权限没有用危险值直接绕过既有确认。上述范围不能由“有菜单”替代。

### CHAT-11：voice record / cancel / device / preparation

固定原版实际注册链：`boot/app-boot/src/profile.ts:224` 的 OPTIONAL_BUNDLES 包含 voice，默认 disabled；`experimental/voice-input-bundle/cordis.patch.yml:13` 挂载 client-ui-voice-input；`experimental/client-ui-voice-input/src/client/mount.ts:24` 注册 conversation.input.activity、同一 bundle 的 plugins.bundle.config。不是未装载实验样例，也不是默认启用功能。确切 key 为 `@deepseek-ai/dsh-experimental-voice-input-bundle`。

完整深读该 client 的 VoiceInput/controller录音请求、recording、stop/transcribing、generation+Abort、cancel、blur/hidden/unmount、pending插入冲突、retry；readiness镜像；microphone-device 的 client-local `dsh.voice-input.microphone` id/label、默认空id；MicrophonePicker 的可见页预览、换设备、missing/hotplug、默认输入与清理；PreparationCard全部 phase/steps/source/progress/failure/provider/language/configure；VoiceSetupDialog 只引导，不隐式安装或录音。Native对照 speech_recording、speech_insertion、voiceinput/controller、capture_windows 和 cmd/composer_voice。

`67a6924` 增加 native device/preview API 和 client-local controller（`voicecapture/devices_windows.go:110`、`voiceinput/microphone.go:60`）。Windows用 WinMM 设备接口ID保存选择，record/preview都准确解析该设备，失效不偷偷回退；默认空id走系统默认。preview只采样计算RMS/清空缓冲，不生成WAV/不转写；关闭、切页、最小化/隐藏和shutdown由owner取消并join释放；2秒枚举处理hotplug，失败需要用户Retry，不重复开拒绝设备。只读核对 Microsoft DRV_QUERYDEVICEINTERFACE / SIZE 和 WAVEINCAPSW 文档，未调用任何设备。

`5c2e6fc`：`speech_settings.go:54` 保留完整catalog与独立connection error，所有页面动作捕获 client/mux/generation；重试也原子核owner后开流；provider/language是Host持久选择，麦克风id/label只写现有local config两个字段。`VoicePreparationView:56` 支持所有 phase、展开steps、等待计时、known/unknown bytes、source自动/单源/人工选择、失败source/reason/HTTP/code、prepare/retry/cancel（waking不取消、cancelling禁重复）。`voice_setup.go` 按connected和ready/standby/waking门控，host-local unprepared提示安装，其余提示详情。移除旧文件转写路径的隐式 prepare。

`cmd/voice_devices.go` 的 `renderVoiceDeviceSettings(c,*key)` 实际签名是 `renderVoiceDeviceSettings(c *ui.Context,key string) bool`，仅认上述bundle config；`pauseVoiceDevices()` 和 `finishVoiceDeviceSettingsFrame()` 管理页面可见生命周期。`openVoiceSettings()` 通过现有 pluginmanager frozen owner，在冷 inventory完成后进入现有bundle details；frame/main callback由 root 共享集成。`composer_voice.go` 在手势时冻结所选device并先释放preview，再交给既有录音controller。完整语音链仍未实机/Host/网络执行。其他OS的设备picker是明确unsupported实现，Windows为本项目目标平台；不能报告跨平台全功能。

### CHAT-12：Markdown / thinking / stream / tools

深读固定 ui-chat reasoning row/content/summary、ui-primitives MarkdownText compact样式，以及 native chat/main/read-only/group、thinking、tool round/body/retry/usage/disclosure。`67a6924` 修正已完成thinking首行去粗体标记、streaming取最后完整段首行，避免每chunk截半个词；live/read-only都传streaming。`5c2e6fc` 主消息明确Chat.Size，thinking preview和compact body同一secondary（<=14减1，>14减2），不把所有明确次级字号变正文。widgets Markdown默认大小由 root 独占修改。

没有把 Markdown lexer/parser、KaTeX/数学、每种语法高亮、所有tool render Factory与流式patch变体读完。通用tool入口存在不能证明全部custom tool body一致；这些是实际剩余深读项，非已确认缺功能。

### CHAT-16：agents hierarchy / navigation / detail

深读固定 AgentsPanel 的main retained owner、subagentCatalog优先、fallback list、external禁打开、mode/status/tag与jobs入口；ui-subagent replacement 的unknown/one-shot/continuable/parentOffline；native agents/panel、child_editor/readonly，`agent_navigation.go:125` sync、`:167` open、`:195` back、`:234` earlier、`:327` rows、`:388` page、`:419` prompt/interrupt；retained scope与当前client/owner/token/binding校验。已有真实child identity/mode与captured read/write路径，不重复实现。

本组没有穷尽 jobs provider全部分支、任意多级后代的全部并发交错、所有custom detail body；既有布局提交不能提升为整个agents全覆盖。此次未另改agents产品源。

### CHAT-17：trajectory / context / compaction / history range

深读固定 ui-trajectory definitions/compaction/header/request/raw/detail、compaction/compaction/src/types.ts:37、:57 的真实ContentBlock[]，ContextMeter显示门控与manual compaction入口；对照 trajectory_requests/timing、history_navigation/page、inspector、context_meter和compaction动作。

`67a6924` 修复实际schema差异：compaction summary/rawOutput为ContentBlock[]，旧string字段导致整事件unmarshal失败；现在保留原始blocks并提取可读文本，兼容旧string但不丢真实字段。`trajectory_requests.go:125` 与 `trajectory_timing.go:176` 消费真实session/end-seed，将未完成compaction标为interrupted并保留duration/end时间；旧session/end兼容保留。历史range与olderpage共享上述dense/join检查。原版 source contract包含typed blocks，不以旧string夹具反推协议。

完整每一种模型请求header/redaction/usage timing、自动压缩所有触发条件与custom trajectory factory未穷尽。已有入口或原始JSON可见不等同全部详情正常。所有本节修复仅源码落地，未获编译/回归/实际视觉通过结论。

### 后续窄收口：permission opaque ID / Plugins Creator

继续对照固定 PermissionSelect 的 opaque value，发现 native 风险确认虽然最初选择按value，confirm却按caption反查，重名caption会误拒；`permission_risk.go:73` 改为核对已捕获value仍在当前catalog，风险token/acknowledge/owner规则保持。

固定 `P/ui-agent-preset/src/client/CreatePluginMenuItem.tsx` 完整阅读：ready roster含cordis才可选，idle/loading/error/missing分别描述；选中先onDismiss，再startCreatorDraft。`index.ts:190` 使用既有workspace startSession，clearPreviousDraft:false、cordis、beforeOpen stage/apply，无额外review modal、seed文本或send。两文件相对e353e8e无dirty差异。

新增 `cmd/whaleisle/plugin_creator_task.go` 独立Controller接线复用现有session.CreatorTask：唯一菜单click Prepare→Create(ticket)，成功后Open；没有第二确认、自动重试create/select或模型请求。过程和不确定回执在manager notice显示准确Host/Cwd/Session/phase/error；Locate只读该ID，explicit Open打开同一确认目标。page owner按实际main.go paintMainContent的 `marketOpen && marketManager`，不是Settings内置插件列表。observePluginCreator/postframe与suspendPluginCreator/hide由root接线。`CreatorPresetCatalog` 区分已读空roster、loading、failed和cordis可用，复用原有process roster；Add重开通过refreshPluginCreatorRoster显式刷新。新任务创建、presets选择等外部动作只存在产品回调源码，本轮未执行。

### 手动 carrier 重连的 interaction delivery 收口

固定原版 `api/gateway/src/client/stream-client.ts` 的 reconnect 退役当前carrier/logical stream；`client/remote-events.ts:168` finally abort各delivery并await tasks。server `api/gateway/src/index.ts:571` 新client注册时重放同一pending，`:688`推送同一pending.frame/eventId，`:717`断开只移除client delivery而不删除Host pending。因此复用client/interaction Store的Go手动重连不能把stream退出当真实Host Cancel，也不能让两个attempt等待同一answer channel。

已确认两条可达缺陷：旧FollowEvents synthetic OnCancel永久写`cancelled[eventId]`，新replay被误作cancelled；忽略旧Cancel又会让新旧handler争读旧waiter。另普通approval没有Scoped EventDelivery时，旧HTTP回执仅按eventId处理，会删除/报错到新replay卡。

新增 `interaction/events_delivery.go` 的 BeginEventsDelivery / HandleWaterfallForDelivery / OnResultForDelivery / OnCancelForDelivery / RetireEventsDelivery。Begin同锁退役旧owner，取消其HTTP权限、释放其live waiter/claim、记住callId draft；不写Host取消tombstone、不清continued RPC卡。新replay建fresh waiter；结果同锁核owner、waiter与本次response authority指针。每次返回的outcome合并原Scoped authority与attempt context，carrier退休在HTTP开始前/进行中均可撤权。真实Host Cancel保留原语义。`dsh.EventsFuncs.Retire func()` 与可选OnRetire合同把generation退出和Host取消分开；具备Retire合同的FollowEvents在退役后join本次handler任务。旧调用接口仍兼容。main captured-owner接线由root/settings组集成；本批仅产品源、gofmt、diffcheck，未tests/build/连接Host。

### 工作区 header 的 open-in-app 原生收口

依据为固定 e353e8e，不采用原版 dirty 测试改动。完整读取 `P/ui-open-in-app/src/client/{index.ts,controller.ts,OpenInAppAction.tsx,OpenTargetButton.tsx,OpenTargetButton.module.css,applications.ts,locales.ts,open-failure-toast.tsx}` 和 `host/open-in-app/src/{index.ts,shared.ts}`；本组生产源相对固定点无 dirty 差异。header utilities 注册在 index.ts:131-133，id open-in-app/order -10；同文件:92 注册 workspace.openLocal，作用域为 main conversation 且无 active panel。文件 association/reveal 是另一套 OpenPathAction 路径，不能拿来冒充目录打开。

controller.ts:24-106 的子状态包括初始 apps=null、每 controller 只读一次、失败变空并隐藏、过滤未知应用、存储 dsh.open-in-app.choice、记忆项失效回退第一个、所有控件共用忙态、POST 成功/失败。OpenInAppAction.tsx:43-57 表明 key=absolutePath 负责局部 feedback 生命周期，只有菜单显式选应用且打开成功才 choose，主按钮不改选择。OpenTargetButton.tsx:92-151 包括 pending/global busy、单应用无菜单、多应用右对齐菜单/键盘首项/默认标记/应用图标失败回退、忙态 spinner 与禁止动作。CSS:57 目录 pill 高27/圆角14，:85 disabled opacity=1。错误 toast 保留到用户关闭，离开该绝对路径或控制卸载则退休。

真实 Host 证明：shared.ts 定义 GET /open-in-app/apps 返回 {apps:string[]}、GET /open-in-app/icon/<id> 返回图标、POST /open-in-app/open body {app,path}；index.ts:186 校验已有 connection requestRejection，:193/206/241 分别注册三路由，:274 起校验已安装应用及绝对目录再启动。新 `internal/store/session/open_in_app.go:29` 复用 captured Client.HTTP、Endpoint 和 Cookie，禁止重定向；Applications/Icon 使用这些真实路由。CaptureOpen 在同步手势锁内核 Host epoch/client/selected Session/准确 Cwd，返回冻结 POST 闭包；worker 不改读下一 Session，不重试不确定 POST。

新 `internal/ui/frame/conversation_open_in_app.go` 的 WorkspaceOpenInAppState 持有 once-per-Host 应用/图标缓存和客户端选择；ObserveHost 在页面不挂载时仍退休旧 Host，普通 Session/页面切换不清缓存。载入/图标/打开晚回执在 captured source/context 下发布；Close 只用于窗口最终关闭。launch 全局串行；菜单成功后更新 choice 并通过 root 的 save 回调持久化，保存失败明确回执“应用已打开，但无法记住选择”。FinishFrame 只清已卸载控件的 toast，路径变更也撤销其旧反馈，绝不清应用缓存。OpenCurrent 与 split button 复用同一默认应用/busy/capture，给既有 workspace.openLocal 路由使用。主 header utilities、Model/mainState/config 生命周期与命令资格门控由 root 窄接线；本文件不修改共享 main/frame/header。

此项本轮仅源码实现、API 对照、gofmt 与精确 diffcheck。未连接 Host、未调用打开接口、未读写用户配置，未运行 tests/build/应用；菜单实际 popup 焦点/窄窗/长应用列表滚动、busy 的 toolkit 视觉 seat 和全部 Host 平台应用启动仍待统一验证。不能据此声称产品/视觉验收通过。

### 首轮统一回归失败后的定向纠正

root 提供 `evidence/source-depth-unified-20261019T1940Z/root/{failures.txt,root-all.jsonl}` 的已执行失败信号。本轮不自行重跑，集中修正后交 root 统一重跑失败包。固定 e353e8e `api/gateway/src/client/journal-stream.ts:188` 的 prepend 先核整页升序连续，再滤去 first>=before，最后核 accepted tail 与 before 接合；Go 曾在过滤前拒绝所有 overlap，已改为同一顺序。倒序、内部缺口、不接合和跨 through 仍拒绝；补定向负例，原 overlap 用例改为合法 [1,2] 并继续核旧 seq2 内容不被“dup”覆盖。turn navigation/ordinary page 旧夹具分别以80..81接180、120..180接480，确有原始 seq 缺口；补不产生消息的 extension metadata，使真实 journal 连续，保持消息数、页数、through、turn boundary、streaming tail、重试断言。

AssistantProtocolRepair 的旧夹具每次成功快照后都主动 End，accepted EOF 已按原版重开，所以30次请求不是修复逻辑再坏；改为保持第二条 follow 活跃，观察 Hello! 后显式取消，继续要求恰好两条流和原子基线。没有加超时、skip 或减少结果断言。

固定 `ui-model-selection/src/client/ModelSelect.tsx` 的 show 在 currentChoice 缺失时直接开 model，否则开 root；pending 不移除 trigger，父 locked 才禁用 trigger，pending 禁用选择行/提交但允许浏览。Go 原先总 root 且 selection busy 禁整个 trigger，已窄对齐；模型测试改走 root→model、键盘右箭头钻入，并保持 provider/search/current Host value/selection/dismiss 断言。该原版文件存在 dirty17/15 行，本轮均读 git show e353e8e；不采用 worktree 的 slot/layout 变动。

固定 experimental/client-ui-voice-input VoiceInput.tsx feedback 的 retryRecording 为 disabled={!usable||locked}；Go 漏掉 !usable，已补。idle 不可用测试走麦克风→setup dialog→插件详情，继续要求不启动录音、不改草稿；feedback 另核可用时重录、离线/未选/未准备时禁止重录。上述 journal/voice 固定生产源无 dirty 差异。所有本节后续修改只完成源码与 gofmt/diffcheck，尚未取得重跑结果或实机验收结论。

## 2026-10-19 bounded conversation follow-up (fixed e353e8e)

This pass is limited to the conversation families already named in the integration ledger: Factory/client contributions, first-send, preset Creator/guide, compression, and the preset introduction. All original reads use `git show e353e8e:<path>`; the dirty original worktree is not the reference. No tests, builds, application, Host, model, device or real user configuration were executed/read for this pass.

| Registered feature | Fixed original producer/controller/view | Current Go classification and change | Remaining observation boundary |
|---|---|---|---|
| First-send transition | `P/ui-conversation/src/client/skeleton/ConversationContent.tsx:137-254`, `ConversationRoot.module.css:441-455`: same-session previous hero plus promptAttempted; measured card top; paused pin, structural quiet80ms/cap450ms; text mutation excluded; resize/scroll repin; reduced motion; end/cancel/session/hero/unmount cleanup | **Confirmed missing, shared implementation assigned to root/extensions_depth.** Existing phase/resident editor is not the animation. Exact source conditions supplied to that owner; this pass does not edit frame. | Native motion and layout still require coordinated validation and actual window observation. |
| Settings Creator | `P/ui-agent-preset/src/client/AgentPresetSection.tsx:94-109`, `index.ts:190`: explicit Creator click starts the task with cordis and draft preservation, without a second review or prompt | **Fixed.** `cmd/whaleisle/settings_creator_task.go` now Prepare+Create from the original explicit click; `internal/ui/settings/creator_task.go` removes the extra review step. Existing captured Host/session/workspace/ticket checks, cordis roster gate, exact uncertain target, Locate/Open remain. The successful exact receiving task opens and Settings closes; no prompt/default mutation is added. | New source has not been run. Existing review-step fixtures must be updated only after all product source closes. |
| Preset guide lifecycle | `P/ui-agent-preset/src/client/PresetGuideDialog.tsx:87-157`: explanation/usage reader, selected tab focus, modal Escape/trapping, independently retained panel scroll, return to opening action; built-in known IDs only | **Existing content retained; state gaps fixed.** `internal/ui/settings/agent_preset_guide.go` keeps both opening controls mounted with stable Handles; a dedicated modal has per-page `ui.ScrollState`, selected-tab autofocus, close/backdrop/Escape return focus. Reopening resets reader positions; unmount drops local state. The existing EN/ZH guide dictionaries and shipped/built-in gate are reused. SDK `ui/runtime.go:612-629` disposes absent local element states. | Keyboard/modal and independent scroll need coordinated fixtures and real UI validation. No general settings dialog was changed. |
| Read-only preset configuration | `P/ui-agent-preset/src/client/AgentPresetSection.tsx:173-210`: View reads and opens once, keeps broken preset readable, modal dismissal returns to its card, unmount retires late reads | **Fixed entry/state mismatch.** `cmd/whaleisle/settings_agent_preset_reader.go` validates the captured selection ticket then Select+Begin+Load on one gesture. `internal/ui/settings/{agent_presets,agent_preset_reader}.go` opens the existing inert reader in a modal and retains the exact per-card focus Handle. Busy disables other reads; error offers explicit retry; existing Host epoch, visible page, cancellation and ticket fences, full-source copy and UTF8-safe paging remain. No second generic reader or new protocol. | No RPC or clipboard action executed; modal return focus, late close and retries await coordinated verification. |
| Creator preset introduce | `P/ui-agent-preset/src/client/seat-store.ts:34-51,147-164,187`, `AgentPresetSeat.tsx:46-137`, CSS:67-130: external stage cue, 150ms icon expo scale/opacity, capped40ms/200ms character stagger, 400ms ease-out from y4; reduced-motion acknowledges immediately | **Fixed.** `session/creator_task.go` creates a cue only on opening the captured, applied Creator target. `agent_preset_draft.go` exposes a one-shot sequence for that selected blank Session; acknowledgment checks Host generation, Session and sequence under one lock. Manual preset picks clear the old cue. `composer/preset_introduction.go` uses a finite frame timeline (maximum750ms), preserves full text width and accessibility caption, acknowledges reduced motion without a timer. `preset_picker.go` accepts only an optional local caption renderer; existing permission picks remain unchanged. Root integrates `OnAgentPresetIntroduced: m.Store.AcknowledgeAgentPresetIntroduction` in its composer model. | No actual animated frames/input examined this pass. Beam/fluid/SVG files were not edited. |
| Automatic compression triggers and marker | Host `packages/compaction/compaction-basic/src/index.ts:169-250` owns pre-step pressure, exact routed target policy, provider-confirmed CONTEXT_WINDOW_EXCEEDED recovery, cancellation/progress proof and bounded retries; `P/ui-chat/src/client/conversation-nodes/{compaction,command}.ts` owns landed checkpoint/summary identity; `chat/CompactionItem.tsx:34-101` defaults collapsed and disables missing summary | **Host behavior already served; client trigger is not recreated.** Existing Go automatic projection correlates durable compactionId and landed checkpoint, retaining full summary/counts and partial-history evidence. **Fixed view gap:** `internal/ui/chat/compaction.go` now renders a collapsed marker and expands its Markdown summary only on explicit user click. Missing/empty summary has no expansion, while counts remain visible. | Automatic pressure/overflow/model behavior requires an actual authorized Host/model run; none was attempted. Existing auto-marker fixture expecting an always-visible body is now stale and awaits the later fixture phase. |
| Model label/custom command contribution (“Factory” in earlier notes) | Fixed `P/ui-model-selection/src/client/ModelSelect.tsx:488-495` is a named session-scoped `conversation.input.model.label` **slot**, not a labelFactory field. The actual shipped `ui-deepseek-model-label-1017` is mounted by `packages/bundle/web-app/cordis.patch.yml:459`; its component only marks exact official deepseek-flash / DeepSeek-V4.1-Flash in trigger/menu/option. `P/ui-commands/src/client/contract.ts` has action/popupSelect contribution and decoration callbacks; `service.ts` owns registration/disposal/collision. | **Explicit architecture implementation difference, not closed by descriptors.** Native model widgets and slash controller do not execute arbitrary React slot or commandUi callbacks; the existing served full-page client owns those registrations. No universal Factory mechanism or unconditional1017 badge was added. Host `client/modules/src/index.ts:1054-1116` filters live Loader sources and publishes desired WebBootGraph; `graph():711`/HTML bootstrap and `client/hmr/src/index.ts:162,176` send graph entries. Browser `modules/src/client/entries.ts:12-18,39-53` publicly exposes only syncing/failures; managed ACTIVE fibers are private. Existing native ClientModulesSnapshot similarly has no successful mounted-ID/slot-render authority. Desired entry IDs or an unrelated connected served page cannot certify a native decoration. | Native inline model-label/third-party action-popup bridging remains an implementation gap; served page registration/runtime also needs actual acceptance. This bounded pass does not inspect every plugin/factory or label it complete. |

All fixes above are source changes only. `gofmt` and exact-path `git diff --check` were used; neither is a compile, regression, real-input, visual or full-product acceptance result. The unchanged finite boundary does not turn previously unread Markdown/tool/trajectory factories into completed coverage.
### Bounded Spec/read-only fixture correction after the coordinated run

The coordinated run reported `TestPresetConfigurationSingleClickModalAndReturnFocus` at `agent_preset_readers_test.go:171` (read-only source not visible). Source diagnosis shows `Tester.HasText` observes labels/text nodes (`third_party/mygo/ui/runtime.go:851-863`), not a TextArea buffer. The fixture now focuses the named read-only source control, selects and copies its real buffer through the headless clipboard, checks exact bytes, attempts typing/backspace/paste, and copies again to prove immutability. It retains the single-admission, retry, modal focus, close/retirement and original-card return assertions. No real clipboard or Host is used. The product receives a semantic accessible label for the source editor.

The two Spec P2 findings are product corrections: `preset_introduction.go` now shapes and lays out the caption with `c.Theme().Font`, caches the complete `ui.Font` separately from cue identity, and refreshes glyphs on a font change without restarting the animation or acknowledgment. The existing reduced-motion fixture now changes theme families and retains the exact once-only acknowledgment check. `preset_icon.go` copies fixed `P/ui-primitives/src/icons/index.tsx:469-480` `IconAgentPresetOutlineRegular` (16px viewBox, regular stroke1, unchanged four path geometries); both animated and static Hero chips use it in place of the house text glyph. The original icon's center scale/opacity motion is preserved. These changes received gofmt/exact diffcheck only here; affected package execution and visual acceptance remain with root's coordinated recheck.

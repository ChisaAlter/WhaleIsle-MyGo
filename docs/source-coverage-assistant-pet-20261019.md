# Assistant、IM 与桌宠源码覆盖记录（2026-10-19）

这是 `native-full-function-acceptance-20261019.md` 中 BOT-01～03、REMOTE-04、PET-01～02 的分动作源码记录，不是整行功能验收，也不是原版全部模块覆盖证明。

原版只读目录：`C:\AI\Deepseek-Harness-Desktop`，读取时 HEAD `e353e8e13fa0e7f9cb6bc16368e2fc00d6b3be49`。Go 读取基点 `7870118c03043cfc04bc219ad1605b7f5ebb7437`；本记录还包含下述 PET 输入穿透源码修正。并发其他域后续提交不自动包含在此覆盖结论内。下表原版路径均相对此原版目录，Go 路径相对此仓库。

## 等级与边界

- **S2（动作链已核）**：在列明范围内核对真实用户入口、回调及对应协议/状态/存储链；不表示全部异常分支或同状态视觉均已覆盖。
- **S1（局部已核）**：定位并读取了所列模块/投影的一部分，尚未把整条用户链和所有分支闭合。
- **U（未知）**：未完成相应源码或平台行为核对，不据此断言没有实现。
- **G（明确差异）**：真实入口及其调用契约已确认，存在具体代码差异；与“没有实机证据”分开记录。

本记录不新增模型、账号、麦克风、截图、外部消息或宠物窗口操作。历史 metadata/fixture 证据不能认证本轮修改后的真实运行。本域此前测试源码与编译修复的提交存在，但本次 PET 源码覆盖没有自行运行测试或构建。

## BOT-01：Whale 设置、任务与 routine

| 真实动作范围 | 原版依据 | Go 对应链 | 等级与结论 |
|---|---|---|---|
| Whale 名称/称呼、额外人设、长期记忆保存/清空 | `vendor/dsh-whale/client/client.js` 的设置表单、`settings/update` 等回调 | `internal/ui/assistant/whale_settings*.go`、`cmd/whaleisle/assistant_whale_settings.go`、assistant store | S2；已有明确设置/保存/错误反馈链，真实模型对话效果未认证 |
| Whale 模型/推理档、IM 默认会话、技能开关 | 同一 client 的 model/skill 设置 | `whale_preferences.go` UI/controller/pluginrpc | S2；使用现有公开模型与技能数据，未在本次调用发现或账号接口 |
| 任务详情、参与者会话、pause/resume/cancel/retry/stop | `vendor/dshbot/client/client.js` Task panel/control callbacks | `tasks.go`、`assistant_tasks.go`、`task_controls.go`、`dshbot_tasks.go` | S2；操作入口及确认/原 RPC 已有；真实委派执行和停止仍需独立验收 |
| 任务搜索、6 状态筛选、updatedAt/createdAt 排序 | 同文件 `taskSearchText`、`visibleTasks`、`taskStatusOptions` | `internal/ui/assistant/tasks.go`、`list_filters.go` | S2；本批补齐；原版全目录列表与 Go 现有“所选 Bot 的发出/收到任务”组织方式不同，未宣称跨 Bot 全目录搜索完全等价 |
| routine 创建、启停、运行、删除、频率/Cron/时区/次数 | 同文件 `RoutineEditorModal`、routine panel 和控制回调 | routine_create/schedule/calendar、routine_controls、routines UI、cmd controller | S2；创建使用一次原版 save，包括所选 advanced 配置；运行按钮存在不等于执行已通过 |
| contextFrom/self、silentAllowed、URL/command watch | 同文件 editor 及 `routine/save` 字段 | `routine_advanced.go`、`routine_create.go`、pluginrpc routine contracts | S2；原版默认与字段保留已对照；未执行 watch、command 或外部 routine |
| 历史、last output/error、trigger 显示/复制 | 同文件 routine panel/history/trigger 回调 | `routines.go`、routine_trigger store/controller/UI | S2；既有原 RPC/显示链；外部触发与真实结果未认证 |
| routine 搜索、nextRunAt/name 排序、missed-run 提示关闭 | 同文件 `routineSearchText`、`visibleRoutines`、`dismissedRoutineBanners` | `list_filters.go`、`routines.go` | S2；本批补齐，关闭仅在 Host+canonicalSession 内存范围隐藏提示，不修改 Host missed 状态 |

## BOT-02：Bot 与联系人

| 真实动作范围 | 原版依据 | Go 对应链 | 等级与结论 |
|---|---|---|---|
| 联系人搜索、kind/activity/source/hidden 筛选、分组/排序/分配/删除撤销、置顶/已读 | dshbot client 的 BotPage/section/contact callbacks | `contacts.go`、`contact_controls.go`、`assistant_contacts*.go`、pluginrpc contact contracts | S2；已有用户入口和状态链；目录外来源的所有扩展字段未做逐字段 census |
| 新建 Bot、工作区/模型/推理档继承、打开 canonical session | dshbot client editor/create 与 profile/model policy | `bot_create.go` UI/store/pluginrpc、`assistant_bot_create.go`、session 缓存 getter | S2；本批此前补齐当前 Host 的已知缓存选择器及空模型继承；创建和打开的实际结果不能由源码推断 |
| 名称/说明/文本配置、头像/图片裁剪、隐藏/置顶、复制/删除 | dshbot client `EditorOverlay`、AvatarPicker、contact actions | assistant `bot_*` UI/controllers/stores 与对应 pluginrpc | S2（列明动作链）；所有头像姿态/逐像素效果为 U |
| 既有 Bot 模型/工作区 route、工具/技能/MCP 能力、sender policy、notify、objective、memory/review | dshbot client EditorOverlay 与原控制面 | `bot_route/capabilities/sender_policy/notify/objective*/memory*` 及对应 controls | S2（已有固定入口/协议链）；工具实际执行、模型产出、通知投递为未验证，不能标成“缺少代码” |

## BOT-03：Room、群聊、面板与通知

| 真实动作范围 | 原版依据 | Go 对应链 | 等级与结论 |
|---|---|---|---|
| 创建/修改/删除 Room、成员与组参数、文本配置 | dshbot client Room editor、`room-preset.js`/group control plane | `room_create/edit/text/delete` UI/store/controllers、profile-local adapter | S2；原定义注册及目录/成员参数链已核；真实多成员回合未认证 |
| group/runtime、pending/error、打开现有成员会话 | dshbot client `GroupPendingInteraction` 与 runtime actions | `room_runtime.go` UI/store、`assistant_room_runtime.go` | S2；只读查询与明确打开动作已有；没有把打开会话当作自动批准/回答 |
| BotPage 挂载时所有 Room 状态刷新 | dshbot client `BotPage` 3618–3628：active room 1.5s，否则 5s | `assistant_contact_room_runtime.go`、`assistant_contacts_activity.go` | S2；本批补齐真实挂载时观察，隐藏取消；刷新只读，不发 prompt、不重连 |
| 群聊正文的运行态刷新、话题/回复 | dshbot client `GroupRoomBody`、thread projection/reply helpers | `assistant_group_runtime.go`、`assistant_group_threads.go`、`ui/chat/group_threads.go`、session group projection | runtime 生命周期 S2；完整 legacy/late-results/附件 thread projection 为 S1；未宣称全部兼容分支闭合 |
| 自定义面板定义、导入、预览、实例操作 | dshbot/Whale panel contributions | `internal/ui/custompanel`、panel stores/controllers | S1；已定位用户回调，未完成全部 widget/schema/runtime 分支逐项对照 |
| Bot/routine 通知、bridge alerts、通知点击与已读 | dshbot client 通知贡献 | `internal/desktop/botnotify`、`bridgealerts`、cmd observers | S1；已定位 tracker、poll、open/continue 链，未认证 OS 投递、全部状态组合或外部接收 |

更正：原版 `GroupRoomBody` 6528 在 `!session.running` 时不设周期定时器，不能把其 6529 单独看成“永远 1.5s”。Go 保留此 running 门槛；idle 5s 来自 BotPage 的独立 all-room effect。

## REMOTE-04：Channels 与 IM

| 真实动作范围 | 原版依据 | Go 对应链 | 等级与结论 |
|---|---|---|---|
| 平台切换、初始/错误/未连接状态、15s 状态观察 | `vendor/dsh-im/plugin-src/client/channels/shared/token-channel.js`、Feishu client | `ui/im/settings.go`、onboarding UI/store、root main 实际 render 生命周期 | S2；本批补状态轮询及嵌入页生命周期接线；未连接账号不是代码缺失 |
| 连接错误、lastCheckedAt、last-message error/code/reference/time | `channel-card-meta.js`、`last-message-error.js`、token-channel cards | `imonboard.Bot` 白名单投影、`channelHealthDetails` | S2；只显示公开状态字段，不输出凭据/原始 payload |
| 凭据绑定、QR/验证码、取消接入、删除绑定 | 各平台 client/API/provision callbacks | onboarding Review/Confirm/run/provision poll | S2；入口与 RPC 已核，平台实际授权/设备流程均未认证 |
| 重连与检查/测试消息 | shared token client、Weixin/WhatsApp/Feishu reconnect callbacks | `bot.reconnect`、`bot.reconnect-test`、`reconnectTestResult` | S2；全部现有支持平台测试动作已可 review，原 Host 决定目标；本域没有实际发信 |
| 每连接工作区和 Agent Preset | WorkspaceEditor、AgentPresetEditor，`bot.workspace.set`/`bot.preset.set` | standalone IM controls、embedded `channel_bindings.go`/`bindings.go` | S2；补 embedded 可达入口；公开 catalog 选择、null 继承和草稿保留已核 |
| 飞书回调修复/群权限/响应模式、Telegram/WhatsApp 访问策略 | 对应平台 client/API | onboarding 明确动作及 policy draft | S2（固定动作链）；真实权限效果和错误恢复未认证 |
| IM 历史、会话选择、消息发送与投递 | 原版 IM desktop 与 channel runtime | `internal/ui/im/im.go`、IM store、desktop adapters | S1；本轮侧重配置页，未做所有消息类型/送达/编辑态/平台扩展 census |
| AI Office | dsh-im Office contribution | `ui/im/office.go`、office store/adapters | S1；只定位既有入口，未逐动作核完远端 Office 状态机 |

未找到本基线统一的“channel enable” RPC，不因矩阵示例自行新增。每平台状态、授权与消息投递仍是不同范围。

## PET-01：窗口、外观与交互

| 真实动作范围 | 原版依据 | Go 对应链 | 等级与结论 |
|---|---|---|---|
| 显示/隐藏、设置入口、离线进入设置、恢复默认 | `src/main/desktop-live2d.js`、PetSection、真实 PANEL_ACTIONS | `pet.Companion`、actions/reset_behavior、`cmd/whaleisle/pet_settings.go` | S2；关闭 OS 窗口与持久 opt-out 有区分；真实窗口/焦点未认证 |
| scale/opacity、personality/activity、selfTalk/wander、drag mode、powerSave/clickSound、chat/look route | `PetSection.tsx`、`src/main/pet-settings.js` | `internal/ui/settings/pet*.go`、config PetBehavior、behavior/personality/look controllers | S2（设置入口和保存字段）；原版 SCALE_OPTIONS/OPACITY_OPTIONS 与 Go 列表一致，不能用底层 step 推断缺菜单值 |
| 拖动、抛掷、漫游、位置恢复、多屏边界 | `src/main/desktop-live2d.js`、pet-physics/wander、renderer gestures | `motion*.go`、`interaction.go`、recovery、renderer interaction | S1；已读公开 native cursor/position 与保存来源划分；跨屏/多 DPI/拖放实际体验为 U |
| 透明空白区域的鼠标穿透 | 原版 desktop-live2d 420/651/674：hit regions、setShape、setIgnoreMouseEvents | `hit_regions.go`、interaction tick/stop、Companion 初始 ignore、runtime visible regions | S2；本批修正原先缺少的输入门控。只根据本页当前 epoch 的可见角色/card/bubble 等矩形，以 ContentBounds/CSS viewport 比例映射 DIP 和 ±8 边界；pressed/pickup 保持交互，隐藏/关闭清空并拒绝旧报告。未宣称原生 shape、alpha 边缘或实际点击已验收 |
| THA4、ONNX/ORT、rig/sprite fallback、状态姿态/眼球 | `src/renderer/pet-live2d.js` 与模型/rig 资源 | `neural.go`、render manifest、tha4/pet-rig/runtime assets | S1；资源校验、fallback 路由、render-state 入口已定位；所有模型算子、完整动作/表情及像素等价未逐项核完 |

## PET-02：真实菜单、聊天、照料、成长与事件

| 真实动作范围 | 原版依据 | Go 对应链 | 等级与结论 |
|---|---|---|---|
| 真实菜单集合 | `pet-live2d.js` 1698–1709 `PANEL_ACTIONS` | pet.html、taskAction exact allowlist、Companion action switch | S2；原版真实集合 chat/look/pat/nap/settings/hide 均有对应入口；Go 另有既有粮碗/主会话等入口，不据数量判断缺项 |
| compact chat、共享历史/模型/推理档、发送、主窗口打开、关闭 | renderer chat functions、desktop-live2d 1135–1242、pet-chat | QuickChat view/window/protocol、pet_quickchat、Whale pet RPC | S2（可见控件与路由）；Go 卡片为独立原生窗口，原版为 overlay DOM，窗口焦点/IME/同状态布局为 U；模型实际执行未认证 |
| Host 不可用时 direct 临时聊天 | `src/main/pet-chat.js` 及 desktop bridge fallback | pet_directchat、petdirect、quickchat_direct | S1；已读路由决策和明确 Send 入口，未遍历全部凭据来源/失败组合，本次未读取凭据或调用 API |
| Look 的配置/调用入口与结果显示 | 原版 desktop-live2d 1246–1284、pet-chat.look | settings pet_look、cmd pet_look、desktop pet look entry | S2（固定动作链）；实际截图、显示器选取、模型回复为 U/未运行 |
| 摸头、tap/poke、throw、sleep/wake、文件投喂 | 原版 pet-stats、真实 panel/gesture/drop handlers | petcare stats/interaction、config pet_care、pet pat/rest/feed_drop | S2（已列照料效果与固定事件链）；所有动画/手势/文件拖放系统行为未认证 |
| 饱食/心情/亲密显示与衰减、同类 cooldown | `src/main/pet-stats.js` | `internal/petcare`、config stats、CareStatus | S2（列明算法/字段）；原版三条 bar 与 Go 文本表现不等同于像素复刻 |
| token 粮碗、首次基线、refresh/feed、等级/本地积分、分钟扫描 | `src/main/pet-growth.js`、desktop-live2d growth/feed | petgrowth/config、cmd pet_growth、growth_clock、GrowthStatus | S2（入口→扫描→存储→状态链）；本次没有扫描用户数据、喂食或改变成长值；完整日志格式/回滚边界为 S1 |
| DSH 工作状态、里程碑/休息/久别、outbox/贴纸、dismiss/open | pet-dsh-watch、desktop-live2d event push | petwatch、pet_notifications、notificationAction/renderer | S1；已定位 observer、source owner、ack/open 与媒体路由；全部事件生产/恢复/时序未逐项核完，实际通知为未验证 |

### 明确排除的残留代码

`pet-live2d.js` 4793–4825 虽保留 play/come/tease 分支，但唯一 `runAction(cell.id)` 调用来自菜单 cell；1698–1709 已明确把这些一次性 flavor verbs 从真实菜单移除。其残留 stats 常量和移动代码不是当前用户入口。本次没有为其新增菜单、照料奖励或移动行为；未实施的候选不计作已修复缺口。

## 当前结论

四行 assistant/IM 不应统一标记“完整覆盖”：BOT-03 面板/通知、REMOTE-04 消息投递/Office，以及若干完整扩展分支仍为 S1。桌宠同样只对表内已核动作给 S2；渲染、跨屏、完整事件及平台实际行为有明确未知范围。

本批新确认的 PET 输入穿透门控已完成源码修正，并准备纯协议、裁剪/间隙、负坐标/DIP 映射、8px 边界、拖动释放及旧 epoch 生命周期测试源码；没有执行这些测试。窗口焦点、透明像素精确形状、混合 DPI 跨屏和真实鼠标体验均未认证。已核 SDK Windows `SetIgnoreMouseEvents` 会切换原生穿透，Linux 该方法当前为 stub，不能把 Windows 源码链扩大为跨平台通过。

没有为已 cut 的残留动词新增入口，也没有把未实测当作缺功能。本文不是全量矩阵或原版全部模块的最终覆盖结论。

## Follow-up source audit after af11d25 (no runtime verification)

This appendix narrows the earlier S1 rows by named actions. It does not upgrade the whole BOT-03 or REMOTE-04 matrix row, and does not claim an exhaustive census of dynamically registered contributions.

| Named scope | Original source inspected | Go source inspected | Source conclusion |
|---|---|---|---|
| Office configuration control plane | `vendor/dsh-im/plugin-src/client/channels/office/index.js`, `api.js`; `src/channels/office/protocol.mjs`, `config-store.mjs`, `office-controller.mjs`; `plugin-src/host/channels/office/rpc.mjs` | `internal/ui/im/office.go`; `internal/store/office/model.go`, `store.go`; `internal/store/im/store.go`; IM settings entry and Office lifecycle | S2 for the actual configure/test/reconnect/remove controls, their existing RPC endpoints and payloads, status/health presentation, token write-only draft, and Hook preview. The original status includes schemaVersion=1; test returns tested=true plus snapshot; remove sends confirm=true. These contracts exist in Go. Remote connection, transport retry and Job execution remain unverified and are not declared missing code. |
| Room transcript projection | `vendor/dshbot/client/client.js` 5879-6469: content/pass/NEXT stripping, thread envelopes, declared call IDs, 15-minute legacy bucketing, late notice/inbox carriers, carrier deduplication, participant result correlation, pending-submission admission and attachments | `internal/store/session/group_threads.go`; `internal/ui/chat/group_threads.go`; `cmd/whaleisle/assistant_group_threads.go` | S2 for the named entry/callback path and these core projection branches: canonical-room recognition, member mapping, transcript/thread buckets, reply draft, late carrier admission, pending request deduplication and existing user-image/file rendering. Not an assertion of every historical event or equal visuals. Result-state alias caveat below remains S1. |
| Bot notification triggers and navigation | dshbot client 3517-3603: first observation baseline, attention false-to-true, objective entering blocked, new terminal non-silent run, notify opt-out and focus suppression | `internal/desktop/botnotify/tracker.go`; `cmd/whaleisle/assistant_bot_notifications.go` | S2 for those trigger/baseline/opt-out/focus rules and native-notification click callback to its captured current Bot/session. Permission handling uses the native SDK. Actual OS delivery/focus restoration and all concurrent timing combinations remain unverified. |
| WhaleBridge quota reminders | `src/main/whalebridge-notifications.js`; `system-notifications.js` | `internal/desktop/bridgealerts/monitor.go`; `cmd/whaleisle/bridge_alerts.go`; managed component adapter | S1: reminder formatting, source ownership/cursor and management-open callbacks were read. The Go component uses an explicit shared claim cursor; no claim/request was performed. Exact runtime component compatibility and OS delivery remain unverified. This scope is distinct from Bot notifications. |
| Declarative custom-panel JSON/widget library | Targeted dshbot/Whale clients and tracked source search for custom-panel/custompanel/panel-library terms | `internal/store/custompanel/store.go`; `internal/ui/custompanel/panel.go`; `cmd/whaleisle/custom_panels.go` | Go has reachable native create-draft/import/preview/save/template-copy/edit/reorder/remove controls and six widget kinds. An equivalent original declarative JSON/widget/library user entry was not located. Original sidebar tests register a generic main-slot key called custom-panel; that test is not evidence of this feature. Original-equivalence coverage is U, not S2 and not an established missing Go feature. The earlier table's vague dshbot/Whale panel-contributions attribution is withdrawn pending an exact original caller. |

### Narrow unresolved Room state alias

Original `groupMemberResultState` (dshbot client 5911-5934) recognizes `/timeout|timed.?out/`; Go `groupResultState` recognizes `timeout`, `timed out`, and `timed-out`. The original host's `lib/group-chat.js` 771-773 accepts `timedout` as an input alias but normalizes it to `timed-out`. It is not established that an unnormalized `timedout` value reaches the transcript event consumed by either client. This is a parser difference requiring caller/producer evidence, not a confirmed end-user gap. No product modification or synthetic event execution was performed for it.

No Host, model, account, notification, Office operation, service, test or build was invoked during this follow-up. Only this coverage document was updated after the six-file PET commit.

### Room timeout compatibility correction (source only)

Following the preceding audit, the existing result-state parser now preserves the original JavaScript timeout|timed.?out recognition, including the no-separator alias timedout and one-character separators, existing case normalization and capped-before-timeout precedence. JavaScript line-terminator/non-u dot semantics are retained. Only the existing timeout state is used; no new control, RPC or request was added. `internal/store/session/group_timeout_compat_test.go` prepares positive/negative fixtures through actual room transcript projection (including status precedence and visible-reply retention for nonmatching text). It was formatted but not run. The prior appendix's parser difference is therefore source-corrected, with runtime acceptance still pending.

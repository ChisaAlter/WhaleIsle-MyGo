# Windows 主工作区还原记录（2026-10-19）

本轮按原版 TSX/CSS 的组件树、布局、状态条件和事件行为还原 MyGo 原生 UI。源码等价实现、原生回归和最终原版实机对照分别记录；编译或测试通过不等于视觉验收通过。

## 实际基线与保护范围

- 现用 Go 仓库：`C:\AI\WhaleIsle-MyGo`，`main@b7fb9f9`。PID24668 使用仓库根目录 `whaleisle.exe`；嵌入构建信息为 Go1.27.2、b7fb9f9、vcs.modified=true，文件写入时间08:53:05（本机+08:00）。
- 原版进程：PID32964，`C:\软件\Whale Isle\Whale Isle.exe`；FileVersion0.3.9，ProductVersion0.3.9.0。原版源码只读参考为 `C:\AI\Deepseek-Harness-Desktop@e353e8e13f`，未假定该源码 SHA 等于已安装 EXE 的发布源码。
- 隔离工作副本：`C:\Users\48818\Documents\Codex\2026-10-19\task-2\WhaleIsle-MyGo`；分支 `codex/native-workspace-parity-20261019`；origin 指向 `https://github.com/ChisaAlter/WhaleIsle-MyGo.git`。
- 原仓库三处未提交崩溃修复（main.go、quickchat_view.go、resize.go）逐字保存在独立基线提交 `aeb77c5`；未覆盖原工作副本，未动原版、鯨 code、PR#2 或远端 main。
- 两个现用实例 PID/启动时间在本轮只读复核中保持不变。没有发送模型请求、读取凭据、删除会话、清配置、改网络/安全设置、发布或触发 CI。

## 原版规则 → 原生实现

以下原版路径前缀均为 `vendor/deepseek-harness/packages/client/`。

| 原版规格 | Go 对应 | 本轮处理 |
|---|---|---|
| `ui-layout/columns.ts`；conversation column W=clamp(680,column×0.64,920)，卡片W+32 | frame.go、conversation_width.go | 保留已有列契约；Hero/active使用同一输入卡边界，不重复扣Hero24侧距；保留手动宽度 |
| `ui-conversation/.../ConversationMainPanel.tsx`；`ConversationSession.tsx`；sessions/session.ts | session/store.go、follow.go、lifecycle_events.go、composer_pending.go、composer_submit.go；conversation_phase.go | 历史loading/ready/error独立于消息数；Host summaryBlank例外；按Host/会话保存PromptAttempted；过期generation隔离；共同错误与重试 |
| `ConversationRoot.module.css`、`HeroShell.module.css` | frame.go、conversation_phase.go、conversation_header.go、session_ancestry.go | Hero底内距32，标题26/32字重500、鱼34、gap10/7；标题/操作/工具/角落分组；560DIP阈值收起文字；使用原版SVG |
| `InputBar.module.css`、`InputBar.tsx` | composer.go、draft_input.go、font_preferences.go、tool_actions.go、command_launcher.go | 原生/后备输入稳定key；14字体/24行高，变更为fontSize+10；自然最小36/52；工具回现有+菜单；运行态Send/Stop规则；窄栏模型/发送分组 |
| 原版非空附件列表条件 | attachment_display.go | 修正非nil DraftState零附件仍挂载210DIP轨道的直接根因；错误反馈独立保留；不通过高度上限或重置配置掩盖 |
| `ui-workspace/rows/Rows.module.css`、`Rows.tsx`、`WorkspaceBrowser.tsx` | sidebar.go、session_menu.go、workspace_heading.go、workspace_groups.go | 会话32、工作区34、状态槽16×20；选中仅换底色；操作进菜单；搜索/多选同排，Esc清空收起；无重复工作区层级 |
| `ui-theme/styles/base.css`、`design-platform.css`、`derive.ts` | theme/preferences.go、theme.go；cmd/theme_font_preferences.go | 原版完整字体回退；保留显式覆盖；pressed/hover交互token；当前深色accent#7aaaff |
| 原版GoalBar无目标返回null | frame.go | 无实际计划/目标/队列/todo/编辑器时不挂载占位计划面板 |
| `MessageIconActions`、`TurnTailNodeView` | chat/message_actions.go、message_icons.go、feedback_editor.go | 检查后追加的聊天尾部控件还原；最终结果见下方验证记录 |
| 原版dock/pane header及上下文菜单 | workspace_panes.go、workspace_dock_tabs.go、workspace_dock_drop.go | 检查后追加的右栏布局控件还原；最终结果见下方验证记录 |

## 验证及证据

执行器使用已缓存的 Windows Go1.27.2：`C:\Users\48818\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.27.2.windows-amd64\bin\go.exe`。`Program Files\Go` 的基础版本实际为1.27.0，未降低仓库版本要求。GOMODCACHE、GOCACHE、GOTMPDIR 均放在本任务目录；原缓存缺少完整校验元数据的依赖由普通 Go 模块下载补齐，未修改系统网络设置。

- session整包：通过5.736s；包括真实本机假HTTP/WebSocket、回放、错误重试、快速选择切换、首次发送失败。沙箱首轮拒绝回环连接的结果保留为环境失败；允许本机测试服务后通过，未修改测试断言制造通过。
- sidebar整包：通过2.118s；新增层级、取消置顶、重命名路径定向通过1.349s。
- theme整包：通过1.717s；最终cmd整包通过3.832s，含字体/原生交互token/5DIP滚动条映射。
- composer：最终整包通过2.807s；覆盖零附件空白、工具菜单、自然/保存尺寸、运行态Send/Stop、输入身份与IME client选择范围。首轮菜单单击缺陷已修复后验证。
- frame：最终整包通过6.514s，重新生成宽/窄、明/暗、右栏、长标题、Hero/active/settling/error证据；覆盖空计划占位、同宽输入卡、焦点保持及dock交互。
- chat：最终整包通过3.945s；包括原版消息尾部图标、反馈模态布局与开关/失败草稿/键盘焦点/提交中关闭。
- feedback：最新定向生命周期回归通过3.018s；关闭后重开草稿不被旧请求覆盖，重复成功通知不吞失。
- Windows候选及隔离原生截图程序已构建成功；最终候选SHA及文件校验值待集成补齐。

证据目录在工作副本外：`..\evidence\frame`（原生软件渲染器PNG/JSON，包含400/900/1500中栏、Hero→首条消息、长祖先标题、settling、失败、窄/宽窗与明暗主题）；`..\evidence\native`（独立MyGo进程调用内置Window.CapturePage所得PNG）。

原生捕获实测客户区1200×800DIP、图像1800×1200px、scale1.5。测试窗口Hidden=true，未抢占现用会话；无Host、凭据或模型。它证明本机原生布局/字体栅格路径可运行，不是原版同状态对照。可复现捕获源保留在忽略目录 `.toolchain/native-parity/main.go`。

## 明确保留的缺口

- 未获得原版与最终Go版的同窗口/同DPI/同字体/同页面数据实机对照，未签署像素或视觉验收。当前会话缺少computer-use所需node_repl，CUA原生API禁用；未绕过这些限制。原版只读源码仍用于推进实现，未把缺截图当作实施前提。
- 当前用户Host接受的composer手动高度/宽度未能确认：只读检查已知公开settings.json未发现对应持久化key；没有清除设置。回归分别覆盖自然高度和保存高度220DIP。
- 原生IME client身份及选区测试已通过，实际Windows中文输入设备/输入法候选窗交互尚未验证。
- 现有原生API不能精确表达原版鱼标的CSS旋转中心/位移/缓动、滚动条仅悬停滑块显现、反馈遮罩模糊和composer锚定通知位置；未为这些绘制差异修改工具包或搭新框架。
- 其他页面（设置、插件、任务、账号等）本轮未逐页还原；富内容/复杂工具卡、非空计划的完整视觉、全部原版动画尚不能据本轮测试宣称完成。
- CI本轮触发0次，未push；历史累计未通过次数未知，未用本轮任务重置它。

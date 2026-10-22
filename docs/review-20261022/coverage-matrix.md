# 覆盖矩阵（弹窗 / 页面 / 功能）

日期：2026-10-22  
状态：审计进行中；✅ 已实机或测试验证，🟡 已改样式但未逐项实机，⛔ 待处理，➖ 非模态/不适用。

## 覆盖统计

- `ui.DialogBase`：**47 处 / 40 个文件**
- 已带玻璃或共享表面：**40 处**
- 剩余 7 处为有意的全屏/媒体面板，或受包循环限制的 widgets 确认框：
  - boot 启动日志 2
  - 子代理图片画廊 1
  - composer 附件查看器 1
  - projects 未保存更改 1（28px + 多层阴影）
  - usage 价格详情 1（20px 圆角）
  - widgets 通用确认框 1（28px + 阴影，无玻璃）
- 设置 12 个子页、主页面 3 个、核心弹层 5 个已实机进入。

## 一、弹窗

| 文件 | 数量 | 状态 | 说明 |
| --- | --- | --- | --- |
| frame/permission_risk.go | 1 | ✅ | 权限确认；死复选框已修，回归测试 + UIA 实机 |
| frame/archive_confirmation.go | 1 | ✅ | 归档确认；共享表面 |
| frame/session_feedback.go | 1 | 🟡 | 共享表面 |
| frame/workflow_member.go | 1 | 🟡 | 共享表面 |
| settings/confirm.go | 1 | ✅ | 通用确认；共享表面，widgets 测试覆盖按钮 |
| settings/permission_defaults.go | 1 | ✅ | 默认权限确认；共享表面 + 设置测试 |
| settings/shortcuts.go | 1 | ✅ | 速查弹窗 480×600/圆角 24/下移 30；行悬停铅笔；行内录键；见 `round3-shortcuts.md`、`round4-dialog-surface.md` |
| settings/agent_preset_guide.go | 1 | ✅ | 模式指南；共享表面 + 设置测试 |
| settings/body_helpers.go | 1 | ✅ | `bodyDialog` 通用弹窗；覆盖多个设置内容弹窗 |
| settings/pet_whale.go | 1 | ✅ | 鲸鱼娘记忆；共享表面 + 页面测试 |
| settings/effect_preferences.go | 1 | ✅ | 620×min(760,85% 视口) 编辑面板；共享表面玻璃/描边；快照 `settings-effect-dialog.png` |
| settings/onboarding.go | 2 | ⛔ | 首次引导 |
| settings/modal.go | 1 | ➖ | 设置主模态，已是 28px 玻璃 |
| terminal/pane_close.go | 1 | ✅ | 共享表面 + terminal 测试 |
| remote/file_actions.go | 1 | ✅ | 共享表面 + remote 测试 |
| remote/file_workspace.go | 1 | ✅ | 共享表面 + remote 测试 |
| composer/permission_command.go | 1 | ✅ | 共享表面 + composer 测试 |
| composer/voice_setup.go | 1 | ✅ | 共享表面 + composer 测试 |
| composer/attachment_display.go | 1 | 🟡 | 遮罩/20px 圆角/裁剪 |
| pluginmanager/confirmationDialog | 1 | ✅ | 共享表面 + pluginmanager 测试 |
| pluginmanager/installDialog | 1 | ✅ | 560 宽内容自适应、共享表面（淡入/玻璃/投影/描边）、22/14/12/24 头部；见 `round4-dialog-surface.md` |
| pluginmanager/index-sources | 1 | ✅ | 同上；底部重复 `关闭` 与内层重复标题已移除 |
| pluginmanager/market-choice | 1 | ✅ | 同上；头部关闭 + 底部 `取消/安装` |
| market/confirmationView | 1 | ✅ | 共享表面 + market 测试 |
| market/details | 1 | ⛔ | 已是 28 圆角/玻璃/淡入，但遮罩、头部（16 内边距 + 文字关闭）、正文内边距为手写值；下一轮接共享表面 |
| diffsurface/commit_dialog.go | 1 | ✅ | 共享表面 + diffsurface 测试 |
| diffsurface/caption_actions.go | 1 | ✅ | 创建分支；共享表面 + 测试 |
| diffsurface/caption_review.go | 1 | ✅ | Git Review；共享表面 + 测试 |
| schedule/presentation.go | 1 | ✅ | 排期删除；`DialogDecor` 解决裁剪/延迟点击 |
| projects/lifecycle.go | 2 | 🟡 | 28px + 玻璃层 |
| projects/modal.go | 1 | 🟡 | 28px + 玻璃层 |
| projects/request_changes.go | 1 | 🟡 | 28px + 多层阴影 |
| dirpick/panel.go | 1 | 🟡 | 28px + 玻璃层 |
| workspacepicker/picker.go | 2 | 🟡 | 280 菜单 / 380 确认框 |
| chat/child_image_gallery.go | 1 | 🟡 | 20px 圆角/裁剪 |
| chat/visual_reply.go | 2 | 🟡 | 共享表面 |
| chat/user_references.go | 1 | 🟡 | 共享表面 |
| chat/feedback_dialog.go | 1 | 🟡 | 已有 28px 玻璃 |
| usage/pricing.go | 1 | 🟡 | 20px 圆角/裁剪 |
| widgets/confirm_dialog.go | 1 | ✅ | 28px；按钮行为有测试；因包循环无玻璃 |
| boot/boot.go, scene.go | 2 | 🟡 | 启动日志 20px 圆角/裁剪 |
| 其他 assistant 弹窗 | 13 | ✅ | 回调同步；现有测试通过，未改 |
| composer/attachment 预览等 | 2 | 🟡 | 见 composer |

## 二、页面

| 页面 | 状态 | 证据 |
| --- | --- | --- |
| 首屏 | ✅ | `final-hero.png` |
| 插件页 | ✅ | UIA 进入，插件卡渲染 |
| 自动化任务 | ✅ | 8 个文本节点 |
| 鲸鱼娘 | ✅ | 57 个文本节点 |
| 设置·通用 | ✅ | 30 文本 |
| 设置·外观 | ✅ | 41 文本 |
| 设置·界面 | ✅ | 26 文本；开关 On/Off/On |
| 设置·模型 | ✅ | 17 文本；模型空状态 |
| 设置·内置插件 | ✅ | 12 文本 |
| 设置·技能 | ✅ | 4 文本；空状态 |
| 设置·MCP | ✅ | 10 文本 |
| 设置·远程 | ✅ | 15 文本 |
| 设置·Agent 预设 | ✅ | 12 文本 |
| 设置·用量统计 | ✅ | 4 文本；空状态 |
| 设置·鲸鱼娘 | ✅ | 22 文本 |
| 设置·关于 | ✅ | 10 文本 |\n| 首次引导确认 | 🟡 | 440px 共享表面 |
| 工作区选择器 | ✅ | 打开/取消 |
| Git 菜单 | ✅ | 打开/关闭 |
| 模型选择器 | ✅ | 打开；暂无模型（运行时） |
| 侧栏搜索 | ✅ | 展开、输入“新”、清空 |
| 侧栏折叠 | ✅ | 收起/展开 |

## 三、功能

| 功能 | 状态 | 说明 |
| --- | --- | --- |
| 新建会话 | ✅ | 按钮/行 |
| 会话行选择 | ✅ | UIA 点击 |
| 发送按钮空草稿禁用 | ✅ | UIA IsEnabled=false |
| 输入后发送启用 | 🟡 | 单元测试 `TestSendButtonFollowsDraftText` 验证空草稿不发送、有文字发送一次；实机键入仍受注入限制 |
| 停止生成 | ⛔ | 需生成中状态 |
| 终端抽屉 | ✅ | 选择工作区后打开：终端 1 PowerShell，提示符正常；无工作区时显示“终端不可用” |
| 右侧面板 | ✅ | 打开并渲染文件面板 |
| 工作区文件操作 | ⛔ | 未逐项执行 |
| Git 实际操作 | ⛔ | 仅菜单打开；未提交/切分支 |
| 插件启用/安装 | ⛔ | 未执行，避免改环境 |
| 市场安装 | ⛔ | 未执行 |
| 排期创建/删除 | ✅/🟡 | 删除弹窗测试通过；创建未实机 |
| 设置保存 | 🟡 | 部分页面测试覆盖 |

## 四、下一轮优先

1. `market/details`、`settings/onboarding` 接共享表面（遮罩/头部/内边距/关闭盒）。
2. 完整执行一次“输入 → 发送启用 → 不发送”的输入验证（需要可用键入方案）。
3. 终端不可用与模型目录为空的运行时原因。
4. Git/工作区/插件的实际执行路径。

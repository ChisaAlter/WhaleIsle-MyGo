# 覆盖矩阵（弹窗 / 页面 / 功能）

日期：2026-10-22  
状态：已收口；差异/修复/残留三张总表见 `final-convergence.md`。✅ 有测试或像素证据，🟡 已接共享表面但无参考对照（原生功能），⛔ 需真实执行。

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
| settings/shortcuts.go | 1 | ✅ | 速查弹窗 480×600/圆角 24/下移 30；行悬停铅笔；行内录键；关闭盒/Esc/遮罩/重置四类手势都先收行内编辑器（参考 `closeReference`），标签不可聚焦；`重新读取` 只在草稿脏/读取失败时出现，作为原生恢复路径保留；见 `round3-shortcuts.md`、`round4-dialog-surface.md`、`round5-close-gestures.md` |
| settings/agent_preset_guide.go | 1 | ✅ | 模式指南；共享表面 + 设置测试 |
| settings/body_helpers.go | 1 | ✅ | `bodyDialog` 通用弹窗；覆盖多个设置内容弹窗 |
| settings/pet_whale.go | 1 | ✅ | 鲸鱼娘记忆；共享表面 + 页面测试 |
| settings/effect_preferences.go | 1 | ✅ | 620×min(760,85% 视口) 编辑面板；共享表面玻璃/描边；快照 `settings-effect-dialog.png` |
| settings/onboarding.go | 2 | ➖ | 参考 `OnboardingSurface.module.css` 即全屏 overlay（`inset: 0`、无圆角），原生同为 100%×100%、圆角 0 |
| settings/modal.go | 1 | ➖ | 设置主模态，已是 28px 玻璃 |
| terminal/pane_close.go | 1 | ✅ | 共享表面 + terminal 测试 |
| remote/file_actions.go | 1 | ✅ | 共享表面 + remote 测试 |
| remote/file_workspace.go | 1 | ✅ | 共享表面 + remote 测试 |
| composer/permission_command.go | 1 | ✅ | 共享表面 + composer 测试 |
| composer/voice_setup.go | 1 | ✅ | 共享表面 + composer 测试 |
| composer/attachment_display.go | 1 | 🟡 | 遮罩/20px 圆角/裁剪 |
| pluginmanager/confirmationDialog | 1 | ✅ | 共享表面 + pluginmanager 测试 |
| pluginmanager/installDialog | 1 | ✅ | 560 宽内容自适应、共享表面（淡入/玻璃/投影/描边）、22/14/12/24 头部；空态选项行按 `.optionsRow`（引导文字开关 + 28 高注册表 chip）；展开的引导为 `.guide` 面板（0.5 描边/16 圆角/示例行 + “填入示例”）；spec 页有固定 footer（`.installSafety` warn 面板 + `.wide` 整宽 40 高主按钮），只有正文滚动；见 `round4-dialog-surface.md`、`round5-close-gestures.md`、`round6-install-footer.md` |
| pluginmanager/index-sources | 1 | ✅ | 同上；底部重复 `关闭` 与内层重复标题已移除 |
| pluginmanager/market-choice | 1 | ✅ | 同上；头部关闭 + 底部 `取消/安装`；安全提示与 spec 页同用 `.installSafety` warn 面板（参考两句文案、12 圆角、14 DIP 图标、10/12 内边距），不再是一行旧短句；见 `final-convergence.md` D33 |
| market/confirmationView | 1 | ✅ | 共享表面 + market 测试 |
| market/details | 1 | ✅ | 共享表面 + 参考头部（22/14/12/24、28 关闭盒）+ 24 体边距；`market-details-wide/narrow.png` |
| diffsurface/commit_dialog.go | 1 | ✅ | 共享表面 + diffsurface 测试 |
| diffsurface/caption_actions.go | 1 | ✅ | 创建分支；共享表面 + 测试 |
| diffsurface/caption_review.go | 1 | ✅ | Git Review；共享表面 + 测试 |
| schedule/presentation.go | 1 | ✅ | 排期删除；`DialogDecor` 解决裁剪/延迟点击 |
| projects/lifecycle.go | 2 | ✅ | 380 宽 28px 玻璃卡 + 200ms 遮罩/面板淡入（`.24` 浅色 / `.5` 深色遮罩）；1440×920 证据 `projects-rename-1440.png`、`projects-rename-fade-1440.png`，窄窗 `projects-rename-narrow.png`、`projects-archive-narrow.png` |
| projects/modal.go | 1 | 🟡 | 28px + 玻璃层 + 200ms 淡入（同 `DialogReveal`）；页面级弹窗未逐页截 1440×920 |
| projects/request_changes.go | 1 | 🟡 | 28px + 多层阴影 + 200ms 淡入；未单独截图 |
| dirpick/panel.go | 1 | 🟡 | 28px + 玻璃层 + 200ms 淡入；新建文件夹卡 1440×680 证据 `directory-picker-create.png` |
| workspacepicker/picker.go | 2 | 🟡 | 280 菜单 / 380 确认框 + 200ms 淡入；改名确认框证据 `workspace-rename.png` |
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

## 四、收口状态与后续

收口轮（`final-convergence.md`）收口了安装卡操作区/引导面板/两张安装卡的安全提示与原生弹窗的入场动画。矩阵里仍为 🟡 的条目统一属于三类：

1. **无参考对应物**：projects/dirpick/workspacepicker/chat 反馈/usage 价格/boot 日志都是原生功能；鲸屿本体没有同款弹窗，只能按共享表面（28px 玻璃、0.5 发丝线、200ms 淡入）自查，不能再逐像素对照。已按此标准接表面并给代表卡留 1440 证据。
2. **实机不可达**：插件启用/安装、市场安装、Git 实际提交、排期创建、工作区文件操作。执行会改本机环境；只做到打开/取消。
3. **实机截图受门禁**：Host provenance 与 150% DPI 下窗口超出屏幕，UIA 坐标不可靠；证据以生产渲染 `ui.Tester` + 键盘/指针事件 + 像素断言为准。

后续若要继续，优先级是：实机窗口截图 → 插件安装/市场安装的真实执行 → 输入→发送的实机键入 → 菜单子菜单展开。

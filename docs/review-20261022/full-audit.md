# 弹窗 / 页面 / 功能全量审计

日期：2026-10-22  
状态：进行中（第一轮盘点 + 第一批修复）

## 弹窗盘点

`ui.DialogBase` 共 **47 处 / 40 个文件**。按形态分：

### A. 已统一为 `chrome.DialogSurface`

| 区域 | 弹窗 |
| --- | --- |
| frame | 权限确认、归档确认、会话反馈、Workflow member |
| settings | 通用确认、默认权限确认、快捷键恢复、模式指南、`bodyDialog` 系列、鲸鱼娘记忆、特效设置 |
| terminal | 关闭运行中的终端 |
| remote | 新建/重命名/删除、关闭未保存文件 |
| composer | slash 权限确认、语音设置 |
| pluginmanager | 插件操作确认、添加插件、索引源、市场安装确认 |
| market | 市场操作确认 |
| diffsurface | 提交、创建分支、Git Review |

表面：28px 圆角、玻璃 + 实色打底、0.5px 发丝边、菜单阴影、遮罩淡入。

### B. 已有独立完整 chrome，但未走统一表面

| 区域 | 说明 |
| --- | --- |
| settings | 设置主模态本身：28px 玻璃，已对齐 |
| projects | `request_changes.go`：28px + 多层阴影，接近参考 |
| widgets | 通用 `ConfirmDialog`：已改为 28px、24 内边距、12/32 阴影；因 chrome↔widgets 包循环仍无玻璃层 |
| diffsurface | commit dialog：16px，自有样式 |

### C. 全屏 / 媒体 / 特殊面板（不应套 28px 小弹窗）

child image gallery、visual reply、attachment display、usage pricing、boot drawer/scene、dir picker、workspace picker。

第 7 轮已补：visual reply 两处改为 28px 共享表面；图片画廊/附件/价格详情补 20px 圆角、裁剪、遮罩与内边距；文件引用错误框改为 28px 共享表面。

第 9 轮已补：启动日志 drawer/scene 从 12px 直角感面板改为 20px 圆角、裁剪、24→20 内边距。

### D. 待迁移的普通弹窗

assistant whale settings、boot 两处 drawer、chat feedback / user references、composer voice setup、diffsurface 创建分支 / review / commit、market 两处、pluginmanager 四处、projects rename/archive/modal、settings agent guide / body helpers / effect preferences / onboarding / pet / shortcuts、usage pricing、workspacepicker。

## 本轮发现并修复

### 权限确认弹窗

- **死控件**：`acknowledged := request.Acknowledged` 每帧重置，复选框永远回弹，主按钮永远禁用。
- 修复：确认状态存 `ui.Local`，跨帧稳定。
- 回归测试：`TestPermissionRiskAcknowledgementEnablesConfirm`。
- 实机：UIA 勾选 → 主按钮启用 → 确认后权限 chip 变为“完全权限”。

### 统一表面迁移

本轮迁移：settings 通用确认、默认权限确认、terminal 关闭面板、remote 两处、frame 四处（上一提交）。

## 页面巡检

设置 12 个子页全部可进入并渲染：

| 页面 | 文本节点 | 结果 |
| --- | --- | --- |
| 通用设置 | 30 | 正常 |
| 外观 | 41 | 正常 |
| 界面设置 | 26 | 正常 |
| 模型 | 17 | 正常 |
| 内置插件 | 12 | 正常 |
| 技能 | 4 | 空状态（需选会话） |
| MCP | 10 | 正常 |
| 远程 | 15 | 正常 |
| Agent 预设 | 12 | 正常 |
| 用量统计 | 4 | 空状态（此范围暂无用量） |
| 鲸鱼娘 | 22 | 正常 |
| 关于 | 10 | 正常 |

主页面：

- 插件：可进入，文本查询遇到窗口重建（导航成功）。
- 自动化任务：8 个文本节点，正常。
- 鲸鱼娘：57 个文本节点，正常。
- 设置主模态、返回、rail 全部可点。

## 功能点击验证（第 2 轮）

- 设置 → 界面设置 → “显示归档列表”：实机 UIA Toggle `On → Off → On`，恢复原值。
- 模型选择、命令列表、终端抽屉、右侧面板均可点击打开；终端/右侧面板渲染正常。
- 插件页实机可进入并渲染插件卡、版本、启用/更多操作；先前文本计数为 0 是导航后窗口重建导致的查询失败，不是空白页。
- 侧栏搜索实机展开为输入框，输入“新”后出现归档结果过滤条（隐藏/显示/仅归档），可清空恢复。
- 终端抽屉内容显示“终端不可用”，来自运行时/插件状态，不是弹窗样式问题，已单列。
- assistant 系列 13 个 `ack := r.Acknowledged` 已核对：回调同步更新 review 状态并调用 `updateAssistantTasks()`，现有测试全部通过，因此不做无依据改写；若后续接入异步 RPC，再统一改为 `ui.Local`。
- 排期删除弹窗已解决：保留原宽度/滚动/底部按钮布局，改用 `DialogDecor` 只叠加玻璃、圆角、阴影与静态遮罩；`TestScheduleSaveAndDeleteKeepCapturedOwner` 通过。`DialogSurface` 的淡入会延迟测试点击，因此不用于该弹窗。

## 功能点击验证（第 6 轮）

- 工作区选择器：`选择工作区…` 打开“选择工作目录”，`取消` 能关闭。
- 模型选择器：点击可打开，当前显示“暂无可用模型”；模型页同样提示需 Harness Ready 且已拉目录，属于运行时状态。
- Git 菜单：`Git 操作选项` 打开“切换分支 / 提交并推送”，再次点击可关闭。
- projects 重命名/归档/未保存更改弹窗：保留 28px 圆角与阴影，新增玻璃层绘制。
- 目录新建弹窗、工作区移除/注册弹窗：28px 字段面新增玻璃层。
- 会话行点击可切换；侧栏“收起左侧栏 → 展开左侧栏”可往返；空草稿时“发送消息”为禁用态，符合预期。

## 下一轮

1. 迁移 widgets 通用 `ConfirmDialog`、diffsurface、projects、settings 其余弹窗。
2. 对主页面逐页做可点击清单：侧栏、工作区、会话行、搜索、面板开关、终端、Git、模型选择、发送/停止。
3. 每个修复补测试；最后输出完整覆盖矩阵。

## 运行时状态定位（第 13 轮）

- 终端不可用：	erminal.View 收到 m.Panel == nil；面板由 state.terminalPanel 在存在工作区时初始化。当前会话没有工作区根目录，因此为空，不是渲染缺陷。
- 会话重复写句柄：强制结束 MyGo 进程会留下子 harness，旧 harness 仍持有 session；清理孤儿 node 后报错消失。正常关闭应等待子进程退出，这是测试环境残留，不是 UI 缺陷。
- 模型目录为空：清理孤儿 harness 后仍为空，说明是 harness 目录/凭据未就绪，不是模型选择弹窗状态绑定问题。

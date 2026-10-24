# Round160 · 设置页原生对照（General/Appearance/Interface/Models/Account/About）

用原生 harness web 管线截取并测量了设置对话框，拿真实 DOM 数字校准 MyGo 的 `internal/ui/settings`。导航可达，未出现阻断；本轮修 3 处几何漂移，文案无新增漂移（round144 已修完），其余标为 equal / blocked。

## 1. 方法

- 起服务：`runtime\node\node.exe runtime\vendor\deepseek-harness\apps\cli\lib\bin.js web --host 127.0.0.1 --port 3099 --no-open`，`DSH_HOME` 用临时目录，从 stdout 取 token URL（3097 被占用）。原生应用只读，未改 runtime/vendor。
- 截图/测量：`scripts/native-web-shot.mjs <url> <png> 1440 920 1.5 --eval probe.js`，probe 内依次点「继续」（预览版说明）→ 侧栏「设置与账号」→ 菜单项「设置」，然后读取 `getBoundingClientRect` + `getComputedStyle`。
- 导航没有卡住：首屏的「预览版说明」用「继续」关闭后，侧栏底部「设置与账号」→ 菜单「设置」即打开设置对话框；「开始使用」空态对话面板在同一次会话里仍开着，但不影响设置菜单（它的关闭按钮没有可精确匹配的文本，脚本未点）。
- MyGo 侧用 `ui.NewTester` 量同一行原语（564 宽内容列）。

## 2. 原生 DOM 数字（1440×920，DPR 1.5，CSS px）

| 项 | 原生值 |
|---|---|
| 设置对话框 | x320 y60 800×800，圆角 28px |
| 左栏 | 导航按钮 x332 宽 164，起点 y124；文本 x368 |
| 导航项 | 高 40，内边距 9px 16px 9px 12px，圆角 12，项距 4（步进 44） |
| 导航文字 | 14px/22，未选 400 #0F1115；选中 600 蓝色、底 rgb(228,237,253) |
| 内容列 | x532 宽 564 |
| 行 | 上下内边距 16px，行间 1px rgba(0,0,0,.1)；行高 69（无描述）/77（有描述）/75–95（描述换行、开关行） |
| 行标题 | 14px/22 #0F1115，行顶 +16（无描述行因垂直居中为 +23） |
| 行描述 | 12px/18 #81858C，标题下 4px |
| 控件（下拉/胶囊按钮） | 高 36，内边距 0 14，底 rgb(245,246,247)，圆角 18（权限/语言/发送行为）或 12（步骤/快捷键按钮），字 14px/22，右缘与内容列右缘齐平（x1096） |
| 开关 | 36×20，圆角 999，内边距 2；开 #0F1115，关 rgba(0,0,0,.12) |
| 字号步进 | 数字框 + px + 17×12 圆角 4 的步进箭头 |
| 导航文案 | 通用设置 / 外观 / 界面设置 / 模型 / 内置插件 / 技能 / MCP / Agent 预设 / 关于 |
| General 行文案 | 权限·选择新会话的默认权限模式；语言；字号大小·仅影响会话内容的字号；工作步骤展示·选择希望看到多少工具调用细节；工作步骤收起时机·选择何时自动收起工作步骤；显示代码工作视图·开启后显示轨迹、本轮代码差异，新对话中的 Agent 预设切换；快捷键·查看和编辑当前可用的快捷键和输入操作；繁忙时的发送行为·智能体运行时 Enter 键和发送按钮的行为；Cmd/Ctrl+Enter 使用另一行为；自定义指令·给智能体定几条规则，后续对所有任务都生效；性能与用量·选择性能与用量信息展示的详细程度；视觉回复·允许智能体生成交互式页面并在会话内展示。截图检查需要支持图片的模型，或已配置的识图模型；关闭后已生成页面仍可查看。；在使用官方模型 API 时上传 Session Log·帮助改进 DeepSeek 模型与产品 |

## 3. 对照结果

| 表面 | 状态 | 说明 |
|---|---|---|
| 对话框尺寸/圆角 | equal | `settingsModalSize` 800×800 上限 + `panel.Radius(28)`，与原生一致 |
| 左栏宽/内容列起点 | equal | 180 rail + 16 gap + 16 padding → x532，与原生相同 |
| 导航项几何 | **fixed** | 38→40 高、上下内边距 8→9、项距 8→4（原生 40+4=44 步进） |
| 行内边距/标题/描述 | equal | 上下 16、14px/22、12px/18、标题下 4px；无描述行标题垂直居中偏移 23，与原生 214−191 一致 |
| 行高下限 | **fixed** | `MinHeight` 64→68：原生 36 高控件 + 上下 16 = 68 |
| 控件右对齐 | **fixed** | 值列原来左对齐在 180 宽列里（胶囊离右缘 42px）；改为列内 `AlignItems(End)` 后与原生一样贴内容列右缘 |
| 开关尺寸 | equal | 36×20，圆角 999 |
| 行分隔线 | 差异（保留） | 原生 1px rgba(0,0,0,.1)；MyGo 0.5 DIP muted 25%，属全局发丝线约定，DPR1.5 下视觉一致，未改 |
| General 文案 | equal | 行标题/描述由 Host schema 下发，与原生 DOM 完全一致 |
| Appearance/Interface/Models/Account/About 文案 | equal（round144 后） | 这些页的行也用同一 `settingsFieldRow` 原语；文案在 round144 已逐条对照修复 |
| 其余 8 个 tab 的 DOM | blocked | 本轮只测了 General（时间盒）；需要逐 tab 再跑一次 probe |
| 导航条目集合 | 差异（记录） | 原生见 9 项（无账号/市场/远程/用量/宠物）；MyGo 的条目按 Host 能力条件显示（Account 需已登录、Market/Remote/Usage 需对应回调、Pet 需 pet 模型），桌面端接线时可能多于原生，属产品面差异 |
| 深色主题 / 字号缩放 | blocked | 未在深色与不同 content font-size 下复测 |

## 4. 修复与测试

- [settings.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/settings.go>)：导航按钮 `Height(38)→40`、`Padding(8,16,8,12)→(9,16,9,12)`、导航滚动 `Gap(8)→4`。
- [preference_row.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/preference_row.go>)：值列 `AlignItems(ui.End)`（原生控件贴右缘）；行 `MinHeight(64)→68`（宽窄两种布局都改）。
- 新增 [settings_native_parity_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/settings_native_parity_test.go>)：钉住原生数字——行标题 y=23/高 22、下一行 y=84、描述 y=110/高 18、138 宽胶囊右缘不越过 564、开关 36×20。
- 未改 `internal/ui/i18n`（本轮无新增文案漂移）。

门禁：`go test ./internal/ui/settings ./internal/ui/i18n ./internal/ui/frame -count=1` → settings 12.250s、i18n 0.509s、frame 14.154s，全部 ok；`go vet ./internal/ui/settings ./internal/ui/i18n` → 退出 0。未提交、未构建 exe、未改 cmd/runtime/vendor/third_party。

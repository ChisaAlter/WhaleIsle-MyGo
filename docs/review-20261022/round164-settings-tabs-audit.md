# Round164 · 设置其余 tab 的原生对照（外观/界面设置/模型/关于）

承接 round160 的 General 测量，本轮把设置对话框其余 tab 逐个点开、量了原生 DOM 并对照 `internal/ui/settings`。导航没有阻断；修了 2 处文案漂移（其中一处是 round144 按源码快照改错、以安装版为准修回），几何行原语已与 round160 一致，其余差异按 equal/fixed/blocked 记录。

## 1. 方法

- 起服务：staged CLI `runtime\node\node.exe runtime\vendor\deepseek-harness\apps\cli\lib\bin.js web --host 127.0.0.1 --port 3099 --no-open`，`DSH_HOME` 临时目录，stdout 取 token。
- 一次 `scripts/native-web-shot.mjs 1440 920 1.5 --eval probe.js` 内完成：继续（预览版说明）→ 侧栏「设置与账号」→ 菜单「设置」→ 依次点九个导航项；每个 tab 返回 controls/labels 的 `getBoundingClientRect` + `getComputedStyle`，并另截「外观」整页图核对选中态。原生应用只读。
- 导航项（按 DOM y，均 164×40 r12，步进 44）：通用设置 124、外观 168、界面设置 212、模型 256、内置插件 300、技能 344、MCP 388、Agent 预设 432、关于 476；选中底 rgb(228,237,253)、字重 600。头部：打开配置文件 94×28 r8、关闭 28×28 r8。**账户 tab 在未登录状态不存在**（blocked）。
- 收尾：harness 已停，临时 DSH_HOME 与探针文件已删。

## 2. 各 tab 原生 DOM 数字

### 外观（颜色模式 / 主题）

| 项 | 原生值 |
|---|---|
| 区块标题 | 颜色模式，16px/24，y114 |
| 描述 | 选择浅色、深色或跟随系统。12px/18 #61666B，y150 |
| 三个模式卡 | 浅色/深色/跟随系统，各 183×84，r16，间距 8；选中（跟随系统）1.5px 深色描边 + 底色 |
| 主题标题 | 主题，16px/24，y298 |
| 主题按钮 | 创建主题、导入主题，86×36，r12，右对齐（右缘 1096） |
| 主题卡网格 | 两列；每卡 = 浅色半 + 深色半，各 137×88，外圆角 14（14 0 0 0 / 0 14 0 0）；行距 159，列距 15 |
| 卡内文字 | 浅色/深色 11px、当前模式徽标 10px、主题名 12px（DeepSeek/Codex/Claude/午夜/青瓷/暮紫…） |

### 界面设置

| 项 | 原生值 |
|---|---|
| 行 | 与 General 同一原语：上下 16、标题 14/22、描述 12/18 #81858C；单行描述行高 77，两行描述 95 |
| 控件 | `input[role=switch]` 44×28（命中区；轨道同 General 的 36×20 视觉）；需要重置的行另有 28×28 图标按钮 |
| 行文案 | 标题栏 Git 操作·显示初始化、切换分支、提交及菜单。关闭后仍可通过进行中的进度与对话框完成操作。；终端抽屉开关·在标题栏显示终端抽屉按钮。关闭后仍可使用 Ctrl+\` 切换。；右侧栏按钮·在标题栏显示右侧栏按钮。关闭后仍可使用 Ctrl+\\ 开合同一条右栏。；发送消息时的思考炫光·智能体发送或思考时，在输入条周围播放彩色边光。；拖动调整输入框大小·允许拖动输入区上、左、右边框改变高度和宽度。关闭后高度仍按草稿自动增高，宽度回到栏宽。；会话统计·在输入框上方显示轮次、耗时、缓存与 token。关闭后隐藏数字，保留这一行间隔。；会话累计费用·在输入框下方显示峰谷状态条与当前会话累计费用。；官方峰谷时·会话累计费用开启时，在会话统计下方显示官方峰谷时段状态条。开启后始终显示；关闭时仅在检测到 DeepSeek API 路线时显示。；对话与轨迹页签·在会话顶栏显示「对话」「轨迹」切换。关闭后仍停留在当前视图。 |

### 模型

| 项 | 原生值 |
|---|---|
| 标题/简介 | 模型 16/24；填入各提供商的 API 密钥即可使用其模型。14px，muted，y150 |
| 提供商卡 | 名称 14/22 + 组织 12/18 muted；编辑 46×28 r14；未配置警告 12px 橙 rgb(221,134,41) |
| 编辑态 | API 地址输入 506×24；取消 58×36 r18、保存 56×36 r18（深色实底）；提示 请填写兼容 Anthropic Messages 协议的 API 地址。 |
| 识图模型 | 不启用 530×36 r18 全宽胶囊；说明 12px 两行 |
| 模型行 | 两个输入 227/155 宽 + 两个 28×28 图标按钮；添加模型提供商 564×36 r18 全宽 |

### 关于

| 项 | 原生值 |
|---|---|
| 标题 | 关于，**18px/24**（比其余 tab 的 16px 大） |
| 文案 | 本桌面应用与官方 Harness。13px；Whale Isle 15px/600；桌面版本 — 12px；Based on DeepSeek Harness 12px；版本检测仅在桌面应用中可用。12px |
| 控件 | 仅头部按钮；无页面内操作 |

## 3. 对照结果

| 表面 | 状态 | 说明 |
|---|---|---|
| 导航项/顺序/头部 | equal | 九个标签与 MyGo `sectionLabel` 完全一致（通用设置/外观/界面设置/模型/内置插件/技能/MCP/Agent 预设/关于），顺序同位，按钮 164×40 r12 已由 round160 对齐 |
| 外观标题 | **fixed** | `色彩模式`→`色制`（round144）→**`颜色模式`**：安装版 `scheme.title` 与实机 DOM 都是 颜色模式，源码快照里的 色制 是旧版本；en 仍为 "Color scheme" |
| 外观三模式卡 | equal | MyGo `Basis(180)+Padding(20,32)+Radius(16)` ≈ 183×84 r16；选中 1.5px 描边 + 底色与截图一致 |
| 外观描述 | equal | 选择浅色、深色或跟随系统。与 MyGo 文案逐字相同 |
| 外观主题库 | 差异（记录） | 原生是「浅/深预览对」两列卡片网格 + 创建/导入主题按钮；MyGo `themeLibraryPane` 是自有主题库结构，卡片内部构成不同，属布局差异（本轮不改结构） |
| 界面设置文案 | equal | 九行标题/描述均由 Host schema 下发，MyGo 走同一 `fieldRow` 渲染，文案一致 |
| 界面设置开关 | 差异（记录） | 原生命中区 44×28、轨道 36×20；MyGo `ui.Switch` 按钮 36×20（无 44×28 命中区）。视觉一致、点击区略小，severity 低 |
| MyGo 独有界面项 | 保留（记录） | `interface.go` 的 Bots（测试中）/远程工作区/归档列表是 MyGo/Host 能力条件项，原生此 tab 无；按产品面差异保留，不删 |
| 模型简介 | **fixed** | `管理模型供应商、模型目录与视觉模型。`→**`填入各提供商的 API 密钥即可使用其模型。`**，en → "Enter your API keys to use models from the following providers."（安装版 ui-settings-models `intro`） |
| 模型卡结构 | 差异（记录） | 原生编辑态是内联输入 + 56/58×36 r18 保存/取消 + 564×36 全宽「添加模型提供商」；MyGo 是自有 `galleryModelCard` 结构，字段文案一致、几何未逐像素对齐（本轮未改结构） |
| 关于标题/文案 | equal | 18/24 + 13 + 15/600 + 12 与 MyGo 逐项相同；MyGo 多出诊断/凭据存储 disclosure（桌面端自有，保留） |
| 关于版本行 | 差异（state） | 原生（无桌面更新器）显示「版本检测仅在桌面应用中可用。」；MyGo 桌面端显示更新状态块，属状态差异非文案漂移 |
| 账户 tab | blocked | 未登录状态原生没有该导航项；需已登录 DSH_HOME 才能测 |
| 深色主题 / 字号缩放 | blocked | 本轮未切换主题或改内容字号复测 |
| 内置插件/技能/MCP/Agent 预设 | 已探测未对照 | probe 数据已采集（34/23/22/34 个 label），本轮时间盒内只对照了指定五个 tab |

## 4. 修复与测试

- [appearance.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/appearance.go>)：区块标题 `色制`→`颜色模式`（以安装版 `ui-theme` 字典与实机 DOM 为准；round144 采信了旧源码快照）。
- [models.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/models.go>)：简介改为安装版 `intro` 的中英文案。
- [appearance_scheme_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/appearance_scheme_test.go>)：新增标题/描述断言（"Color scheme" + "Choose light, dark, or follow the system."）。
- 新增 [settings_native_copy_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/settings_native_copy_test.go>)：`TestModelsPaneUsesInstalledIntroCopy` 在中英两种 locale 下断言模型简介。
- 未改 `internal/ui/i18n`（这两条是 settings 内联文案）。

门禁：`go test ./internal/ui/settings ./internal/ui/i18n ./internal/ui/frame -count=1` → settings 12.326s、i18n 0.482s、frame 13.983s，全部 ok；`go vet ./internal/ui/settings ./internal/ui/i18n` → 退出 0。未提交、未构建 exe；未改 cmd/runtime/vendor/third_party；harness 已停、临时目录已清。

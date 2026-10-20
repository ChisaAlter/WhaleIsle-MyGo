# 第二轮全面对比（2026-10-20，对照安装版 0.3.9）

第一轮 101 条偏结构/功能；这一轮直接用**安装版真实界面**做像素级对照，补的是“看起来还是不一样”的那些细节。

## 1. 方法

- 安装版：`C:\软件\Whale Isle`（0.3.9，Electron 44 / Chrome 152）。用独立用户目录 + `--remote-debugging-port=9333` 启动**参考实例**，不碰用户正在用的 3080 实例。
- 采集：CDP `Page.captureScreenshot`，视口固定 1440×920、DPR 1；设置弹窗 13 个分页逐个点开采集。
- 原生侧：PrintWindow 抓真实窗口（150% DPI 下 1440×920 物理 = 960×613 DIP），并换算到同一 DIP 视口比较。
- 截图：`docs/review-20261020/evidence/compare/`（`ref-*` 为安装版，`ours-*` 为原生）。

已采集参考页：hero、账户菜单、设置（通用/外观/界面设置/模型/内置插件/技能/市场/远程/MCP/Agent 预设/用量统计/鲸鱼娘/关于）、侧栏折叠、鲸鱼娘页头。

## 2. 结论先说

按钮配方（Wave 11）已经和原版一致；**剩下的差距主要不在单个控件，而在壳层（sidebar/标题栏/背景）和设置页的“内容组织”**。

### P0：进入应用第一眼就能看出

| # | 项目 | 安装版 | 当前原生 | 证据 |
| --- | --- | --- | --- | --- |
| 1 | 侧栏空状态 | 始终展开 280 DIP：品牌字标 `鲸屿 WHALE ISLE / BASED ON DEEPSEEK HARNESS`、`新会话` 按钮、插件/自动化任务/鲸鱼娘、工作区段与图标组、Project 段、底部 `设置与账号` + 头像 | 无工作区时收成 64 DIP 图标轨，品牌字标/分组/账号文字全部消失 | `ref-hero-1440.png` vs `ours-hero-1440.png` |
| 2 | 调试文案泄漏 | — | 内容区左上角出现英文 `no workspace folder ready` | `ours-hero-1440.png` |
| 3 | 工作区行重复 | hero 下只有 `Whale Isle` + `标准模式` 两个入口 | `无工作目录 ⌄` 与 `无工作目录` 并列，语义重复 | `ours-hero-1440.png` |
| 4 | 默认背景 | 全幅柔和渐变（绿/蓝/紫），无硬边圆形 | 大块半透明白色圆形叠层，像占位图 | 两张 hero |
| 5 | 标题栏 | 右侧 `初始化 Git` 胶囊 + 终端/右栏/最小化/最大化/关闭 5 个 32×32 控件 | 只有关闭（其余不可见），无 Git 胶囊 | `ref-hero-1440.png` |
| 6 | 设置弹窗宽度/结构 | 800×793，左侧导航 180、行高 40；内容列 x=532，行高 64，标题 14 + 说明 12，右侧控件高 32 | 结构与尺寸接近，但行距/分组留白偏紧（以组件画廊为准） | `ref-settings-general-1440.png` |
| 7 | 外观页缺失 | 颜色模式三卡 + 主题网格（DeepSeek/午夜/青瓷/暮紫/琥珀/宣纸…，每卡浅深双预览）+ 创建主题/导入主题 | 仅有基础偏好行，无主题网格与创建/导入 | `ref-settings-appearance-1440.png` |

### P1：进入具体页面后能看出

| # | 项目 | 安装版 | 当前原生 | 证据 |
| --- | --- | --- | --- | --- |
| 8 | 设置通用页控件 | 权限/语言/工作步骤展示/收起时机/链接打开方式/繁忙发送/关闭窗口 全是 32 高圆角 8 的下拉；字号是 14 + px 单位；开关是 44×24 | 控件样式已接近，但下拉底色/边框粗细与行间距未逐项对齐 | `ref-settings-general-1440.png` |
| 9 | 模型页 | 账号卡 + 供应商卡（API 密钥输入 + 未检测环境变量提示 + `> 自定义设置` 折叠 + 取消/保存）+ 虚线 `+ 添加模型提供商` + 识图模型卡 | 有同类元素，但卡片层次与提示文案排布不同 | `ref-settings-models-1440.png` |
| 10 | 鲸鱼娘页 | 顶部两个 segmented（聊天与能力/桌面形象与行为）、开关 + 名字/称呼双输入 + 默认模型双下拉 + IM 开关 + 额外人设/长期记忆/技能 行（右侧蓝色文字动作） | 已有主体，分段控件与蓝色文字动作的间距/字号待对齐 | `ref-settings-pet-1440.png` |
| 11 | 账户菜单 | 240 宽，四项：设置 Ctrl+, / 远程 / 联系我们 / 登录；圆角 12、半透明材质 | 已有菜单，宽度/快捷键提示/分隔待对齐 | `ref-account-menu-1440.png` |
| 12 | Sidebar 导航 | 行高 36、图标 16、文字 14、左右 14 内边距、hover 铺满 252 宽圆角 8 | 图标轨模式下只有 28 图标；展开态需按此规格 | `ref-hero-1440.png` |
| 13 | 品牌区 | 字标 38 高，鲸鱼 logo 40，“鲸屿” 22 衬线 + `WHALE ISLE` 12 字距，副标题 9 大写字距 | 展开/折叠品牌差异待实现 | `ref-hero-1440.png` |

### P2：细节

| # | 项目 | 说明 |
| --- | --- | --- |
| 14 | Composer 尺寸 | 安装版 774×98、圆角 28、白 80%、占位 14/24；原生实测约 704×107（960 DIP 视口）、圆角 24 |
| 15 | 发送键 | 安装版 34×34 圆形 `rgb(65,118,230)`；原生空态显示浅蓝禁用感 |
| 16 | 预览徽章 | 安装版白底半透明胶囊 + 10px 字；原生为浅蓝底 |
| 17 | 终端空态文案 | 安装版 `暂无终端会话`；原生 `请先打开可访问的项目会话，再启动终端。`（更长，位置在 hero 下） |
| 18 | 设置导航选中态 | 安装版选中行只有极浅底色，无左侧色条；需核对原生 |
| 19 | 插件页 | 标题 20 + 副标题 13「安装、启用和配置插件」+ 右侧「添加插件」拆分按钮（主色 + chevron）；加载态是 40px 头像 + 两行骨架条；内容左缩进 32、最大宽 960 | `ref-plugins-1440.png` |
| 20 | 定时任务页 | 标题行 20 + 右上 28 高「新建」主按钮（plus）；`全部/已开启/已结束` chip 组；搜索框 36 高圆角 8；加载态居中 spinner | `ref-schedule-1440.png` |
| 21 | 鲸鱼娘页 | 主区头部（鲸鱼图标 + 16 标题 + 智能体团队 + ⋯ + 齿轮）、`对话/轨迹` 下划线 tab、底部 composer 792 宽圆角 28（与 hero 同配方） | `ref-pet-page-1440.png` |
| 22 | 页面背景 | 所有页面共用同一全幅渐变；原生 tester 画廊在白底渲染，真实窗口应透出渐变 | 各 `ref-*.png` |

## 3. 与第一轮审计的关系

- 第一轮关闭的是**结构/功能/特效**（89/101）：市场、插件列表、定时任务、远程、Project、终端配色、透明主题、按钮等。
- 本轮新增的是**壳层与设置内容组织**：侧栏展开态、标题栏、默认背景、外观主题网格、模型/鲸鱼娘页的控件编排。
- 本轮不改第一轮已关闭的结论；新增条目记在 `CLOSURE-AUDIT.md` §8。

## 4. 修复顺序（本轮起）

1. **壳层 P0**：侧栏展开态（品牌/分组/底部账号）、去掉调试文案、去重工作区入口、默认渐变背景、标题栏 Git 胶囊 + 控件。
2. **设置 P0**：外观页（颜色模式三卡 + 主题网格 + 创建/导入主题）、设置通用页行距/控件对齐。
3. **P1**：模型页卡片层次、鲸鱼娘页控件编排、账户菜单尺寸、Sidebar 导航/品牌规格。
4. **P2**：composer 尺寸与发送键、徽章、终端空态文案。

## 5. 截图索引

```
ref-hero-1440.png                    安装版工作区（空 Project）
ref-hero-960.png                     安装版 960 DIP 视口
ref-settings-general-1440.png        设置 · 通用设置
ref-settings-appearance-1440.png     设置 · 外观（颜色模式/主题网格）
ref-settings-interface-1440.png      设置 · 界面设置
ref-settings-models-1440.png         设置 · 模型
ref-settings-builtin-1440.png        设置 · 内置插件
ref-settings-skills-1440.png         设置 · 技能
ref-settings-market-1440.png         设置 · 市场
ref-settings-remote-1440.png         设置 · 远程
ref-settings-mcp-1440.png            设置 · MCP
ref-settings-presets-1440.png        设置 · Agent 预设
ref-settings-usage-1440.png          设置 · 用量统计
ref-settings-pet-1440.png            设置 · 鲸鱼娘
ref-settings-about-1440.png          设置 · 关于
ref-account-menu-1440.png            账户菜单
ref-plugins-1440.png                插件页（加载态骨架）
ref-schedule-1440.png              定时任务页（chip 组 + 搜索 + 新建）
ref-pet-page-1440.png              鲸鱼娘页（主区头部 + 对话/轨迹 + composer）
ours-hero-1440.png                   原生 hero（1440 物理 / 960 DIP 缩放）
```

## 6. 设置分页逐页对照（W10 复核，2026-10-20 第二轮）

对照方式：逐张看 `evidence/compare/ref-settings-*-1440.png`（安装版 0.3.9 实机截图），与本仓库 `internal/ui/settings` 的实现逐项比对。
"已修"指本轮 task-39 改过的项；未列的项即**尚未处理**。未看过的分页单独标注，不写印象。

### 外观（ref-settings-appearance）→ 已修
| 差异 | 状态 |
| --- | --- |
| 参考有「颜色模式」三卡（浅色/深色/跟随系统，图标+标题，选中描边） | 本仓库已有，本轮补 tester 断言（`TestAppearanceOffersThreeColorModes`） |
| 参考有「主题」网格：每族一张卡，上半浅色预览/下半深色预览 + 名称 + 当前模式徽标 | **已修**：`themeLibraryPane` 重写为两列网格，双半可点、当前主题加冠 |
| 参考右上「创建主题」「导入主题」两个 sm 按钮 | **已修**：Ghost sm / Outline sm；未接导入时 disabled + tooltip |
| 参考两个按名称下拉（浅色主题/深色主题） | 保留，收进「按名称选择主题」折叠 |

### 通用（ref-settings-general）→ 部分已修
| 差异 | 状态 |
| --- | --- |
| 弹窗 800 宽 | 已对齐（`settingsModalSize` min(800, w-48)） |
| 左侧导航 180 | **已修**（原 188），并给 rail 加 key 便于测量 |
| 内容列 x=532 | **已修**：rail 180 + 16 gap + 面板 16 padding → 532 |
| 行高 64 | **已修**：两个行原语加 `MinHeight(64)` |
| 控件高 32 | **已修**：host 偏好下拉、schema 枚举/数字输入加 `Height(32)` |
| 分组底部 hairline | 已有（0.5px 前导线，首行不画） |
| `打开配置文件` Outline sm + 关闭 X | **已修**：新增 `Model.OnOpenConfig`，nil 时 disabled + 说明 tooltip；X 已有 |
| 行内齿轮（发送消息时的思考炫光） | 未做（需按命名空间挑出带自定义的项） |

### 模型（ref-settings-models）→ 已修（task-43/45）
| 差异 | 状态 |
| --- | --- |
| 账号卡（DeepSeek 账号 + 编辑） | 未做（账号信息在 account 分页） |
| 供应商卡：名称+provider id 标题行、API 密钥输入、**橙色"未检测到环境变量"提示**、`> 自定义设置` 折叠、取消/保存按钮对 | 标题行已有；**按钮改 Outline sm「编辑」（已修）**；橙色环境变量提示、每卡取消/保存、自定义设置折叠**未做**（需要 store 提供"是否检测到环境变量"与按卡草稿/保存语义） |
| 虚线 `+ 添加模型提供商` | 未核对渲染样式（`configurationCreateProvider` 存在） |
| 识图模型卡（标签 + 下拉 + 灰字说明） | 已有 `visionPane`，未逐项比对样式 |

### 鲸鱼娘（ref-settings-pet）→ 已修（task-42/45）
| 差异 | 状态 |
| --- | --- |
| 顶部两段 segmented（聊天与能力 / 桌面形象与行为，等宽白药丸） | **已修**：`settingsSegmented`（38 高、白药丸、等宽段），宠物页两段已替换 |
| 「助理」分节标签 | 已有（`bodyHeading`），本轮保留 |
| 名字与称呼：两个并排输入；默认模型：两个并排下拉 | **未修**：`pet.Companion` 与 settings 域没有名字/称呼/默认模型字段，需 store 提供（见 §7 缺字段清单） |
| 额外人设/长期记忆/技能 行右侧**蓝色文字动作**（编辑 / 查看与编辑 / 管理） | **未修**：无对应回调（persona/memory/skills 不在 settings 域） |
| 行间 hairline + 行高 64 + 控件 32 | 随通用页原语已生效 |

### 界面设置（ref-settings-interface）→ 已修（task-46）
| 差异 | 状态 |
| --- | --- |
| 纯开关行（44×24 蓝色），右侧开关 | **已修**：settings 域自绘 `settingsToggle`（44×24、20 卡榫、2 内缩），`bodyLocalToggle` 全部走它 |
| 某些行有行内齿轮（更多设置） | 未做 |
| 「机器人（Bots）」行带「测试中」徽标 | **已修**：`bodyToggleRowBadge`，标题右侧小徽标，行高仍 64 |
| 行间 hairline | 已有 |

### 内置插件（ref-settings-builtin）→ 已修（task-46）
| 差异 | 状态 |
| --- | --- |
| 参考是 **2 列卡片网格**，加载态是骨架卡（两条灰条） | 本仓库是文本行；骨架网格未做 |

### 技能（ref-settings-skills）→ 已修（task-44）
| 差异 | 状态 |
| --- | --- |
| 内容区头部三个动作：`浏览 GitHub 技能`（Outline sm + 放大镜）、`+`、刷新 | **已修**：`skillsBrowseButton`（28 高/radius 8/0.5 描边/12px）；无 Hub 回调时 disabled + tooltip；`+` 与刷新沿用 `bodyIconAction` |
| 搜索框 + `全部` 下拉同一行（下拉约 88 宽） | **已修**：搜索 32 高、占宽、占位「搜索名称、描述、分组或何时使用」；下拉 88 宽、首项「全部」 |
| 分组标题「工作区与个人技能 7 项」（名称 + 计数 + 项） | **已修**：单一未分组时也画该行（13px muted）；有分组时同样格式 |
| 每行：24 图标 + 名称 14 semibold + 描述 12 muted 单行省略 + `捆绑` 徽标 + 44×24 开关；行高 64、无分隔线 | **已修**：图标 24、名称 600、描述单行省略、`skillSourceBadge`（白底 + 0.5 描边）、`settingsToggle` 44×24、行 MinHeight(64)、无分隔线 |
| 启停语义 | 未改：仍调 `OnSetModelInvocable` / `OnSetGroupInvocable`，只是把开关换成自绘 44×24（点击后由调用方取反上报） |
| 空态/加载态 | 沿用现有（空态一句 muted 文案；加载仍是 Busy 文案），参考无更强证据 |

### 市场（ref-settings-market）→ 入口移除（官方市场）
| 差异 | 状态 |
| --- | --- |
| 标题「插件市场」+ 说明 + 右上单个刷新 icon | 本仓库头部有标题/刷新，但无该说明句与内容区右对齐 layout |
| 参考截图正文为空白（加载/滚动区） | 无法据此判断卡片层，需另采已加载态 |

### 未查看（本轮不写结论）
`ref-settings-remote`、`ref-settings-mcp`、`ref-settings-presets`、`ref-settings-usage`、`ref-settings-about` —— 五页本轮**没有逐张看**，故不列差异；远程页在 W9 做过结构改造（机器卡/文件页/配对步骤），但那是对代码而非对参考图。

### 远程（ref-settings-remote）→ 未修（结构完全不同）
参考是**网关配置页**：顶部两个 tab「网关 / 消息渠道」；行内是「中继主机」输入 + 黑色 sm「保存」+ 文字动作「恢复默认」；「旧版配置清理」分节；「连接方式」右侧「局域网」标签 + 黑色「服务器」按钮；「监听端口」输入 + 黑色「保存」；「监听范围」下拉（仅本机）；「传输加密」纯文本行；「刷新配对码」右侧文字动作「刷新」。
本仓库同页是**机器列表 + 文件浏览 + 配对**（W9 重做过），与参考的信息架构不同：参考没有机器卡，只有网关参数。→ 差异属"页面定位不同"，不是样式差；若要还原，需要把机器管理整体移出设置页。

### MCP（ref-settings-mcp）→ 已修（task-46）
参考：标题「MCP 服务器」+ 两行说明；内容区右上 `+` 与刷新两个 icon 按钮；搜索框（带放大镜）+ `全部` 下拉；空态是**虚线圆角卡片**内居中一句「还没有 MCP 服务器。添加一个服务器以连接本地工具或远程服务。」
本仓库：有搜索与列表，但缺内容区右上两个 icon 按钮与虚线空态卡样式。

### Agent 预设（ref-settings-presets）→ 已修（task-47）
参考：「内置」分节标签 + **2×2 预设卡**（约 275×165：标题行＝名称 + 徽标「任务默认」/「内置」+ 右侧 muted id；3–4 行描述；底部 `模式说明` / `如何使用` 链接 + 右侧书本 icon）；「自定义」分节 + **虚线整宽按钮**「+ 让 Agent 帮我创建预设模式」。
本仓库：预设页存在（agent_presets.go / agent_preset_guide.go），但未对照卡片网格/徽标/底部链接这个形态改造。

### 用量统计（ref-settings-usage）→ 已修（task-47）
参考：标题行＝图表 icon + 「用量统计」，下方「数据更新于 11:06 · UTC」；右侧 `导出`、`设置`、`刷新` 三个带 icon 的小按钮；正文空态是**白卡片**：「暂无统计数据」+「开始使用 DeepSeek Harness 后，这里会展示 Token 消耗情况」。
本仓库：W1 已做成 KPI 卡 + 日历 + 排行（内容更丰富），但空态样式与右上三按钮组未对齐参考（参考的空态是卡片，不是虚线框）。

### 关于（ref-settings-about）→ 已修（task-47）
参考：标题「关于」+ 一句说明；「Whale Isle」（16 bold）、「桌面版本 0.3.9」、「Based on DeepSeek Harness」；一行版本更新（禁用输入 `...` + 深色徽标「在线安装」）；「凭据存储：系统钥匙串加密（safeStorage）」、「运行目录 …」；`打开运行目录` 按钮；两行**蓝色链接** `GitHub 发行版`、`官方 Harness`。
本仓库：about.go 有版本/凭据/目录/检查更新，但排版与参考的左对齐单列＋蓝色链接行不同，且参考的「在线安装」徽标未实现。

## 7. 本轮修复进度（2026-10-20 晚）

截图：`ours-check.png`（修复后真实窗口，1440×920 物理 = 960×613 DIP）对照 `ref-hero-1440.png`（安装版 1440×920 DIP）。

### 已修（壳层）

| 项 | 结果 |
| --- | --- |
| 侧栏默认展开 | 根因是断点写成 `width < 1024`；改为 `<768`（原版 `columns.ts` 的 narrow 只在非 landscape 或 < PHONE_MAX 时成立）。960 DIP 下保持 280 DIP：品牌区、+ 新会话、插件/自动化任务/鲸鱼娘、工作区分组、底部 设置/账号与登录 全部回来 |
| 导航行距 | 40 → **36**（去掉 4 DIP gap） |
| 调试泄漏 | `no workspace folder ready` 不再写入状态；失败路径改用本地化 `KeyComposerDirNoWorkspace` |
| 默认背景 | 新增 `wallpapers.PresetColors("dune")`（采样安装版）；渐变改为全幅 wash + 5 个柔和径向 bloom，去掉 `p.Shadow` 造成的硬边圆斑；相邻像素最大跳变 70→3/255 |
| 标题栏 | ≥768 构建 trailing：分支胶囊 + `提交并推送` + 终端/右栏切换；最小化/最大化由 frame 自绘（原生 `shownButtons()` 只在窗口带 WS_MIN/MAX style 时给三键，本窗口没有），加自绘关闭；真实窗口已见 终端/右栏/最小/最大/✕ |
| 滚动条 | `ScrollbarAlways = true → false`，恢复“仅 hover/拖动出现” |
| 启动器行 | 侧栏树里的「启动器」行删除（原版没有；功能保留在标题栏 `…` 菜单） |

### 已修（composer）

- hero 卡片：圆角 28、白 80%、内边距 20/16、占位 14/24、高 100 DIP；底部只留 `+`/权限/模型/34×34 圆形发送键。
- 空态隐藏麦克风与上下文百分比（有配置/用量时恢复）。
- 工作区入口与 preset 同一行：左「工作区名/选择工作区」+ 右「标准模式」；去掉第二个「无工作目录」。
- 会话无目录时 harness 把 cwd 放在 `.../dsh-home/no-workspace`，现在识别该哨兵并显示「选择工作区」。

### 已修（设置）

- 外观：颜色模式三卡 + 主题网格（浅/深双预览、当前卡描边与 ✓、创建/导入按钮，导入未接时 disabled+tooltip）。
- 通用：导航 180、内容列 x=532、行高 ≥64、控件高 32、`打开配置文件` Outline sm。
- 鲸鱼娘：两段 `settingsSegmented`（38 高白药丸）。
- 界面设置：自绘 44×24 开关；Bots 行「测试中」徽标。
- 模型页：供应商卡动作改 Outline sm「编辑」。

### 仍未修（需要数据或其他域）

1. **模型页**：橙色“未检测到环境变量”、每卡取消/保存、`> 自定义设置` 折叠、虚线添加卡——需要 store 提供环境变量探测与按卡草稿。
2. **鲸鱼娘页**：名字/称呼、默认模型双下拉、IM 开关、额外人设/长期记忆/技能三行——`pet.Companion` 与 settings 域缺字段/回调。
3. **界面设置**：行内齿轮（需要“是否有额外设置”标记）。
4. **内置插件**：2 列卡片网格 + 骨架加载态；**技能**：三头部动作/`全部` 下拉/分组计数/`捆绑` 徽标；**市场**：说明句与内容区布局（需已加载态参考）。
5. **远程设置**：原版是网关配置页（中继主机/端口/连接方式/加密），本仓库是机器列表/文件/配对——页面定位不同，需产品决定是否搬移。
6. **MCP/Agent 预设/用量/关于**：差异已逐条记在 §6，未修。
7. **Project 空态块**：侧栏有 Project 段位置，但 `internal/ui/projects` 没有空态内容，需在 projects 域补。
8. **背景色浓度**：当前 bloom 比参考略浓（青/绿更饱和），可再降 10–20% 不透明度。

## 8. 结论

第一眼的壳层差距（侧栏、标题栏、背景、composer、调试文案）已经收敛；剩余主要是**需要数据模型支持的分页内容**（模型/鲸鱼娘/技能/内置插件）和**页面定位差异**（远程）。这些不适合再用“改样式”解决。
## 9. 第三轮：数据层 + 三页收口（2026-10-20 夜）

### 数据层

- **dsh-whale 客户端** `internal/desktop/whale`：`New/Catalog/Update/ReplaceMemory/ClearMemory/ToggleSkill`，httptest 11 条用例。字段按插件源码：catalog 扁平（modelProvider/modelModel/modelReasoningEffort），update 只认嵌套 `model:{...}`，这个不对称被测试锁住；memory 上限按 UTF-16 40000 本地拦截；skills 名称本地校验并透传插件原因。
- **接线** `cmd/whaleisle/pet_whale.go`：按 origin+cookie 缓存 catalog，打开设置页后台拉取（15s 超时、失败 5s 退避），写操作后强制重拉；Host 未就绪时 `Whale` 与 5 个回调全 nil。
- **模型页数据** `internal/store/settings`：`ProviderEnv(ns)`（显式映射：llm-deepseek→DEEPSEEK_API_KEY、openai/anthropic/google/gemini/openrouter/groq/mistral/xai/azure；已 describe 的 apiKeyEnv 优先）、`NamespaceDraft(ns)`、`SaveNamespace(ctx,ns)`；确认 drafts/dirty 本就按 ns 隔离，补的是访问面。`ModelsAccess.ProviderEnv` 已在 main.go 接线。

### 三页（tester 快照见 evidence/compare/ours-settings-*.png）

| 页 | 已实现 | 未实现/差距 |
| --- | --- | --- |
| 鲸鱼娘 | 名字/称呼双输入（失焦或回车 Patch）、默认模型只读显示（空=跟随会话默认/在模型页选择）、IM 开关、额外人设行内编辑、长期记忆对话框（保存/清空二次确认）、技能列表 + 44×24 开关、Whale==nil 占位 | 默认模型下拉缺“选项列表”数据（catalog 只给当前路由）；旧 `RenderAssistant` 字段待清理 |
| 模型 | 凭据字段优先（API 密钥/密码框/env 占位）、橙色未检测提示（含变量名）、`> 自定义设置` 折叠、卡内取消/保存（仅本 ns）、每卡独立编辑态、添加卡 | 添加卡是 1px 细边框而非虚线（mygo 无 dashed，需要 chrome 层 pattern）；识图卡内部未重排 |
| 技能 | 头部 3 动作（浏览 GitHub 技能 Outline sm + `+` + 刷新）、搜索 + 88 宽 `全部` 下拉、分组标题「… N 项」、行 24 图标/14 semibold/12 单行/`捆绑` 徽标/44×24 开关/64 高 | 空态卡无参考证据；行内仍保留打开目录/删除两个 icon（功能未丢） |

### 验证

`go build ./cmd/... ./internal/...`；`go test ./internal/ui/... ./internal/store/... ./internal/desktop/whale ./cmd/whaleisle` 全绿；`go test ./internal/ui/settings` 81 pass；`SETTINGS_GALLERY_DIR` 可重出两张设置页快照。
## 10. 第四轮：界面设置/内置插件/MCP/预设/用量/关于（2026-10-20 深夜）

截图：`evidence/compare/ours-settings-{interface,builtin,builtin-loading,mcp,presets,usage,about}.png`，与同目录 `ref-settings-*` 对照。

| 页 | 已实现 | 未实现/差异 |
| --- | --- | --- |
| 界面设置 | 光效行 `[齿轮][开关]`（齿轮打开与旧 Configure 相同的对话框，原子保存测试改为点齿轮）；其余行 44×24 开关 + hairline；Bots「测试中」保持 | 无 |
| 内置插件 | 标题 18 + 说明；2 列白卡（273×60、radius 12、12/16 padding）；加载态 2×2 骨架（120×10/180×8，muted 10%）；`pluginStatusLabel` 本地化（已启用/已停用/加载失败/按条件启用/状态未知） | 卡片第二行用状态而非参考的「版本/来源」；参考只采到加载态，无法逐项比对 |
| MCP | 标题 18 + 三行说明；右上 32×32 `+`/刷新；搜索 32 + 88 宽 `全部` 下拉（替换三 chip）；空态 **真 dashed** 圆角卡两行文案；删除 legacy 整宽刷新按钮 | Catalog 为 nil 时（Host 未提供目录）筛选行不渲染，参考始终显示；若要恒显示需把搜索/下拉做成 disabled 常驻 |
| Agent 预设 | 标题 18 + 说明；内置/自定义 13 muted 小节；2×2 卡（(w-12)/2、170 高、radius 12、白底、0.5 分隔线）；标题 14 + 任务默认黑徽标/内置描边徽标 + 右上 id 10 mono；描述 3 行；底部 12 muted 文字动作 `模式说明 · 如何使用` + 书本 16 icon；自定义整宽 46 高 **虚线**按钮 | 参考卡片底部左右分布更松（书本靠右）；其余一致 |
| 用量统计 | 标题行 图表 16 + 18 + 导出/设置/刷新 Outline sm；`UpdatedAt` 透传后显示「数据更新于 HH:MM · UTC」；空态白卡 H=108/radius 12/0.5 描边/轻阴影，14 semibold + 12 muted | 导出/设置 disabled+tooltip（无导出器/范围对话框）；空态卡尺寸常量，测试断言文案与上下关系 |
| 关于 | 标题「关于」18；应用名 16 + 桌面版本 + Based on；版本只读下拉 150 + 深色「在线安装」徽标；凭据存储/运行目录 12；`打开运行目录` 28 高 Outline 内容宽；两条 14 accent 链接（GitHub 发行版/官方 Harness） | 应用数据与诊断折叠为额外项（参考没有）；凭据状态取决于平台后端 |

验证：`go build ./cmd/... ./internal/...`；`go test ./internal/ui/... ./internal/store/... ./internal/desktop/whale ./cmd/whaleisle` 全绿；`SETTINGS_GALLERY_DIR` 可重出全部设置页快照。
## 11. 第五轮：停做自有市场，接入 dsh 官方插件市场（2026-10-20 深夜）

产品决定：不再维护自有“市场”界面，插件发现统一走 dsh 官方市场 `https://awesome-dsh-plugin.com`（也就是 `internal/desktop/market/catalog.go` 里 `CatalogURL` 的同一份官方 registry）。

| 位置 | 改动 | 保留 |
| --- | --- | --- |
| 设置左栏 | 移除“市场”入口（`planSectionOrder` 删除 SectionMarket） | `SectionMarket` 常量、`market`/`ui-settings-market` 深链映射、market 面板渲染 |
| 插件页 | 移除“已安装 / 发现”双 tab，只显示“已安装”；`discoverView` 代码保留但 UI 不可达 | 已安装列表、可更新徽标与“更新”按钮、安装/卸载流程 |
| 插件页 ⌄ 菜单 | 新增「插件市场」：描述“在 dsh 官方市场浏览插件（awesome-dsh-plugin.com）”，点击 `OnOpenURL` 只开浏览器、不切页；无浏览器能力时 disabled+tooltip | “安装已有插件”“让 Agent 创建插件”、AddActions |
| 代码 | 未删 `internal/ui/market`、`internal/desktop/market`、`internal/store/market` | 更新检查、安装管线、封面/详情等底层能力 |

测试：`pluginmanager` 新增“无发现 tab / 菜单含插件市场 / 点击只回调 URL 且不改 state / 无回调时无副作用”四条；settings 新增“导航无市场但深链仍解析并渲染”。

验证：`go build ./cmd/... ./internal/...`；`go test ./internal/ui/... ./internal/store/... ./internal/desktop/whale ./cmd/whaleisle` 全绿。
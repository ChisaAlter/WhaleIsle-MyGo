# 输入区与发送按钮（composer）· 原生 vs 原版鲸屿

审查范围：输入条本体（卡片/输入框/工具栏/发送按钮/状态行）与它的两个浮层（`/` 命令、`@` 引用）以及附件条。
本文件只读审查，未改动任何源码；唯一写入就是这个 md。

**证据来源**

| 代号 | 位置 | 说明 |
| --- | --- | --- |
| 截图·原版 | `docs/review-20261020/evidence/original-electron-3080.png`（1441×912） | Electron 0.3.9 实际运行，深色主题，会话运行中（输入为空） |
| 截图·原生 | `docs/review-20261020/evidence/native-go-3090.png`（1298×822） | `whaleisle.exe` 实际运行，浅色主题，新会话（输入为空，空闲） |
| 原版·源码 | `runtime/vendor/deepseek-harness/packages/client/...`（alpha.1 TSX/CSS） | 结构、token、尺寸真值 |
| 原版·构建 | `C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\...\lib\client.js`（rc.2） | 只读核对：线上组件是否与 alpha.1 一致（差异处标 `version-drift`） |
| 原生·源码 | `internal/ui/composer/*.go`、`internal/ui/frame/frame.go`、`internal/ui/sessioncost`、`third_party/mygo/ui` | 现状实现 |

> 两个截图主题/状态不同：凡是可比的（几何、字号、色值、结构）都在条目里说明可比性；受状态影响的（如发送按钮此时是"停止"）不当作缺陷。

---

## 一、5 个最致命差距

1. **卡片下方的状态行（dock）退化成双语调试串，且丢失了上下文百分比与统计口径。**（明细表 #1）
   原版是居中、与卡片同宽的两簇紧凑统计（截图 y≈897：`12 轮 280 步 · 237 tok/s`、`77.7M tok · 缓存命中 99.7%`，`StatsPills.tsx:242-253` / `:293-309`）；原生是左对齐的一整行 11px 双语串 `含子代理计费 Token / Billed tokens incl. subagents · 命中 / Hit 0 · 未命中 / Miss 0 · 输出 / Output 0`（`internal/ui/sessioncost/view.go:65,79-86`），行首实测 x≈422，比卡片左缘（x=430）还靠左 8px，没有居中、没有 pill、没有"轮/步/tok/s"。

2. **权限档位 pill 直接显示 Host 机器值 `workspace-write`，且丢了盾牌图标与中文产品名。**（明细表 #2）
   原版这一格是"图标 + 中文产品名 + 12px chevron"的 28px chip（截图 x≈490–580：`⛨ 完全权限 ⌄`；`PermissionSelect.tsx:198-216`、`PermissionSelect.module.css:1-32`），内置档位由 `displayPermissionPreset()` 映射为 `仅可查看 / 工作区内修改 / 完全权限`（`presentation.ts:15-19,44-54`、`locales.ts:12-14`）；原生把 `PermissionOption.Label` 原样画出来（`internal/ui/composer/preset_picker.go:16-24,52-58`），于是成了英文 id。

3. **模型 seat 在目录未加载时根本不是控件：点不开、没有 ▾，加载后也不是原版那个 chip。**（明细表 #3、#4）
   原版永远渲染同一个 trigger 按钮：加载中显示 `trigger.loading`，点开是菜单（空态 + 重试）；就绪后是 28px 透明 chip，`label-secondary` 模型名 + `label-caption` 强度 + 12px chevron（`ModelSelect.tsx:451-456,486-510`、`ModelSelect.module.css:9-43,64-75`）。原生在 `ModelLabels` 为空时只画一段文字、没有按钮（`internal/ui/composer/model_picker.go:24-31`），本次截图正是这个状态——用户看到 `deepseek-official / deepseek-flash` 却点不了；目录就绪后用的是 mygo Button 默认外观，也没有 chevron（`model_picker_grouped.go:179-193`）。这是输入条上唯一"功能入口会消失"的地方。

4. **上下文占用指示（原版 `◔ 49%`）在原生里既没有环形，也经常整块消失。**（明细表 #7、#8）
   原版是 14px 双环（`track`=border-l3、`fill`=label-tertiary，2px）+ 13px tabular-nums 百分比，hover 有底色（`ContextMeter.tsx:152-163`、`ContextMeter.module.css:9-44`）；原生只有一个文字按钮 `上下文 49%`（`context_meter.go:23`），并且 `ContextOccupancy == nil` 时整块 `return`（`context_meter.go:14-21`）——本次原生截图（会话未产生 contextPressure 帧）里这格完全不存在。

5. **`/`、`@` 弹层被画进输入卡片内部，而不是浮在卡片上方；行样式也退化了。**（明细表 #5、#6）
   原版是定位在卡片上方的独立浮层：`bottom: calc(100% + 4px)`、与卡片同宽、`elevation-prominent` 阴影、320/400px 高度上限、分组标题、右侧灰描述、Tab 键帽提示、加载骨架（`MenuView.module.css:1-29,51-82,159-168`、`PopupSelectView.module.css:3-31,40-53`）；原生在同一张卡里 `MaxHeight(180)`、`Radius(6)` 的手写面板，选中行用 **muted 实色**做底（`composer.go:851-862、865-874`），卡片会随弹层长高。

> P1 里最该先修的（差一点进前五）：发送/加号/停止按钮的 hover 与按下反馈完全失效，见明细表 #9。

---

## 二、明细表

共 17 条：**P0 2 条 / P1 5 条 / P2 10 条**。编号按输入条上的区域顺序排列（状态行 → 权限 → 模型 → 弹层 → 上下文 → 按钮 → 卡片 → 其余细节），排期优先级以"严重度"列为准。

| # | 项目 | 原版行为（证据） | 原生现状（证据） | 差距 | 严重度 | 修复方向 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 卡片下方状态行（dock） | 与卡片同宽、居中、两簇统计胶囊（`radius:999px`、透明底、仅 hover/展开填色，12px 字），`gap:4px 12px`，`·` 分隔；`12 轮 280 步 · 237 tok/s` 与 `77.7M tok · 缓存命中 99.7%`（截图 y≈897；`InputBar.module.css:13-26`、`StatsPills.module.css:19-51`、`StatsPills.tsx:242-253,293-309`） | 单行左对齐 11px 文本，`Padding(4,8,4,8)` 直接铺在外层列里（`sessioncost/view.go:79-86`；`frame.go:1116-1123`）；文案为 `含子代理计费 Token / Billed tokens incl. subagents · 命中 / Hit 0 · 未命中 / Miss 0 · 输出 / Output 0`（`sessioncost/view.go:65,83`）；行首实测 x≈422 <卡片左缘 430 | 结构、对齐、文案、数字格式（无 K/M 紧凑化，`%.0f` 原样）全不同；缺"轮/步/tok/s"与缓存命中胶囊 | P0 | 新增 composer dock 组件：`ui.Row` 居中 + 两个胶囊命中区；数字走紧凑格式（77.7M / 99.7%）；文案只保留中文（或 locale），不要 `中 / EN` 拼接 |
| 2 | 权限档位 pill | 28px chip：可选 14px 图标 + `FlipText` 中文名 + 12px chevron（开时旋转 180°），`label-secondary` 13/20 500，hover `interactive-bg-hover`，`:focus-visible` 2px 环（`PermissionSelect.tsx:198-216`、`PermissionSelect.module.css:1-32,79-101`）；内置值 → 产品名映射（`presentation.ts:15-19,44-54`） | `presetSelect` 直接渲染 Host `Label`（`preset_picker.go:16-24,52-58`），截图为 `workspace-write`；无图标；chevron 是 `ui.Text("▾")`，不旋转（`preset_picker.go:23`）；trigger 来自 `ui.SelectBase`，底座 `coreButtonBase` 只做交互、不带任何外观，`presetSelect` 也没自绘 hover（`third_party/mygo/ui/base.go:21-30`） | 中文产品名、图标、chevron 状态、hover/焦点反馈全缺；这是用户每天都看到的一格英文机器值 | P0 | 在 `permissionChoices()` 里先过一层映射（复用/修正 `permission_command.go:8-26` 的 `nativeOptionLabel`），trigger 前加盾牌 `ui.Icon`，chevron 用 SVG 并在展开时旋转；hover 用 `Draw`+`Hovered()` 自绘或给 `ui.SelectBase` 增加 hover token |
| 3 | 模型 seat 的外壳与可点性 | 无论目录是否就绪都渲染同一个 trigger 按钮（加载中显示 `trigger.loading`），点击才出菜单；28px chip、透明底、hover 才填色、12px chevron（`ModelSelect.tsx:451-456,486-510`、`ModelSelect.module.css:9-43`） | `ModelLabels` 为空（目录未加载/加载失败）时**只画一段文字**，没有按钮、没有 chevron、点不开任何菜单（`model_picker.go:24-31`）——本次截图正是这个状态；目录就绪时才是 `ui.Button`（`Height(28)`、`TextColor(muted)`、`MaxWidth(220)`），沿用 mygo Button 默认外观（Surface 填充 + 默认内边距/圆角，`third_party/mygo/ui/widgets.go:182-198`），没有按原版做透明底 chip（`model_picker_grouped.go:179-193`） | 目录不可用时模型**无法切换**（不是样式问题，是入口消失）；有目录时外壳（填充、内边距、chevron、hover）也与原版不同 | P1 | 任何目录状态都渲染同一个 28px chip trigger：未加载显示"加载中…"并允许打开菜单（空态给"重试/刷新"，原生已有 `OnRetryModels`）；底色改透明、hover 才填 `interactive-bg-hover`，尾部补 12px chevron |
| 4 | 模型名/加载态文案与两段式着色 | 加载中显示 `trigger.loading`，没有任何选择时显示 `trigger.fallback`，只有"已选但目录里找不到"才退回 `provider/model`；模型名 `label-secondary`、强度后缀 `label-caption`（`ModelSelect.tsx:451-456`、`ModelSelect.module.css:64-75`） | 无论"目录尚在加载"还是"目录缺失"，都直接画 `provider + " / " + model`（`model_picker.go:24-34`、`models.go:94-101`）；截图即此态：`deepseek-official / deepseek-flash`；模型名与强度两段都用 `muted`（`model_picker_grouped.go:183,188`） | 加载中/未选择两档文案缺失（原版是本地化短句），分隔符写法与原版 `${provider}/${model}` 也不同；两段同色，失去"主名 + 次色强度"的层级 | P2 | 补 `加载中…` / `选择模型` 两档文案；路由串仅在"已选但目录缺失"时显示且统一成 `provider/model`；强度后缀改用 `Caption` 色 |
| 5 | `/`、`@` 弹层位置与外壳 | 浮层，`position:absolute; bottom:calc(100%+4px); left/right:0`（与卡片同宽），`max-height:320/400px`，`elevation-prominent` 阴影 + border-l1 发丝，4px 内边距，超出时底部渐隐（`MenuView.module.css:1-42`、`PopupSelectView.module.css:3-31`） | 面板放在 `card.Children` 里的最上方，属于卡片内布局流：`MaxHeight(180)`、`Border(1,border)`、`Radius(6)`、`Padding(2)`，无阴影（`composer.go:851-852`；调用点在 `card.Children` 内 `composer.go:254-256`） | 位置（卡内 vs 卡上）、尺寸上限（180 vs 320/400）、阴影/圆角/发丝全不同；打开时卡片会瞬间长高 | P1 | 改用 `ui.Popover` 锚在卡片上缘（原生已有 `paintNativeCommandPopup` 的 Popover 用法可参照），面板 320px 上限 + elevation 阴影 |
| 6 | 弹层行样式与信息层级 | 行高 `min-height:34px`、13/20、`radius-md`，唯一高亮是 `interactive-bg-hover`（键盘与指针共用一个 highlight）；分组标题 11px；描述右对齐 12px 灰；drillable 行才显示 Tab 键帽；有骨架行与面包屑（`MenuView.module.css:51-82,140-199,232-246`） | `ui.Column` 行 `Padding(6,8,6,8)`，标题 12px **Bold** + 副标题 11px，选中底色是 `muted`（真色值）而非 hover 叠加（`composer.go:858-874`） | 选中态用中灰实色做底，深色字压在上面，对比度和原版完全不同；无分组标题、无骨架、无键帽提示、无右侧描述对齐 | P1 | 选中态改成 `Interactive-bg-hover`；行高统一 34px；补分组标题与 Tab 键帽（`nextPrompt` 已有键帽样式可复用） |
| 7 | 上下文占用指示 | 14px 环（2px stroke，track=border-l3 / fill=label-tertiary）+ `49%` 13px tabular-nums，外层 `padding:1px 8px`、`radius-sm`，hover 有底色（`ContextMeter.tsx:152-163`、`ContextMeter.module.css:9-44`） | 文字按钮 `上下文 49%`，`Height(28)`、`FontSize(12)`、无环、无 hover 态（`composer.go:455` → `context_meter.go:23`）；`ContextOccupancy==nil` 时整块消失（`composer.go` 同处 / `context_meter.go:14-21`），本次截图该格不存在 | 视觉标识（环）缺失、文案多了"上下文"三字、位置在 trailing 组内但样式不成 pill | P1 | 按原版画 14px 双环 SVG + 百分比；缺数据时保留占位（`—%`）而不是整块消失；面板颜色改用主题 token（见 #8） |
| 8 | 上下文面板配色 | 面板用 `elevation-prominent` 菜单面、分段色来自主题 `colorSystem/colorTools/colorMessages`，数字紧凑格式（`ContextMeter.tsx:31-35,185-208`） | 面板里硬编码 `RGB(210,210,210)` 轨道与 `RGB(99,102,241)/RGB(14,165,233)/RGB(16,185,129)` 三段（`context_meter.go:32,40`） | 深色主题下浅灰轨道 + 固定靛蓝色与主题不符；`~%g` 输出未紧凑化 | P2 | 换主题 token；数字走紧凑格式；面板套用原生已有 Popover 面板样式 |
| 9 | 发送 / 加号 / 停止按钮的交互反馈 | `:hover` 换 `button-info-hover`（深色 `#679EFE`→更亮），`.add:hover` 换 `interactive-bg-hover-solid`，`.primary:disabled{opacity:.4}`，过渡 100ms（`InputBar.module.css:434-455,492-521`） | 三个按钮都显式 `Background(infoFill/selectorFill)`，而 `styleButton` 的 hover/pressed 逻辑在 `b.bg != base` 时直接 `return`（`third_party/mygo/ui/widgets.go:199-208`；调用点 `composer.go:438,473,482`），所以**完全没有 hover/按下反馈**；禁用透明度为框架统一 0.5（`third_party/mygo/ui/paint.go:123-125`） | hover/按下态丢失（原版各按钮都有）；禁用透明度 0.5 vs 0.4（实测原生发送盘 ≈(159,186,242)=0.5 叠加） | P1 | 给 `ui.Button` 增加"自定义底色 + hover/pressed 变体"能力（例如 `BackgroundWithHover(base,hover,pressed)`），或在 composer 里用 `Draw`+`Hovered()` 自绘 |
| 10 | 发送/停止图标 | 16×16 SVG：上箭头路径、矩形 `rect(3,3,10,10,rx=3)` 停止块（`InputBar.tsx:604-611`） | 文字字形 `"↑"`/`"■"`：`FontSize(16)`/`FontSize(12)`，靠字体基线居中（`composer.go:471,480`） | 笔画粗细、箭头形状、停止块圆角都与原版不同；字形居中依赖字体度量，比例易漂 | P2 | 换成 16×16 SVG（原版路径可直接搬），保证两态同心 |
| 11 | 权限内置档位中文名 | `workspace-write` → `工作区内修改`，`danger-full-access` → `完全权限`，`read-only` → `仅可查看`（`presentation.ts:15-19`、`locales.ts:12-14`） | 原生已有映射但写成 `工作区写入`（`permission_command.go:17-19`），与产品词不一致；且该映射只用于 `/permission` 弹层，未用于输入条 pill | 同一个档位在两种界面里出现两种中文名，且与线上产品词不同 | P2 | 统一词表（`工作区内修改`），映射函数下沉成 store/composer 共用 |
| 12 | 权限下拉细节 | `Menu`（portal，`side="top"`）+ 选中勾 + badge + 描述；trigger `:focus-visible` 2px 环（`PermissionSelect.tsx:124-135,189-216`） | `ui.SelectBase` 弹出 `Padding(Space1)`、`Radius+2`、`Border(1)` 的自绘面板；高亮用 `Accent` 实色（`preset_picker.go:26-32`） | 弹层外壳（无阴影/毛玻璃）、选中语义（Accent 实底 vs 勾选）、键盘焦点环未确认 | P2 | 复用原生 Popover 菜单外壳；选中行改用淡底 + ✓ |
| 13 | 附件条 | 图片 chip 64×64、`radius-xl(20)`、0.5 发丝、**加载期有 `interactive-bg-hover` 底色**；关闭键 18px 圆形、`button-contrast-fill` 底；左右滚动箭头（`ComposerAttachments.module.css:11-22,40-46`；文案 `locales.ts:34-36`） | 图片 chip 64×64/`Radius(20)`、边框一致（`attachment_display.go:217,228`），但加载期无底色；关闭键 18px 仅 `borderWidth(0)` 透明底、hover 才显形（`attachment_display.go:293-296`）；无左右滚动箭头 | 空态/加载态观感不同；关闭键在图片上对比度不足；无滚动提示 | P2 | 补加载底色与关闭键底色（对比色圆底）；`ScrollHorizontal` 加左右箭头或遮罩渐隐 |
| 14 | 工具栏麦克风 | 原版这一行没有麦克风：左侧只有 `+` → 权限 → Plan（截图 x≈490 起即权限 pill；`InputBar.tsx:538-571`） | 原生在 `+` 与权限 pill 之间常驻一个 28px 圆形麦克风（`tool_actions.go:68-100`，`OnStartVoice` 非空即显示），截图 x≈460–490 | 左组多一个强视觉控件，权限 pill 被推右；与原版行内节奏不一致（语音是鲸屿自有能力，位置需另定） | P2 | 语音入口收进 `+` 菜单/右侧 activity 席位，或至少与 `+` 之间用同一 `gap:12` 且去掉圆底 |
| 15 | 发送/停止 tooltip 与快捷键提示 | tooltip 文案会随状态切换 `发送消息 / 排队发送 / 插话发送 / 停止生成`，并带 `shortcutKeys`（停止的连按 Esc 提示）（`InputBar.tsx:372-391,580,595`） | 固定 `i18n.T(KeyComposerSend/Stop)`，`Tooltip` 不带快捷键（`composer.go:471-474,480-483`）；双层 Esc 停止逻辑已实现（`stop_sequence.go`）但界面无提示 | 用户看不到"正在排队还是插话"，也发现不了 Esc×2 停止 | P2 | tooltip 按 `primarySubmitMode`/`running` 选文案；`ui.Tooltip` 支持键帽段（Esc Esc） |
| 16 | 拖拽与粘贴提示 | 悬停拖入时是整卡 overlay：标题 `文件或图片拖动到此处即可添加` + 限额说明 + 被拒态（`locales.ts:37-42`；`InputBar.tsx:278,497-508`） | 拖入只在文本区下方追加一行 `KeyComposerDropHint` 11px 文字（`composer.go:418-426`），引用拖拽才有一行 `添加文件引用 / Add file reference` | 缺少 overlay 与限额文案，用户无法预知"能放几个/多大" | P2 | 在卡片上加 `DragOver` 时的半透明描边 + 居中提示层，文案带当前限额 |
| 17 | 输入卡材质与圆角 | `border-radius: var(--dsw-radius-panel)` = **28px**；`box-shadow: inset 0 0 12px 1px rgba(255,255,255,.25), 0 0 0 .5px border-l2`；填充 80% `input-major`（`base.css:29`、`design-platform.css:270-271,396-397`、`InputBar.module.css:97-109`；rc.2 一致：`C:\软件\Whale Isle\...\ui-theme\lib\client.js:6863`、`ui-conversation\lib\client.js:23810`） | `Radius(22)`，`.Draw` 只画 beam、`.DrawOver` 只画 0.5px 描边，**没有内高光**（`composer.go:330-339`） | 圆角小 6px（原版 28 vs 原生 22）；暗色主题下原版顶部可见的 12px 白色内高光缺失（原版截图 x=700 处 y=730≈(74,66,82) 向下渐深至 (49,47,50)，即内高光）；玻璃填充 80% 与描边 0.5px 两边一致，不算差距 | P2 | `Radius(28)`；在 `card.Draw` 里叠加 `inset 0 0 12px` 白色 25% 内阴影（可用圆角矩形描边 + 模糊近似） |

---

## 三、待验证项

1. **输入卡高度差（未定性）**：原版截图卡片实测 x 475–1239（765×**156**），原生 x 430–1142（713×**98**）。原版 `.input` 只有 `min-height:36px`（`InputBar.module.css:287`），156px 只能用"曾被拖高且持久化了 `--dsh-composer-resized-height`"解释（`ComposerResizeHandles.tsx:9,42`）。需在同一 resize 状态下各截一张再判断，避免误判成布局缺陷。
2. **焦点环**：原版输入区 `outline:none`、只有 chip 有 `:focus-visible` 2px 环（`PermissionSelect.module.css:25-27`、`ModelSelect.module.css:36-38`）；原生 `ui.Box` 文本域是否由框架画焦点环尚未在截图里验证（截图可能未聚焦）。需聚焦态截图确认是否多出一圈框。
3. **运行态整体观感**：原版截图是"运行中"（有彩色流动边光 beam、右下是停止块），原生截图是空闲态，无法对比 beam 的保真度（`beam_raster.go`/`beam_blur.go` 实现齐全，`ComposerBeam` 默认开：`internal/store/settings/chat_preferences.go:46`）。需要一张原生运行中截图。
4. **圆角实测偏差**：原生源码写 `Radius(22)`，但按上缘直边起点测（y=697 首个暗像素 x=447，左缘 x=430）弧长只有 ~17px，绘制与声明不符；需确认 `p.Stroke(...,22,0.5)` 的圆角是否被裁/被近似。
5. **语音录制态**：录制时原生追加 `■`/`×`/提示文字（`tool_actions.go:100-116`），原版无对应控件，需产品确认目标形态后另行对照。
6. **`next prompt` 的 Tab 键帽**：原版 `nextPromptKey` 有 `border:.5px` 键帽（rc.2 `client.js` `gVwOaG_nextPromptKey`）；原生 `paintNextPrompt` 是否同样带键帽未逐一核对。
7. **IME / 输入框内滚动 / 14 行上限**：原生上限 336px（`draft_client.go:15`）与原版 `--dsh-composer-text-max-height:336px`（`ConversationRoot.module.css:502`）一致，IME 有 `client.composing()` 守卫（`suggest_keyboard.go:96`、`draft_input.go:196`）——**已一致**，但如果要验收中文长文输入，建议录一次屏确认光标与撤销栈行为。
8. **`version-drift` 记录（已对齐，勿回退）**：rc.2 把 `ContextMeter` 从卡片下方的 `.dock` 移进了 trailing 行（alpha.1 在 `InputBar.tsx:619-624` 的 `.dock` 内；安装版 `C:\软件\Whale Isle\...\ui-conversation\lib\client.js:24360` 紧跟 `conversation.input.right` 之后渲染）。原生现在的位置与 rc.2 一致——修 #7 时只改样式，不要挪回 dock。另外 rc.2 新增了 `--dsw-radius-composer:22px`，但它只用于设置页的预览卡，**线上输入卡仍是 `--dsw-radius-panel:28px`**，别据此把 #17 判成"已对齐"。

### 附：已核对一致、无需返工

- placeholder 文案与颜色：`发消息或创建任务, / 调用指令, @ 文件或对话`，`#ADB2B8`(亮)/`#81858C`(暗)（原版 `locales.ts:20`、`InputBar.module.css:329-342`；原生 `i18n.go:1050`、`theme.go:72`，实测原生占位字最暗像素 (173,178,184)≈`#ADB2B8`）。
- 输入区最小高度 36px、行高 24px、左右内边距 14/8、顶部 4px（`InputBar.module.css:283-300` vs `draft_client.go:11-15`、`draft_input.go:29`）。
- 发送盘几何：34×34 圆、`button-info-fill` 底色、`translateY(-2px)` 上移 2px（`InputBar.module.css:492-512` vs `composer.go:431-433,480-483`）——截图实测原版 34×34 @(1197–1230, 838–871)。
- 空草稿时按钮 `disabled`（原版 0.4 / 原生 0.5 透明度的淡蓝盘）**两边都是预期态**，截图里"原生发送键很淡"不是缺陷。
- `+` 按钮：28px 圆、`selector` 填充、居中 16px 加号（`InputBar.module.css:434-446` vs `composer.go:438`）。
- 卡片宽度轴：`content+32` 两侧再留 16px（原版 `ConversationRoot.module.css:39-40` vs 原生 `frame.go:841,1088` + `composer.go:206` 的 16px 侧距）。
- 权限"完全权限"风险确认弹窗已有等价实现（`permission_command.go:27-59` vs `PermissionSelect.tsx:219-234`）。

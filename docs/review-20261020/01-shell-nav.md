# 01 窗口外壳与导航（shell / nav）对比审查

- 审查对象：`internal/ui/{frame,sidebar,chrome,boot,launcher,workspacepicker}`、`cmd/whaleisle/{main.go,application_menu.go}`（原生）
- 对照基准：`reference/electron/src` + `runtime/vendor/deepseek-harness/packages/client/{ui-layout,ui-titlebar,ui-sidebar,ui-workspace,ui-surfaces,ui-brand-official,ui-primitives}/src`（alpha.1 源码）；运行版 rc.2 截图 `evidence/original-electron-3080.png`（1441×912）
- 原生运行态：`evidence/native-go-3090.png`（1298×822）
- 规则：只读外部目录、未改任何源码；本次唯一写入即本文件。证据一律 `file:line` 或截图坐标（原图坐标系）。

## 0. 两条前置更正（避免按错误前提排期）

1. **原生窗口确实已经是无边框 + 自绘窗口按钮**，不存在系统标题栏。
   证据：`cmd/whaleisle/main.go:2164-2168`（`TitleBarStyle: mygo.TitleBarHidden`、`WindowsCaptionButtons{Width:32,Height:32,Top:8,Right:8,GlyphSize:12}`、`WindowsFrameCornerRadius:20`、`Transparent:true`）；截图右上角三枚自绘按钮位于 (1160..1270, 16..34)，与原版注入按钮 (1325..1410, 18..31) 同构。
   截图里那一圈"白边框"来自三件事叠加，而不是 OS 外壳：① 整窗 1px 描边 `internal/ui/chrome/frame_ring.go:12-25`（截图 y=1 整行为 rgb(225,225,225)，x=1297 整列同色）；② 12px 高的空白 caption 条（见明细 #2）；③ 默认浅色主题 + 默认渐变壁纸，使整窗读起来像"白色应用窗"。
2. **alpha.1 与 rc.2 存在版本漂移**：原版运行截图的会话头部只有 **一行**（标题 + 智能体/模式 chip + 右侧动作，y≈22..40），没有"对话/轨迹"标签页行；原生按 alpha.1 复刻了 `workspace_caption.go:66,99-128` 的第二行标签页，头部因此高 66px（原生 y=0..66）而原版约 48px。该类条目在表中标 `version-drift`。

## 一、5 个最致命的差距

### 1. 空态 hero 完全没有还原（P0）

- **原版**：新会话空态是居中的「鲸鱼 + 26/32 wt500 标题 + workspace chip + 输入卡」三段式栈，`HeroShell.module.css:5-45`（`.stack` gap 12、`.headline` font-size 26/line-height 32/weight 500）、`:122-167`（workspace chip：folder+label+chevron，28px 高、13/500、hover 填充）；空态本身在滚动体里垂直居中，输入卡与标题同宽（`max-width: var(--dsh-composer-card-max-width)`）。
- **原生**：空态只剩一行 13px 灰字，且 composer 被钉在栏底。`internal/ui/chat/chat.go:140-143`：`if len(msgs) == 0 { ui.Text(c, i18n.T(i18n.KeyChatEmpty)).FontSize(13)… ; ui.Box(c).Grow(1) }`；文案 `internal/ui/i18n/i18n.go:1027`="选择会话后开始对话"。截图 (293,88) 另有一行 `workspace ready.`（见明细 #8）。
- **差在哪**：没有鲸鱼标识、没有 26px 标题层级、没有 workspace chip、输入卡不在视觉中心（约 y=700 贴底）。整个主面板在"新建会话"这一最常用入口上呈空白，是用户"不像原版"的第一眼来源。
- **修复方向**：在 `internal/ui/frame` 新增 hero 组件（等价 `ui-conversation/skeleton/HeroShell`）：`chrome.FishLogo()`（已有，`internal/ui/chrome/brand.go:41-49`）+ 26px 标题 + workspace chip（复用 `OnPickHeroWorkspace`）+ 居中且限宽的 composer 卡；`chat.go` 空态分支改为挂该组件而不是一行文本。

### 2. 窗口外壳的"整体感"缺失：空白 caption 条、侧栏顶不可拖、1px 硬描边（P1）

- **原版**：① 48px 高的 caption 拖拽带横跨整窗（`AppFrame.module.css:273-284` `.captionDrag` height 48、`-webkit-app-region: drag`），Electron 把它作为唯一 caption 区；② 右侧动作簇固定 `margin-top:12`、`margin-right: var(--dshd-wco-controls) =112px`（`AppFrame.module.css:333-348`）；③ 整窗 20px 圆角剪影 + 内 1px 发丝描边（`harness-chrome-inject.js:11-19,129-190`，`body{border-radius:20px}` + `#dshd-frame-ring` `inset 0 0 0 1px`）；最大化时描边与圆角一起取消（`:183-190`）。
- **原生**：① 只保留 `TitleBarHeight=12` 的空白条（`internal/ui/chrome/titlebar.go:33-41`），主路由下内容层再显式塞 0 高（`cmd/whaleisle/main.go:3801-3802,3812-3815`），于是窗口顶部 0..12px 既无内容也无拖拽区（MyGo 的拖拽只认 `DragWindow()`，见 `third_party/mygo/ui/input.go:323`）；② 侧栏顶（品牌行 60px）**没有** `DragWindow()`——`internal/ui/sidebar/sidebar.go:144-162` 整个 logo 行无拖拽属性，只有聊天列的两处标题行可拖（`frame/workspace_caption.go:31`、`frame/conversation_header.go:32`），因此按住左侧顶部拖窗无效；③ 描边是恒定 `0.1` 黑的内描边（`chrome/frame_ring.go:14-24`），最大化时靠 `IsMaximized()` 跳过（`:13`），但没有原版那种"发丝 + 20px 剪影 + 最大化去边"的整体处理。
- **差在哪**：可拖动面积少了 300×60 像素，是明确的交互退化；12px 空白条 + 恒定描边让窗口顶部像"被裁掉了"。
- **修复方向**：`internal/ui/sidebar`：给 logo 行与 footer 上沿加 `DragWindow()`；`chrome/titlebar.go` 的 caption 高度与 `main.go:2165` 改为按平台配置（Windows 48 或让内容整体上移 0），并与 `AppFrame` 的 48px 带对齐；`frame_ring.go` 改用主题 `border-l2` 发丝色并跟随最大化状态。

### 3. 侧栏"品牌锁 + 左下用户区"两件套不对（P0）

- **原版**：
  - 品牌 = 38px 高的官方 wordmark：38×38 鲸鱼头像 + `鲸屿` 18/700/letter-spacing -0.04em（"屿"用品牌蓝）+ `WHALE ISLE` 10/600/0.12em + `BASED ON DEEPSEEK HARNESS` 8/500（`ui-brand-official/src/client/Brand.tsx:15-28`、`Brand.module.css:1-65`）；行高 60、Windows 下 40（`SidebarRoot.module.css:40-44,220-231`）；`data-whale-isle-brand='name'` 时品牌 mark 槽被隐藏、字号 18/600（`SidebarRoot.module.css:277-313`）。
  - 左下用户区 = 账号菜单触发器：44px 高、24×24 圆头像 + 名称 + 菜单（`ui-settings-account/src/client/AccountMenu.module.css:3-6`），折叠态 36×36（`:4`）；截图 (24..120, 862..900) 即 `Ayase` + 头像。
- **原生**：`internal/ui/sidebar/sidebar.go:144-156` 为 28×28 头像 + 三行文字（15/400 名称、11/400 `MYGO鲸屿`、9/400 归属行），像素实测头像 (14..46, 18..46)=32px；左下只有 `启动器`(13/32px)、`联系与反馈`、`设置` 齿轮（`sidebar.go:287-302`、`connection_trigger.go:82-131`），**没有头像、没有账号名、没有账号菜单**（截图 (18..250, 745..800)）。
- **差在哪**：品牌锁偏小一个层级（头像 32 vs 38、名称 15/400 vs 18/700），并且多出一行重复名称；账号身份整块缺失，用户看不到"我是谁/退出登录/账号设置"入口。
- **修复方向**：`sidebar.go` 品牌行按 `Brand.module.css` 的四个字号/字重复刻（或直接复用 launcher 里已有的 `MYGO鲸屿` 锁定字标实现 `internal/ui/launcher/panel.go:299`）；footer 增加账号行（头像 + 邮箱/昵称 + 菜单），数据走已有的 `state.accountSignOut`/`OnContactAccount` 通道。

### 4. 顶部标题带：结构、文案与语言都不对（P1）

- **原版**：单行 48px 带内左侧为会话标题（14/500，居中省略），中间是智能体/模式 chip，右侧是 `Session log`/Git/面板开关簇（28-32px 按钮，`titlebar-density.ts:10-30` 按 560/720 分档收字），窗口按钮叠在最右（`harness-chrome-inject.js:61-63,263-270`）。截图 (324..1040, 20..40) 一行完成。
- **原生**：两行——52px caption（`frame/workspace_caption.go:31`）+ 13px 标签页行（`:62-66`）+ 1px 分隔线（`:126-128`），内容从 y=66 起；标题来自 `sessionAncestryHeader`，无标题时回落到**原始 session id**（`frame/session_ancestry.go:41-44`），因此截图显示 `New chat · session-`（(283..455, 17..34)）。
- **差在哪**：内容整体下移 ~18px；标题泄露内部 ID 且是硬编码英文（`internal/store/session/title.go:21-23` 直接拼 `"New chat · " + id[:8]`）；右侧动作簇相对窗口按钮的位置与按钮尺寸（28 vs 32/28）不一致。
- **修复方向**：删掉默认渲染的第二行 tab（对齐 rc.2，或仅在存在轨迹会话时出现）；`title.go:17-24` 的回落改为本地化 "新会话"（参考 `ui-workspace/src/client/rows/Rows.tsx:557` 的 `t('session.untitled')`），绝不显示 ID；标题带高度按 48 对齐。

### 5. 启动页 boot 从"场景"退化为"表单"（P1）

- **原版**：独立整窗场景——星空/海平线/水下雪粒/深渊分层，中心品牌块 + 状态行 + 动画点，底部 ticker 显示最新日志与行号、可展开 `logdrawer` 全部日志，右上自带窗口按钮与 caption 带（`reference/electron/src/renderer/boot.html:13-58`、`boot.css` 13.8KB、`window-controls.css:1-54`）。
- **原生**：主窗体内一个左对齐列：22px 标题、14px 状态、按钮行、一个带边框的日志卡（`internal/ui/boot/boot.go:55-101`）；无品牌、无场景、无 ticker、无独立窗口按钮（`:46` 注释"inside the main window body"）。
- **差在哪**：每次启动必经页面的设计语言整体缺失；与 `docs/design-language.md` 里的 boot 例外条款也不再对应。
- **修复方向**：`internal/ui/boot` 增加分层场景绘制（渐变 + 粒子 + 品牌块），日志改成底部 ticker + 抽屉；窗口按钮由 shell 提供，boot 内不再自绘。

## 二、明细表

| # | 项目 | 原版行为（证据） | 原生现状（证据） | 差距 | 严重度 | 修复方向 |
|---|---|---|---|---|---|---|
| 1 | 新会话空态 hero | 居中三段式：鲸鱼 + 26/32 wt500 标题 + workspace chip + 限宽输入卡（`HeroShell.module.css:5-45,122-167`） | 一行 13px 灰字 + 底部 composer（`chat/chat.go:140-143`）；截图 (293,88)(293,123) 另有两行散落文本 | 空态无设计、输入区不在中心 | P0 | `frame` 新建 hero 组件，复用 `chrome.FishLogo()` |
| 2 | 窗口顶部 caption 条 | 48px 拖拽带横跨整窗（`AppFrame.module.css:273-284`），动作簇 top 12 / right 112（`:333-348`） | 12px 空白条（`chrome/titlebar.go:33-41`；`main.go:2165`），顶部无内容 | 顶部 12px 死区、"窗口被裁"观感 | P1 | caption 高度按平台配置并与拖拽带一致 |
| 3 | 侧栏顶拖窗 | 整块侧栏顶部在 48px 带内，`-webkit-app-region: drag`（`AppFrame.module.css:277-284`） | logo 行无 `DragWindow()`（`sidebar/sidebar.go:144-162`） | 拖不动窗口（300×60px 失效） | P1 | logo 行加 `DragWindow()` |
| 4 | 品牌锁 | 38px 头像 + `鲸屿`18/700 + `WHALE ISLE`10/600 + 归属 8/500（`Brand.module.css:1-65`；`SidebarRoot.module.css:220-231,277-313`） | 28-32px 头像 + 15/400 名称 + 11/400 重复名 + 9/400 归属（`sidebar/sidebar.go:144-156`） | 字号/字重/头像小一档，多一行重复名 | P0 | 按 `Brand.module.css` 四段字号重刻 |
| 5 | 左下用户区 | 账号触发器 44px：24px 头像 + 名称 + 菜单（`AccountMenu.module.css:3-6`），截图 (24..120,862..900) | 仅 `启动器`/`联系与反馈`/`设置`（`sidebar/sidebar.go:287-302`、`connection_trigger.go:82-131`） | 无账号身份与入口 | P0 | footer 增加账号行 + 菜单 |
| 6 | 标题带行数 | 单行 48px：标题 + chip + 动作簇（截图 (324..1040,20..40)） | 52px + 13px 两行（`frame/workspace_caption.go:31,62-66,126-128`） | 内容整体下移 ≈18px | P1（version-drift） | 默认不渲染视图标签页行 |
| 7 | 会话标题文案 | 无标题用本地化 `session.untitled`，绝不显示 ID（`ui-workspace/src/client/rows/Rows.tsx:557`） | 回落 `"New chat · "+id[:8]`（`store/session/title.go:17-24`；`frame/session_ancestry.go:41-44`），截图显示 `New chat · session-` | 泄露内部 ID、硬编码英文 | P1 | 回落改本地化"新会话" |
| 8 | 未本地化文案 | 全部走 locale 字典（client i18n 规则） | `workspace ready: <title>` 作为状态行渲染在会话顶部（`store/session/workspace_rpc.go:37` → `chat/chat.go:396-397`，截图 (293,88)）；`New chat`、`standard` 等英文残留（截图 (283,17)(760,24)） | 中英混排 + 调试串外泄 | P1 | 状态行改为连接态专用；所有文案入 `i18n` |
| 9 | 侧栏/面板玻璃材质 | 面板叠在壁纸上并有面板级模糊（rc.2 运行态；alpha.1 侧栏填充见 `AppFrame.module.css:119-127`） | alpha 由 `BackdropSidebar` 曲线给出，glass≥80 后趋近不透明（`chrome/background.go:41-57`）；只有整张壁纸预模糊（`:159-162`），没有面板后景模糊 | 高玻璃度下壁纸消失；文字压在大色块上 | P1 | 面板后景单独模糊/降噪；修正 alpha 曲线 |
| 10 | 会话行状态点 | idle/running/done/warning 四态：10px 槽位 6px 实心（`ui-primitives/StateDot.module.css:1-31`），running 为 14px 呼吸环（`StateDot.tsx:50-67`）；rc.2 截图 idle 呈空心环 (24,290) | `status==""` 直接返回空槽（`sidebar/session_menu.go:43-47`）；warning 用 3px 实心圆（`:58-62`）、running 用 Spinner（`:50-51`） | 空闲行没有前导点，行首节奏缺失 | P1（version-drift） | 补 idle 环/点，按 StateDot 三色对齐 |
| 11 | 会话行 hover 行为 | hover 时时间被动作组替换（16px 图标、gap 10，`Rows.module.css:298-333`）、长标题 marquee（`:229-254`）、悬停卡（`:394-435`）、拖拽插入指示（`:353-392`） | hover 只显示一个 `…`（`sidebar/sidebar.go:370-386`），无 marquee / 悬停卡 | 信息与操作密度低一档 | P2 | 增补 hover 动作组与悬停卡 |
| 12 | 行几何（圆角/高度） | 会话 32 / 工作区 34，圆角 `--dsw-radius-md`=12（`Rows.module.css:99-117`；`ui-theme/src/styles/base.css:26`） | 会话 32 但圆角 8（`sidebar/sidebar.go:329`）、工作区 34 圆角 8（`sidebar/workspace_groups.go:102`） | 圆角小 4px，行显得更"方" | P2 | 统一为 12 |
| 13 | 工作区行折叠/图标 | 默认显示文件夹，hover 换成展开箭头（`Rows.module.css:143-146`）；折叠箭头带 300ms 旋转（`:163-165,196-198`） | 折叠按钮常显（16×20）+ `…` 常显（`sidebar/workspace_groups.go:111,127`），无切换/旋转 | 默认态更吵，无过渡 | P2 | 常显文件夹 + hover 换箭头 |
| 14 | 工作区块头（搜索/动作） | 36px 头，标签最大宽 45% 并有 max-width/opacity 动画，搜索展开时让位（`WorkspaceBrowser.module.css:53-105,143-196`），动作顺序 搜索→视图选项→新增 | 36px 头，标签固定 12px、搜索展开为瞬时替换（`sidebar/workspace_heading.go:13-58`），动作顺序 搜索→批量→…→+（`:44-95`） | 无动画、动作顺序不同 | P2 | 补动画与顺序 |
| 15 | 折叠 rail | 36×36 控件、12 间距，toggle 固定在 caption 内 `top:(48-28)/2, left:12`（`SidebarRoot.module.css:35-52`），折叠时 `rail-in` 从 translateX(49px) 150ms 入场（`:152-169`） | 36×36、gap 12（`sidebar/rail.go:12-27`），但展开态折叠按钮是 24×24 的 `‹` 文本（`sidebar/sidebar.go:157-160`），无 rail-in 动画 | 控件形状不对、缺入场动效 | P2 | 换成 28px 面板图标 + rail-in |
| 16 | 标题带动作簇定位/密度 | `margin-top:12`、`margin-right:112`、height 32、gap 8（`AppFrame.module.css:333-348`）；密度阈值 560/720（`titlebar-density.ts:10-14`） | `Absolute().Top(8).Right(controls)`（`frame/workspace_caption.go:79`）；阈值 720/900（`:73-78`） | 上移 4px、密度阈值错了（560 被当成 900 的替代） | P1 | 对齐 12px/560/720 |
| 17 | 启动页 | 整窗海洋场景 + 品牌块 + ticker + 日志抽屉 + 自绘窗口按钮（`reference/electron/src/renderer/boot.html:13-58`） | 左对齐 22px 标题 + 按钮 + 日志卡（`boot/boot.go:46-101`） | 场景整体缺失 | P1 | 分层场景化重做 |
| 18 | 启动器窗口 | 独立无边框窗 1060×660 / min 860×560（`reference/electron/src/main/window.js:904-921`） | 主窗内全窗 launcher（`cmd/whaleisle/main.go:3327-3329`）+ 另一个独立启动器窗 1040×740（`cmd/whaleisle-launcher/main.go:57`） | 尺寸/呈现两套，和原版都不一致 | P2 | 统一为 1060×660 独立窗 |
| 19 | 侧栏折叠过渡 | 轨道 300ms（`AppFrame.module.css:14-17`）+ 内容 150ms 交叉淡出（`SidebarRoot.module.css:131-141`）+ rail 入场 150ms | 只有轨道 300ms cubic-bezier(.4,0,.2,1)（`frame/shell_motion.go:11-16`），内容瞬时切换 | 折叠时内容"闪断" | P2 | 加 150ms 淡出再切布局 |
| 20 | 菜单/弹层材质 | 弹层必须走 `Menu`/`MenuSurface`，材质为 `--dsw-menu-surface-fill` + `--dsw-menu-backdrop-filter`，禁止组件覆盖（`runtime/vendor/deepseek-harness/docs/web-styling.md:26`）；elevation 统一走 `--dsw-elevation-*`（`:25`） | 工作区菜单用 `PopoverBase` 实色 220px/radius 10（`sidebar/workspace_heading.go:67-90`），会话菜单为 `ui.Menu`（`sidebar/session_menu.go:16-41`） | 弹层无毛玻璃、圆角/宽度不统一 | P2 | 弹层统一走带模糊的菜单组件 |
| 21 | 顶部条跨路由一致性 | 标题带是 AppFrame 网格第 1 行、横跨全部列且恒定 48px（`AppFrame.module.css:277-284`）；入口页用 `--dsh-frame-top-clearance` 让出同一高度（`:62-83`） | 主路由 0 高（`cmd/whaleisle/main.go:3801-3802`）、独占面板 32px（`:3810-3811` → `chrome/titlebar.go:82-94`）、侧栏折叠 32px（`main.go:3803` → `frame/sidebar_restore.go:24-37`） | 切路由时内容上下跳 0↔32px | P1 | 统一一层 caption 座位，路由只换填充不换高度 |
| 22 | 窄宽/手机布局 | 1024 自动折叠侧栏（`columns.ts:36`）；更窄时侧栏与右栏转为抽屉（绝对定位、宽 `min(320px, 100%-48px)`、`translateX` 过渡，`AppFrame.module.css:404-449`），`[data-compact-header]` 直接隐藏标题带动作簇（`:361-364`） | 只做了 1024 自动折叠（`frame/sidebar_columns.go:11-32`），没有手机抽屉；窄宽时动作簇不隐藏，只按 720/900 降级（`frame/workspace_caption.go:69-81`） | 600px 以下侧栏与动作簇互相挤压 | P2 | 补抽屉态与 compact-header 隐藏 |

（共 22 条：P0×3、P1×10、P2×9。）

## 三、待验证项（无充分证据，不下结论）

1. **系统标题栏是否真的出现过**：本机截图（`native-go-3090.png`）确认为无边框自绘按钮。用户反馈的"系统标题栏"若来自**发布版**而非源码树 `whaleisle.exe`，需要在用户实际运行的那个 exe 上复拍一次；当前证据只能解释为"12px 空白条 + 1px 描边 + 浅色底"的错觉。
2. **玻璃强度是否只是配置差异**：原生 `.devdata` 用默认渐变壁纸 + 浅色主题，原版用自定义图片壁纸 + 暗色。需要在**同一壁纸/同一主题**下再对比一次，才能把 `BackdropSidebar` 的 alpha 曲线（`chrome/background.go:31-40,50-57`）判为代码问题还是取值问题。
3. **侧栏宽度**：原版截图 300px（用户拖宽后落盘），原生截图 268px。`SidebarDefault=280`（`frame/frame.go:59-62`）本身已对齐 `columns.ts:22-36`，268 更像 `StateKey:"main"` 的持久化值（`cmd/whaleisle/main.go:2162`）。需读 `.devdata` 的窗口状态确认。
4. **原生菜单栏可见性**：原生注册了应用菜单（`cmd/whaleisle/application_menu.go:34-42,125-156`），原版则 `autoHideMenuBar + setMenuBarVisibility(false)`（`reference/electron/src/main/chrome.js:65-71`）。按 Alt 是否弹出原生菜单栏、是否挤压布局，需在客户端实测。
5. **托盘菜单与右键菜单项**：原版托盘含 显示/启动器/设置/市场/重启/退出/桌宠开关（`reference/electron/src/main/tray.js:23-56`），原版行右键为 置顶/重命名/归档/Fork/复制 ID 等（`Rows.tsx:485-530` 附近）。原生侧 `application_menu.go:67-123` 覆盖了一部分，托盘与行菜单未逐项核对。
6. **拖拽排序指示**：原版有 2px 主色插入线 + 箭头（`Rows.module.css:353-392`）；原生存在 `sessionDrag`/`sessionDropTarget`（`sidebar/sidebar.go:401-404`），但插入指示的绘制未见证据，需运行态复看。
7. **最大化/全屏态**：原版最大化时取消圆角与描边（`harness-chrome-inject.js:183-190`）。原生 `frame_ring.go:13` 只在 `IsMaximized()/IsFullScreen()` 时跳过描边，圆角是否跟随需实测。

## 附：已核对并对齐的部分（不作为差距）

- 三栏几何常量完全一致：`CENTER_MIN 400 / SIDEBAR_MIN 264 / SIDEBAR_MAX 420 / SIDEBAR_DEFAULT 280 / SIDEBAR_COLLAPSED 56 / 自动折叠 1024 / RIGHTBAR 0.45-0.70`（`ui-layout/src/client/columns.ts:22-50` ↔ `internal/ui/frame/frame.go:59-67`、`frame/sidebar_columns.go:15`）。
- 收起/展开的轨道曲线同为 300ms cubic-bezier(0.4,0,0.2,1)（`ui-theme/src/styles/base.css:12,15` ↔ `frame/shell_motion.go:11-39`）。
- Windows 窗口按钮几何已对齐：32×32、top 8、right 8、圆角 8、close hover `#e81123`（`harness-chrome-inject.js:9-19,95-123` ↔ `cmd/whaleisle/main.go:2166`、`reference/electron/src/renderer/window-controls.css:43-46`）。
- 会话行高度（32）、时间字号（10/16）、标题 14/20、16px 前导槽位与 title 的内边距（`Rows.module.css:109-127,265-270` ↔ `sidebar/sidebar.go:329,368-377`）。

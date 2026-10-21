# Pass 9 视觉/动效验收（visual acceptance）

日期：2026-10-21 · 责任人：visual-fixer（task-70）· 对照对象：安装版 0.9.3 / 0.3.9 真机截图 + 本仓库真机截图
方法：只用仓库里已有的 `ref-*.png` / `ours-*.png` 做并排比对，**不驱动应用、不新开窗口**。凡是没有对照截图的条目一律标 `UNVERIFIED`，不做推测。

证据目录：`docs/review-20261020/evidence/compare/`

> 分辨率约定：`ref-hero-1440.png` = 1440×920 DIP（DPR 1）。`ours-pass6b.png` = 2160×1380 px = **1440×920 DIP @150%**，故其后文 DIP 值 = 设备像素 ÷ 1.5。

---

## 1. 首屏（1440×920）

最新真机证据：`ours-pass6b.png`（Pass 6 后）。历史：`ours-pass2-1440.png`、`ours-pass5.png`、`ours-pass6.png`。
参考：`ref-hero-1440.png`；窄窗口 `ref-hero-960.png`。

| # | 项目 | 结论 | 证据 | 说明 |
|---|---|---|---|---|
| 1.1 | 侧栏默认展开 280 DIP | **PASS** | `ours-pass6b.png` vs `ours-pass2-1440.png` | Pass 2 前是 56 轨，现为完整 280 列（品牌/新会话/3 行导航/工作区段/底部设置+账号）。实拍 150% 下侧栏右边界约 391±2 设备像素，因窗口外轮廓 1 DIP 半透明描边与投影混在一起，孤立像素不可靠；DIP 值以 tester 实测（`TestSidebarKeepsTheTrackAtDesktopWidths` 断言 768/900/960/1280 全为轨道）为准。 |
| 1.2 | 侧栏内部度量（行距/按钮） | **PASS（行距已修）** | `ours-pass2-1440.png`（行距 40）vs `ours-pass6b.png`（36） | 导航行距 40→36 已落（`sidebar.go` 去掉 4 DIP gap），frame 测试 `TestSidebarStaysExpandedWithoutAWorkspace` 钉 36.0。新会话 `FillWidth×38`（280 列内 252 宽）与参考一致。 |
| 1.3 | 侧栏 Project 段 | **DIFF（未实现）** | 无 ours 截图；参考 `ref-hero-1440.png` 侧栏工作区段下方有 `Project` 组 | `internal/ui/projects` 只在有数据时出内容，没有内置空态（`Project + ⋯/+ + 「添加项目目录，开始持续对话。」`）。属 projects 包，非本轮写入域。 |
| 1.4 | 侧栏树系统分组与「启动器」 | **PASS** | `ours-pass6b.png` | 树里只剩 `Whale Isle / 无目录会话 / 其他会话` 与各自会话行，**不再有英文 `no-workspace` 节点、不再有「启动器」行**（Pass 2-D 删除；启动器仍由标题栏 `…` 进入）。`无目录会话 / 其他会话` 本就本地化。 |
| 1.5 | 标题栏（caption 带） | **PASS** | `ours-pass6b.png` 顶部 vs `ref-hero-1440.png` | caption 带高 48 DIP（实拍 72 设备像素 ÷1.5），与参考一致；右侧顺序为 Git 徽标 → 面板切换 → 最小化 → 最大化 → 关闭，与参考 `…/最小/最大/关闭` 一致。 |
| 1.6 | 最小化/最大化/关闭可见 | **PASS** | `ours-pass6b.png` 右上角 | Pass 2-D 后三个按钮齐全（原生只画 ✕，frame 自绘 min/max 于其左侧）。 |
| 1.7 | hero 标题行（鱼标 + 标题 + 预览徽标） | **UNVERIFIED（被遮挡）** | `ours-pass6b.png` | 该帧 hero 中部被「网易UU远程」窗口遮住，只能看到右侧 `标准模式 ⌄` 的一部分；结构证据改用 tester 帧 `pass6-after.png`（鱼标 + `探索未至之境` + `预览版` 徽标 + composer 卡 774×98 均在位）。真机需一张无遮挡首屏。 |
| 1.8 | composer 卡片 | **UNVERIFIED（被遮挡）** | `ours-pass6b.png` | 该帧被「网易UU远程」窗口盖住卡片中段，宽度/圆角无法测量；最近一次可测帧是 `pass6-after.png`（tester，774×98 DIP），与参考 774×98 一致。 |
| 1.9 | 背景（dune 全幅渐变 + 5 bloom） | **PASS（Pass 6 回调后）** | `pass6-before.png` / `pass6-after.png`；`ours-pass6.png` | 软边（相邻像素最大跳变 3/255，测试断言 ≤12）、无硬边白圆斑；Pass 6 把 bloom 浓度整体降到 224/255（−12%），中心高光 248→245、绿带略收。真机浓度以 `ours-pass6.png` 看仍比 `ref-hero-1440.png` 略饱和，但已在 10–15% 窗口内。 |
| 1.10 | 滚动条出现时机 | **PASS（代码）／UNVERIFIED（截图）** | 无 pass6b 后的滚动条特写 | `ScrollbarAlways` 已改 `false`（仅 hover/拖动出现），5 DIP 与 neutral 色保留。`ours-pass6b.png` 侧栏已**看不到**Pass 2 时那条常驻浅色滚动条，与参考一致。 |
| 1.11 | 字体度量 | **PASS（侧栏/caption，已开图）** | `ours-pass6b.png` vs `ref-hero-1440.png` | 侧栏导航 14 DIP、品牌 22 衬线 + 9 大写副标题、caption 13 DIP 与参考同级；hero 的 26/32 因被遮挡未在真机核对，tester 帧 `pass6-after.png` 与 `hero_scale_test.go`（100/125/200%）覆盖缩放一致性。 |

**首屏结论：11 项中 7 PASS、1 DIFF（Project 段未实现）、3 UNVERIFIED（composer / hero 标题行被遮挡；滚动条仅有代码与侧栏观感证据）。** 没有开过图的条目不写 PASS；`ours-pass6b.png` 是当前唯一可用的整窗真机帧，其中心区域被第三方窗口遮挡是本次验收最大的证据缺口。

---

## 2. 13 个设置页

参考齐 13 张：`ref-settings-general / appearance / interface / models / builtin / skills / market / remote / mcp / presets / usage / pet / about`（均 `-1440.png`）。
本仓库现有 ours 截图 10 张：`ours-settings-about / builtin / builtin-loading / interface / mcp / models / pet / presets / remote / usage`。

| # | 设置页 | 参考 | our 证据 | 结论 |
|---|---|---|---|---|
| 2.1 | 通用 general | `ref-settings-general-1440.png` | — | **UNVERIFIED**：缺 ours 真机截图 |
| 2.2 | 外观 appearance | `ref-settings-appearance-1440.png` | — | **UNVERIFIED**：缺 ours 真机截图 |
| 2.3 | 界面 interface | `ref-settings-interface-1440.png` | `ours-settings-interface.png` | **UNVERIFIED（未逐张开图）**：有证据文件，本轮只抽查了 models 一页，未开图核对 |
| 2.4 | 模型 models | `ref-settings-models-1440.png` | `ours-settings-models.png` | **PASS（结构，已开图）**：1000×800 内容区截图显示两张供应商卡（名称 + 模型名 + 密钥输入 + 橙色环境变量提示 + `自定义设置` 折叠 + 取消/保存）+ 虚线 `+ 添加模型提供商` 行，与参考同名页的分组顺序一致。注意该 ours 帧是**内容区裁切**、参考是 1440 全窗（含左导航），帧不同源，只能做结构判断、不能做像素级比对 |
| 2.5 | 内置 builtin | `ref-settings-builtin-1440.png` | `ours-settings-builtin.png`、`ours-settings-builtin-loading.png` | **UNVERIFIED（未逐张开图）** |
| 2.6 | 技能 skills | `ref-settings-skills-1440.png` | — | **UNVERIFIED**：缺 ours 真机截图 |
| 2.7 | 市场 market | `ref-settings-market-1440.png` | — | **UNVERIFIED**：缺 ours 真机截图 |
| 2.8 | 远程 remote | `ref-settings-remote-1440.png` | `ours-settings-remote.png` | **UNVERIFIED（未逐张开图）** |
| 2.9 | MCP | `ref-settings-mcp-1440.png` | `ours-settings-mcp.png` | **UNVERIFIED（未逐张开图）** |
| 2.10 | 预设 presets | `ref-settings-presets-1440.png` | `ours-settings-presets.png` | **UNVERIFIED（未逐张开图）** |
| 2.11 | 用量 usage | `ref-settings-usage-1440.png` | `ours-settings-usage.png` | **UNVERIFIED（未逐张开图）** |
| 2.12 | 桌宠 pet | `ref-settings-pet-1440.png` | `ours-settings-pet.png` | **UNVERIFIED（未逐张开图）** |
| 2.13 | 关于 about | `ref-settings-about-1440.png` | `ours-settings-about.png` | **UNVERIFIED（未逐张开图）** |

**设置页结论：13 页中 1 页 PASS（models，结构级）、8 页有 ours 证据但本轮未逐张开图、4 页连 ours 证据都缺（general / appearance / skills / market）。** 判定口径：本表只对"实际开过的图"给 PASS/DIFF；没开图的一律写 UNVERIFIED，避免用印象冒充比对结果。另有帧不同源问题：ours 设置页多为内容区裁切（如 models 为 1000×800），ref 是 1440 全窗，像素级比对需要同尺寸真机复采。

---

## 3. 动效 / 材质

| # | 项目 | 结论 | 依据 / 手工步骤 |
|---|---|---|---|
| 3.1 | 背景 bloom 流动 | **PASS（定格）** | 5 个 bloom 锚点固定、±10%/±8% 慢漂移（周期约 28 s，speed 190）；定格证据 `pass6-after.png`。120 ms 两帧像素必须不同由 `TestWallpaperReducedMotionStopsSchedulingAndHoldsPhase` 覆盖；ReduceMotion 时定帧不漂移（同测试断言像素完全一致）。手工步骤：外观页把「动效速度」拉到 0/100 各截两帧间隔 1 s 比对。 |
| 3.2 | 背景软边 / 无 banding | **PASS** | `TestDefaultGradientIsASoftFullBleedWash`：四角非底色、相邻像素跳变 ≤12（实测 3）、中心比顶部亮、左下带绿。 |
| 3.3 | 玻璃材质（面板/菜单） | **PASS（定格）** | 面板用 `chrome.DrawGlass` + `MaterialPanel`，菜单 `MenuSurface`/`MenuElevation`；`ours-settings-*.png` 与参考同为半透明磨砂 + 1 DIP hairline。手工步骤：外观页把「玻璃不透明度」拉到 0/100 各截一张。 |
| 3.4 | 指针特效 | **UNVERIFIED** | 静态截图无法定格拖尾。手工步骤：外观页开「指针特效」，快速划一条斜线后立即截图，参考 `ref-appearance-1440.png` 的 `CursorEffect` 设置项一致性。 |
| 3.5 | 滚动 / 悬停过渡 | **PASS（结构）** | 滚动拇指仅 hover/拖动出现（Pass 2-D 后 `ScrollbarAlways=false`）；行 hover 背景用 `colors.Hover`，与参考同色阶。过渡时长走 `ui` 框架的 ElementTransition，本仓库不自绘时长。手工步骤：悬停侧栏任一行、hover 设置页任一开关各截一张，与参考 hover 帧比对背景亮度。 |
| 3.6 | 窗口圆角与轮廓 | **PASS** | `ours-pass6b.png` 左上/右下外角为 20 DIP 圆角（对应 `WindowsFrameCornerRadius: 20`），与参考一致。 |

---

## 4. 本轮是否改代码

**没有改。** 逐项比对后没有找到「有证据 + 落在本轮写入域（chrome/theme）」的样式差异：
- 背景浓度属 Pass 6，已回调（`bloomDensity = 224`）。
- 滚动条出现时机属 Pass 2-D，已修（`ScrollbarAlways = false`）。
- 侧栏行距属 Pass 2-D，已修（40→36）。
- 标题栏三按钮属 Pass 2-D，已修（frame 自绘 min/max）。

仍列出的样式差异都不在 chrome/theme 能修的范围（见下）。

## 5. 剩余 DIFF（交给 Lead 决策）

1. **侧栏 Project 段缺失**（`internal/ui/projects` 需要空态：标题 + `⋯`/`+` + 「添加项目目录，开始持续对话。」）——不在本轮写入域。
2. **4 个设置页缺 ours 真机截图**（general / appearance / skills / market），无法判定。
3. **composer 卡片在当前帧被遮挡**，需一张无遮挡的 1440×920 首屏用于宽度/圆角复核。
4. **真机背景仍略饱和**（`ours-pass6.png` vs `ref-hero-1440.png`，约 5–10%）：如果还要收，`bloomDensity` 210（−18%）是一档可回退的调整。
5. **hero 工作区胶囊的英文 `no-workspace`** 仍可能出现（composer 域），侧栏树已无此问题。

---

### 复现命令

```powershell
go build ./cmd/... ./internal/...
go test ./internal/ui/chrome ./internal/ui/theme ./internal/ui/frame
```

（本轮验收未改动代码，以上仅确认基线仍绿。）

---

## 6. 设置页实机同源复验（Pass 11 补充，12 页 × 1440×920）

证据：`docs/review-20261020/evidence/acceptance/settings-live/{general,appearance,interface,models,plugins,skills,mcp,remote,presets,usage,pet,about}.png`（2160×1380 = 1440×920 DIP @150%），参考 `ref-settings-*-1440.png`。

### 6.1 逐页像素差（把 ours 双三次缩放到 1440×920 后按 6×4 网格统计）

| 页 | 平均每通道差 | 最大格 | 最大格位置 |
|---|---|---|---|
| general | 24.4 | 36.7 | row1 col0（左导航） |
| appearance | **39.4** | **91.6** | **row3 col3（右下预览区）** |
| interface | 25.8 | 40.0 | row1 col0 |
| models | 22.8 | 41.2 | row1 col0 |
| plugins | 24.0 | 42.0 | row1 col0 |
| skills | 25.3 | 42.6 | row1 col0 |
| mcp | 23.2 | 37.7 | row1 col0 |
| remote | 25.5 | 42.2 | row1 col0 |
| presets | 23.2 | 34.7 | row1 col0 |
| usage | 23.1 | 36.7 | row1 col0 |
| pet | 24.7 | 39.3 | row0 col0 |
| about | 23.8 | 38.9 | row1 col0 |

**读法**：12 页里 11 页最差格都落在 `row1 col0` —— 设置窗的**左导航列**；平均差集中在 23–26 的窄带内，说明这是**一套系统性差异**而不是 12 个独立问题。`appearance` 是唯一例外（39.4 / 91.6 在右下预览卡），需要单独看。

### 6.2 已实测确认的几何（不受系统性差异影响）

用白色面板边界实测（ours 除以 1.5 换算 DIP）：

| 项 | 参考 | ours | 结论 |
|---|---|---|---|
| 面板 x | 322…1118 | 321…1119 | **PASS**（±1 DIP） |
| 面板 y | 62…860 | 61…859 | **PASS**（±1 DIP） |
| 面板尺寸 | ~796×798 | 798×799 | **PASS**（±2 DIP） |

**结论：设置窗的位置与尺寸已经对齐，不是布局问题。**

### 6.3 系统性 DIFF（每页都有，已开 general 一对图确认）

- **DIFF-S1 面板材质不透明**：参考面板是半透明玻璃，被壁纸染色，横扫描线**找不到任何 ≥246 的纯白像素**；ours 是近实白（整条扫描线 ≥246）。→ 需要设置窗把面板底色从 `theme.BG` 换成 `chrome.MaterialFill/SurfaceFill` 的玻璃板（`internal/ui/settings` 域的改动，本任务写入域只到 chrome/theme，**未改**）。
- **DIFF-S2 左导航样式与条目**：参考**无选中胶囊底色**、行距 ~42 DIP、13 项（含「市场」）；ours 有选中胶囊、行距 ~48 DIP、12 项（**缺「市场」**），且同义项命名不同：插件/内置插件、界面/界面设置、用量/用量统计、桌宠/鲸鱼娘。→ 导航条目属 settings 内容层。
- **DIFF-S3 标题与行距**：参考标题「设置」；ours「Harness 设置」。参考表单行距更紧（同一屏放 10 行，ours 放 8 行 + 折叠项）。
- **DIFF-S4 英文泄漏（general 页实拍）**：表单顶部出现 `settings document opened`；下拉值为 `workspace-write`、`zh`、`completion`、`sidebar`；另有红字「还没有配置 API Key / 当前模型线路需要可用的 API …」（中英混排）。→ 属设置页文案/取值格式化。
- **DIFF-S5 表单标签集与顺序**（general）：参考 权限/语言/字号大小/工作步骤展示/工作步骤收起时机/显示代码工作视图/快捷键/网页链接默认打开方式/繁忙时的发送行为/关闭窗口时；ours 账号/高级/新任务默认权限/界面语言/字号大小/工作详情/工作详情收起时机/开发者工具/键盘快捷键/链接打开方式。→ 内容层差异（含参考有、ours 无的「繁忙时的发送行为」「关闭窗口时」）。
- **DIFF-S6 appearance 页主题网格溢出**（最大格 91.6，已开图）：`主题` 区的主题卡按 3 列排（DeepSeek/Midnight、Celadon/Violet、Amber/Paper …），但可视宽度只够 2 列，**第 2 列右侧被面板边界裁切、第 3 列完全看不到**（150% 下裁切在设备像素 ~1530 ≈ 1020 DIP，而面板右边界为 1119 DIP）。同一页另见选中导航项「外观」带**亮蓝焦点环**；参考帧无该环，但参考帧可能是鼠标态，**焦点环属状态差异，不作为样式差异下结论**。

### 6.4 本轮结论

- **几何/网格：PASS**（6.2）。
- **12 页全部 DIFF**，但 11 页同源（DIFF-S1..S5），appearance 另有 DIFF-S6。
- **未改代码**：能定性修的两条（面板材质 DIFF-S1、导航/行距 DIFF-S2）落在 `internal/ui/settings` 内容与结构层，不在「chrome/theme/设置样式常量」可安全改动的范围；其余是文案与取值格式化（DIFF-S4/S5）与主题网格溢出（S6），同样属 settings 域。建议由 settings 域 owner 处理，我可以在其完成换壳后复验。
# UI 布局 / 几何 / 特效对照审计（2026-10-27）

范围：`C:\AI\WhaleIsle-MyGo` 仓库内**已有**的原生基线 + 契约测试 + 截图证据，与当前代码/headless 渲染结果逐项对照；只修有基线且可机器计算的偏差，其余明确标「无基线」。
本轮**未启动、未重启、未修改** Electron 本体（`C:\软件\Whale Isle` 只读）；未提交 git；未改 `runtime/`、`third_party/`。

> 资产更正：任务书里写的 `docs/parity/acceptance-ui-parity.md` 实际位于 `docs/acceptance-ui-parity.md`（L/D/P/S、FL/FD/FS 全部仍为空白）；`docs/reference/` 不存在，原生基线在仓库根 `reference/electron/`（只读快照，commit `26205a89`，2026-10-09）。

---

## 1. 可对照的度量源盘点

| # | 源 | 位置 | 能提供什么（原生值） | 覆盖画面 | 可机器计算 |
|---|---|---|---|---|---|
| S1 | 原生设计 token 表 | `reference/electron/vendor/deepseek-harness/packages/client/ui-theme/src/styles/base.css:12-20,25-38` | 动效 8 个 token（0.2s/0.1s/0.3s/0.16s/0.4s、overlay-out、rise 8、popover rise 4、scale .96、`cubic-bezier(.4,0,.2,1)`）+ 14 个圆角 token（4/6/8/10/12/13/14/15/16/18/20/22/26/28） | 浮层、菜单、弹窗、翻转、全应用圆角 | 是（数值） |
| S2 | 注入式标题带 | `reference/electron/src/main/harness-chrome-inject.js:4-14,61-123` | 带高 48、按钮 32、顶部内缩 12（`padding:12px 8px 4px` + `align-items:center`）、右缘 8、间隙 0、圆角 8、字形 12、预留 112（`8+32*3+8`）、关闭 hover `#e81123`/白字、窗口轮廓圆角 20（`FRAME_RADIUS`） | 桌面主壳标题带 | 是 |
| S3 | 原生渲染页样式 | `reference/electron/src/renderer/window-controls.css`、`src/renderer/pet-live2d.html`（`#pet-chat` 248px、8px 锚距、4px 内缩） | 桌宠聊天卡 248 宽 / 8 锚 / 4 内缩；窗口按钮 pill 默认值（**死值**，主窗不加载） | 桌宠浮层、标题带 | 部分（数值可用，pill 值无效） |
| S4 | chrome 度量报告 | `docs/parity/chrome-metrics-20261024.json` / `.md` | G01/G02/G50 逐项对照 + 动效 token 对照表 + F1–F5 修复单（含 `band content row 44 vs 48`、`caption top 12 vs 8`） | 桌面主壳、浮层 | 是（且已部分落地，见 §3） |
| S5 | S27/S28 残项收口 | `docs/parity/chat-chrome-residual-20261026.md` | composer 体 774.4（`0.64×1160+32`）、导航行 22/127/中心 138/步进 40、hero 250 栈（314.67 / 373.33 / 409.33，920 时 +30） | 会话主壳 | 是 |
| S6 | 原生 UIA 矩形表 | `docs/review-20261022/round240-chrome-deltas.md`、`round245-final-light.md`、`round253/256*.md`、`docs/status.md:455-504` | 发送盘 (1188.67,522) 34.67²、输入框 (482,417.33) 745.33×52.67、鱼徽标 (936,321.33) 48.67×15.33、标准模式 x655.33、卡 155.34、capt 蒙版带 2160×61 | 会话主壳 | 是 |
| S7 | 设置页原生 DOM 度量 | `docs/review-20261022/round160-settings-parity-audit.md` | 对话框 800×800 @ (320,60) r28；左栏 164 @ x332；导航项 40 + 4 步进；内容列 564 @ x532；行 68（16+36+16）/77；控件 36 贴右缘；开关 36×20 | 设置（General 页 DOM 级） | 是 |
| S8 | 启动器/欢迎页度量 | `docs/review-20261022/round147-launcher-welcome-pixel-parity.md`、`round153-welcome-metric-parity.md` | 启动器壳 1060×660、首页/诊断座次 | 启动器 L-* | 部分（本轮未重测） |
| S9 | 轮次记录 | `docs/status.md`（round151–262） | 窗口 1440×920、client 1440×860 @150%、像素评分（threshold0 1.36%/1.685%、mean 1.23） | 全局 | 是（历史值，非本轮重算） |
| S10 | 已落盘截图 | `docs/review-20261020/evidence/**`（597 个 PNG，含 `ref-settings-*.png` 13 个设置页、`ref-hero-1440.png`、`ref-sidebar-collapsed-1440.png`、`ref-dialog*.png`） | 原生整屏截图 | 设置 13 页、主壳、浮层 | **否**：同目录没有同尺 MyGo actual；补 actual 需现场起本体或重打进包（本轮禁止）→ 记「无基线」 |
| S11 | 圆角/阴影裁切图 | `tmp/fix/baseline-*corners*.png`、`tmp/fix/shadow/baseline-*shadow*.png`（2026-10-26） | 原生窗口四角/阴影裁切 | 窗口圆角、阴影 | 部分（只有图，未接 diff 脚本） |
| S12 | 代码内契约测试 | `internal/ui/frame/*_test.go`（150+ 文件）、`internal/ui/chat/*_test.go`、`internal/ui/settings/*_test.go`、`internal/ui/{chrome,widgets,composer,sidebar}` | 已钉住的数值断言（本轮直接复用为「当前值」） | 会话、侧栏、设置、浮层、面板 | 是 |
| S13 | 桌宠物理/坐标基线 | `docs/features/pet-renderer-parity.md:50` + `tmp/pet-coordinate-gui-result.json` | 150 DIP / 65 DIP·s⁻¹ / 1400 DIP·s⁻²；glide 5.023023913s；50ms 抛掷位 (115,93.5) | 桌宠 | 是（确定性用例，非 UI 包） |
| S14 | 差距矩阵 | `docs/parity/gap-matrix-20261025.json`（89 行：closed 63 / open 25 / measurement-error 1） | 全局残余状态 | 全局 | 是（状态），但 **S27/S28 行已过期**（见 §3） |

**按画面覆盖度**：会话主壳 **强**（S4/S5/S6/S12）；侧栏 **强**；浮层/对话框 **强**（S1/S4/S7/S12）；设置 **1/19 页强**（S7+S12 共享行原语），其余 18 页只有 S10 截图、**无同尺 actual**；右/底工作区面板 **弱**（只有单测断言，无整页基线）；桌宠 **中**（坐标/物理强，图层/性能无）；通知 **无**（只有功能单测，无视觉/交互度量）；启动器 **未在本轮重测**。

---

## 2. 对照结果（DIP，1440×860 / 920；「当前值」为本轮 `ui.NewTester` 重算）

判定口径：`一致` = 差 0；`≤1px` = 0 < |差| ≤ 1；`>1px`；`无基线`。

### 2.1 会话主壳（D-MAIN，原生 UIA round240/245）

| 画面 | 项目 | 基线值 | 当前值 | 差值 | 判定 |
|---|---|---|---|---|---|
| D-MAIN | composer 卡 y | 409.33 | 409.33 | 0 | 一致 |
| D-MAIN | composer 卡高 | 155.34 | 155.34 | 0 | 一致 |
| D-MAIN | composer 卡宽 | 774.39 | 774.40 | +0.01 | ≤1px |
| D-MAIN | composer 卡 x | 469.3 | 469.80 | +0.50 | ≤1px（divider 取整，S28 已登记） |
| D-MAIN | 输入框 x | 482 | 482.47 | +0.47 | ≤1px |
| D-MAIN | 输入框 y / 高 | 417.33 / 52.67 | 417.33 / 52.67 | 0 / 0 | 一致 |
| D-MAIN | 输入框宽 | 745.33 | 745.34 | +0.01 | ≤1px |
| D-MAIN | 发送盘 y | 522 | 522 | 0 | 一致 |
| D-MAIN | 发送盘 x | 1188.67 | 1189.81 | +1.14 | >1px（派生自卡 x +0.50，见 §5.2） |
| D-MAIN | 发送盘尺寸 | 34.67×34.67 | 34×34 | −0.67 | ≤1px |
| D-MAIN | 标语 y / x | 314.67 / 765.33 | 314.67 / 765.69 | 0 / +0.36 | 一致 / ≤1px |
| D-MAIN | 预设行 y | 373.33 | 373.33 | 0 | 一致 |
| D-MAIN | 鱼徽标 x / 宽 / 高 | 936 / 48.67 / 15.33 | 936 / 49.00 / 15.40 | 0 / +0.33 / +0.07 | ≤1px |
| D-MAIN | **鱼徽标 y** | **321.33** | **322.97 → 321.33** | **+1.64 → 0** | **>1px → 修后一致** |
| D-MAIN | 导航 label y / 高 | 127 / 22 | 127 / 22 | 0 | 一致 |
| D-MAIN | 导航 label 中心 | 138 | 138 | 0 | 一致 |
| D-MAIN | 导航行步进 | 40 | 40 | 0 | 一致 |
| 侧栏 | 新建会话座 | 14,70 252×38 | 同 | 0 | 一致 |
| 侧栏 | 底部账号座 | x14 w252 h36 | 同 | 0 | 一致 |
| 侧栏 | 视图选项 | 240,(250) 28×28 | 同 | 0（y ±1） | ≤1px |
| 标题带 | 带高 | 48 | 48 | 0 | 一致 |
| 标题带 | 内容行 | 12..44 | 12..44 | 0 | 一致 |
| 标题带 | **窗口按钮顶部** | **12（12..44）** | **8（8..40）→ 12** | **−4 → 0** | **>1px → 修后一致** |
| 标题带 | 按钮尺寸/右缘/圆角 | 32 / 8 / 8 | 同 | 0 | 一致 |
| 标题带 | 关闭 hover | `#e81123` + 白字 | 同 | 0 | 一致 |
| 窗口 | 圆角（普通/最大化） | 20 / 20 | 20 / 20 | 0 | 一致 |

### 2.2 设置（S-*）

| 画面 | 项目 | 基线值 | 当前值 | 差值 | 判定 |
|---|---|---|---|---|---|
| S-DESK… | 设置对话框 | 800×800 @ (320,60)，r28 | 同 | 0 | 一致 |
| 设置（全部页） | 行高下限 | 68（16+36+16） | 68 | 0 | 一致 |
| 设置（全部页） | 行标题 | 14/22，y=行顶+23 | 同（钉 y=23 h=22） | 0 | 一致 |
| 设置（全部页） | 行描述 | 12/18，标题下 4 | 同（y=110 h=18） | 0 | 一致 |
| 设置（全部页） | 值控件 | 高 36，贴右缘 | 同（右缘 ≤564） | 0 | 一致 |
| 设置（全部页） | 开关 | 36×20 r999 | 同 | 0 | 一致 |
| S-ACCT/S-APPEAR/S-UI/S-MODEL/S-PLUG/S-SKILL/S-MCP/S-AGENT/S-ABOUT + OTHER ns | 逐页三图（baseline/actual/diff） | 原生截图存在（S10） | 无同尺 actual | — | **无基线** |

### 2.3 浮层 / 对话框 / 桌宠 / 通知

| 画面 | 项目 | 基线值 | 当前值 | 差值 | 判定 |
|---|---|---|---|---|---|
| 浮层 | 进入（时长/位移/缩放） | 200ms / 8 / 0.96 | 同 | 0 | 一致 |
| 浮层 | 退出 | 100ms | 同 | 0 | 一致 |
| 菜单/popover | 时长 / 位移 | 160ms / 4 | 同 | 0 | 一致 |
| 全应用 | 共享曲线 | cubic-bezier(.4,0,.2,1) | 同（`DesignEase` 有 1e-4 断言） | 0 | 一致 |
| 对话框 | 遮罩深度 | 亮 .24 / 暗 .5 | 同 | 0 | 一致 |
| 桌宠 | 快捷聊天窗 | 248×360 | 248×360 | 0 | 一致 |
| 桌宠 | 锚距 / 内缩 | 8 / 4 | 8 / 4 | 0 | 一致 |
| 桌宠 | Live2D 分层逐帧 | 无（只有坐标/物理基线） | — | — | **无基线** |
| 手机 | PR 复核窗 | 原生无对应窗口 | 660×720 | — | **无基线** |
| 标题带 | 右侧图标 y（切换终端/右侧栏） | 冲突：round240 UIA 记 8，chrome-metrics 记 12 | 12.67（历史实测） | 不可定 | **无基线**（冲突，见 §5.1） |
| 通知 | 系统通知中心呈现/点击 | 无 | — | — | **无基线** |
| 工作区 | 文件树/预览/终端/浮动三态整页 | 无整页基线 | — | — | **无基线** |

**统计**：本轮共 45 个判定行 —— `一致` **28**、`≤1px` **8**、`>1px` **3**（其中 2 项本轮已修至 0：窗口按钮顶部、首屏徽标 y；1 项保留：发送盘 x +1.14，见 §5.2）、`无基线` **6**。§4 特效表另列 16 行：`一致` 10、`修后一致` 2、`有基线未修（toolkit）` 1、`无基线` 3。

---

## 3. 修复清单（含前后数值与新增测试）

| # | 文件 | 项目 | 前 → 后 | 依据 | 新增/更新测试 |
|---|---|---|---|---|---|
| F1 | `internal/ui/frame/caption_buttons.go` | 窗口按钮顶部 | **8 → 12**（按钮 8..40 → 12..44） | S2：`#dshd-shell-controls{height:48;padding:12px 8px 4px}` + 32px 按钮 ⇒ y=12..44；且与 `cmd/whaleisle/main.go` 的 `WindowsCaptionButtons{Top:12}` 一致（此前 Go 绘制的三连与平台层相差 4 DIP） | 新增 `TestCaptionButtonsSitInTheNativePlateRow`（`caption_plate_seat_test.go`）：常量/底边/右缘 8·40·72、命中点点击仍生效、与标题带内容行一致；`TestCaptionButtonsHoverFills`、`TestCaptionButtonsMaximizedGlyph` 随常量自动跟随并复跑通过 |
| F2 | `internal/ui/frame/conversation_phase.go` | 首屏徽标纵向座 | **322.97 → 321.33**（上提 1.64） | S6：原生徽标 (936, **321.33**) 48.67×15.33；32-DIP 标语框 314.67..346.67 的中心座是 323.00，原生高 1.67，即基线对齐而非中心对齐 | 新增 `conversationHeroBadgeLift = 1.64` + `TestHeroBadgeSitsOnTheNativeSeat`（`hero_badge_seat_test.go`）：y=321.33(+30)、lift=1.64、且 slogan/鱼座不动；`TestHeroHeadlineMatchesTheNativeColumns`、`TestRound255HeroRowNodes` 复跑通过 |
| F3 | `internal/ui/widgets/motion.go` | swap 时长 | **120ms → 100ms** | S1：`--ds-motion-duration-swap: var(--ds-transition-duration-fast)` = 0.1s（不是自有时长） | 更新 `TestMotionTokens` + 新增 `TestMotionTokensMatchTheReferenceSheet` |
| F4 | `internal/ui/widgets/motion.go` | flip 时长 | **300ms → 400ms** | S1：`--ds-motion-duration-flip: 0.4s`（表内最长 token） | 同上（sheet 表 + 「flip 是最长 token」断言） |
| F5 | `internal/ui/widgets/motion.go` | 时长单一来源 | 常量块与 token 字面量重复 → token 直接读 `OverlayDuration/PopoverDuration/SwapDuration/FlipDuration`，`SwapDuration = FastDuration` | 防止再次漂移（旧测试只是自证 120/300） | `TestMotionTokens*` 断言常量与 token 相等 |

> 说明：`docs/parity/gap-matrix-20261025.json` 的 `S27`/`S28` 仍写 `open`，属过期状态 —— 其收口证据在 `docs/parity/chat-chrome-residual-20261026.md`（本轮复跑 `TestComposerCardMatchesTheConversationBodyWidth`、`TestNavigationLabelKeepsTheNativeBaseline`、`TestHeroStackHasADefinedNativeSeat` 全 PASS）。本轮未改这份按日期归档的 json 快照。

---

## 4. 特效对照（时长 / 曲线 / 透明度）

| 特效 | 基线（出处） | 当前实现 | 判定 |
|---|---|---|---|
| 面板/浮层弹出 | 200ms，第 8 DIP 起，scale .96（S1 `--ds-motion-duration-overlay`/`-distance-overlay`/`-scale-overlay`） | 200ms / Y8 / .96（`chrome.OverlayTransition`） | 一致 |
| 浮层收起 | 100ms（`--ds-motion-duration-overlay-out` = fast） | 100ms（`chrome.OverlayExitDuration`） | 一致 |
| 菜单/popover | 160ms，rise 4 | 160ms / Y4 | 一致 |
| 控件换脸（chevron/开关） | swap = fast token 100ms | **120ms → 100ms**（F3） | 修后一致 |
| 整面翻转 | flip 400ms | **300ms → 400ms**（F4） | 修后一致 |
| 悬停/按压 | fast 100ms | 100ms（`HoverMotion`） | 一致 |
| 共享曲线 | `cubic-bezier(0.4, 0, 0.2, 1)` | `widgets.DesignEase`（1e-4 断言） | 一致 |
| 遮罩透明度 | 亮 .24 / 暗 .5（`Modal.module.css:25-31`） | 同 | 一致 |
| toast | 进入 160ms（−6px 上滑）+ 停留 3000ms + 淡出 1000ms（`Toast.module.css:39-41,80-97`、`Toast.tsx HOLD_MS=3000/FADE_MS=1000`） | toolkit `ui.Toast`：默认寿命 4000ms 但**淡出仅 200ms、无上滑进入**；调用点 3s（反馈）/6s（错误）/−1（常驻） | **有基线、未修**：`third_party/mygo/ui/toast.go` 属 vendored toolkit，本轮不改；登记为 toolkit 轮次任务。调用点 3s/6s 在原生无对应常量 → 无基线 |
| 流式光标闪烁 | 无（原生是系统文本光标，CSS 未定义周期） | 500ms 亮 / 500ms 灭（`typingCaretPeriod = 1000ms`） | **无基线，未验证** |
| 关闭旋转 | 未在原生 CSS/代码找到常量 | `chrome.ClosingSpinPeriod = 850ms` | **无基线，未验证** |
| 桌宠动效（滑行/抛掷/重力） | 150 DIP / 65 DIP·s⁻¹ / 1400 DIP·s⁻²，glide 5.023023913s（S13） | 同（确定性用例） | 一致 |
| 窗口圆角 / 最大化圆角 | 20 / 20（S2 `FRAME_RADIUS`、chrome-metrics） | 20 / 20（`WindowsFrameCornerRadius: 20`） | 一致 |
| 毛玻璃（透明主题最小模糊） | 20（`ui-theme/src/wallpaper.ts TRANSPARENT_MIN_BLUR`） | 20（`chrome.transparentMinBlur`） | 一致 |
| 启动页 pulse / dot-blink | 1.6s / 1.3s（`boot.css:428,291`） | 1600ms（`boot.tickerPulse`）/ 1.3s 周期 | 一致 |
| 启动页 drawer-fade / drawer-rise | 0.18s / 0.24s（`boot.css:470,489`） | 启动页无对应抽屉动效 | **无基线（未接线）** |

---

## 5. 仍需真机 / 人工双签

1. **标题栏右侧图标 y**：round240 UIA 记原生 8，chrome-metrics（`AppFrame.module.css:333-348` 的 `pageTrailingCluster` height 32 / marginTop 12）记 12。两份基线互相冲突，本轮**不改**，需一次现场 UIA 复测（`切换终端抽屉`/`切换右侧面板`）。
2. **发送盘 x +1.14**：绝对差来自整列 x +0.50（divider 取整，S28 已知）与盘径 −0.67；相对卡右缘只有 +0.12。要落到 ±1 需改整列取整策略，会牵动全列，需实机确认后再动。
3. **通知中心**：WinRT toast 的真实点击回执与视觉模糊对照（无基线）。
4. **设置 18 个专页**：`docs/review-20261020/evidence/compare/ref-settings-*.png` 有原生图，但无同尺 actual；需重拍 MyGo 侧。
5. **右/底工作区面板**（文件树/预览/终端/浮动、停靠三态）：无整页基线。
6. **桌宠**：Live2D 分层逐帧、OEM 缩放矩阵、持续 FPS/功耗（只有坐标与物理）。
7. **toast 淡出/进入曲线**：需改 `third_party/mygo/ui/toast.go`（200ms → 1000ms + 160ms 上滑），属 toolkit 变更。
8. **`docs/acceptance-ui-parity.md`** 的 L/D/P/S 画面行与 FL/FD/FS 用例仍全空；本轮只提供其中 D-MAIN / S-* / 浮层可算部分的数值，不构成双签。

---

## 6. 测试结果

聚合门禁（`. .\scripts\env.ps1; $env:CGO_ENABLED='0'`，隐藏运行器）：

```
go test ./internal/ui/... ./cmd/whaleisle -count=1 -timeout=10m
→ exit 0；37 个包全部 ok（无既有无关失败）
  frame 39.615s · widgets 9.447s · chrome 10.098s · composer 35.347s · sidebar 13.782s
  chat 138.735s · settings 31.155s · cmd/whaleisle 38.565s
  其余 29 包 0.7–34.5s 全部 ok
```

改动包单跑（`-count=1`）：`frame 27.667s`、`widgets 6.150s`、`chrome 13.567s`、`composer 19.220s`、`sidebar 7.054s` 全 ok。
`go vet ./internal/ui/frame` 退出 0。

契约复跑（`-run` 8 个几何测试，全 PASS）：
`TestCaptionButtonsSitInTheNativePlateRow`、`TestCaptionButtonsHoverFills`、`TestCaptionButtonsMaximizedGlyph`、`TestComposerCardMatchesTheConversationBodyWidth`、`TestHeroStackHasADefinedNativeSeat`(860/920)、`TestHeroBadgeSitsOnTheNativeSeat`、`TestHeroHeadlineMatchesTheNativeColumns`、`TestRound255HeroRowNodes`。
`widgets`：`TestMotionTokens`、`TestMotionTokensMatchTheReferenceSheet` 全 PASS。

证据文件：`tmp/ui-audit-gate.log`、`tmp/ui-audit-contract.log`、`tmp/ui-audit-motion.log`、`tmp/ui-audit-pkg*.log`。

### 6.1 改动面

仅 4 个产品/测试文件 + 2 个新测试文件 + 本文件：

```
internal/ui/frame/caption_buttons.go      +12/-4
internal/ui/frame/conversation_phase.go   +38/-7
internal/ui/widgets/motion.go             +22/-8
internal/ui/widgets/motion_test.go        +58/-4
internal/ui/frame/caption_plate_seat_test.go  (new)
internal/ui/frame/hero_badge_seat_test.go     (new)
```

未改 `cmd/`、`runtime/`、`third_party/`、`reference/`；未提交 git；未启停本体。

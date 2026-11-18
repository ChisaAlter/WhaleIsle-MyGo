# 极简模式与启动卡顿建议（lane 交付记录）

日期：2026-11-18。工作树：`C:\AI\WhaleIsle-MyGo-mini`（detached `e6e19841`）。未提交、未推送，等待 PR #8 合并后由调度方指定落地分支。

## 1. 范围

补齐本体（`C:\AI\Deepseek-Harness-Desktop`，只读对照）有、MyGo 缺的一组外观能力：

1. 设置 → 外观顶部的「极简模式」开关：一个开关暂停壁纸、动态背景、透明模糊、指针与输入装饰、输入框流光、界面过渡。
2. 覆盖不销毁：暂停的每一项偏好原样保存在设置里，关闭极简模式后完全恢复。
3. 启动卡顿建议：观察启动成本，明显卡顿时一次性建议开启，可延后且跨重启有效，不重复打扰。
4. 文案：无原始技术串、无性能数字，全部 zh + en，失败信息说明下一步。

## 2. 三条关键决定

### 2.1 持久化放 MyGo 自己的 config.json，不放 Host 的 `ui-theme.minimalMode`

本工作树依赖的 vendored Host 没有这个字段：`runtime/vendor/deepseek-harness/packages/client/ui-theme/lib/index.js` 内 `minimalMode` 命中 0 次（`backgroundEffect`、`cursorEffectEnabled` 等同批字段都在），而 `ThemeBinding.Queue` 只接受 Host `describe` 出来的字段（`internal/store/settings/host_theme.go:140-149`），写入会被拒绝。

改为客户端策略后：第一帧就生效、Host 未连接也成立、关闭即刻恢复。代价是与 Electron 本体各存一份（本体在 `settings.yaml` 的 `ui-theme.minimalMode`），同一 profile 双端不同步。

字段（`internal/config/config.go:177-187`）：

- `goMinimalMode`（bool，默认关）
- `goMinimalModeSuggestionsOff`（bool）：以「关」的形式存储，使从未设置过的旧 profile 默认仍会收到建议。
- `goMinimalModeSnoozedUntil`（RFC3339）：读不出来的值在 `normalize` 里清空（`config.go:530`），坏时间不会把建议永久封死。

### 2.2 覆盖行的处理：保持可编辑，行内注明「被极简模式覆盖」

与本体文档一致（`docs/handbook/modules/settings.md`「下方特效偏好仍可编辑，关闭极简模式后按保存的偏好恢复」）。不置灰、不改值：用户是在预先调节「关掉极简之后要恢复成的样子」。

带该提示的 8 行：背景图、背景特效（含速度/光团数）、指针特效（含速度/大小）、玻璃透明度、终端透明度、透明主题、输入特效、发送消息时的思考炫光。实现 `internal/ui/settings/minimal_mode.go:55`。

### 2.3 「界面过渡」用一个框架 seam，而不是 38 个 `.Transition(` + 101 个 Reveal 站点

仓库里 `.Transition(`/`ElementTransition{` 38 处、`Reveal*` 101 处（非测试）。逐点判断不可维护，且新增界面会漏。

`third_party/mygo/ui/motion_policy.go:1` 新增 `ui.SetReduceMotion(bool)`，`preferences.go:32` 把应用级覆盖 OR 在桌面设置之上；框架原本就在 transition（`transition.go:284,549`）、menu（`menu.go:655`）、toast（`toast.go:489`）、list（`list_motion.go:152`）、router（`router.go:592`）、decorative（`decorative_effects.go:60,207,269`）读 `Preferences.ReduceMotion`。这是本 lane 唯一改到 `third_party/mygo` 的地方，release 构建（`-tags release`）不依赖任何测试符号。

## 3. 实现清单

| 位置 | 内容 |
| --- | --- |
| `internal/renderpolicy/renderpolicy.go:21-57` | 策略层：`Minimal()` / `SetMinimal()` / `Wallpaper()` / `Background()` 纯投影，不写回存储 |
| `internal/ui/settings/minimal_mode.go:66` | 外观页首整块：标题 16 / 描述 12 / 右侧 44×28 switch；第二行「启动卡顿时提示」可重新开启（`:100` 起） |
| `internal/ui/settings/appearance.go:41` | 插在颜色模式之前；`AppearanceAccess.Minimal` |
| `internal/ui/settings/wallpaper_modals.go:68` | 背景图行提示 |
| `internal/ui/settings/background_effects.go:31,73,74` | 背景特效开关、速度、光团数 |
| `internal/ui/settings/pointer_effects.go:28,65,66` | 指针特效开关、速度、大小 |
| `internal/ui/settings/wallpaper_background.go:498,513,524` | 玻璃透明度描述、终端透明度、透明主题 |
| `internal/ui/settings/effect_preferences.go:411` | 输入特效 / 思考眩光 |
| `cmd/whaleisle/minimal_mode.go:28-91` | 应用策略、落盘、写失败留错、外观页 access |
| `cmd/whaleisle/minimal_mode_offer.go:46` | 一次性建议卡（标题 / 说明 / 三个答案 / 失败槽） |
| `cmd/whaleisle/wallpaper_background.go:211` | 极简下任何皮肤都不画背景（含 codex） |
| `cmd/whaleisle/glass_backdrop.go:26` | Acrylic → None |
| `cmd/whaleisle/pointer_effects.go:24` | 指针装饰投影关闭 |
| `internal/ui/frame/frame.go:898-900` | 输入框流光 / 输入特效关闭 |
| `cmd/whaleisle/main.go:1979` | 启动即应用 + 建观察器 |
| `cmd/whaleisle/main.go:2957` | 建议卡渲染挂载 |
| `cmd/whaleisle/main.go:3412` | 极简下了 Acrylic 后，平暗画布自绘不透明 |
| `cmd/whaleisle/main.go:4040` | 外观页接 `Minimal` |
| `cmd/whaleisle/ui_diagnostics.go:145` | 每帧成本喂给观察器 |
| `internal/config/config.go:177-187`, `:530` | 三个字段 + RFC3339 规范化 |
| `internal/ui/settings/search.go:72-73`, `search_target.go:166-169` | 设置搜索的两条新入口 |

### 3.1 启动卡顿建议的判定参数（`internal/startupsuggest/startupsuggest.go`）

- 前台就绪累计 10s 后才开始观察；观察窗口 60s。
- 卡顿判据：单帧视图构建 > 16ms（`StallThreshold`，与仓库既有慢帧口径一致）。
- 需 ≥3 次（`MinStalls`）且落在 ≥2 个不同 15s span（`StallSpan`/`MinSpans`），单次爆发不算。
- 帧间隔最多计 250ms（`MaxGap`）：空闲与休眠跳变不是证据；失焦/隐藏帧不累计。
- 无定时器、无 goroutine、无空闲逐帧循环：只由已有的 `recordViewFrame` 帧钩子驱动；决定后每帧只剩一次指针判断。
- 每次进程启动最多提一次；闸门在 `cmd/whaleisle/minimal_mode.go:117`（已开启 / 已关提示 / 7 天延后都不提）。
- 三个答案：开启（写 `goMinimalMode`）、暂不开启（`goMinimalModeSnoozedUntil` +7 天）、不再提醒（`goMinimalModeSuggestionsOff`）。关闭卡片（遮罩 / Escape）= 暂不开启。

## 4. 证据（本机真实运行）

目录 `tmp/minimal-mode/`（`tmp/` 在 `.gitignore:32` 下，不会进提交）：

| 文件 | 大小 | reviewer 是否需要 |
| --- | --- | --- |
| `appearance-minimal-off.png` | 181,504 B | 需要（开关关） |
| `appearance-minimal-on.png` | 186,673 B | 需要（开关开 + 8 行覆盖提示） |
| `startup-offer.png` | 27,791 B | 需要（建议卡） |
| `render.json` | 3,390 B | 需要（量测） |
| `offer.json` | 506 B | 需要（量测） |

量测（`render.json` / `offer.json`）：

- off 屏：`Turn on minimal mode` switch 位于 `(676, 18, 44, 28)`；`overriddenRows=0`。
- on 屏：同坐标 `Turn off minimal mode`；`overriddenRows=8`，`overrideNotes` 逐条列出 8 行原文。
- 两屏行位置完全一致（背景图 y=812、思考眩光 y=2307.24），只有提示文本变长 → 开关不搬布局。
- 建议卡 900×700：标题 `(250,260,400,24)`，三个答案 y≈419，从右到左「开启 / 暂不开启 / 不再提醒」。
- 全页高度 2600：特效行在 1400 高的窗口里位于折叠线以下，因此证据用了 900×2600 一次出全。

复现：

```powershell
$env:WHALEISLE_MINIMAL_MODE_EVIDENCE="C:/AI/WhaleIsle-MyGo-mini/tmp/minimal-mode"
go test ./internal/ui/settings/ ./cmd/whaleisle/ -run "TestMinimalModeRendersOffOnAndOverridden|TestMinimalModeOfferPresentsThreeAnswers"
```

## 5. 验证

- `gofmt`：本次改动的文件全绿。
- `go vet`：`internal/ui/settings`、`internal/renderpolicy`、`internal/startupsuggest`、`internal/ui/frame`、`internal/config`、`cmd/whaleisle` 全过。
- `go build -p 2 ./...` 通过。
- 包测试：`internal/ui/settings` 16.7s、`internal/ui/frame` 13.1s、`internal/config` 9.2s、`internal/renderpolicy`、`internal/startupsuggest`、`cmd/whaleisle` 整包 27.7s —— 全绿。
- 新增测试覆盖：开关真点往返（回调 + 页面状态）、建议开关可恢复、8 行覆盖提示、观察器 5 种场景（慢 / 快 / 单次爆发 / 后台 / 休眠跳变）、闸门与 7 天 snooze（含坏时间被清）、投影不改存储、`desktopGlassMaterial` 与 `chrome.BackdropActive()` 在极简开/关下的实际绘制路径。

## 6. 已更正：静态视图渲染的边界（toast 观察的结论）

> 本节原先把「默认 toast look 在我用的渲染 harness 里不进帧」写成了对应用的怀疑，这个推论是错的，已由 boot lane 在真实窗口上推翻。测量保留，推论撤回，并记下规则。

**测量（仍然成立，只是它描述的是 harness）**：400×300 视图，`ui.NewTester`，时钟推进 0 / 500ms / 2s 三帧，比较 `tt.Image().Pix` 的 SHA-256：

| 视图 | 帧 1 / 2 / 3 | 标签树 |
| --- | --- | --- |
| 视图内 `c.AddToast(ui.Toast{ID:"probe", Title:"Probe toast title", Timeout:-1, Action:"Do it"})` | `954366b8de34af36` 三帧完全相同 | 含 `Probe toast title`、`Do it` |
| 同上，但视图调用 `ui.ToastViewportBase` 并画一张有色卡片 | `ad610e773bad02b2` 三帧相同，且与上行不同 | 含 `Probe toast title` |

**已推翻的部分**：toast 在应用里是可见的。boot lane 在真实窗口上用确定性失败取证：同一条 toast 矩形在失败前有 0 个像素，失败后有 172,151 个接近主题反色的近乎白色像素加 4,608 个墨色像素。代码路径也不需要应用侧挂载：`c.AddToast` 只是登记并请求一帧（`third_party/mygo/ui/toast.go:134`），引擎每帧在视图之后无条件挂载并绘制 viewport（`third_party/mygo/ui/runtime.go:416-422`、`toast.go:240`）；另外 toast 入场前 160ms 从透明度 0 开始（`toast.go:16`），冻结的时钟看到的本来就是空帧。

**规则（本次要沉淀的）**：静态视图渲染只证明视图自己的元素树，不证明引擎在视图之后挂载的覆盖层（toast、菜单覆盖层、`c.overlay` 对话框）。不要用一张静态视图渲染去断定某个界面在应用里不可见。

**对代码的影响**：无。建议卡保持主窗口内共享模态面卡片（`cmd/whaleisle/minimal_mode_offer.go:46`），但理由不再与 toast 可见性有关：它是一次性的、必须等到答案的三角提问，不设自动过期，这正是共享模态面的形状，而不是 toast 的「单个动作 + 定时消失」。36 个文件、44 处既有 `c.AddToast` 调用一律不受影响，调查期间列的调用点清单已无意义，撤掉。

## 7. 未验证项

- **Electron 本体与 MyGo 各存一份开关**：同一 profile 下两端互不同步。属于跨客户端一致性问题，未设计也未实测。
- **codex 皮肤**：极简下同样不画背景（走 `wallpaperPainter` 的统一分支），但该皮肤的玻璃走 `flatChrome` 分支（`glass_backdrop.go:39`），没有做皮肤实机断言。
- **真实窗口观感**：所有证据来自 `ui.NewTester` 软件渲染，未跑真实窗口 / DPI / 动效观感。
- 建议卡在真实窗口里与输入法、其他模态的叠加顺序，未实测（代码上它在会话路由下渲染，settings / market / 远端等路由下不出现：`minimal_mode_offer.go:32`）。

## 8. blast radius（已核）

- `settings.AppearanceAccess` 构造点：`cmd/whaleisle/main.go:4037`（embedded 空结构，`Minimal == nil` 时整块不渲染）、`main.go:4040`（已接）、`internal/ui/settings/appearance_scheme_test.go:24`（nil-safe）。
- Host 侧未动：`ui-theme` schema、`ThemeBinding` 无改动；`composerBeam` / `typingFx` 只在 `internal/ui/frame/frame.go` 做显示门控，Host 偏好原样 → 关闭极简即恢复。
- `ui.SetReduceMotion` 是进程级：仅 `applyMinimalMode`（1 处）与测试调用；影响所有窗口的动效，这正是本功能的意图。
- `wallpaperSliderStep` / `wallpaperSlider` 私有签名新增 `hint` 参数：8 个调用点全部更新（编译通过即为证）。
- 设置搜索新增两条：只影响搜索结果。
- `tmp/minimal-mode/` 全在 `.gitignore:32` 的 `/tmp/*` 下，`git status` 里只有源码改动，没有大文件或生成物。

## 9. 后续

等 PR #8 合并后按调度方指令：从当时 main 建 `feat/minimal-mode` → 提交本 lane 源码改动与本文档 → 开 PR。证据 PNG/JSON 留在工作树 `tmp/minimal-mode/`，作为 PR 描述里的量测来源，不进提交。

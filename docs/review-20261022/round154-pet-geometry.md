# Round154 · 桌宠窗口几何对齐（原生 240×260）与实测

## 1. 原生证据（只读）

任务里写的 `reference/electron/src/main/pet.js` **不存在**；原生桌宠有两个实现，几何分别在：

- **现役 live2d 桌宠** `reference/electron/src/main/desktop-live2d.js`：
  - `:35-36` `const PET_WIDTH = 240; const PET_HEIGHT = 260;`
  - `:37` `const EDGE_GAP = 24;`
  - `:82-88` `defaultPosition(workArea)`：`x = area.x + max(0, area.width - PET_WIDTH - EDGE_GAP)`，`y = area.y + max(0, area.height - PET_HEIGHT - EDGE_GAP)`（右下角、四周留 24）
  - `:27-33`、`:757-790`：窗口本身是**覆盖整个虚拟屏幕的透明 overlay**（`overlayBounds()` = 所有显示器并集，`getBounds` 后再次 `setBounds(overlayBounds())`；`transparent:true, frame:false, resizable:false, minimizable:false, maximizable:false, fullscreenable:false, skipTaskbar, alwaysOnTop, focusable:false`），**没有任何 `setSize`/`setBounds` 用于桌宠本体**；窗口永不随行为/成长改变大小。
  - `:437` 命中区域：`{x, y, w: PET_WIDTH * scale, h: PET_HEIGHT * scale}` —— 成长只放大**精灵的命中范围**，不是窗口。
  - `:670`、`:1229-1240`：显示器/截图查找以 `pet.x + PET_WIDTH/2, pet.y + PET_HEIGHT/2` 为中心。
  - `:950-956`：持久化位置按 home display 夹紧（`home.x + home.width - PET_WIDTH` / `… - PET_HEIGHT`）。
- **旧版精灵桌宠** `reference/electron/src/main/desktop-pet.js`：`:8-9` `PET_SIZE = 88`、`EDGE_GAP = 16`，另有 `:12` `DESKTOP_PET_FEATURE = false`（默认关闭），不是本轮目标。

结论：原生 240×260 是**桌宠精灵在屏幕上的足迹**（默认位置、夹紧、命中、显示器中心都用它），不是 OS 窗口矩形；窗口是整屏透明 overlay，因此**不存在**「随行为变化的高度」。

## 2. MyGo 改动

之前：`companion.go` `Width: 220, Height: petWindowHeight(behavior), MinHeight: 438`，`behavior.go` `petWindowHeight = 438 + ceil(max(0, scale-1)*110)`，`applyBehavior` 调 `SetSize(220, petWindowHeight(...))`——即窗口随行为 Scale 变高、宽度不是原生的 240。

现在（新增 [`internal/desktop/pet/geometry.go`](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/geometry.go>)）：

| 常量 | 值 | 来源 |
|---|---|---|
| `PetStageWidth` | **240** | desktop-live2d.js:35 |
| `PetStageHeight` | **260** | desktop-live2d.js:36 |
| `PetEdgeGap` | **24** | desktop-live2d.js:37 |
| `PetStatusPanelHeight` | 178 | MyGo 页面自带的状态/操作面板（web 渲染层成本，非原生窗口） |
| `PetWindowWidth` | 240 | = 原生足迹宽 |
| `PetWindowHeight` | 438 | = 260 舞台 + 178 页面面板 |

- `companion.go`：窗口 `Width: PetWindowWidth, Height: PetWindowHeight, MinHeight: PetWindowHeight`，创建后 `w.SetSize(petWindowSize())`。
- `behavior.go`：`petWindowHeight(behavior)` 保留函数名（调用方/测试稳定），但**恒返回 `PetWindowHeight`**——原生从不因成长改窗口；`applyBehavior` 改为 `SetSize(petWindowSize())`（幂等常量，不再随 Scale 变化）。页面仍通过 `--pet-scale` 放大精灵本身。
- 没有改页面布局：状态卡依旧在 150px 偏移处并可滚动，438 高度与原值相同，只是宽度 220→240（+20 DIP，页面 `max-width` 均为固定像素，不会被拉伸）。

## 3. 测试

- 新增 [`geometry_test.go`](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/geometry_test.go>)：钉住 240/260/24 与 240×438（=260+178）；`petWindowSize()` 一致；`petWindowHeight` 在 scale 1/1.1/1.5/2/3 下都等于 438（不随行为变化）。
- `go test ./internal/desktop/pet` → ok 2.3 s（含既有 wander/quickchat/rig/motion 用例）；`go vet ./internal/desktop/pet` → 0。
- 曾尝试把 `motion_state_test.go`/`quickchat_window_test.go` 里的 220×438 夹具一并改成 240×438，两个用例的前提值随之失效（它们用窗口宽度参与漂移/放置期望），已还原：那些数字是**输入夹具**而非窗口断言，窗口尺寸由 `geometry_test.go` 钉住。

## 4. 实测（临时构建，pass277 未覆盖）

`tmp/whaleisle-pet-geometry.exe`（当前树 + 本轮改动）；throwaway profile（真实 config + `mygoDesktopPet.enabled=true` + welcome marker + 临时 workspace，port 3098），`WHALEISLE_VENDOR` 指向默认 vendor，`EnumWindows` + `GetWindowRect` + `GetDpiForWindow`（per-monitor-v2，物理像素；DIP 由 DPI 换算）：

| 窗口 | 物理 | DIP | 位置 | DPI |
|---|---|---|---|---|
| Whale Isle（主窗） | 2160×1380 | 1440×920 | (200,0) | 144 |
| **桌宠（Whale Isle companion）** | **未出现** | — | — | — |

主窗几何正常，端口 3098 已监听，但**桌宠窗口没有创建**（同一 PID 的 `MyGoWindow` 只有主窗；launcher 关闭后也只剩主窗）。

已排除配置读取问题：用同一个 profile 单独跑 `config.Open(paths.UserDataDir())` → `DesktopPet().Enabled == true`（`config path: …\wl-pet-geom2\config.json`）。也就是说启动链路 `pet.New(cfgStore)`（`companion.go:70-75`，`Enabled: cfg.Enabled`、`ready: true`）→ `SetReady(!isolatedDesktopRun())`（`main.go:2322`，`companion.go:130` 会立刻 `p.sync()`）应该走到 `sync()` 的建窗分支，但窗口没有出现，且 `sync()` 之后再无入口（只有 `Save`/`SetEnabled`/`recovery` 会再调它）。最可能的原因：`SetReady` 在主窗 `NewWindow` 的创建回调里被调用，此时 `mygo.NewWindow` 创建第二个窗口失败（`p.Error = KeyDesktopPetWindowCreateFailed` 后 `close()`），而启动后没有任何重试。**这是启动路径的真实缺陷，不是本轮几何改动引入的**，需要单独一轮修（例如把首次 `sync()` 放到 `App` 就绪/主窗首次 Show 之后，或在 `SetReady(false→true)` 时重试一次）。

因此**窗口实测值缺失**；按代码推算 150% 下应为 240×438 DIP = **360×657 物理**（dpi 144）。这一项要在桌宠能被启动后复测，不能当作已验证。

## 5. 渲染层差异（保留，不伪造）

- 原生把 live2d 精灵直接合成在整屏透明 overlay 上，240×260 是精灵足迹；MyGo 用 web 页面（`pet.html`：canvas 192×208、CSS 96×104、rig/neural 88×110，`--pet-scale` 缩放）加状态/操作面板，窗口是页面表面 240×438。
- 差异：原生无窗口级尺寸（整屏 overlay，点击穿透由 renderer 命中测试切换），MyGo 是真实小窗、总是可交互；原生 24 边缘间隙用于默认位置与夹紧，MyGo 目前保存绝对位置（无 ratio 模型、无 EDGE_GAP 夹紧），本轮只固定了 `PetEdgeGap=24` 常量备用，未改拖拽/持久化模型。
- 本轮**没有**改渲染器/tier，也没有声称 live2d 等价。

## 6. 残留

1. 桌宠窗口在本轮 throwaway profile 下未创建，需确认启动时 `sync()` 的触发条件后复测 240×438（150% → 360×657）。
2. 边缘间隙/位置：原生 ratio + 24 夹紧，MyGo 绝对坐标；若要完全对齐需要改 `config.PetPosition` 语义（本轮未做）。
3. 无行为相关高度：已确认原生不存在，MyGo 的 `petWindowHeight` 现为常量，但页面状态面板仍占 178 DIP——渲染层差异如实保留。

## 7. task-29 补记：窗口为什么不见（已定位），以及仍未拿到的实测

用 `WHALEISLE_PET_DEBUG=<file>` 打开了 pet 包里的轻量埋点（`internal/desktop/pet/debug.go`，未设置该变量时零 I/O），在同一次启动里拿到了完整因果链：

```
sync enabled=true ready=true win=false
newWindow nil=false destroyed=false      <- 窗口创建成功，不是 NewWindow 失败
sync enabled=true ready=true win=true
didFailLoad url=whale-pet://localhost/index.html code=14 desc=navigation failed (COREWEBVIEW2_WEB_ERROR_STATUS 14)
didFailLoad: retrying once
didFailLoad ... code=14                  <- 重试仍失败
close: destroy window                    <- 自家 OnDidFailLoad 里 p.close()
```

结论（已证实，非猜测）：
1. **不是**创建失败——`mygo.NewWindow` 返回了可用窗口；round154 的「第二次 NewWindow 在创建回调里失败」假设不成立。
2. 真正的失败是 **`whale-pet://localhost/index.html` 自定义 scheme 导航被 WebView2 中止**（状态 14 = `CONNECTION_ABORTED`）。
3. 页面失败后 `w.Page().OnDidFailLoad` 直接 `p.close()` 销毁窗口，且**启动后再无人重新创建**（`sync()` 只由 `Save`/`SetEnabled`/`SetReady`/`recovery` 触发），于是整个会话都没有桌宠。

本轮已落地的改动（都在 `internal/desktop/pet/` + 一行 `cmd/whaleisle` 生命周期接线，未改渲染器）：
- **生命周期 seam**：建窗改走 `openCompanionWindow` 变量（`geometry.go`），测试可替换；主窗 `win.OnShow` 时重新 `SetReady(!isolatedDesktopRun())`（`main.go:2322-2328`），主窗重建（启动器→桌面、恢复重载）后桌宠会重新出现。
- **加载失败可观测 + 一次重试**：`OnDidFailLoad` 先记 `url/code/desc`，未重试过则 `Reload()` 一次，仍失败才记录 `p.Error` 并关闭。
- 新增 `lifecycle_test.go`：未 ready 不建窗；ready 建一次；`Close()` 后再次 `SetReady(true)` 会重建；禁用时不再建窗且不残留句柄（用 seam 计数，无需 GUI）。

**仍未完成**：`whale-pet://` 导航中止的根因要在 WebView2/自定义 scheme 层继续查（mygo 约定「scheme 必须在使用它的窗口创建前注册」，而 pet 的 `Protocol.Handle` 在 `New()` 里、建窗前执行；需要对比主窗 `whaleisle://` 的注册时机与 webview 初始化顺序）。因此**实测 240×438（150% → 360×657）依然缺失**，本轮没有 `pet-window-live.json`，不把它当作已验证。

## 8. task-30 补记：提前注册无效，问题在窗口的 WebResourceRequested 过滤器

**假设**（task-30）：注册发生在主窗 WhenReady 回调里太晚，scheme 列表已被快照，于是 `whale-pet` 没进 WebView2 过滤器。

**改动**：新增 `internal/desktop/pet/protocol.go`——包 `init()` 在 `mygo.App.Run` 之前调用 `Protocol.Handle("whale-pet", petProtocol{})`，`New()` 只把当前 Companion 发布到 `atomic.Pointer`；`petProtocol` 每次请求读当前实例，无 Companion 时 404。`protocol_test.go` 覆盖：包初始化后 `IsHandled("whale-pet")`；无 Companion 时 404；`New()` 后返回 200 且是宠物页面；替换实例后分发到新实例。

**验证结论**：提前注册**没有修复**导航中止。实测日志（temp build）：

```
sync ready win=false
newWindow nil=false destroyed=false
sync ready win=true
didFailLoad url=whale-pet://localhost/index.html code=14 COREWEBVIEW2_WEB_ERROR_STATUS 14
didFailLoad: loading http://whale-pet.localhost/index.html   <- mygo 在 Windows 上的映射形式
didFailLoad url=whale-pet://localhost/index.html code=14      <- 仍然 14
close: destroy window
```

映射后的 `http://whale-pet.localhost/index.html` 同样 14，说明该窗口的请求根本没进 WebResourceRequested：mygo 的 Windows 后端（`internal/windows/webview.go:26-33` 映射、`:312` 按 `o.Schemes` 逐个 `addFilter("http://<scheme>.localhost/*")`）拿到的 `o.Schemes`（`window.go:510` 由 `Protocol.schemes()` 求值）不含 `whale-pet`。而 `schemes()` 是动态读 map、init 已注册，所以「注册晚」不是原因；剩下最可能的是过滤器只按 WebView2 环境/首个 webview 生效，或 pet 窗口的 platform 选项来自另一份快照。

**mygo 需要暴露/修正的两点**（本轮不动 third_party）：
1. 记录或提供查询：每个窗口实际生效的 `Schemes` 列表（便于像本轮这样一次定位）。
2. 保证「注册早于任何窗口创建」的 scheme 一定带上过滤器；若过滤器确实只在环境创建时生效，则要求在环境创建前完成注册（或在窗口创建时对新增 scheme 追加 `AddWebResourceRequestedFilter`）。

**因此实测仍缺失**（240×438 DIP / 150% → 360×657 px 与截图），未生成 `pet-window-live.json`，不记录任何未测数字。本轮的实际产出：scheme 分发器（注册与实例解耦、幂等、可测试）、失败可观测（URL/code/desc 落日志）、映射 URL 的一次兜底重试，以及把问题精确定位到「窗口过滤器里没有 Whale-pet」。

测试：`go test ./internal/desktop/pet` ok 3.2 s；`go test ./cmd/whaleisle -run "Pet|Companion"` ok 5.9 s；`go vet ./internal/desktop/pet ./cmd/whaleisle` → 0。

## 9. task-34 补记：真正的原因就是「建了窗口但从不 Show」，实测已拿到

**根因（已证实）**：`companion.go` 建窗时 `Hidden: true`，而 pet 包里**从来没有调用过 `Show()`**。实测把窗口在加载前 `Show()` 之后：

```
sync ready win=false
newWindow nil=false destroyed=false
showing before load
didFinishLoad: showing the companion      <- 页面加载成功（240×438 的页面）
willNavigate whale-pet://localhost/interactive-regions?data={…Width:240,Height:438…}
didFailLoad ... code=14
didFailLoad: ignored, the page already loaded
```

即：**隐藏的 WebView2 controller 会中止自定义 scheme 的导航（状态 14）**；一旦窗口可见，`whale-pet://` 正常加载，页面自己上报的命中区域宽度/高度正是 240/438。之后页面加载完成的再导航失败（`interactive-regions` 触发的同 URL 14）不再关闭窗口（`p.loaded` 保护）。

**生命周期（本轮落地，`internal/desktop/pet/`）**
- `sync()` 建窗后立即**按原生默认位置摆位**：主屏工作区右下角、四边留 `PetEdgeGap = 24`（`desktop-live2d.js:37,82-88`）；有记忆位置时按工作区 clamp（`clampPetPosition`，对应 `desktop-live2d.js:950-956`），显示器变化不会把桌宠丢到屏外。
- 为避免白闪：先保持 `Hidden: true`，页面 `OnDidFinishLoad` 后 `p.loaded = true` 再 `Show()`；加载失败（且未加载成功过）仍按旧逻辑重试一次映射 URL，再失败才 `p.Error` + 关闭。
- 禁用/`Close()` 仍走 `p.close()` 销毁窗口；重新启用/主窗重建后由 `SetReady` + `OnShow` 重新建窗并显示（task-29 的 seam）。
- 新增/更新测试：`TestDefaultPetPositionMatchesNative`（右下角 24 间距、负坐标屏、越界 clamp、区内不动）、既有 `geometry_test.go` / `lifecycle_test.go` / `protocol_test.go` 全绿；`go test ./internal/desktop/pet` ok 3.5 s，`go test ./cmd/whaleisle -run "Pet|Companion"` ok 3.7 s，`go vet` → 0。

**实测（temp build，pass278 未覆盖；throwaway profile + `mygoDesktopPet.enabled=true` + welcome marker + 临时 workspace，默认 runtime/vendor，port 3089）** —— `evidence/pet-window-live.json`：

| 项 | 值 |
| --- | --- |
| HWND / 类 / 标题 | 6294250 / `MyGoWindow` / `Whale Isle companion` |
| 可见性 | **visible = true**（加载后显示） |
| 物理矩形 | **360×657 px**，位于 (2165, 675)，dpi 144 |
| DIP | **240×438**（= 240×260 原生舞台 + 178 页面状态面板） |
| 页面 | `didFinishLoad` 成功；页面命中区域自报 `Width 240 / Height 438` |
| 截图 | `evidence/screens/pet-window-240x438.png`（鲸鱼娘分层 rig + 状态按钮，见文档旁附图） |

结论：**窗口几何与可见性已实测对齐**（240 宽=原生足迹、438 高=原生 260 + 页面面板 178），位置符合原生右下角 24 间距规则；渲染层差异（web 页面 + 状态面板 vs live2d 整屏 overlay）继续保留为已知差异，未伪造渲染器。

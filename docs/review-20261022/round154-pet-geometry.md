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

# Round161 · 桌宠静息呈现对齐原生 live2d（去掉常驻标签，状态改为原生式唤出）

## 1. 原生「静息时可见什么」（file:line）

- `reference/electron/src/renderer/pet-live2d.html:275-303`：`<body>` 里只有 `<canvas id="pet">` 常驻；`#pet-chat`（:277）与 `#pc-picker`（:296）都带 `hidden`，由交互唤出。**静息时没有任何常驻文字标签或按钮**。
- `reference/electron/src/main/desktop-live2d.js:35-37`：精灵足迹 `PET_WIDTH 240 / PET_HEIGHT 260`，`EDGE_GAP 24`；`:437` 命中区域按足迹 × 缩放。
- `reference/electron/src/renderer/pet.css`（旧精灵页）：圆形 72×72 底、`is-sprite` 时无边框无阴影——即「只有角色本体」这一取向。
- 状态/菜单由原生右键与托盘提供（`desktop-live2d.js` 的窗口菜单/托盘路径），不是页面常驻按钮。

## 2. 改动（internal/desktop/pet/assets + 一处 JS 逻辑）

| 项 | 之前 | 现在 |
| --- | --- | --- |
| 渲染器标签 | `#render-notice` 常驻显示「原版分层动画 / Layered rig」（`pet-runtime.js:224` 无条件写入） | 成功绘制后清空文本，配合新 CSS `#render-notice:empty{display:none}` 隐藏；仅当 shell 服务端已给出降级说明（`{{RENDER_NOTICE}}` 非空）时保留 |
| 状态入口 | 常驻「状态 / Status」按钮 | 静息不可见；**悬停**、**键盘 focus-visible** 或 **右键（contextmenu）** 才出现/打开；长按 600ms 亦可 |
| 状态卡 | 同上 `hidden` 默认 | 不变：仍默认关闭、`Escape`/点外部关闭、关闭后焦点回到入口（键盘可达），卡内所有动作（喂文件/摸摸头/刷新粮碗/喂 Token/休眠/聊聊/宠物主会话/看看屏幕）原样保留 |
| 布局 | 220 → 240 宽，标签+按钮占顶部 | 标签与按钮不再占位，静息只看到角色本体 |

CSS 关键行（`pet.html`）：

```css
#pet-status-toggle{position:fixed;top:4px;right:6px;z-index:3;opacity:0;pointer-events:none;transition:opacity 120ms ease}
body:hover #pet-status-toggle,#pet-status-toggle:focus-visible,#pet-status-toggle[aria-expanded="true"]{opacity:1;pointer-events:auto}
#render-notice:empty{display:none}
```

`pet-status.js` 新增：精灵/后备图上的 `contextmenu` → `show(true)`；长按；打开时把焦点移入卡片第一项；`Escape`/外部点击关闭后焦点回到触发按钮。

## 3. 窗口尺寸评估（是否缩到 260）

原生静息只需要 240×260 的足迹，但 MyGo 的状态卡是**页面内的面板**：`.task{position:absolute;top:calc(150px*scale);bottom:0}`，内容含聊天气泡、任务、成长、休息、多个动作链接。若把窗口高度压到 260：

- 卡片从 150px 起只剩 110px 可视高度，`max-height:calc(100vh-130px)` 虽可滚动，但一次只能看到 1–2 个动作，且卡片会与 260 高舞台重叠；
- 需要额外开一个 popover/独立窗口来承载卡片——那会引入第二个窗口与定位/焦点/关闭同步成本。

结论：**本轮保持 240×438**（240 原生足迹宽 + 178 页面状态面板），把「缩到 260 + 卡片外置」列为后续可选项，不在本轮冒险。静息视觉已与原生一致（只有角色），差异只在窗口高度这一结构项。

## 4. 前后对照（temp build，pass279 未覆盖）

| 阶段 | 文件 | 观察 |
| --- | --- | --- |
| 之前 | `evidence/screens/pet-window-240x438.png` | 角色 + 「原版分层动画 / Layered rig」标签 + 「状态 / Status」按钮，下方大块空白 |
| 之后 | `evidence/screens/pet-window-rest-after.png` | **只有角色本体**，无标签无按钮；360×657 px / 240×438 DIP @150%，位置 (2165,675) 不变 |

两次测量都来自同一台机器、同 dpi 144、同一 throwaway profile 套路（`mygoDesktopPet.enabled=true` + welcome marker + 临时 workspace，默认 runtime/vendor），窗口矩形一致，说明本轮只改了页面呈现。

## 5. 测试与检查

- 新增 `resting_test.go`：页面必须包含新的静息样式（隐藏入口、`:empty` 隐藏标签）、状态卡默认 `hidden`、`pet-status.js` 含 `contextmenu`/`Escape`/焦点处理、`pet-runtime.js` 不再无条件写 rig 标签且成功绘制会清空文本。
- `node --check pet-status.js` / `pet-runtime.js` → 0。
- `go test ./internal/desktop/pet` ok 2.9 s；`go test ./cmd/whaleisle -run "Pet|Companion"` ok 3.2 s；`go vet ./internal/desktop/pet ./cmd/whaleisle` → 0。

## 6. 残留

1. 悬停/右键打开的**视觉验证**只有代码与 DOM 断言，没有自动化截图（无头浏览器不在本轮环境内）；长按 600ms 的触屏路径同样只做了代码级检查。
2. 窗口高度仍为 438（见 §3）；若要 260，需要状态卡外置窗口。
3. 渲染层差异（web 页面 rig vs 原生 live2d 整屏 overlay、点击穿透语义）保持不变。
4. 未提交、未构建 passN exe、未改 runtime/ 与 third_party/mygo。

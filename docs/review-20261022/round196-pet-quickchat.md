# Round196 · 桌宠快捷聊天（G33）原生呈现与对齐

原生把快捷聊天做成**桌宠页内的一张浮动小卡**（`#pet-chat`），不是独立窗口；MyGo 原来开一个 460×620 的独立窗口放在桌宠旁边。本轮把独立窗口对齐到原生卡片的尺寸、锚点、间距与排版，并记录结构差异（MyGo 桌宠是原生 canvas 窗口，没有 pet.html 可挂 DOM 覆盖层）。

## 1. 原生证据（含数字）

- 卡片样式：`reference/electron/src/renderer/pet-live2d.html:25-273`。`#pet-chat`：`position: fixed`、`z-index: 11`、**width 248px**、背景 `bg-layer-1`、1px `border-l2`、圆角 **12px**、字体 12px、flex 列、`overflow:hidden`；隐藏用 `[hidden]{display:none}`，**无 transition/动画**（开合是瞬时显示）。
- 头部：`padding 8px 10px 6px`、底边 1px、gap 6；头像 16×16、名字 12px/600；关闭 ✕ 与 ↗ 按钮 11px、`padding 2px 4px`、opacity .6（hover 1）。
- 线程：`padding 8px 10px`、gap 5、**max-height 148px** 可滚；空态「和她说点什么…」11px/opacity .45；气泡 `max-width 86%`、`padding 5px 9px`、圆角 10（对方右下 4）、12px/1.5；错误行 11px 居中。
- 输入区：`padding 6px 10px 10px`、gap 6；textarea 1px 边框、圆角 9、12px、`padding 5px 8px`、**max-height 66px**、聚焦时边框变业务色；发送按钮 **26×26 圆形**。
- 模型选择：位于卡片内一行（`padding 5px 8px 3px`、高 28 胶囊、圆角 24、11px），菜单 `#pc-picker` 单独 portal：宽 `min(240px, 100vw-8px)`、max-height `min(280px, 100vh-8px)`、圆角 16、blur 16。
- 定位：`pet-live2d.js:1836-1849 chatAnchor()`——卡片水平居中于她的头部，优先放在头顶**上方 8px**；上方放不下时翻到脚下 8px；空间都不够时选露出她更少的一侧；四周再夹紧到宿主 **4px** 内。`syncChatPos()`（1853-1869）每帧按 `offsetWidth/offsetHeight` 重新贴附（卡片高度随对话增长）。
- 打开/关闭与焦点：`toggleChat()`（2260-2287）——`chatEl.hidden=false`、`dropAmbientBubbles()`、`chatRenderEmpty()`、`syncChatPos()`、`setInteractive(true)`（卡片加入 petBounds 以接收点击）、`chatFocusWindow(true)` 请求主进程把窗口改为可聚焦，60ms 后 `#pc-input.focus()`；关闭时隐藏、清空 chatRect、恢复不可聚焦、停轮询。`window focus` 事件（2424）在打开状态下再次聚焦输入框。
- 打开时的她：`wanderSuppressed()`（1703-1705）含 `chatOpen`，即打开聊天时**停止乱逛**；环境闲聊（priority<2）直接丢弃（1351-1356），审批/通知类气泡仍可见；拖拽/点击仍由 petBounds（含卡片）处理，卡片不挡她身体。

## 2. 截图证据

用 `scripts/native-web-shot.mjs` 以 file:// 加载 `pet-live2d.html`，注入两条消息并取消 `hidden` 后测得的卡片：**248×152**（两条气泡时）、radius 12px、font 12px（独立渲染缺少 dsh token，边框颜色不可见，尺寸为准）。截图：[round196-native-pet-chat-150.png](<evidence/screens/round196-native-pet-chat-150.png>)。

## 3. MyGo 现状（改前）

- [quickchat.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/quickchat.go>)：独立窗口 `Width 460, Height 620, MinWidth 420, MinHeight 580`，`Parent: p.win`、`AlwaysOnTop`、`SkipTaskbar`。
- [quickchat_window.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/quickchat_window.go>)：位置在桌宠**右侧** 8px（右侧放不下就放左侧），底边对齐桌宠底部，再夹到工作区。
- [quickchat_view.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/quickchat_view.go>)：外层 `Padding(14) Gap(10)`；名字 17px 粗体；气泡用会话字号；输入框用 composer 字号。

## 4. 决策：路径 (b) 对齐独立窗口

原生是页内 DOM 覆盖层，但 MyGo 的桌宠是 Go 原生 canvas 窗口（`internal/desktop/pet/motion.go` 绘制，没有 pet.html / pet-runtime.js 可挂载 DOM），做 (a) 等于把桌宠渲染改成 HTML 页，超出本片范围且会牵连动效/命中区。因此取 (b)：让独立窗口使用原生卡片的几何与排版，并把结构差异写进残余。

## 5. 实现

- `quickChatWidth=248`、`quickChatHeight=360`、`MinWidth=248`、`MinHeight=220`、`quickChatAnchorGap=8`、`quickChatInset=4`（[quickchat_window.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/quickchat_window.go>)）。
- `quickChatPosition` 按 `chatAnchor()` 重写：水平居中于桌宠、优先头顶上方 8px、放不下翻到脚下 8px、两侧空间都紧张时选露出她更少的一侧，最后夹到工作区 4px 内。
- [quickchat.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/quickchat.go>) 窗口选项改用上述常量（460×620 → 248×360，最小 420×580 → 248×220）。
- [quickchat_view.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/quickchat_view.go>)：外层 `Padding(10) Gap(6)`（接近原生头部 8/10/6 的节奏）、名字 12px/600 单行（原 17 粗体）、气泡 12px（原会话字号）、输入框 12px（原 composer 字号）。
- 打开时的她已与原生一致：`motion.go:186` 的 busy 含 `p.quickChatOpen`，即聊天打开时暂停乱逛；桌宠窗口保持可见（子窗口不遮她的锚点区），关闭后恢复。

## 6. 测试与截图

- [quickchat_window_test.go](<C:/AI/WhaleIsle-MyGo/internal/desktop/pet/quickchat_window_test.go>)：
  - `TestQuickChatEnterAndPlacement`：Enter / Shift+Enter / IME 不变；4 个位置用例——头顶上方（y=132、水平居中）、无空间翻到脚下（526）、右缘夹紧（-252）、左缘夹紧（-1916）。
  - `TestQuickChatNativeCardGeometry`：钉住 248 / 8 / 4 与最小尺寸；断言上方放置恰好留 8px、水平居中于头部；下方放置恰好离开脚 8px（不压她的身体）。
- 原生卡片截图见 §2；MyGo 窗口截图 blocked：需要在临时 profile 里启用桌宠并点击打开聊天（本片时间盒内未做），几何由测试钉住。
- 门禁：`go test ./internal/desktop/pet/... -count=1` → 3.030s ok；`go vet ./internal/desktop/pet/...` → 退出 0。未提交、未构建 passN；未改原生应用、runtime/vendor、third_party。

## 7. 残余

- **结构差异**：原生是页内覆盖层，卡片加入 petBounds、每帧随桌宠重锚、窗口在打开时切成可聚焦；MyGo 是独立子窗口——位置随桌宠移动/显示器变化重算（`OnMove`/`OnResize`/`OnDisplaysChanged`），但没有「同窗点击穿透/可聚焦切换」这套机制，且窗口不会随气泡数量自动长高（固定 360，原生按内容 148 线程上限自适应）。
- **内容差异**：原生卡片只有头像/名字/↗/✕、线程、模型胶囊、输入行；MyGo 窗口额外有会话 ID、直连/共享提示、取消等待/停止/刷新等按钮与底部说明（保留，未删）。原生模型菜单是 portal 弹层（240 宽、圆角 16、blur），MyGo 用窗口内 Select。
- **动画**：原生无过渡，MyGo 也未加；如需淡入需另立一项。
- 原生截图用的是独立 HTML（缺 dsh token），只作尺寸证据；颜色 token 以 round168 表为准。

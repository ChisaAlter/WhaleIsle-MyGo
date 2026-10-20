# W5 滚动条：MyGo 提供了什么、缺什么

调查对象：`third_party/mygo/ui`（只读，未改动第三方代码），以及本仓库四个自绘界面
（files / terminal / diffsurface / usage）。

## 结论（一句话）

**有 API，但只有"全局主题级"的宽与色：没有 per-container 样式，也没有 hover 加深。**
统一成 8px 圆角 thumb 只需要在应用主题里改两行（不在这轮写入域内）；"hover 加深"
必须由框架加字段，本轮不自绘、也未改第三方。

## 现有的 API

| 能力 | 位置 | 说明 |
| --- | --- | --- |
| 颜色 | `ui.Theme.Scrollbar Color`（`third_party/mygo/ui/theme.go:43`） | 亮/暗默认 `RGBA(0,0,0,0.32)` / `RGBA(255,255,255,0.35)` |
| 宽度 | `ui.Theme.ScrollbarWidth float32`（`theme.go:45`） | DIP；`scrollbarWidth()` 缺省 6 |
| 圆角 | `paint.go:748,755` | thumb 用 `bar.W/2` 全圆角,宽度 8 时圆角自然是 4,不需要额外字段 |
| 出现时机 | `paint.go:720-730` | 只在容器 hover 或正在拖动时绘制,否则完全不画 |
| 拖动态 | `paint.go:745-754` | 拖动时 thumb 加宽 2px |
| 位置 | `ui.Element.ScrollbarInsets(top,right,bottom,left)`（`element.go:587`） | 把轨道往容器内收,用于避开圆角/内边距 |

本仓库四个界面共 41 处 `ui.Scroll` / `ScrollHorizontal` / `ScrollBoth`,全部走框架绘制,
**没有任何一处自绘滚动条**（grep `Scrollbar|ScrollThumb` 在五个包里只命中壁纸缩略图）。
因此主题一旦改成 8px,四个界面会同时生效——这正是我们想要的"统一"。

## 缺口

1. **样式是全局的,无法按容器区分。** painter 读的是当前 context 的 `c.theme`
   （`paint.go:719`）,没有 `Element.ScrollbarStyle(...)`。文件树、diff 横向滚动、
   用量长列表只能共用一套。
2. **没有 hover / active 配色。** `hovered` 只决定"画不画"（`paint.go:728`）,
   没有 `ScrollbarHover` 之类的第二颜色,所以"hover 加深"无法表达。
3. **终端滚动条不走主题。** 终端字形绘制插件 `third_party/mygo/plugins/terminal/paint.go:596`
   有私有 `drawScrollbar`,与 `paint.go` 的滚动条是两套实现；改主题不会改变终端历史区的滚动条外观。
4. **没有"滚动位置"读取接口。** `myterm.Terminal` 只导出 `SelectionText/Title/Dir/…`,
   滚动偏移在私有 `view` 里。想做"已滚到底部"提示或"回到最新"按钮,需要框架先导出
   `ScrolledToBottom() bool` / `ScrollToBottom()`。

## 需要框架补什么（按优先级）

1. `Theme.ScrollbarHover Color` + painter 里 `hovered` 分支（10 行以内），解决"hover 加深"。
2. `Element.ScrollbarStyle(width float32, color Color)`（或 `ScrollbarWidth/C` 两个 setter），
   解决 per-container 区分；`ScrollbarInsets` 已经证明 element 上有 `barInset` 这类逐元素字段,加同构字段即可。
3. 终端插件改用主题的 `Scrollbar`/`ScrollbarWidth`（删掉私有 `drawScrollbar` 的颜色常量）。
4. `myterm.Terminal.ScrolledToBottom()`，给终端的"回到底部"提示铺路。

## 应用侧落地补丁（不在本轮写入域，交 theme/chrome owner）

`internal/ui/theme/interface_context.go` 的 `ApplyInterface` 已经是"复制当前主题再改字段"
的模式，加两行即可把全局滚动条做成 8px 圆角：

```go
next := *c.Theme()
next.Font, next.FontSize = font.Family, font.Size
next.ScrollbarWidth = 8
next.Scrollbar = next.Scrollbar.Alpha(1.15) // 或 chrome.Color(colors.Muted).Alpha(.45)
c.SetTheme(&next)
```

注意 `SetTheme` 是整棵树生效的，四个界面会同步变化；若只想改某一块，需要上面第 2 项的
per-container API。

## 本轮实际改动

无（不自绘）。设置页"高级"提示、diff 固定行号、终端快捷键提示、用量 tabular/加载态见
task-26 的回报。

# Round259 · 标题栏「新建会话」胶囊：它不是展开态控件

任务假设：原生有两个 新建会话（caption band 16/16.67/168.67 与 header 14/70/252），MyGo 缺前者。本轮用 native UIA + 截图 + 行为测试核查，**结论是假设需要修正**：那个 168.67 宽的胶囊是**收起态（rail）**的控件，在展开态里既不显示也不起作用；展开态唯一可见的仍是 y=70 的 252 宽按钮。所以**没有在展开态加可见胶囊**——那会把品牌区覆盖掉（视觉回归），只记录真实差异与下一步。

## 1. 实测证据

native 窗口（pid 15560，`Chrome_WidgetWin_1`，1512×911 DIP，150%）UIA：

| 元素 | rect (DIP) | offscreen | enabled | patterns |
| --- | --- | --- | --- | --- |
| 新建会话（caption band） | 16, 16.67, 168.67, 38.67 | **false** | true | Invoke, ScrollItem |
| 新建会话（header） | 14, 70, 252, 38 | false | true | — |
| 收起侧边栏 | 240, 22, 28, 28 | false | true | — |

截图（`evidence/round259/native-caption-left.png`）显示该矩形处画的是**品牌锁**：鲸鱼头像 + 鲸屿 WHALE ISLE + BASED ON DEEPSEEK HARNESS，下面是 252 宽的 ⊕ 新会话。也就是说 168.67 的按钮被品牌层遮住（或在 rail 断点才可见），a11y 树里仍然存在。

**行为测试**：对 caption band 那个元素调用 `InvokePattern.Invoke()`，等待 3 s 后重新枚举会话行——行列表与调用前**完全一致**（没有新增「新会话」行）。所以在展开态它确实是一个**无效果**的重复控件。

## 2. 我们这边缺的是什么

`internal/ui/sidebar/sidebar.go` 的 `collapsedRail` 已经有新建会话入口（`sidebar-rail-new`），但形态不同：

| | native rail | MyGo rail（现状） |
| --- | --- | --- |
| 形态 | 168.67×38.67 胶囊（图标 + 「新建会话」文字） | **36×36 图标按钮** |
| 位置 | caption band 起点 y=16.67 | rail 内、品牌按钮下方 |

也就是说真正的差异是**收起态 rail 的宽度与新建入口形态**（native rail 约 200 DIP 宽、胶囊 168.67；我们 rail 是窄图标条 56 左右）。这不是"展开态少了一个胶囊"，而是"收起态布局不同"。

## 3. 本轮未改代码的原因

1. 在展开态放一个可见胶囊会**覆盖品牌锁**（截图已证），与原生视觉不一致；
2. 真正的修复在 rail 态，需要先把 native 切到收起态测量（rail 宽度、胶囊内图标/文字尺寸、与 brand 按钮的间距），再改 `collapsedRail` 的布局——属于一次独立的结构调整，不能在 45 分钟盒子里带着 rail 宽度假设硬推。

## 4. 门禁

本轮无代码改动；`go test ./internal/ui/sidebar ./internal/ui/frame` 与 `go vet` 均未因本切片变化（上一次全绿状态见 round254）。

## 5. 待办（下一轮，45–60 分钟）

1. native 点「收起侧边栏」（240,22）进入 rail 态 → UIA 抓 rail 宽度、pill rect（预期仍是 16/16.67/168.67）、图标尺寸、与展开按钮的相对位置；随后复原展开态。
2. 按实测把 `collapsedRail` 的新建入口改成原生胶囊（宽度/圆角/字号/图标），并核对 rail 宽度常量是否也要从当前值改到原生值。
3. 补测试：rail 态 pill rect、点击创建会话（`OnCreate` 调用一次）、展开态 header 仍为 (14,70,252,38)。
4. temp light 构建 + UIA 验证 + 截图。

## 6. 清理

本轮只做只读 UIA 与一次 Invoke（未产生新会话，行列表前后一致）；截图存于 `evidence/round259/`；原生窗口状态未变（未移动、未关闭）。

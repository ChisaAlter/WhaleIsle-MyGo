# MyGo鲸屿 ↔ 鲸屿 视觉/交互对齐最终报告

日期：2026-10-22  
参考：Electron 鲸屿 0.3.9 安装版（`C:\软件\Whale Isle`）及其 `app.asar` 内的 CSS/Radix 实现。  
范围：布局、UI 样式、动画特效、交互体验。共 30 轮，25 个提交（`5291a3cf`…`ffa4c4f0`）。

## 已对齐并验证

| ID | 差异 | 处理 | 证据 |
| --- | --- | --- | --- |
| A1 | 侧栏品牌副标题 MYGO鲸屿 | 改为 WHALE ISLE | `final-hero.png` |
| A2 | 新建会话按钮纯文本 “+ 新会话” | 圆圈加号 + 居中文字 | `final-hero.png` |
| B1 | 底部两行“设置/账号与登录” | 合并为单行“设置与账号”，菜单保留账号/退出 | `final-hero.png` |
| B2 | 背景过亮过饱和（204/232/238，sat .173） | dune 默认背景 + 固定调色板 + 22% 遮罩 → 202/212/217，sat .086 | 像素统计 |
| B3 | 侧栏宽度观感偏窄 | 实测 280.7 DIP，与参考 280 一致 | 像素测量 |
| B4 | 设置导航蓝色焦点圈、行距 4 | 去掉 FocusRing，行距 8（46 节距） | `settings-general.png` |
| B8 | 代码块“文字按钮”旧结论 | 实际已是图标 + 复制对勾，文档过时 | 代码核对 |
| B9 | 发送/停止 tooltip “发送/停止” | “发送消息 · Enter”“停止生成  Esc Esc” | 代码/测试 |
| B10 | Tooltip 600/400ms | 700/300ms（参考 Radix `delayDuration=700`、`skipDelayDuration=300`） | asar 提取 |
| B11 | 禁用透明度 .5 | .48（参考 CSS） | 测试 |
| B12 | 禁用控件指针默认箭头 | 命中禁用可点击元素显示 not-allowed，新增单测 | `TestDisabledCursorIsNotAllowed` |
| B13 | 折叠栏账号头像 + 24px logo | 只留齿轮 + 状态点；logo 38 | `collapsed-pass93.png` |
| B14 | 深色对比度未核 | FG 5.7、Muted 8.5、Caption 4.9，均 ≥4.5 | 计算审计 |
| B15 | Composer 补全弹层在卡内长高 | 实际已绝对定位浮在卡外，新增回归测试 | `TestSuggestPopupFloatsOutsideCard` |
| B16 | 外观页未对照 | 色彩模式/主题卡/选中态一致 | `settings-appearance.png` |
| B17 | 界面设置无说明行 | 9 行补两行说明；“发送消息时的思考炫光”文案对齐 | `settings-interface.png` |
| B18 | 关于页产品名 MyGo鲸屿 | 产品名/窗口标题统一 Whale Isle | `settings-about.png` |
| B20/B21 | 旧审查称工作区行无动画、圆角 8 | 实际 hover 文件夹→箭头、300ms ease-out、行圆角 12 | 测试 `row_hover_test.go` |

## 已实现但交互待实机验证

**B5 原生 HMENU → 应用内 overlay**

- `showMenu` 在真实 host 不再调用 `host.popupMenu`，改在根 overlay 绘制；
- 手动 DIP 定位：按钮菜单向上、上下文菜单在指针处、越界翻转；
- 面板 12 圆角、0.5 发丝边、阴影、34 高行、hover/键盘高亮；
- 已补面板聚焦、↑/↓、Enter、Esc 代码；
- headless 测试保留原生模拟，MyGo 菜单测试全绿。

证据：`menu-overlay.png`、`menu-overlay-keyboard.png`。  
已验证：UIA Invoke 菜单项“设置”成功打开设置页；键盘 Enter 成功打开设置页。修复了 overlay 在选择当帧被 `endPass` 清空 `menu.chosen` 的问题（新增 `chosenHold`）。  
仍未做：子菜单只画 `›`，未展开；真实鼠标坐标注入不稳定，但点击路径已由 accessibility action 覆盖。

## 有意差异

- **B19 模型页信息架构**：当前按“已配置供应商/刷新/识图/探测/当前任务模型”组织，参考是单供应商密钥卡。属于产品结构差异，没有按截图强行重排。截图：`settings-models.png`。
- **关于页版本号**：显示 `dev`，不伪造成参考的 `0.3.9`。
- **关于页诊断行**：显示本机目录读取失败等真实状态，不用文案掩盖。

## 验证命令

```sh
go test ./internal/ui/... ./internal/store/... ./internal/desktop/whale ./cmd/whaleisle
# 通过

cd third_party/mygo && go test ./ui -run 'TestMenu|TestContextMenu'
# 通过

cd third_party/mygo && go test ./ui
# 仍有已知无关失败：
# TestShapeTextLineNativeFallbackAndGraphemeEllipsis
# text_layout_raster_test.go:147 width 24 did not use the native one-line ellipsis
```

## 证据目录

`docs/review-20261022/evidence/`：

- `final-hero.png`
- `settings-general.png`
- `settings-appearance.png`
- `settings-interface.png`
- `settings-models.png`
- `settings-about.png`
- `menu-overlay.png`
- `menu-overlay-keyboard.png`

## 结论

布局、颜色、字体层级、设置子页、折叠栏、tooltip/禁用态等主要视觉差异已收敛并有实机截图或像素数据支撑。  
剩余风险集中在“菜单交互的真实输入验证”和“弹层退出动画”，两者都不是配色/间距层面的差异，已单独记录。

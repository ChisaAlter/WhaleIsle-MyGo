# MyGo鲸屿 ↔ 鲸屿 对齐清单（2026-10-22）

参考：`docs/review-20261020/evidence/compare/ref-hero-1440.png`、`ref-settings-*.png`、`ref-plugins-1440.png`、`ref-schedule-1440.png`、`ref-account-menu-1440.png`。
当前实机：`.devdata/hero-pass75c.png`（2160×1380，150% DPI）。

## 已对齐

| ID | 差异 | 处理 |
| --- | --- | --- |
| A1 | 侧栏品牌副标题是“MYGO鲸屿”，参考是“WHALE ISLE” | 改为 WHALE ISLE（中英文都改） |
| A2 | 新建会话按钮是文字“+ 新会话”，参考是圆圈加号 + 居中文字 | 改为 18 DIP 圆圈描边 + 10 DIP 加号 + 8 间距的居中组 |
| A3 | 权限/模型/设置下拉逐项边框、蓝底选中 | 已统一 MenuRow / Select 样式（前序提交） |
| A4 | 设置面板纯白 255 | 收到 238-240（参考 235-240） |

## 待对齐

| ID | 差异 | 计划 |
| --- | --- | --- |
| B1 | 侧栏底部是两行“设置”“账号与登录”，参考是一行“设置与账号” | **已对齐**：合并为单行“设置与账号”，菜单保留设置/联系/账号/退出 |
| B2 | 侧栏和工作区背景饱和度偏高（青绿更亮），参考更灰更柔 | **已对齐**：新配置默认 dune 背景 + 固定 dune 调色板 + 18-22% 遮罩；右下区域均值 202/212/217（参考 191/205/212），饱和度 0.086（参考 0.115） |
| B3 | 侧栏宽度观感比参考窄 | **已核对**：分隔线在 421px/1.5 = 280.7 DIP，与参考 280 一致，无需改 |
| B4 | 设置导航选中胶囊出现蓝色外圈 | **已对齐**：导航行关闭默认 FocusRing，保留参考的浅灰选中胶囊；行高 38/间距 8（参考 46 节距）/圆角 12 |
| B5 | 账号/会话/项目/插件/右键仍是原生 HMENU | 受 MyGo overlay 渲染阻塞，需框架层修复 |
| B6 | 窗口偶发以 0×0 隐藏启动，需 ShowWindow 才显示 | **未复现**：启动日志 hidden=false，窗口 80 秒内正常 200,0,2160,1380；判断为上一实例未退干净/启动慢，非代码隐藏 |
| B15 | 旧审查说 Composer 补全弹层在卡片内长高 | **已核对**：当前 `paintSuggestPopup` 已用绝对定位浮在卡片外，宽度 ≤ 卡片；新增 `TestSuggestPopupFloatsOutsideCard` 回归。旧文档过时 |\n| B14 | 深色主题对比度未核对 | **已核算**：deepseek.dark 在 `#151517` 上 FG 5.7、Muted 8.5、Caption 4.9，均 ≥4.5；直接改 config 不会生效（Host 外观覆盖），实机深色截图需从设置页切换 |\n| B13 | 折叠栏底部同时有账号头像和设置齿轮，顶栏 logo 24 | **已对齐**：折叠栏只保留设置齿轮+状态点（参考 `ref-sidebar-collapsed-1440.png`）；品牌 mark 放大到 38 |\n| B12 | 禁用控件指针仍是默认箭头 | **已对齐**：参考 CSS `cursor:not-allowed`；MyGo 命中禁用可点击元素时显示 not-allowed |\n| B11 | 禁用控件透明度 0.5 | **已对齐**：参考 CSS `.primary-btn:disabled,.ghost-btn:disabled{opacity:.48}`；MyGo 全局禁用透明度改为 0.48 |\n| B10 | Tooltip 延迟 600ms 冷/400ms 热 | **已对齐**：参考 asar 的 Radix `TooltipProvider` 为 `delayDuration=700`、`skipDelayDuration=300`；MyGo 改为 700/300ms |\n| B8 | 旧审查说代码块工具条是文字按钮 | **已核对**：当前已是 16px 图标（换行/复制）+ 复制后对勾 1s，文案本地化；旧文档过时 |\n| B9 | 发送/停止 tooltip 为“发送”“停止” | **已对齐**：改为“发送消息 · Enter”“停止生成  Esc Esc”，中英文同步 |\n| B7 | 动效未逐项对齐 | **部分对齐**：侧栏/分栏 300ms cubic-bezier(.4,0,.2,1) 已有；设置弹层 200ms；会话进入 300ms；本轮给 Composer 权限/模型/上下文/`@` 菜单统一加 150ms 淡入。菜单行 hover/pressed 填充改为 150ms 颜色过渡；退出动画尝试 Transition Enter/Exit 会让 `@` 文件补全的面包屑点击测试失败，已回退；本轮再试仅 Exit 同样失败。侧栏导航和设置导航的 hover 填充已改为 150ms 颜色过渡 |

## 验证方式

- 每次修改后：`go build` + 相关包测试 + PrintWindow 实机截图。
- 像素/几何差异记录在 `docs/review-20261020/visual-acceptance.md`。

# 验收证据索引（2026-10-25 · 刷新版）

合同：[docs/acceptance-ui-parity.md](acceptance-ui-parity.md)（唯一验收合同，266 行，2026-10-18 起未改）。
本索引只做「把每一行接到已有证据 / 指出还缺什么」，**不代替用户复核，也不填任何签名**：合同所有 `完成` / `审查完成` 列仍空白，只有用户本人（或用户指定的另一人）可以签。

> 本轮变化（相对 02:47 版）：**D-MAIN 有了第一套完整验收工件**（`evidence/accept-D-MAIN/`：baseline / actual / diff / diff-matched / geom / controls / mask / structure / fixture / accept-log），L-HOME 有了 MyGo 半边抓图（native 半边待决策），远程面板有了 live/QR 实机抓图（headless 亦出图），设置页有了逐 tab DOM 审计，S01/S02/S05 已修、S04 属测量错误、S03 已实现但仅 Windows。签名状态不变：**0 行可签**。

> 时效：证据以 `docs/review-20261022/evidence/` 与 round159–176 文档为准；`accept-D-MAIN/accept-log.txt` 记录 native `0.3.10`、MyGo `pass283 / 3e8fd30b`。工作树仍有并行改动，签任何一行前以届时 `git rev-parse HEAD` 与最新证据为准。

---

## 0. 计数（诚实口径）

| 口径 | 行数 |
| --- | --- |
| 合同可变行合计（E7 + L9 + D14 + P8 + S12 + FL8 + FD18 + FS14 + G6） | **96** |
| **证据就绪（合同六件套齐全，可开始逐项审查）** | **1**（D-MAIN） |
| 其中严格标准（masked 0 且结构 1 DIP 全过）已满足 | **0** |
| 部分证据（有抓图/审计/测试，缺合同工件） | **59** |
| 无证据 | **36** |
| 组汇总行（§2.5 五组 / §3.4 三组 / §6） | 9（无可签内容） |
| 20261025 矩阵遗留 open（本轮复核后） | **20**（S01/S02/S04/S05 已结，S03 仅 Windows） |

分组：

| 组 | 行数 | 证据就绪 | 部分 | 无 |
| --- | --- | --- | --- | --- |
| E-* 环境 | 7 | 0 | 7 | 0 |
| L-* 启动器 | 9 | 0 | 1 | 8 |
| D-* 桌面主壳 | 14 | 1 | 9 | 4 |
| P-* 功能面板 | 8 | 0 | 4 | 4 |
| S-* 设置 | 12 | 0 | 8 | 4 |
| FL-* 启动器功能 | 8 | 0 | 8 | 0 |
| FD-* 桌面功能 | 18 | 0 | 13 | 5 |
| FS-* 设置功能 | 14 | 0 | 9 | 5 |
| G1–G6 门禁 | 6 | 0 | 0 | 6 |

---

## 1. 必须由用户决定的事项（D1–D4）

这四项不决定，D-MAIN 也无法从「证据就绪」走到「可签」。

### D1 验收判据：实用判据 vs 严格 0

`round175` 在 theme-matched 的 D-MAIN 上实测：threshold 0 下失配 **2,786,143 / 2,786,400（99.99%）**，其中绝大部分是壁纸与数据状态，不是 chrome；分区均值 |Δ| 37–148。严格 0 在两套渲染器 + 渐变壁纸下**按构造不可达**（1/255 的差就算失配）。

| 选项 | 内容 | 代价/风险 |
| --- | --- | --- |
| **A. 实用判据** | ① mask 外失配 = 0（要求登记壁纸/渐变）；② 结构 1 DIP 容差（caption-relative，先扣双方 caption/hero 原点）；③ 每区 >阈值（如 >8）像素占比 ≤0.1%；④ 文本抗锯齿单独计（1 DIP 膨胀外统计） | 需用户在合同 §0.1 改措辞，并把壁纸列为受控 fixture |
| **B. 严格 0** | 维持 0 通道差 | 需双方同壁纸、同数据状态、同字体栅格；当前无一台机器可复现，等于无限期不可交付 |

### D2 L-HOME 的 native 半边：是否允许冷启动 + 临时 user-data

`round175 §5`：native 启动器**只能**通过「no-workspace profile 冷启动」得到 1060×660 窗口；运行中实例二次启动只多两个 `main` 进程、不出窗口。MyGo 半边已抓：`screens/mygo-launcher-pass283-1060x660.png`（1590×990 px @150%）。

| 选项 | 内容 | 影响 |
| --- | --- | --- |
| A. 允许冷启动 | 用临时 `user-data`（不动用户数据）重启 native 抓启动器 | L-HOME 可补齐；需用户同意重启自己的 app |
| B. 不重启 | L-HOME 永远只有 MyGo 半边 | L-HOME 不能签，L 组无法成立 |

### D3 壁纸策略（决定 D-MAIN/S-* 是否可能达标）

| 选项 | 内容 | 评价 |
| --- | --- | --- |
| A. 让 MyGo 用同一张壁纸 | 读取 native 壁纸文件，写入 MyGo 壁纸存储 | 最诚实；需新增读写路径（产品代码，本片不做） |
| B. 把壁纸登记为蒙版 | 只评 opaque chrome（侧栏面、caption 带、composer 卡） | 便宜，符合合同「蒙版登记」设计；需用户接受「D-MAIN 定义 = 蒙版外」 |
| C. 渲染 Windows 桌面壁纸 | 直接截桌面壁纸当两侧底图 | 不可控（动态壁纸/多屏），不建议 |

### D4 harness-Web CDP 基线是否可接受

合同 §0.1 要求「与正在运行的 Electron 本体逐屏对照」。现状：D-* / S-* 的**内部内容**基线来自 harness Web UI 的 CDP 抓图（round152/155/164/168），壳窗部分来自实机（round175）。两者不是同一基线来源。

| 选项 | 内容 |
| --- | --- |
| A. 分层基线 | 壳窗用实机、页面内容用同源 harness-Web CDP；在合同里写明分层 |
| B. 全部实机 | 需每次启动 Electron + 自动导航到每个状态（round164 已证部分 tab 不可自动化，如账户 tab 未登录不存在） |

---

## 2. 逐行索引

路径前缀：`docs/review-20261022/evidence/`（下称 `evidence/`），截图在 `evidence/screens/`。**证据就绪**=合同六件套齐全；**部分**=有抓图/审计/测试；**无**=没有可打开的证据。

### 2.1 环境 E-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| E1 | 部分 | `accept-D-MAIN/accept-log.txt`（native `Whale Isle.exe` 0.3.10 路径+版本已写入）；`screens/native-harness-chat-1440x920.png` | 把 E1 提升为环境级记录（现在只在 D-MAIN log 里） | slice-verify | 打开 `accept-log.txt` 看 native 行 |
| E2 | 部分 | `accept-log.txt` 记录 MyGo `pass283 git=3e8fd30b`；`round176-build.json`、`harness-start-pass284.json` | 绑定「交付 commit」的单一二进制记录（现在是 passN） | slice-verify | 看 log 的 mygo 行 + `round176-build.json` |
| E3 | 部分 | `accept-D-MAIN/geom.json`：client 1440×860 DIP，native 2160×1290 px / mygo 2160×1291 px，dpi 144 | 客户区宽高一致（现差 1 px，合同允许 ≤1）→ 仍需写入 E3 并双签 | slice-verify | 打开 `geom.json` 的 `client` 段 |
| E4 | 部分 | `geom.json` dpi 144；`round138/149` 换算 | 同监视器/同缩放声明（log 已含 dpi，缺显式声明） | slice-verify | 看 `geom.json` 的 dpi |
| E5 | 部分 | `accept-D-MAIN/fixture.md`（1440×860、zh-CN、dark、单空会话、终端关） | fixture 需覆盖 L-*/P-*/S-* 各行的数据状态；现只有 D-MAIN | slice-verify | 打开 `fixture.md` |
| E6 | 部分 | log 记录 `%TEMP%\whaleisle-accept-2026-10-24\D-MAIN`，并镜像到仓库 | 环境级 `%TEMP%\whaleisle-accept-<日期>\accept-log.txt`（不是逐行子目录） | slice-verify | 打开 `accept-log.txt` 末行 `out=` |
| E7 | 部分 | `mask.json`：本稳态无蒙版区域 | 按 D1/D3 决定壁纸是否入蒙版；蒙版格式与审批人 | slice-verify | 打开 `mask.json` |

### 2.2 启动器画面 L-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| L-HOME | 部分 | **MyGo 半边**：`screens/mygo-launcher-pass283-1060x660.png`（1060×660 DIP @150%，深色，rail/首页卡/版本管理/启动桌面端）；旧：`screens/launcher-pass275.png`、`screens/reference-launcher-1060x660.png` | **native 半边（D2）+ 三图 + geom/controls** | slice-launcher | 打开 `screens/mygo-launcher-pass283-1060x660.png` 与 `screens/reference-launcher-1060x660.png` |
| L-HOME-DIAG | 无 | 无 | 诊断展开抓图（native 侧同样受 D2 约束） | slice-launcher | — |
| L-COMP | 无 | 无 | 组件页 | slice-launcher | — |
| L-VER | 无 | 无 | 版本页 | slice-launcher | — |
| L-IMP | 无 | 无 | 导入空态 | slice-import | — |
| L-IMP-HIT | 无 | 无 | 导入有结果 | slice-import | — |
| L-PLUG | 部分 | S05 已修（round166：`startuprec/forensics.go` 四条原生正则 + 测试）；`round139` 插件排查 | 插件排查页抓图 + 禁用/启用/重启用例 | slice-launcher | 读 `round166` S05 段与 `internal/startuprec/forensics.go` |
| L-SET | 无 | 无 | 启动器设置 tab | slice-launcher | — |
| L-ROUTE | 无 | 无 | 线路菜单 | slice-shell | — |

### 2.3 桌面主壳 D-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| D-MAIN | **证据就绪**（不可签） | `accept-D-MAIN/`：baseline/actual/diff/diff-matched/geom/controls/mask/structure/fixture/accept-log；`screens/mygo-chat-after-round156-150dpi.png`、`screens/native-harness-chat-1440x920.png` | ① D1 判据 + D3 壁纸；② 结构 0/17 within 1 DIP（round176 计划 caption-relative）；③ 数据态（5 工作区/会话树/登录账号）按 fixture 对齐 | slice-chat | 打开 `accept-D-MAIN/diff.png` 与 `structure.json`，再读 `round175` 数字表 |
| D-MAIN-SEL | 部分 | `sidebar-*.png`（8 张）、`screens/mygo-chat-pass275-150dpi.png` | 已选会话态三图；数据态需与 native 同 fixture | slice-chat | 打开 `sidebar-workspace-groups.png` |
| D-TERM | 部分 | `terminal-*.png`、`terminal-pass259-*checks.json` | 终端开态整屏三图 | slice-terminal | 打开 `terminal-friendly-tab.png` |
| D-PLAN-ON | 无 | 无 | 计划面板展开 | slice-chat | — |
| D-PLAN-OFF | 无 | 无 | 计划面板收起 | slice-chat | — |
| D-FILES | 部分 | `preview-pass252-native-and-filesystem.txt`、`tool-image-pass266-preview-authority.json` | 文件 tab+树+预览整屏三图 | slice-workspace | 看 `preview-pass252-native-and-filesystem.txt` |
| D-DIFF | 部分 | `tool-diff-pass263-native-metrics.json`、`-geometry-comparison.json` | 工作区 Diff 空态整屏 | slice-workspace | 打开 `-geometry-comparison.json` |
| D-DIFF-HIT | 部分 | `tool-diff-pass263-native-preset-geometry.json` | 有内容态整屏 | slice-workspace | 同上 |
| D-GIT-STRIP | 部分 | `git-caption-review-pass242.png`、`git-action-disabled.png` | 提交条开态整屏 | slice-git | 打开 `git-caption-review-pass242.png` |
| D-GIT-PR | 部分 | `git-action-noring.png` | PR 面板开态 | slice-git | 打开 `git-action-noring.png` |
| D-INTERACT | 部分 | `permission-dialog-checked.png`、`-unchecked.png` | 审批可见整屏 | slice-chat | 打开 `permission-dialog-checked.png` |
| D-LIGHTBOX | 部分 | `tool-image-pass266-…-local-lightbox.png` | 灯箱整屏 + 关闭态 | slice-chat | 打开该 png |
| D-BOOT | 无 | 无（`remote-app-boot.json` 不是 boot 页） | boot 启动中抓图 | slice-launcher | — |
| D-BOOT-FAIL | 无 | 无 | boot 失败态（需失败注入） | slice-launcher | — |

### 2.4 功能面板 P-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| P-BROWSER | 部分 | `preview-pass252-*`（15）、`inspector-pass255-reference-picker-browser*.png`；**S04 复核：缩放档位与 `preview-zoom.js:4-6` 完全一致（矩阵行过时）** | 预览整屏三图 + PiP/缩放/注解态 | slice-preview | 打开 `preview-pass252-full-toolkit.txt` |
| P-ASSIST | 无 | 无（`companion-*` 是桌宠） | 助理面板整屏 | slice-assistant | — |
| P-WEBVIEW | 无 | 无 | 插件 Web 面板整屏 | slice-assistant | — |
| P-REMOTE | 部分 | **live**：`screens/remote-panel-live.png`、`remote-panel-qr-final.png`、`remote-panel-qr-lan.png`、`remote-panel-mounted.png`、`remote-panel-block-headless.png` + 同名 JSON（pass282，lan/3499/127.0.0.1，headless QR 已出图） | 整屏三图；G44（mobile runtime）/G45（LAN 发现/TLS）仍需真实设备 | slice-remote | 打开 `screens/remote-panel-qr-final.png` 与 `remote-panel-live.json` |
| P-MARKET | 部分 | `market-*.png`（13）、`plugins-market-choice-1440.png`；S02 已修（`market/space.go`） | 整屏 + 安装/详情/历史态 | slice-market | 打开 `market-details-wide.png` |
| P-SCHED | 部分 | `automation-nav-noring.png`、`automation-new-clean.png` | 整屏 + 新建/编辑/删除 | slice-schedule | 打开 `automation-new-clean.png` |
| P-IM | 无 | 无 | 消息面板整屏 | slice-im | — |
| P-USAGE | 无 | 无抓图；仅 `docs/parity/native-usage-stats.md` 等文档 | 用量整屏（7 日/28 日/筛选） | slice-usage | — |

### 2.5 设置画面 S-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| S-DESK | 无 | 无 `settings-desktop*` | 桌面页整屏 | slice-settings | — |
| S-ACCT | 无 | 仅 `sidebar-account-seat-signedout.png`；round164：账户 tab 未登录态不存在 | 账号页（需登录 fixture；先决定是否允许登录态） | slice-account | — |
| S-CRED | 无 | 无 | 凭据页整屏 | slice-settings | — |
| S-APPEAR | 部分 | `settings-appearance.png`、`-bottom.png`、`settings-font-*.png`（5）、`settings-theme-*.png`、`settings-wallpaper-sources.png`；round164 原生 DOM（模式卡 183×84/r16、主题卡 137×88、行/列距） | 三图 + 与 round164 数字的逐项对照表 | slice-settings | 打开 `settings-appearance.png` + `round164 §2` |
| S-UI | 部分 | `settings-interface.png`、`-gear.png`、`-single-save.png`；round164 行高 77/95、switch 44×28 | 三图 + 逐项对照 | slice-settings | 打开 `settings-interface.png` |
| S-MODEL | 部分 | `settings-models.png`、`-compact.png`；round164 模型 tab DOM | 三图 + 空/有模型态 | slice-settings | 打开 `settings-models.png` |
| S-PLUG | 部分 | `settings-builtin-loading.png`；round168 内置插件数字（标题 18px、搜索 564×36、卡 275 宽/两列/列距 12、禁用徽标） | 三图 + 逐项对照（中英） | slice-settings | 打开 `settings-builtin-loading.png` + `round168 §2` |
| S-SKILL | 部分 | `settings-skills-browse.png`、`-empty.png`；round168（"0 workspace and personal skills" 语序、空态 14px） | 三图 + 逐项 | slice-settings | 打开 `settings-skills-empty.png` |
| S-MCP | 部分 | `settings-mcp-compact.png`；round168 MCP tab DOM | 三图 + 健康态 | slice-settings | 打开 `settings-mcp-compact.png` |
| S-AGENT | 部分 | `settings-agent-presets-grid.png`、`-tight.png`；round168 Agent 预设 tab | 三图 + 指南弹窗 | slice-settings | 打开 `settings-agent-presets-grid.png` |
| S-ABOUT | 部分 | `settings-about.png`、`-clean/-brand/-diagnostics.png` | 三图 + 诊断展开态 | slice-settings | 打开 `settings-about.png` |
| S-OTHER-<ns> | 无 | 无 | 先枚举 ns（契约空白 A2） | slice-settings | — |

### 2.6 启动器功能 FL-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| FL-RAIL | 部分 | `round139/151`、`screens/launcher-task25-icons-150dpi.png` | `cases-FL-RAIL.md` + 执行日志 | slice-launcher | 打开该 png 对照 6 图标 |
| FL-HOME-CTA | 部分 | `round147/151`、`launcher-after-homecard-1060x660.png` | 用例清单 + 每 CTA 结果屏 | slice-launcher | 打开 `launcher-after-homecard-1060x660.png` |
| FL-HOME-DIAG | 部分 | `docs/status.md` 启动器段 | 用例清单 + 展开/折叠日志 | slice-launcher | 读 status 启动器段 |
| FL-VER | 部分 | `round126` 更新进度行 | 检查/下载/安装三态日志 | slice-shell | — |
| FL-IMP | 部分 | `internal/desktop/importer` 测试 | 空/有结果/失败三态日志 | slice-import | — |
| FL-PLUG | 部分 | S05 已修（round166）：OOM/EACCES/EADDRINUSE/ENOENT-node 分类与原生一致 | 用例清单 + 注入崩溃日志 | slice-launcher | 读 round166 S05 |
| FL-SET-TAB | 部分 | round139 | 用例清单 | slice-launcher | — |
| FL-ROUTE | 部分 | `internal/desktop/deeplink`、round134 | 全部线路用例 | slice-shell | — |

### 2.7 桌面功能 FD-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| FD-TITLE | 部分 | `settings-shortcuts-dialog-1440.png`、`menu-overlay.png`、round134；S03：Ctrl+Shift+I 已实现（round171，Windows 可用；linux/darwin 引擎不暴露按键） | 菜单全项集合 + 快捷键集合全等清单 | slice-shell | 打开 `menu-overlay.png` + round171 |
| FD-TERM | 部分 | `terminal-*.png`、`terminal-pass259-*checks.json` | 终端内全部操作用例 | slice-terminal | 打开 `terminal-pass259-affected-checks.json` |
| FD-SESS | 部分 | `sidebar-*.png`、round 文档 | 搜索/新建/刷新/置顶/重命名/归档/删除用例 | slice-chat | 打开 `sidebar-archived-toggle.png` |
| FD-CHAT | 部分 | `conversation-*.png`、`final-hero.png` | 更早/反馈/编辑/引用/灯箱用例 | slice-chat | 打开 `final-hero.png` |
| FD-PLAN | 无 | 无 | 计划面板全部控件 | slice-chat | — |
| FD-INTERACT | 部分 | `permission-dialog-*.png`、`settings-permission-select.png` | 审批全部路径 | slice-chat | 打开 `permission-dialog-checked.png` |
| FD-COMPOSE | 部分 | `composer-*.png`、round156（composer 114 / 252×38） | 全选项 + 排队/技能/目录/发送停止/chip/拖放 | slice-chat | 打开 `composer-sent-pass243.png` |
| FD-FILES | 部分 | `preview-pass252-native-and-filesystem.txt` | 树/预览/编辑/外部打开/PDF/Diff 用例 | slice-workspace | 打开该 txt |
| FD-DIFF | 部分 | `tool-diff-pass263-*` | 工作区 Diff 全部操作 | slice-workspace | 打开 `-native-metrics.json` |
| FD-GIT | 部分 | `git-*.png` | fetch/pull/push/commit/PR 用例 | slice-git | 打开 `git-caption-review-pass242.png` |
| FD-PANEL-BROWSER | 部分 | 同 P-BROWSER | 面板 controls 全量 | slice-preview | 同 P-BROWSER |
| FD-PANEL-ASSIST | 无 | 无 | 面板 controls 全量 | slice-assistant | — |
| FD-PANEL-WEBVIEW | 无 | 无 | 面板 controls 全量 | slice-assistant | — |
| FD-PANEL-REMOTE | 部分 | 同 P-REMOTE（live + QR） | controls 全量 + G44/G45 | slice-remote | 同 P-REMOTE |
| FD-PANEL-MARKET | 部分 | 同 P-MARKET；S02 已修（`market/space.go` 256 MiB 预检） | controls 全量 + 空间不足态 | slice-market | 同 P-MARKET |
| FD-PANEL-SCHED | 部分 | 同 P-SCHED | controls 全量 | slice-schedule | 同 P-SCHED |
| FD-PANEL-IM | 无 | 无 | controls 全量 | slice-im | — |
| FD-PANEL-USAGE | 无 | 无 | controls 全量 | slice-usage | — |

### 2.8 设置功能 FS-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| FS-NAV | 部分 | `settings-general.png`、`settings-market-rail.png`、round144/164（导航 164×40/r12/步进 44） | 导航+过滤+全部 OTHER ns 用例 | slice-settings | 打开 `settings-general.png` |
| FS-PAGE-DESK | 无 | 无 | 页面用例 + 整屏 | slice-settings | — |
| FS-PAGE-ACCT | 无 | 无（未登录无此 tab） | 页面用例（需登录 fixture） | slice-account | — |
| FS-PAGE-CRED | 无 | 无 | 用例（掩码/显示/保存） | slice-settings | — |
| FS-PAGE-APPEAR | 部分 | `settings-appearance.png`、字体/主题/壁纸抓图 + round164 DOM | 全控件用例 | slice-settings | 打开 `settings-font-preview.png` |
| FS-PAGE-UI | 部分 | `settings-interface.png` + round164 行高 | 全控件用例 | slice-settings | 打开 `settings-interface.png` |
| FS-PAGE-MODEL | 部分 | `settings-models.png` | 全控件用例 | slice-settings | 打开 `settings-models.png` |
| FS-PAGE-PLUG | 部分 | `settings-builtin-loading.png` + round168 数字 | 全控件用例（中英） | slice-settings | 打开该 png |
| FS-PAGE-SKILL | 部分 | `settings-skills-*.png` + round168 数字 | 全控件用例 | slice-settings | 打开 `settings-skills-browse.png` |
| FS-PAGE-MCP | 部分 | `settings-mcp-compact.png` | 全控件用例 | slice-settings | 打开该 png |
| FS-PAGE-AGENT | 部分 | `settings-agent-presets-grid.png` | 全控件用例 | slice-settings | 打开该 png |
| FS-PAGE-ABOUT | 部分 | `settings-about*.png` | 全控件用例 | slice-settings | 打开 `settings-about.png` |
| FS-PAGE-OTHER-<ns> | 无 | 无 | 每 ns 一页用例 | slice-settings | — |
| FS-BACK | 无 | 无 | 返回后 D-MAIN 1:1（依赖 D-MAIN 可签） | slice-chat | — |

### 2.9 交付门禁 G1–G6

| ID | 现状 | 还差什么 | owner |
| --- | --- | --- | --- |
| G1 | 无 | 每一画面行双签（当前 0） | lead |
| G2 | 无 | 每一功能行双签 + 用例总数=通过数>0（当前 0 份 cases） | lead |
| G3 | 无 | E1–E7 双签（当前 0） | slice-verify |
| G4 | 无 | 证据绑定同一交付 commit（现为 pass283/284） | lead |
| G5 | 无 | 无抽检声明逐行写 | lead |
| G6 | 无 | 历史错误验收未混入说明 | lead |

---

## 3. 合同硬门槛（除 D-MAIN 外的行共用）

要标「完成=是」，每一行至少还要下面六件；除 D-MAIN 外全仓缺 4–6 件：

1. `baseline`（运行中 Electron 本体）+ `actual`（MyGo）+ `diff`（高亮）三图，同客户区、同 DPI；
2. `geom.json`：每个可见区块外接矩形与本体误差 ≤1 px（现仅 D-MAIN 一份）；
3. `controls.json`：该屏全部控件枚举（现仅 D-MAIN 一份）；
4. `cases-<ID>.md`：控件 × 状态全量用例（**全仓 0 份**）；
5. `accept-log.txt`（现仅 D-MAIN 一份，且不在环境级目录）；
6. 双签（**全空**）。

---

## 4. 剩余缺口 → 验收行（20261025 矩阵复核后 20 条 open）

| 缺口 | 严重度 | 现状（本轮复核） | 阻塞行 | 下一 slice |
| --- | --- | --- | --- | --- |
| G44 手机远程 mobile runtime | high | open（round157/158：LAN 配对已通、headless QR 出图；mobile runtime 仍缺） | P-REMOTE、FD-PANEL-REMOTE | slice-remote（真机） |
| G45 LAN 发现 / TLS-SSH | high | open | 同上 | slice-remote |
| G46 remote shell / git dispatch | medium | open | 同上 + FD-GIT | slice-remote |
| G39 通知呈现 | medium | open（合同无行） | — | slice-shell |
| G31b 桌宠渲染层 | medium | open（合同无行） | — | slice-pet |
| G32 桌宠行为族 | medium | open（合同无行） | — | slice-pet |
| G33 桌宠快捷聊天窗 | low | open（合同无行） | — | slice-pet |
| G34 手机 PR 复核窗 | low | open | P-REMOTE | slice-remote |
| G27 设置专页深度 | medium | open（round164/168 只补 DOM 审计与文案；DESK/ACCT/CRED 仍无页面证据） | S-DESK/ACCT/CRED/OTHER、FS-* | slice-settings |
| G28 字体/滚动条/选择复制/tooltip | medium | open（D-MAIN 结构 0/17 已量化） | 所有像素行 | slice-chrome |
| G09 工作区面板 | medium | open | D-FILES/D-DIFF/FD-FILES/FD-TERM | slice-workspace |
| G52 i18n（cmd 254 处 inline） | medium | open（round164/168 修了 4+2 处文案；inline 未扫） | D-MAIN、FD-TITLE/COMPOSE、S-UI | slice-i18n |
| G21 Inspect/trajectory | low | open | D-MAIN、FD-CHAT | slice-chat |
| G10 标题栏控件逐图标 | low | open | D-MAIN、FD-TITLE | slice-chrome |
| G03 菜单栏可见性 | low | open（headless 不可断言） | D-MAIN、FD-TITLE | slice-chrome（实机） |
| S27 hero 纵向 seat | low | open | D-MAIN | slice-chat |
| S28 composer 772 vs 774.4 / 导航基线 1.7 | low | open | D-MAIN | slice-chat |
| S30 通知 AUMID | low | open（合同无行） | — | slice-shell |
| S31 验收零双签 | high | open（只能用户签） | G1–G6 | lead |
| S34 project stage 成本/校验口径 | low | open | 影响 E2/E6 可复现 | slice-runtime |
| ~~S01~~ 下载 | medium | **fixed**（round163 `webview/downloads.go` + 16 组测试） | D-MAIN 下载态 | 关闭 |
| ~~S02~~ 磁盘预检 | low | **fixed**（round166 `market/space.go`） | P-MARKET | 关闭 |
| ~~S03~~ DevTools 快捷键 | low | **实现但仅 Windows**（round171；linux/darwin 引擎无按键钩子） | FD-TITLE | 记录平台差异 |
| ~~S04~~ 预览缩放 | low | **测量错误**（round166：档位与原生完全一致） | — | 关闭 |
| ~~S05~~ 插件取证 | low | **fixed**（round166 四条正则 + 测试） | L-PLUG/FL-PLUG | 关闭 |

---

## 5. 契约空白与歧义（A1–A8，仍未解决）

| ID | 歧义 | 影响 |
| --- | --- | --- |
| A1 | 桌宠、系统通知、首启「预览版说明」**在合同里没有行** | 这些表面无法双签；需用户决定补行或声明不在范围 |
| A2 | `S-OTHER-__` / `FS-PAGE-OTHER-__` 的 ns 数量随 profile/插件变化 | §2.4/§3.3 分母不确定 |
| A3 | §2.5 / §3.4 组汇总的「组内行数」「用例总数」在合同里为空 | 组级双签无法计数 |
| A4 | E7 蒙版格式/审批方式未定义（D3 会决定壁纸是否入蒙版） | 阈值判定无法落地 |
| A5 | E6 要求 `%TEMP%\whaleisle-accept-<UTC日期>\`，现行证据在仓库 | 迁移还是改口径？ |
| A6 | D4：分层基线（壳窗实机 + 页面 CDP）是否算「与 Electron 本体对照」 | D-*/S-* 的基线合法性 |
| A7 | G4「绑定本 commit」：passN 每轮变 | 交付二进制/commit 未锁定 |
| A8 | §2.1 L-* 的「分壳备注」格式与核对人未定义 | 组级审查口径 |

---

## 6. Top-10 交付阻塞（含下一 slice）

1. **D1 判据未定**：严格 0 已被 round175 实测证明不可达（99.99% 失配中绝大部分是壁纸/数据）。下一 slice：用户决策，lead 记入合同。
2. **D3 壁纸未定**：D-MAIN 的 hero/composer/sidebar 都被壁纸透过，均值 Δ 37–65；不 mask 或同图，任何阈值都过不了。下一 slice：slice-chat + 用户决策。
3. **结构 0/17 within 1 DIP**（round175）：caption 起点差 4–6 DIP + 数据态差异；round176 计划改 caption-relative 键。下一 slice：slice-chat（键控方式，不改渲染）。
4. **数据态未对齐**：native 有 5 工作区/会话树/登录 Ayase；MyGo 单工作区/单会话/未登录。fixture 必须引用同一数据快照。下一 slice：slice-chat + slice-verify。
5. **L-HOME native 半边缺**（D2）：需冷启动 + 临时 user-data。下一 slice：slice-launcher。
6. **零 `cases-<ID>.md`**：FL/FD/FS 40 行无法证明用例全量。下一 slice：slice-verify 生成模板（先 FD-COMPOSE 试点）。
7. **S-DESK / S-ACCT / S-CRED / P-ASSIST / P-WEBVIEW / P-IM / P-USAGE / D-PLAN / D-BOOT 无证据**。下一 slice：按 §2 顺序补抓图（S 页优先，占 FS 13 行）。
8. **G44/G45 远程高缺口**：P-REMOTE 有 live/QR 图但仍需真机联调。下一 slice：slice-remote。
9. **字体/滚动条/选择复制（G28）与 composer 残项（S28）**：阈值下必然失配。下一 slice：slice-chrome。
10. **A1/A3 契约空白**：宠物/通知无行、组分母为空，导致「总行数」无法确定。下一 slice：用户澄清，lead 更新合同。

---

## 7. 离线不可验证

- P-REMOTE / FD-PANEL-REMOTE：G44 mobile runtime、G45 LAN 发现/TLS-SSH 需真机与真实网络（QR 已 headless 出图，但配对后链路未验）；
- 通知（G39/S30）：需安装后 AUMID 与通知中心观察；
- S-ACCT：需登录态（round164 证明未登录时该 tab 不存在）；
- G03 菜单栏可见性：headless/UIA 不可断言；
- 像素审计（G28/S28）：需同 DPI 实机与 D1 判据；
- G46 remote shell/git dispatch：需第二台主机；
- 插件真实安装/市场安装：需可回滚环境；
- L-HOME native（D2）：需用户同意冷启动。

---

## 8. 使用建议

1. 先回 §1 D1–D4 四个决策；它们决定 D-MAIN 能否从「证据就绪」变成「可签」。
2. 再按 §6 顺序补证据；D-MAIN 的工件模板（`evidence/accept-D-MAIN/`）可直接复制到其他行。
3. 任何一行由**你本人**打开「审查快速路径」核对后，在合同对应行填 `是 YYYY-MM-DD <署名>`；本索引不代填、合同未改。

# 验收证据索引（2026-10-25）

合同：[docs/acceptance-ui-parity.md](acceptance-ui-parity.md)（唯一验收合同，266 行，2026-10-18 起未改）。
本索引只做「把每一行接到已有证据 / 指出还缺什么」，**不代替用户复核，也不填任何签名**：合同里所有 `完成` / `审查完成` 列仍为空白，只有用户本人（或用户指定的另一人）可以签。

> 一句话结论：**按合同 §0.1/§2/§3 的字面要求，目前没有任何一行达到「可签」**。现有证据是逐轮的原生/MyGo 对照截图、几何 JSON、构建/测试日志，但没有一屏同时具备 `baseline + actual + diff` 三图、`geom.json`、`controls.json`、`cases-<ID>.md` 与 `%TEMP%\whaleisle-accept-*\accept-log.txt`。所以本索引的「现状」只有两档：**部分证据**（有可打开的抓图/指标/测试）与**无证据**。

> 时效：本索引以 2026-10-25 矩阵（HEAD `9290c9b9`）与当时的 evidence 目录为准。索引撰写期间工作树已出现 round160–163 的并行改动（例如 `internal/desktop/webview/downloads.go` 正在关闭 S01、`cmd/whaleisle/phone_diag.go`），这些尚未进入证据目录，因此 S01 等行仍需在提交后重跑索引；签任何一行前请以届时 `git rev-parse HEAD` 与最新证据为准。

---

## 0. 计数（诚实口径）

| 口径 | 行数 |
| --- | --- |
| 合同可变行合计（E 7 + L 9 + D 14 + P 8 + S 12 + FL 8 + FD 18 + FS 14 + G 6） | **96** |
| 其中：满足合同（三图 + geom + controls + cases + log + 双签） | **0** |
| 其中：部分证据（有抓图/指标/测试，缺合同要件） | **57** |
| 其中：无证据 | **39** |
| 另有组汇总行（§2.5 五组 / §3.4 三组 / §6 总签署） | 9（均无可签内容） |
| 20261025 矩阵 open 缺口 | 25（见 §3 映射） |

分组明细：

| 组 | 行数 | 部分证据 | 无证据 | 合同就绪 |
| --- | --- | --- | --- | --- |
| E-* 环境 | 7 | 4 | 3 | 0 |
| L-* 启动器 | 9 | 1 | 8 | 0 |
| D-* 桌面主壳 | 14 | 10 | 4 | 0 |
| P-* 功能面板 | 8 | 4 | 4 | 0 |
| S-* 设置 | 12 | 8 | 4 | 0 |
| FL-* 启动器功能 | 8 | 8 | 0 | 0 |
| FD-* 桌面功能 | 18 | 13 | 5 | 0 |
| FS-* 设置功能 | 14 | 9 | 5 | 0 |
| G1–G6 门禁 | 6 | 0 | 6 | 0 |

> 说明：FL/FD/FS 的「部分证据」指有轮次文档、单测与若干抓图；合同要求的是 `cases-<ID>.md` 全量用例 + 执行日志，这两样全仓 **0 份**。

---

## 1. 逐行索引

列说明：**现状**（部分/无）；**已有证据**（可直接打开的相对路径，均在 `docs/review-20261022/evidence/` 下，`screens/` 为子目录）；**还差什么**（要标 完成=是 的最小缺口）；**owner**（下一 slice）；**审查快速路径**（审查人先看哪一个文件、看什么）。

### 1.1 环境 E-*

| ID | 现状 | 已有证据 | 还差什么才能标「完成=是」 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| E1 | 部分 | `screens/reference-launcher-1060x660.png`、`screens/native-harness-chat-1440x920.png`、`docs/parity/native-usage-stats.md` 的对照方法 | 现场 Electron 路径+版本写入 `accept-log.txt`；本仓库只有源码快照与 CDP 抓图，无「运行中本体」版本行 | slice-verify | 打开 `reference/electron/package.json` 与 `screens/native-harness-chat-1440x920.png`，确认版本与来源；缺 log 行即未就绪 |
| E2 | 部分 | `round159-build.json`、`harness-start-pass279.json`、`round126-build.json` … | 绑定「本交付 commit + 该 commit 现场构建二进制」的单一记录；现在每个 round 一个 build，无交付 commit | slice-verify | 任选 `harness-start-pass279.json` 看 `ready=True`；但它绑的是 pass279 而非交付 commit |
| E3 | 部分 | `docs/review-20261022/evidence/window-geometry-pass273.json`、`…pass274.json`（启动器 1590×990 px / 主窗 2160×1380 px @144 DPI） | 同一时刻两窗客户区 W×H 的同屏记录；现有是 MyGo 单侧 | slice-verify | 打开两个 geometry JSON 看 `client` 宽高是否都为 1440×920 DIP / 1060×660 DIP |
| E4 | 部分 | `round138`、`round149` 文档 + geometry JSON 的 `dpi:144` | 同监视器、同缩放的声明写入 accept-log；无声明文件 | slice-verify | 看 geometry JSON 的 dpi 字段（144/150%） |
| E5 | 无 | 无 `fixture.md`；仅有 `docs/fixtures/browser-file-input.html` 与 `docs/fixtures/fixtures/` | 新建可复现 `fixture.md`（数据/工作区/会话/插件快照清单） | slice-verify | 搜 `fixture.md` → 0 命中即未就绪 |
| E6 | 无 | 证据在仓库 `docs/review-20261022/evidence/`，不是 `%TEMP%\whaleisle-accept-<UTC>` | 按 §0.3 建 `%TEMP%\whaleisle-accept-<日期>\` + `accept-log.txt`（或在合同里改口径，见 §5 A5） | slice-verify | 检查 `%TEMP%` 下是否有该目录 |
| E7 | 无 | 无蒙版清单；全部抓图均未声明蒙版 | 蒙版清单+理由+坐标，或按「无蒙版」填「无」并由审查人确认 | slice-verify | 合同 §0.3 E7 行仍空 |

### 1.2 启动器画面 L-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| L-HOME | 部分 | `screens/launcher-pass275.png`、`screens/mygo-launcher-pass275-150dpi.png`、`screens/reference-launcher-1060x660.png`、`launcher-after-homecard-1060x660.png`、`docs/parity/launcher-boot-plan-20261024.md` | actual/baseline/diff 三图 + geom + controls；现只有单张 MyGo 与单张参考 | slice-launcher | 并排打开 `screens/mygo-launcher-pass275-150dpi.png` 与 `screens/reference-launcher-1060x660.png` 看首页 |
| L-HOME-DIAG | 无 | 无（`settings-about-diagnostics.png` 是设置页诊断，不是启动器） | 诊断展开抓图 + 三图 | slice-launcher | 搜 `launcher.*diag` → 0 命中 |
| L-COMP | 无 | 无 | 组件页抓图 | slice-launcher | — |
| L-VER | 无 | 无 | 版本页抓图 | slice-launcher | — |
| L-IMP | 无 | 无 | 导入空态抓图 | slice-import | — |
| L-IMP-HIT | 无 | 无 | 导入有结果抓图 | slice-import | — |
| L-PLUG | 无 | 无 | 插件排查页抓图 | slice-launcher | — |
| L-SET | 无 | 无 | 启动器设置 tab 抓图 | slice-launcher | — |
| L-ROUTE | 无 | 无 | 线路菜单打开态抓图 | slice-launcher | — |

### 1.3 桌面主壳 D-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| D-MAIN | 部分 | `screens/mygo-chat-pass275-150dpi.png`、`screens/mygo-chat-after-round156-150dpi.png`、`screens/native-harness-chat-1440x920.png`、`native-main-pass245/246.png`(+`.json`)、`final-hero.png` | 三图+geom+controls；且原生基线是「harness Web UI」而非 Electron 壳窗（见 §5 A6） | slice-chat | 打开 `screens/mygo-chat-after-round156-150dpi.png` 与 `screens/native-harness-chat-1440x920.png` 并排 |
| D-MAIN-SEL | 部分 | `sidebar-*.png`（8 张）、`screens/mygo-chat-pass275-150dpi.png` | 已选会话态三图；现有是侧栏局部 | slice-chat | 看 `sidebar-workspace-groups.png` 与 `sidebar-collapsed-rail.png` |
| D-TERM | 部分 | `terminal-*.png`（`terminal-divider.png`、`terminal-friendly-tab.png`、`terminal-margin.png`）、`terminal-pass259-*checks.json` | 终端开态整屏三图 + 几何 | slice-terminal | 打开 `terminal-friendly-tab.png` |
| D-PLAN-ON | 无 | 无 | 计划面板展开抓图 | slice-chat | — |
| D-PLAN-OFF | 无 | 无 | 计划面板收起抓图 | slice-chat | — |
| D-FILES | 部分 | `preview-pass252-native-and-filesystem.txt`、`tool-image-pass266-preview-authority.json` | 文件 tab+树+预览整屏三图 | slice-workspace | 看 `preview-pass252-native-and-filesystem.txt` 的对照记录 |
| D-DIFF | 部分 | `tool-diff-pass263-native-metrics.json`、`tool-diff-pass263-geometry-comparison.json`、`tool-diff-pass263-*.png` | 工作区 Diff 空态整屏（现有是工具卡 diff） | slice-workspace | 打开 `tool-diff-pass263-geometry-comparison.json` |
| D-DIFF-HIT | 部分 | 同上（`tool-diff-pass263-native-preset-geometry.json`） | 有内容态整屏 | slice-workspace | 同上 |
| D-GIT-STRIP | 部分 | `git-caption-review-pass242.png`、`git-caption-review-closed-pass242.png`、`git-action-disabled.png` | 提交条开态整屏三图 | slice-git | 打开 `git-caption-review-pass242.png` |
| D-GIT-PR | 部分 | `git-action-noring.png`、`git-caption-review-*` | PR 面板开态 | slice-git | 看 `git-action-noring.png` |
| D-INTERACT | 部分 | `permission-dialog-checked.png`、`permission-dialog-unchecked.png` | 审批可见整屏 + 三态 | slice-chat | 打开 `permission-dialog-checked.png` |
| D-LIGHTBOX | 部分 | `tool-image-pass266-reference-…-local-lightbox.png` | 灯箱整屏 + 关闭态 | slice-chat | 打开该 png |
| D-BOOT | 无 | 无（`remote-app-boot.json` 是远程面板启动，不是 boot 页） | boot 启动中抓图 | slice-launcher | — |
| D-BOOT-FAIL | 无 | 无 | boot 失败态抓图（需可复现失败注入） | slice-launcher | — |

### 1.4 功能面板 P-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| P-BROWSER | 部分 | `preview-pass252-*.txt/json`（15 个文件）、`inspector-pass255-reference-picker-browser*.png` | 浏览器预览整屏三图；PiP/缩放/注解态各一张 | slice-preview | 打开 `preview-pass252-full-toolkit.txt` 与 `inspector-pass255-reference-picker-browser.png` |
| P-ASSIST | 无 | 无（`companion-*.png` 是桌宠，不是助理面板） | 助理面板整屏 | slice-assistant | — |
| P-WEBVIEW | 无 | 无 | 插件 Web 面板整屏 | slice-assistant | — |
| P-REMOTE | 部分 | `remote-app-boot.json`、`remote-app-live.json`、`remote-daemon-live.json`、`remote-pairing-app.json` | 远程面板整屏；且 G44/G45 未闭（见 §3） | slice-remote | 打开 `remote-pairing-app.json` 看配对态 |
| P-MARKET | 部分 | `market-*.png`（13 张，含 confirm/details/install/history） | 市场整屏 + 安装/详情/历史三态整屏 | slice-market | 打开 `market-details-wide.png` 与 `market-confirm-dialog.png` |
| P-SCHED | 部分 | `automation-nav-noring.png`、`automation-new-clean.png` | 定时任务整屏 + 新建/编辑/删除态 | slice-schedule | 打开 `automation-new-clean.png` |
| P-IM | 无 | 无 | 消息面板整屏 | slice-im | — |
| P-USAGE | 无 | 无抓图；仅 `docs/parity/native-usage-stats.md`、`usage-date-filters.md` 等文档 | 用量面板整屏（含近 7 日/28 日与筛选） | slice-usage | — |

### 1.5 设置画面 S-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| S-DESK | 无 | 无 `settings-desktop*` 抓图 | 桌面页整屏 | slice-settings | — |
| S-ACCT | 无 | 仅 `sidebar-account-seat-signedout.png`（侧栏座位，不是页面） | 账号页整屏（未登录/已登录） | slice-account | — |
| S-CRED | 无 | 无 | 凭据页整屏（掩码/显示密钥） | slice-settings | — |
| S-APPEAR | 部分 | `settings-appearance.png`、`-bottom.png`、`settings-font-*.png`（5）、`settings-theme-*.png`、`settings-wallpaper-sources.png`、`settings-color-placeholders.png` | 浅/深两色整屏 + 三图 | slice-settings | 打开 `settings-appearance.png` |
| S-UI | 部分 | `settings-interface.png`、`settings-interface-gear.png`、`settings-interface-single-save.png` | 开关三态整屏 | slice-settings | 打开 `settings-interface.png` |
| S-MODEL | 部分 | `settings-models.png`、`settings-models-compact.png` | 空态/有模型态整屏 | slice-settings | 打开 `settings-models.png` |
| S-PLUG | 部分 | `settings-builtin-loading.png`、`settings-market-rail.png` | 插件页整屏（内置/市场/禁用三态） | slice-settings | 打开 `settings-builtin-loading.png` |
| S-SKILL | 部分 | `settings-skills-browse.png`、`settings-skills-empty.png` | 技能页整屏（含安装/编辑） | slice-settings | 打开 `settings-skills-empty.png` |
| S-MCP | 部分 | `settings-mcp-compact.png` | MCP 页整屏（含健康态） | slice-settings | 打开 `settings-mcp-compact.png` |
| S-AGENT | 部分 | `settings-agent-presets-grid.png`、`-tight.png` | 预设页整屏 + 指南弹窗 | slice-settings | 打开 `settings-agent-presets-grid.png` |
| S-ABOUT | 部分 | `settings-about.png`、`-clean/-brand/-diagnostics.png` | 关于页整屏 | slice-settings | 打开 `settings-about.png` |
| S-OTHER-<ns> | 无 | 无 | 运行时每个 ns 一行，先枚举 ns 清单（见 §5 A2） | slice-settings | — |

### 1.6 启动器功能 FL-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| FL-RAIL | 部分 | `round139/151` 文档 + `screens/launcher-task25-icons-150dpi.png` | `cases-FL-RAIL.md`（rail 全部 tab × 状态）+ 执行日志 | slice-launcher | 打开 `screens/launcher-task25-icons-150dpi.png` 对照 6 个图标 |
| FL-HOME-CTA | 部分 | `round147/151` 文档、`launcher-after-homecard-1060x660.png` | `cases-FL-HOME-CTA.md` + 每个 CTA 的结果屏（新屏需先过 §2） | slice-launcher | 打开 `launcher-after-homecard-1060x660.png` |
| FL-HOME-DIAG | 部分 | `docs/status.md:216` 诊断字段说明 | 用例清单 + 展开/折叠执行日志 | slice-launcher | 读 `docs/status.md` 启动器段 |
| FL-VER | 部分 | `round126` 更新进度行 | 用例清单 + 检查更新/下载/安装三态日志 | slice-shell | — |
| FL-IMP | 部分 | `docs/import_flow`、`internal/desktop/importer` 测试 | 用例清单 + 空/有结果/失败三态日志 | slice-import | — |
| FL-PLUG | 部分 | `round139/145` 插件排查 + `pluginRecovery` | 用例清单（禁用/启用/跳过/重启） | slice-launcher | — |
| FL-SET-TAB | 部分 | 启动器设置 tab（round139） | 用例清单 + 每个控件结果 | slice-launcher | — |
| FL-ROUTE | 部分 | `internal/desktop/deeplink` + round134 菜单 | 用例清单（全部线路/深链） | slice-shell | — |

### 1.7 桌面功能 FD-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| FD-TITLE | 部分 | `settings-shortcuts-dialog-1440.png`、`menu-overlay.png`、round134 菜单矩阵 | 菜单全项集合与本体全等的清单 + 每项结果 | slice-shell | 打开 `menu-overlay.png` |
| FD-TERM | 部分 | `terminal-*.png`、`terminal-pass259-*checks.json` | 终端内全部操作用例（输入/缩放/关闭/多标签） | slice-terminal | 打开 `terminal-pass259-affected-checks.json` |
| FD-SESS | 部分 | `sidebar-*.png`（归档/组/收起）、round 文档 | 搜索/新建/刷新/置顶/重命名/归档/删除的确认取消全用例 | slice-chat | 打开 `sidebar-archived-toggle.png` |
| FD-CHAT | 部分 | `conversation-*.png`、`tool-*` 证据、`final-hero.png` | 更早/反馈/编辑/引用/灯箱用例清单 | slice-chat | 打开 `final-hero.png` |
| FD-PLAN | 无 | 无 | 计划面板全部控件用例 | slice-chat | — |
| FD-INTERACT | 部分 | `permission-dialog-checked.png`、`permission-dialog-unchecked.png`、`settings-permission-select.png` | 审批全部路径（允许/拒绝/记住/风险） | slice-chat | 打开 `permission-dialog-checked.png` |
| FD-COMPOSE | 部分 | `composer-*.png`（accessible/control-ink/placeholder/sent）、round156 composer 114 | 模型/权限/预设全选项 + 排队/技能/目录/发送停止/chip/拖放用例 | slice-chat | 打开 `composer-sent-pass243.png` |
| FD-FILES | 部分 | `preview-pass252-native-and-filesystem.txt` | 树/预览/编辑/外部打开/PDF/Diff 全用例 | slice-workspace | 打开该 txt |
| FD-DIFF | 部分 | `tool-diff-pass263-*`（工具卡 diff） | 工作区 Diff 全部操作用例（与 D-DIFF 同源） | slice-workspace | 打开 `tool-diff-pass263-native-metrics.json` |
| FD-GIT | 部分 | `git-*.png`、`git-caption-review-pass242.png` | fetch/pull/push/commit/PR 全用例（需可回滚远端） | slice-git | 打开 `git-caption-review-pass242.png` |
| FD-PANEL-BROWSER | 部分 | 同 P-BROWSER | 面板 controls 全量用例 | slice-preview | 同 P-BROWSER |
| FD-PANEL-ASSIST | 无 | 无 | 面板 controls 全量 | slice-assistant | — |
| FD-PANEL-WEBVIEW | 无 | 无 | 面板 controls 全量 | slice-assistant | — |
| FD-PANEL-REMOTE | 部分 | 同 P-REMOTE | 面板 controls 全量 + G44/G45 | slice-remote | 同 P-REMOTE |
| FD-PANEL-MARKET | 部分 | 同 P-MARKET | 面板 controls 全量（安装/详情/历史） | slice-market | 同 P-MARKET |
| FD-PANEL-SCHED | 部分 | 同 P-SCHED | 面板 controls 全量 | slice-schedule | 同 P-SCHED |
| FD-PANEL-IM | 无 | 无 | 面板 controls 全量 | slice-im | — |
| FD-PANEL-USAGE | 无 | 无 | 面板 controls 全量 | slice-usage | — |

### 1.8 设置功能 FS-*

| ID | 现状 | 已有证据 | 还差什么 | owner | 审查快速路径 |
| --- | --- | --- | --- | --- | --- |
| FS-NAV | 部分 | `settings-general.png`、`settings-market-rail.png`、round144 文案审计 | 左栏导航+过滤+全部 OTHER ns 用例 | slice-settings | 打开 `settings-general.png` |
| FS-PAGE-DESK | 无 | 无 | 页面用例 + 整屏 | slice-settings | — |
| FS-PAGE-ACCT | 无 | 无 | 页面用例 + 整屏 | slice-account | — |
| FS-PAGE-CRED | 无 | 无 | 页面用例（写入/掩码/显示） | slice-settings | — |
| FS-PAGE-APPEAR | 部分 | `settings-appearance.png`、`settings-font-*.png`、`settings-theme-*.png` | 全控件用例（字体/主题/壁纸/效果） | slice-settings | 打开 `settings-font-preview.png` |
| FS-PAGE-UI | 部分 | `settings-interface.png` | 全控件用例 | slice-settings | 打开 `settings-interface.png` |
| FS-PAGE-MODEL | 部分 | `settings-models.png`、`-compact.png` | 全控件用例（增删改/发现/凭据引用） | slice-settings | 打开 `settings-models.png` |
| FS-PAGE-PLUG | 部分 | `settings-builtin-loading.png` | 全控件用例（启用/禁用/安装/卸载） | slice-settings | 打开 `settings-builtin-loading.png` |
| FS-PAGE-SKILL | 部分 | `settings-skills-browse.png`、`-empty.png` | 全控件用例（安装/编辑/删除） | slice-settings | 打开 `settings-skills-browse.png` |
| FS-PAGE-MCP | 部分 | `settings-mcp-compact.png` | 全控件用例（导入/健康/表单） | slice-settings | 打开 `settings-mcp-compact.png` |
| FS-PAGE-AGENT | 部分 | `settings-agent-presets-grid.png` | 全控件用例（复制/指南/编辑） | slice-settings | 打开 `settings-agent-presets-grid.png` |
| FS-PAGE-ABOUT | 部分 | `settings-about.png`、`-diagnostics.png` | 全控件用例（诊断/复制/链接） | slice-settings | 打开 `settings-about.png` |
| FS-PAGE-OTHER-<ns> | 无 | 无 | 每 ns 一页用例（先枚举，见 A2） | slice-settings | — |
| FS-BACK | 无 | 无 | 返回后 D-MAIN 1:1（依赖 D-MAIN 已双签） | slice-chat | — |

### 1.9 交付门禁 G1–G6

| ID | 现状 | 已有证据 | 还差什么 | owner |
| --- | --- | --- | --- | --- |
| G1 | 无 | 见 §1.2–1.5 | 每一画面行双签（当前 0） | lead |
| G2 | 无 | 见 §1.6–1.8 | 每一功能行双签 + 用例总数=通过数>0（当前 0 用例文件） | lead |
| G3 | 无 | 见 §1.1 | E1–E7 双签（当前 0） | slice-verify |
| G4 | 无 | round*-build / harness-start json | 证据绑定同一交付 commit/二进制 | lead |
| G5 | 无 | — | 无抽检声明（需在审查备注逐行写） | lead |
| G6 | 无 | — | 历史错误验收未混入的说明 | lead |

### 1.10 组汇总与总签署（§2.5 / §3.4 / §6）

| 项 | 现状 | 说明 |
| --- | --- | --- |
| §2.5 E/L/D/P/S 组计数 | 无 | 合同「组内行数」「双签通过行数」两列本身为空，需用户先确认口径（A3） |
| §3.4 FL/FD/FS 组计数 | 无 | 用例总数取决于尚未生成的 `cases-*.md` |
| §6 总签署 | 无 | 依赖 G1–G6；执行人/审查人/日期均空 |

---

## 2. 合同硬门槛（所有行共用）

要标「完成=是」，每一行至少还要下面六件；目前全仓缺 4–6 件：

1. `baseline`（运行中的 Electron 本体）+ `actual`（MyGo）+ `diff`（高亮）三图，同客户区、同 DPI；
2. `geom.json`：每个可见区块外接矩形与本体误差 ≤1 px（全仓 **0 份**）；
3. `controls.json`：该屏全部控件枚举（**0 份**）；
4. `cases-<ID>.md`：控件 × 状态全量用例（**0 份**）；
5. `accept-log.txt`：位于 `%TEMP%\whaleisle-accept-<UTC日期>\`（**0 份**）；
6. 双签：`完成` 与 `审查完成` 均填 `是 YYYY-MM-DD 署名`（**全空**）。

---

## 3. 20261025 缺口 → 验收行映射（25 条 open）

| 缺口 | 严重度 | 阻塞的验收行 | 下一 slice |
| --- | --- | --- | --- |
| G27 设置各专页深度 | medium | S-DESK、S-ACCT、S-CRED、S-OTHER-*、FS-PAGE-DESK/ACCT/CRED/OTHER、FS-NAV | slice-settings：先补桌面/账号/凭据三页整屏与用例 |
| G28 字体基线/滚动条/选择复制/tooltip | medium | 所有 D-*/S-*/L-* 的像素行（阈值 0 下必然失配） | slice-chrome：字体度量与滚动条像素规格 |
| G09 工作区面板 | medium | D-FILES、D-DIFF、D-DIFF-HIT、FD-FILES、FD-DIFF、FD-TERM | slice-workspace |
| G52 i18n 漂移（cmd 254 处 inline） | medium | D-MAIN、FD-TITLE、FD-COMPOSE、S-UI、FS-NAV | slice-i18n |
| G44 手机远程 G1 无 mobile runtime | high | P-REMOTE、FD-PANEL-REMOTE | slice-remote（需真机） |
| G45 LAN 发现 / TLS-SSH | high | P-REMOTE、FD-PANEL-REMOTE | slice-remote（需真机/网络） |
| G46 remote shell / git dispatch | medium | P-REMOTE、FD-PANEL-REMOTE、FD-GIT（远程 git） | slice-remote |
| G39 通知呈现 | medium | 无对应合同行（见 A1） | slice-shell |
| G31b 桌宠渲染层 | medium | 无对应合同行（见 A1） | slice-pet |
| G32 桌宠行为族 | medium | 无对应合同行（见 A1） | slice-pet |
| G33 桌宠快捷聊天窗 | low | 无对应合同行（见 A1） | slice-pet |
| G34 手机 PR 复核窗 | low | P-REMOTE（子窗） | slice-remote |
| G21 Inspect/trajectory 面板 | low | D-MAIN（工具行 Inspect 路径）、FD-CHAT | slice-chat |
| G10 标题栏右侧控件 | low | D-MAIN、FD-TITLE | slice-chrome |
| G03 菜单栏可见性 | low | D-MAIN、FD-TITLE | slice-chrome（需实机） |
| S01 harness 页面下载 | medium | D-MAIN、FD-CHAT（下载结果屏未验） | slice-shell |
| S02 安装空间预检 | low | P-MARKET、FD-PANEL-MARKET | slice-market |
| S03 DevTools 快捷键 | low | FD-TITLE（菜单外快捷键集合） | slice-shell |
| S04 浏览器预览缩放 | low | P-BROWSER、FD-PANEL-BROWSER | slice-preview |
| S05 插件崩溃取证 | low | L-PLUG、FL-PLUG | slice-launcher |
| S27 hero 纵向 seat | low | D-MAIN | slice-chat |
| S28 聊天 chrome 残项（772 vs 774.4 / 1.7 DIP） | low | D-MAIN（阈值 0 不可过） | slice-chat |
| S30 通知 AUMID | low | 无对应合同行（见 A1） | slice-shell |
| S31 验收零双签 | high | G1–G6、§6 全部 | lead（只能用户签） |
| S34 project stage 成本/校验口径 | low | 无直接验收行（影响 E2/E6 可复现性） | slice-runtime |

---

## 4. Top-10 交付阻塞

1. **没有一屏满足 §0.1 三图 + geom + controls**：96 行全部无法签，这是唯一的结构性阻塞。
2. **`cases-<ID>.md` 0 份**：FL/FD/FS 共 40 行无法证明「用例总数=通过数>0」。
3. **`accept-log.txt` / `%TEMP%\whaleisle-accept-*` 0 份**：E6 与 G4 无法成立。
4. **`fixture.md` 缺失**：E5 无法成立，且所有抓图缺「同等数据状态」声明。
5. **原生基线不是 Electron 壳窗**：D-*/S-* 目前只有 harness Web UI 的 CDP 参考（round152/155），§0.1 要求「运行中 Electron 本体逐屏对照」。
6. **E7 蒙版登记缺失**：阈值 0 下任何抗锯齿/字体差异都算失配，没有蒙版就无法通过。
7. **四类屏完全没有证据**：S-DESK、S-ACCT、S-CRED、P-ASSIST/P-WEBVIEW/P-IM/P-USAGE、D-PLAN、D-BOOT。
8. **远程面板只有 JSON 状态**：P-REMOTE/FD-PANEL-REMOTE 无法出图，且被 G44/G45 高缺口卡住。
9. **像素残项明确存在**：composer 772 vs 774.4、导航基线 1.7 DIP、字体/滚动条未对齐（S28/G28），阈值 0 下必挂。
10. **契约本身有空白口径**：组汇总计数、S-OTHER 数量、宠物/通知是否在范围（见 §5），未澄清前无法给出「总行数」。

---

## 5. 需要用户澄清的契约歧义（A1–A8）

| ID | 歧义 | 影响 |
| --- | --- | --- |
| A1 | **桌宠（P-PET/companion）、系统通知、首启「预览版说明」在合同里没有行**，但产品有这些表面 | 这些功能无法被双签；要么补行，要么明确「不在验收范围」 |
| A2 | `S-OTHER-__` / `FS-PAGE-OTHER-__` 要求「运行时每一个 ns 一行」，但 ns 数量随 profile/插件变化 | 无法确定 §2.4/§3.3 的总行数与分母 |
| A3 | §2.5 / §3.4 组汇总表的「组内行数」「用例总数」两列在合同里是空的 | 组级双签无法计数；建议交付时由用户确认分母 |
| A4 | §0.1 允许「已登记挥发蒙版」，但 E7 未定义格式与审批方式 | 当前所有抓图都无蒙版，阈值 0 下无法判定失配像素 |
| A5 | E6 要求证据根在 `%TEMP%\whaleisle-accept-<UTC日期>\`，与仓库内 `docs/review-20261022/evidence/` 的现行做法冲突 | 是迁移证据，还是改合同口径？ |
| A6 | §0.1 要求「与正在运行的 Electron 本体逐屏对照」；D-*/S-* 的内部内容现在用 harness Web UI 的 CDP 抓图作参考 | 是否接受「壳窗 + 同源 Web UI」作为 baseline？ |
| A7 | G4「证据绑定本 commit/二进制」：交付 commit/pass 尚未锁定（round 每轮都换 passN） | 无法确定 E2/G4 的绑定对象 |
| A8 | §2.1 要求 L-* 每帧附「无桌面会话/发送/文件」的分壳备注，但未规定备注格式与由谁核对 | 组级审查口径不明 |

---

## 6. 离线不可验证（需要真机/真网/真安装）

- **P-REMOTE / FD-PANEL-REMOTE**：G44（mobile runtime）、G45（LAN 发现、TLS/SSH）需要真实手机、真实中继与 SSH 目标；
- **通知相关（G39/S30）**：需要安装后的 AUMID 与系统通知中心观察，开发态不注册 `Electron.lnk`；
- **下载（S01）**：需要一次真实下载落盘；
- **G03 菜单栏可见性**：headless/UIA 下不可断言；
- **像素审计（G28/S28）**：需要同 DPI 实机与 Electron 壳窗基线；
- **G46 remote shell / git dispatch**：需要第二台主机；
- **S05 插件崩溃取证**：需要真实 OOM/EACCES 崩溃注入；
- **安装/市场/插件真实安装（FD-PANEL-MARKET、FS-PAGE-PLUG）**：会改本机环境，需要可回滚环境。

---

## 7. 使用建议（给用户）

1. 先看 §4 的 Top-10：前 5 条是流程/证据基建，不是代码；建议先由你确认 A1–A8，再让团队按 §3 的 slice 顺序补证据。
2. 任何一行在你打开 §1 的「审查快速路径」文件后，若认为达标，请**你自己**在合同对应行填 `是 YYYY-MM-DD <你的署名>`；本索引不会代填。
3. 本索引不改变合同任何文字；如合同需要补行/改口径（A1/A5/A6），应由你修改合同后再重跑索引。

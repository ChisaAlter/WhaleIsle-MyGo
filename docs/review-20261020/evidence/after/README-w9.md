# W9 三块结构还原：市场 / 插件列表 / 定时任务

本目录的 `surface-{market,plugins,schedule}.png` 由 `SURFACES_GALLERY_DIR` 门控的 tester 快照生成，
不依赖运行中的窗口：

```sh
SURFACES_GALLERY_DIR=docs/review-20261020/evidence/after \
  go test ./internal/ui/market -run TestMarketSurfaceSnapshot
SURFACES_GALLERY_DIR=docs/review-20261020/evidence/after \
  go test ./internal/ui/pluginmanager -run TestManagerNativeNavigationConfigurationsAndNarrowLayout
SURFACES_GALLERY_DIR=docs/review-20261020/evidence/after \
  go test ./internal/ui/schedule -run TestScheduleSurfaceSnapshot
```

（PowerShell：`$env:SURFACES_GALLERY_DIR='docs\review-20261020\evidence\after'` 后跑同样三条。）

| 快照 | 覆盖的结构 |
| --- | --- |
| `surface-market.png` | 市场卡片：44px 图标块 + 名称 + 作者/星标 + 最多两行描述 + 分类与安装状态（左下分类、右侧操作），两列自适应网格；头部与分类 tabs 由 `marketHeader`/`marketTabs`/`catalogCategories` 提供 |
| `surface-plugins.png` | 插件列表：图标 + 名称 + 标签 + 两行摘要 + `vX.Y.Z · 来源` 元信息行 + 启用 Switch + `⋯` 动作菜单；按「已启用 / 已停用」分组，实验性插件单独成组并保留「更多」 |
| `surface-schedule.png` | 定时任务：任务目录 + 任务行（状态点、cron 摘要、最近运行、开关）与详情面板；时间编辑器为时/分选择器，分钟按 15 分钟刻度，表单内显示「下次运行」预览与就地校验 |

## 与原版的差异（有意为之）

- 安装版 `ui-plugin-market` 的条目是 48px 图标的**单列列表**；本任务按 CLOSURE-AUDIT 的表述做成
  **两列卡片**（44px 图标块、两行描述、右下安装状态），保留了原版的标题行/副标题/描述/操作四段结构。
- `ui-schedule` 的分钟选择在原始实现里是 0–59 全量；这里按任务要求收敛到 15 分钟刻度，
  已存在的非刻度任务保留原值（`minuteMarks`）。
- 定时任务的「下次运行」在原生侧只做**预览**（45 天窗口内按 cron 匹配），实际调度仍由 host 负责。

## 已知缺口

- 插件列表的搜索框是界面本地状态（`ui.Local`），不进入 manager 状态机；分页/索引来源仍在「发现」页。
- 市场卡片图标暂用标题首字母方块，未接市场条目的远程图标（会引入网络依赖与缓存失效处理）。

## W9-R 追加：远程与 Project

| 快照 | 覆盖结构 |
| --- | --- |
| `surface-remote-machines.png` | 机器行卡片：状态点（已连接/当前/离线三种色）、名称 + 当前/已连接标记、`user@host:port` 等宽行、延迟/最近连接/工作区元信息、`⋯` 动作菜单（选择/测试/连接/编辑/移除，右键同菜单） |
| `surface-remote-files.png` | 文件页：可点击面包屑（逐段返回上级）+ 每行目录/文件图标 + 名称 + 类别；加载/空/错误三态与「新建文件夹」内联表单 |
| `surface-pairing.png` | 配对三步：生成配对码 → 等待扫码 → 配对完成，按运行时状态着色（进行中/完成/超时失败） |
| `surface-project.png` | **未生成**：ObserverView 需要完整的 projectflow fixture（Listing + Coordinator 会话 + 观察器状态），本轮余额不足；Project 观察器头部已重构（标题 + 状态词 + 等宽路径行 + 无 `● ` 前缀的分区 chip），由既有 projects 测试覆盖 |

复现（PowerShell）：`$env:SURFACES_GALLERY_DIR='docs\review-20261020\evidence\after'` 后
`go test ./internal/ui/remote -run TestRemoteMachineCardsSnapshot|TestRemoteFilesSnapshot|TestRemotePairingSnapshot`。

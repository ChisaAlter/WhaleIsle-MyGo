# 原生重写 vs 原版鲸屿：对比审查方法（2026-10-20）

## 目的

用户反馈：原生 Go 界面在 UI、布局、功能、特效上与原版鲸屿差距仍然很大。本轮审查要给出**可执行、有证据**的差距清单，而不是再写一份"已还原"的说明。

## 对照基准

| 角色 | 位置 | 说明 |
| --- | --- | --- |
| 运行中的原版 | `C:\软件\Whale Isle`（Electron 0.3.9，监听 3080） | 视觉真值，截图见 `evidence/original-electron-3080.png` |
| 原版前端源码（rc.2 构建） | `C:\软件\Whale Isle\resources\vendor\deepseek-harness\packages\client\<pkg>\lib\client.js`（含样式字符串） | 只读；与运行版一致，但为构建产物 |
| 原版前端源码（alpha.1，可读 TSX/CSS） | `runtime\vendor\deepseek-harness\packages\client\<pkg>\src` | 只读参考；比安装版旧，新增组件可能在安装版才有 |
| Electron 外壳源码 | `reference\electron\src` | 窗口、菜单、托盘、壁纸等外壳行为 |
| 原生实现 | `cmd\whaleisle`、`internal\ui\*`、`internal\desktop\*` | 待审查对象 |
| 原生运行实例 | 源码树 `whaleisle.exe`（3090，数据目录 `.devdata`） | 截图见 `evidence/native-go-3090.png` |

**版本提示**：原生按 alpha.1 源码还原，而线上产品是 rc.2；凡是 rc.2 新增/改动的组件，都可能被"还原成旧版"却没人注意。发现这类问题请在条目里标注 `version-drift`。

## 每条的写法（Markdown 表格）

```
| # | 项目 | 原版行为（证据） | 原生现状（证据） | 差距 | 严重度 | 修复方向 |
```

- 证据必须是 `路径:行号`，或截图坐标；不允许"大概/应该"。
- 严重度：
  - **P0**：一眼可见的观感断裂（无边框窗口、玻璃/壁纸、发送按钮、布局比例、空态、字号层级），或用户每天都会用的功能缺失。
  - **P1**：某个界面的结构/交互明显不同，但主流程可用；影响专业感。
  - **P2**：细节（间距、hover、阴影、动效时长、文案）。
- 每份文档先写"5 个最致命差距"，再展开明细表。
- 只写查证过的；没查到证据的放到"待验证"。

## 规则

1. 只读外部目录：`C:\软件\Whale Isle\...`、`reference\...`、`runtime\vendor\...`。不得修改、移动、删除。
2. 不改源码；本任务只产出文档。唯一写入范围：自己那一份 `docs\review-20261020\<area>.md`。
3. Windows 上递归扫 `C:\软件\...` 会顺着 pnpm 的 junction 爆炸式递归，**不要**对它 `Get-ChildItem -Recurse`；请用 `grep` 工具按包目录、或用 `read` 读具体文件。
4. 原生侧用 `grep`/`read`；Go 文件是 UTF-8。PowerShell 5.1 的 `Get-Content` 会按 GBK 误读，读中文请用 `read` 工具。
5. 结论要能直接排期：给出"修复方向"（改哪个文件/加什么能力），不必给完整实现。

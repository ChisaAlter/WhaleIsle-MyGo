# F16 Windows Git long worktree configuration evidence

## 当前决定：用户批准短物理工作树布局

2026-11-17 用户批准改用短物理存储布局，保留业务 ID、分支和回执。Windows 新托管工作树使用 `%USERPROFILE%/.whaleisle/worktrees/<profile-key>/w-<workspace-key>`；生成路径、所有权、执行目录授权和清理共用实现。旧回执保持原址，不自动搬动正在使用或用户自选的目录。

[真实 Git 路径验证](compact-worktree-path.log)已覆盖 prepare/reuse、snapshot、baseline/changes/revert、cleanup 及 hook 内子 Git：旧布局281字符，新物理路径132字符。旧布局脏工作树原位续接、篡改回执不增加执行授权通过；[本机只读盘点](compact-worktree-existing-inventory.json)为32条 user-owned、0条 Project-owned，无迁移对象。[嵌入 runner 源闭包](compact-worktree-embedded-closure.log)通过。新布局的 Go 桥接集成与最终桌面包尚未重编，受磁盘条件影响。

下文保留原 canonical 超长目录与 Git runtime 的否证记录。原 Git 的任意长 cwd 限制没有改变；新方案解决的是应用自动生成的托管工作树布局，不把用户自己的长目录静默搬走。

## 原目录与 runtime 调查记录

本轮实验使用 Git 2.55.0.windows.5、Node v24.20.0，只操作一个自有合成仓库。真实 worktree 路径为 285 个字符，关联的 Git 管理目录为 84 个字符。完整 argv、进程物理 cwd、Git 管理目录和实际文件内容在下方记录；临时目录已回收。

**结论：本轮 11 组配置、alias 与目录参数组合均不能让当前 Git runtime 正确执行长工作树的 status/reset。** `core.worktree` 单独配合短 Git 管理目录时返回 0，但真实工作树仍保留 `owned worker edit`，`base` 被写入了短进程 cwd，不能作为修复采用。

| 调用方式 | status | reset | 实际目标文件 |
| --- | --- | --- | --- |
| 短 cwd + `--git-dir` + `--work-tree` | 128，Filename too long | 同左 | 保留原编辑 |
| `GIT_CONFIG_COUNT` 提前声明 longpaths | 128，Filename too long | 同左 | 保留原编辑 |
| `--config-env` 声明 longpaths | 128，Filename too long | 同左 | 保留原编辑 |
| `core.worktree=canonical` + `core.bare=false` | 0 | 0 | **没有检出；文件误写入短 cwd** |
| 上述配置同时显式 `--work-tree` | 128，Filename too long | 同左 | 保留原编辑 |
| alias 包装内建命令，显式目标 | 128，Filename too long | 同左 | 保留原编辑 |
| alias 内部 `-C canonical` | 128，Filename too long | 同左 | 保留原编辑 |
| alias 内部设置 Git/worktree 参数 | 128，拒绝 alias 改环境 | 同左 | 保留原编辑 |
| shell alias 包装，显式目标 | 128，Filename too long | 同左 | 保留原编辑 |
| namespaced `--work-tree` | 128，Filename too long | 同左 | 保留原编辑 |
| `extensions.worktreeConfig` + `config.worktree` | 128，Filename too long | 同左 | 保留原编辑 |

每组 reset 前先在真实目标写入 `owned worker edit`。可接受结果要求：真实目标变为 `base`、source 保持 `base`、短进程 cwd 没有 `result.txt`。因此返回码 0 不会掩盖写错目录。

## 固定版本源码解释

核对的是与本机版本一致的官方 Git for Windows `v2.55.0.windows.5`，没有把浮动 main 的行为当作本机证据。

- [`compat/mingw.c:284-302`](https://github.com/git-for-windows/git/blob/v2.55.0.windows.5/compat/mingw.c#L284)：longpaths 配置在 repository config hash 初始化前使用关闭默认值。`-c`、环境配置和早期 alias 读取本身不能保证这个状态已经建立。
- [`compat/mingw.c:1661-1724`](https://github.com/git-for-windows/git/blob/v2.55.0.windows.5/compat/mingw.c#L1661)：真实路径解析先进行受上述配置控制的路径转换，再调用 Windows 获取最终物理路径。短 junction、subst 和 8.3 别名会被还原，不能把它们当成 canonical 工作树身份的替代。
- [`compat/mingw.c:1737-1761`](https://github.com/git-for-windows/git/blob/v2.55.0.windows.5/compat/mingw.c#L1737)：getcwd 的两个宽字符缓冲区仍为 MAX_PATH；读取 cwd 和解析最终路径均检查这个上限。即使绕过最早的进程 cwd 限制，Git 自身仍可能失败。
- [`alias.c:84-90`](https://github.com/git-for-windows/git/blob/v2.55.0.windows.5/alias.c#L84)：alias 查询使用早期配置读取，解释了 alias 未能提前启用 longpaths 的实验结果。
- [`setup.c:496-505`](https://github.com/git-for-windows/git/blob/v2.55.0.windows.5/setup.c#L496) 与 [`git.c:479-500`](https://github.com/git-for-windows/git/blob/v2.55.0.windows.5/git.c#L479)：需要工作树的内建命令会在 setup 后切换到解析出的真实工作树。只从短 cwd 启动进程无法保持后续 Git 一直工作于短 cwd。
- [`setup.c:786-805`](https://github.com/git-for-windows/git/blob/v2.55.0.windows.5/setup.c#L786)：linked worktree 对 common config 和 per-worktree config 的处理不同。仅传 `core.worktree` 的返回成功必须读回实际位置；本次单独启用 per-worktree config 仍然遇到真实长路径限制。

## 产品适配边界

F16 的验收保持真实 canonical 目录、Project/workstream ID、worktree 登记、分支、原编辑和创建回执。不能把 `unsupportedPath` 诊断当作已修复，也不能降低原来的长路径成功测试。

若采用兼容 Git runtime，适配需要同时覆盖进程启动、Git 早期真实路径解析和 cwd 查询；随后验证 prepare/reuse、snapshot、baseline、changes、revert 和 cleanup。Go 写 owner 的仓库 identity probe 也使用同一个 runtime 与路径合同。单独使用 bare plumbing 加 `checkout-index --prefix` 只解决首次文件物化，无法替代上述持续使用闭环。

原 `TestWindowsLongWorktreeCheckoutAndInterruptedContent` 的最低路径断言是 225 个字符；本轮专门区分超过 MAX_PATH 的 285 字符实际目录。该原测试没有修改。下方是本轮原始记录，不包含之前未留逐条 argv 的探索性实验。

## 上游修复线索与可执行路线

2026-11-17 通过官方 GitHub API 核对，[Git for Windows PR #3877](https://github.com/git-for-windows/git/pull/3877) 仍为 open、未合并；head 为 `82225ba08e1ba9c492b293b239d0d131fe3acdf5`。已读取其[四段补丁](https://patch-diff.githubusercontent.com/raw/git-for-windows/git/pull/3877.patch)。这是基于 2022 年代码的修复提案，需迁移到所选维护版本。

### 后续候选核查：尚无可直接接入的完整 runtime

用户批准继续处理未关闭项后，已重新查询官方发行、当前源码、提案 head 的构建记录和本机工具链；原始结果见 [f16-candidate-sources.json](f16-candidate-sources.json)。

- 当前正式发行是 [2.56.1.windows.1](https://github.com/git-for-windows/git/releases/tag/v2.56.1.windows.1)，其 [`mingw_getcwd`](https://github.com/git-for-windows/git/blob/v2.56.1.windows.1/compat/mingw.c#L1742) 仍使用两个 `MAX_PATH` 缓冲区，发行 manifest 也未声明 `longPathAware`。官方 main 在本次核对的 `e766978a9db5cdf196fdf3aaa63541e02ca513d6` 同样保留该实现。不能把更新到最新正式 Git 当作 F16 修复。
- PR #3877 的 head 没有官方或作者仓库的 Actions run / check，作者仓库也没有 Release。已查来源中没有可验证的预构建修复包；没有下载和运行无修复版本来重复旧失败。
- 这不只是增大父进程缓冲区的问题。微软 [SetCurrentDirectory 合同](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-setcurrentdirectory)说明，当前目录超过 `MAX_PATH` 会使 `CreateProcessW` 失败。Git 维护者在[该提案的解释](https://github.com/git-for-windows/git/pull/3877#issuecomment-2149394609)中指出，hook、alias、脚本和子 Git 同样受影响。因此仅移植该补丁并验证 `init/status`，无法满足现有完整 Git 工作流合同；不能通过禁用 hook 或改工作树身份获得表面成功。
- 修正此前环境判断：本机已安装 VS Build Tools 2022，包含 MSVC 19.44.35229、CMake 3.31.6、Ninja 1.12.1 和 Windows SDK 10.0.26100.0，只是不在当前 PATH。无需据旧 PATH 查询再安装 SDK。Git 的依赖闭包、完整长路径子进程方案与构建峰值仍需先确定；当前 C 盘约 43 GiB，预计新增超过 1 GiB 的构建继续受 50 GiB 余量要求约束。

**F16 仍是功能未修复。** 受控 Git runtime 是保留现有目录与 hooks 语义的候选研究方向，并非已证实可完成的单补丁方案。下一步要先得到覆盖父 Git、子 Git 和 hook 启动的可信候选及维护方案，再做产品接线。若没有该候选，需要明确重新评估底层 runtime 或物理工作树布局；布局变更属于新的产品取舍，不能悄悄应用到现有工作树。本轮没有修改系统 Git、下载 SDK、构建不完整 runtime 或改变用户目录。

| 补丁 | 对本问题的作用 | 采用时需要补齐的证据 |
| --- | --- | --- |
| 1/4、2/4 | 把 longpaths 配置读取提前到仓库发现之前，并使显式 Git 管理目录长度检查使用可支持的上限 | 当前配置接口已变化；验证子进程配置确实提前生效，保留用户配置优先级 |
| 3/4 | 增加长路径 `git init` 回归用例 | 还需覆盖本项目 linked worktree 的完整使用流程 |
| 4/4 | 扩大 getcwd、临时文件和路径所有权检查的缓冲区；用 `SetCurrentDirectoryW` 替代 `_wchdir`；增加 longPathAware manifest | 构建产物、错误返回、子进程与本项目调用链需要实测 |

补丁 4 明确保留了长路径 submodule 问题。该提案提供了可移植的修复方向；当前仍没有通过本项目验收的兼容 runtime。

下一步按以下顺序推进，由 Git 集成负责人承接候选 runtime 与产品接线：

1. 选择受维护的 Git for Windows 基线，移植上述机制并处理当前代码差异；固定来源、版本和构建入口。若准备构建，先按本机磁盘约束估算 SDK、源码和中间产物峰值；当前研究没有安装工具链或构建 Git。
2. 先用短进程 cwd 和真实 canonical target 验证候选。明确子进程早期配置来源，不修改用户全局配置；验证 `SetCurrentDirectoryW` 失败能返回正确错误。逐条确认文件实际位置与 linked worktree 索引，避免本轮已发现的假成功。
3. 通过后将候选接入同一个 Git 进程 Adapter，覆盖 Project JS 调用与 Go `AcquireWrite` 的 identity probe。保留 canonical workspace 身份、工作树登记、索引和 common Git directory 锁语义；支持明确选定可执行文件与回退，不静默替换系统 Git。
4. 保留原长路径成功测试，再用本轮大于 260 字符的目录验证 prepare/reuse、snapshot、baseline、changes、revert、cleanup，以及脏目录中断恢复和 source 编辑保留。包含 submodule 的仓库需单独验证递归操作；这部分未通过前，不能宣称完整 Git 长路径支持。最终验收还要确认实际交付包能选择并运行同一候选。

完成条件是上述调用链在真实长目录上正确读写并保留恢复状态。PR 存在、补丁可应用、Git 编译成功或单个 `init` 通过，分别只是路线、源码、构建和局部行为证据。

## Raw evidence

```json
{
  "time": "2026-11-17T05:37:44.244Z",
  "node": "v24.20.0",
  "executable": "C:\\Program Files\\Git\\cmd\\git.exe",
  "owned": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0",
  "environmentOverrides": {
    "GIT_CONFIG_NOSYSTEM": "1",
    "GIT_CONFIG_GLOBAL": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\absent-global",
    "GIT_TERMINAL_PROMPT": "0",
    "LC_ALL": "C"
  },
  "fixture": [
    {
      "label": "fixture",
      "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source",
      "argv": [
        "C:\\Program Files\\Git\\cmd\\git.exe",
        "-c",
        "core.longpaths=true",
        "init",
        "--quiet"
      ],
      "envOverrides": {},
      "exit": 0,
      "error": null,
      "signal": null,
      "stdout": "",
      "stderr": ""
    },
    {
      "label": "fixture",
      "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source",
      "argv": [
        "C:\\Program Files\\Git\\cmd\\git.exe",
        "-c",
        "core.longpaths=true",
        "add",
        "--",
        "result.txt"
      ],
      "envOverrides": {},
      "exit": 0,
      "error": null,
      "signal": null,
      "stdout": "",
      "stderr": ""
    },
    {
      "label": "fixture",
      "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source",
      "argv": [
        "C:\\Program Files\\Git\\cmd\\git.exe",
        "-c",
        "core.longpaths=true",
        "-c",
        "user.name=Fixture",
        "-c",
        "user.email=fixture@example.invalid",
        "-c",
        "commit.gpgSign=false",
        "-c",
        "core.hooksPath=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0",
        "commit",
        "--quiet",
        "-m",
        "fixture"
      ],
      "envOverrides": {},
      "exit": 0,
      "error": null,
      "signal": null,
      "stdout": "",
      "stderr": ""
    },
    {
      "label": "fixture",
      "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source",
      "argv": [
        "C:\\Program Files\\Git\\cmd\\git.exe",
        "-c",
        "core.longpaths=true",
        "worktree",
        "add",
        "--no-checkout",
        "-b",
        "fixture-long",
        "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
        "HEAD"
      ],
      "envOverrides": {},
      "exit": 0,
      "error": null,
      "signal": null,
      "stdout": "",
      "stderr": "Preparing worktree (new branch 'fixture-long')\n"
    },
    {
      "label": "fixture",
      "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source",
      "argv": [
        "C:\\Program Files\\Git\\cmd\\git.exe",
        "-c",
        "core.longpaths=true",
        "config",
        "extensions.worktreeConfig",
        "true"
      ],
      "envOverrides": {},
      "exit": 0,
      "error": null,
      "signal": null,
      "stdout": "",
      "stderr": ""
    }
  ],
  "cases": [
    {
      "name": "short-cwd-explicit-target",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "short-cwd-explicit-target status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-0",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "status",
          "--porcelain=v1"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "short-cwd-explicit-target reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-0",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "reset",
          "--hard",
          "HEAD"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "env-early-longpaths",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "env-early-longpaths status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-1",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "status",
          "--porcelain=v1"
        ],
        "envOverrides": {
          "GIT_CONFIG_COUNT": "1",
          "GIT_CONFIG_KEY_0": "core.longpaths",
          "GIT_CONFIG_VALUE_0": "true"
        },
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "env-early-longpaths reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-1",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "reset",
          "--hard",
          "HEAD"
        ],
        "envOverrides": {
          "GIT_CONFIG_COUNT": "1",
          "GIT_CONFIG_KEY_0": "core.longpaths",
          "GIT_CONFIG_VALUE_0": "true"
        },
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "config-env-longpaths",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "config-env-longpaths status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-2",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "--config-env=core.longpaths=F16_LONGPATHS",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "status",
          "--porcelain=v1"
        ],
        "envOverrides": {
          "F16_LONGPATHS": "true"
        },
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "config-env-longpaths reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-2",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "--config-env=core.longpaths=F16_LONGPATHS",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "reset",
          "--hard",
          "HEAD"
        ],
        "envOverrides": {
          "F16_LONGPATHS": "true"
        },
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "core-worktree-canonical",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "core-worktree-canonical status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-3",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "core.bare=false",
          "-c",
          "core.worktree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "status",
          "--porcelain=v1"
        ],
        "envOverrides": {},
        "exit": 0,
        "error": null,
        "signal": null,
        "stdout": "D  result.txt\n",
        "stderr": ""
      },
      "reset": {
        "label": "core-worktree-canonical reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-3",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "core.bare=false",
          "-c",
          "core.worktree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "reset",
          "--hard",
          "HEAD"
        ],
        "envOverrides": {},
        "exit": 0,
        "error": null,
        "signal": null,
        "stdout": "HEAD is now at 7a5fc24 fixture\n",
        "stderr": ""
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "base\n"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "core-worktree-and-explicit-canonical",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "core-worktree-and-explicit-canonical status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-4",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "core.bare=false",
          "-c",
          "core.worktree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "status",
          "--porcelain=v1"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "core-worktree-and-explicit-canonical reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-4",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "core.bare=false",
          "-c",
          "core.worktree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "reset",
          "--hard",
          "HEAD"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "alias-explicit-target",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "alias-explicit-target status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-5",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=status --porcelain=v1",
          "-c",
          "alias.f16reset=reset --hard HEAD",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "f16status"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "alias-explicit-target reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-5",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=status --porcelain=v1",
          "-c",
          "alias.f16reset=reset --hard HEAD",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "f16reset"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "alias-C-after-config",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "alias-C-after-config status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-6",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=-C \"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" status --porcelain=v1",
          "-c",
          "alias.f16reset=-C \"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" reset --hard HEAD",
          "f16status"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: cannot change to 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work': Filename too long\n"
      },
      "reset": {
        "label": "alias-C-after-config reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-6",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=-C \"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" status --porcelain=v1",
          "-c",
          "alias.f16reset=-C \"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" reset --hard HEAD",
          "f16reset"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: cannot change to 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "alias-worktree-after-config",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "alias-worktree-after-config status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-7",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=--git-dir=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/source/.git/worktrees/work\" --work-tree=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" status --porcelain=v1",
          "-c",
          "alias.f16reset=--git-dir=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/source/.git/worktrees/work\" --work-tree=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" reset --hard HEAD",
          "f16status"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: alias 'f16status' changes environment variables.\nYou can use '!git' in the alias to do this\n"
      },
      "reset": {
        "label": "alias-worktree-after-config reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-7",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=--git-dir=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/source/.git/worktrees/work\" --work-tree=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" status --porcelain=v1",
          "-c",
          "alias.f16reset=--git-dir=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/source/.git/worktrees/work\" --work-tree=\"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\" reset --hard HEAD",
          "f16reset"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: alias 'f16reset' changes environment variables.\nYou can use '!git' in the alias to do this\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "shell-alias-explicit-target",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "shell-alias-explicit-target status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-8",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=!git status --porcelain=v1",
          "-c",
          "alias.f16reset=!git reset --hard HEAD",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "f16status"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "shell-alias-explicit-target reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-8",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "-c",
          "alias.f16status=!git status --porcelain=v1",
          "-c",
          "alias.f16reset=!git reset --hard HEAD",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "f16reset"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "namespaced-work-tree",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "namespaced-work-tree status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-9",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=\\\\?\\C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "status",
          "--porcelain=v1"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path '//?/C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "namespaced-work-tree reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-9",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "--work-tree=\\\\?\\C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
          "reset",
          "--hard",
          "HEAD"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path '//?/C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    },
    {
      "name": "per-worktree-config-canonical",
      "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
      "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
      "before": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "status": {
        "label": "per-worktree-config-canonical status",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-10",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "status",
          "--porcelain=v1"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "reset": {
        "label": "per-worktree-config-canonical reset",
        "physicalCwd": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\probe-10",
        "argv": [
          "C:\\Program Files\\Git\\cmd\\git.exe",
          "-c",
          "core.longpaths=true",
          "--git-dir=C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
          "reset",
          "--hard",
          "HEAD"
        ],
        "envOverrides": {},
        "exit": 128,
        "error": null,
        "signal": null,
        "stdout": "",
        "stderr": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long\n"
      },
      "after": {
        "target": "owned worker edit\n",
        "source": "base\n",
        "processCwd": "<ENOENT>"
      },
      "wroteCorrectTarget": false
    }
  ],
  "cleanup": {
    "path": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0",
    "removed": true,
    "scope": "single owned synthetic fixture; no links created"
  },
  "gitVersion": "git version 2.55.0.windows.5",
  "paths": {
    "source": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source",
    "target": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
    "targetLength": 285,
    "canonicalTarget": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\actual\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\padding\\work",
    "gitDir": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work",
    "gitDirLength": 84
  },
  "perWorktreeConfig": {
    "path": "C:\\Users\\48818\\AppData\\Local\\Temp\\whale-f16-config-OXHlX0\\source\\.git\\worktrees\\work\\config.worktree",
    "content": "[core]\n\tlongpaths = true\n\tbare = false\n\tworktree = \"C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work\"\n"
  },
  "summary": [
    {
      "name": "short-cwd-explicit-target",
      "status": 128,
      "statusError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "env-early-longpaths",
      "status": 128,
      "statusError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "config-env-longpaths",
      "status": 128,
      "statusError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "core-worktree-canonical",
      "status": 0,
      "statusError": "",
      "reset": 0,
      "resetError": "",
      "wroteCorrectTarget": false
    },
    {
      "name": "core-worktree-and-explicit-canonical",
      "status": 128,
      "statusError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "alias-explicit-target",
      "status": 128,
      "statusError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "alias-C-after-config",
      "status": 128,
      "statusError": "fatal: cannot change to 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work': Filename too long",
      "reset": 128,
      "resetError": "fatal: cannot change to 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/work': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "alias-worktree-after-config",
      "status": 128,
      "statusError": "fatal: alias 'f16status' changes environment variables.\nYou can use '!git' in the alias to do this",
      "reset": 128,
      "resetError": "fatal: alias 'f16reset' changes environment variables.\nYou can use '!git' in the alias to do this",
      "wroteCorrectTarget": false
    },
    {
      "name": "shell-alias-explicit-target",
      "status": 128,
      "statusError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "namespaced-work-tree",
      "status": 128,
      "statusError": "fatal: Invalid path '//?/C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path '//?/C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    },
    {
      "name": "per-worktree-config-canonical",
      "status": 128,
      "statusError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "reset": 128,
      "resetError": "fatal: Invalid path 'C:/Users/48818/AppData/Local/Temp/whale-f16-config-OXHlX0/actual/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding/padding': Filename too long",
      "wroteCorrectTarget": false
    }
  ]
}
```

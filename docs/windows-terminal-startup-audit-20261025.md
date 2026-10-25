# Windows 启动终端弹窗审查（2026-10-25）

审查对象为本项目 Go 桌面程序及它选择的 runtime/vendor。已实施修复、运行相关测试、构建 Windows GUI 版本，并更新项目根目录的两个主程序入口。按用户要求不保留额外的旧版可执行文件备份，版本历史使用 Git/GitHub 管理。未替换当前提供对话界面的已安装 Electron 程序。

## 主要发现

| 优先级 | 发现 | 证据与影响 | 处理 |
| --- | --- | --- | --- |
| P1 | Git 后台状态读取与数秒闪窗最吻合 | cmd/whaleisle/main.go 的 gitRefreshInterval 为 3 秒；git_refresh.go 在帧更新时节流调用 ReadRepositoryStatus，经 status.go → selectedText/selectedCommand 创建多个 git.exe，PR 另走 gh.exe。重定向输出不保证 Windows 控制台隐藏。 | 保留并复核本轮审查期间已有的 Git 隐藏修复；Windows 辅助函数改为共享 processutil.Background，所有 Git 执行入口已覆盖。 |
| P1 | 文件树忽略判断单独启动 Git | workspacefs/list.go 的 gitIgnoredPaths 在目录读取时调用 git check-ignore，未经过 Git 包配置。展开、文件变化、搜索枚举会重复触发。 | 创建进程后应用共享后台配置。 |
| P2 | 多个外层后台进程缺少隐藏属性 | Project Node、组件 Node、手机远程 Node、WhaleBridge 服务及状态/断开、更新签名 PowerShell、junction 的 cmd、插件安装 runner。父 Node 隐藏不代表后代自动隐藏。 | 共补齐 11 个先前遗漏的 Go 后台执行点。 |
| P2 | 主程序/启动器构建可能使用控制台子系统 | 缺 -H windowsgui 能解释首次启动控制台，单独不能证明每隔数秒的弹窗。审查期间主打包脚本已出现修复。 | 主脚本保留修复；独立启动器打包也补齐；实建 PE Subsystem=2。 |
| P2 | Ready 后立即崩溃可反复重启 | Supervisor 原先每次 Ready 清零 attempt，因此配置的最大重试次数无法限制反复“Ready → 短命退出”。主 Host 本身已有隐藏设置，此缺陷是重复准备/插件启动的放大因素。 | 只有 Ready 持续至少 1 分钟后，下一次故障才恢复重试预算。手动启动仍重置预算，旧 generation 不能影响新代。 |
| P2 | 远程插件启动恢复会调用 PowerShell | dsh-remote 默认启用；lib/index.js 自动恢复 current machine，可经 credential.js:getSecret 启动 DPAPI PowerShell。credential.js 与目录选择 fallback 的 execFile 均未 windowsHide。 | 两处补 windowsHide:true。新增受固定原始/输出 SHA256 约束的脚本，在 runtime staging、clone、打包时重放，并在运行时清单追加两个插件产物校验值。 |

Git 的 3 秒是完成后最少间隔、由下一次渲染触发，并非独立固定 ticker。未抓取修复前实际可见窗口对应 PID，所以“Git 是现场唯一弹窗来源”没有被证明；它是频率最吻合且源码缺陷明确的链路。

审查期间工作区存在其它执行者的修改，Git 隐藏辅助函数和主程序 GUI 构建标记在首读与后读之间出现。本次保留它们及用户其它修改，未 reset、提交或覆盖无关源码。

## 修复边界

共享 processutil.Background 在 Windows 保留现有 SysProcAttr 对象和其它字段，设置 HideWindow=true，并 OR CREATE_NO_WINDOW。其它平台为空操作。已有 Host detach/taskkill 与 Git helper 复用它。

应用内 ConPTY、显式打开编辑器/浏览器/文件管理器、安装器与卸载器维持各自语义，没有全局拦截或扫描后强行隐藏用户窗口。

剩余低优先级依赖分支：固定 ssh2 依赖的 Pageant helper 与 Cygwin cygpath 回退未显式 windowsHide，只在 useAgent=true 和特定 SSH_AUTH_SOCK 配置下触发；默认关闭，不是常规数秒轮询路径。本次未修改第三方依赖。

## 验证结果

- Go 测试通过：internal/desktop/git、workspacefs、components、projectbridge、phoneremote、update、internal/harness、harness/installctrl，以及 cmd/whaleisle。
- internal/processutil Windows 实测通过：以真实 console subsystem 测试进程构造有控制台/无控制台两种父进程。Background 子进程 GetConsoleWindow 均为 0；显式创建控制台的阳性对照均非 0；stdout/stderr 与正常/非零退出码保留。属性保留及幂等测试通过。
- Supervisor 回归通过：短暂 Ready 后崩溃耗尽预算；稳定 Ready 后恢复预算；旧代故障不能重置新代。
- Node 运行时测试通过：session-runtime 11 项；windows-background-runtime 16 项，包括真实字节哈希、双文件预检、重复重放、未知版本/外部链接拒绝、provenance 保留，以及 VM mock 验证真实 getSecret 调用传递 windowsHide:true。
- 相关 Go 包与两个应用入口 go vet 通过；定向 gofmt 检查、git diff --check、两个 PowerShell 打包脚本语法检查通过。
- 桌面与独立启动器 GUI 构建成功；桌面 PE 子系统读取为 2。
- 独立测试数据冷启动：第一次 40 秒 Host 未上线，未通过完整启动验收。第二次 55 秒 Host 监听端口已上线，约 40ms 轮询的可见终端窗口采样未发现归属于测试实例的窗口。该采样不能排除短于采样间隔的窗口，也不等于所有功能已验收。
- 两次自动关闭观察均未在 10 秒内完成，测试实例已清理；退出行为未宣称通过。第二次堆栈表明 Host 进程已启动，UI 主循环正常等待消息。没有把这一测试关闭限制归因为终端弹窗根因。

## 可执行文件

修复版已复制并校验到项目根目录：

- DeepSeekHarness Go.exe
- whaleisle.exe

两者 SHA256 相同：`42c7882db9cd419895f3421be57ed7b471c4dcf805430072b435e2400fb6402a`。

本次创建的两个旧版 EXE 备份及其 `before-fix` 目录已按用户要求删除；后续不再额外创建备份。验证构建、隔离数据和观察结果保留于 `tmp/terminal-audit-20261025/`。没有替换历史 pass/probe 可执行文件或重新生成完整安装包。

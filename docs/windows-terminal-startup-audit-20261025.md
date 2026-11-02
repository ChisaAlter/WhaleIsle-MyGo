# Windows 启动终端弹窗审查（2026-10-25）

审查对象为本项目 Go 桌面程序及它选择的 runtime/vendor。已实施修复、运行相关测试、构建 Windows GUI 版本，并更新项目根目录的两个主程序入口。按用户要求不保留额外的旧版可执行文件备份，版本历史使用 Git/GitHub 管理。未替换当前提供对话界面的已安装 Electron 程序。

## 23:55 运行时反复弹窗修复补充

早一轮只给首级后台程序设置 `CREATE_NO_WINDOW`，并不能阻止未设置 `windowsHide` 的 Node 后代重新创建终端。本轮新增 `processutil.BackgroundTree`：运行时树根使用 `CREATE_NEW_CONSOLE` 与 `SW_HIDE`，在第一条用户指令之前创建隐藏控制台，普通 console 子孙默认继承它；保留原有进程属性、stdout/stderr 管道和退出码。Host、Project Node、通用组件 Node、手机远程 Node、WhaleBridge 服务与插件安装 runner 六条入口已接入。短 Git/taskkill 仍使用原 `CREATE_NO_WINDOW`，应用内 ConPTY 和显式 GUI 打开行为保持各自语义。

本轮完成的验证：

- `tmp/console-runtime-fix/check-build.log`：七个相关 Go 包整包测试、vet、Windows GUI 构建通过。
- `tmp/console-runtime-fix/node-regression.log`：真实 bundled Node 两级默认 `execFile` 调用没有设置隐藏选项，两个 leaf 共享非零但不可见的控制台，stdout/stderr 与退出码 23 均保留；三代 Go 普通后代、互斥标志清理与属性保留测试通过。该 Node 用例没有直接测量 Node 根 HWND。
- `tmp/console-runtime-fix/observer-fixed.log`：窗口事件 hook 在测试程序启动前安装；外层观察器先脱离隐藏 runner 控制台，GUI 根进程以无控制台方式悬挂创建、加入自有 Job 后才恢复，避免继承测试 runner 而掩盖问题。75 秒正常启动/后台运行期间，主窗口可见，owned 终端显示事件、新增未归属终端事件、无法解析的可见窗口事件与 hook 错误均为空。产品 Ready 约 6.879 秒；随后精确关闭自有主窗口，约 5.517 秒自然退出 code 0，observer 清理前自有 Job 已空、两个监听均关闭，没有强制清理。
- 首轮观察器因自身 `FreeConsole` 后运行 PowerShell CDXML cmdlet 而失败，原 `observer.log` 保留；它没有被计为产品成功。修复只移除可选 CIM 订阅和改用 Win32 监听查询，再运行得到上面的有效证据。没有捕获屏幕、点击确认或强行隐藏用户窗口。

项目根目录两个应用入口与配套预览 helper 已按 `tmp/console-runtime-fix/updated-entries.json` 更新，应用 SHA256 为 `c6180db6d21e6b49cdc8570e9f80dc98e2a030c37afd3d1aa2645bf419143d72`。没有修改已安装鲸屿本体、用户配置或此前已校验的完整安装包；此前的完整安装包尚不包含本次修复，应使用更新后的根目录应用入口。

此次实测覆盖正常启动和后台运行。显式创建新控制台、主动 detached 的后代、Job 不可用后 fallback 子进程主动 `CREATE_NO_WINDOW` 再生成的孙进程，不能由控制台继承策略一概保证。窗口事件观察也不等于所有外部命令、远程连接与插件路径穷举；这些边界不应被写成“任意子孙绝不弹窗”。

下方保留此前首轮审计记录，其中旧可执行文件哈希不再代表当前根目录入口。

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
- 早期两次自动关闭观察均未在 10 秒内完成，测试实例已清理；这两次运行没有记录目标 HWND/class、消息处理回执或关闭后的堆栈，原因尚未确定。第二次堆栈采于启动后 8 秒，早于 55 秒观察及关闭阶段，不能用来诊断退出阻塞。
- 后续使用当前共享树的独立 GUI 构建、全新隔离数据，精确向该 PID 唯一的 `MyGoWindow` 发送 `WM_CLOSE` 并核对 `SendMessageTimeout` 回执：无 Host 约 110ms 自然退出，Host 已上线的两次运行分别约 5.998 秒、5.563 秒自然退出，退出码均为 0，无强制清理。最后一次在隐藏 runner 的 Job 尚未关闭前查询，自有新增进程列表已为空，Host 端口也已关闭，确认清理由应用完成。证据位于 `tmp/mac-native-close/diagnosis.md` 与 `host-job-verified.log`。这验证了相应空闲场景；活动任务、待确认保护和全部关闭入口仍需各自验收。生产退出保护未因此改动。

## 可执行文件

修复版已复制并校验到项目根目录：

- DeepSeekHarness Go.exe
- whaleisle.exe

两者 SHA256 相同：`42c7882db9cd419895f3421be57ed7b471c4dcf805430072b435e2400fb6402a`。

本次创建的两个旧版 EXE 备份及其 `before-fix` 目录已按用户要求删除；后续不再额外创建备份。验证构建、隔离数据和观察结果保留于 `tmp/terminal-audit-20261025/`。没有替换历史 pass/probe 可执行文件或重新生成完整安装包。

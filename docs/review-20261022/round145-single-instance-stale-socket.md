# Round145 · 单实例陈旧 socket 导致的静默启动失败

强杀（或崩溃）之后的下一次启动会 exit 0、不显示任何窗口，直到手工删掉 `%TEMP%\mygo-*-1-*.sock`。本轮定位到根因并改为「活着的所有者必须应答」的握手。

## 1. 症状与现场

- 现象：`RequestSingleInstanceLock` 返回 false，`cmd/whaleisle/main.go:2121` 于是直接 return，进程 0 退出、无窗口、profile 里连 `config.json` 都没写。
- 现场：`%TEMP%\mygo--1-0c44173c701a.sock`（key = `ai.deepseek.harness.gui\nMyGo鲸屿`）残留；删掉它后启动恢复正常。
- 平台：Windows。mac/Linux 走完全相同的代码路径，但下面第 2 节的实测只覆盖 Windows。

## 2. 复现（先复现，再改）

`go` 层的 socket 行为在 Windows 上实测（`AF_UNIX`）：

| 场景 | 结果 |
|---|---|
| 只剩 socket 文件、无人监听 | `Dial` → `actively refused`；`Listen` → `WSAEADDRINUSE`；`Remove`+`Listen` → 成功 |
| 有进程持有监听句柄（不 accept） | `Dial` → **成功**（连接进入 backlog） |

因此「陈旧文件」本身不会犯病：老代码 `Dial 失败 → Remove → Listen` 能接管（下面脚本对 pass273 也是 PASS）。真正犯病的是**还能被连上的 socket**：崩溃时若监听句柄被存活进程继承（WebView2 子进程等），或者残留进程仍握着监听，老代码会 `Dial` 成功 → 直接当成「已有实例」→ 写完参数后等待 2s 超时 → **仍然 return false**，于是静默退出。

确定性复现（不依赖真实崩溃时序）：先用一个只监听、从不 accept 的孤儿进程占住 socket，再启动应用。

| 构建 | 孤儿监听占位时的启动结果 |
|---|---|
| pass273（未修复） | `exit 0`，无窗口（缺陷复现） |
| 本轮修复后的临时构建 | 窗口出现，标题 `Whale Isle` |

真实序列（修复后临时构建）：启动（窗口）→ `Stop-Process -Force`（socket 留在磁盘）→ 再启动 → 窗口出现；期间第二次启动在有实例运行时仍然 exit（参数转交保留）。

## 3. 根因

`third_party/mygo/single_instance.go` 原来把「Dial 成功」当作「已有实例在运行」的唯一证据，握手只是单向写参数，随后 `io.Copy` 直到读超时，而无论是否有人读走、是否超时都 `return false`。一个没有活所有者的 socket（崩溃残留 / 被继承的监听句柄）同样能接受连接，于是被误判为活实例，启动静默失败。

## 4. 修复（最小正确改动，纯 Go，无 cgo、无平台文件）

[single_instance.go](<C:/AI/WhaleIsle-MyGo/third_party/mygo/single_instance.go>)：

- 新增所有者握手：接受方读走参数后立即回写 `ok\n`（`secondInstanceAck`），**先应答再交给主线程**，慢应用也不会被当成死实例；`handOverSecondInstance(conn, msg, timeout)` 只有拿到这条应答才返回 true（参数被读走并确认）。
- 第二实例：`Dial` 成功但握手拿不到应答（超时/EOF/写失败）时，不再退出，而是记录 `the single-instance socket has no live owner; taking it over` 并走接管路径（`Remove` + `Listen`）。
- 若接管后路径仍被存活进程占用（`WSAEADDRINUSE` 五次重试都失败），**仍然启动**并记录 `starts without the single-instance socket`：这一次的转发不可用，但绝不会出现「无窗口静默退出」。下一次该进程消失后自动恢复单实例。
- 其余行为不变：活实例握手成功时仍 `return false`（转交参数 + `OnSecondInstance`/深链），退出时按 `os.SameFile` 只删自己的 socket。
- 没有引入平台方法，因此不需要 windows/linux/darwin/fake 分支；非 Windows 平台行为同样是「无应答即接管」，语义一致。

## 5. 证据与回归

确定性复现（表见 §2）用的孤儿监听 + 两个构建；真实序列用修复后临时构建跑通「启动 → 强杀 → 再启动」。另加 [scripts/single-instance-stale-repro.ps1](<C:/AI/WhaleIsle-MyGo/scripts/single-instance-stale-repro.ps1>)：临时 profile + 强杀 + 再启动，输出 `{firstLaunch, staleSocketLeft, secondLaunch, ok}` 并在无窗口时 exit 1（对修复后构建输出 `ok: first 'Whale Isle', second 'Whale Isle'`）。

新增测试 [single_instance_test.go](<C:/AI/WhaleIsle-MyGo/third_party/mygo/single_instance_test.go>)：

- `TestSecondInstanceHandshakeAcknowledged`：活所有者的参数被读走并回 ack → true（转交）。
- `TestSecondInstanceHandshakeRejectsSilentOwner`：只监听不应答、以及读走但不回 ack 两种「看起来像实例」的 socket → false，并在规定 deadline 内返回。
- `TestSilentListenerNeverClaimsTheLock`：面对静默监听整条 `RequestSingleInstanceLock` 必须返回 true（启动），不会把自己当第二实例。
- 既有 `TestSingleInstance` / `TestSingleInstanceIdentifier` 仍通过（活实例转交与标识隔离未回归）。

命令（`third_party/mygo` 是独立模块，需在其目录内执行）：

- `go test . -count=1` → ok，4.4s；`go vet .` → 退出 0。
- `CGO_ENABLED=0 GOOS=linux|darwin|windows go build .` → 三者均退出 0。

## 6. 残余

- **版本混跑**：旧版所有者不会回 ack。升级期间旧实例在运行时，新版本第二启动会被当成「无活所有者」而另开一个窗口（一次性，之后正常）。这是有意的取舍：宁可多开窗口，也不静默失败。
- **存活进程仍持监听句柄**：接管方无法 bind 同名路径，本次启动可以显示窗口但没有参数转发，直到该进程退出；日志会说明。
- 真实崩溃的继承者（WebView2 子进程）无法在无 GUI 的测试环境里稳定重放，所以证据是「孤儿监听」这一等价机制 + 真实强杀序列，未覆盖所有崩溃形态。
- 未提交、未改安装版、未构建 passN exe；验证用的临时构建在 `.devdata/tmp-si/`（已清理）。

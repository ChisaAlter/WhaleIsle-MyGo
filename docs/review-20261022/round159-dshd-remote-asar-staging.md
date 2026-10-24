# Round159 · 从安装版 asar 组装可用的 dshd-remote（手机远程）并实跑 daemon

Date: 2026-10-25. Task: task-35. 目标：解除 round157 的 G1（`runtime/vendor/dshd-remote` 只有发布包布局 → `ResolveResources` 报 `missing runtime resource: src/main/dshd-remote.js`），让 MyGo 的远程 daemon 真正在 loopback 上跑起来并验证配对链路。

约束遵守：安装版只读（未改 `C:\软件\Whale Isle` 任何文件）；未动 `runtime/vendor/deepseek-harness` 与 `third_party/mygo`；未开 LAN/防火墙（daemon 两个监听都绑 127.0.0.1）；未留下监听/进程；无 commit、无 passN exe。

## 1. 结论

- **G1 已解除**：`runtime/vendor/dshd-remote` 现在满足 `phoneremote.ResolveResources` 的完整清单，daemon 能启动。
- 实跑（mode=lan、127.0.0.1:3499、TLS off）：**first start 28.3 s 到 `ready`**，`listening=true`、`relayConnected=false`、**PairingURL 有值**（round158 的 LAN 修复在真实 daemon 上生效）、QR 图已生成；**second start 2.0 s** 直接复用已 stage 的树。
- 监听：`127.0.0.1:3499`（daemon）与 `127.0.0.1:3180`（手机 SPA）都只绑 loopback；`GET http://127.0.0.1:3180/` → **200**、17,013 B、`<title>Whale Isle · 手机远程</title>`；daemon 根 `http://127.0.0.1:3499/` → 404（它不是 SPA，正常）。
- 关键前提：**必须用安装版 Electron 的 Node（ABI 149）跑 daemon**（`ELECTRON_RUN_AS_NODE=1`），因为随包的 `better-sqlite3` 是按 Electron ABI 149 编译的，而 `runtime/node` 是标准 Node ABI 137。

证据：`docs/review-20261022/evidence/remote-daemon-live.json`（源/哈希、命令、时间线、状态、HTTP 形状；配对 token 只记录形状不落盘）。

## 2. 可复现的 staging recipe

输入（全部只读）：

| 来源 | 用途 |
| --- | --- |
| `C:\软件\Whale Isle\resources\app.asar`（77,657,424 B） | `src/main/*`、`src/shared/*`、`mobile/web/*` 的打包内容 |
| `C:\软件\Whale Isle\resources\app.asar.unpacked\src\{main,shared}` | asar 里标记为 unpacked 的 10 个文件（`dshd-daemon-runner.mjs`、`dshd-daemon-hooks.mjs`、`dshd-host-tunnel.js`、`dshd-mux-sse.js` 等，asar 内是 0 字节占位） |
| `runtime/vendor/dshd-remote/node_modules`（安装版发布包自带） | 全部依赖，含 `@chisacode/server`（dist/server/server/*.js）与 `better-sqlite3` 的 ABI 149 原生产物 |
| `C:\软件\Whale Isle\Whale Isle.exe`（245,687,296 B，Electron 40 / Node 24.21.0 / ABI 149） | 以 Node 模式执行 daemon 的 `node.exe`（硬链接） |
| `C:\软件\Whale Isle\{icudtl.dat,*.pak,*.bin,*.dll,locales}` | Electron-as-node 在安装目录外启动所需的 Chromium 数据文件 |
| `reference/electron/src/main/dshd-remote.js`（36,578 B） | 仅作对照；安装版是更新的构建（42,291 B），**实际使用 asar 内容** |

步骤（PowerShell，仓库根）：

```powershell
$stage = "C:\AI\WhaleIsle-MyGo\tmp\dshd-remote-extract"
Remove-Item $stage -Recurse -Force -ErrorAction SilentlyContinue
# 1) asar 解包（tmp/asar_extract.mjs：读 header JSON + offset，无依赖）
node tmp\asar_extract.mjs 'C:\软件\Whale Isle\resources\app.asar' $stage 'src/main'
node tmp\asar_extract.mjs 'C:\软件\Whale Isle\resources\app.asar' $stage 'src/shared'
node tmp\asar_extract.mjs 'C:\软件\Whale Isle\resources\app.asar' $stage 'mobile/web'
# 2) 覆盖 unpacked 的条目（否则 dshd-daemon-*.mjs 是 0 字节）
$dst = "C:\AI\WhaleIsle-MyGo\runtime\vendor\dshd-remote"
Copy-Item "$stage\src" -Destination $dst -Recurse -Force
Copy-Item "$stage\mobile" -Destination $dst -Recurse -Force
Copy-Item 'C:\软件\Whale Isle\resources\app.asar.unpacked\src\main\*'   "$dst\src\main"   -Force
Copy-Item 'C:\软件\Whale Isle\resources\app.asar.unpacked\src\shared\*' "$dst\src\shared" -Force
# 3) daemon 用 __dirname/../../vendor/chisacode-remote/node_modules 解析 @chisacode/server
New-Item -ItemType Directory -Force "$dst\vendor\chisacode-remote" | Out-Null
cmd /c mklink /J "$dst\vendor\chisacode-remote\node_modules" "$dst\node_modules"
# 4) Electron-as-node：node.exe 硬链接到安装版 exe，并把 Chromium 数据文件放在旁边
New-Item -ItemType HardLink -Path "$dst\node.exe" -Target 'C:\软件\Whale Isle\Whale Isle.exe'
Copy-Item 'C:\软件\Whale Isle\icudtl.dat','C:\软件\Whale Isle\resources.pak',
          'C:\软件\Whale Isle\chrome_100_percent.pak','C:\软件\Whale Isle\chrome_200_percent.pak',
          'C:\软件\Whale Isle\v8_context_snapshot.bin','C:\软件\Whale Isle\snapshot_blob.bin',
          'C:\软件\Whale Isle\d3dcompiler_47.dll','C:\软件\Whale Isle\ffmpeg.dll',
          'C:\软件\Whale Isle\vulkan-1.dll','C:\软件\Whale Isle\vk_swiftshader.dll',
          'C:\软件\Whale Isle\vk_swiftshader_icd.json','C:\软件\Whale Isle\dxcompiler.dll',
          'C:\软件\Whale Isle\dxil.dll' $dst -Force
Copy-Item 'C:\软件\Whale Isle\locales' $dst -Recurse -Force
```

结果树（`runtime/vendor/dshd-remote`，7,189 文件 / 364,496,423 B，gitignored）：`src/{main,shared}`、`mobile/web`、`node_modules`、`vendor/chisacode-remote/node_modules`（junction → `..\..\node_modules`）、`node.exe`（硬链接）+ Chromium 数据文件。

验证（等价于 `phoneremote.ResolveResources` 的 16 项清单，全部命中）：

| 文件 | 字节 |
| --- | --- |
| `src/main/dshd-remote.js` | 42,291（sha256 `87402fda…`） |
| `src/main/dshd-daemon-runner.mjs` / `dshd-daemon-hooks.mjs` | 5,531 / 3,715 |
| `src/main/mobile-web-server.js` / `mobile-web.js` | 3,454 / 2,072 |
| `mobile/web/index.html` | 17,013（sha256 `00e269d6…`） |
| `src/shared/{lan,offer,official-deepseek-env,dshd-host-tunnel,dshd-mux-sse}.js` | 10,517 / 1,043 / 2,808 / 18,063 / 10,355 |
| `vendor/chisacode-remote/node_modules/@chisacode/server/dist/server/server/{exports,pairing-offer,relay-device-credential-store}.js` | 2,380 / 2,360 / 8,589 |
| `node_modules/better-sqlite3/build/Release/better_sqlite3.node` | ABI 149（sha256 `5471d232…`） |
| `node.exe` | Electron 硬链接（sha256 `068962b2…`） |

## 3. ABI 分析（本轮最关键的技术点）

| 运行器 | `process.versions.modules` | 结果 |
| --- | --- | --- |
| `runtime/node/node.exe`（标准 Node v24.21.0） | 137 | `better-sqlite3` 加载失败：`The module … was compiled against a different Node.js version using NODE_MODULE_VERSION 149. This version of Node.js requires NODE_MODULE_VERSION 137` |
| 安装版 `Whale Isle.exe` + `ELECTRON_RUN_AS_NODE=1` | **149** | `better-sqlite3` 正常；daemon 全链路可用 |

`node-pty` 在 137 下可加载（其产物兼容 N-API），真正卡住的是 `better-sqlite3`（设备/配对凭据存储）。Electron-as-node 需要 ICU/pak/snapshot 与 DLL 在其 exe 同目录，所以第 4 步把它们复制进了 stage（**未改安装目录**）。`node.exe` 用硬链接，避免再复制 245 MB；安装版升级后需重建硬链接。

## 4. 实跑结果（`tmp/probe-remote-live/main.go`，调用产品包 `phoneremote`）

命令：`$env:ELECTRON_RUN_AS_NODE=1; go run ./tmp/probe-remote-live`，config：
`remoteEnabled=true, remoteMode="lan", remotePort=3499, remoteBindAddress="127.0.0.1", remoteLanTls=false, remoteAppBaseUrl="", harnessOrigin/Cookie=""`。

时间线（`Runtime.Snapshot()` 采样，状态变化才记录）：

| t | status | listening | relayConnected |
| --- | --- | --- | --- |
| 0.0 s | `starting` | false | false |
| **28.3 s** | **`ready`** | **true** | **false** |

- `PairingURL`：`http://127.0.0.1:3180/#offer=…`（`#offer=` 后 468 B，offer v2，`relay: {endpoint: "ayase.cn:443", useTls: false}`，`expiresAtMs=1792866267575`）；QR PNG 已生成（`pairingImage=true`）；`bindAddresses=[192.168.53.103, 192.168.0.1]`（来自原生 `shared/lan.js` 的 LAN 发现）。
- 监听：`127.0.0.1:3499`（daemon）、`127.0.0.1:3180`（手机 SPA）——都是 loopback，没有 LAN/防火墙暴露。
- `GET http://127.0.0.1:3180/` → **200**，`text/html; charset=utf-8`，17,013 B，`<title>Whale Isle · 手机远程</title>`；`GET http://127.0.0.1:3499/` → 404（daemon 根不是 SPA，预期）。
- **第二次 Start**：2.0 s 到 `ready`（同一 stage 树，无重新解包），`pairingURLPresent=true`。
- 停止后：3499/3180 均关闭（`listenersAfterStop` 全 false），无残留 daemon 进程。

## 5. 对产品路径的要求 / 残留

1. **环境要求**：真实 app 启动远程 daemon 时需要 `ELECTRON_RUN_AS_NODE=1` 在进程环境中（否则 `ResolveResources` 解析到的 `node.exe` 虽能启动 Electron，但没有 Node 模式；或改用标准 Node 则需要 ABI 137 的 `better-sqlite3`）。建议下一轮在 `phoneremote.spawnBridge` 里显式设置该 env（当 stage 的 `node.exe` 是 Electron 时），或提供一个包装脚本作为 `node.exe`。本轮未改产品代码。
2. **打包**：`scripts/build-mobile-runtime.ps1` 仍是正规（联网）路径，产出标准 Node ABI 的 `runtime/vendor/dshd-remote`（含 `node/node.exe` 与 `node-build.json`）；本 recipe 是离线本地 unblock，只写在 gitignored 的 `runtime/` 下。
3. **硬链接语义**：`node.exe` 指向安装版 exe 的同一 inode；安装版升级/重装后需要重建硬链接（脚本可重复执行）。`debug.log` 是 Electron 运行时写的小文件，可忽略。
4. **配对 token 敏感**：evidence 只记录 URL 形状（scheme/host/port/offer 版本/过期时间），不落盘 token；`PairingURL` 本身只活在内存。
5. **未验证**：0.0.0.0 绑定（本轮按约束保持 loopback）、真实手机扫码完成配对/会话代理、relay 模式（需要 token + 外网）。这些仍归 round157 的 G3/G5/G6。

## 6. Round160 补记（task-39）：把 ELECTRON_RUN_AS_NODE 接进产品路径 + 真机验证结果

### 6.1 环境变量接线（已落地）

`internal/desktop/phoneremote/runtime.go` 的 `spawnBridge` 现在给 daemon 子进程显式设置环境：

```go
cmd.Env = daemonEnv(os.Environ())
```

`daemonEnv`（runtime.go）：

- 父环境**逐项原样复制**（不重排、不删除），仅在缺少 `ELECTRON_RUN_AS_NODE` 时追加 `ELECTRON_RUN_AS_NODE=1`；
- key 比较用 `strings.EqualFold`，Windows 上显式写成 `electron_run_as_node=0` 也不会被覆盖、不会重复；
- 标准 Node 忽略该变量（实测 `runtime/node` 在变量存在时仍以 ABI 137 正常运行），所以无条件补值对普通 Node 是安全 no-op；
- 不打印任何环境内容（`spawnBridge` 仍 `cmd.Stderr = io.Discard`），token 不进日志。

测试：`internal/desktop/phoneremote/spawn_env_test.go` —— 缺失时补 1、长度 +1、父环境逐项保持、输入 slice 不被改写、显式 `0/false/空`（两种大小写）优先且不重复、空环境也补 1。

**产品代码路径证明**（不依赖 shell 变量）：`tmp/probe-remote-live` 改为 `os.Unsetenv("ELECTRON_RUN_AS_NODE")` 后直接调用 `phoneremote.Start`：`starting → ready`，`listening=true`、`relayConnected=false`、PairingURL 有值，手机 SPA `GET 127.0.0.1:3180/` → 200，二次 Start 2.4 s 复用同一 stage，停止后 3499/3180 关闭。也就是说：**daemon 能跑起来的原因已经是产品代码补的环境变量，而不是 shell 设置**。

### 6.2 真机（app）验证：阻塞在缺少“启动时拉起远程”的调用

用临时构建（`whaleisle-remote-app.exe`，跑完即删，未覆盖任何 passN）+ 一次性 profile（`remoteEnabled=true, remoteMode=lan, remotePort=3499, remoteBindAddress=127.0.0.1, remoteLanTls=false`，workspace 临时目录，welcome 标记，真实 `runtime/`），shell **不设置** `ELECTRON_RUN_AS_NODE`：

| 步骤 | 结果 |
| --- | --- |
| 首次尝试：exe 放在 `tmp\` | 启动器报 `vendored harness not found; set WHALEISLE_VENDOR or run scripts/build-harness.ps1` —— `paths.VendorDir()` 先找 exe 同目录的 `runtime/vendor`，临时构建必须放在仓库根（或设 `WHALEISLE_VENDOR`） |
| 放仓库根后启动 | 正常进入 launcher → main 1440×920；harness `127.0.0.1:3099` 由 staged runtime 的 node 拉起（`runtime\node\node.exe …deepseek-harness\apps\cli\lib\bin.js`） |
| 远程监听 | **3499/3180 始终没有出现**，也没有 `dshd-remote` 子进程 |
| `--settings` → 远程 → 点“局域网”+“刷新” | 面板正常渲染（连接方式/端口/监听范围/传输加密/刷新），两次点击都成功，但面板只显示 `refresh the phone runtime before managing devices`，daemon 仍未启动 |

根因（file:line）：`refreshPhoneRemote`（`cmd/whaleisle/main.go:763`）只被 UI 动作调用 —— `main.go:5134`、`5141`、`5225`、`5281`、`cmd/whaleisle/phone_gateway.go:49`；**没有在 harness Ready / 启动完成时调用一次**。因此 `remoteEnabled=true` 的配置不会在开机时拉起网关（原生 `remote.js` 是随 config 同步启动的），本轮 app 内验证被这个产品缺口挡住，和 env 修复无关。

建议的下一处改动（scope 属 `cmd/whaleisle`，本轮未改）：在 harness `Ready` 之后（或主窗口首次渲染完成时）若 `cfg.RemoteEnabled` 则 `go state.refreshPhoneRemote()` 一次；随后 6.1 的 env 修复即可让 app 内真正出现 3499/3180 与配对码。

### 6.3 证据与残留

- 证据：`docs/review-20261022/evidence/remote-app-live.json` —— env helper/测试、无 shell 变量的 spawn 路径结果（ready/SPA/二次启动/清理）、app 运行记录（两次尝试、settings 面板、点击结果、无监听/无子进程）、afterKill 全 false。
- 残留：app 内 daemon 启动需 §6.2 的 boot 调用；relay 模式、0.0.0.0、真机扫码仍未验证；打包仍走 `scripts/build-mobile-runtime.ps1`（在线）；临时构建必须与 `runtime/` 同级。

### 6.4 Round161 补记（task-40）：boot 时拉起网关（已落地并实机验证）

**接线点**：`cmd/whaleisle/main.go` 的 `refreshBoot` 在 `harness.Ready` 分支里（持 `s.mu`）调用新方法 `claimPhoneBootStartLocked()`，解锁后执行一次 `go s.refreshPhoneRemote()`：

- `claimPhoneBootStartLocked`：`phoneBootStarted` 一次性门闩；`quitting`、`cfgStore == nil`、`cfg.RemoteEnabled == false` 都直接返回 false（不消耗门闩）；只有“第一次 Ready + 已启用 + 未退出”才置位并返回 true。
- 语义：每次会话最多一次；harness 重启/恢复模式再次 Ready 不会重启 daemon；用户之后开关远程仍走既有 UI 路径（`refreshPhoneRemote`，禁用时会 `manager.Stop()`）。
- 测试：`cmd/whaleisle/phone_boot_test.go` —— `TestClaimPhoneBootStartLocked`（enabled 只成功一次、重复 Ready 不重启、disabled 永不、quitting 跳过、门闩状态正确）与 `TestPhoneBootStartActionRunsOutsideTheGate`（决策不直接执行动作；第二次 Ready 不再触发）。`refreshBoot` 本身不在单测里跑：其尾部经 `mygo.RunOnMain` 等待事件循环，单测没有循环会挂住（已在测试注释里写明）。

**实机证据**（`docs/review-20261022/evidence/remote-app-boot.json`，临时构建 `whaleisle-remote-boot.exe` 放仓库根，跑完删除；shell 未设 `ELECTRON_RUN_AS_NODE`；一次性 profile：`remoteEnabled=true / lan / 3499 / 127.0.0.1 / lanTls=false`，临时 workspace，welcome 标记，真实 `runtime/`）：

| t | 3099 | 3499 | 3180 | 窗口 |
| --- | --- | --- | --- | --- |
| 2 s | – | – | – | 启动器 1060×660 |
| 5 s | – | – | – | 主窗 1440×920 揭示（启动器仍在） |
| 10 s | **✓** | – | **✓** | 启动器关闭 |
| **12 s** | ✓ | **✓** | ✓ | ready（主窗可见） |

- `GET http://127.0.0.1:3180/` → **200**、`Whale Isle · 手机远程`；kill 后 `afterKill` 的 3099/3499/3180 全为 false，进程树清理干净。
- 对比：同一脚本在 task-39（无 boot 接线）跑满 150 s 仍无 3499/3180；现在 12 s 就出现 → 证明是本次接线生效，而不是环境或 staging 的偶然。
- 限制：该次 run 在 14 s 就结束了（脚本以“listeners + shell ready”为达标），daemon 生成 pairing offer 约需 28 s，所以 **app 内的配对码字符串没有抓到**；同一 spawn 路径的 ready+PairingURL 已在 `evidence/remote-daemon-live.json` 证明（`firstStart.status=ready`、`pairingURLShape` 有 `#offer=`、过期时间）。下一轮若要 app 内配对断言：listeners 出现后再 settle ~35 s，然后打开 `远程` 面板读 QR/链接。
- 复跑注意：强制杀掉的临时 app 会留下 `%TEMP%\mygo-*.sock`（task-20 的已知缺陷），下一个实例会“交接给死实例”并静默退出；本次复跑就被它挡过，清理 socket 后恢复。临时构建必须与 `runtime/` 同级。

### 6.5 Round163 补记（task-41）：app 内配对 offer / QR 证明

**新增的可观测钩子**（cmd/whaleisle/phone_diag.go，默认关闭）：设置 `WHALEISLE_REMOTE_DIAG=1` 时，app 把 PhoneRuntime 快照写成 token-free 记录 `%WHALEISLE_USER_DATA%\runtime\remote-pairing.json`（status/listening/relayConnected/hasPairingURL、URL 形状 scheme/host/port/#offer=、offer v2 版本与过期时间、relay endpoint/useTLS、QR 是否存在、URL 的 sha256），**绝不写 token 本身**；`refreshPhoneRemote` 在 Start/Refresh 后写一次，并由一个 1 s 采样的有界 goroutine 跟随 starting→ready 的转换（remote 停止时退出）。单测 `cmd/whaleisle/phone_diag_test.go` 覆盖形状字段、hash 存在、文件中不含 token/完整 offer、未设 env 时不写文件。

**实机证据**（`docs/review-20261022/evidence/remote-pairing-app.json`；临时构建 `whaleisle-remote-pairing.exe` 放仓库根，跑完删除；一次性 profile `remoteEnabled/lan/3499/127.0.0.1`，临时 workspace，welcome 标记，真实 `runtime/`，shell 未设 `ELECTRON_RUN_AS_NODE`，仅 `WHALEISLE_REMOTE_DIAG=1`）：

| t | 3099 | 3499 | 3180 |
| --- | --- | --- | --- |
| 2 s | ✓ | – | – |
| 11 s | ✓ | – | ✓ |
| 12 s | ✓ | ✓ | ✓ |

settle 38 s 后读取 `remote-pairing.json`：

```json
{"status":"ready","listening":true,"relayConnected":true,
 "hasPairingURL":true,"pairingURLSha256":"b89f620c…11cb",
 "scheme":"http","host":"127.0.0.1","port":"3180","fragmentPrefix":"#offer=",
 "offerVersion":2,"offerExpiresAtMs":1792868170466,
 "relayEndpoint":"ayase.cn:443","relayUseTls":true,
 "expiresAt":"2026-10-24T18:56:10Z","pairingImage":true}
```

- 手机 SPA：`GET http://127.0.0.1:3180/` → **200**、`Whale Isle · 手机远程`。
- QR：`pairingImage=true`；`phone.go:146` 的渲染条件（pairingURL + ready + listening + relayConnected + 未过期）在这次 run 全部成立，因此 远程 面板会绘制二维码与“复制/打开配对链接”。
- 清理：kill 进程树后 3499/3180 立即关闭，3099 在 1.2 s 内关闭（复查全为 0），无残留进程/临时目录。

**残留**：本轮 UIA 抓 远程 面板只拿到 4 个节点（点击后采样太早，面板尚未绘制），面板文字/QR 像素未截取——但快照字段已证明渲染条件成立；若需要像素证据，下一轮在点击“远程”后多等 3-5 s 再抓 UIA/截图。另：本次 LAN run 的 `relayConnected` 为 true（daemon 同时持有 relay 会话），纯 LAN 无 relay 的路径由 `lan_pairing_test.go` 覆盖；relay 模式离线失败（task 可选）未跑。

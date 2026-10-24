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

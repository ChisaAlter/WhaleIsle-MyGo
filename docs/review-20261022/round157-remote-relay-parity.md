# Round157 · 远程/中继（dsh-remote / dshd-remote）功能对齐审计

Date: 2026-10-25. Task: task-32. Scope: 只读审计 + 本地离线行为取证；**没有改任何产品代码**（结论见 §4：本轮可验证的分歧都在写作用域之外，或需要真实 daemon 才能证明）。native 侧只读，未触碰安装版与真实 profile；未开监听、未改防火墙/LAN；`runtime/` 与 `third_party/mygo` 未改。

## 1. 结论速览

- **MyGo 的远程功能整体是“复用原生 dshd-remote 模块 + 自建控制桥”**（`internal/desktop/phoneremote/runtime.go` 起 Node 子进程跑 `runtime/vendor/dshd-remote/src/main/dshd-remote.js`，`bridge.cjs` 以 JSON 行协议收发），所以协议/会话代理/TLS/设备列表原则上与原生同源，差异集中在**配置映射、启动前置、LAN 配对门禁**。
- **本运行时下远程 daemon 根本起不来**：`ResolveResources` 要求原始 Electron 源码树，而落地的 `runtime/vendor/dshd-remote` 是安装版的“发布包布局”（只有 `node_modules` + 文档）。实测失败串见 §3。
- 两个**已确认的功能性 bug**（LAN 模式下二维码/配对链接永远不出现；relay 模式缺 host token）和一个**模式映射分歧**（`"ssh"`）见 §4，都给了精确 file:line 与最小修法；本轮未改（写作用域/可验证性原因）。

## 2. 功能矩阵（native vs MyGo）

Native 证据：`reference/electron/src/main/{config,remote,remote-tls,relay-client,remote-shell,remote-patch}.js`、`reference/electron/src/shared/{lan,offer,remote-devices}.js`。MyGo 证据：`internal/config/config.go`、`internal/desktop/phoneremote/*`、`internal/desktop/sshremote/*`、`cmd/whaleisle/main.go`、`internal/ui/settings/*`。

| 能力/设置 | Native | MyGo | 判定 | 证据 |
| --- | --- | --- | --- | --- |
| `remoteEnabled` 默认 | `false`，且被 `REMOTE_FEATURE_ENABLED=true` 二次门禁 | 默认 `false`（config.go:168） | 一致 | config.js:30,205；config.go:168,379-382 |
| `remotePort` 默认/范围 | 默认 `6767`；整数 1024–65535，否则回默认（config.js:31,236-239） | 同（config.go:169,381-383）；daemon 内部还有 3180 fallback（remote.js:23,329） | 一致 | 同上 |
| `remoteMode` 枚举 | 只有 `lan`/`relay`，其它一律 `relay`（config.js:235） | 接受 `lan`/`ssh`，其它 `relay`（config.go:384-386），注释还写了 `"relay", "lan", or "ssh"`（config.go:106） | **分歧（P2）**：`ssh` 不是手机远端的模式，native 会折叠成 relay | 见 §4 G4 |
| `remoteBindAddress` 默认/校验 | 默认 `127.0.0.1`（config.js:34）；仅 `127.0.0.1`/点分四段合法，否则回默认（config.js:191-200）。网关层另有 `0.0.0.0` 通配（remote.js:24-35） | 默认 `127.0.0.1`；`0.0.0.0`、点分四段（0–255）合法，否则回默认（config.go:53,438-461） | 一致（含 `0.0.0.0` 放行） | 同上 |
| `remoteLanTls` | 默认 false；仅 `mode==='lan'` 时生效（remote.js:333） | 默认 false；原样传给 daemon（config.go:172；main.go:786） | 一致（生效判断在 daemon 内） | config.js:35,241；main.go:786 |
| relay endpoint/URL | 空/legacy → 内建 `ayase.cn:443`；`remoteRelayEndpoint` 与 `remoteRelayUrl` 同步；`useTls` 对内建 relay 强制 true（config.js:216-224） | 同样归一化：空/legacy → `defaultRelayEndpoint`，强制 `RemoteRelayUseTls=true`，两字段同步（config.go:390-398） | 一致 | 同上 |
| `remoteAppBaseUrl` | 默认 `''`；legacy/默认 SPA 地址回空串（config.js:226-229） | 同（config.go:398-403） | 一致 | 同上 |
| 内置 relay 常量 | `ayase.cn:443`，`DEFAULT_RELAY_ORIGIN=https://ayase.cn`，SPA `https://ayase.cn/dshd`，legacy `125.124.85.212:8411`/`:3389/dshd`（shared/lan.js:7-22） | 同（config.go:45-53） | 一致 | shared/lan.js:7-22；config.go:45-53,390-398 |
| Host token / relay 认证 | 网关把 `config.remoteRelayToken` 作为 relay host token（remote.js:318），`config.remoteToken` 作为本机 token（remote.js:328） | **cmd 组装的 runtimeConfig 不含 `remoteRelayToken`/`remoteToken`**（main.go:782-789），只在设置页以凭据形式保存（credentials.go:58-59） | **缺失（P1）**：relay 模式无法认证 | 见 §4 G3 |
| relay 连接细节 | `RelayClient`：wss 握手 10 s、重试 1 s、非 loopback 必须 https（relay-client.js:20-21,100-127） | 由 dshd-remote 模块内部负责（同源），MyGo 不重复实现 | 一致（委托） | relay-client.js；bridge.cjs:52-58 |
| LAN 地址发现 | `listLanCandidates/listLanAddresses`：过滤 WSL/Hyper-V/Docker/VPN 等虚接口，QR 优先真实网卡（shared/lan.js:31-62） | 无独立发现；只用 daemon snapshot 的 `addresses`（bridge.cjs:82；runtime.go:283-284） | **缺失（P2）**：本运行时实测 `binds=[]` | 见 §3 |
| 配对 URL / QR | 网关 `snapshot()` 总是按 mode 构造 `urls[].pairingUrl`；`relayDown` 只标记 relay 模式（remote.js:327-360） | 桥 `usableURL()` 要求 `s.listening && s.relayConnected`（bridge.cjs:21），Go 侧还要求 `Listening && RelayConnected` 才置 `ready` 并保留 URL（runtime.go:204,292） | **分歧（P1）**：LAN 模式无 relay → 二维码/链接永远被清空 | 见 §4 G2 |
| 配对 offer 校验 | offer v2 + `authBootstrap.expiresAtMs`（shared/offer.js；remote.js 侧 pinning fp） | `offerExpiry()` 校验 v2/serverId/key/token/expires，且限制 128 KiB/URL 形状（runtime.go:314-340） | 一致（更严） | runtime.go:314-340 |
| 设备列表/改名/解绑 | `createDevice/normalizeDevices/publicDevices`（shared/remote-devices.js） | `devices.go` + `device_actions.go`，带 fixture（device-actions.fixture.cjs / device_actions_test.go） | 一致（复用同一快照字段） | devices.go, device_actions.go |
| 会话列表 / HTTP+WS 代理 | remote.js 自带代理：hop-by-hop 头、gzip、cookie/redirect 重写、WebSocket upgrade、登录页（remote.js:48-260） | 交给 dshd-remote：桥注入 `harnessOrigin`/`harnessCookie`（bridge.cjs:57-58；main.go:788） | 一致（委托），但**依赖 daemon 可启动**（§3） | 同上 |
| LAN TLS 物料 | `remote-tls.js`：ECDSA P-256 自签、10 年、CN `Whale Isle Remote`、30 天前续期、指纹进 offer（remote-tls.js:26-31） | 无本地实现（daemon 负责） | 未验证（委托） | remote-tls.js |
| SSH tunnel（远程工作区） | `remoteWorkspaceEnabled` 默认 true（config.js:60-63）；SSH 传输在 dsh-remote overlay / workspace-authority（asar 内） | `internal/desktop/sshremote`：tunnel/known_hosts/probe/registry + loopback 测试；与原生无同名模块可直接对比 | 无法离线比对（P2） | sshremote/*.go；config.js:60-63 |
| 设置页入口 | renderer 远程设置分段 | `settings.SectionRemote`（navigation.go:90，`DeviceOutline` 图标）+ 凭据行 | 一致（入口）/页面内容在 cmd 内，未审计 | navigation.go:90；credentials.go:58-59 |

## 3. 离线行为取证：daemon 起不来（精确失败）

探测程序（`tmp/probe-remote/main.go`，gitignored，直接调用产品包）：

```go
res, err := phoneremote.ResolveResources(userData, `C:\AI\WhaleIsle-MyGo\runtime\vendor`)
rt.Start(userData, vendor, map[string]any{"remoteEnabled": true, "remoteMode": "lan",
    "remotePort": 3499, "remoteBindAddress": "127.0.0.1", ...})
```

输出：

```
resources: node="" module="" home="" err=missing runtime resource: src/main/dshd-remote.js
snapshot: status="missing" listening=false relay=false message="missing runtime resource: src/main/dshd-remote.js" binds=[]
```

根因（file:line）：

- `internal/desktop/phoneremote/runtime.go:75-107` 要求 `vendor/dshd-remote` 下是**原始 Electron 源码树**（`src/main/dshd-remote.js`、`src/main/mobile-web-server.js`、`mobile/web/index.html`、`vendor/chisacode-remote/node_modules/@chisacode/server/...`、`node/node.exe`）。
- 现状：`runtime/vendor/dshd-remote` 来自安装版 `resources/vendor/dshd-remote`，只有 `node_modules` + `LICENSE/NOTICE/AGPL-SHIPPING.md/WORKER-CHECKLIST.md`，**没有 `src/`、`mobile/`、`node/`**；round146 前的 `runtime/vendor.pre-rc2-backup-20261024/dshd-remote` 也一样（本 checkout 从未构建过移动运行时）。
- 原始 daemon 确实存在：安装版 `app.asar` 里 `/src/main/dshd-remote.js`（42,291 B，asar 头解析确认），源码树 `reference/electron/src/main/dshd-remote.js`（36,578 B）+ `mobile/web/` + `vendor/chisacode-remote/`。
- 仓库的正规构建路径 `scripts/build-mobile-runtime.ps1` → `scripts/mobile-runtime.mjs prepare/bundle`：需要 `npm ci --ignore-scripts`、`npm rebuild esbuild better-sqlite3 node-pty`、`npm run build:server`（**联网 + 原生 ABI 重建**），产出 `runtime/vendor/dshd-remote`（含 `node/node.exe`、`node-build.json`）。

**因此远程/配对/二维码在当前离线环境无法端到端跑通**；本轮没有伪造“启动成功”，也没有监听任何端口。要真正验证：需在有网环境跑 `build-mobile-runtime.ps1`（或 stage 一份带 native 模块的移动运行时），再用 `remoteMode=lan`/`127.0.0.1:3499` 起 daemon 抓 `/`、`/pairing` 页面与 `#offer=` 报文。

## 4. 已确认的分歧与最小修法（本轮未改）

### G1（P0）移动运行时未 stage → 远程整体不可用

- 现象/证据：§3。
- 修法：`scripts/build-mobile-runtime.ps1`（联网）产出 `runtime/vendor/dshd-remote`，或从安装版 asar 解出 `src/main/*` + `mobile/web/*` 并配套预编译的 `vendor/chisacode-remote/node_modules`（含 better-sqlite3/node-pty 的 Node ABI 产物）。属运行时物料问题，不是 Go 缺陷。
- 影响：远程设置页只能显示 “missing”；会话列表/配对/设备管理全部不可用。

### G2（P1）LAN 模式下二维码/配对链接永远被清空

- Native：`remote.js:327-360` 不管 `relay.connected` 都会返回 `urls[].pairingUrl`；`relayDown` 仅用于 relay 模式的提示。
- MyGo：`bridge.cjs:21` `usableURL()` 直接 `if (offerBlocked || !s.listening || !s.relayConnected) return ''`；`runtime.go:204`（Snapshot 清空）与 `runtime.go:292`（只在 `Listening && RelayConnected` 时置 `ready`）同样要求 relay。
- 修法（最小、可测）：桥与 Go 都改为 `listening && (relayConnected || mode === 'lan')`，Go 侧把 `mode` 从 `Start/Refresh` 的 config 记到 Runtime（`lanMode bool`），并在 `runtime_test.go` 补一条 fake child 事件（`listening=true, relayConnected=false, mode=lan, 合法 offer URL`）断言 `Status=="ready"` 且有 `PairingURL`；relay 模式仍要求 relay 连接。

### G3（P1）relay 模式缺 host token / relay URL

- Native：网关用 `config.remoteRelayToken`（remote.js:318）与 `config.remoteToken`（remote.js:328）；两者都存在 credentials 层（config.js:473-474）。
- MyGo：`cmd/whaleisle/main.go:782-789` 的 `runtimeConfig` 没有 `remoteRelayToken`/`remoteToken`（也没有 `remoteRelayUrl`，不过 daemon 有 `remoteRelayEndpoint || remoteRelayUrl` 兜底，见 dshd-remote.js:571/617/675）。凭据只在设置页保存（internal/ui/settings/credentials.go:58-59）。
- 修法：`refreshPhoneRemote` 组装 config 时注入 `remoteRelayToken`/`remoteToken`（来自 credentials 快照），并在 token 变化时触发一次 refresh。写作用域是 `cmd/whaleisle`，本轮未动。

### G4（P2）`remoteMode` 接受 `"ssh"`

- Native `config.js:235` 只有 `lan`，其它一律 `relay`；MyGo `config.go:384-386` 明确保留 `"ssh"`，`config.go:106` 注释也写 `"relay", "lan", or "ssh"`。SSH 是远程工作区（`remoteWorkspaceEnabled`）的能力，不是手机远端 gateway 的模式。
- 修法：`switch c.RemoteMode { case "lan": default: c.RemoteMode = "relay" }`（或保留 ssh 时在 daemon 侧显式映射为 relay），并改注释。`internal/config` 不在本轮写作用域。

### G5（P2）LAN 地址发现缺失

- Native `shared/lan.js:31-62` 自己枚举 IPv4 并过滤虚拟网卡，QR 用真实网卡地址；MyGo 只回显 daemon snapshot 的 `addresses`（`bridge.cjs:82`、`runtime.go:283-284`），本运行时实测 `binds=[]`。若 daemon 不提供地址，远程设置页的“可访问地址”会为空。修法：在 `phoneremote` 内复用 native 过滤规则实现 `listLanCandidates`（后续轮次）。

### G6（P2）SSH tunnel / TLS 物料未验证

`sshremote` 有完整实现与 loopback 测试（`tunnel_loopback_test.go`、`known_hosts_test.go`），但没有可对照的原生同名模块（SSH 传输在 dsh-remote overlay / workspace-authority，位于 asar），协议一致性与 `remote-tls.js` 指纹固定（fp）行为需要真实 daemon + 手机端才能验证。

## 5. 验证

- `go test ./internal/desktop/phoneremote ./internal/desktop/sshremote ./internal/ui/settings -count=1` → `ok 3.106s / 2.122s / 13.243s`。
- `go vet` 同三个包 → 无输出（clean）。
- 未启动任何监听：探测使用 temp userData，`ResolveResources` 在启动前即失败；进程已退出（探针为 `go run` 前台进程），无残留 socket/端口。临时目录自动删除。

## 6. 下一轮建议（按优先级）

1. **G1**：在有网环境跑 `build-mobile-runtime.ps1` 或 stage 移动运行时（这是远程一切功能的前置）。
2. **G2 + G3**：LAN 配对门禁 + relay token 注入；两者都有明确修法与可离线单测（fake child / config 组装）。
3. **G4**：模式枚举与注释对齐 native。
4. **G5/G6**：LAN 发现与 TLS/SSH 的真实设备验收。

# 2026-10-26 收口轮：剩余差异 + 防弹窗打包 + 整体验收

本轮在 2026-10-25 十类差距修复的基础上做了三件事：补齐仍披露的两处行为差异、把运行时防弹窗修复重新打进完整 Windows 安装包、跑一轮可复现的整体验收。**本轮不宣称与鲸屿本体完全一致**；未完成的像素/真机/硬件项在最后一节逐条列出。

## 1. 本轮补齐的差异

### 1.1 桌宠选屏与坐标（task-13）

- 恢复参考实现的**固定名义中心优先**选屏：以锚点所在显示器的 DPI 换算固定 120×130 DIP 偏移，不再让角色可视宽高或目标 DPI 反向影响选屏。
- 用合成边界扫描复现旧做法的六轮振荡（同锚点 -150 在中心 30/-30 之间来回），新做法 21,636 次边界检查 0 振荡；旧轨迹保留在日志中。
- 修复真实跨屏放置失败：MyGo 的 `WM_WINDOWPOSCHANGING` 在 `movable=false` 时强制 `SWP_NOMOVE`，导致程序自身 `SetWindowPos` 只改尺寸不改原点；现只在同步自有 placement 期间临时允许移动并 defer 恢复，不改 `third_party`。
- 真实双屏（D1 2560×1440@150%，D2 (2560,0) 1920×1080@100%）实测：两屏都得到 **240×260 CSS**；D1 物理角色 360×390、DPR 1.5、viewport 1707×960；D2 物理 240×260、DPR 1、viewport 1920×1080；边界连续放置 8 次 + 回程 4 次稳定，`IsMovable` 仍为 false。
- 统一漫游/重力/释放速度/页面速度的物理像素与 CSS 单位换算（150DIP → 225px、65DIP/s → 97.5px/s、重力 1400 → 2100），位置 marker 重开不重复缩放。
- 证据：`tmp/pet-coordinate-check.log`、`tmp/pet-coordinate-gui-result.json`；pet 全包 4.458s、命令包 29.013s、reference 2/2、渲染 5/5。文档：`docs/features/pet-renderer-parity.md`、`docs/features/pet-quick-chat.md`。独立只读复核无 must-fix。

### 1.2 会话级插件贡献（task-12）

- 新增 session 模式：通过真实 `uiWorkspace.openSession` 进入原会话，沿原 `SessionProvider`/`SlotRegistry` 渲染贡献；缺 owner、未选中会话、Host generation/cookie 变化时清除座席与动作，不创建替代表单。
- 界面上是一个局部 browser seat（top-level overlay + DialogBase，自带裁剪与 modal 遮挡），**不是逐行 inline 原生绘制**；composer 由原 owner 在座席内完整接管，原生草稿不双写。
- 真实 Host fixture 通过：注册 → 会话 header 座席出现 → `Session.rename` 回写并经 `useProjection('title')` 回显 → 切换会话后旧 page/revision/epoch 动作被拒 → withdraw 撤销 → 缺 owner 空 surface。未 mock Session 或 bridge。
- Lead 在 `cmd/whaleisle/main.go` 只加两处接线：`syncPluginSession`（紧随 `syncPluginContributions`）与 `renderPluginSessionSlot`（`workspaceNavigationView`+`directoryPickerView` 之后）。
- 证据：`tmp/plugin-session-functional.log`、`tmp/plugin-session-real-20261026/session-2883241719/result.json`、`tmp/plugin-session-final-check.log`（pluginui 2.846s / pluginmanager 9.923s / cmd 28.428s）。文档：`docs/features/native-plugin-slots.md`。
- 边界：tool/composer 等其他贡献仅在原 owner 已挂载对应 outlet 时可选，未逐类独立 E2E；不是任意槽位的全原生等价。

### 1.3 防弹窗修复进入安装包

此前的完整包不含 2026-10-25 晚的隐藏控制台修复。本轮把该修复（`processutil.BackgroundTree` + 六处后台树根 + Git/taskkill 保持 `CREATE_NO_WINDOW`）随正式构建一起打包。

## 2. 完整安装包

| 项 | 值 |
| --- | --- |
| 安装包 | `tmp/release-chain-full-package-20261025/WhaleIsle-Setup-0.2.1-rc.2-consolefix.exe` |
| 大小 | 293,890,576 字节 |
| SHA256 | `0512b3d5f97d06253253c2fef9e218843746341b4ff71cfcff104333b3a7dceb` |
| blockmap | 4,485 块（由新字节重算） |
| 内嵌桌面程序 | `e2ba60bdf832d497fca1313fccb9191bd1d19edb8f63f942f381a14630773f30`（96,824,832 字节，`-X main.buildVersion=0.2.1-rc.2`） |
| preview / 链接 helper | `810051c6…` / `3a03300e…`（均未变） |
| catalog | `WhaleIsle-MyGo-releases.json` 单条 schema2，只指向 `-consolefix.exe` |

旧包 `WhaleIsle-Setup-0.2.1-rc.2-final.exe`（293,892,796B / `53ee3662…`，**不含防弹窗修复**）及其 6 个 sidecar 已归档到 `archive-pre-console-fix-20261026/`，逐文件哈希前后一致，未删除。

构建证据：`tmp/release-chain-console-setup.log`（NSIS 编译 1345.5s、payload 1,087,349,845B、156,085 instructions、25,698 实体 include/3,146 links）、`tmp/release-chain-console-catalog.log`（strict wholegraph + 实际字节比对 + Go `DecodeNativeCatalog`/`Validate(true)` + mobile `ResolveResources` 全 PASS，23.0s）、`tmp/release-chain-console-versioned-stage-receipt.json`（appVersionGate 绑定 mac GREEN receipt SHA `b9cb31f4…`）。

项目根目录两个启动入口（`whaleisle.exe`、`DeepSeekHarness Go.exe`）也已更新为同一个版本化构建 `e2ba…`（`tmp/acceptance-final/update-entries.log`，替换前后哈希记录在案；仅覆盖未被占用的目标），`whaleisle-preview.exe` 保持 `810051c6…`。

## 3. 整体验收结果

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 正式代码根整包测试（cmd/internal/tools/scripts） | **整包并发跑中 3 个包失败，机器空闲单独复跑全部 PASS** | `tmp/acceptance-final/roots.log`（失败原始）、`tmp/acceptance-final/rerun-light.log`（phoneremote 15.401s、market 11.121s）、`tmp/acceptance-final/rerun-importer.log`（importer 172.02s） |
| 失败归因 | `importer` 的 NSIS smoke 在并发 LZMA 压缩 + 全根测试下超过其内部 120s 等待；`phoneremote` live LAN 与 `market` SVG 渲染同样只在负载下失败 | 三者在机器空闲、单独运行时全 PASS；未修改产品代码或放宽测试 |
| 全根 vet | PASS | `tmp/acceptance-final/roots.log` |
| Windows CGO=0 GUI 构建 ×4（主程序/preview/integration/launcher） | PASS，PE Subsystem=2 | `tmp/acceptance-final/builds.json` |
| 打包应用真实启动（e2ba，stage 相邻 runtime） | PASS：Ready 7.944s、版本 0.2.1-rc.2、无 override | `tmp/acceptance-startup/startup-settled-90.log` |
| 关闭与资源释放 | PASS：Ready 后 90.116s 精确 WM_CLOSE，5.726s 自然退出 code0，forced=false，Job[]、两个端口关闭 | 同上 receipt |
| 窗口事件 | owned 可见终端 0、未归属新增终端 0、unresolved 0、hookError 0、主窗口 SHOW 1 且可见 | 同上 |
| 最终安装包字节 | PASS：size/SHA 一致；**4,485 块逐块重算全部匹配**；SHA256SUMS 与新 basename 一致 | `tmp/acceptance-final/artifact.log`、`artifact.json` |
| catalog 与 Go 消费 | PASS：单条 schema2、sha/size/name 一致、Go `DecodeNativeCatalog`+`Validate(true)`、mobile `ResolveResources` | 同上 |
| 隔离安装事务（GUID 身份、非产品注册） | PASS：静默安装 → 当前字节 `e2ba…` 校验 → 同一安装器再次安装（升级，无残留退役树）→ 静默卸载，安装根/退役根/注册表项均消失 | `tmp/acceptance-final/nsis-transaction.log` |
| 独立复核（mac-native，只读） | PASS：独立重算 size=293,890,576 / SHA256=`0512b3d5…` 且 hash 期间文件 length/mtime 未变；catalog 实读 schema2 单条，sha/size/URL basename 与 metadata、SHA256SUMS 一致；runtimeIncluded/windows/amd64/nsis/full/prerelease 与 Host `687f…`/`mygo-session-runtime-readiness-v1` pins 正确；未发现差异 | 该轮只读复核消息（未新建文件；未重算 4,485 块、未启动 app/setup） |

### 关闭行为的诚实说明

同一版本曾出现一次严格 FAIL：Ready 92.006s、Ready 后 12.6s 关闭、60s 未退出。随后两次诊断（Ready 后 66.4s 关闭、Ready 后 12.4s 关闭）都自然退出且无堆栈，**未能复现，也未取得“等待任务确认”的堆栈证据**。因此本轮只把它记为“一次未复现的早期关闭失败”，不推断为产品缺陷，也不宣称已定性根因。可交付场景（Ready 后后台工作静默、90s 再关闭）通过。

## 3.5 首启欢迎页与「预览版说明」已移除（用户要求）

2026-10-26 用户实测反馈：首次启动被「预览版说明」阻塞层挡住且点「继续」无响应，随后要求“不要欢迎页”。按此**移除三处首次启动 surface**：

1. 删除 `cmd/whaleisle/preview_notice_layer.go` 与 `mainView` 中的绘制调用（`internal/ui/chat/preview_notice.go` 组件与单测保留，生产不再调用）。
2. `welcomeRequired()` 恒为 false：不再弹出独立的登录/API Key 欢迎窗（`welcome_window.go` 实现保留，`welcome_window_test.go` 改为断言“不再隐式弹出”）。
3. `renderDesktopOnboarding()` 恒为 false：不再绘制主窗口内的「开始设置 / credits」引导页（控制器与账户数据保持不动）。
4. `config.PreviewNoticeAck` 字段保留以兼容旧配置。

新构建（最终）：SHA256 `985215c2a1b5cf27ec7b0102b09c5db9297e6909039416c4f0f9db251df68161`、94,322,176 字节、版本 0.2.1-rc.2；`internal/ui/chat`（87.603s）与 `cmd/whaleisle`（38.496s）测试通过（`tmp/acceptance-final/build-nonotice.log`）。项目根目录两个入口已换成该构建并重启：窗口 1440×920、进程 ready、Host 子进程与端口正常、`last-desktop-start` 记录 `ok=true` / `appVersion=0.2.1-rc.2`。

注意：**已交付的 `-consolefix.exe` 安装包仍含旧弹层与欢迎页**；如需安装包同步，需要重新压缩（约 22 分钟）。

## 4. 仍未完成 / 未验证
- `docs/acceptance-ui-parity.md` 的严格合同仍未满足：没有全应用每个控件 × 空/载/错/成功的穷举，也没有全部页面的同尺 baseline/actual/diff、阈值 0 像素与几何 ≤1 物理像素证据。
- 手机：无 Android 真机；LAN 发现、TLS/SSH、relay、断线重连的完整网络矩阵未测。
- macOS：无真机；Keychain ACL/TCC/音频设备、Apple SDK 与 GUI 未实测，仅 amd64/arm64 交叉构建与 vet。
- Windows 通知中心：未做真实点击与视觉模糊对照。
- 桌宠：OEM/其它缩放组合、连续拖拽甩动、持续 FPS/内存/功耗未测。
- 插件：tool/composer 等逐类 E2E 未做；会话座席是局部 seat，不是逐行 inline 原生。
- 安装包未签名、未上传、未用产品身份安装；catalog 中 HTTPS 地址只是目标位置。


## 5. 性能修复轮（2026-10-26 晚，perfround）

结论先行：**239MB 的新包不是有效安装包，而是 LZMA 压缩中途被终止后留下的截断文件；stage、打包清单与关键目录完整，没有漏文件。** 在当前文件上不能安装、不能上传、也不应生成 sidecar；必须重新完整压缩一次。

### 5.1 包身份与实测体积

| 项 | 值 |
| --- | --- |
| 安装包 | `tmp/release-chain-full-package-20261025/WhaleIsle-Setup-0.2.1-rc.2-perfround.exe` |
| 实际大小 | 239,147,553 字节 |
| SHA256 | `b37fa63e1d226c94507c883087a440e9150b339c4b5276d2025f25ec2ec94a22` |
| NSIS 块头 `length_of_all_following_data` | **0（未定稿）**；正常值应为文件长度 − 137,728 = 239,009,825 |
| 构建日志 | `tmp/release-chain-perfround-setup.log`，最后一行停在 `Compressed data:`，没有 `Total size` |
| 应有大小（同一 stage 完整重编实测） | 291,494,145 字节（旧 consolefix 293,890,576 字节 / 压缩率 26.74%；重编 payload 少 2,362,880 字节） |
| 缺失压缩尾部 | 52,346,592 字节（约 18%） |
| sidecar | blockmap / `final-installer-metadata.json` / `SHA256SUMS-final.txt` / catalog **均未生成** |

注：§2 表中的 `-consolefix.exe` 及相关 sidecar/收据（归档共 13 个文件）已整体移入 `archive-pre-perfround-20261026/`（逐文件哈希前后一致，未删除）；本轮 stage 的 preview 是新的 `7fe8f234…` / 33,202,688 字节，项目根目录的 `whaleisle-preview.exe` 仍是旧的 `810051c6…`（不在包内）。

判定依据（只读、可复现）：

1. NSIS 首部偏移 137,732 的签名之后，+20 字节是“后续数据总长度”。所有完成包该值都等于“文件长度 − 137,728”：`-final.exe` 293,892,796 → 293,755,068；`-consolefix.exe` 293,890,576 → 293,752,848；`WhaleIsle-Smoke-*`、`shellonly-check`、`uninstall-smoke-*` 全部相等。perfround 包该值为 **0**，说明压缩流写完后没有回填长度。
2. 同一缺陷也出现在归档的 `WhaleIsle-Setup-0.2.1-rc.2.exe`（254,876,193 字节，块头同为 0）；而 `-final.exe`、`-consolefix.exe` 两个归档包均为已定稿。可见这是构建被中断，不是压缩级别变化。
3. 归档步骤把旧包的 13 个 sidecar 全部移入 `archive-pre-perfround-20261026/`（逐文件哈希前后一致），新包的构建没有走到 metadata/blockmap/catalog 生成步骤。
4. 本轮用同一 stage 重新完整编译，实测 291,494,145 字节（921.1 秒）；交付文件比它少 52,346,592 字节，正好是缺失的压缩尾部。重编时也观察到安装包在 LZMA 压缩阶段以 0 字节创建、逐步写入，证明该阶段被终止就会留下这种截断文件。

### 5.2 体积差异逐项核对（没有漏文件）

| 关键载荷 | 文件数 | 字节 |
| --- | --- | --- |
| runtime（include 覆盖） | 25,698 | 928,321,379 |
| runtime/node（node.exe） | 1 | 93,580,104 |
| runtime/vendor（Host + 依赖 + mobile） | 25,696 | 831,672,859 |
| ├ 其中 `dshd-remote`（mobile runtime） | 7,191 | 365,412,421 |
| resources | 60 | 3,273,495 |
| pet-render-assets（THA4） | 10 | 38,610,327 |
| 根：whaleisle.exe / preview / ghostty / 许可 / 终端清单 | 5 | 129,836,486 |

- `runtime-files.nsh` 共 25,698 条 `File`，与 stage 的 25,698 个 runtime 文件**一一对应**：0 缺失、0 多余、0 重复目标、0 悬空源；字节数 928,321,379 与 stage/收据完全一致。
- 打包清单共 25,773 个 File 条目（runtime 25,698 + 根 5 + resources 60 + pet 10），源文件原始字节 1,100,041,687；NSIS 该次编译自报 `Install data` = 1,084,986,965 字节，差 15,054,722 字节来自 stage 内 2,621 组同内容文件（共 15,343,786 字节重复），属 NSIS 去重口径，不是漏文件。
- 与旧 consolefix 包相比，`Install data` 只少 2,362,880 字节，恰等于桌面程序差值 2,366,464 字节减去 preview 差值 3,584 字节；runtime / resources / pet 未重打，字节未变。
- 三包均为 NSIS 3.13、`SetCompressor /SOLID lzma`，LZMA 属性字节相同（8MB 字典，`5D 00 00 80 00`），**不存在压缩级别变化**。
- mobile runtime 没有少打包：`runtime/vendor/dshd-remote` 的 7,191 个文件 / 365,412,421 字节都在 include 与 stage 中；插件以 `node_modules/@deepseek-ai/dsh-plugin-*`、`cordis-plugin-*` 等 87 个包目录形式存在，均随 runtime 继承。
- 桌面程序 `whaleisle.exe` = `a175f15c…`（94,458,368 字节，`-X main.buildVersion=0.2.1-rc.2`），preview = `7fe8f234…`（33,202,688 字节），runtime helper = `3a03300e…`，三者都在 stage 与 include 中；项目根目录两个入口 `whaleisle.exe` / `DeepSeekHarness Go.exe` 也已是 `a175f15c…`。

### 5.3 隔离事务（GUID 产品名 + 同一 stage 全量重编）

因为交付文件是截断的，事务不能在它上面执行。为了区分“包坏了”还是“stage/脚本坏了”，用同一 stage、同一脚本重新完整编译了一个 GUID 产品名/APP_ID 的隔离安装包（不覆盖发布文件，输出在 `tmp/perfround-isolated-20261026/`）：

| 步骤 | 结果 |
| --- | --- |
| 重编完整包 | 291,494,145 字节，耗时 921.1 秒；与完整包应有的量级一致，说明 stage 与压缩参数均正常 |
| 静默安装 | exit=0，239.4 秒；安装目录内 `whaleisle.exe` = `a175f15cb18d673c948d50fa7863fd09cab9ea2fe0b28d2f4dc36edace4da6e5`（= a175f15c…） |
| 安装树逐文件对比 | 安装 25,773 个文件 / 1,100,041,687 字节，与 stage 的 25,773 个 File 条目完全一致；missing=0 / extra=0 / 内容不符=0；整树摘要 `ddd8f095dec01f936e05cdc48592797e16f29370d7e670435b6689ddb0ea81fb` |
| 同一安装器再装（升级） | exit=0，390.8 秒；exe 哈希仍为 a175…；退役树 0 个，文件数 25,773（补充检查确认无 `_wi-iso-*` 残留） |
| 静默卸载 | exit=0，172 秒；安装根删除=True、注册表删除=True、桌面快捷方式删除=True、开始菜单目录删除=True |
| 与用户 Electron 版隔离 | 真实 `ai.deepseek.harness.gui` 键前后均不存在；Electron 卸载键（`af06f756-…`，Whale Isle 0.3.10）前后一致；`C:\软件\Whale Isle` 的 31,551 个文件 / 1,238,327,017 字节中，测试窗口内被写入 0 个 |

事务结论：`PASS-isolated-full-install-upgrade-uninstall`。证据：`tmp/perfround-isolated-transaction.log`、`tmp/perfround-isolated-20261026/transaction-*.json`、`tmp/perfround-isolation-supplement.json`、`tmp/perfround-stage-inventory.json`、`tmp/perfround-package-facts.json`、`tmp/perfround-archive-inventory.json`。

### 5.4 修复建议（未擅自重打发布包）

1. 重新执行完整 NSIS 压缩，超时给足（本轮完整重编 921.1 秒；consolefix 实测 1,345 秒；建议 ≥2,700–3,600 秒），不要在中途杀进程。
2. 压缩结束后先验收块头：`length_of_all_following_data == 文件长度 − 137,728`，日志出现 `Total size:`；大小应在 291–294MB 量级。
3. 再生成 blockmap / `final-installer-metadata.json` / `SHA256SUMS-final.txt` / schema2 单条 catalog（runtimeIncluded=windows/amd64/nsis/full/prerelease，Host `687f282e…` / `mygo-session-runtime-readiness-v1`），重算约 4,450 个 64KiB 块的哈希，并重跑 Go `DecodeNativeCatalog`/`Validate(true)` 与 mobile `ResolveResources`。
4. 当前的 239,147,553 字节文件保留原处（未删除、未改名），但不要安装、上传或用它生成 sidecar。

## 6. 最终可交付包（2026-10-26 深夜，perfround2）

结论先行：**按 §5.4 的四条要求重做了一遍：从当前工作树重建 stage、完整重新压缩、重新生成全部 sidecar、再用 GUID 隔离身份跑完四步事务。`-perfround2.exe` 是有效安装包（块头已定稿、日志有 `Total size:`），`-perfround.exe`（239,147,553B）原样保留且仍不可交付。**

### 6.1 包身份、体积与块头

| 项 | 值 |
| --- | --- |
| 安装包 | `tmp/release-chain-full-package-20261025/WhaleIsle-Setup-0.2.1-rc.2-perfround2.exe` |
| 体积 | 291,585,042 字节 |
| SHA256 | `4a7dce448bccffb1d3dbada0d3d54313075ecbc5f9dd898701d6d606f0e0a2c6` |
| NSIS 块头 `length_of_all_following_data`（偏移 137,752） | **291,447,314 = 291,585,042 − 137,728**（已定稿） |
| 构建日志 | `tmp/perfround2-20261026/nsis-final-compile.log`；`Compressed data: 291,447,310 / 1,096,253,350`、`CRC (0x25AC606E)`、`Total size: 291,585,042 / 1,096,292,266 bytes (26.5%)` |
| NSIS 耗时 | 765.8 秒（超时给到 5,400 秒，未中途终止） |
| Install data | 1,085,056,597 字节（距 §5 实测的 1,084,986,965 只差 69,632 字节，正是 app/preview 的新旧差值） |
| sidecar | `<setup>.blockmap.json`（4,450 × 64KiB，逐块重算全部匹配）、`final-installer-metadata.json`、`SHA256SUMS-final.txt`、schema2 单条 `WhaleIsle-MyGo-releases.json`、`native-release-config-perfround2.json` |
| 与截断包对比 | `-perfround.exe`：239,147,553B / declared=**0** / 日志停在 `Compressed data:`；本轮：291,585,042B / declared=291,447,314 / 日志有 `Total size:` |

### 6.2 stage（从当前源码重建，不只是替换 exe）

| 项 | 值 |
| --- | --- |
| stage | `tmp/release-chain-full-package-20261025/whaleisle` |
| 构建链 | `scripts/package.ps1` 全量：桌面/preview/integration 构建、resources、ghostty、THA4、Host runtime release、Node、mobile 校验、物理 include |
| 构建耗时 | 792.8 秒 |
| app | `da198fb62e3fd1c865383c86ae13a8885cea839b36858b7b1fd208b77894defb` / 94,509,056B / `-s -w -X main.buildVersion=0.2.1-rc.2 -H windowsgui` / PE amd64 windowsgui / 版本字符串已嵌入 |
| preview | `b75041dd4c4a63efed0dd0c228c19d22ad2faa7f9028a698e354927ee0db2fcc` / 33,221,632B |
| integration（不进包） | `6690e8da9d8faeb958bf53de162e21f727370b52dfc72eb28c998e07f946a918` |
| runtime-links helper | `3a03300e188775869478d36191397b64bb16eb83d8f7d38e29a03ddd59d724e5`（沿用） |
| runtime | 25,698 文件 / 928,321,379 字节（Node 1 + vendor 25,696；graph 25,695 files / 3,146 links），与旧 stage 逐文件一致 |
| include | `runtime-files.nsh` 6,899,252B / `ccfe6bed5bb3e569ea75e00ec6dc3fbbd87d25923fe80c365dc7b4829466120a`，与上一轮**逐字节相同**；25,698 条 `File` 与 runtime 文件一一对应（missingTargets=0 / notIncluded=0 / notOnDisk=0） |
| resources | 60 文件 / 3,273,495 字节（digest `afc876e6…`，与旧 stage 一致） |
| pet-render-assets | 10 文件 / 38,610,327 字节（`f7c3bb7b…`） |
| 载荷 | 25,776 文件 / 1,102,111,233 字节（含 stage 根 8 个文件） |
| junction 修复 | 新 stage 从临时构建目录移动到正式路径后，3,146 个绝对 junction 全部重写为正式路径；随后 `session-runtime.mjs verify` 与 mobile `verify` 均 PASS，include 重新生成后与旧文件逐字节相同 |
| 启动验收 | `tmp/perfround2-20261026/startup/run-11d4d669/receipt.json`：Ready 13.882s、Ready 后 90.232s 精确 WM_CLOSE、5.808s 自然退出、exit 0、owned 可见终端 0、未归属终端 0、主窗口 SHOW 1、版本 `0.2.1-rc.2` |
| 旧 stage 归档 | `archive-pre-perfround2-20261026/whaleisle-stage-pre-perfround2`（旧 app `a175f15c…` / preview `7fe8f234…` / include 副本相同）；用 lstat 逐文件核验 runtime：25,698 / 928,321,379B，与 include 目标逐项一致（digest `83f894d8…`，missing=0） |

### 6.3 sidecar 与消费者校验

- blockmap：4,450 个 64KiB 块，从最终字节逐块重算 SHA256，**全部匹配**；`fileSize=291,585,042`。
- `SHA256SUMS-final.txt`：`4a7dce44…  WhaleIsle-Setup-0.2.1-rc.2-perfround2.exe`。
- catalog：schemaVersion=2、单条；`runtimeIncluded=true`、`windows/amd64/nsis/full`、`prerelease=true`；packageID `v0.2.1-rc.2/windows/amd64/full/runtime`；Host `687f282e8625c6b8d11ff58211a90524be2e4317` / `mygo-session-runtime-readiness-v1`（与之前一致）。
- Go `DecodeNativeCatalog` + `Validate(true)`（`internal/desktop/update`）PASS；mobile `ResolveResources`（`internal/desktop/phoneremote`）PASS；严格 wholegraph 发布器 19.1 秒。
- 独立 artifact 校验 16 项全 PASS（`tmp/perfround2-20261026/artifact.log`、`perfround2-artifact.json`），含块头、逐块 blockmap、include 1:1、载荷计数、staged 二进制钉住。

### 6.4 GUID 隔离四步事务

用同一 stage、同一脚本、GUID 产品名（`whaleisle.perfround2.1bab9c0976f3`）重新压缩一个隔离安装包（291,544,272B / `0e25cd52…` / 686.3s，块头同样定稿），事务只对它执行；交付包字节在事务前后不变（`4a7dce44…`）。

| 步骤 | 结果 |
| --- | --- |
| 静默安装 | exit=0，163.5s；安装目录 `whaleisle.exe` = `da198fb6…`（= 本次 stage app），preview = `b75041dd…` |
| 安装树逐文件对比 | 安装 25,773 文件 / 1,100,111,319B，与 stage 载荷完全一致；missing=0 / extra=0 / differing=0；整树摘要 `432860d0…` 相同 |
| 同一安装器再装（升级） | exit=0，298.2s；exe 仍为 `da198fb6…`；退役树 0 个；文件数 25,773 |
| 静默卸载 | exit=0，118.7s；安装根删除、注册表项删除、桌面快捷方式删除、开始菜单目录删除 |
| 与 Electron 版隔离 | 真实 `ai.deepseek.harness.gui` 键前后均不存在；Electron 卸载键（Whale Isle 0.3.10）与 `C:\软件\Whale Isle` 31,551 文件 / 1,238,327,017B 指纹前后一致；自本轮开始以来该目录被写入文件数 = 0 |

事务结论：`PASS-isolated-full-install-upgrade-uninstall`。证据：`tmp/perfround2-20261026/transaction.log`、`tmp/perfround2-20261026/isolated/transaction-1bab9c0976f3.json`。

**一处需要说明的缺陷与补证**：主事务收据里 Electron 目录字段被 Windows PowerShell 5.1 按 GBK 误解码成乱码路径，导致该收据中的 `electronDir` 值不是真实目录（该收据因此不是合法 JSON，已原样保留）。随后用按码点构造路径的脚本重新做了完整补证：安装同一隔离包 → 校验 exe → 卸载 → 比较前后指纹与注册表快照（`tmp/perfround2-20261026/perfround2-isolation-supplement.json`，结论 `PASS-electron-isolation-supplement`：31,551 文件 / 1,238,327,017B，指纹 `01047323…` 前后相同，注册表快照相同，会话开始后被写入 0 个文件）。补证安装 exit=0、卸载 exit=0、安装根与注册表均清理。

### 6.5 根目录入口

两个入口已同步为本次打包版 `da198fb6…` / 94,509,056B / PE subsystem 2，替换前备份在 `tmp/perfround2-20261026/root-backup/`。**注意**：替换前根目录的 `whaleisle.exe` 实际是 112,605,696B / `bb994747…` 的 console 子系统探针构建，并非任务描述中的 `a175f15c…`；`DeepSeekHarness Go.exe` 才是 `a175f15c…`。本次两者都换成了打包版。根目录 `whaleisle-preview.exe` 不在第 5 步要求内，仍为旧的 `810051c6…`。

### 6.6 保留的历史产物

`-perfround.exe`（239,147,553B / `b37fa63e…`，块头 0，不可用）、`archive-pre-perfround-20261026/WhaleIsle-Setup-0.2.1-rc.2.exe`（254,876,193B / `a38825b4…`，同样截断）、`archive-pre-perfround-20261026/WhaleIsle-Setup-0.2.1-rc.2-consolefix.exe`（293,890,576B / `0512b3d5…`）、`archive-pre-console-fix-20261026/WhaleIsle-Setup-0.2.1-rc.2-final.exe`（293,892,796B / `53ee3662…`）全部原样保留；清单与哈希见 `tmp/perfround2-20261026/perfround2-preservation.json`。旧 sidecar 与旧 stage 均未删除。

## 7. 最终包（2026-10-27，final；toast 动效对齐 + 全量重打包）

结论先行：**任务 A 的 toast 动效已改为 160ms 上滑 8 DIP + 淡入、停留 3000ms、淡出 1000ms，默认寿命 4000ms；任务 B 从当前工作树重建 stage、完整 NSIS 压缩、重新生成全部 sidecar，Go/mobile 消费、GUID 隔离四步事务和最终 stage 启动验收全部 PASS。`-final.exe` 块头已定稿，旧包原样保留。**

### 7.1 toast 动效（先做，改完立即测试）

| 项 | 修改前 | 修改后 |
| --- | --- | --- |
| 默认寿命 | 4000ms | 4000ms = hold 3000ms + fade 1000ms |
| 停留（满不透明） | 无独立值 | 3000ms（`toastHold`） |
| 退出淡出 | 200ms | 1000ms（`toastFade`） |
| 进入 | 无 | 160ms（`toastEnter`）上滑 `toastRise=8` DIP + 淡入（EaseOut） |
| 行动 toast | 8000ms | 8000ms（7s 停留 + 最后 1s 淡出，保持原调用方时长） |

实现位于 `third_party/mygo/ui/toast.go`：新增 `toastMotion(age,left)` 统一计算透明度与上滑位移；`buildToasts` 在进入/淡出阶段请求动画帧，在停留结束前 `After(left-fade)` 唤醒，淡出起点为 3000ms；根节点用 `OffsetAfterLayout` 做 8 DIP 位移，命中、绘制与无障碍几何同帧一致。`Timeout`、`AddToast`、`CloseToast`、`ToastAction` 的 API 与调用语义未变，只改时长与动效。

测试：`third_party/mygo` 模块内 `go test ./ui -run 'Toast|Motion' -count=1` → **ok，0.990s**；`go test ./internal/ui/... -count=1 -timeout=10m` → **全部 PASS**；绿灯套件 `go test ./internal/store/session ./cmd/whaleisle ./internal/harness ./internal/ui/... -count=1 -timeout=10m` → **全部 PASS**。新增 `TestToastMotionTiming` 断言 3000/1000/160/8 与总寿命 4000，`TestToastEnterRisesIntoPlace` 断言首帧低于落点 8 DIP、80ms 时仍在上升、160ms 到位且保持。根模块不能直接测试嵌套模块的 `./third_party/mygo/ui`，本轮在 `third_party/mygo` 内以 `./ui` 执行同一测试集。

**说明**：本 checkout 的 `Toast.module.css` 进入关键帧是 `translate(-50%,-6px) → 0`（向下 6px）。任务书明确要求“160ms 上滑（rise 8px）”，因此本轮按任务书的 8px 向上位移实现；若要严格照 CSS 的 6px 向下位移，只需改 `toastRise` 与位移符号。

### 7.2 包身份、体积与块头

| 项 | 值 |
| --- | --- |
| 安装包 | `tmp/release-chain-full-package-20261025/WhaleIsle-Setup-0.2.1-rc.2-final.exe` |
| 体积 | 291,548,253 字节 |
| SHA256 | `2682f61f6a401fe1462445f9bf3365eb234375e9c587227ab7bb2725a4760d69` |
| NSIS 块头 `length_of_all_following_data`（偏移 137,752） | **291,410,525 = 291,548,253 − 137,728**（已定稿） |
| 构建日志 | `tmp/final-20261027/nsis-final-compile.log`；`Compressed data: 291,410,521 / 1,096,256,934`、`CRC (0x00873EBD)`、`Total size: 291,548,253 / 1,096,295,850 bytes (26.5%)` |
| NSIS 耗时 | 630.2 秒（超时给到 4,200 秒，未中途终止） |
| Install data | 1,085,060,181 字节（较 perfround2 的 1,085,056,597 多 3,584 字节，正好等于新 app 的体积增量） |
| blockmap | `<setup>.blockmap.json`：4,449 个 64KiB 块，最后一块 44,125B；从最终字节逐块重算全部匹配，`fileSize=291,548,253` |
| sidecar | `final-installer-metadata.json`、`SHA256SUMS-final.txt`、schema2 单条 `WhaleIsle-MyGo-releases.json`、`native-release-config-final.json` |
| 与截断包对比 | `-perfround.exe`：239,147,553B / declared=**0** / 日志停在 `Compressed data:`；本轮：291,548,253B / declared=291,410,525 / 日志有 `Total size:` |

### 7.3 stage（从当前源码完整重建）

| 项 | 值 |
| --- | --- |
| stage | `tmp/release-chain-full-package-20261025/whaleisle` |
| 构建链 | `scripts/package.ps1` 全量：桌面/preview/integration、resources、ghostty、THA4、Host runtime release、Node、mobile 校验、物理 include；构建前临时移走 makensis，避免 staging 阶段误编译 |
| 构建耗时 | 689.6 秒 |
| app | `d2bf513a468ed31430541b3086e983cc2b923d0e863ffe0a000917302fc44ee3` / 94,512,640B / `-s -w -X main.buildVersion=0.2.1-rc.2 -H windowsgui` / PE amd64 windowsgui / 版本字符串已嵌入；用同一源码与 ldflags 复建得同一哈希（gofmt clean） |
| preview | `ef8dc2df32fec676d367fe4f1ae25fd036acbb2f5d806c6b943aa31e704b114f` / 33,221,632B |
| integration（不进包） | `6690e8da9d8faeb958bf53de162e21f727370b52dfc72eb28c998e07f946a918` |
| runtime-links helper | `3a03300e188775869478d36191397b64bb16eb83d8f7d38e29a03ddd59d724e5`（确定性重建，与旧值相同） |
| runtime | 25,698 文件 / 928,321,379 字节（digest `8d0ac45e…`，与旧 stage 逐文件一致；vendor 25,695 + Node 1 + helper/图） |
| runtime graph | 25,695 files / 3,146 links；3,146 个绝对 junction 从 scratch 路径全部重写到正式 stage 路径，逐链接验收后 `session-runtime.mjs verify` 与 mobile `verify` 均 PASS |
| include | `runtime-files.nsh` 6,899,252B / `ccfe6bed5bb3e569ea75e00ec6dc3fbbd87d25923fe80c365dc7b4829466120a`，与上一轮**逐字节相同**；25,698 条 `File` 与 runtime 文件一一对应（missingTargets=0 / notIncluded=0 / notOnDisk=0） |
| resources | 60 文件 / 3,273,495 字节（digest `afc876e6…`，与旧 stage 一致） |
| pet-render-assets | 10 文件 / 38,610,327 字节（`f7c3bb7b…`，与旧 stage 一致） |
| 载荷 | 25,776 文件 / 1,102,114,817 字节（含 stage 根 8 个文件）；隔离安装逐文件对比的 stage 载荷为 25,773 文件 / 1,100,114,903 字节（排除 `.runtime-staged`、`.pet-render-staged`、integration） |
| 旧 stage 归档 | `archive-pre-final-20261027/whaleisle-stage-pre-final`（旧 app `da198fb6…` / preview `b75041dd…`）；旧二进制与旧 include 另存 `archive-pre-final-20261027/*-pre-final.*` |
| 启动验收 | `tmp/final-20261027/startup/run-86a471f7/receipt.json`：Ready **6.422s**、Ready 后 90.188s 精确 WM_CLOSE、**5.785s** 自然退出、exit 0、owned 可见终端 0、未归属终端 0、主窗口 SHOW 1、hook 错误 0、版本 `0.2.1-rc.2` 匹配 |

### 7.4 sidecar 与消费者校验

- blockmap：4,449 个 64KiB 块，从最终字节逐块重算 SHA256，**全部匹配**；`fileSize=291,548,253`，最后一块 44,125B。
- `SHA256SUMS-final.txt`：`2682f61f…  WhaleIsle-Setup-0.2.1-rc.2-final.exe`。
- catalog：schemaVersion=2、单条；`runtimeIncluded=true`、`windows/amd64/nsis/full`、`prerelease=true`；packageID `v0.2.1-rc.2/windows/amd64/full/runtime`；Host `687f282e8625c6b8d11ff58211a90524be2e4317` / `mygo-session-runtime-readiness-v1`（未变）。
- Go `DecodeNativeCatalog` + `Validate(true)`（`internal/desktop/update`）PASS；mobile `ResolveResources`（`internal/desktop/phoneremote`）PASS；严格 wholegraph 发布器 18.8 秒。
- 独立 artifact 校验 **16 项全 PASS**（`tmp/final-20261027/artifact.log`、`final-artifact.json`），含块头、逐块 blockmap、include 1:1、载荷计数、staged 二进制钉住、catalog 身份。

### 7.5 GUID 隔离四步事务

用同一 stage、同一脚本、GUID 产品名（`whaleisle.final.eabbf9011ff7` / WhaleIsle Final eabbf9011ff7）重新压缩隔离安装包（291,523,416B / `dff95599…` / 627.4s，块头同样定稿），事务只对它执行；交付包字节在事务前后不变（`2682f61f…`）。

| 步骤 | 结果 |
| --- | --- |
| 静默安装 | exit=0，142.2s；安装目录 `whaleisle.exe` = `d2bf513a…`（= 本次 stage app），preview = `ef8dc2df…` |
| 安装树逐文件对比 | 安装 25,773 文件 / 1,100,114,903B，与 stage 载荷完全一致；missing=0 / extra=0 / differing=0；整树摘要 `e5fe0fd6…` 相同 |
| 同一安装器再装（升级） | exit=0，290.0s；exe 仍为 `d2bf513a…`；退役树 0 个；文件数 25,773 / 1,100,114,903B |
| 静默卸载 | exit=0，116.1s；安装根删除、注册表项删除、桌面快捷方式删除、开始菜单目录删除 |
| 与 Electron 版隔离 | 真实 `ai.deepseek.harness.gui` 键前后均不存在；Electron 卸载键（Whale Isle 0.3.10）与 `C:\软件\Whale Isle` 31,551 文件指纹前后一致（mtime/size 全量快照） |

事务结论：`PASS-isolated-full-install-upgrade-uninstall`。证据：`tmp/final-20261027/transaction.log`、`tmp/final-20261027/isolated/transaction-eabbf9011ff7.json`、`final-transaction.json`。

### 7.6 根目录入口

两个入口已同步为本次打包版 `d2bf513a…` / 94,512,640B / PE subsystem 2，替换前备份在 `tmp/final-20261027/root-backup/`；两处 SHA256 与 stage app 逐一相等（`final-root-sync.json`）。

### 7.7 保留的历史产物

`-perfround2.exe`（291,585,042B / `4a7dce44…`）、`-perfround.exe`（239,147,553B / `b37fa63e…`，块头 0，不可用）仍原样保留在 release 根目录；旧 sidecar 已原样归档到 `archive-pre-final-20261027/sidecars-pre-final/`，旧 stage 归档到 `archive-pre-final-20261027/whaleisle-stage-pre-final/`。旧包未删除、未改名、未被本次事务触碰。

### 7.8 执行说明

首次把“启动验收 + stage 收据合并”放在同一个隐藏任务里时，启动验收本身 PASS，但随后合并脚本在设置 `[Console]::OutputEncoding` 时因启动探针已 detach 控制台而报 “The handle is invalid”，任务以 exit 1 结束。改为在新隐藏控制台单独执行 `stage-accept.ps1` 后成功；这不是产品失败，启动收据与 stage 字节均未改变。其余所有长任务（stage 689.6s、final NSIS 630.2s、隔离 NSIS 627.4s、安装/升级/卸载 142.2/290.0/116.1s）均由隐藏运行器在给足超时下完成。

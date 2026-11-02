# 本轮差距实现与验证记录（2026-10-25）

本轮依据当前安装版 Whale Isle 0.3.10 的 Whale UI 运行产物、只读参考源码和已安装 rc.2 Host 运行包，对此前列出的主要实现差距进行修复。工作目录保留已有用户改动，未替换用户安装版、清空 dist 或访问真实账户凭据。构建与测试使用 `scripts/run-hidden.ps1` 的隐藏控制台和自有进程树。

本记录是实现与证据清单。`gap-matrix-20261025.md/.json` 为较早的历史快照，其中版本、行号、桌宠尺寸、托盘通知和运行包描述不能直接当作本轮状态。本记录不改写历史矩阵的计数，也不代替 `../acceptance-ui-parity.md` 的逐屏像素、全量控件状态和独立验收要求。该合同的签字未被本轮定向测试替代。

后续运行时弹终端修复已完成，见[终端审计最新补充](../windows-terminal-startup-audit-20261025.md)。根目录应用入口已更新为该修复版本；本记录下方的完整安装包保留其当时哈希，尚不包含后续隐藏进程树修复。

## 已实现与实测范围

| 能力 | 当前实现 | 已执行的证据 | 尚未证明的边界 |
| --- | --- | --- | --- |
| 手机端保存桌面设置 | 严格字段白名单和类型校验；IPv4 绑定、公开 HTTP(S) URL 与端口校验；`UpdateChecked` 提交候选深拷贝；无效混合补丁不写内存或磁盘 | `../../tmp/lead-remote-final.log`：配置、桥接与真实打包 Node 的 HTTP → bridge → desktop → Store 路径，合法值重新打开仍在，非法/部分保存被拒绝 | Android 真机、LAN 发现、TLS/SSH、relay 与断线重连的完整网络矩阵 |
| Whale UI v2 与对话面板 | 导入自有 MIT 0.3.10 产品；四种历史/当前语言；真实 `render_ui` 归属；组件引擎、状态、动作回执、导出分片、面板持久化与撤销；v2 离线 CSP；复用图片、地图、HTML 与原生视频适配 | `../../tmp/whale-ui-browser.log`：真实 Mermaid 节点、图片可见且解码、HtmlView capability 与脚本按钮、地图选择、视频 seat 回执、210 KiB 程序、390px 双主题；`../../tmp/whale-ui-check.log` 的 Go 四包结果及 owner 原生 helper 检查 | 系统保存对话框实点、远程 YouTube 实际播放、公共数据服务可用性、macOS GUI。较早 check 日志末尾保留的 HTML 测试字符串失败由单独最终 browser GREEN 收口 |
| Markdown 图文与增量内容 | 图片按正文顺序；统一平衡括号/带空格路径解析；本地文件行号链接；脚注、math 围栏；实际流式语法/着色输入有界策略 | `../../tmp/markdown-final.log`；`../features/markdown-parity.md` 中实际输入字节计数；聊天与 Files 作用域、草稿、分页和最终态回归 | 未宣称每帧总布局耗时有相同上界；流式跨边界脚注结束后重解析 |
| 行内原子引用 | 文本与 chip 共享形状布局、光标和选择；UTF-16 原子操作、原文复制、IME/撤销/拖动、持久化与真实文件入口 | `../../tmp/composer-check.log`；`../composer-reference-activation-20261017.md`；输入框整包与实际 NativeTextInputClient 行为测试 | 实机 TSF 候选窗、GPU、macOS/Linux 输入法和系统文件/图片剪贴板 |
| 桌宠与快捷聊天 | 单个活动显示器覆盖；240×260 CSS 舞台；Windows 物理坐标与旧位置迁移；248×360 DIP 快捷聊天；真实 Anime4K 与 THA4 常驻 GPU 输入、首帧验证和清理 | `../../tmp/pet-check.log`；`../../tmp/pet-gui-result.json` 与两张自有窗口图：150% 显示器、实际 WebGPU/capture/Anime4K 帧、缩放和透传样式 | 真正混合 DPI 多屏、持续 FPS/内存/功耗、物理空间速度、所有姿态像素对照和真实 Host 聊天。锚点优先选屏与参考中心优先的边界差异已在文档写明 |
| macOS 凭据 | 无 cgo 的原生 Security/CoreFoundation Keychain；原同步 v10 信封只读兼容；AES-256-GCM 保存与失败保护 | `../../tmp/mac-native-check-final.log`；`../../tmp/mac-native-app-check-both.log`；Windows 注入 ABI/所有权/失败测试，darwin amd64/arm64 整应用构建与 vet | 真 macOS Keychain、签名 ACL、cgo/Apple SDK 与用户授权界面；本轮未访问真实密钥 |
| macOS 语音 | purego 原生 AudioQueue/CoreAudio/AVFoundation；设备 UID、权限、取消、PCM/WAV 上限和资源生命周期 | 与上行相同的注入测试、两种 darwin 架构整应用构建与 vet；`../macos-voice-capture.md` | 真机麦克风、TCC、热插拔、系统驱动以及 Apple SDK 静态断言 |
| Windows 通知与玻璃 | 真实 WinRT XML/history/actions、COM 冷启动、生命周期和精确卸载；DWM Acrylic 与暗角；通知原始目标随安装身份事务保存 | `../../tmp/win-surfaces-check.log` 的原生和跨平台检查；`../../tmp/win-notification-nsis-check.log` 的实际编译/隔离 argv fixture；独立代码审查 | 通知中心 UI 物理点击与实际视觉模糊对照；未把 API 读回当作截图验收 |
| 插件动态贡献 | 原生插件卡/配置/动作/徽标/section/add/activation 接入真实 SlotRegistry；原页面保留表单和回调；局部 browser seat 保持完整视口、裁剪与遮挡；generation/subject/revision 撤销 | `../../tmp/plugin-ui-functional.log`：Win32 region/full viewport 1.08s、真实 Host + Chrome 20.02s；fresh profile 的十项结果在 `../../tmp/plugin-ui-real-20261025/run-3005010434/result.json`；源码冻结清单在 `../../tmp/plugin-ui-source-freeze.json` | 任意 React 控件不自动转换为 Go；session/tool/composer slots 仍需真实 session owner；sidebar/main/settings 保持既有 Host view。不能据此称任意插件全原生等价 |
| rc.2 运行包、更新目录与安装事务 | 严格 session/mobile ABI 和资源清单、闭合物理文件图、精确 NSIS include、可信链接 helper 与回滚补偿、可选样本缺省策略 | `../../tmp/release-chain-closeout-3-utf8.log` 前七个真实 Windows helper；`../../tmp/release-chain-final-fixtures-utf8.log` 的 JS 与真实临时安装/升级/强制回滚/卸载；`../../tmp/release-chain-samples-integration.log` | 完整 Windows 资源包、真实 catalog 与 Go 客户端消费均已通过；部分卸载重试有新增三项 helper 与实际 NSIS 重试证据并经独立复核。未签名、未上传、未发布，也未执行产品身份安装。Host 每次启动验证核心 pins/readiness，不重新哈希全体约 25k 文件 |

## 集成检查

正式代码根为 `cmd/...`、`internal/...`、`tools/...` 和 `scripts/...`。仓库内原有 scratch `tmp/` Go 包不作为发布代码根。首轮全根测试记录在 `../../tmp/lead-integration-final.log`；它保留了三项真实失败：并行编辑期间的链接 helper 编译、旧 NSIS 测试复用 dist、Git 初始化测试落在 checkout 内。

后三项均针对原因修复并重跑：

- 链接 helper 原七个真实 Windows 测试与实际 NSIS 事务成功，见前表。独立审查已复核中途 detach 补偿、hardlink 别名、未知 reparse tag、缺失链接和旧文件变更五条失败路径。随后发现的部分卸载重试已修为：仅 detach 且 alias 已不存在时允许缺失物理子树，所有现有祖先仍查 raw reparse，存在 alias 与其他命令仍严格。`../../tmp/release-chain-retry-check-3.log` 记录原七项加新三项 helper 全通过；实际 NSIS 首次清理保留未知 sentinel、已删除 owned 目标和 graph，产品重试依原宏安全跳过缺失 graph，移走测试 sentinel 后二次卸载成功。graph 仍存在但目标/父目录缺失的分支由三项真实 helper 用例覆盖，未混称实际 NSIS 用例覆盖。
- `../../tmp/lead-importer-smoke.log`：独立输出编译 151.31s、GUID 应用身份的真实 silent install/uninstall 149.37s，合计 301.963s。测试不复用旧 setup，不写真实产品注册或快捷方式；卸载后断言安装根、退役根和独立注册表项均消失，失败时不递归清理伪造成功。
- `../../tmp/composer-diffsurface-check.log`：初始化用例 0.22s、diffsurface 整包 11.317s。Go 1.27 的 `t.TempDir` 遵守 GOTMPDIR，原 fixture 在 checkout 内向上发现父 Git 仓库；修复仅隔离测试目录，未放宽 15s 等待或修改生产初始化行为。

`../../tmp/lead-vet-build.log` 和 `../../tmp/lead-frozen-integration.log` 记录所有正式根的 vet，以及主程序、预览和通知清理三个 Windows `CGO_ENABLED=0` GUI 构建。最终源码另通过 macOS amd64/arm64 的整应用无 cgo 交叉构建和相关原生集成 vet，见 `../../tmp/lead-final-darwin.log`；未执行 macOS 二进制。最终 UI 冻结后的受影响九包整包测试通过，包含聊天 61.085s、主应用 26.830s。最后发现的既有 `/api/wiki` fallback 白名单遗漏已窄修，`../../tmp/whale-ui-final.log` 的 request/reply 回归通过；`../../tmp/release-chain-refresh-binaries.log` 记录 NSIS 开始之前对打包 app/preview 的重新构建。app SHA256 为 `f6364e13632f8892144705cffb2338b6fb5bd516c4ae689f25942ec5e2f9940f`，preview 为 `810051c6338e4dcc8e3dc85e6c671b4941b32fc15d712bb41199826b3e63fc49`。修复前的 Lead 构建哈希保留为当时检查记录，不代表最终发布包。

## 完整包记录

> 2026-10-26 更新：本节的 `-final.exe` 已归档为不含防弹窗修复的历史包。当前可交付包是
> `WhaleIsle-Setup-0.2.1-rc.2-consolefix.exe`（293,890,576B，SHA256 `0512b3d5…`，桌面程序 `e2ba60bd…`），
> 详见[收口轮验收记录](acceptance-round-20261026.md)。

自有输出为 `../../tmp/release-chain-full-package-20261025/`。输入为已验证的独立 rc.2 stage，含 25,695 个物理文件与 3,146 个内部别名；终端库和 THA4 资源经固定哈希检查。`../../tmp/release-chain-full-package.log` 是最初 staging 与旧 helper 封装记录；旧压缩 Job 因 helper 已更新而被取消，目录中的无 `-final` 安装程序是保留的过期产物，不能交付。

`../../tmp/release-chain-final-setup.log` 记录重新构建 helper、完整 NSIS 编译和最终压缩成功。最终带 `-final.exe` 后缀的安装程序为 293,892,796 字节，SHA256 为 `53ee366210c8281345ede5d53046f6213a1da8879904d1a46090efa74a789319`。相邻 `final-installer-metadata.json`、`SHA256SUMS-final.txt` 与 4,485 块的 `.blockmap.json` 均从最终真实字节重新生成。`final-binary-source-hashes.json` 记录实际装包 helper 的 SHA256 `3a03300e188775869478d36191397b64bb16eb83d8f7d38e29a03ddd59d724e5` 及其源码、app 和 preview 哈希。

`../../tmp/release-chain-catalog.log` 记录发布器再次严格验证完整资源图后生成 schema 2 catalog，独立安装包元数据与 catalog 的 SHA256/size/文件名比对通过，Go `DecodeNativeCatalog` 与 `Validate(true)` 往返通过（整包 1.092s），实际打包 mobile `ResolveResources` 通过（用例 1.11s，整包 2.747s）。只有 `-final.exe` 被 catalog 引用。安装包只生成，未签名、未执行产品身份安装，也未上传或写发布服务器；catalog 中 HTTPS 地址仅是目标位置，不代表已存在远端发布资产。

`../../tmp/lead-closeout-integrity.log` 记录 33 项冻结源码与实际打包二进制哈希比对、最终 helper vet 和 tracked diff check 均通过。

`../../tmp/final-stage-startup/startup-settled.log` 记录最终 app 与相邻 runtime 的实际启动：fresh profile/workspace、空闲自有端口且无 Node/vendor override，约 7.18s 达到产品 Ready。Ready 后 12.021s 发送精确自有 HWND 的 WM_CLOSE，5.428s 自然退出 code 0；runner 尚未清理时自有 Job 已无子进程、两个监听端口均关闭，未强制清理。此前 Ready 后约 2s 就关闭的两轮仍记录未通过：10s/60s 内不退出。仅修改诊断 hook 延迟的 tmp overlay 在关闭后约 21.8s 捕获堆栈，显示 `first.NeedsConsent()` → `confirmHostTasks` → `askConfirmBlocking` 正等待用户回答；不能把待确认视为清理死锁，也未自动同意确认或用 runner 清理代替自然退出。此项证明分场景行为，未证明所有任务状态均能无确认退出。

## 未完成的整体验收

本轮没有执行全应用每个控件 × 空/载/错/成功的穷举，也没有生成当前本体与最终 MyGo 全部页面的同尺 baseline/actual/diff、阈值 0 像素检查和几何 ≤1 物理像素证据。手机网络、多平台硬件、真实通知中心 UI、长时间 GPU 性能，以及上表明确保留的插件与坐标行为边界尚不能报通过。

这些限制是尚需完成或验证的具体项目；实现回归通过不等同于本体所有功能和像素已完全一致。本轮也没有代替用户或审查人签署整体通过。

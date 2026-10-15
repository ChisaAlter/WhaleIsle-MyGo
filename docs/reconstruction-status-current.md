# 重构当前状态

更新：2026-10-15。依据模块交付记录及本次原版/原生代码契约对照；不把未运行验收误写成唯一剩余工作。

## 整体判断

原生桌面的主要功能链已大幅接通，包括聊天、文件、浏览器、Project/Team、插件、远程与桌宠；已有可交叉编译的 Windows 程序和离线便携候选。当前仍处于功能补齐阶段。下列真实功能缺口未完成，代码级和云端审查也未结束；不得进入整体验收或宣称全面重构完成。Windows 候选和历史构建证据仅记录既有成果，不是要求用户审批测试的开发前置条件。

## 已接通的核心功能域

- **聊天与会话**：会话独立草稿、附件/剪贴板图片、编辑/分支、无目录会话、队列移除/插话及完整文本编辑、交互批准、工具结果图片、行内文件打开；反馈采用真实版本条件和明确提交。麦克风开始/停止/取消接入授权转写草稿，不自动发送。
- **工作区与导航**：共享主侧栏、恢复面板与草稿、文件多标签/冲突保护、Git 暂存和提交入口、工作区独立终端、右栏浏览器及全页浏览器；本地 PDF 使用单文件隔离预览。终端原生 DLL 已有固定来源离线打包链。
- **Project / Team**：原版桌面桥与目录授权、项目创建/改名/归档/恢复、Lead 导航、成员与权威任务状态、停止/解除暂停、工作过程与历史报告、要求卡/交付物、笔记和偏好。重试仅准备 Lead 草稿，不伪造重试接口或自动运行任务。
- **设置、市场与插件**：模型/识图、MCP、provider、指令/技能组等编辑链，市场安装/更新/历史及待重启生效；真实插件插槽消费与会话级 HTTP/WS 授权网关，关闭或 Host 变化撤销旧连接。
- **用量与远程**：真实用量汇总、UTC 日期范围、模型/会话/工作目录排行、价格及导出；远程机器、目录/镜像、按会话绑定的文件读写和冲突保护。手机原版 runtime 桥、瞬态配对链接与二维码、构建/暂存链已实现。
- **生命周期与组件**：导入、退出恢复、隐藏到托盘；Host 任务保护覆盖重启/停止/退出和更新安装器交接。鲸桥首次安装、就绪核验、安全更新/回滚与维护保护已接入。
- **桌宠**：独立窗口、任务返回/专属聊天、可选 THA4 渲染、文件喂食、真实本地结算用量粮与成长、摸头和临时睡眠/唤醒。成长是本地状态，不是账户余额或充值。
- **Visual replies**：授权历史源码读取/另存、受严格门禁的静态 PNG 预览与隔离截图管线；显式启用前合成探针验收，Host 每个生成检查点查询当前进程隔离状态，重启不能沿用旧成功状态。

## 2026-10-15 已补齐的代码模块

下列模块已分别提交并推送至开发分支 dot/mygo-reconstruction-20261014，经过一次必要整合代码核对。云端文件系统空间耗尽使 shell 无法启动，本轮 Go 编译与仓库测试尚未执行；未开展验收，不把“已提交”写成“已验证可用”。

1. **导入后重装插件** — [997bbd3](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/997bbd3205934cbb02158c9e76f5a70fcb7f4f54)：来源规格白名单、选择、冲突复检、逐项结果与日志、取消传播、明确构建批准；修正 GitHub 仓库名不同于 package.name 的覆盖重装识别。
2. **手机配对设备管理** — [83ddc6c](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/83ddc6cbaa5b7f8708dfacb62a32eec8a4749ab5)：真实设备 ID、重命名、确认解绑、配对 offer 更新，子进程与请求身份校验。保持原版设备身份和旧 offer TTL，不直接修改内部凭据文件。单次内存 JS fixture 9 项通过，不代表真实设备验证。
3. **浏览器元素选取与截图** — [d6eabbd](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/d6eabbd53b1d0ab6f292694d9b88557ed749ff7e)：实际工具栏到原生选取/可见区域截图闭环；选取追加发起会话草稿、不发送，截图本地保存；导航、会话和 Host 变化隔离迟到结果。
4. **首次安装与指定版本选择** — [84663eb](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/84663eb9a1717c78d3827dbb9d2a9212004cd1bb)：MyGo 兼容 catalog、版本选择、哈希/大小校验、下载进度/取消/重试及既有任务保护安装交接；修正 NSIS runtime 目录与 bundled Node 发现。仓库尚无实际兼容发布资产，不能声称现在即可下载安装。

## 2026-10-15 后续工作流修复

- **助手真实路由** — [4f7fbe0](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/4f7fbe0b62abf42c030a26adcd34815628b03791)：Bot/房间选择经真实 bot/open 解析专属会话；修复仅加前缀却发往当前会话的错误，describe 传递 botId；仅 prompt accepted 后清草稿。
- **命令执行与历史** — [73b0f03](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/73b0f039db8321a20deba4aadccef8eb721cc497)：持久化配对 run/done 卡片和错误/输出，异步目录及变更刷新、空附件数组、多行命令/技能分流、会话/Host 隔离。手动压缩专用卡与通用命令卡的融合尚未补齐。
- **插件配置与启停** — [f812aa4](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/f812aa4851e14433f886dfbf5dcadb79871466e9)：原版 plugins 配置 owner 路由；启停遵循 Host 库存、只读限制及 ChangeResult，不提前伪改本地状态或错误降级 RPC。
- **定时任务编辑恢复** — [d1257b7](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/d1257b710fa050310923da1d6e08d681edf7a800)：冲突刷新 rebase 保留用户编辑，原会话有效性检查后跳转，历史请求/连接身份隔离。

- **设置安全保存** — [4fceecc](https://github.com/ChisaAlter/WhaleIsle-MyGo/commit/4fceecc7b08eb2b07aa1b4c785ad3ee5acf913cc)：仅提交已修改字段，显式 unset/reset，细粒度 JSON 路径操作及隐藏密钥路径保护；不重建继承值/被脱敏值，保留指令空白，完善 provider/agent 配置导航。

以上仍只有必要代码检查及部分内存 JS fixture，Go 编译/测试未运行。正在补 Files/Git/Terminal 和 Team 实际工作流缺口；不把有限范围检查称作完整全项目审查。

## 尚未关闭的实现与集成问题

- **动态 HTML 文档执行**：当前只有静态、禁 JavaScript 的安全预览。现有 WebView2 隔离不能证明覆盖 WebRTC/ICE/STUN/TURN 的全传输禁网，因此保持脚本禁用；需要可验证的隔离实现，不以猜测 flags 或资源拦截替代。
- **兼容运行包供应**：安装流程已接线，但实际可验证 MyGo runtime 发布资产尚不存在。Windows 候选仍未验证，不可伪造发布清单或误装 Electron 发行包。
- **本轮累计编译/云端检查**：存储故障尚未恢复；本轮新增模块源码仅在开发分支，本地候选及 v26 备份仍是旧 05668bef 源码，不能用旧二进制代表新功能。
- 全项目功能完成度仍须按实际覆盖范围确认；不把本轮四项修复等同全量功能对齐。

已排除的假缺口：原版同样禁止编辑带附件的队列项；独立 Electron launcher 的 peer 握手不能直接视为原生同进程架构缺失；Project 重试以准备 Lead 草稿承接真实契约，不虚构前端重试 RPC。

推进顺序：功能补齐 → 必要代码级检查及云端审查 → 再安排验收。避免反复全量测试与视觉细节审查挤占功能实现。

## 已有验证证据

- 各模块采用作者定向 mock/合成 fixture；整合阶段运行累计 Windows amd64、CGO_ENABLED=0 编译。编译通过不等于 Windows GUI 或原生 API 已运行。
- 固定官方 Host 的 Linux x64 完整构建、带 readiness guard 的产物清单和暂存、原版 Project overlay 已成功。
- **一次隔离真实 Linux Host 烟测成功**：新临时 HOME/数据目录、loopback 约束下 ready，认证根页 HTTP 200，原版 Project list HTTP 200 且 available=true；未创建项目或会话、未调用模型，正常退出。HTML fresh-process status 保持 false。
- 手机 runtime 的 Linux 依赖/native/JS 构建和暂存成功；Windows 官方固定资产已按哈希/ABI核对并装配候选。没有把 Linux 构建清单改写成 Windows 验收结果。

## 尚未完成的验收与明确边界

1. **Windows 便携候选仍为 prepared，runtime_verified=false。** 已有 EXE、原生资源和依赖图；必须在真实 Windows 恢复内部链接，并通过显式 native-load/原有验证门禁。该步骤尚未执行；即使通过，也不等于 Host 启动、UI 或安装升级验收。
2. Windows 的窗口/焦点/DPI、终端 DLL/PTY、PDF、WebView 隔离、麦克风权限与驱动、THA4/GPU 动画和性能均未实机验收。HTML 探针不满足隔离条件时应保持不可用，不能因代码编译而宣称生成已可用。
3. 真实账户登录、模型/语音 provider、SSH/远程文件、插件服务、Project/Team 工作执行、鲸桥实际运行切换、用户文件/Git 和安装器升级等尚无端到端验收；已有 mock 不替代这些结果。
4. 手机真实配对、relay/局域网互通仍待验收；桌面 Git 隧道已接原版闭包与显式临时授权门禁，仅 mock/编译验证，真实 Git/手机链仍未运行。二维码或桥接实现不代表手机远程整体验收完成。
5. 本地 HTML 文档 Browser 预览已补独立快照、授权工作区内 CSS/图片/字体有限内联与静态禁JS opaque sandbox，尚未实机验收；动态脚本能力的具体缺口见上节；mini 形态和原生布局差异需独立比较，不以同名 Electron API 缺失判定功能未实现。原生 Project 过程预览是只读快照；部分详情采用原生展开面板，不是逐像素复制原版浮层。
6. 尚无完整全量回归或全项目功能逐项验收结论，也没有发布签名/安装包正式发布结论。不提供未经证实的完成百分比。

更细的实际功能和限制见 [模块交付记录](module-delivery-20261014.md)；Windows 候选与显式验收步骤见 [候选激活说明](windows-candidate-activation-20261014.md)。

# 2026-10-14 模块整合交付

## 已整合功能
- Files / Git：多文件标签和未保存草稿；关闭时明确保存或丢弃；外部文件变更阻止覆盖；逐文件暂存/撤销暂存；复核已暂存内容再提交。
- 聊天 / 附件：会话独立草稿、稳定输入状态、附件上传和发送流程。
- 设置 / 计划：设置草稿保持、会话模型设置隔离、计划编辑面板和历史查看。
- 退出恢复：正常关闭前保存草稿、文件标签和面板状态；损坏快照与多实例写入冲突不静默覆盖；恢复不执行终端命令。
- 市场：目录及详情、安装进度/等待/取消、构建批准状态和过期响应保护。
- 市场收藏 / 安装历史：持久化收藏、过滤和撤销；记录安装最终结果，保留失败状态。
- 浏览器 / 终端：独立浏览器标签与页面历史，HTTP(S) 页面禁用应用 Go 绑定；工作区独立终端会话，隐藏不终止，应用退出关闭全部会话。


- IM 连接：真实机器人状态、重连、工作区及 preset 选择/清除、忙碌互斥和断开世代保护。
- 聊天回合分支：按 turn/end 的精确 atSeq 分支，重复请求禁用、失败反馈、分页与会话切换隔离。
- Agent 预设：管理预设并持久化后续新任务默认配置；没有虚构团队 CRUD 接口。
- 市场更新：检查并比较稳定版本，目录改变后失效旧结果，更新沿用安装批准和兼容豁免回执。
- 插件面板：重开恢复路径/查询/片段并适配新 Host origin，真正刷新、外部目标保持和导航错误反馈。
- 侧栏：补齐归档会话显示、过滤与删除状态处理。
- 远程机器：列表、新增/编辑/移除、显式选择/清除、连接/重连、非法端口及过期结果保护。
- 远程目录：使用真实 /ls、/mirror 浏览目录；创建镜像与采用工作区是两个明确操作。
- 用量价格：读取和编辑原 billing 配置，峰/谷三项与 flat 单价、添加/删除/保存/放弃、失败保留草稿；未知价格不猜测，全部历史费用估算明确不随日期选区变化。
- 用量日期范围：UTC 起止输入、日选择/扩展、7/14/30 预设，选区摘要/日图/模型排行与 JSON/CSV 一致；清除恢复全部历史，缺少日期契约的会话排行在选区中隐藏。
- 用量排行分页：会话通过 sessions.more 按 Token 分页，模型按完整 byModel 本地分批；失败可重试、重复点击禁用、刷新和选区变化使旧页失效。
- 用量日志修复：仅失败会话候选，逐会话明确确认后调用真实修复接口；展示备份及修复结果、失败保留统计并提示写入结果不确定，不自动重试。
- MCP 设置：选择编辑/保存/取消、删除目标确认、只读保护，并发及失败时保留草稿；未连接真实账号或运行 MCP 服务。
- Provider 档案：列出档案并两阶段确认删除，仅允许可写配置中的目标；关联凭证清理限于该页管理且可写的条目，失败反馈不伪报成功。
- 工具批准/交互输入：按会话过滤请求，采用真实单次批准状态，ACK 后移除卡片、失败保留答复重试；支持异步取消、custom/detail 与超时 claim，并移除卡片中无契约的 always 授权。
- 数据导入：预览与确认、逐项进度、取消及部分结果保留、重叠根拒绝；启动/重启共用忙碌准入，设置/MCP 按所选节及 MCP id 合并，默认跳过冲突，覆盖也保留未选节；畸形目标拒写，失败重命名保全旧字节。
- 桌宠当前会话状态：展示 Running/Idle/Unknown/Offline，点击回对应会话；隐藏主窗口仍由现有刷新驱动，不推断任务完成或全工作区状态。
- 桌宠核心：默认关闭的离线独立透明置顶小窗、已有 Codex v1/v2 皮肤与 fallback、窗口位置恢复、独立配置及窄拖窗权限；不修改 Electron pet/live2dPet，THA4 由后续模块提供可选渲染链，仍需真实设备验收。
- 技能组：原生分组管理、会话目录降级，只更新可写技能并保留用户调用权限；失败回滚，清空组显式写入空列表。
- 自定义指令：真实 general 优先并保留 schema，专用多行入口和全局作用域提示；按 UTF-16 计数限制1500，超限禁止保存且不发 RPC，保持显式保存和失败留稿。
- 页面快捷键：6个现有命令入口复用原动作；派发时排除编辑器/IME、自定义输入与终端、弹窗和弹层，并抑制重复键，不注册系统或网页全局快捷键。
- 消息编辑：仅空闲会话最新回合起始消息可编辑，按会话保存/恢复编辑草稿与附件；支持取消及 IME 安全 Escape，斜杠文本不作为命令裁决，替换旧回合投影。
- 工具结果图片：root/ptc settled 内容有序图库、数量折叠与光箱；严格元数据校验、坏块回退文本，附件请求去重/超时/失败重试，opaque ID 缓存名哈希化。
- 侧栏折叠：头部按钮/Ctrl+B 切换56 DIP收起轨，恢复先前拖动宽度；恢复快照向后兼容，折叠时不保留拖动目标，快捷键保留输入所有权。
- 账户登录入口：显式点击发起、重复响应防护、浏览器打开失败保留链接，并提供重试/取消阶段反馈；只用 mock 验证，未实际登录或扩展账户权限。
- 手机网页入口：移除把 daemon 配对端口当 SPA 的错误回退；只允许校验后的 HTTP(S) 入口，无有效 URL 时隐藏复制/打开。后续模块接入 offer v2 与二维码内存桥，仍不代表手机远程完成。
- 手机 runtime 桥：接入原版 DshdRemote 的受控 Node 适配、状态刷新与 offer v2 临时链接；禁用/重启/退出停止进程，外部打开不写入内嵌浏览器历史。Windows完整原版 built 资源仍待验收，Git 隧道由后续模块接通，未验证真实手机连接，不代表完整可运行。
- 无目录会话：等待 Host baseline 后使用 scratch 专用创建/安全复用；Hero 与侧栏分节入口，草稿搬移不抢新导航，发送/文件根采用选中会话实际 cwd，不接管插件所有会话。
- 运行中任务保护：close-to-tray 先隐藏且不误停任务；重启默认取消确认、防重复和导入忙碌检查，确认通过后才重启。
- Host 任务保护：原生退出/重启接入 inspect、acquire/drain、世代与租约复核、renew/release；未知覆盖和活动任务默认取消确认，错误阻止操作，保留导入忙碌和隐藏到托盘路径。仅虚构 token/mock 验证，更新安装交接由后续模块覆盖；未覆盖其他安装/reload 或 peer 协调。
- 启动器独立 Stop：复用 Host 保护协调器、默认取消风险提示和并发准入；确认后停止手机桥及 Host，桌面窗口保持打开，失败保留进程与窗口。
- 会话费用：消费 Host billedUsage 列表/快照/实时投影，按 provider/model 合并并递归计入 subagent、排除 fork；复用全局价格，输入区展示真实计费 Token、估算费用和峰谷明细/倒计时。缺失计数或价格明确提示，未加载数据不编造；原生详情为有界展开面板，不是 Electron 锚定浮层。
- 浏览器链接入口：聊天、输入草稿及工具结果共享 HTTP(S) 提取/校验和显式打开动作，支持 Markdown/裸链接、去重且拒绝嵌入凭据；工具长结果在截断前提取，点击走已有浏览器面板，渲染不自动导航或提交草稿。未改账户授权/手机配对链接。
- 识图模型配置：模型页消费真实 session/modelCatalog，按 image 能力筛选并保留目录外当前值；保存全局 llm-vision-fallback provider/model，关闭显式写空字符串，失败保留值与未提交表单草稿。未改 Host 识图流水线或凭据，未真实模型调用。
- 会话队列动作：移除/插话按真实 action.kind 和 accepted 回执更新，插话同步移除旧排队项，follow 世代拒绝旧响应；修正编辑协议，编辑入口由后续完整文本草稿模块接管。
- 目录选择：保留已等价的本地系统目录对话框；修复 SSH 内嵌卷列表哨兵不能被选为工作区或用于新建目录，采用路径入口增加一致防护。未替换或重造目录浏览器。
- 手机资源构建脚本：原版源码准备、纯 Node ABI 构建/暂存及 SHA256 清单门禁接入打包；只通过惰性文件 staging fixture，未执行 npm、PowerShell、原生资源构建或 daemon。当前原版快照缺 silero_vad.onnx，后续显式校验输入可补齐该源码资产；脚本不代表可运行资源已交付。
- 队列完整编辑：读取真实 inbox/next-turn 内容并使用独立完整文本草稿，显式保存/取消、失败留稿，不借发送框；原 itemId 更新保留 Host 身份，拒绝含附件项，token/世代/seq 防止晚到重开和旧项复活。
- 手机模型资源输入：显式本地 Silero 文件按固定原版 commit/blob、size/SHA256 校验，只复制进新构建工作树并记录来源；缺参/损坏拒绝，不自动下载、不提交二进制、不改参考源。已验证 source-only prepare，未安装依赖或完成原生构建，独立模型授权仍未核实。
- 交付文件点击：以实际会话 cwd 解析相对路径，校验既有本地文件及 realpath 边界后复用 Files 标签/未保存草稿；无法本地处理保持原路径交 Host，不再双开。拒绝 URL 伪路径；原版 browser-doc mini 未覆盖，正文词表由后续模块实现。
- 手机配对二维码：复用原 server qrcode 在内存生成 PNG 并显示，offer revision 防异步旧图回填，断连/过期/停止清理图像；无外部二维码服务、不保存敏感配对图。仅 Node fixture 通过及累计编译，真实 runtime/配对未运行。
- 更新安装器交接：全量/增量包先完成校验且不启动，明确安装并退出后保存快照，再持 Host 保护锁启动已复核 SHA/size 的安装器；启动失败保留窗口/Host，成功才退出。仅 mock，未真实下载、执行安装或停止进程。
- 正文文件提及：同轮显式交付及已成功根级写文件工具建立词表，行内代码精确路径/唯一 basename 原位点击复用交付文件打开；未知或 PTC 归属不猜测，不新增 browser-doc mini。
- 鲸桥首次安装：真实原版 release/manifest 契约和 SHA256/size 校验，后台下载走所选线路，独占暂存后原子发布安装记录；已有安装/状态/版本记录一律拒绝覆盖，成功保持 stopped，启动另行显式操作。仅本地 HTTP mock，未真实下载安装；更新/回滚由后续安全切换模块补齐；运行就绪由后续模块补齐，DSH 同步由官方组件负责。
- 鲸桥启动就绪：后台显式启动，30 秒 deadline 和进程早退通道，状态 ID/PID/版本与认证 loopback API 双核验；失败不伪报成功，只清理本次启动句柄，不动已有实例。仅 mock，无真实进程/组件账号连接。
- 鲸桥发布源对齐：按已核实新版官方独立仓读取 latest manifest，再固定 manifest.version 对应发布 EXE；CNB 保留列表并补 vendor Accept。坏版本/哈希拒绝，不从旧共享仓猜资源；仅 mock，未实际下载。
- THA4 桌宠：默认关闭的可选神经渲染，同源固定资源白名单，WebNN→WebGPU→WASM 降级、task.json 更新不重载模型、关闭释放；修复 MyGo 自定义 scheme localhost 路由，固定资源独立暂存/NSIS清单。模型静态形状与 mock/资源 staging 已核验，资源二进制不入 git，未实测 Windows 动画或 GPU 性能。
- 最新 Host 可选组合：默认保留旧定制，显式选择固定官方 commit 的独立构建目录；平台/架构、产物 SHA 和真实桌面桥能力门禁，缺产物或桥接能力保持关闭。当前仅源码装配/配置 fixture，Linux x64 官方 native-system→Host→Client→Web 完整构建 exit0，固定 commit 的12关键产物清单校验通过；未启动 Host，不能作为 Windows 运行时。
- HTML 预览桥边界：独立有界请求解析和 HTML/尺寸/主题校验，当前合法请求明确返回 503 unavailable，能力为 false；尚未具备严格隔离截图实现，不宣称截图或 HTML 生成可用。
- Project 桌面桥：固定官方原样 helper，经现有 install-control 认证和受限固定操作分发；只接受桌面已选/注册根，路径授权及原版 overlay 成功后挂载。启动依赖匹配新 Host 产物和真实 Node/overlay Ready，匹配最终构建清单未就绪时保持未启用；未运行 Git/PR/真实 Host。
- 已发布 visual reply 阅读：识别真实原生/PTC 发布引用，匹配新 Host 能力后调用授权 readVisualReply，完整只读源码与明确另存；会话/世代/token 隔离晚到结果，25MiB 解码上限和身份/字节数校验，旧 Host 失败不回退普通附件或网页。未执行 HTML、未开启生成/截图，匹配 Host 最终构建清单未就绪时仍受门禁。
- HTML 隔离平台基础：Windows 可选独立临时 WebView2 环境/UDF 校验及退出清理，拒绝资源、导航、弹窗、下载和权限，禁用宿主桥；普通窗口默认不变。仅编译基础，尚无安全 shell/capture 生命周期或实机隔离验收，预览能力仍 false。
- 原生 Project 入口：通过真实 /dsh-project pluginrpc 列表/创建/改名/详情→Lead 会话，创建仅当前已授权目录；忙碌/Host代际/关闭/切会话隔离，不自动发送 prompt。后端或桌面桥不可用时明确不可用，仍需匹配当前平台 Host 构建及桥接能力，Linux 构建成功不等同 Windows 运行验收。
- HTML 隔离渲染管线：opaque sandbox/CSP shell、独立 Windows 窗口、内存 PNG、取消/20 秒超时/4000 高度边界；核心仅 mock 验证，实际设备验收探针尚未接入，Available 仍 false，不开放真实 capture。
- Project 目录选择：真实系统目录对话框只采用用户选中路径，校验并经现有 EnsureWorkspace 登记后才更新配置/会话，原桥再次核验根；取消/代际变化不登记不回退，拒绝 .git、并发与登记失败。仅临时目录/mock，未触真实工作区。
- HTML 运行时验收 API：显式 Probe 仅运行固定合成页，成功验证独立环境、隔离边界和真实内存 PNG 后才置 Available；失败/重试关闭能力，暴露未隔离 WebRTC 则拒绝。此 Linux 环境未执行 Windows 探针，设置入口待后续接线，不能以编译通过宣称可截图。
- Project Team 面板：读取真实详情与 Lead agentTeam 投影，成员状态/权威任务板/工作分页，支持真实 stop/resume 并明确解除暂停不自动运行；idle 不等于任务完成，缺投影明确不可用。目录选择空配置仅用默认路径作初始位置，不预授权。仅 mock，未运行团队任务。
- Visual replies 显式启用：现有设置 enabled 写 true 前先核验当前本机 Host/产物，再执行固定合成 Probe，能力成立才提交设置；失败显示具体原因且不发启用 RPC，关闭不受阻，默认仍 off。真实 Windows 探针未运行，不能宣称隔离验收通过。
- 最新 Host 暂存验证：真实 Linux 产物经过临时 staging 与 manifest 校验，原版 Project overlay 生成成功，7 个 peer 均解析到 staged Harness；未启动 Host，Linux 产物不能充当 Windows 部署。
- HTML 隔离状态查询：现有 token 认证下只读 GET /desktop/html-preview/status 返回当前进程 available/reason，no-store、无 Probe 副作用，其他方法拒绝；Host 全 checkpoint 通过后续保护补丁查询当前隔离状态。
- Team 工作详情/过程：真实 workstreams/get 保持原 childId，session/follow 首快照后立即取消并按 throughSeq 分页，continuable 只读查看；失败重试仅准备原 Lead 草稿，已有草稿/附件/发送/编辑时不覆盖，不假造 retry API 或自动运行模型。
- 隔离截图 PNG 流：仅隔离 CapturePage 按 IStream 逻辑长度读取并限制 64MiB，不将分配容量尾字节作为图片传回；普通截图路径不变。未运行真实 WebView 截图。
- Host 跨重启 HTML 门禁：仅新构建副本在原 requireEnabled checkpoint 查询认证 readiness，保留用户持久 enabled；manifest 记录 guard 补丁/hash，stage/launch 拒绝未加保护的 current runtime，legacy 无 visual 包不受影响。带 guard 的 Linux 增量官方构建 exit0，manifest 与暂存校验通过；旧纯官方产物因缺补丁记录被拒绝，不再作为 HTML 生成就绪依据。
- Host 显式启动路径：configured DshBin 指向 Harness CLI 时也校验真实根的 readiness guard，避免配置优先级绕过 vendored runtime 门禁；不修改无 visual 包的 legacy 启动。
- 历史 visual reply 安全图片预览：先走原授权源码读取，再经显式 Probe/RenderSaved 门禁，成功仅显示内存 PNG 与裁剪说明；失败保留源码/另存并显示原因。关闭/重读/切会话或连接变化取消渲染并丢弃迟到结果；只 mock 和编译，未实际 Windows HTML 渲染。
- 插件插槽消费：应用自有 Cordis overlay 按真实 registrant/owner 打开 main/settings/sidebar/remote 视图，嵌套设置与卸载同步；临时 HTTP/WS 网关仅转发当前精确 loopback Host，认证留在 Go，关闭/Host变化撤销连接，插件页面无宿主绑定。仅 mock/编译，未启动真实 Host/WebView2。
- Project 用户要求卡：真实 requests/list/get 三态分页、目标/版本/交付证据和剩余问题；完成态显示已交付并区别 Team 底层任务。归档前重读并遵循 Host completed/current 原子校验，支持恢复；变更成功但刷新失败明确提示且不自动重放。仅 mock，未修改真实 Project。
- Project 交付物操作：原 workstreamId/delegationRef/path 经 Host 授权打开并显示 canonicalPath/modifiedSinceReport，缺引用拒绝；文件复用现有脏草稿安全打开，目录/diff走真实接口，diff 明确当前工作区而非历史回滚。仅 mock，未操作真实交付文件。
- 市场待重启生效：保留 restart-required 状态并提供显式受保护重启，取消/失败不清除，只有新 Host generation Ready 后才完成并刷新插件；仍停在插件面板才恢复逻辑页与新认证，离开不抢回。复用原热重载/owner清理，不自动中断或重复安装。
- Project 生命周期：整个项目真实 archive/restore，与要求卡归档分开；先读详情，活动或未知状态需明确确认停止并归档，成功核验生命周期，恢复不自动启动旧任务。无 delete 假接口，只 mock 未操作真实项目。
- 鲸桥安全更新/回滚：先验证清单与完整字节再受保护切换，原 Host 不重启时成功释放任务锁；认证 maintenance 拒绝活动 DSH，Windows 固定句柄核验确切进程路径，失败就绪恢复旧版本，退出不确定拒绝重复启动。保留旧目录/data与停止态，Stop复用保护；仅mock/编译，未执行真实组件/Windows进程操作。
- 手机 runtime 实际构建/暂存：隔离 Linux x64 原版依赖安装、native rebuild、server/CLI 与 browser bundle 成功，SQLite 内存/node-pty加载及 4.4GiB 暂存清单校验通过；提交仅跨平台脚本、固定来源缺件与许可证，保留真实依赖链接。不是 Windows runtime，也未启动 daemon/Host/配对/模型；大体积产物不入源码交付。
- 原生终端资源：固定官方 MyGo/Ghostty DLL 离线 SHA 校验、许可证与来源 sidecar，打包/NSIS必需暂存，运行前校验且禁止首次打开隐式下载；缺资源可补齐后重试，不改现有 PTY 会话链。真实 DLL 字节及离线 staging 已验证，未执行 Windows DLL/终端命令，二进制不入源码。
- 远程会话文件：真实 sessionBound/status、ls及按sessionId读写，mtime冲突保留草稿且禁止重放，二进制/截断内容禁写；显式下载两阶段确认镜像同名覆盖，不造同步RPC/进度。每session+origin独立草稿，切换/断连取消和世代隔离；仅mock，未连接真实SSH/Host或操作文件。
- 本地 PDF 预览：Files/交付卡/Office结果通过单文件有界内存快照和短期精确URL能力进入独立只读PDF页，保留脏草稿；仅该PDF GET/HEAD允许，禁脚本/宿主绑定/外网/下载，关闭/会话失效撤销。加载失败可明确系统打开，合成PDF fixture通过；未实测Windows PDF渲染或真实文档。
- Project 笔记/偏好：真实 store/read/write 原生编辑，notes 只改 userText、generatedText只读；显式保存、失败留稿、按Project/path隔离草稿，归档项目禁写。仅mock，未修改真实项目资料。
- Project 历史报告：真实 store/list 与 store/results 分页，按精确 delegationRef 读取详情并保留 request/version 元数据；交付物复用授权解析及脏草稿安全打开，原工作复用 Team/detail。仅mock，无历史回滚或真实文件操作。
- 桌宠专属聊天：单独 Pet chat 动作调用真实 assistant/ensure 后选中返回的持久会话，与当前任务返回分开；不发prompt/自动启插件，缺插件可见失败，重复/关闭/Host或导航变化隔离迟到响应。只mock/编译，未实测Windows交互或模型调用。
- 用量工作目录排行：真实 projects.more 的 tokens/cost 排序分页，按工作目录聚合而非 Project ID，保留无目录桶；失败留页可重试，排序/Host/刷新/价格变化隔离旧结果。仅mock，未连接真实用量服务。
- 顶层共享导航：聊天/市场/计划/助手复用完整侧栏，可直接互切、选会话回聊天，保留草稿/面板与尺寸状态；设置、PDF/WebView等继续原隔离边界。仅fixture与编译，未实测Windows交互。
- 右栏浏览器停靠：右栏 Browser 使用受限viewport停靠，不再切走整页聊天；切tab仅隐藏保留页面/历史与聊天/文件/终端状态，原全页入口继续可用。仅布局fixture与编译，未实测Windows嵌入视图。
- 桌宠文件喂食：原生选文件明确触发原版 fileEat care，持久化 live2dPet.stats 饱食+2/心情+1并保留未知字段/成长设置；不读取或保存文件内容/路径/名称，不伪造每日额度。仅mock/编译，无真实用户文件或模型执行。
- 桌宠 Token 粮成长：仅显式扫描本应用 sessions 的真实结算用量，沿用原 SHA256 高水位/首次基线，新增量才记本地粮，恢复/重扫不重复奖励；明确消费粮、单级上限及 feedToken care 同次原子保存，保留未知配置。不是账户余额/充值，无模型或网络调用；只合成fixture与编译，未读取真实用户日志。
- 桌宠摸头：显式 pat 复用原版care，心情+10/亲密+4，同类十分钟内亲密+1；沿用原子stats保存，不消费粮/积分、不调用模型。仅合成fixture与编译。
- 隔离 Linux Host 烟测：一次新临时HOME/数据目录、最小环境和loopback网络约束启动成功，认证根页200、原版Project list 200且available=true；未创建项目/会话、未模型调用，HTML fresh status仍false，SIGTERM正常exit0。仅Linux基础启动/组合验证，不代表Windows原生UI/手机配对验收。
- 桌宠临时睡眠/唤醒：睡眠不落配置/不发奖励，显式叫醒原版心情−7且重复wake幂等，成功照料/接触唤醒不追加惩罚；关闭重建清醒，睡眠暂停既有sprite/THA4循环，醒来复用实例。仅合成fixture与编译，未运行模型。
- Windows离线runtime候选：已用官方固定Node/ABI137 SQLite及35个精确平台包装配Host/mobile依赖图，静态PE/内部链接/来源hash检查完成；298426文件、17738相对链接和431相对cmd包装。状态prepared/runtime_verified=false，保留Linux来源清单，不伪造Windows成功marker；尚需Windows链接恢复/native-load验收，不能称解压即用，巨型产物不入源码附件。
- Windows便携候选布局：复用144c8d3已编译EXE、终端sidecar/resources/9项宠物资源与runtime图，依赖链接改清单并提供普通权限junction/hardlink恢复；静态拒逃逸/异常覆盖，不提权或改策略。实际本地目录已装配，runtime_verified=false、无生产成功marker；未运行任何PE，需Windows验收，巨型目录不入源码交付。
- Windows候选显式验收入口：固定Win32/x64 Node/ABI/hash、干净临时home的受限native加载，再通过原Host/mobile verifier才写成功marker，失败撤回新记录；mobile目标包置于自身根内，不放宽链接边界。入口已装配但未执行Windows，runtime_verified仍false，native通过也不等于Host/UI验收。
- 聊天剪贴板图片：显式粘贴才读PNG，16MiB/24MP界限并复用既有附件发送；会话/Host/编辑身份变化丢弃迟到结果，仅清理本次新缓存。图片-only不擦除选中文字；仅合成fixture与编译，未读取真实剪贴板。
- 主聊天麦克风录音：显式开始/停止/取消，WinMM内存PCM/WAV有界采集，复用真实speech catalog/transcribe授权；会话/Host/面板/窗口变化取消，UTF16原光标写草稿、冲突保留手动插入。无自动安装/prepare/发送，仅合成fixture与编译，未访问真实麦克风或验收Windows驱动/权限。
- 消息反馈：修正真实positive/negative与null/opaque版本CAS，赞踩先编辑再显式提交，删除冲突保护/失败留稿；按Host业务ok判定，不再缺接口本地伪成功，不增加sessionFeedback外发。仅mock与编译，未发送真实反馈。
- 手机 Git 桥：原版 DshdRemote.git/dispatch/Git闭包原字节接入，严格操作allowlist与私有stdio逐次核对桌面session/canonical cwd；默认关闭、显式临时授权，切会话/工作区/关闭面板或服务撤权并丢弃过期结果。只mock，无真实Git/隧道/手机配对验收，原源码许可证/哈希保留。
- 本地 HTML 文档：交付卡/Files 的 htm/html/xhtml 保存快照进入独立文档预览，opaque sandbox仅静态CSS/data图片，原生禁JS且CSP script-src none，禁外部资源、bindings/Host凭据/权限；保留源码草稿和系统fallback。仅合成fixture/编译，未执行真实HTML或验收Windows隔离。
- 原生用量：真实 Token 汇总/覆盖度、7/14/30 天 UTC 图、模型及会话排行、会话跳转、原生保存 CSV/完整 JSON；失败刷新标明旧数据。

- HTML 静态资源图：仅授权 cwd 内声明的 CSS/图片/字体有限内联，canonical/.git/symlink 边界；4MiB 单资源、16MiB 总量、64 文件、512 引用、CSS 深度4，跳过原因可见。仍原生/CSP/sandbox 三层禁JS；作者单合成 fixture 通过，累计 Windows 编译通过，未验真实渲染/字体。

## 实际验证范围
模块按顺序独立提交，共享 main/frame 接线由整合阶段处理；累计代码通过 Windows amd64、CGO_ENABLED=0 的 cmd/whaleisle 交叉编译。作者提供的定向 fixture/测试结果保留在各自交付中，整合未重复运行全量测试、race 或额外审查。

这是交叉编译验证，未运行 Windows 桌面 UI、未连接真实 Host、未在用户真实工作区执行文件或 Git 操作。运行时 UI/端到端体验仍需在用户授权环境验收。浏览器隔离是应用绑定权限边界，不是独立 cookie/存储配置。恢复覆盖正常退出/重启，不保证崩溃自动保存。

插件插槽消费与隔离已接代码，尚未经真实 WebView2 验收。本记录不代表全项目功能对齐完成。

## 构建
使用 Go 模块依赖，Windows amd64、CGO_ENABLED=0，运行 go build -o <源码目录外的路径>/whaleisle.exe ./cmd/whaleisle。go.mod 将 MyGo 指向仓库内 third_party/mygo；保留上游许可证和实现所需补丁，因此不依赖交付机器的绝对路径。

## 2026-10-15 云端 API 增量（尚未编译）

以下四项逐模块提交到开发分支。云端执行器因空间耗尽在启动 shell 前失败，因此本节仅完成必要代码接口/生命周期核对，未运行 Go 编译、仓库测试、GUI 或端到端验收。前文“累计 Windows 交叉编译通过”截至 05668bef，不覆盖本节提交；本地候选 EXE 与 Library v26 也仍对应 05668bef。

- 导入插件重装（997bbd3205934cbb02158c9e76f5a70fcb7f4f54）：按原 GitHub/npm 版本规格重新安装所选支持插件，保留已装冲突项，明确覆盖才重装；具体构建脚本需确认，取消/失败恢复当前安装项并保留已完成项目和日志。与市场互斥，使用确认目标目录和原包名校验，避免 GitHub 仓库名不同或 pnpm 链接导致误判。内置、本地及不支持来源仍明确跳过。本节新增 fixture 已编写，未执行。
- 手机设备管理（83ddc6cbaa5b7f8708dfacb62a32eec8a4749ab5）：原版运行时真实设备 ID 的重命名、确认解绑、临时配对令牌更新；解绑先停止连接再修改原注册表并恢复服务，其他设备身份保留。child/requestID 隔离晚到结果，超时不报成功，Git 恢复失败保持关闭。作者纯 JS 内存 fixture 9 项通过；Go/仓库 Node fixture 和真实配对均未执行。
- 浏览器元素选择与截图（d6eabbd53b1d0ab6f292694d9b88557ed749ff7e）：显式选择元素后仅向发起会话草稿追加有界页面注释，不发送；截图只保存当前可见区域 PNG 并显示位置。会话/Host/导航/关闭取消过期结果，按实际绘制保持右栏浏览器活动；PDF、静态 HTML 和认证插件页不开放该动作。截图有内存/像素边界，Windows读取流逻辑长度。未编译、未执行测试或真实网页操作。
- 原生发布目录与版本选择（84663eb9a1717c78d3827dbb9d2a9212004cd1bb）：固定所选 MyGo 发布条目，校验平台、架构、NSIS格式、大小、SHA及附带Host来源/门禁；支持有界下载进度和取消，下载后仍需显式“安装并退出”，复用任务保护、会话保存和末次哈希校验。缺运行时时只接受含匹配运行时的包，不将 Electron 或 prepared 候选当作有效发布。当前没有真实 MyGo 发布资产，目录和安装链已实现，实际安装仍受该外部缺件阻塞；未下载/执行安装器。

动态 HTML 的脚本交互仍是明确实现缺口：当前 WebView2 资源拦截/CSP不能证明 WebRTC 传输全面隔离，其他窗口的 Probe 结果也不能授权新文档上下文，故本地文档保持三层禁 JS 的静态资源快照。此项不能写成已完成而仅待验收。带附件队列编辑经原版契约确认同样禁止，不列为功能缺口。

代码实现、代码级检查和运行验证分别记录；上述进展不构成功能全面对齐或验收完成结论。后续继续实现真实剩余缺口，不以 Windows 本地验收作为继续开发前置。

## 2026-10-15 助手、命令、插件与计划契约增量（尚未编译）

本批沿用上述 API 开发方式。仅一轮必要代码接口/生命周期核对，不代表 Go 编译、Windows运行或全面验收通过；候选 EXE/Library v26 未含本批源码。

- 助手目标路由（4f7fbe0b62abf42c030a26adcd34815628b03791）：Dshbot 明确选择 Bot/群聊，使用真实 describe(botId) 与 bot/open 获取、核对 canonical session，再把原文本发送至该会话，移除向当前聊天加前缀的替代路由。bot/open 本身不执行版本CAS，客户端核验返回条目及会话身份；发送检查 accepted 回执后才清除匹配草稿，失败或后续编辑保留。列表不再仅显示前几项，冷会话通过原 open 恢复。新增回归源码，未执行测试或真实 Bot 调用。
- 聊天命令生命周期（73b0f039db8321a20deba4aadccef8eb721cc497）：command/run、command/done按真实commandId合并显示运行/完成/失败及输出，保留原journal和分页游标；命令目录异步读取、按session/client/版本隔离，并响应commands/change与preset变化，提供加载/失败重试。commands/execute无附件时传空数组，遵循命令输入/附件契约并核验结果。作者有界源码/JSON检查通过，Go测试未运行；sourceEventSeq已保留，手动压缩等专用命令卡片融合仍未实现。
- 插件设置视图与生命周期（f812aa4851e14433f886dfbf5dcadb79871466e9）：插件自定义配置入口沿原main/settings子slot和owner导航，不绕过其Host表单状态；设置页启停改用真实Host库存、只读限制和ChangeResult，取消先改本地配置及失败后猜另一接口的行为。操作复用世代/busy/重启待生效链，inspect结果同样隔离旧Host。作者JS mock一轮通过；Go编译/测试及真实插件UI未运行。
- 计划编辑与导航（d1257b710fa050310923da1d6e08d681edf7a800）：计划CAS冲突/任务结束后刷新原记录，草稿按字段保留本地改动并采用最新基线，结束任务只读；只允许从当前live会话清单导航，归档或未知会话不被隐式恢复。历史分页绑定任务、连接和请求版本，旧请求不能覆盖新展开项。作者有界源码契约检查通过，Go编译/测试与真实计划操作未运行。

本批补的是代码契约中的实际遗漏。Settings 的隐藏secret子树保存保护尚在修正，动态HTML隔离仍未开放；其余未覆盖功能域继续有界对照与实现，不以本批提交数量表示整体完成度。


### 2026-10-15 Settings：修改字段与敏感值保存边界

- 提交：4fceecc7b08eb2b07aa1b4c785ad3ee5acf913cc。
- 仅提交 dirty fields，显式 unset/reset，保留继承值和指令空白；嵌套 JSON 采用最小路径更新。
- 普通 JSON set/unset 与已配置 secret 路径祖先/后代重叠时拒绝，避免数组替换或对象类型变化丢失隐藏密钥；非敏感叶编辑仍可用。整体替换不从脱敏 effective 值重建秘密。
- 增加 schema-backed provider/agent/search/log/subagent/shell 导航，保留现有 MCP/技能实现。
- 一次必要代码整合检查完成；云端存储故障，Go 编译、测试未运行，无实际账户/密钥修改和验收。

## 2026-10-15 Project、文件/Git/终端、用量与隔离核心增量（尚未编译）

以下已逐模块发布到开发分支，均完成一次必要源码接口/生命周期检查。云端空间耗尽仍阻塞 shell，未执行 Go 编译、仓库测试、Windows GUI 或真实用户服务。现有候选 EXE 与 Library v26 仍截至 05668bef，不包含这些 API 提交。

- Team 分页与状态（50820d106980d5913f6db8ddfa84cb92627f7c05）：上一页、刷新保留 currentCursor、失效锚点回首页、排队等待原因及 Host worker roster 补全。
- 文件动作（19ead52f4fb3e67b16a5c1df3da89446f11092fe）：激活/手动重读干净标签；外部更改或读取失败保留脏草稿与原保存基线；分别复制相对/绝对路径，Host 与原生打开/定位互斥。
- Git 分支（f444f69370d15fe266426bac5ebddba50e205696）：列表/筛选/刷新、正常切换及创建分支、远端 tracking；保留 dirty-tree/worktree 保护，pull 使用 ff-only，不强制覆盖。
- 动态 HTML 隔离核心（a7713a95856a6e8faac22f5d4add5fa7b12790d7）：新增默认关闭的独立 helper；fresh AppContainer 零 capabilities、原子 Job 绑定、实际 helper/browser/进程树 token 核验后才传快照，取消回收，失败关闭。BrowserProcessID slot 37 已由固定 Microsoft 官方 vtable 源码核实。此提交没有产品入口/打包接线；同 SID/零 capabilities 的实际 WebView2 兼容性未验证，不能据此宣称动态预览已可用。此前静态禁 JS 边界继续有效。
- 终端会话组（834a6ef2379183ed061e43dfa57dc3f64ea9736a）：真实原生 PTY 独立组、每组最多四个分栏、当前终端关闭/重启；隐藏保留进程。移除不可见的 Host PTY 探测创建/写入链。
- Project 恢复草稿（2361f5ad957a04839a547022e913aabf2ea0cf0e）：刷新原 Lead、workstream、要求版本和报告/终态身份，仅向空的原 Lead 草稿准备 summary/handover 重试文字；不新造 retry RPC、不发送 prompt。导航、Host、原始身份及正在编辑/发送保护拒绝迟到覆盖。
- 手动压缩卡（c74b597f2891bd7e57f5bcd2c1212d4c7d02668e）：补齐前节所述专用卡片缺口，按明确 command/compaction 标识融合原 replace checkpoint 与对应摘要，展示历史条目/token数量、失败或未加载摘要；保留原 journal。
- SSH/远程配置（6843b08feebf5e67b0397369f137554ed934ceda）：SSH config 导入、草稿/已保存配置测试、认证/跳板/主机密钥控件、明确忘记密钥、转发生命周期和审计尾部。单跳 ProxyJump 解析用户/端口/IPv6，拒绝不支持多跳；同跳板留空密码沿 Host 保留原 secret。未执行 SSH、转发或密钥操作。
- Git 初始化/发布/提交并推送（63bce2d2ebe5522bcc509c3d7d7ae96dea5abbaf）：显式初始化；已有 URL 或 gh 创建仓库，默认私有、不覆盖 origin；确认目标与可见性后才发布。推送复核分支/远端/ref/URL，提示默认分支；提交仅包含已核对暂存内容，推送失败保留本地提交，避免重试时重复提交。未执行真实 Git 或创建远端。
- 用量目录与排序（0cba439485431eafc8c6df5034704c25e21cf30c）：真实 billing.models 服务商/模型选择，不虚构价格；sessions.more 的 tokens/cost 排序分页，使用已保存价格与完整分时桶估算；主/子会话、最近时间、all-time provider 和覆盖状态展示。缺价/缺数据明确未计价，估算不等于实际账单。
- 终端最大化/选文（bd06cf2803d309a9228f35ba6a70b537d48551bc）：最大化保留原布局、还原与拖拽退出；读取真实选区，复制或添加 fenced 片段到捕获的当前会话草稿，保留已有文本/附件；发送、IME 和会话切换保护，无自动发送。

本节记录代码能力与未验证边界，不以提交数代替完成比例。动态产品入口仍另行整合，原生 runtime 发布目录仍无实际 MyGo 安装资产；其余真实功能缺口继续逐域闭合。


### 2026-10-15 显式隔离预览、Remote 目录及 Files 完整入口

- 3d9962d379305e367affd26fcb1a684524347eed：静态默认不变，点击后冻结授权图片/CSS/JS依赖快照，启动受限helper；tab/context撤销关闭整个Job。静态fallback与错误原因保留；标准/离线打包携带受控helper，不提升全局readiness。
- 89870e000f491a49c0aa7aba5617157ab3ac013a：Remote会话目录创建、改名、明确确认永久删除、路径复制；确认Key含RemoteIdentity，不将旧主机确认复用于新绑定。
- e121693fd170237675a2168ed988f71308f9ba15：Remote picker Home、路径跳转、新建目录后选择，保留Windows卷层CanChoose门禁。
- e3a671b84a404e29b57f6539ddd57939f75238e3：Files Markdown默认Source/Rendered、可选源码和语法颜色、完整选中行加入草稿、跳行、有限图片/PNG型ICO；复杂格式兼容仍在收口。

### 2026-10-15 01:17 UTC 累计编译（不是验收）

清理经逐字节核对无独有内容的失败副本以及官方可重建Go缓存，恢复约1.3GiB空间。177+7路径加Files最终8项按远端Git SHA同步，Go格式化后Windows amd64 main/helper累计编译一次通过，无需逻辑修补。未运行测试套件、真实服务、GUI或用户文件操作。

- main 39,107,584 bytes，SHA256 afb20152d715cf2cdcd30a33be8d1cb95957f5eb7f83bfe1305dc3fa2d7a1f82。
- helper 13,556,224 bytes，SHA256 ce820c52ded9415e9fd5ce347064dd71a1720199996c74a966ffd524d291d4bf。
- 证据只覆盖本次代码快照的编译，不代表Windows运行兼容、ARM64、功能全量验收或实际发布资产。现有候选与Library v26尚未据此更新。

### 2026-10-15 01:33 UTC 桌宠动作、图片兼容与导入事务

- abb09b3855b7e8c1e43f328853d22aa789a5cd93：桌宠Hide写保存配置成功后关闭，直接进入已有Pet设置；保留其它配置和会话。
- 63d44bfeb81a287ad73b45392ecf365b9d284ebf：显式隔离GIF/ICO/AVIF/SVG兼容快照，授权单文件、格式/大小/像素/帧边界、固定data image与静态禁脚本；无法解码或范围外格式明确提示，不运行外部解码器。
- fc42f579128ae9a3a7c225816eaf06938c0690dd：明确点击且逐次确认后才向所选真实视觉route发送一张桌宠所在显示器缩略图；Host/会话/窗口/目录错误保护，取消与冷却，无自动截图或模型调用。开发期间未实际截图/调用模型。
- 导入a61f07a82a05caf9c39031141902eb53bb0577f7：受锁控制的私有暂存、树指纹、可恢复目录替换；启动前/导入前恢复，未决或原Electron遗留事务阻启动并保留副本。保留完成项；进程中断恢复不等同断电耐久或外部写入协调。

每模块一次Windows amd64 main增量编译通过；helper代码未改变，不重复编译。随后本项目cmd/internal的Windows目标go vet一次通过；本轮9个改动JS/CJS/MJS仅node --check全部通过。没有运行测试套件、GUI、页面、真实Host/账户/用户文件/截图/模型。最新main39,391,232bytes，SHA256 235384c4fc3d9f40698de26e16149fda977076afbee15a2193b6417464dee99a。

仍未完成：Pet更多设置的实际行为及紧凑聊天卡；实际WebView2进程树能否满足严格AppContainer门禁尚未知，失败保持静态，不因编译通过宣称动态功能已验收。兼容runtime发布资产仍缺；旧candidate/Library v26尚待同步新源码和已有编译产物。这些属于独立的实现/交付/运行边界，不提供未经证实的完成百分比。

## 2026-10-15 02:04 UTC 桌宠设置与紧凑聊天收口

- 显示设置 A：749f0125a975d677991ed5fe3edf7a2d3fe54d28。大小、角色透明度、锁定拖动/Shift拖动、点击音和聊天开关持久化；仅设置变化不重建渲染器或重置睡眠。
- 本地行为 B：2bd20d715c157c741a1a304632c02e071a7a32b9。四性格本地话语池、自言自语与活跃度、真实原生漫游、手动拖动锚点持久化和显示器变化夹限；睡眠/忙碌/互动暂停漫游，省电降低动画频率，无后台模型请求。
- 性格同步 C：6f8ab5ac52049b646211d2de20a6ec184514ff87。每次成功显式设置保存经原 settings/update 同步；重启仅重放明确已配置的合法性格，未设默认不覆盖 Host。连接与请求身份隔离、原版有限拒绝重试和传输重试。
- 紧凑聊天 D：55674debdf7441f11ea89b1a10073f23dad65b70。独立原生卡读取同一持久宠物会话历史/模型/思考强度，明确发送、取消本地等待、停止该会话当前轮及打开主会话；失败/取消保留草稿，成功仅清理匹配提交稿。关闭和 Host 变化取消本地请求。原服务不支持指定消息取消或原子 expected-session 变更，界面如实区分；没有虚构独立上下文或重试发送。

四模块均完成必要 Windows amd64 增量编译；C 的启动恢复补件在初版编译后到达，最终合成同一提交再完成必要编译。最终限定 go vet 只覆盖 cmd/whaleisle、internal/config、internal/desktop/pet、internal/ui/settings，通过；本轮两个 JS 的 node --check 通过。未运行测试套件、真实桌宠/窗口/声音/截图/模型/Host 或像素级 UI 验收。

最终主 EXE 39,640,064 bytes，SHA256 e05798992ded882cc6e5db7883b93c31e0d28fa62d0a7ad90928049d10c21a66；helper沿用未改动源码的13,556,224 bytes构建，SHA256 ce820c52ded9415e9fd5ce347064dd71a1720199996c74a966ffd524d291d4bf。已最小替换便携候选中的两个程序并同步哈希说明，未执行链接恢复、激活或runtime加载，runtime_verified=false。Library v26仍为历史快照，后续源码备份以最终提交为准。

以上更新覆盖前文按时间记录的“Pet设置/紧凑卡未实现”和“candidate旧版本”历史状态；隔离预览的实际 WebView2 进程树兼容性、可发布 runtime 资产及真实用户服务运行边界仍未关闭。

### 2026-10-15 02:07 UTC 有界人工代码审查收口

针对本轮Pet新增配置、草稿、会话和Host生命周期完成一次人工静态审查，发现并修复一项P2：成功保存禁用聊天后，取消并失效仍等待的主会话ensure请求，完成回调再校验当前设置和窗口；重新启用不能复活旧响应。审查者仅复核该小差异后确认闭合，未发现其他可由本轮最终代码证明的阻断或高价值中等问题。修复后必要Windows主程序增量编译通过。该结论限定本轮代码，未运行测试套件、GUI、Host、截图或模型，不等同全产品验收。

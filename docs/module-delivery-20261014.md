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
- 手机 runtime 桥：接入原版 DshdRemote 的受控 Node 适配、状态刷新与 offer v2 临时链接；禁用/重启/退出停止进程，外部打开不写入内嵌浏览器历史。仍缺完整原版 built 资源和 Git 隧道，未验证真实手机连接，不代表完整可运行。
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
- 原生用量：真实 Token 汇总/覆盖度、7/14/30 天 UTC 图、模型及会话排行、会话跳转、原生保存 CSV/完整 JSON；失败刷新标明旧数据。

## 实际验证范围
模块按顺序独立提交，共享 main/frame 接线由整合阶段处理；累计代码通过 Windows amd64、CGO_ENABLED=0 的 cmd/whaleisle 交叉编译。作者提供的定向 fixture/测试结果保留在各自交付中，整合未重复运行全量测试、race 或额外审查。

这是交叉编译验证，未运行 Windows 桌面 UI、未连接真实 Host、未在用户真实工作区执行文件或 Git 操作。运行时 UI/端到端体验仍需在用户授权环境验收。浏览器隔离是应用绑定权限边界，不是独立 cookie/存储配置。恢复覆盖正常退出/重启，不保证崩溃自动保存。

插件插槽消费与隔离已接代码，尚未经真实 WebView2 验收。本记录不代表全项目功能对齐完成。

## 构建
使用 Go 模块依赖，Windows amd64、CGO_ENABLED=0，运行 go build -o <源码目录外的路径>/whaleisle.exe ./cmd/whaleisle。go.mod 将 MyGo 指向仓库内 third_party/mygo；保留上游许可证和实现所需补丁，因此不依赖交付机器的绝对路径。

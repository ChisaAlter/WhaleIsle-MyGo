# 真实会话下 UI 卡顿诊断（2026-10-26）

现象：用真实账号/长会话数据启动后，一操作就无响应；Process.Responding=false，UI 进程持续占满一个核。

证据（`tmp/hang.txt`、`tmp/hang2.txt`）：连续 goroutine dump 中，UI 线程在 `main.mainView` 内反复经过 `session.(*Store).Snapshot()`、`turnUsageLocked()` 的 JSON 解码及 `trajectoryRequestsLocked()`。当时每帧重建完整会话视图，历史越长单帧越慢，输入无法及时处理。

最初两处降低开销：

1. `navigationFileDropModel` 不再每帧调用 `Snapshot()`，改用 `Store.SelectedSessionID()`（`internal/store/session/store.go`）。
2. 会话插件同步改为 250ms 节流，不可见时不读取 snapshot（`cmd/whaleisle/plugin_session_host.go`）。

后续已按 `viewRev` 缓存 `Snapshot()` 的投影，并让帧内复用一次克隆；修改后的单测验证了构建与命中次数（[snapshot_cache_test.go](../../internal/store/session/snapshot_cache_test.go)）。这消除了上述「每次调用都重新聚合」路径。当时尚无长会话原生测量；下文已补入合成长历史经生产 `mainView` 的原生测量、两次 profile 和后续修复。合成数据的结果仍不能代替原账号／原会话下的完整复验，也不据此认定全部卡顿已解决。

说明：验收用的 fresh profile 没有历史，所以之前的启动/退出冒烟无法暴露该问题。

## 2026-10-27：消除聊天投影的两处二次扫描

在虚拟列表创建前，`projectTranscript()` 原来为每轮再次扫描全部消息，检查最终回答所在步骤是否含工具，并收集折叠组成员；`assistantActionTargets()` 原来为每个最终回答重新扫描此前历史拼接同一来源的可见正文。这两处不受列表只绘制可见行的保护，轮数随历史长度增加时会产生近似平方增长。

当前改为每轮建立工具步骤索引并在一次遍历中收集有序成员；复制按钮正文按来源逐次写入 `strings.Builder`，在候选回答出现时保存其当时的正文前缀。原顺序、重复或空 ID、跨轮重复来源、用户插入、流式与不可复制块、结束状态及旧资料回退保持原语义。定向回归与 100 组随机输入的旧函数对照通过；聊天包完整测试通过。

同一进程交替执行原函数与新函数的合成基准，每组重复三次，下表取中位数，单位为毫秒/次：

| 消息行数 | 过程分组：原实现 | 过程分组：修改后 | 回答操作：原实现 | 回答操作：修改后 |
| --- | ---: | ---: | ---: | ---: |
| 500 | 0.961 | 0.568 | 0.284 | 0.096 |
| 2,000 | 11.994 | 2.004 | 4.554 | 0.284 |
| 8,000 | 167.439 | 8.448 | 76.908 | 1.102 |

8,000 行下，回答操作的每次分配从约 0.916 MB 增至 1.246 MB（2,031 次增至 6,060 次），以来源索引和正文前缀换取扫描减少；过程分组约 5.466 MB，基本未变。证据为 `tmp/composer-long-history-20261027/after-check-verified.log`，基准不访问真实资料、Host、原生窗口或模型。

独立的现有合成 `Snapshot` 基准在 8,000 行下约为：缓存读取 3.00 ms/4.84 MB，新帧读取 3.27 ms/4.84 MB，同帧复用 71 ns/零分配；数据变更后的重投影仍约 240.6 ms/66 MB。此轮没有修改 Store 的快照所有权或深复制合同。这些测量证明了局部算法开销减少，尚不证明真实长会话的帧延迟、输入响应或全部卡顿已经解决。

## 同轮复核：缓存必须核对来源内容

回归测试进一步复现了坐标缓存的旧缺陷：同一 ID、序号和字节数不代表原始事件相同。等长内容替换、工具调用更换及切换到使用相同序号型 ID 的另一会话时，旧缓存会返回上一条记录的轮次或步骤。

坐标缓存现在先按 ID、序号、长度定位候选，再与缓存自有的原始字节逐一比较；只缓存当前记录自身的坐标，缺失坐标时重新跟随当前工具调用。最多保留 16,384 条及 8 MiB 原始字节，超大单条不缓存且不清空正常条目。原实现下四类定向回归均失败，修改后与边界和原分组对照测试通过（`tmp/composer-coordinate-memo-20261027/old-red.log`、`verify.log`）。

精准比较有成本：普通小记录的热命中约 70–85 ns，64 KiB 记录约 1.70–1.92 µs，均零分配；随后不同时间负载下，8,000 行过程分组约 10.74–13.48 ms，不能把前一节 8.448 ms 当作加入正确性门禁后的固定结果。

相邻两个缓存也已用旧实现复现错误：同长度的文件交付路径替换后仍链接旧文件；原始正文不变而消息的 Streaming/Kind/Role/来源/轮次/触发状态改变时仍显示旧投影；工作流数量不变但名称/状态改变时也不刷新。现在文件提及持有自己的 raw 字符串及结束态，投影按全部值字段、原始/内容字节和独立时间戳副本比较，工作流按名称/状态、阶段、成员、事件原始字节及序号比较。复杂 metadata 没有已定义的独立身份时保守重建，不用指针或长度推断内容未变。

最终 `tmp/composer-source-memos-20261027/freeze-check.log` 中，旧函数对照、原地修改、关闭态、时间戳、全部工作流字段与安全重建回归通过；2,000 行三次热缓存基准为：纯文本 99–102 µs、工具 96–116 µs、带时间戳工具 113–122 µs、工作流 107–116 µs，均零分配。该阶段 Command/Retry、实时 preparing Tool、Web、compaction、submission、Files 和内存会话坐标等复杂字段仍会触发重建；这部分性能没有据上述数字判定解决。后文 task55 才补齐实际 normal-follow 输入中的 Files 身份，不能将此阶段的纯文本热命中数字当作当时已覆盖交付场景。

## 同屏多个最终回答：文件引用缓存反复失效

后续从实际绘制路径确认：[chat.go](../../internal/ui/chat/chat.go) 会为每个可见 Markdown 消息分别请求文件引用 resolver；Standard 模式仍显示不同轮的最终回答。同一个回答拆成多个共用 raw 的块会命中缓存，但同屏两个不同回答会交替替换唯一条目。此前的单个结束消息测试和投影基准没有覆盖这个路径。

同一份 2,000 行、500 轮、228,843 字节 raw 的合成历史，先填充缓存，再对最后 1/2/4/8 个不同回答重复轮遍。计数通过独立 overlay 获取，性能使用未加计数的生产函数，每组重复三次，单位为完整一轮：

| 不同回答数 | 缓存填充后每轮重建次数 | 耗时中位数 | 每轮分配 |
| ---: | ---: | ---: | ---: |
| 1 | 0 | 21.24 µs | 0 |
| 2 | 2 | 20.61 ms | 7.26 MB |
| 4 | 4 | 46.70 ms | 14.52 MB |
| 8 | 8 | 97.87 ms | 29.00 MB |

证据：[构建计数](../../tmp/composer-mention-thrash-20261027/counts.log)、[生产函数基准及源码未变校验](../../tmp/composer-mention-thrash-20261027/audit.log)。一次 miss 还复制整份 raw source，约 272,768 字节／2,002 次分配；八个回答一轮仅此复制约 2.18 MB／1.15 ms。主要成本仍包含重复 JSON 解码、排序和路径建表，不能把总耗时归因于字符串复制。

确认后已改为共用一份独立的精确 journal source，并保留最多 16 个按最近使用顺序淘汰的结束消息 resolver，包括无可链接文件时的 nil 结果。raw、Role 和 Streaming 共同识别每个结束消息；journal 等长替换或原地修改会使全部旧条目失效。独立 raw 字节总计限 8 MiB，超大 source／closing 直接计算且不驱逐有用缓存；此限制不包含路径 map、slice header 或调用者保留的函数。复制和 JSON 构建在锁外完成，代次校验拒绝旧视图在新 source 之后回写。

旧代码 overlay 下，2/4/8 个回答的 warm 轮遍均按预期失败；新代码的 16 个回答重复轮遍固定只构建首批 16 次，并保留同一 source。逐项测试覆盖第 17 个条目仅淘汰最久未用项、精确 8 MiB／超大旁路、nil 缓存、全部来源失效、慢旧构建不阻塞新 source 或污染回写，以及只读输入的并发命中。定向测试及完整聊天包通过，证据为 `tmp/composer-mention-lru-20261027/baseline-red.log`、`verify.log`。

同一 2,000 行 fixture 的修改后完整轮遍耗时（三次中位数）为：1/2/4/8/16 个回答约 19.08/38.13/78.61/153.92/311.27 µs，均零字节、零分配；比较 source 仍为线性开销。该测量没有运行 GUI，不代表实际帧率或输入延迟。

## 原生 mainView 测量与窄读取修复

独立临时 overlay 在真实 Windows 原生 UI 窗口中调用生产 `mainView`，通过 Store 的 `inspectorfixture` snapshot 入口加入 2,000 条合成消息。它不调用生产 `main()`，不创建 Host、client 或模型请求，窗口显示但不抢焦点，HTTP 尝试为零。每组先预热 20 次，再串行采集 100 次 `Window.Update` 到主线程回调及下一次 `mainView` 返回的时间；已有 `MYGO_FRAME_STATS=all` 另记录 CPU build/layout/paint/present 调用。present 不是 DWM 扫描输出，也不据此报告 FPS。

第一轮证据 `tmp/markdown-native-long-run-01/proof.json` 在可见 2/3/3 个不同最终回答下，主视图回调 p50 分别为 88.246/82.793/80.456 ms，p95 为 137.167/111.146/125.793 ms；Update 排队 p95 约 4.76–4.82 ms。目标 HWND 的 `WM_KEYDOWN` 被原生观察器接收，`WM_CHAR` 使草稿由空变成 Q，只作定向输入冒烟，观察到的 170 ms 不解释为真实键盘延迟。该轮与其他诊断/构建存在负载重叠，不是机器独占测量。首轮 fixture 把交付事件排在最终回答之后，生产代码正确将其过滤，因此它没有显示交付卡片；缺少 schedule store 也使四张路由图只能支持侧栏状态，不能用作任务页面验收。原始结果保留，下一轮修正 fixture 后另存。

实际调用复核随后发现四处不必要的历史复制：导航栏和窗口标题的拖放路径只读取 SelectedID；会话拖放再读 SelectedID 校验归属；归档确认 UI 只读取当前确认状态。它们现在分别使用已有 `SelectedSessionID()` 和新增 `ArchiveConfirmation()` 窄 getter。后者在锁内只复制当前确认框及嵌套 activity/items，保留调用者可修改副本和当前 token/Busy/Error 的语义。没有跨帧共享可修改底层数据，也未改变 `Snapshot`/`FrameSnapshot` 的所有权合同。

定向 session 回归和现有 frame 拖放/归档测试通过；新测试覆盖嵌套副本独立、旧 token 拒绝、最新状态和不构建完整投影。合成「两次 selection + 一次空归档」在 2,000 行下由约 2.84 ms／3.65 MB／17,781 分配变为 56.39 ns／零分配；8,000 行下由约 8.73 ms／14.52 MB／71,031 分配变为 65.13 ns／零分配。证据 `tmp/composer-narrow-reads-20261027/verify.log`。这只证明窄读取减少复制，不能把局部差额当作整帧收益；后续原生测量单独记录。

修正 fixture 后的第二轮使用已含窄读取修复的二进制，经 normal follow → Store Snapshot 的 2,000 条消息中有 400 条带非 nil `Files`、输入 `TurnDelivery` 为零；生产投影输出 400 张交付卡，视区为 3 张卡片、2 个不同最终回答。三个 100-sample 批次的 mainView p50 为 66.486/62.502/58.460 ms，p95 为 87.871/110.673/86.143 ms；保留最大 648.952 ms 离群值。FrameStats 的 layout/paint/present-call p95 分别约 1.6–2.1/1.0–1.1/2.2–2.5 ms，主要耗时仍是 CPU build。每帧全进程分配约 29.9 MB，87–94/100 帧观察到 GC，不能把这些分配都归属于 mainView。七张窗口内容截图及实际任务页面已核对。证据 `tmp/markdown-native-long-summary.md`、`tmp/markdown-native-long-run-02/`；首轮与该轮 fixture 和代码都不同，未作精确 A/B。

随后单独在同一 fixture 的 8-tail 场景采集 Go `runtime/pprof` CPU profile，包含 20 次预热和 100 次更新。14.21 s 时间窗得到全进程累计 17.58 s CPU 样本（含多 goroutine），不是前述 Node 的 I/O wall-sample。应用累计路径包括 `chat.View` 4.25 s、完整聊天投影 1.74 s、TurnRail 1.33 s、ImageRefs 1.15 s、交付投影 1.02 s、ScheduleCards 0.85 s、RecordedPerformance 0.77 s；Snapshot 克隆 0.58 s。函数包含关系重叠，不可相加。全进程 GC drain 累计 4.65 s；Windows runtime 采样中的 semasleep/preemption 也不能直接解读为产品锁阻塞。证据 `tmp/markdown-native-long-profile-summary.md`、`tmp/markdown-native-long-run-03/cpu.pprof` 及保存的 top/list 报告。该轮启用 profiler 后的耗时不作为前后速度对照，后续仅修复经源码合同确认的工作。

## 第二次源码修复与原生复测

共享图片引用入口现在对不含字面量 `[` 的输入直接返回 nil；支持的 Markdown 图片、引用式图片和 `[image]` 占位符都必须含 `[`，HTML `img` 原来不在该 API 收集范围内。完整旧 Goldmark 路径作独立 oracle，16 组代表输入和 64 组 Unicode 文本结果一致，定向 parser/session/TurnRail 测试通过。纯文本短例从约 64.7 µs／190 分配变为 15.65 ns／零分配，Unicode 例从 144.5 µs／253 分配变为 80.59 ns／零分配。证据 `tmp/images-fast-negative-20261027/`；不是整帧收益。

投影 memo 随后为 TurnDelivery 和 Files 定义了精确、独立的身份，miss 时连返回投影的嵌套内容也独立复制；warm 不分配。比较保留 nil/empty、顺序、全部 target 和字符串字段，其他九类复杂字段仍按原合同重建。最初仅覆盖 TurnDelivery 的 task52 不能解决原生数据，因为真实 normal follow → Snapshot 中有 400 个非 nil Files、输入 TurnDelivery 为零。task55 再补 Files 身份并实际通过该生产 parser/Store 入口验证：2,000 条 raw 消息、400 个 Files、零个阻止缓存字段，仍输出 400 张交付卡，连续投影命中同一 backing。合成真实形状 warm 基准 2k／400Files 约 18.50 ms／4.29 MB → 366 µs／零分配，8k／1600Files 85.18 ms／17.17 MB → 1.48 ms／零分配；相近时间批次不等于同进程严格 A/B。定向、normal-follow 与完整 chat 均通过。证据 `tmp/composer-delivery-projection-20261027/`、`tmp/composer-files-projection-20261027/`。

加入这些修复后的原生 run04 保留同一 2k／400 卡、20 次预热、100 次测量、可见 3 卡／2 个不同最终回答和草稿 Q 的强校验，HTTP 为零，未启用 profiler。实际 mainView p50／p95 **104.690／1008.137 ms**，最大 1412.604 ms，未观察到延迟改善；更新排队 p50／p95 6.036／30.139 ms，提交至回调结束 119.343／1020.088 ms。FrameStats 全进程每帧分配均值约 74,079 次／17,044 KB，较旧批次下降但不据此忽略延迟。测量机在随后 19:13 的独立读数 CPU 负载 91%，说明机器非独占；不能用该后置样本解释每个慢帧，也不能据此排除代码问题。原始离群值全部保留在 [run04 证据目录](../../tmp/markdown-native-long-run-04/)及[原始日志](../../tmp/markdown-native-long-run-04.log)。随后另采 CPU 和默认采样率 allocs profile 定位，再实施下文三项修复；run04 的慢尾观察没有被后续批次覆盖或删除。

独立 profile05 在本轮重任务暂停后完成，自有进程 PID23992 自然退出；其他用户进程和 OS 后台工作未控制。进程于 2026-10-27 19:29:34–19:29:41（Asia/Shanghai）运行，CPU profile 覆盖 20 次预热及 100 次 Update 观察，实际窗口 5.753s。前后 `runtime.MemStats` 的全进程分配增量为 2,117,279,896B／8,982,917 次，GC 61 次，累计暂停 92.546ms；默认 `MemProfileRate=524,288B`，未修改采样率或强制 GC。这些增量包括窗口内所有 goroutine 和额外触发的原生帧，不能直接除以 120 声称每次 mainView 的专属分配。主视图 p50／p95 为 39.9002／71.2173ms，带 profiler 且机器条件不同，只用于定位，不替换 run04。

缓存前后精确 key/source/workflow 均匹配，实际输入 `Files=400`／`TurnDelivery=0`／阻止缓存的复杂字段计数为零。仅凭探针相等不足以证明实际绘制使用了它，因此同时核对生产 [chat.View](../../internal/ui/chat/chat.go#L127) 的调用及 [projectChatMessages 行采样](../../tmp/markdown-native-long-run-05/list-chat-cache-cpu.txt)：30ms／0.44% 在真实热命中比较处，miss-builder 分支没有采样到成本。没有把该零采样解释为绝对零成本。

6.82s 全线程 CPU 样本中，`RecordedPerformance` 累计 850ms（12.46%）、`ScheduleCards` 820ms（12.02%），各有 790／700ms 在逐条 JSON 解码；`projectTranscript` 650ms（9.53%），`Store.Snapshot` 510ms（7.48%）。全进程 `gcDrain` 1.06s（15.54%），`mallocgc` 770ms（11.29%），累计路径有包含关系，不能相加，也不能概括为 60% GC。默认采样 allocs 自进程启动累计估算 2,123.44MB，`cloneMessages` 路径约 1,018.48MB、其中 `Store.Snapshot` 约 915.16MB；技能提示整段复制约 183.02MB，任务解析约 148.65MB。这是含启动的抽样累计估计，与上述 MemStats 窗口增量不同；不能直接作每帧除法、与无 profiler 批次相减，或将其归因于后来 task69 单项修复。详见 [profile05 报告](../../tmp/markdown-native-long-run-05-summary.md)、[CPU profile](../../tmp/markdown-native-long-run-05/cpu.pprof)、[allocs profile](../../tmp/markdown-native-long-run-05/allocs.pprof) 和保存的源码行报告。

## 三项剩余热点修复及一次无 profiler 原生复测

profile05 确认热点后完成了三项窄修复，分别验证语义后再合并测量：

1. **task68：UI 自有的精确衍生缓存。** 新增 [HistoryDerivedMemo](../../internal/store/session/history_derived.go)，每个 UI owner 保留一份有序来源，限制为 8,192 行／4 MiB（包含保守 metadata 估量）。按 history binding、session ID、nil/empty、逐行 Seq／Streaming、全部 Raw 字节及独立复制的时间戳值比较；等长或原地修改会失效，超限或 NaN 输入走原纯函数且不驱逐有用的小来源。原 `RecordedPerformance`／`ScheduleCards` 纯函数保持不变。性能统计按值返回，任务结果每次返回独立 map、slice、Raw，外部修改不能污染缓存。接线位于聊天 root 的任务衍生、[性能状态行](../../internal/ui/chat/performance_usage.go#L147)及[费用栏](../../internal/ui/sessioncost/view.go#L120)各自的 `ui.Local` owner；关闭费用统计时跳过该衍生读取，实时任务目录、`EnsureContext` 和收据状态仍按原路径更新。warm 命中仍须线性核对来源，不是 O(1)。
2. **task69：只复用同一次 render pass 的只读视图。** `frame.Model.FrameView` 接收生产 mainView 已建立的 `sessionFrameViews.view(store)`；[renderSnapshot](../../internal/ui/frame/frame_snapshot.go)先核对 [BelongsTo](../../internal/store/session/frame_snapshot_owner.go) 的 exact Store 归属，匹配才使用该 pass，否则回退普通 `Store.Snapshot()`。原 `FrameSnapshot.Snapshot()` 每次读取仍检查 `viewRev`，同帧中途写入也会刷新。导航、侧栏、工作区等同步绘制读取改走这个入口，事件回调和异步读取不保留该句柄；普通 Snapshot 的独立可修改副本、跨 pass 独立性和原 FrameSnapshot 实现没有放宽。nil Store 不注册无效 pass。生产 mainView 的两个 `FrameView` 传参均已进入实际测量二进制。
3. **task70：没有 context 时跳过技能历史复制。** [userMessageSkillNames](../../internal/ui/chat/user_references.go#L259)在原无效 target Seq 检查后，先不分配地扫描 `MessageKindContext`；没有 context 则必无技能名，直接返回 nil。有任意 context 时完整保留旧复制、稳定排序、Raw 解析与批次边界路径，不依据 Role、ID、Raw 拼写或历史已排序作假设。14 组旧实现 oracle 对照和 2,000 行／400 交付行无 context 场景通过，并断言零分配。函数微基准从 386,001ns／765,176B／1,600 次分配降至 1,560ns／零分配；这是局部函数结果，不是 GUI 收益。

task68 的独立 oracle／边界／UI 测试及三包完整整合通过：[共享验证日志](../../tmp/history-derived-memo-20261027/verify.log)中全 session 8.786s、chat 93.089s、sessioncost 2.059s，包含 task70 的冻结代码，未重复运行完整聊天包。task69 的 Store 合同 4.885s、frame 8.253s、cmd 5.097s 以及最终 nil guard 定向 3.526s 通过；覆盖归属、同帧版本变更和 UI 图像等价。task70 [定向日志](../../tmp/skill-fast-negative-20261027/verify-02.log)通过 3.754s，移除新增 guard 后其余生产源码与原件一致。[task69 冻结记录](../../tmp/composer-frame-snapshot-20261027/freeze.log)和 [task70 验收记录](../../tmp/skill-fast-negative-20261027/summary.md)保留具体文件与哈希。

### native06 的实际入口、输入和功能验证

三项修复全部完成后仅运行一次无 profiler 的 single8 批次。夹具仍通过 `nativeLongView → mainView(c)` 调用生产入口，没有自行构造 `frame.Model` 或替代 frame renderer；不调用生产 `main()`，所以不能作为真实 Host 启动或原账号会话验收。相对 run04，夹具源码只替换版本标签，1440×1440 请求窗口、20 次预热、100 次 Update 观察及全部强断言均未改变；软件 scene 捕获仍为 2138×1380。自有 PID34104 于 2026-10-27 19:56:30 启动，19:56:33–19:56:36 为测量标记区间，随后测试成功并正常退出，事后再次核对 PID 不存在。

输入经 normal follow parser → Store Snapshot，2,000 条 raw 消息中有 400 条非 nil `Files`、输入 `TurnDelivery` 为零；生产投影输出 2,000 行和 400 张交付卡。实际视区强验为 3 张卡片、两个不同最终回答 `FINAL-08-399`／`FINAL-08-400`，HistoryReady=true、HistoryLoading=false。自有 HWND 经 PID 校验后接收定向 OS 消息，原生观察器记录一个 key event，真实 composer draft 从空变为 `Q`；没有提交消息。52.0388ms 的草稿观察还包含排队更新和读回，不是实际键盘延迟。夹具自身 HostStarted=false、modelRequests=0、deniedHTTPAttempts=0、physicalInput=false；软件截图已核对三张卡片和两个最终回答，Q 的证明来自实际草稿读回而非截图。

### native06 的普通 wall-time 与全进程帧统计

| 100 次 Update 观察 | p50 ms | p95 ms | 最大 ms | 均值 ms |
| --- | ---: | ---: | ---: | ---: |
| 生产 mainView 回调 | 21.2720 | 30.2765 | 85.4366 | 22.8996 |
| 提交到主线程回调的排队 | 2.0170 | 3.8774 | 4.9409 | 1.9187 |
| 提交到下一次 mainView 返回 | 22.9103 | 32.0539 | 86.4460 | 24.8963 |

原始 p95 为 30.2765ms，不通过取整宣称低于 30ms，也不新增用户未指定的 30ms 验收门槛。相比 run02／run04，本批观察到的耗时更低，但代码、测量时负载和并发条件不同；它是三项修复共同生效后的实测，不能当作精确 A/B 或拆成各项独立收益。run04 的全部慢尾与失败观察仍保留。CPU build 和分配仍有可见成本，不能据此宣称原始卡顿完全解决。

FrameStats 共记录 127 帧，其中恰有 100 帧在本批测量标记内；所有测量帧均走 GPU renderer。每帧记录的是全进程分配，不是 mainView 专属分配，也不是此次三项修复中的单项分配。

| native06 标记内 100 个原生帧 | 均值 | p50 | p95 | 最大值 |
| --- | ---: | ---: | ---: | ---: |
| 全进程分配次数 | 58,823.78 | 58,351 | 62,552 | 62,921 |
| 全进程分配 KB | 12,163.707 | 11,860.0 | 13,023.9 | 13,121.2 |
| CPU total ms | 25.132 | 23.2 | 32.4 | 87.4 |
| CPU build ms | 23.021 | 21.3 | 31.2 | 85.4 |
| CPU layout ms | 0.893 | 1.0 | 1.5 | 1.7 |
| CPU paint ms | 0.230 | 0.0 | 1.0 | 1.5 |
| CPU present-call ms | 0.712 | 0.5 | 2.0 | 2.7 |

39／100 帧观察到 GC，每帧有一次新 text layout；慢帧没有过滤。作为描述性量级，run02 eight-tail 均值约 168,291 次／29,919KB，run04 约 74,079 次／17,045KB；profile05 的标记内 101 个原生帧均值约 74,142 次／17,056KB，与其 100 次 Update 回调观察不是同一个计数。native06 仍约 12MB／帧。以上是不同批次的全进程读数；native06 没有再采 CPU／allocs profile，因此没有修复后的 `cloneMessages`／`Store.Snapshot` 累计样本，不用 profile05 的约 915MB 与此数相减、相除或宣称 task69 单独节省了它。

### 真实并发阶段与结论边界

Lead 请求暂停本方重任务时，plugin-ui 的 mainrouter PID25328 已于 19:56:15.6992937 启动；其 HostNode28492 在 +1080.367ms spawn，+9207.271ms 就绪，约为 19:56:24.9066。runtime gate 为 cold=false，没有额外 runtime warmup。native06 在 19:56:33–36 采样时，该 Host 已就绪；重叠的主要工作是 peer 的 WebView2 采图和输入功能测试，直到其 Host 于 +26232.644ms、约 19:56:41.9319 停止。该时序已直接读取 [peer startup trace](../../tmp/plugin-inline-production-p2/main-eb550ef613294ee2a30aef99d45fce37/startup-trace.json)，采图／输入阶段由 peer 的执行记录说明。不能把重叠笼统称作采样期间 Host 冷启动，也不能声称本方任务独占或全机 CPU／GPU 独占；其他进程和 OS 后台工作未控制。这一并发事实随原始结果保留，没有补跑一批覆盖它，也未强停 peer 的进程。

`mainView` 回调与提交到其返回的时间不包含随后 layout／paint／present；FrameStats 的这些 phase 另列。present-call 是 CPU 端提交／阻塞调用耗时，不是 GPU 完成、DWM scanout、帧率或输入到显示延迟。`CapturePage` 是最后原生 scene 的软件光栅化，不是屏幕 framebuffer。功能断言通过与普通 wall-time 改善是本批实际结论；真实账号长会话、机器条件受控的速度对照及完整输入延迟仍不在此测量覆盖范围内。

构建前后 20 项源码／overlay 哈希一致，实际运行二进制 SHA256 为 `5A82A68A853C9B979E6DA3F6CA382933E1B525FD4133894203389A0608740566`，包含 task68 helper／三个 UI 接线、task69 main 两个 FrameView 传参／归属 helper／nil guard、task70 guard 及原 task51／55 修复。task69 另两处导航／工作区源码也与作者冻结前记录和构建后实际哈希一致；不据此声称整个工作区冻结。证据：[构建前后清单](../../tmp/markdown-native-long-build-06.log)、[额外 task69 与夹具一致性记录](../../tmp/markdown-native-long-extra-manifest-06.log)、[完整 run06 报告](../../tmp/markdown-native-long-run-06-summary.md)、[功能／回调 proof](../../tmp/markdown-native-long-run-06/proof.json)、[原始日志](../../tmp/markdown-native-long-run-06.log)、[单场景帧统计](../../tmp/markdown-native-long-run-06/framestats-single-case.json)、[scene 截图](../../tmp/markdown-native-long-run-06/history-8.png)及[进程退出记录](../../tmp/markdown-native-long-final-receipt-06.log)。本次只补文档，不新增测试、GUI、profile 或基准运行。
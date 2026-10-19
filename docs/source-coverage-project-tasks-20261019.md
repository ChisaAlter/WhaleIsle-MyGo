# Project / Tasks 源码覆盖记录（2026-10-19）

本记录对应总矩阵的 PROJ-01..05、TASK-01..04（简称 PROJ01..05、TASK01..04），只登记本代理实际阅读、对照和修改的范围。**一行包含多个子模块，不代表这一行的全部功能已经逐条审完，更不代表全部功能已正常运行。** 不继承总矩阵中的历史 Host PASS 为本轮源码审查结论。

原版只读基线为 `C:\AI\Deepseek-Harness-Desktop`，不是 `C:\AI\whale-code`。下文 `V/` 表示原版 `vendor/`，`C/` 表示原版 `vendor/deepseek-harness/packages/client/`；Go 路径以本仓库为根。函数行号是本次原版文件中的定位锚点，后续改版可能移动。最终产品源码阶段父线程标记为 `911ff74`；本记录准备时最近读取的仓库提交为 `0345aed`，期间其他代理仍在工作，不能据此宣称完整工作树已经审完。

覆盖等级：

- **behavior-compared**：实际阅读了原版和 Go 的对应实现，比较了表中明确限定的行为。
- **entrypoint-sampled**：阅读了部分入口、回调或状态归属，但没有贯通完整对应行为。
- **not-reviewed**：未阅读足以判断行为的实现；路径或文件名定位不算源码行为审查。

## 逐项记录

| ID | 原版具体入口 | Go 具体入口 | 等级及实际范围 | 未实现 / 未知边界 |
| --- | --- | --- | --- | --- |
| PROJ-01 | `V/dsh-project/client/client.js` 的 Project 行、菜单、目录及弹窗入口；本代理没有完整阅读这些入口的原版函数体 | `cmd/whaleisle/project_flow.go` 的 `projectModel`、`runProjectAction` 部分回调；`project_assets.go` 的 `runProjectAsset`、`projectAssetReference` 目录分支；`internal/projectflow/client.go` 的 `Open` | **entrypoint-sampled**。阅读 Go 的目录身份检查、Project 原 Lead 路由及部分公共绑定；没有完整双边审查行菜单/创建弹窗 | 行菜单、创建、目录选择、取消/失败/重命名和全部 modal 行为仍未知。不能把 TASK 卡片阅读范围或已有后端验收外推到本行 UI。 |
| PROJ-02 | `V/dsh-project/client/client.js:684` `ProjectTaskCard` 的各卡 `saved` 状态和读取效果；`:626` `ProjectTaskDelivery`；`:937` `ProjectTaskDetail`；`:1139` `retryHandover` 及 `controls` | `internal/ui/projects/{observer,task_cards,request_changes}.go`；`cmd/whaleisle/project_requests.go`、`project_request_cards.go`、`project_request_detail.go`、`project_assets.go`、`project_request_revert.go`、`project_recovery.go` | **behavior-compared，仅独立卡片行为**。比较展开 A/B、读取原卡、文件 immutable report 引用、非 Git 工作 diff、工作详情、交付恢复、撤销与协调者请求；修复最后读 B 导致 A 灰掉或错绑 B。队列菜单/请求读入口是 **entrypoint-sampled** | 没有逐条审完整个 Team、委派执行、成员 provisioning/调度、队列所有 mutation/并发、observer 持久化/键盘/全部布局。已对照子路径没有保留确认未实现项；其余未知不能写成“无差异”。 |
| PROJ-03 | `V/dsh-project/client/client.js:1328` `ProjectVisualResults` 的 `renderFactorySlot('visual-reply.page', {sessionId,file,title,height})`；任务 overview/results 使用点；`V/dsh-project/lib/domain.js` 的 page.file 对象/height schema | `internal/projectflow/deliveries.go` 的 `DeliveryPageReference`/`DeliveryPageFile`，`requests.go` 的 `RequestCard.Delivery.Pages`；`internal/ui/projects/{visual_results,deliveries,activity}.go`；`cmd/whaleisle/project_visual_{results,interactive}.go`；`internal/store/session/visual_reply_reader.go`；既有 `visual_reply_surface.go`、隔离 `previewprocess` 和 chat seat | **behavior-compared，仅 published page 与结果引用适配**。比较真实对象 DTO、session/attachment 归属、页面高度、每卡独立页面；复用已授权 HTML reader 和既有隔离 inline seat；按公开布局测 DIP 宽度，交互失败明确回退 PNG | notes/preferences/charter/materials/documents 的完整源码行为本代理 **not-reviewed**。历史 delivery 的完整列表/分页/报告流程只有结果页相关入口抽查。未运行真实隔离 helper、跨平台像素输入或原版像素比较；安全限制下的页面执行也不能等同完整浏览器能力。 |
| PROJ-04 | `V/dsh-project/client/client.js:937` 的每卡 work、versions、delivery 活动来源；`:626` 的 `showRequestDiff(card.id)` / `showDiff(project.id,workstreamId)`；`:1149` 的 `controls` | `cmd/whaleisle/project_activity.go`、`project_request_detail.go`、`project_request_revert.go`；`internal/projectflow/{activity,request_changes}.go`；`internal/ui/projects/{request_changes,request_diff_presentation}.go`；`internal/ui/diffsurface/recorded_patch_document.go` / `patch_document.go` | **behavior-compared，仅卡片活动/工作归属与改动入口**。活动 browser / 页面不再跟随 currentCard 或 B 的请求代次退休。Go recorded/unified diff 的顺序、原始 hunk、元数据 union 和截断规则实际读过，但原版 DiffFile 内部渲染没有完整双边阅读，因此该渲染子项只是 **entrypoint-sampled** | 全部 reports、历史 process、terminal/report 引用和各返回/分页路径未完整审查。不能将改动入口修复或纯内存 parser 测试准备写成这些子模块已全部覆盖。 |
| PROJ-05 | `V/dsh-project/client/client.js:626` `ProjectTaskDelivery` 的 revertible/lifecycle、冲突 Modal、`controls.revert(card)`、`controls.ask(...)`；`:1149` 控制器携带 requestId/requestRevision；原版其他生命周期入口未完整阅读 | `cmd/whaleisle/project_request_revert.go`、`project_request_cards.go`、`project_recovery.go`；`internal/projectflow/request_changes.go` 的 `RevertRequest` / `PrepareRevertRequest`；`internal/ui/projects/request_changes.go`；`project_requests.go` 中 archive/restore 入口 | **behavior-compared，仅 request revert、显示修订和冲突恢复**。缓存精确修订授权，A/B 冲突分离；Host CAS 保留显示修订；Ask 捕获原卡并沿原 Lead composer；modal 样式保持。archive/restore 菜单与 wrapper 仅 **entrypoint-sampled** | Project rename/archive/restore 全链、材料 dirty close、所有请求生命周期的失败和并发状态未在本代理完整双边审查。没有新发现可直接确认为未实现的其他生命周期功能。 |
| TASK-01 | `C/ui-jobs/src/client/JobListAction.tsx`（本轮仅用文件索引确认位置） | `internal/jobflow/{controller,model}.go`、`internal/store/session/jobs.go`、`internal/ui/frame/jobs.go`（索引定位） | **not-reviewed**。本代理未阅读双方 Jobs roster/output/cursor/mount 的完整实现 | 生命周期、输出跟随、窗口关闭/重连、任务 ownership 均为本代理源码未知。总矩阵的实际 rc.2 Host 记录不是本轮 source review。 |
| TASK-02 | 同一 `JobListAction.tsx` 的 cancel/kill/clear 行为（本轮未读函数体） | `internal/jobflow/controller.go`、`internal/store/session/jobs.go`、`internal/ui/frame/jobs.go`（索引定位） | **not-reviewed** | cancel/kill 确认、生产者结束、clear 保留 Host 记录、busy/error 和 unmount 收敛均未在本代理源码对照。不能继承他人运行证据声称已审。 |
| TASK-03 | `C/ui-schedule/src/client/{TaskManagerPage,TaskDetail,DeliveryHistory,ScheduleCatalogAction,ScheduleTaskTab}.tsx`（本轮仅确认文件名） | `internal/ui/schedule/{panel,presentation,delivery_history,open_task}.go`、`internal/store/schedule/{store,history,catalog_controls}.go`（索引定位） | **not-reviewed** | catalog/detail/history 的完整状态、失败和时区语义未阅读。历史未来任务 fixture 不代表源码或实际交付已审。 |
| TASK-04 | `C/ui-schedule/src/client/{ScheduleCreateCard,TaskDetail,TaskMenu}.tsx`、`task-timing.ts` / `task-cron.ts`（仅索引定位） | `internal/ui/schedule/{draft,task_cron,timing_picker}.go`、`internal/store/schedule/{edit,catalog_controls}.go`（索引定位） | **not-reviewed** | create/edit/run/cancel/delete、排程边界、权限和 model/delivery 全部仍为本代理源码未知；没有执行这些动作。 |

## 已关闭的具体源码差异

“已关闭”只指源码修改已提交，仍需统一回归和运行验收。

| 差异 | 提交 | 关闭范围 |
| --- | --- | --- |
| page.file 误当 string、缺 Height，真实对象页面可能使整个 delivery/activity 解码失败；已读卡结果缺可达页面预览 | `1b9ba61ba35b056a27c15a371f6accb5f9bd551f`，父线程在 `b48c10d` / `0bdb900` 接公共字段与生命周期 | 改对象 DTO、独立 session reader、结果页 native presentation；没有把主会话或直接 Host 文件访问当授权 |
| 固定 728px 及缺少原版页面交互 | `7c3ec88a3edb5bd59071ab3223943265884e003c` | 实际 slot DIP 宽度、旧 capture 取消、同 HTML 复用；existing isolated inline surface / clipped seat，失败有 PNG 静态标记；没有另造窗口入口 |
| 读 B 导致已展开 A 页面变 metadata 或重启 JS | `9b2ceb4b67d53383c899b188e77329465ff899ca`，公共 cache/读卡钩子由父线程 `0bdb900` 接入 | Host/controller 权威 RequestCard 缓存，冻结 page 身份，不以 UI copy 或全局 currentCard 授权；真实关闭/导航失效仍取消 |
| A 的 Undo 永久灰、冲突属于全局最后读取卡、Ask 可能作用 B | `38b7565b1fb56439c1d0e281995e4346cbec4f39` | 精确 cached revision、逐卡冲突、确认保留显示 revision、Host CAS 和原卡 Ask；保留原 Modal 样式 |
| 文件、非 Git diff、work details、handover、activity 继续误用 currentCard | `91097185adec26d0140537a14afb932b4a1457a8` | 全部既有 TaskCard 相关动作捕获 cardID/revision 并核验 cache；保留既有 RPC、原生文件 dirty guard 和 Lead composer；activity/page 不因另卡读取而退休 |

上述提交没有穷尽 Project 的各个行级模块。父线程实现但本代理未贯通阅读的差异，不能写进本代理行为审查成果。

## 测试和验收状态

`3b37750c2df86c3376b6a9514b039d4c2c031664` 准备了三个新测试文件并更新一个旧文件：13 条新增 test 函数、2 条原测试的授权语义更新。覆盖内存 controller ownership、精确引用、活动来源、Host/nav/cache 失效、显示 revision 的内存 RoundTripper RPC、DTO 对象及 80/2000 保真、headless UI 高度/宽度/逐卡按钮、unified diff 顺序及截断。它们不是实际 Host / helper / model E2E。

本代理只执行过 gofmt 和 diff 静态检查，没有运行或编译这批测试。父线程告知统一 run01 因编译阻塞未到达 cmd 测试；相关修复随后已有源码提交，待父线程后续统一执行。此记录不把“测试已准备”“源码已提交”改写成测试通过，也不声称所有 Project 或 TASK 模块正常可用。

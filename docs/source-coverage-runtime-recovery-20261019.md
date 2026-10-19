# Runtime / recovery 源码覆盖记录（2026-10-19）

本记录只覆盖 BOOT-02、BOOT-03、BOOT-06、SAVE-04 中下列选定行为，不代表启动、恢复、持久化模块全部穷尽。原版 `C:\AI\Deepseek-Harness-Desktop` 只读；没有读取用户 runtime 选择、config、session log、凭据或恢复文件。本轮未运行 Host、测试、构建或模型。历史 runtime / 两进程恢复 receipts 不继承为当前源码通过结论。

`behavior-compared` 只指明确列出的双端行为已读源码；`entrypoint-sampled` 指入口或部分状态路径抽样；`not-reviewed` 表示尚未审过对应契约。等级不是整个模块完成度，也不是实机验收。`D/` 为原版根，`V/` 为其 `vendor/deepseek-harness/`，`P/` 为 `V/packages/client/`；Go 路径相对本仓库。

| ID | 等级 | 实际选定范围 | 尚未覆盖 |
|---|---|---|---|
| BOOT-02 | behavior-compared | Host/Node/source 选择、匹配资源准入；生产依赖图仅入口抽样 | 当前用户真实运行版本；完整提取/升级/回滚；所有平台、备用启动及全部 native kits 闭包 |
| BOOT-03 | behavior-compared | 严格事件读者及启动前 required-event / provenance 准入 | 所有历史格式、损坏/截断/fork/repair、各 storage backend、实际旧/新读者执行 |
| BOOT-06 | behavior-compared | 文件缓冲、pane/selection/layout 的序列化、恢复与 checkpoint 时机 | 新源码的异常退出/实机回归；所有旧 snapshot 迁移；composer 已审部分不重复，question 由另一审计者负责 |
| SAVE-04 | entrypoint-sampled | 上述恢复；Files WordWrap、shortcut 事务与显式 reload；terminal drawer 展示/创建门控 | Browser storage controller、全部 preview viewer、终端 Host/PTY 完整生命周期、所有 shortcut import/advanced binding 与系统多窗口 |

## BOOT-02：实际资源选择

原版已读 `D/src/main/dsh.js` 的 `resolveNodeBin`、`resolveDshBin`、`sourceHarnessStatus` 和 source 启动分支；`D/scripts/production-runtime.js` 的 `productionClosure/selectHarnessRuntimeSources`；`runtime-instance-graph.js` 的 `assembleRuntimeInstances` 入口和物理实例身份。`harness-controller.js` 只抽样启动/损坏恢复片段，未通读所有恢复分支。

Go 已读 `internal/paths/{paths,host_runtime}.go`，`internal/harness/{launch,prepare,current_host,session_runtime}.go`，`scripts/session-runtime.mjs` 的 inspect/inventory/stage 前半，`cmd/whaleisle/portable_desktop.go` 私有资源准入及 main 启动参数接线。

原版 source 存在时优先 source；Go 明确配置 DshBin 优先，再 vendor、PATH，npx 需显式允许。Go bundled Node 优先 PATH，rc.2 profile 检查 Node 字节身份；Windows 不走仅 macOS 的 managed Host selection。此处记录实际源码策略差异，不推断用户选择了哪个 runtime，也不推断任何 fallback 已经执行。

没有调用 Resolve/stage 或读取已安装选择。source CLI 事件预检不能泛化为自定义可执行文件、PATH/npm fallback 的 reader 兼容证明。图包装只抽样关键入口，不据此声称所有依赖/native 资源完整。

## BOOT-03：严格兼容准入

原版已读 `V/packages/core/session/src/known-event-types.ts` 的事件目录，以及 `V/packages/session/session-persistence/src/storage-contract.ts` 的 stored-id、format-version、unknown-required/ignorable、retired request/header fallback 与 append contiguity 契约。

Go `internal/harness/session_runtime.go` 从构建 lib 的 KNOWN_SESSION_EVENT_TYPES 检查五种 desktop required events，并验证已支持 rc.2 provenance、关键 artifact 与匹配 Node hash。`scripts/session-runtime.mjs:inspectSessionRuntime` 通过实际 persistence 导出对 synthetic required/unknown-required 事件做准入，拒绝混合 package version。这里只读了脚本，没有执行 probe。

format v4 本身不建立兼容性。Go 不改写 session，也不把 unknown-required 标为 ignorable；有限五事件预检同样不是完整 schema 等价证明。实际读取仍依赖匹配 Host 的严格 persistence validator；本轮没有读取用户日志或重放会话。

## BOOT-06：文件与 pane 恢复

原版已读 `P/ui-files/src/client/desktop-file-state.ts`：稳定 resource address、per-edit dirty buffer 持久化、close/save 失败保留；`P/ui-sidebar-right/src/client/persistence.ts`：按 session 保存 surface、引用验证、恢复时无旧 undo；`P/ui-layout/src/client/{persist,stores}.ts`：surfaces/drawer 当前开关与 last-open geometry，closed=0。sidebar/rightbar 的其余状态主要为会话期状态，不能笼统称原版全布局都跨启动保存。

Go 已读 `internal/recovery/store.go` 的 bounded schema、无效/新版本原文件保留、跨进程 lock、exact-baseline CAS、原子替换；`cmd/whaleisle/{recovery,workspace_sessions}.go` 的 pending selection、workspace/session scope、文件/布局恢复；`internal/ui/files/tabs.go` Export/Restore；frame WorkspacePanes Export/Restore、captureContent/restoreContent。

确认原 Go 生产 saveResume 调用只有受保护退出/更新/卸载等关闭路径，Export 能力不等于编辑时持久化。`59c3582` 补 combined checkpoint：mainView 结束后由原生 UI 线程捕获独立 payload，500ms 节流/SHA 去重，600ms trailing debounce，单 writer 只写最新 pending revision。`6755c17` 加每批首 pending 至多 5s 的上界，连续编辑不无限延期；正在写入时新 pending 独立计时。只写恢复文件，不写工作区文件或历史，不执行恢复动作。

Combined writer 仅拥有 Workspace、SelectedSession、Layout、Drafts、Files；Update 保留当时 Store 的 QuestionDrafts，避免覆盖独立回答草稿 writer。失败保留 UI/最新 pending，停止无依据自动重试，以常驻 native error toast 提供显式 Retry。`02c6bd3` 失效失败捕获的 fingerprint，保证 A→capture error→相同 A Retry 仍能入队。

正常退出先在 mainState 锁外 Pause/barrier 等待旧 writer，再保持 question.Flush 在 mainState 锁外，最终通过 RunOnMain 捕获最新 UI，写 combined owned 字段；旧 revision 不能在最终保存后回写。`3a3064f` 补 OnHide/OnMinimize 的 UI 线程即时捕获：SDK Invalidate 最终依赖 Windows surface paint，隐藏窗口不保证继续 mainView，因此不能让最后一帧受节流的编辑只等待下次绘制；生命周期回调直接 stage 独立 payload，仍沿同一 debounce/max-wait/CAS writer。此处是源码契约，尚未运行新回归或进行异常退出实机验收。

## SAVE-04：展示状态、shortcut 与边界

已读 `internal/ui/files/preview_preferences.go` 的 WordWrap 随 Files record 保存；未重审每个 viewer。Browser 这里只读到 pane snapshot 接线，未重审其 storage/partition controller，不继承历史 browser receipt。

原版 `ui-layout` 的 `toggleTerminalDrawer/setTerminalDrawer` 同时保存当前 open/closed 与 lastDrawer；旧 Go 只存高度，启动固定隐藏。`59c3582` 增加 Layout.TerminalOpen，恢复时仅展示 drawer 几何；terminalResumePending 阻止 ensureTerminal 走 Sessions.Get/Open，沿现有 nil-panel Restart 入口由用户显式启动。未保存或重放旧命令。实际读到的 terminal `stores.ts` 仅 session/group/emptyState 入口，不据此宣称原版 Host/PTY 跨重启契约已完整比较。

快捷键实际已读原版 `P/shortcuts/src/persistence.ts` 的 `ShortcutPersistence.readCurrent()`、`edit()` 的 rereadBeforeWrite，以及 `read()` 的重新读取 raw、从 read diagnostic 回 ready。Go 已读 `internal/shortcutprefs/store.go` Get/Apply/readLocked、`internal/config/unified_shortcuts.go`、`cmd/whaleisle/settings_shortcuts.go` 与 native UI discard/reload。

已确认旧 Go ShortcutDocument 受旧 cfg baseline 阻塞：外部任意 config 变化后同 reader 持续 conflict，UI 仅 Discard/Reconcile 不能重新取 baseline/清 read diagnostic。已将精确来源交主线程，其独占 shortcut files 实施；`9cf5e01`/`5109f46` 是主线程提供的来源，本作者没有重新读完该补丁，不据此提升覆盖等级或声明通过。

## 当前结论

四项的选定来源和具体差异已读；checkpoint、TerminalOpen 的新源码及回归准备不等于实机通过。仍不支持所有 runtime 都兼容、所有原版模块逐源比完、全部恢复功能可用等结论。统一验证应在主线程确认全源码稳定后进行；历史 receipts 只说明其明确版本与当时执行范围。

## Native Files capture boundary (source only)

The follow-up source review found that the native string TextArea publishes its Model value only on a later build. bca41bb gives each editable FileTab a public TextBuffer, synchronizes capture/save/dirty-close, and marshals committed native data for both primary Tabs.Export and nested pane JSON. Preview remains the disk baseline; read-only/paged sources keep their existing string control. Runtime buffers are not serialized and never retain a current-Model pointer.

99f732e adds the narrow capture-only TextBuffer.CommittedSnapshot API: native preedit can temporarily remove a nonempty selection from the ordinary Snapshot, so excluding only the marked characters is insufficient. The committed capture retains the real transaction origin, including earlier committed native edits, across frames. e11ebc0 also preserves that origin across explicit-range candidate updates inside the same unfinished native transaction. Ordinary Snapshot/String/native publishing/undo semantics stay unchanged. Real ReplaceText/Unmark, external Set/Restore/Replace, and owner-scoped loadBuffer clear the capture override. The SDK has no separate public cancel callback; this record does not claim a new cancellation behavior.

Prepared regressions use the guarded native input adapter (headless getter only), rather than writing Draft or TextBuffer directly. They cover no-paint ReplaceText, nonempty-selection preedit, explicit candidate replacement, cross-frame IME, commit/Unmark, external replacement, stale editor ownership, undo/wrap, dirty close, read-only source, and nested pane export. The application fixture prepares the public Files surface -> native input -> no further Frame -> visibility capture helper -> existing debounced/CAS writer -> reopen the actual private recovery file, while checking the workspace file is unchanged. These regression sources have not been executed. No new runtime, real-window, or model-request acceptance is claimed.

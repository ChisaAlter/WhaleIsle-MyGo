# cmd/ inline Chinese text audit — 2026-10-26

Gap G52 (`docs/parity/gap-matrix-20261025.json`): the desktop entrypoint owns inline
Chinese copy outside the shared catalog. This audit covers **all of `cmd/`**, not only the
round-144 settings slice, and this round migrates the first high-visibility block.

| metric | before | after |
| --- | ---: | ---: |
| Chinese string literals | 194 | 145 |
| Non-test literals | 150 | 101 |
| Test-only literals | 44 | 44 |
| Files with at least one literal | 32 | 29 |
| Migrated to `internal/ui/i18n` | — | 49 literals / 48 call sites |
| New catalog keys (+ reused) | — | 45 new + 2 existing (`cmd.cancel`, `cmd.close`) |

A literal means one Go string literal containing a Han character or full-width/CJK
punctuation. The AST scan does not count comments, identifiers or documentation.

## 1. How the scan works

The scanner is a throwaway Go AST tool (`tmp/scan_cmd_inline/main.go`); it is not part of
the product build and is reproduced here so the numbers can be re-derived:

```go
// walk cmd/**/*.go, parser.ParseFile, then for every *ast.BasicLit of token.STRING:
//   strconv.Unquote(lit.Value) and keep it when hasHan(value)
// hasHan = unicode.Han || U+3000-U+303F || U+FF01-U+FF5E
```

```powershell
go run .\tmp\scan_cmd_inline\main.go .\cmd tmp\cmd-inline.json
```

Each hit carries file, line, enclosing function, innermost call name and the decoded
value. `_test.go` hits are counted separately and never treated as product copy.

## 2. Per-file inventory

Before is the scan that opened this round; after is the same scanner run after the
migration. A zero in the after column means every user-facing literal in that file is
now a catalog lookup.

| file | before | after | non-test / test | examples (after) |
| --- | ---: | ---: | --- | --- |
| `whaleisle-launcher/main.go` | 2 | 2 | 2 / 0 | `MyGo鲸屿 Launcher`<br>`MyGo鲸屿 Launcher` |
| `appearance_native_acceptance_windows_test.go` | 7 | 7 | 0 / 7 | `WhaleIsle 界面 中文 Aa 012345`<br>`会话正文 hello 世界` |
| `application_commands_acceptance_windows_test.go` | 2 | 2 | 0 / 2 | `你`<br>`preserve draft你` |
| `application_menu_test.go` | 16 | 16 | 0 / 16 | `打开工作区…`<br>`打开启动器` |
| `assistant_bot_notifications.go` | 1 | 1 | 1 / 0 | ` · 通知目标：` |
| `assistant_contacts.go` | 1 | 1 | 1 / 0 | `原 Host/账户已切换，操作未发送` |
| `component_changes.go` | 7 | 7 | 7 / 0 | `鲸桥组件管理仅支持 Windows x64`<br>`运行环境已改变，请重新检查组件并确认` |
| `composer_native_files_test.go` | 1 | 1 | 0 / 1 | `拼` |
| `git_push_review.go` | 5 | 0 | 0 / 0 | — |
| `host_bundle.go` | 3 | 3 | 3 / 0 | `其他生命周期操作正在进行，请稍后选择 Host`<br>`请先在启动设置中清空显式 dshBin/nodeBin，并移除 WHALEISLE_V…` |
| `image_document.go` | 5 | 4 | 4 / 0 | `正在校验并打开隔离图片快照，编辑草稿保持不变…`<br>`隔离图片预览未打开：` |
| `import_flow.go` | 4 | 0 | 0 / 0 | — |
| `import_plugins.go` | 10 | 10 | 10 / 0 | `正在重新安装插件 `<br>`；来源 ` |
| `launcher_window.go` | 3 | 3 | 3 / 0 | `正在启动鲸桥并验证就绪状态…`<br>`正在下载并校验鲸桥组件…` |
| `main.go` | 3 | 3 | 3 / 0 | `  The original file is preserved. Back up a…`<br>`导入尚未结束，暂不能启动内核。` |
| `market_batch.go` | 4 | 4 | 4 / 0 | `请选择已确认版本的更新。`<br>`无法保存批量操作记录；更新未开始。` |
| `market_catalog.go` | 29 | 29 | 29 / 0 | `中断记录保存失败，请查看历史。`<br>`运行时未连接，无法安装。` |
| `market_catalog_test.go` | 1 | 1 | 0 / 1 | `已应用` |
| `media_preview.go` | 6 | 1 | 1 / 0 | `媒体预览失败；当前 Host/backend 必须支持有界字节读取。可在 Files …` |
| `native_plugin_manager.go` | 6 | 6 | 6 / 0 | `子代理`<br>`网页搜索` |
| `notify_test.go` | 6 | 6 | 0 / 6 | `尚未连接 Host，请稍后重试`<br>`尚未连接 Host，请稍后重试` |
| `office_document.go` | 15 | 4 | 4 / 0 | `、`<br>`转换后的 PDF 超过预览传输大小上限；请缩小文件后重试。` |
| `pet_whale_test.go` | 4 | 4 | 0 / 4 | `鲸鱼娘`<br>`鲸鱼娘` |
| `plugin_contributions.go` | 5 | 3 | 3 / 0 | `插件内容`<br>`插件内容未连接：%s` |
| `plugin_creator_task.go` | 11 | 5 | 5 / 0 | `打开一个独立的 Creator 任务来创建插件`<br>`当前 Host 没有可用的 cordis 预设` |
| `plugin_session_host.go` | 11 | 2 | 2 / 0 | `未选中可用会话。`<br>`选择一个已挂载的内容。灰色条目没有当前 owner。` |
| `project_directory.go` | 1 | 0 | 0 / 0 | — |
| `resume_native_file_checkpoint_test.go` | 6 | 6 | 0 / 6 | `native 文件草稿`<br>`待提交` |
| `settings_body_wiring.go` | 1 | 1 | 1 / 0 | `目录读取失败` |
| `update_check_lifecycle.go` | 4 | 4 | 4 / 0 | `（`<br>`）` |
| `welcome_window_test.go` | 1 | 1 | 0 / 1 | `sk-中文` |
| `workspace_document_preview.go` | 13 | 8 | 8 / 0 | `正在转换 Office 文档并准备 PDF 预览…`<br>`正在准备 PDF 只读预览…` |

Totals: **145** literals in **29** files; **101** are
non-test and **44** are test-only. This round removed **49**
literals (194 → 145) across the ten files below.

## 3. Migrated to internal/ui/i18n

The migration is deliberately partial: **48 call sites / 49 literals** of the most
user-visible copy — confirmation dialog titles and buttons, plugin/Office/media preview
status and empty states, import errors and a native picker title. `pluginCopy`,
`pluginCreatorText`, `gitActionLabel` and `marketMessage` were ad-hoc zh/en helpers; the
migrated call sites now read the shared catalog instead. `gitActionLabel` became unused
and was deleted; the other three still serve the remaining deferred copy.

Every key below is registered in `internal/ui/i18n/catalog_cmd_copy.go` with both
locales. `internal/ui/i18n/catalog_cmd_copy_test.go` asserts non-empty zh/en, different
copy, no Han characters in the English text, and matching `%s/%d/%v/%q` verb counts.
`cmd/whaleisle/cmd_inline_copy_audit_test.go` asserts each call site reads the catalog and
that no migrated Chinese literal remains quoted in `cmd/`.

### Git push review dialog

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.confirm_push_destination` | `git_push_review.go:34` | 确认推送目标 | Confirm push destination |
| `cmd.confirm_push` | `git_push_review.go:36` | 确认推送 | Confirm push |
| `cmd.push_review_committed` | `git_push_review.go:29` | 将普通推送已提交内容，不会暂存或提交编辑器草稿。 | This pushes committed history without staging files or committing editor drafts. |
| `cmd.push_review_default_branch` | `git_push_review.go:31` | 目标是默认分支。仅在确认要更新它时继续。 | The destination is the default branch. Continue only if you intend to update it. |

### Session plugin dialog

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.session_plugin_title` | `plugin_session_host.go:156<br>plugin_session_host.go:168` | 会话插件 | Session plugins |
| `cmd.session_plugin_hint` | `plugin_session_host.go:173` | 局部内容使用原会话上下文；输入区由原页面完整接管，原生草稿保持不变。 | Local content uses the original Session owner. Its complete composer keeps a separate draft. |
| `cmd.session_content_unavailable` | `plugin_session_host.go:179` | 会话内容不可用。 | Session content unavailable. |
| `cmd.connecting_session_content` | `plugin_session_host.go:184` | 正在连接会话内容… | Connecting Session content… |
| `cmd.session_owner_unavailable` | `plugin_session_host.go:188` | 真实会话 owner 尚不可用。 | The original Session owner is unavailable. |
| `cmd.no_session_plugin_content` | `plugin_session_host.go:192` | 当前会话没有可显示的插件内容。 | No Session plugin content is available. |
| `cmd.session_plugin_content` | `plugin_session_host.go:211` | 会话插件内容 | Session plugin content |

### Plugin Creator panel

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.creator_needs_host` | `plugin_creator_task.go:74` | 连接 Host 后可创建插件 | Connect to the Host to create plugins |
| `cmd.creator_unavailable` | `plugin_creator_task.go:86` | 当前 Host 的 Creator 不可用 | Creator is unavailable on this Host |
| `cmd.creator_checking` | `plugin_creator_task.go:88` | 正在检查 Creator… | Checking Creator… |
| `cmd.creator_presets_unreadable` | `plugin_creator_task.go:90` | 无法读取 Creator 预设；请刷新预设列表 | Could not read Creator presets; refresh the preset list |
| `cmd.creator_result` | `plugin_creator_task.go:153` | Creator 创建结果 | Creator result |
| `cmd.creator_open_existing` | `plugin_creator_task.go:169` | 打开已有任务 | Open existing task |

### Office preview

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.office_preview_ready` | `office_document.go:45` | Office 转换完成，PDF 只读预览已打开。 | Office conversion finished; the read-only PDF preview is open. |
| `cmd.office_preview_missing_fonts` | `office_document.go:51` | 缺失字体：%s。本次预览无法使用这些字体，文字和排版可能与原文档不同。 | Missing fonts: %s. This preview cannot use them, so text and layout may differ from the original document. |
| `cmd.office_preview_unavailable` | `office_document.go:57` | Office 预览不可用。请在运行 DeepSeekHarness Go Host 的主机上启用文档预览服务，并确认 LibreOffice Kit 运行依赖可用；然后点击“转为 PDF 预览”重试。 | Office preview is unavailable. Enable the document preview service on the host running DeepSeekHarness Go, confirm the LibreOffice Kit runtime is available, then retry “Preview as PDF”. |
| `cmd.office_preview_cancelled` | `office_document.go:64` | Office 预览已取消；点击“转为 PDF 预览”可重试。 | Office preview was cancelled; retry with “Preview as PDF”. |
| `cmd.office_preview_too_large` | `office_document.go:72` | Office 文件或转换后的 PDF 超过预览大小上限；请缩小文件后重试。 | The Office file or the converted PDF exceeds the preview size limit; shrink the file and retry. |
| `cmd.office_preview_invalid` | `office_document.go:74` | 无法预览此 Office 文件。文件可能已损坏、受密码保护，或与扩展名不符。 | This Office file cannot be previewed. It may be damaged, password-protected, or not match its extension. |
| `cmd.office_preview_timeout` | `office_document.go:76` | Office 转换超时；点击“转为 PDF 预览”可重试。 | Office conversion timed out; retry with “Preview as PDF”. |
| `cmd.office_preview_busy` | `office_document.go:80` | Office 预览任务较多；请稍后点击“转为 PDF 预览”重试。 | Too many Office previews are running; retry with “Preview as PDF” later. |
| `cmd.office_preview_source_changed` | `office_document.go:82` | Office 文件在读取时已更改；请重新打开预览。 | The Office file changed while it was read; reopen the preview. |
| `cmd.office_preview_failed` | `office_document.go:84` | Office 转换失败，未生成可用的 PDF；请检查文件后点击“转为 PDF 预览”重试。 | Office conversion failed and produced no usable PDF; check the file and retry with “Preview as PDF”. |

### Workspace document preview

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.preview_switch_session_workspace` | `workspace_document_preview.go:102` | 请切回文档所属会话和工作区 | Switch back to the session and workspace that own this document |
| `cmd.preview_source_closed` | `workspace_document_preview.go:115` | 来源代理面板已关闭，请重新打开来源 | The source agent panel is closed; reopen the source |
| `cmd.preview_session_changed` | `workspace_document_preview.go:146` | 文档所属会话已改变 | The document's session changed |
| `cmd.preview_child_unsupported` | `workspace_document_preview.go:162` | 子会话文档入口当前只提供 PDF 和 HTML | The child-session document entry currently supports only PDF and HTML |
| `cmd.preview_restore_unsupported` | `workspace_document_preview.go:209` | 此预览类型尚不支持从源文件恢复 | This preview type cannot be restored from the source file yet |

### Media preview

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.media_path_required` | `media_preview.go:18` | 音视频预览需要当前会话工作区和明确支持的单文件路径；不接受 URL 或路径遍历。 | Audio/video preview needs the current session workspace and one supported file path; URLs and path traversal are rejected. |
| `cmd.media_host_service_required` | `media_preview.go:28` | 媒体预览需要当前会话的 Host 文件读取服务；不会把 Host 路径当成本机路径打开。 | Media preview needs the current session's Host file-read service; Host paths are never opened as local paths. |
| `cmd.media_reading_snapshot` | `media_preview.go:42` | 正在读取当前 Host/会话的单个音视频快照（最大 32 MiB）… | Reading one audio/video snapshot from the current Host/session (up to 32 MiB)… |
| `cmd.media_preview_unavailable` | `media_preview.go:80` | 媒体预览不可用，可在 Files 显式外部打开：%s | Media preview is unavailable; open it explicitly outside from Files: %s |
| `cmd.media_snapshot_opened` | `media_preview.go:82` | 只读音视频快照已打开。请点击播放器播放；编解码支持取决于 WebView2，30 分钟后过期。 | The read-only audio/video snapshot is open. Use the player to start playback; codec support depends on WebView2 and the snapshot expires after 30 minutes. |

### Image preview

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.image_preview_unavailable` | `image_document.go:66` | 当前系统的隔离图片 WebView 不可用；图片未在兼容预览中显示 | The isolated image WebView is unavailable on this system; the image was not shown in the compatible preview |

### Import flow

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.import_busy` | `import_flow.go:27` | 插件操作或运行时切换尚未结束，请稍后再导入。 | Plugin work or a runtime switch is still running; import later. |
| `cmd.import_scan_first` | `import_flow.go:33` | 请先扫描导入来源。 | Scan the import source first. |
| `cmd.import_nothing_selected` | `import_flow.go:41` | 未选择任何项，未写入。 | Nothing was selected; nothing was written. |
| `cmd.import_restart_blocked` | `import_flow.go:168` | 导入尚未结束，暂不能重启内核。 | The import has not finished; the core cannot restart yet. |

### Native picker and plugin contribution

| key | call site | zh | en |
| --- | --- | --- | --- |
| `cmd.choose_project_directory` | `project_directory.go:39` | 选择 Project 工作目录 | Choose Project workspace folder |
| `cmd.plugin_content_open` | `plugin_contributions.go:161` | 打开插件内容 | Open plugin content |
| `cmd.plugin_content_collapse` | `plugin_contributions.go:166` | 收起插件内容 | Collapse plugin content |

Two existing keys replaced the repeated button words, so the audit did not create
duplicates: `cmd.cancel` for the git-push cancel button and `cmd.close` for the session
plugin dialog close button. They are counted in the 48 call sites but not in the 45 new
keys.

## 4. Remaining items and why

| reason | count | meaning |
| --- | ---: | --- |
| `deferred` | 90 | user-visible, same pattern as the migrated block; next audit slice |
| `test` | 44 | test fixtures and assertions, not product copy |
| `punctuation` | 6 | punctuation/format fragments and one notification join, not sentences |
| `brand` | 2 | fixed product/window name |
| `bilingual` | 2 | one literal already carries zh and en together |
| `internal-match` | 1 | string matched against a Go error, never displayed |

### 4.1 `deferred` — user-visible, next slice (90)

These are real UI copy, but the agreed scope for this round was 30–50 occurrences. The
largest remaining clusters are the market catalog statuses (33), the launcher/WhaleBridge
lifecycle messages (14) and the plugin build-script approval dialog (10).

**`whaleisle/assistant_contacts.go`**

- `cmd/whaleisle/assistant_contacts.go:147` `assistantContactsModel` — `原 Host/账户已切换，操作未发送`

**`whaleisle/component_changes.go`**

- `cmd/whaleisle/component_changes.go:68` `changeWhaleBridge` — `鲸桥组件管理仅支持 Windows x64`
- `cmd/whaleisle/component_changes.go:79` `changeWhaleBridge` — `运行环境已改变，请重新检查组件并确认`
- `cmd/whaleisle/component_changes.go:92` `changeWhaleBridge` — `确认期间运行环境已改变，请重新检查组件并确认`
- `cmd/whaleisle/component_changes.go:102` `changeWhaleBridge` — `其他生命周期操作正在进行，已保留现有实例和已校验版本`
- `cmd/whaleisle/component_changes.go:106` `changeWhaleBridge` — `运行环境已改变，请重新操作`
- `cmd/whaleisle/component_changes.go:123` `changeWhaleBridge` — `操作未执行或任务保护未完成；请查看启动器提示后重试`
- `cmd/whaleisle/component_changes.go:164` `confirmBridgeManagement` — `无法显示组件操作确认；未更改组件`

**`whaleisle/host_bundle.go`**

- `cmd/whaleisle/host_bundle.go:125` `selectHostBundle` — `其他生命周期操作正在进行，请稍后选择 Host`
- `cmd/whaleisle/host_bundle.go:133` `selectHostBundle` — `请先在启动设置中清空显式 dshBin/nodeBin，并移除 WHALEISLE_VENDOR 覆盖；现有外部运行时未更改`
- `cmd/whaleisle/host_bundle.go:157` `selectHostBundle` — `运行环境已改变，请重新选择 Host`

**`whaleisle/image_document.go`**

- `cmd/whaleisle/image_document.go:36` `openLocalImage` — `正在校验并打开隔离图片快照，编辑草稿保持不变…`
- `cmd/whaleisle/image_document.go:54` `openLocalImage` — `隔离图片预览未打开：`
- `cmd/whaleisle/image_document.go:68` `openLocalImage` — `已打开隔离图片查看器（不代表编码解码成功）；`
- `cmd/whaleisle/image_document.go:68` `openLocalImage` — `；无脚本/网络/应用桥接。AVIF 显示取决于当前 WebView 编解码器，源文件与草稿不变`

**`whaleisle/import_plugins.go`**

- `cmd/whaleisle/import_plugins.go:22` `importPluginInstaller` — `正在重新安装插件 `
- `cmd/whaleisle/import_plugins.go:22` `importPluginInstaller` — `；来源 `
- `cmd/whaleisle/import_plugins.go:26` `importPluginInstaller` — `导入状态或运行时状态已改变，插件未安装。`
- `cmd/whaleisle/import_plugins.go:38` `importPluginInstaller` — `批准插件构建脚本`
- `cmd/whaleisle/import_plugins.go:39` `importPluginInstaller` — `插件：%s\n来源：%s\n目标：%s\n\n以下包要求在本机运行构建脚本，可能访问本机文件和网络：\n%s\n\n仅批准本次列出的脚本并重试该插件安装？`
- `cmd/whaleisle/import_plugins.go:40` `importPluginInstaller` — `跳过此插件`
- `cmd/whaleisle/import_plugins.go:40` `importPluginInstaller` — `批准并继续`
- `cmd/whaleisle/import_plugins.go:71` `installImportPlugin` — `安装器未返回有效的构建脚本清单；未批准。`
- `cmd/whaleisle/import_plugins.go:77` `installImportPlugin` — `已批准的构建脚本仍被拒绝；请检查安装日志后重试。`
- `cmd/whaleisle/import_plugins.go:81` `installImportPlugin` — `未批准构建脚本，跳过此插件。`

**`whaleisle/launcher_window.go`**

- `cmd/whaleisle/launcher_window.go:478` `launcherPanel` — `正在启动鲸桥并验证就绪状态…`
- `cmd/whaleisle/launcher_window.go:515` `launcherPanel` — `正在下载并校验鲸桥组件…`
- `cmd/whaleisle/launcher_window.go:525` `launcherPanel` — `鲸桥已安装并校验，可点击启动；未自动运行`

**`whaleisle/main.go`**

- `cmd/whaleisle/main.go:2629` `startHarness` — `导入尚未结束，暂不能启动内核。`

**`whaleisle/market_batch.go`**

- `cmd/whaleisle/market_batch.go:24` `runMarketBatchUpdates` — `请选择已确认版本的更新。`
- `cmd/whaleisle/market_batch.go:42` `runMarketBatchUpdates` — `无法保存批量操作记录；更新未开始。`
- `cmd/whaleisle/market_batch.go:57` `stopMarketBatch` — `已取消待执行的批量项目（%d）；当前安装可单独取消。`
- `cmd/whaleisle/market_batch.go:67` `clearMarketBatchLocked` — `待执行记录保存失败，请查看历史。`

**`whaleisle/market_catalog.go`**

- `cmd/whaleisle/market_catalog.go:66` `invalidateMarketLocked` — `中断记录保存失败，请查看历史。`
- `cmd/whaleisle/market_catalog.go:86` `runMarketCatalogInstall` — `运行时未连接，无法安装。`
- `cmd/whaleisle/market_catalog.go:88` `runMarketCatalogInstall` — `安装器正忙，请稍后重试。`
- `cmd/whaleisle/market_catalog.go:90` `runMarketCatalogInstall` — `正在刷新插件目录，请稍后重试。`
- `cmd/whaleisle/market_catalog.go:140` `beginMarketAttemptForOwner` — `构建授权已过期，请重新安装以确认。`
- `cmd/whaleisle/market_catalog.go:185` `beginMarketAttemptForOwner` — `无法保存操作记录；安装未开始。`
- `cmd/whaleisle/market_catalog.go:190` `beginMarketAttemptForOwner` — `无法保存操作记录；安装未开始。`
- `cmd/whaleisle/market_catalog.go:228` `marketResultFor` — `结果未知：`
- `cmd/whaleisle/market_catalog.go:232` `marketResultFor` — `结果未知，请刷新库存。`
- `cmd/whaleisle/market_catalog.go:250` `marketResultFor` — `操作失败。`
- `cmd/whaleisle/market_catalog.go:257` `marketResultFor` — `已应用；正在核对库存。`
- `cmd/whaleisle/market_catalog.go:261` `marketResultFor` — `已保存，需重启运行时后生效。`
- `cmd/whaleisle/market_catalog.go:264` `marketResultFor` — `此变更被更高优先级配置覆盖；请检查插件配置。`
- `cmd/whaleisle/market_catalog.go:267` `marketResultFor` — `安装已取消。`
- `cmd/whaleisle/market_catalog.go:270` `marketResultFor` — `操作失败。`
- `cmd/whaleisle/market_catalog.go:272` `marketResultFor` — `结果未确认，请刷新库存。`
- `cmd/whaleisle/market_catalog.go:278` `marketResultFor` — `需要明确授权以下构建脚本。`
- `cmd/whaleisle/market_catalog.go:301` `finishMarket` — `结果保存失败；历史仅在本次会话可见。`
- `cmd/whaleisle/market_catalog.go:312` `finishMarket` — ` 批量更新已停止，剩余 %d 项未执行。`
- `cmd/whaleisle/market_catalog.go:315` `finishMarket` — ` 批量更新暂停，等待此插件的授权。`
- `cmd/whaleisle/market_catalog.go:421` `runMarketInstallForOwner` — `已保存，需重启运行时后生效。`
- `cmd/whaleisle/market_catalog.go:424` `runMarketInstallForOwner` — `结果未确认。`
- `cmd/whaleisle/market_catalog.go:446` `runMarketChangeForOwner` — `插件库存不可用，请刷新。`
- `cmd/whaleisle/market_catalog.go:471` `runMarketChangeForOwner` — `无法保存操作记录；变更未开始。`
- `cmd/whaleisle/market_catalog.go:477` `runMarketChangeForOwner` — `无法保存操作记录；变更未开始。`
- `cmd/whaleisle/market_catalog.go:525` `cancelMarketInstall` — `安装已取消。`
- `cmd/whaleisle/market_catalog.go:542` `cancelMarketInstall` — `已取消，等待安装请求结束并核对库存。`
- `cmd/whaleisle/market_catalog.go:544` `cancelMarketInstall` — `已进入应用阶段，无法取消。`
- `cmd/whaleisle/market_catalog.go:546` `cancelMarketInstall` — `无活动安装请求；结果尚未确认。`

**`whaleisle/media_preview.go`**

- `cmd/whaleisle/media_preview.go:64` `openMediaFile` — `媒体预览失败；当前 Host/backend 必须支持有界字节读取。可在 Files 显式外部打开。`

**`whaleisle/native_plugin_manager.go`**

- `cmd/whaleisle/native_plugin_manager.go:163` `pluginManagerView` — `子代理`
- `cmd/whaleisle/native_plugin_manager.go:165` `pluginManagerView` — `网页搜索`
- `cmd/whaleisle/native_plugin_manager.go:167` `pluginManagerView` — `代理循环`
- `cmd/whaleisle/native_plugin_manager.go:169` `pluginManagerView` — `终端`
- `cmd/whaleisle/native_plugin_manager.go:171` `pluginManagerView` — `终端`
- `cmd/whaleisle/native_plugin_manager.go:247` `pluginManagerView` — `插件提供的操作`

**`whaleisle/office_document.go`**

- `cmd/whaleisle/office_document.go:89` `officePreviewFailure` — `转换后的 PDF 超过预览传输大小上限；请缩小文件后重试。`
- `cmd/whaleisle/office_document.go:94` `officePreviewFailure` — `Office 预览失败：`
- `cmd/whaleisle/office_document.go:94` `officePreviewFailure` — `。点击“转为 PDF 预览”可重试。`

**`whaleisle/plugin_contributions.go`**

- `cmd/whaleisle/plugin_contributions.go:170` `renderPluginContributionSurface` — `插件内容`
- `cmd/whaleisle/plugin_contributions.go:190` `pluginContributionNotice` — `插件内容未连接：%s`
- `cmd/whaleisle/plugin_contributions.go:198` `pluginContributionNotice` — `正在连接插件内容…`

**`whaleisle/plugin_creator_task.go`**

- `cmd/whaleisle/plugin_creator_task.go:83` `pluginCreatorAccess` — `打开一个独立的 Creator 任务来创建插件`
- `cmd/whaleisle/plugin_creator_task.go:92` `pluginCreatorAccess` — `当前 Host 没有可用的 cordis 预设`
- `cmd/whaleisle/plugin_creator_task.go:94` `pluginCreatorAccess` — `正在创建 Creator 任务…`
- `cmd/whaleisle/plugin_creator_task.go:96` `pluginCreatorAccess` — `请先查看下方保留的创建结果`
- `cmd/whaleisle/plugin_creator_task.go:159` `renderPluginCreatorNotice` — `定位已提交目标`

**`whaleisle/plugin_session_host.go`**

- `cmd/whaleisle/plugin_session_host.go:95` `openPluginSession` — `未选中可用会话。`
- `cmd/whaleisle/plugin_session_host.go:207` `renderPluginSessionSlot` — `选择一个已挂载的内容。灰色条目没有当前 owner。`

**`whaleisle/workspace_document_preview.go`**

- `cmd/whaleisle/workspace_document_preview.go:165` `readWorkspaceDocument` — `正在转换 Office 文档并准备 PDF 预览…`
- `cmd/whaleisle/workspace_document_preview.go:175` `readWorkspaceDocument` — `正在准备 PDF 只读预览…`
- `cmd/whaleisle/workspace_document_preview.go:200` `readWorkspaceDocument` — `正在打开已保存的 HTML 快照，编辑草稿保持不变…`
- `cmd/whaleisle/workspace_document_preview.go:226` `readWorkspaceDocument` — `Office 已转换，但 PDF 阅读器无法打开：`
- `cmd/whaleisle/workspace_document_preview.go:231` `readWorkspaceDocument` — `文档预览失败，可在 Files 使用系统打开：`
- `cmd/whaleisle/workspace_document_preview.go:233` `readWorkspaceDocument` — `静态 HTML 快照已打开；`
- `cmd/whaleisle/workspace_document_preview.go:233` `readWorkspaceDocument` — `；源文件与编辑草稿保持不变`
- `cmd/whaleisle/workspace_document_preview.go:235` `readWorkspaceDocument` — `PDF 只读预览标签已打开；原文件和编辑草稿保持不变`

### 4.2 `test` — test fixtures and assertions (44)

Test-only literals exercise Chinese input, menu label parity or fixture names. They are
not shipped copy and must stay in the test that asserts them.

| test file | literals |
| --- | ---: |
| `whaleisle/appearance_native_acceptance_windows_test.go` | 7 |
| `whaleisle/application_commands_acceptance_windows_test.go` | 2 |
| `whaleisle/application_menu_test.go` | 16 |
| `whaleisle/composer_native_files_test.go` | 1 |
| `whaleisle/market_catalog_test.go` | 1 |
| `whaleisle/notify_test.go` | 6 |
| `whaleisle/pet_whale_test.go` | 4 |
| `whaleisle/resume_native_file_checkpoint_test.go` | 6 |
| `whaleisle/welcome_window_test.go` | 1 |

### 4.3 `punctuation` — fragments, not sentences (6)

- `cmd/whaleisle/assistant_bot_notifications.go:290` — ` · 通知目标：`
- `cmd/whaleisle/office_document.go:47` — `、`
- `cmd/whaleisle/update_check_lifecycle.go:64` — `（`
- `cmd/whaleisle/update_check_lifecycle.go:64` — `）`
- `cmd/whaleisle/update_check_lifecycle.go:145` — `（`
- `cmd/whaleisle/update_check_lifecycle.go:145` — `）`

`update_check_lifecycle.go` wraps a version in full-width parentheses for zh and ASCII
parentheses for en at the call site; those four literals are formatting, not copy.
`office_document.go:47` is the `、` list separator. `assistant_bot_notifications.go`
joins ` · 通知目标：` onto a notification title.

### 4.4 `internal-match` — matched, never displayed (1)

- `cmd/whaleisle/settings_body_wiring.go:92` — `目录读取失败`

`settings_body_wiring.go:92` matches `目录读取失败` inside a Go error to choose a settings
state; localising it would break the match, and the user sees the formatted body, not
this token.

### 4.4 `brand` — fixed product name (2)

- `cmd/whaleisle-launcher/main.go:45` — `MyGo鲸屿 Launcher`
- `cmd/whaleisle-launcher/main.go:57` — `MyGo鲸屿 Launcher`

`cmd/whaleisle-launcher` is a tiny independent binary; its OS-level app name is the
product identifier and does not follow the interface locale.

### 4.5 `bilingual` — one literal, two languages (2)

- `cmd/whaleisle/main.go:2286` — `  The original file is preserved. Back up and repair config.json, then restart. / 原文件已保留，…`
- `cmd/whaleisle/main.go:2999` — `  Unstaged files and editor drafts are excluded; no push. / 不含未暂存文件或编辑器草稿，不会推送。`

Both startup messages already embed an English sentence and a Chinese sentence in the
same literal (separated by ` / `). Splitting them into a catalog key would change the
bilingual fallback behaviour of dialogs shown before the locale is known, so they are
left as-is and recorded here.

## 5. Re-running the audit

```powershell
. .\scripts\run-hidden.ps1
Invoke-HiddenTask -ScriptPath .\tmp\scan-cmd.ps1 -LogPath .\tmp\scan-cmd.log -TimeoutSeconds 300
python .\tmp\gen_audit.py   # rewrites this file from the fresh scan
```

A new run must keep section 3 in sync with `cmd_inline_copy_audit_test.go`: that test
fails when a migrated key disappears from the document or a quoted Chinese literal
returns to a migrated file.

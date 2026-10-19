# Files / Desktop 源码覆盖记录（2026-10-19）

本文件记录本次 WORK-03..12、DESK 只读审计实际覆盖的源码，补列 WORK-01/02 的未审阅范围。矩阵依据 `docs/native-full-function-acceptance-20261019.md`。这是源码覆盖记录，不是全模块还原或全功能正常使用的结论；不继承该验收表中的历史 PASS 为本次结果。

源码身份：初次审计以父线程分配的 `9abd649` 及并行工作区为起点；本记录落盘前读取的 HEAD 为 `4041667056b405cec2efb8ae30c0506267b0eadd`。工作区当时存在其他代理的未提交修改，不能将本记录解释为整个 HEAD 或工作区的全面审阅。原版只读目录为 `C:\AI\Deepseek-Harness-Desktop`。没有运行测试、构建、应用、服务，也没有模型、账户、设备或系统集成操作。

覆盖等级：

- **behavior-compared**：实际阅读了原版和 Go 对应动作的实现，比较了下表明确列出的行为；仍仅覆盖这些行为，不代表该行全部功能。
- **entrypoint-sampled**：阅读了部分入口、回调、状态代码或符号位置；尚未贯通原版与 Go 的完整行为链。
- **not-reviewed**：仅根据矩阵或路径枚举定位模块，未阅读足以进行行为判断的源码。

下表原版相对路径以原版目录为根；`C/` 表示 `vendor/deepseek-harness/packages/client/`。Go 相对路径以本仓库为根。目录或模块名定位不等于已经阅读其中全部文件。

| ID | 原版入口 | Go 入口 | 覆盖等级 | 本次实际覆盖、遗漏与未知 |
| --- | --- | --- | --- | --- |
| WORK-01 | `C/ui-workspace/src/client/WorkspacePicker.tsx`、`rows/WorkspaceBrowser.tsx:1427-1504`、`navigation.ts:437-476,553-584,629-634` | `internal/ui/workspacepicker/picker.go`、`cmd/whaleisle/{workspace_navigation,no_directory}.go`、`internal/store/session/{workspace_navigation,workspace_follow,no_directory}.go` | behavior-compared | 追加精读添加/选择、改名校验、删除回执与列表回声、无目录 scratch baseline/blank reuse、旧导航取消与失败保留。确认并修复移除当前 Workspace 后原会话视图未清理：源码 `3a7a94d`；只在回执+权威列表都确认、成员与 selection/Host 归属仍相符时清空视图，不删会话。专有回归准备 `7aa7e62`，未执行。任意 feature directoryActions 插件及全部 sidebar 动画/排序辅助实现未逐一对照。 |
| WORK-02 | `C/ui-directory-picker-browse/src/client/{DirectoryBrowser.tsx,flow.ts,index.ts}` | `cmd/whaleisle/directory_picker.go`、`internal/ui/dirpick/{panel,state}.go` | behavior-compared | 追加精读原版 DirectoryBrowser:448-694 与 Go 完整扫描/两腿 landing/选择/路径 debounce/取消/原名创建/刷新/失败重试/Host generation 链；核对隐藏与点前缀、路径 Enter/Escape、双列和创建 UI。未发现此已读核心链的确定阻断。Remote 子 flow 的 SSH/machine 实现不在本次目录浏览对照范围，CSS/所有焦点布局细节未逐行对照。 |
| WORK-03 | `C/ui-files/src/client/FileTree.tsx`、`apply.ts` | `internal/ui/files/files.go`、`tree_actions.go`、`preview_toolbar.go` | behavior-compared | 比较原版 FileTree 的只读展开/打开、引用、复制路径、打开方式入口，与 Go 树行/context menu/拖拽引用回调。原版没有应用内创建、改名、隐藏项开关，不能据此推定缺口。未贯通完整异步树刷新、外部编辑器识别与系统调用结果。 |
| WORK-04 | `C/ui-files/src/client/apply.ts`、`fileSaveCoordinator.ts`、`FileClosePrompt.tsx` | `internal/ui/files/autosave.go`、`reload.go`、`tabs.go`；`internal/ui/frame/frame.go` 的 `OnSaveEdit` | entrypoint-sampled | 阅读原版注册保存和关闭守卫入口；Go 阅读保存队列、外部修改时保留脏基线、错误/截断阻止保存及本地写回调。原版协调器/关闭提示主体、全部关闭与恢复链未对照。Go 远程预览保存明确转交 Remote 文件编辑器；未贯通原版同场景，不认定为缺失。 |
| WORK-05 | `C/ui-sidebar-documentpreview/src/client/{text,code,markdown}/index.ts` 及各 `*Body.tsx`、`TextPreview.tsx:130-203`；`C/ui-files/src/client/FilePreview.tsx:200-344,480-564` | `internal/ui/files/{source_actions,text_read,text_read_state,text_read_line,text_pages,text_preview,markdown_metadata,preview_toolbar}.go`、`internal/desktop/workspacefs/text_page.go` | behavior-compared | 追加阅读正文 renderer、分页/版本/只读、行定位、源码选择/引用、换行/保存入口及 Markdown frontmatter。确认原版自动继续分页到目标行而 Go 夹到已载入末行；由另一 owner 源码修复 `a00c313`，专有记录 `source-coverage-files-line-navigation-20261019.md` 与准备回归 `8239234`，本代理读到提交差异，未执行。原语级全部语法高亮/Markdown 扩展/搜索实现与每种格式视觉仍未逐一对照，不能借本行认定渲染完全一致。 |
| WORK-06 | `C/ui-sidebar-documentpreview/src/client/{index.ts,TextPreview.tsx}`、`document/{registry,unviewable}.ts`、`{html,image,pdf,office,excel}/index.ts` 及下文各主体 | `internal/ui/files/{media,image_read,image_source,document_surface,table_preview,table_view,text_read,reload}.go`；`cmd/whaleisle/{document_viewer,workspace_document_preview,remote_image,remote_workbook,office_document}.go`；下文具体 helper | behavior-compared | 已贯通注册/分派、会话读取、Native/Document/Auto、HTML静态/交互入口、图片/PDF错误与重试、Office→PDF、只读表格及外部打开。实际默认图片误读本地来源 gap 以 `84fd323` 修复，保留 MyGo 图片 renderer。原版 Office 有条件支持 Word/PowerPoint，AVIF 明确 unsupported。未逐行对照解码库算法/全部资源 pack，未验证每个格式样本；GPU/GIF 深度仍是父线程暂停项。 |
| WORK-07 | 矩阵所列 `ui-sidebar-right`、`ui-dockkit`；`C/ui-files/src/client/floating-preview.ts` | `internal/ui/files/{tabs,floating}.go`、`internal/ui/frame/workspace_panes.go` | entrypoint-sampled | 只抽样标签关闭符号、保存后关闭及浮动预览分派；停靠、尺寸、拖拽、浮动生命周期、恢复序列与原版 dock 契约未完整阅读。 |
| WORK-08 | `C/ui-sidebar-browser/src/client/browser/{BrowserController,BrowserNavigation,BrowserPersistence}.ts`、`electron/workspace.ts` | `internal/ui/browserpreview/{actions,recording,workspace_scope}.go`、`cmd/whaleisle/browser_workspace.go` | entrypoint-sampled | 阅读 Go 截图/元素/录制取消、资源归属检查、导出状态和可信 workspace owner；原版浏览器文件仅定位。导航、标题/历史、分区存储、失败与恢复的双方状态链未完整对照。 |
| WORK-09 | `C/ui-git/src/client/{apply.ts,BranchMenu.tsx,CreateBranchDialog.tsx}` | `internal/desktop/git/`、`internal/ui/diffsurface/repository_actions.go` | entrypoint-sampled | 只定位状态/diff/分支模块与动作函数；没有阅读完整 Git 命令和原版 UI 状态链，不作该模块无缺口的结论。 |
| WORK-10 | `C/ui-git/src/client/{CommitDialog,PublishDialog}.tsx` | `internal/desktop/git/publish.go`、`internal/ui/diffsurface/{repository_actions,remote_workflow}.go` | entrypoint-sampled | 只抽样 commit/publish/review 动作与取消入口；授权、推送、失败恢复、PR/stack 完整链未审阅。未进行任何远端操作。 |
| WORK-11 | `C/ui-user-terminal/src/client/{TerminalDrawer,TerminalPane}.tsx`、`apply.ts`、`shell.ts` | `internal/ui/terminal/{panel,workspace}.go` | entrypoint-sampled | 阅读 Go 新建、四 pane 上限、横纵 split、激活、关闭、restart 和 cwd 保留；原版主体未对照。终端 DLL、ConPTY、输入/IME、resize 与隐藏恢复未审阅主体。 |
| WORK-12 | 矩阵所列 `ui-sidebar-terminal`、`settings-shell`（本次未解析全部具体文件） | `internal/ui/terminal/{host,host_shells,host_rename,host_selection}.go`、`cmd/whaleisle/terminal_target.go`、`internal/store/hostterminal/` | entrypoint-sampled | 抽样 Go 选择/刷新 shell、连接/重连/接管/关闭、复制/引用、选择目标及权限禁用入口；只完整阅读 `host_shells.go`。Host duplex、归属释放和完整权限状态机未贯通；不继承历史 runtime 结果。 |
| DESK-01 | `src/main/menu.js` | `cmd/whaleisle/application_menu.go` | behavior-compared | 完整阅读双方菜单模板及 Go 路由回调，比较工作区、启动器、设置、市场、MCP、技能、重启、视图、帮助动作。Go 刷新为重绘且保留草稿的明确实现。未贯通每个下游 controller 或原生菜单 SDK 分发。 |
| DESK-02 | `src/main/tray-menu.js` | `cmd/whaleisle/desktop_tray{,_menu}.go` | behavior-compared | 完整阅读双方托盘菜单及 Go 显示/各页面/重启/pet/退出回调、托盘失败和关闭到托盘入口。未完整阅读 notice、任务退出守卫与原生 tray SDK 主体。 |
| DESK-03 | `src/main/{window,close-behavior,native-window-motion}.js` | `internal/ui/chrome/`、`internal/desktop/taskguard/`、`cmd/whaleisle/main.go` | entrypoint-sampled | Go 仅抽样窗口关闭菜单、show/restore/focus 与 hide-to-tray 路径；原版窗口实现仅定位。caption、drag/resize/snap、任务退出与多窗口焦点整体未审阅。 |
| DESK-04 | `src/main/shortcuts.js`、矩阵所列 `ui-shortcuts` | `cmd/whaleisle/application_menu.go`、`web_application_shortcuts.go`、`internal/{shortcuts,shortcutprefs}/` | entrypoint-sampled | 阅读菜单 bindings resolve、keycap、pending command dispatch 与原版 commandItem。全局快捷键注册、编辑器/IME/modal/focus 优先级、配置冲突和命令参考未贯通。 |
| DESK-05 | `src/main/index.js`、原版 main/preload 系统集成入口 | `internal/desktop/deeplink/parse.go`、`cmd/whaleisle/main.go` 的 `applyDeeplink`/单实例/URL scheme 注册 | entrypoint-sampled | 阅读 Go URL 解析和各 Action 路由、定位单实例与注册回调。原版接收/转发、开机启动、外部打开和 OS 注册主体未审阅。Session/Agents 路由已由父线程修复，见下节；不是本行全功能关闭。 |
| DESK-06 | `src/main/update.js:254-355,600-833`、`update-updater.js:284-388`、`update-dialog.js`；`build/installer.nsh`、原版 `installer-directories.nsh` | `cmd/whaleisle/{native_releases,update_check_lifecycle,update_attention,task_protection}.go`、`internal/desktop/update/{update,apply,start_staged,native_releases}.go`、本仓库 NSIS 事务 helper | behavior-compared | 追加精读 check single-flight、版本固定、full/delta 下载入口、取消、校验与缓存、下载后确认/任务保护/启动前重验，以及 Go 真实 native release UI 回调、选择旧版本与 staging 状态。发现安装目录覆盖/回滚缺口并在 NSIS 源码补齐，见下节。原版 `downloadFile` 内部所有续传/重试分支、CNB 镜像适配、完整启动通知与发布端接口仍未全量逐行对照。源码进度不等于下载/签名/安装通过。 |
| DESK-07 | `build/installer.nsh` 全文及 `remove-directory.nsh`；`vendor/deepseek-harness/apps/desktop/scripts/installer-directories.nsh`；`scripts/windows-directory-installer.cjs`；原版 pinned `app-builder-lib/templates/nsis/uninstaller.nsh` | `scripts/nsis/whaleisle.nsi`、新增 `payload-transaction.nsh`、准备的 `transaction_source_test.go` | behavior-compared | 追加精读原版目录 stage/promote/rollback/commit、逐失败卸载和不跟链接清理，比较品牌/中英语言/许可/目录/结束页及真实 native Install/Uninstall sections。补齐自有 manifest 限定的目录事务与可重试失败管理，commit `b62ec7f`、`472ea92`、`863ce35`。原版完整多用户/elevation/坏记录 hygiene 支线和全部 electron-builder 生成模板未全面移植或逐行认证；未知/非 registered 旧管理入口不猜动。未 makensis 或安装/卸载。 |

## 已关闭项与排除项

- **已关闭（仅源码修复）**：父线程确认 Session/Agents deep link 需要回到 conversation workspace；旧工具页会在该工作区之前返回，导致会话/Agents 目标不可达。`e322792aa4e94394d9ffa132c1d28abe1a63eae7` 在 `applyDeeplink` 中仅针对 `ActionSession`/`ActionAgents` 清理 IM、Remote、Market、WebView、Browser、Assistant 和工具菜单路由，并推进 Assistant conversation generation。本次用 `git show e322792 -- cmd/whaleisle/main.go` 阅读了实际九行修改；没有执行验证。`ActionOpen` 保持显示当前页。
- **排除，非 bug**：先前提出 Settings/Launcher 未清旧路由 flag 的候选不成立。父线程实际追踪后确认 Settings Modal 最后无条件绘制，Launcher 在前置分支 early return，不能仅由残留 flag 推断被遮蔽。本记录不将该候选列为待修缺口；该绘制顺序核实来自父线程，非本代理独立复核。
- 除下文已记录的源码修复，本代理没有另外确认可直接分配修复的 WORK/DESK 阻断。未确认缺口不等于没有缺口；`entrypoint-sampled` 及各行明确列出的未读行为仍然是源码覆盖欠账。

## 使用边界

本表覆盖了所列矩阵 ID 的登记，不覆盖全部原版实现。下一轮应针对未贯通的具体行为链阅读原版实现与 Go 实现，再决定修改；不能把本轮入口抽样、其他代理模块收尾、已有测试文件或历史验收记录改写为“全部原版模块源码已覆盖”。本次不准备或执行 QA；用户要求先完成全部源码修改，再统一验证。

## 追加精读与源码修复（本轮后续）

父线程要求继续读取 WORK-01/02/05、DESK-06/07，后追加 WORK-06 的原版和 Go 核心实现，对应行已更新为 **behavior-compared**，含义仍是明确列出的行为已作源码对照，而非整行所有辅助模块逐行看完。当前登记数为：9 行 behavior-compared、10 行 entrypoint-sampled、0 行 not-reviewed；不能把这一登记统计写成“19 行全部已完成还原”，未贯通范围仍逐行保留。

### 工作区移除与跨页行定位

- 原版 `ui-workspace/navigation.ts:629-634` 在删除 Workspace 成功后，仅当当前 main 仍是该 Workspace 原有成员时 `clearMain`。Go 原来的 `ConfirmWorkspaceRemoval` 与 remove echo 只清 Workspace ID，不清所选会话。`3a7a94d` 捕获删除前成员与 selected，继续保留既有 client/generation/token 校验，并等待成功回执与权威列表不含该 ID 后走 `selectSessionLocked("")`。新导航、Host 更换、其他 Workspace 的所选会话及失败回执均不清；会话本体和配置目录保留。
- `7aa7e62` 准备六种回归情形，并把已有 fixture 中“成功删除仍保留该会话视图”的旧断言改为原版行为，额外断言会话条目仍存在。没有执行这些测试。
- 原版 `TextPreview.tsx:136-158` 在行目标超出已加载页面时继续加载到目标或 EOF；原 Go `requestLineNow` 会直接夹到当前末行。该缺口交由父线程分配其他 owner，产品提交 `a00c31309b8d1fede7511b94b9f74fac03b852be`，准备回归与独立附录提交 `8239234a4bcd021e05532612065e4d607c923210`；本代理实际阅读了前者的目标状态与调用差异，没有代其声称测试通过。

### NSIS 安装、回滚和卸载失败语义

实际原版依据：`installer-directories.nsh:95-132` 先生成新 stage，`:170-215` 才 backup/promote，失败走 `:148-168` restore；宿主 `scripts/windows-directory-installer.cjs:27-49` 强化恢复失败保留 backup，且仅在新安装 commit 后将旧目录 retirement 失败降为明确警告。`build/installer.nsh:117-176` 的卸载清理失败用非零状态和 Abort 保留管理；原 pinned `uninstaller.nsh:160` 起先执行清理，再删除安装注册。

本次确认的真实缺口是原 native `.nsi` 向现有 `$INSTDIR` 逐文件覆盖、卸载直接 `RMDir /r` 并删除管理记录，没有上述目录事务和失败语义。父线程随后明确授权本代理修改 `.nsi` 与专用 helper，源码变更为：

- `b62ec7f`：程序载荷先写唯一同级 `.new-{GUID}`，从该 fresh stage 生成 UTF-16 自有文件 manifest；生成失败中止。随后 backup 原目录、promote 新目录，提取/promote/注册/快捷方式失败走恢复；恢复本身失败保留 backup 并报告实际路径。只从 manifest 白名单根删除自有文件与空目录，逐级拒绝 reparse，不递归清理现有未知目录；normal/extended Windows 与 UNC 路径分别处理。
- `472ea92`：现有目标目录无法枚举时停止，不能把读取失败推断为空目录。旧无 manifest 的 backup 保留并明确提示，不推定其内容均可删除。
- 独立 fresh source review 发现：保留下来的旧卸载程序可能改指或删除新安装的同 App ID 注册/快捷方式。`863ce35` 添加 stage GUID 的 manifest identity 与注册 `NativeInstallID`；任何卸载重指/删除全局管理前检查自身捕获的 source path、identity 与当前两处 InstallLocation。对**当前已注册且 exact quoted UninstallString 归属该旧根**的 pre-ID 旧入口，在 backup 后、promote 前改为非可执行 `.retired-uninstaller`；失败中止，rollback 先恢复该入口再恢复目录。未知/非 registered 旧文件不猜测改名或删除。有匹配 manifest identity 的未来受控旧卸载器由 identity guard 限制。
- 卸载 payload 清理失败保留注册和快捷方式；仅在它们仍属于本次源安装时重指实际 `.uninstall-{GUID}` 的重试入口。错误文本承认部分自有文件可能已删除，不能假称完整恢复；成功清理后才允许注销本次源安装。已 commit 的新安装不因旧文件 retirement 警告而回滚。

`transaction_source_test.go` 仅准备了静态包装契约回归源码（顺序、manifest/no-follow、失败管理保留、旧身份与 legacy 入口退休）。这些测试不代替 Windows 安装/卸载和故障注入。本轮只执行过源文件读写、Git 和 gofmt，**没有运行 tests、build、makensis、安装器、应用、服务或账户/系统动作**。独立 reviewer 已只读复核已报旧入口问题的修复；这仍然不是编译或运行证据。

剩余范围必须保留原样：原版所有辅助实现仍未逐行对照，编译/真实安装回滚/卸载失败验证待用户要求的统一验证阶段；当前没有工具阻断，但不能将“按要求未执行”写成验证通过，也不能反推为某功能缺失。

### WORK-06 追加格式对照（产品源码 `84fd323`）

原版共同入口：`ui-sidebar-documentpreview/index.ts:124-131` 注册 renderer；`TextPreview.tsx:69-80,118-125` 根据 live registry、后缀/扩展优先级和 binary 声明选择实现，再分派 text-pages / bytes-complete / renderer。没有匹配实现的已知二进制显示 unsupported 和 Host-open contribution（`:206-225`）；`unviewable.ts` 的 Office 后缀仅在没有 renderer 时生效，不能误读成 Office 一律不可预览。`ui-open-in-app/index.ts:139-157` 实际注册文档 action/empty action，后者 `OpenPathEmptyAction.tsx:19-20` 复用打开菜单。

Go Files 的 `document_surface.go` 提供 Native/Document/Auto。`cmd/whaleisle/document_viewer.go:11-29` 验证 Host client/endpoint/session 后建页；`pluginslots/document_viewer.js:27-85` 保留原版 TextPreview 为 renderer/读取/订阅 owner，按该文档 session+path 采纳原 seat。`webview/client_modules.go:38-49` 仅将 live 扩展 renderer 或已挂载内建表格推荐为自动 viewer。这是实际原组件载体，不以目录存在当成实现证明。

| 格式 | 已读原版契约 | 已读 Go 对应链与边界 |
| --- | --- | --- |
| HTML | `html/index.ts` 注册 html/htm；`HtmlBody.tsx:35-102` 默认 basic sandbox iframe，developerTools 启用后 pack 相对资源到 opaque script iframe；内容/模式变化撤销旧 Blob，UTF-8/pack 失败显示错误。 | `workspace_document_preview.go:198-206` scoped Host snapshot；`htmldocument/{document,remote}.go` 资源快照/lease，remote:30-55 为相对资源保留同一 Host/session reader；`browserpreview/{html,interactive_html}.go` 静态结果、显式交互手势、helper 缺失/失败提示和 tab/owner 取消。未逐行比较全部 CSS/module pack 或 helper 内核。 |
| 图片 | `image/index.ts` 注册 png/jpg/jpeg/gif/webp/bmp/ico/svg；`ImageBody.tsx:68-130` 完整 bytes→Blob/img、加载/失败、tab zoom、卸载释放；SVG 为静态 img。AVIF 未注册，并在 `document/unviewable.ts:27` 明确列为未支持。 | `files/media.go` 原有格式/字节/像素限制与 GIF/WebP timeline，`image_read.go` 单 worker/最新 generation/cancel，`reload.go` 清旧像素后重读。发现 native 默认仅本地读取；`84fd323` 新 `ReadImageContext` / `remoteImageReader`，所有当前 Host（包括 local Host）权威 `ReadSnapshot`，失败绝不回退本机同名文件，复用既有解码器和动画。缓存/请求均绑定 client/session/follow generation/root/path/tab occurrence；未扩 AVIF 契约或恢复暂停的 GPU/GIF 深度任务。 |
| PDF | `pdf/index.ts` bytes-complete；`pdf.tsx:45-102,105-205` load、lazy page/text canvas、fit-width/zoom、tab page/scroll、load/page failure retry 与 dispose。 | `workspace_document_preview.go:173-185` scoped bytes；`pdfpreview/document.go:119-147` 验证快照/建 lease，`reader.go` 仅产品自有 assets；已读非 minified `reader.js` 的 page/zoom/selection/可见页 render/worker error/retry/dispose。`browserpreview/{pdf,workspace_documents}.go` 负责 expiry、owner、Reread、替换成功才释放旧能力与 source 恢复。未审 PDF.js 算法或验证像素。 |
| Office | `office/index.ts:70-117` remote.officeToPdf/workspaceFiles 条件注入，read授权+stat版本校验、generation cache/reset/dispose；注册 doc/docx/ppt/pptx，provider 缺失返回 unavailable；`OfficeBody.tsx:44-77` 共享 PDF、失败 reload；index:121-130 分类错误。 | `session/office_pdf.go:43-95` captured selected client/session→generation/render→完整 multipart PDF bytes/缺失字体；`office_document.go:42-88` 映射 size/invalid/timeout/unavailable/busy/source-changed 等；`workspace_document_preview.go:159-172,220-228` shared PDF 和完成前 owner 复核。原文档 viewer 仍可选。不把外部 Office 编辑器/账号当成原版预览契约；未调用真实转换。 |
| Excel | 独立 `excel/index.ts` 注册 xlsx/xls/csv/tsv；`LazyExcelBody.tsx` 按扩展分格式；`parse.ts:14-59` worker/大小/超时/取消；`excel.tsx:21-65` parse failure retry、unsupported notice、sheet tabs/formula bar/copy，allowEdit=false/forceCalculation=false。 | 自动推荐原版 live spreadsheet owner；native fallback 走 `remote_workbook.go` scoped snapshot→`workspacefs/workbook.go` 格式分派→`files/{text_read,table_preview,table_view}.go` 只读分页/sheet/selection/copy/error/reload。均不计算公式；未逐行对照全量 XLS/XLSX/style parser，也未对任意工作簿证明视觉一致。 |

实际 gap 的源码证据：修复前 `files/image_read.go:151-154` 无条件调用本地 `loadImagePreviewContext(root,path)`，后者使用 os/EvalSymlinks；不会走原版按当前会话 `workspaceFiles.readBytes` 的来源。内建图片不满足自动 document recommendation，故默认 remote 图片失败或读取本地同名文件。`84fd323` 通过 main→frame→Files 接线有界字节 reader；`mediapreview/reader.go:80-121` 复用现有 scope/stat/range/version/final-stat 校验，无 reader 也提供明确 error closure，不能误判为 local fallback。SVG 1 MiB、其余图片 16 MiB 和像素限制保持原规则。原本不传 callback 的纯 local fixture 入口仍可读本地。

回归/OS 分层：新增专有 `internal/ui/files/image_source_test.go`、`cmd/whaleisle/remote_image_test.go` 仅准备 fake-reader 正向解码、同名本机不读取、错误/超限/取消、Host/session变化与同 client/session 新 generation 清缓存等源码。既有测试文件不算执行证据。本轮只读写源码、格式化和读取 Git 差异，未执行 tests/build、真实文件预览、Host/网络调用、预览服务、Office 转换、系统打开或账号操作。当前没有工具阻塞；OS/renderer/转换验证仍待统一验证阶段。

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
| WORK-01 | `C/ui-workspace/src/client/WorkspacePicker.tsx` | `internal/ui/workspacepicker/picker.go`、工作区导航 | not-reviewed | 只定位入口；添加、切换、重命名、移除、无目录及持久化失败链均未审阅，不借用历史验收结果。 |
| WORK-02 | 矩阵所列 `ui-directory-picker-browse`；本次未解析具体实现文件 | `internal/ui/dirpick/panel.go`、`cmd/whaleisle/directory_picker.go` | not-reviewed | 只定位入口；Miller 浏览、隐藏项、新建目录、路径输入、失败重试及焦点行为未审阅。 |
| WORK-03 | `C/ui-files/src/client/FileTree.tsx`、`apply.ts` | `internal/ui/files/files.go`、`tree_actions.go`、`preview_toolbar.go` | behavior-compared | 比较原版 FileTree 的只读展开/打开、引用、复制路径、打开方式入口，与 Go 树行/context menu/拖拽引用回调。原版没有应用内创建、改名、隐藏项开关，不能据此推定缺口。未贯通完整异步树刷新、外部编辑器识别与系统调用结果。 |
| WORK-04 | `C/ui-files/src/client/apply.ts`、`fileSaveCoordinator.ts`、`FileClosePrompt.tsx` | `internal/ui/files/autosave.go`、`reload.go`、`tabs.go`；`internal/ui/frame/frame.go` 的 `OnSaveEdit` | entrypoint-sampled | 阅读原版注册保存和关闭守卫入口；Go 阅读保存队列、外部修改时保留脏基线、错误/截断阻止保存及本地写回调。原版协调器/关闭提示主体、全部关闭与恢复链未对照。Go 远程预览保存明确转交 Remote 文件编辑器；未贯通原版同场景，不认定为缺失。 |
| WORK-05 | `C/ui-sidebar-documentpreview/src/client/{text,code,markdown}/index.ts` | `internal/ui/files/source_actions.go`、`text_read.go`、`text_read_state.go`、`preview_toolbar.go` | entrypoint-sampled | 阅读 Go 行定位、Markdown 源码/渲染入口、换行、异步读取和脏草稿基线保护。原版注册模块仅定位；搜索、选择、完整 Markdown 渲染与只读边界未完成双方行为对照。 |
| WORK-06 | `C/ui-sidebar-documentpreview/src/client/{html,image,pdf,office,excel}/index.ts` | `internal/ui/files/{media,image_read,image_view,table_preview,document_surface}.go`；`internal/desktop/{pdfpreview,mediapreview,webview}/` | not-reviewed | 主要为格式分派路径定位，未审阅各格式解码/导航/转换主体。不能以文件名或历史格式验收推定还原。GPU/GIF 深度项目为父线程已有暂停项，本次未核实或扩展。 |
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
| DESK-06 | `src/main/update*.js`、`update-dialog.js` | `internal/desktop/update/`、`cmd/whaleisle/update_attention.go`、`internal/startuprec/` | not-reviewed | 仅定位矩阵/文件；检查、下载、取消、签名、安装、回滚和恢复均未在本次审阅。 |
| DESK-07 | `build/installer.nsh`、原版 packaging resources | `scripts/nsis/whaleisle.nsi`、`scripts/nsis/branding/` | not-reviewed | 仅定位文件；语言、许可、欢迎/结束、升级及卸载行为均未审阅。 |

## 已关闭项与排除项

- **已关闭（仅源码修复）**：父线程确认 Session/Agents deep link 需要回到 conversation workspace；旧工具页会在该工作区之前返回，导致会话/Agents 目标不可达。`e322792aa4e94394d9ffa132c1d28abe1a63eae7` 在 `applyDeeplink` 中仅针对 `ActionSession`/`ActionAgents` 清理 IM、Remote、Market、WebView、Browser、Assistant 和工具菜单路由，并推进 Assistant conversation generation。本次用 `git show e322792 -- cmd/whaleisle/main.go` 阅读了实际九行修改；没有执行验证。`ActionOpen` 保持显示当前页。
- **排除，非 bug**：先前提出 Settings/Launcher 未清旧路由 flag 的候选不成立。父线程实际追踪后确认 Settings Modal 最后无条件绘制，Launcher 在前置分支 early return，不能仅由残留 flag 推断被遮蔽。本记录不将该候选列为待修缺口；该绘制顺序核实来自父线程，非本代理独立复核。
- 本代理没有另外确认可直接分配修复的 WORK/DESK 阻断。未确认缺口不等于没有缺口；特别是 `entrypoint-sampled` 和 `not-reviewed` 的原版行为仍然是明确的源码覆盖欠账。

## 使用边界

本表覆盖了所列矩阵 ID 的登记，不覆盖全部原版实现。下一轮应针对未贯通的具体行为链阅读原版实现与 Go 实现，再决定修改；不能把本轮入口抽样、其他代理模块收尾、已有测试文件或历史验收记录改写为“全部原版模块源码已覆盖”。本次不准备或执行 QA；用户要求先完成全部源码修改，再统一验证。

# M2 主路由与 A2 生命周期

2026-11-17，导航分线交接。实现属于 `codex/route-a-system-optimization` 工作区，未单独提交。

## 已实现

- `main_route.go` 统一工作台互斥页面。旧 `*Open` 字段仅由一个 setter 投影；侧栏、菜单、深链接、启动/恢复、桌宠、原生设置、文件预览与 assistant 跳转均已接入。文件标签、会话草稿和旁栏布局继续独立保留。
- 路由 epoch 撤销旧的异步开窗资格。账号平台结果和半建窗口到达时必须仍属于当前路由与 Host 代次；账号 lease 幂等关闭，取消后拒绝接纳迟到 Gateway/窗口。
- 兼容 Host 在壳锁内分离，在锁外撤销 Gateway 和销毁窗口；资源收尾在 UI 线程执行。短暂确认/目录弹层只遮挡当前表面，实际导航离开释放它。
- Browser Controller 新增 `Suspend`，离开独立浏览器路由立即释放普通 WebView，保留标签、文档描述和草稿。显式独立 mini-player 维持既有寿命；普通浏览器标签切换仍用已有 TTL。
- 插件开窗和撤销使用 `Coordinator.InstalledPluginState` 的完整包库存 Owner/Revision，避免把 Module 当包名。未知、读取中或失败库存不撤销；已知禁用/缺失拒绝打开，新的完整库存可撤销现有窗口。
- Host 换代撤销旧兼容连接；重建仍要求当前显式兼容路由。重复启动通知不再抢走用户已打开的原生页。
- assistant 关闭、打开标准会话及 bot/room 跳转通过原有 `RevealMainConversation` 回调同步主路由，防止 Service.open 与主路由各自决定当前页。

## 主程序整合

已按其他工作流 owner 的接口接线：`AttachMux` 后绑定提交恢复；刷新回调核对提交结果；当前 composer draft 解锁后绑定文件准备流程。Creator、远程、凭据和 Project owner 已有修改均保留。

`main.go` 编辑权已交回父任务。

## 验证状态

已完成自身改动的 `gofmt`、`git diff --check` 和生产主页面 flags 写入检索。所有生产 flags 写入只剩 `setMainRouteLocked`。

新增回归覆盖：跨原生页打开插件仍独占可见；路由重开拒绝旧回调；账号代理退出后端口不可连接且 cancel 一次；迟到代理拒收；临时弹层保留 lease；raise-only 深链接；重复启动通知；浏览器立即释放/按需恢复/旧 timer 不误关/mini-player；assistant 关闭保存草稿并提交导航。

Go 编译和执行由父任务集中完成，相关包为：

```powershell
go test ./cmd/whaleisle ./internal/ui/browserpreview ./internal/app/assistant
```

实际隐藏窗口验收 `native-a2-overlay-fix.json` 已通过：真实生产主程序和 Host 默认未观察到 WebView2，显式打开后真实 Client Runtime 连接完成，离开到市场、设置、计划页、点击原生返回以及 Host 换代均销毁旧窗口、关闭 Gateway，所属 WebView2 进程分别在 291、311、305、341、353 ms 内退出。整个运行的窗口保持隐藏。`native-a2-overlay-fix.json.native.png` 是原生 Content 捕获。

这份运行使用 `dsh-usage-panel`，记录的浏览器文本为“此插件尚无可独立打开的视图”，因此仅证明兼容运行时和释放生命周期，不证明独立插件面板内容可用。

独立内容最终证据为 `native-a2-panel-final.json`、同名 `.log` 和 `.json.panel.png`：既有 `qa-panel` 通过真实 Host overlay/Client Runtime 加载，Loader 为 active，动态侧栏图标和主面板各 1 个，面板为 1146×744 CSS px，内容精确为 `QA plugin panel view`。使用原有“预览版说明 → 继续”和“开始使用 → 关闭”操作后，没有残留 dialog，root.inert=false；已查看浏览器截图确认选中侧栏和无遮挡内容。返回后 Gateway、窗口及所属 WebView2 在 306 ms 内释放。构建/执行 wrapper 成功退出并回收临时 profile，未复制 runtime、未调用模型、全程隐藏。

此前带首次提示的成功挂载摘要从工具输出恢复为 `native-a2-panel-mount-pass.json`，明确标注恢复来源；对应图为 `.panel.png`。中间提示关闭不完整的失败保留在 `native-a2-panel-onboarding-failure.json/.log`，不与最终结果混用。可见 GPU 呈现及物理点击不在这些隐藏验收范围内。

# MyGo 的原生加载页与工作台

MyGo 的完整安装包 `WhaleIsle-MyGo-Setup-<version>.exe` 自带 Host 运行时。自动启动时，主窗口显示海面加载页，工作区就绪后切换到工作台；关闭自动启动时直接进入工作台。首次账户引导和托盘行为保留。

| 要做什么 | 从哪里打开 |
| --- | --- |
| 选择下次打开应用时是否自动启动 Host | 设置 → 通用 → 启动时 |
| 管理组件、导入数据、检查启动环境 | 设置 → 通用 |
| 检查、下载或安装更新 | 设置 → 关于 → 更新中心 |
| Host 启动失败后重试、查看诊断、导出日志 | 冷启动加载页，或已打开的工作台状态提示 |

此前的独立启动器命令和主程序内置启动器页面均已移除。旧的 `whaleisle://launcher` 链接改为打开桌面设置。历史 Windows 启动 smoke 由 `scripts/qa-gate-zero-webview.ps1 -Visible -RequireWorkspaceReady` 取代，使用合成 profile 分开观察原生界面、Host 和工作区就绪。

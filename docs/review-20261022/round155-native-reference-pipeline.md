# Round155 · 原生聊天界面参考管线（token + CDP 截图）

## 1. 为什么需要

MyGo 主窗是原生 UI，原生鲸屿是 harness Web UI + Electron 注入。要逐表面对比，必须有**原生 web UI 的真实像素**。本轮打通了离线管线（不启动 Electron、不修改安装版）。

## 2. 步骤

1. 起 harness web（暂存 runtime，临时 DSH_HOME）：
   ```
   set DSH_HOME=<temp>
   runtime\node\node.exe runtime\vendor\deepseek-harness\apps\cli\lib\bin.js web --host 127.0.0.1 --port 3097 --no-open
   ```
   stdout 会打印 `dsh web: http://127.0.0.1:3097/?token=…`；裸 `/` 是 401，必须带 token。
2. 用 `scripts/native-web-shot.mjs <token-url> <out.png> 1440 920 1.5` 截图：脚本启动 headless Chrome，通过 CDP（`Emulation.setDeviceMetricsOverride` + `Page.captureScreenshot`）输出 2160×1380 物理像素。直接 `chrome --screenshot` 会在这个应用上挂起，CDP 稳定。
3. 首次启动会出现 **预览版说明** 模态框（挡住 hero）。点「继续」一次后同一 DSH_HOME 会记住，后续截图直接得到主界面。
4. 基线：`evidence/screens/native-harness-chat-1440x920.png`（1440×920 DIP，150%）。

## 3. 对比口径

- MyGo 主窗截图用 `scripts/window-capture.ps1`（per-monitor-v2，物理像素；2160×1380 = 1440×920 DIP）。
- **属于桌面注入、浏览器里没有的差异不算 bug**：鲸鱼娘入口、branch pill、提交并推送、窗口按钮、桌面壁纸等。
- 已确认的真实差异示例：侧栏底部文案（原生 `设置与账号` vs MyGo `账号与登录`）、首次运行 `预览版说明` 模态框（MyGo 未实现）。

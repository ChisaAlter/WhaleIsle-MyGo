# 本地 PDF 交付预览

Files 的 PDF「WebView 打开」、会话 PDF 交付卡，以及当前 Office→PDF 转换结果，进入浏览器面板的独立只读 PDF 标签。保留显式「系统打开」和「复制文件路径」；WebView 不可用时显示错误与系统打开入口，不放行 `file://`。

## 文件与生命周期

- 交付卡仍先调用原有 `files.OpenPresentedPath`，保存当前草稿并遵循真实 Session cwd、远程/越界 Host fallback 与 dirty-close 保护。
- 本地 PDF 使用 `os.Root` 限定在点击会话 cwd 内读取，包括符号链接/junction 边界；只读取选中文件。PDF 最大 32 MiB，检查扩展名、PDF magic/version、EOF 和读取前后文件状态。完整 PDF 结构由原生 PDF 阅读器解析，HTML 不以 HTML MIME 提供。
- 读取结果成为不可变内存快照；后续 HTTP 请求不再读取原路径。外部打开前重新核对原文件身份、大小和修改时间。
- Office 转换明确绑定发起 Session ID；只接受仍有效的该次请求返回的精确 PDF 文件，不授权整个 temp 目录。过期请求不会打开新标签。
- 每个标签使用随机 loopback 端口和 256-bit 随机能力路径，仅 GET/HEAD、精确 Host/路径、Range 读取；无目录、任意路径参数、代理或 Host cookie 转发。地址不显示在工具栏，也不进入复制链接。
- 标签/原生窗口关闭、会话或 Host 身份失效时撤销能力；最长 30 分钟。应用退出关闭所有服务。重新查看需再次点击文件。

## WebView 安全边界

复用已有 Windows 临时隔离 WebView2 环境，通过 `LocalPDFURL` 只允许一个精确 PDF 资源。保持 Go bindings、脚本、弹窗、权限、下载和其余网络/文件资源禁用。普通浏览器与 HTML capture 的原有策略不变；没有新增通用 `file://` 或文件系统权限。

原生阅读器提供分页、缩放、查找等 PDF 阅读能力；这里不声称已完成 Windows 实机渲染、表单、字体或键盘验证。隔离环境或 PDF 阅读器失败时保留系统打开入口。

参考：原版 `ui-deliverables/src/client/PresentedFileCard.tsx` 的明确预览点击、`present-open.ts` 的独立原生打开动作；[WebView2 本地内容](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/working-with-local-content)与[资源请求控制](https://learn.microsoft.com/en-us/microsoft-edge/webview2/how-to/webresourcerequested)。

验证使用一个临时合成 PDF/HTTP fixture：范围读取、只读方法、随机路径、Host/Origin 边界、HTML/超限/越界/符号链接拒绝、源文件变更快照不变、租约失效与关闭、精确转换产物和 dirty 草稿保留。没有运行真实文档、Office 引擎、Host 或原生 WebView。

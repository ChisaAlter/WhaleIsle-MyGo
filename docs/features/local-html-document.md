# 本地 HTML 文档预览

## 原版契约与用户入口

原版 `vendor/deepseek-harness/packages/client/ui-files/src/client/workspacePreview.ts:2`
明确把 `.htm/.html/.pdf/.xhtml` 归入集成 Browser 文档；`apply.ts` 的
`openWorkspaceFile` 以所属 Session cwd 解析文件，然后调用 workspace opener。
本变更只补其中 HTML 三种扩展，不新增 Markdown 渲染、PDF 转换或 visual reply capture。

- 显式点击交付卡/正文已授权文件链接：仍先经过 `files.OpenPresentedPath` 的
  workspace 路径检查和已有 Files tab 生命周期，再打开 HTML 文档标签。
- Files 的 `.html/.htm/.xhtml` 工具栏增加现有「WebView2 打开」动作；源码查看、
  编辑、保存、脏草稿保护和系统打开继续保留。
- 预览磁盘上当前已保存内容，不隐式保存/提交未保存草稿。回到 Files 时草稿不丢。
- 文档标签显示文件名、系统打开与复制路径，不展示/复制内部能力 URL。
  系统打开是用户再次点击的动作，失败不会自动交给外部应用运行。
- 读取失败回到 Files 并显示可使用系统打开；WebView 创建/加载失败的标签保留系统打开。

## 内容与边界

`htmldocument.OpenWorkspace` 只接收当前会话实际 cwd 与明确选中的路径，用 `os.Root`
限制读取，包括 symlink/junction 逃逸；只读取一个 regular file 的 UTF-8 快照。
不使用 `file://`，不把目录或临时路径当作服务根，不从页面接收文件读取参数。
输入上限 8 MiB，sandbox 编码后上限 25 MiB。XHTML 按 HTML 兼容内容显示。

内容通过 `htmlpreview` 构造静态 `sandbox=""` opaque-origin iframe。支持静态 HTML、
内联 CSS、内嵌 data 图片与字体；native WebView、响应 CSP 和 iframe 三层均禁脚本。
不运行 JavaScript，也无法访问应用/Host、源码文件、外网或同目录依赖，不能保存文件。
此入口没有运行 capture 的 isolation Probe（其中 RTC 构造器暴露会判失败），因此
不能仅凭 WebResourceRequested/CSP 允许脚本。若以后开放脚本，必须先满足同等门禁。
不解析、自动授权或加载 sibling CSS/JS/images；依赖这些资源的页面需使用系统打开。
这不是通用网站托管，也不能宣称与原版任意 HTML 网页完全等价。

每个标签有单独的 `127.0.0.1:随机端口/256bit能力/document.html` 服务，仅提供该
快照的 GET/HEAD。精确校验 Host、路由、query、Origin，禁止路径遍历和目录读取；
响应 no-store/no-referrer/nosniff/X-Frame-Options: DENY，不记录能力 URL。
服务不接收或转发 Host cookie/账户凭据，不代理任何网络目标。

新增 `LocalHTMLURL` 与 `LocalPDFURL` 互斥，复用 Windows `IsolatedPreview` 临时
profile 和 fail-closed 策略：

- 禁 native bindings、预加载桥、WebMessage、Host objects、DevTools、默认脚本对话框。
- 顶层只允许精确文档 URL/fragment；子 frame 只允许该窗口首次 `about:srcdoc`。
- resource filter 只允许精确文档 GET/HEAD；其他请求仍拒绝。
- 权限、弹窗、下载、外部导航仍拒绝；PDF 的禁脚本策略不变，HTML 同样禁脚本。

关闭标签/应用、Host generation/endpoint/会话或 workspace 改变时撤销；30 分钟能力
过期后也失效。预览创建失败、加载失败、renderer 退出和原生关闭释放 lease。
系统打开重新校验 owner 与文件 identity/size/mtime，过期会话不能继续打开旧文件。

## 验证

一次临时合成 fixture 首次通过：`TestHTMLDocumentFixture`（0.412s；收紧为静态策略前）。
静态收紧后仅更新同一 fixture 的 shell 预期，按要求未重复运行；修正版交整合者编译。仅临时 HTML/
文本、loopback HTTP、headless UI；覆盖三种扩展、保存快照、dirty tab 保留、工具栏
编辑/预览/系统入口、编码与大小拒绝、越界/symlink、请求能力限制、失败 fallback、
owner 改变和标签关闭撤销。未执行 HTML、真实用户文档、Host、账户或原生 WebView。
fixture 不加入长期测试套件；累计 Windows 编译由唯一整合者执行。

未验证：Windows WebView2 实际呈现、静态预览、系统打开和原生隔离行为。

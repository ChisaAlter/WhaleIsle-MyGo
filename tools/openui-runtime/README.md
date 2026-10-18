# WhaleIsle-MyGo 独立 OpenUI runtime

这是可维护的自有 viewer 源码，使用官方 MIT 许可的 OpenUI 解析/React SDK，以及独立编写的原版公开 DSL 契约组件。默认根节点是 `Col(children, gap?, align?)`，不是标准库的表格 `Col`，也没有静默重写旧程序。

本模块供仓库 `internal/desktop/openui/runtime.go` 直接嵌入生产 HTML。runtime自身正常运行不依赖原版应用安装、用户提供的 viewer、CDN 或 npm；用户HTML显式引用的受限静态classic脚本由Host按需获取。原版 `open-intelligent-ui` 应用实现、bundle、样式、图片和生成提示词均未纳入；官方 SDK 的许可不被当作该应用的许可。

## 当前交付状态

- 自有 React registry 已登记 55 个公开契约组件，覆盖布局、内容、表单、图表、SVG、栅格地图、图片搜索、基本交互与独立 YouTube 媒体入口；自有函数保留 59 个公开应用函数的名称和位置参数。组件登记不等于所有原版行为均已实现或验收。
- 支持范围按行为逐项列在 [COMPATIBILITY.md](COMPATIBILITY.md)。Code复制、CopyToClipboard、Link、Markdown/Text内嵌HTTP(S)链接和OpenUrl仅在真实用户手势后请求owned Host，收到实际结果才提示成功；不是iframe直接复制或导航。Image及Markdown/Text图片经父层固定 `/api/image` 加载公共HTTP(S)图片，viewer只渲染返回的位图data URL，或带真实Host验证标记的严格静态SVG data URL；直接用户data SVG仍拒绝。格式、安全预算及未验收边界见 [PUBLIC-IMAGES.md](PUBLIC-IMAGES.md)。HtmlView在隔离文档内运行本地 HTML/CSS/inline JS，并支持由可信外层预取的原始静态classic script CDN引用；不是任意CDN库或远程页面兼容，精确范围见 [HTMLVIEW-SCRIPTS.md](HTMLVIEW-SCRIPTS.md)。独立编写的 `YouTube(video: string, title?: string)` 在用户点击 Load YouTube video 后，请求 Host 创建独立原生内嵌浏览器视图；具体媒体隔离和未验收边界见 [YOUTUBE-MEDIA.md](YOUTUBE-MEDIA.md)。未知组件仍产生错误。
- 使用真实 SDK `Renderer`、增量解析器、状态存储和动作执行器。SDK 未被修改；自有存储适配器仅包装其公开 Store API。
- 已生成生产资产。没有新增/运行项目测试、typecheck、浏览器 viewer、原生 helper 或原生构建，也没有访问真实图片/地图/媒体/CDN/账户/位置服务。**打包成功不是运行、视觉、视频播放或跨平台验收。**

## 生产构建与直接嵌入

本次生成环境：Node.js 24.19.0、npm 11.9.0。建议使用相同版本以复现字节；浏览器目标为 ES2022，构建脚本的 catalogue 工具目标为 Node 22。

在仓库根目录执行：

```sh
cd tools/openui-runtime
npm ci --ignore-scripts --no-audit --no-fund
npm run build
npm run embed
```

`npm ci` 使用提交的 lockfile 和 npm tarball integrity，不执行依赖 lifecycle 脚本。`npm run build` 只生成 production JS/CSS/HTML、实际 schema/guide 和许可证资料；它不启动 viewer，也不请求资源服务。`npm run embed` 检查产物哈希后，将以下文件安装到仓库 `internal/desktop/openui/viewer/`，最后替换 `viewer.html`：

- `viewer.html`：自包含 JS/CSS 和完整第三方许可文本，Go 默认嵌入此文件
- `THIRD-PARTY-NOTICES.txt`：同一许可文本的人类可读副本
- `provenance.json`：lockfile、产物 SHA-256、实际贡献字节的依赖及通知哈希
- `library-spec.json`、`library-guide.txt`：由本模块的实际 registry 生成；没有复制原版生成提示词

修改自有源码或锁文件后，重新运行 build/embed，并一起提交源码和这五个产物。Go 构建直接使用已提交的 HTML，不会自动运行 npm。源代码、锁文件、构建脚本和许可资料需要随交付保留；不要提交 `node_modules/`、临时 catalogue 可执行文件或研究目录。ESBuild 的中间 JS/CSS、metafile 和贡献包清单保留在本地 `dist/` 供来源核对。

## 桥接和资源

`src/bridge.ts` 验证父窗口、`whale-openui` 通道、version 1 和 `window.name` 中的 64 位十六进制实例 ID。程序更新保留同一个 viewer；父层负责实例所有权、原会话和原生确认。支持 ready/render/state/assistant/height/error/resolved，以及 requestId 对应的 api-request/api-response。

OpenUI viewer/helper 保持无直接网络、无 worker、无媒体和无浏览器剪贴板权限。HtmlView 使用独立的第二层 opaque srcdoc，脚本仅允许 inline 和预备好的 data URL；其内部不能再创建第三层可运行页面。父、子 CSP 均保留 `frame-src 'none'`，native 导航策略按真实 frame 身份和层级准入 srcdoc；不是任意 frame 放行。Image、Markdown/Text图片、SearchImage、Map及可信外层HtmlView脚本预取的固定 API 请求经 `requestResource(path, query, signal)` 发到父层；仅父层访问公共资源，子层只接受校验后的 data URL/路线点。公共图片沿用 `api-request`、`owned-N` requestId及 `api-response`，`/api/image` 的query为 `{src}`，成功body为 `{dataUrl, source, validatedSvg?: true}`；只有既有Host严格静态SVG validator通过且返回SVG data URL时附`validatedSvg: true`，位图省略该字段。source是去除fragment的实际最终重定向URL。HtmlView内层文档没有该资源桥、OpenUI 状态/动作桥、媒体能力或原生凭据，HtmlView内部图片也不因本次接线获得远程加载能力。宿主 OS 沙箱和 native 请求拦截仍是 helper 无网络保证的一部分，不能将任意浏览器里的 iframe/CSP 单独当作相同隔离。

HtmlView的可信外层使用纯解析器parse5 8.0.0（MIT）读取原始HTML和source locations，只将原始活动HTML命名空间classic script的外部src属性替换为Host返回的base64 JavaScript data URL。固定`/api/html-script`请求携带`{src, integrity?}`，成功body为`{dataUrl, source, integrityVerified}`；非全空白SRI必须由Host验证并返回`integrityVerified: true`。全部脚本准备完成后才创建iframe，保持原markup顺序与async/defer属性，但不保留原网络到达顺序。streaming不请求、不执行；HTML/程序修订/卸载取消前端等待并隔离旧回包，不代表已派发Host网络操作已停止。具体协议、安全预算、失败与兼容边界见 [HTMLVIEW-SCRIPTS.md](HTMLVIEW-SCRIPTS.md)。

YouTube 的联网播放器位于 Host 单独管理的原生内嵌媒体视图（media seat），不在上述无网络 helper 中创建 iframe，也不改用外部浏览器链接。`media` 能力默认 false，仅可信 Host 可对 owned runtime 启用并提供当前 `revision`。首个 upsert 必须源自真实 Load YouTube video 点击；固定 `media-<sequence>` 请求只传经过校验的视频 ID、标题及矩形/裁剪/viewport，不转发输入 URL 的 query 或 fragment，也不请求 autoplay。媒体视图采用全新临时 profile/cookie store，设置 `Untrusted` 和 `DisableBindings`，不继承 SessionOrigin、Cookie、Preload、native bindings 或用户 cookie；网站仍可在其临时 store 设置自己的 cookie，生命周期销毁时清理 profile。视图随 native Host 缩放和组件裁剪定位，移出可见区域或被 overlay 遮挡时隐藏。回执 `native-view-created`、`hidden`、`closed` 仅确认原生视图状态，不保证视频可用或正在播放。详见 [YOUTUBE-MEDIA.md](YOUTUBE-MEDIA.md)。

`hostActions`默认false，Host按实际内嵌viewer字节、非legacy路径及当前会话所有权启用。自有按钮同步验证真实click/键盘事件，把固定clipboard/open-url请求送入现有隔离shell；Host再按递增sequence、一次claim和当前owner校验，并在UI线程执行已有系统接口。结果复用原API回包，不新增通用native binding。HtmlView、timer、合成事件或自动render均不能取得动作许可。详见[NATIVE-ACTIONS.md](NATIVE-ACTIONS.md)。

Markdown/Text内嵌链接复用同一`HostLink`，仅接受无凭据、无控制字符的绝对HTTP(S)地址；不创建导航`href`、不在viewer内打开地址。链接的粗体、斜体、内联代码和图片等React子内容保持原样；动作身份由URL payload决定，不序列化React节点。缺少目标地址时显示普通内容，非法地址或非HTTP(S)地址明确显示禁用提示。图片使用同一`PublicImage`公共图片路径，以内联span/img保留alt和真实错误/来源；链接内图片不嵌套重试按钮，独立Image提供显式重试。原有Image的PNG/JPEG/GIF/WebP base64 data输入及alt/aspectRatio/fit/radius契约保持不变。此接线不改变Go桥消息形状；Markdown代码块复制维持原状，独立Code组件已有复制原文能力。

状态适配器一次性恢复初始快照；用户写入即使值相同也成为脏键，`@Reset` 恢复声明默认值并清除脏键。后续流式默认值不覆盖脏键。只回传脏键快照；不把每次 render 的宿主状态再次覆盖进当前交互。细节和未验收边界见兼容清单。

`resolved` 只表示完整程序通过解析及 SDK 错误检查并经过 React 提交调度。它不表示图片/地图等异步资源已成功、YouTube 可播放、所有组件变体都支持或像素验收通过。未知组件、未知函数、Query/Mutation 提供者、无效或未完成程序不会被替换成假成功页面。

## 许可与剩余工作

精确依赖版本/官方来源及浏览器贡献包的许可清单见 [THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md)，完整原文在生成通知中。当前不使用 `@openuidev/react-ui`，不打包 KaTeX 字体；Lucide 的完整 ISC 与 Feather MIT 通知一并保留。Code/Markdown代码块的本地高亮使用 lowlight 3.3.0（MIT）和 highlight.js 11.11.1（BSD-3-Clause），完整BSD通知随包。本目录不会为其他已有项目源码授予新许可。

剩余工作是解除验证限制后，执行已列状态/动作/流式/资源失败场景的协议回归与真实 Windows/macOS WebView 验收，并按兼容清单补齐明确缺失的能力。当前不能宣称与原版全功能/样式一致，也不能将源级实现或生产打包计作运行验收。

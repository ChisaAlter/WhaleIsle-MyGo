# WhaleIsle-MyGo 独立 OpenUI runtime

这是可维护的自有 viewer 源码，使用官方 MIT 许可的 OpenUI 解析/React SDK，以及独立编写的原版公开 DSL 契约组件。默认根节点是 `Col(children, gap?, align?)`，不是标准库的表格 `Col`，也没有静默重写旧程序。

本模块供仓库 `internal/desktop/openui/runtime.go` 直接嵌入生产 HTML。正常运行不依赖原版应用安装、用户提供的 viewer、CDN 或 npm。原版 `open-intelligent-ui` 应用实现、bundle、样式、图片和生成提示词均未纳入；官方 SDK 的许可不被当作该应用的许可。

## 当前交付状态

- 自有 React 组件覆盖布局、内容、表单、图表、SVG、栅格地图、图片搜索与基本交互；自有函数保留 59 个公开应用函数的名称和位置参数。
- 支持范围按行为逐项列在 [COMPATIBILITY.md](COMPATIBILITY.md)。Code复制、CopyToClipboard、Link、Markdown/Text内嵌HTTP(S)链接和OpenUrl仅在真实用户手势后请求owned Host，收到实际结果才提示成功；不是iframe直接复制或导航。HtmlView仅运行隔离的本地 HTML/CSS/inline JS，原版script CDN不可用；YouTube未注册，未知组件仍产生错误。
- 使用真实 SDK `Renderer`、增量解析器、状态存储和动作执行器。SDK 未被修改；自有存储适配器仅包装其公开 Store API。
- 已生成生产资产。没有新增/运行项目测试、typecheck、浏览器 viewer、原生 helper 或原生构建，也没有访问真实图片/地图/账户/位置服务。**打包成功不是运行、视觉或跨平台验收。**

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

viewer 保持无直接网络、无 worker、无媒体和无浏览器剪贴板权限。HtmlView 使用独立的第二层 opaque srcdoc，只有 inline script 执行权；其内部不能再创建第三层可运行页面。父、子 CSP 均保留 `frame-src 'none'`，native 导航策略按真实 frame 身份和层级准入 srcdoc；不是任意 frame 放行。SearchImage、Map 的固定 API 请求经 `requestResource(path, query, signal)` 发到父层；仅父层访问公共资源，子层只接受校验后的 data URL/路线点。HtmlView 没有该资源桥、OpenUI 状态/动作桥或原生凭据。宿主 OS 沙箱和 native 请求拦截仍是无网络保证的一部分，不能将任意浏览器里的 iframe/CSP 单独当作相同隔离。

`hostActions`默认false，Host按实际内嵌viewer字节、非legacy路径及当前会话所有权启用。自有按钮同步验证真实click/键盘事件，把固定clipboard/open-url请求送入现有隔离shell；Host再按递增sequence、一次claim和当前owner校验，并在UI线程执行已有系统接口。结果复用原API回包，不新增通用native binding。HtmlView、timer、合成事件或自动render均不能取得动作许可。详见[NATIVE-ACTIONS.md](NATIVE-ACTIONS.md)。

Markdown/Text内嵌链接复用同一`HostLink`，仅接受无凭据、无控制字符的绝对HTTP(S)地址；不创建导航`href`、不在viewer内打开地址，也不加载远程图片。链接的粗体、斜体、内联代码等React子内容保持原样；动作身份由URL payload决定，不序列化React节点。缺少目标地址时显示普通内容，非法地址或非HTTP(S)地址明确显示禁用提示。此接线不改变Go桥协议；Markdown代码块复制维持原状，独立Code组件已有复制原文能力。

状态适配器一次性恢复初始快照；用户写入即使值相同也成为脏键，`@Reset` 恢复声明默认值并清除脏键。后续流式默认值不覆盖脏键。只回传脏键快照；不把每次 render 的宿主状态再次覆盖进当前交互。细节和未验收边界见兼容清单。

`resolved` 只表示完整程序通过解析及 SDK 错误检查并经过 React 提交调度。它不表示图片/地图等异步资源已成功、所有组件变体都支持或像素验收通过。未知组件、未知函数、Query/Mutation 提供者、无效或未完成程序不会被替换成假成功页面。

## 许可与剩余工作

精确依赖版本/官方来源及 81 个浏览器贡献包的许可清单见 [THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md)，完整原文在生成通知中。当前不使用 `@openuidev/react-ui`，不打包 KaTeX 字体；Lucide 的完整 ISC 与 Feather MIT 通知一并保留。Code/Markdown代码块的本地高亮使用 lowlight 3.3.0（MIT）和 highlight.js 11.11.1（BSD-3-Clause），完整BSD通知随包。本目录不会为其他已有项目源码授予新许可。

剩余工作是解除验证限制后，执行已列状态/动作/流式/资源失败场景的协议回归与真实 Windows/macOS WebView 验收，并按兼容清单补齐明确缺失的能力。当前不能宣称与原版全功能/样式一致，也不能将源级实现或生产打包计作运行验收。

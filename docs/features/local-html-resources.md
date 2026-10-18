# HTML 静态资源快照

## 原版来源与闭环

原版 `src/main/preview-workspace.js:19–36,145–224,227–256` 以授权 cwd/token 路径
实际返回 CSS、图片和字体；`preview-workspace.test.js:242–266` 有 HTML 引用 CSS/PNG
的静态服务断言。Files→Browser 通过 ui-surfaces 的 `previewWorkspaceFile` 进入此服务。
旧 sidebar 的 `html/pack.ts` 仅处理直接 CSS/JS，是另一条有限路径。

MyGo 现在在用户打开 HTML 时，把允许的静态依赖读成同一份 immutable 快照，
再交给既有静态 sandbox。没有复制原版目录服务器，没有新增文件请求 API/URL路由；
浏览器依旧只能 GET/HEAD 一个精确 `document.html` 能力。不会执行脚本或启动模型。

## 支持范围

- 普通 `<link rel="stylesheet">` 转成内联 style，保留 media；style 元素和 style 属性
  中的真实 `url()` 通过 CSS lexer 改写，字符串/注释中的伪 URL 不读文件。
- CSS `@import` 可有限递归展开，保留普通媒体条件；相对路径按声明该 URL 的 CSS
  文件目录解析。循环依赖、过深 import、layer/supports 等复杂条件跳过并提示。
- img/src、常用 img/picture srcset、SVG image href、video poster、background 属性。
- PNG/JPEG/GIF/WebP/SVG 图片与 WOFF/WOFF2/TTF/OTF 字体；资源扩展和内容检查匹配。
  图片检查尺寸，字体只检查类型签名，不声称已执行完整字体校验或实机渲染。
- CSS 与图片/字体均内联；使用 data URL 的二进制资源不再需要请求文件系统。
- Browser 状态栏显示载入数量/快照大小和跳过原因，完整文字可悬停查看；系统打开保留。

明确限制：脚本/事件处理器移除；外部网络、站点根路径、file URL、未知类型、媒体/
嵌套文档不载入。有 base 元素时不猜测相对路径，跳过相对资源并提示。CSS image-set
里的裸资源字串不展开，含该形式的函数被替换并提示；SVG 内部的嵌套资源不再递归打包，
含 image/style 时明确提示限制。现成 data 图片/字体可保留，外部页面不作为资源抓取。

## 授权与限额

仅以当前文档的既有会话 cwd 为根。每个资源 URL 单次解码后相对解析，拒绝绝对路径、
协议、NUL、越界、.git（包括 canonical/symlink 指入 .git）。`os.Root` 约束实际读文件，
再校验 canonical 归属和文件 identity/size/mtime，资源读取前后均检查当前 owner。
按 canonical 文件去重；浏览器不能追加读取 sibling 文件，磁盘改变也不改变已发布快照。

- 主 HTML 仍上限 8 MiB
- 单资源 4 MiB，主文档+唯一资源共 16 MiB
- 最多 64 个资源文件、512 次文件引用、CSS import 深度 4
- HTML/CSS 各最多 200k token；最终 sandbox 编码仍上限 25 MiB
- 图片单边 ≤20,000，像素总数 ≤40,000,000；SVG XML token/深度有界

超限/缺失/不支持依赖显示限制，不扩大根授权。文档关闭、会话/Host 改变和能力到期
继续撤销全部快照。native IsScriptEnabled=false、CSP script-src none、sandbox 无
allow-scripts 三层不变，未绕过 capture 的 RTC fail-closed 门禁。

## 依赖与验证

HTML tokenizer 使用 `golang.org/x/net/html v0.47.0`（BSD-3-Clause）；CSS lexer 使用
`github.com/tdewolff/parse/v2/css v2.7.21`（MIT）。均从官方 Go module registry 固定版本
下载，go.sum 校验；没有下载/运行用户资源，不升级既有依赖版本。

一个临时合成 fixture，最终通过 `TestStaticResourceGraphFixture`（0.007s）。首次断言
误把 CSS 字串里的 `<script` 当成实际元素；改为 HTML tokenizer 检查实际节点后，同一
fixture 通过，未改业务逻辑来迎合该断言。另一次命令在错误 cwd 未执行任何测试。
验证相对 CSS/媒体 import/font/image/srcset、循环/编码/类型/路径/大小/数量边界、
immutable 快照、owner 撤销和三层静态策略。无真实用户文件、Host、账户、网络资源、
浏览器/HTML/字体执行。fixture 不进长期测试套件；Windows 累计编译由唯一整合者执行。

未验证实际 Windows 图片/字体/CSS 呈现，不能将合成测试当实机隔离或视觉验收。

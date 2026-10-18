# HtmlView 静态classic脚本：来源、准备与隔离

本模块独立实现 `HtmlView(html: string, height?: number, title?: string)` 的受限外部脚本支持。它仅准备原始HTML中已存在的活动classic script引用，不是通用CDN代理或任意JavaScript库兼容层。原版应用源码、bundle与生成提示词未复制；纯HTML解析依赖为parse5 8.0.0（MIT），其entities 6.0.1依赖为BSD-2-Clause；版本由lockfile固定，依赖许可与来源由生产构建收集，见[THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md)。

这里描述源码与协议范围；没有下载或执行真实CDN脚本，没有运行测试、typecheck、浏览器、原生helper或原生构建。生产打包不证明脚本行为、WebView兼容或安全验收通过。

## 允许准备的输入

- 仅原始输入中活动HTML命名空间的classic `<script src="…">`。src须为无用户名/密码的绝对公共HTTP(S) URL；相对地址、协议相对地址和其他scheme不进入此路径。
- parse5作为纯解析器读取HTML及source locations，不在viewer DOM里插入输入，不靠执行HTML发现依赖；只定点替换已识别外部script的src属性，并移除已由Host处理的integrity/crossorigin属性，不重新序列化整个文档。其余markup、inline脚本内容、节点次序和async/defer属性保留。
- `template`中的script不是准备对象，不发Host请求。运行后的DOM创建、`document.write`或动态加载也不会触发预取或取得资源接口。外部foreign namespace script明确拒绝；原有inline本地能力不因此扩展。
- `crossorigin="use-credentials"`明确拒绝。此路径不提供登录、账户脚本、用户Cookie或凭据透传能力。
- 静态`type="module"`在准备阶段明确报错；不解析或获取module/import/import()及动态依赖。fetch、动态远程script、额外远程CSS/字体/图片及其他依赖不支持。不能因为顶层classic文件可以获取，就宣称它的整套依赖可用。
- `script-src data:`不是对所有import语法或脚本自带data module的通用禁令；动态外部依赖仍没有资源桥且受网络策略阻止，不因此获得网络或native权限。

## 固定Host协议

请求由可信外层HtmlView组件在创建内层iframe之前发起，复用现有 `requestResource(path, query, signal)`、`api-request`、`owned-N` requestId及`api-response`。没有增加内层可调用的桥或通用native binding。

- path：`/api/html-script`
- query：`{src: string, integrity?: string}`；SRI只使用原script提供的integrity
- 成功body：`{dataUrl: string, source: string, integrityVerified: boolean}`
- dataUrl：固定 `data:text/javascript;base64,` 前缀，内容为Host接受的原始JavaScript字节
- source：Host实际最终规范HTTP(S)来源URL，去除fragment；不能用输入URL假装重定向后的来源
- 失败：通过既有资源错误回包报告；外层显示准备失败，不替换为假脚本或空成功页面

Host负责公共URL、DNS/重定向、响应格式、字节预算与SRI校验。内层只得到完成准备后的HTML与data URL，不得到Host请求对象、原会话、资源接口或凭据。来源URL用于识别实际资源，不代表作者身份、版权许可或再分发授权。

## 格式、预算与SRI

Host接受的响应MIME仅为：

- `text/javascript`
- `application/javascript`
- `text/ecmascript`
- `application/ecmascript`
- `application/x-javascript`
- `text/x-javascript`

charset须缺省或为UTF-8/utf8，内容须是有效UTF-8；此路径不解压、不转码。正确MIME、有效UTF-8或SRI匹配都不证明脚本没有副作用或能在隔离环境中正常运行。

- 单脚本原始字节上限16MiB；固定API回包仍受25MiB上限约束。不新增脚本数量或准备后文档的合计字节上限；逐脚本预算不等于整个文档的内存上限。
- 单次获取总预算8秒，最多4次重定向；URL公共性检查不能只做首跳。
- 原始HtmlView内容仍受256KiB程序预算约束，完整事件仍受64KiB预算约束；不通过截断HTML或脚本满足预算。
- 既有资源传输同时最多4个活跃请求，超出进入队列；15秒前端活跃等待超时保持不变。前端等待与Host获取是不同边界，不用其中一个成功推断另一个已完成。

未提供SRI或integrity全为空白时，Host返回`integrityVerified: false`，不声称已校验内容身份。提供非全空白integrity时，Host支持sha256、sha384、sha512，并要求最强有效算法的候选摘要至少一个匹配才返回true；非空内容没有有效支持hash时返回400，最强hash不匹配时返回412；校验失败不能执行该文档。外层还要求该响应标记严格为true，不把输入HTML有integrity属性或仅获取成功当成SRI完成。保留原始字节用于校验，不通过解压/转码后重新解释来掩盖不匹配。

## 准备完成后再创建文档

1. 只有非streaming、HtmlView能力启用且输入有效时才解析和准备脚本。
2. 全部所需静态脚本成功返回、通过响应检查和SRI条件后，改写对应src属性并移除integrity/crossorigin，避免对opaque data URL再次触发浏览器CORS/SRI处理；Host已在原始字节上完成所请求的校验。
3. 由可信外层为完整结果添加自有CSP与观测前缀，再创建第二层opaque srcdoc iframe。
4. 任一步准备失败时显示真实原因，不创建iframe；任何用户HTML或inline脚本都尚未执行。

浏览器在准备后的文档中按原markup顺序以及保留的async/defer属性执行。因为先等齐资源再创建文档，原始CDN网络到达时序不再存在；尤其不能保证async脚本保持原网页的运行顺序。无async/defer的classic脚本和inline脚本仍保留原有位置，不搬到文档开头或末尾。

执行环境中的`document.currentScript.src`是data URL。依赖该地址计算相对资源路径、运行时下载其他文件、module/import或需要`unsafe-eval`的库不在支持范围。Host已返回资源只表示准备成功；脚本语法/运行错误及CSP拒绝仍如实提示，不能以ready、resolved或预取成功宣称页面功能可用。

## 生命周期与权限不变

- streaming期间不发脚本请求、不创建iframe，并销毁旧活动文档。HTML改变、可信程序修订改变或组件卸载会取消前端准备/等待、清理旧文档并忽略过期回包。
- 只有height/title或普通SDK状态变化时保留同一HTML文档，不重新执行脚本；局部JS状态不回传、不跨文档持久化。
- AbortSignal只取消前端等待/排队任务并阻止旧结果生效，不是已派发Host网络操作停止的native确认。
- 内层sandbox仍只有`allow-scripts`。内层`script-src`在原有inline策略上仅增加`data:`，不允许直接HTTP(S)脚本或`unsafe-eval`；其余网络、worker、frame、media、form、导航和native能力不扩展。
- 外层CSP、既有native frame身份/层级准入、请求拦截及OS无网络边界不变。内层postMessage只可被当前组件作为非可信展示状态处理，不转发资源/状态/动作/assistant消息到Host。
- 任意获准执行的脚本仍可能消耗CPU或内存；没有新增脚本执行时间上限、CPU配额或内存配额；准备后文档累积内存同样没有新的总量保证。

完整iframe与native边界见[HTMLVIEW-ISOLATION.md](HTMLVIEW-ISOLATION.md)。这一脚本例外不为HtmlView图片、媒体或YouTube授予网络能力；独立YouTube仍使用其自己的Host媒体视图。

## 待允许验证后验收

需验证解析位置与特殊HTML上下文、template/foreign namespace/动态节点拒绝、URL/DNS/各重定向及预算拒绝、MIME/charset/UTF-8/SRI失败、全部完成前不执行、async/defer/inline顺序、HTML/修订/streaming/卸载取消与旧回包隔离，以及Windows/macOS WebView的data脚本、CSP、错误与高度行为。还需逐库检查是否依赖相对资源、动态加载、module或eval；不能从单文件预取推导任意CDN库兼容。

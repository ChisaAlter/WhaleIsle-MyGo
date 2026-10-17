# 源级兼容清单（2026-10-17）

此表描述独立实现代码的实际行为，不是测试通过清单。当前 registry 登记 55 个公开契约组件，完整位置参数和枚举以同一 registry 生成的 `library-spec.json` 为准。UI 样式、具体默认间距和浏览器原生控件外观为自有设计，不宣称原版像素一致。Node 子槽经 SDK `renderNode` 递归渲染；没有将未知组件转换成普通文本或同名但语义不同的标准库组件。

## 真实 SDK 对应关系

| 需求 | 使用的公开 API | 本模块处理 |
|---|---|---|
| 解析/表达式 | lang-core 1.0.0，createStreamingParser | 相同公开位置调用语法、绑定和内置表达式；不执行任意 JS |
| 组件库 | react-lang 1.0.0，createLibrary/defineComponent | 自有 `whaleisle-independent-col-v1`，root=Col；不导入 react-ui 的表格 Col |
| 渲染/流式 | Renderer(response, library, isStreaming) | 稳定挂载，完成前不发送 resolved；流式期间禁止表单/动作 |
| 绑定 | reactive/useStateField/FormNameContext | 表单和 `$binding` 使用 SDK 状态，不另造事件格式 |
| 状态持久化 | lang-core createStore 的 get/set/initialize/subscribe/dispose | 自有 dirty-key 适配器，官方 SDK 文件未修改 |
| 动作 | useTriggerAction/onAction | 本地 Set/Reset，assistant 交回宿主；不提供任意原生动作 |
| 错误 | onError/onParseResult，ParseResult.meta | 缺失组件/函数、无效参数、未解析引用、未完成程序明确错误 |
| 主题 | 宿主 light/dark | 自有 CSS 和地图 light/dark；未实现宿主任意 CSS 变量映射 |

SDK 官方来源为 thesysdev/openui 的 `f7109dd01b7bdfb14c8c9006eaca5a8b3cfc3a20`；MIT 原文随实际 npm 包保留。公开 SDK 导出和安装包声明已核对。标准 react-ui 不是兼容层：它的 Stack 根、表格 Col 以及标准 Text/Button 参数不能替代原版公开契约。

## 组件逐项范围

| 名称 | 已实现的参数/行为 | 仍有的区别或限制 |
|---|---|---|
| Col | children、gap、start/center/end/stretch；纵向 flex | 默认间距自有；gap 为4px单位 |
| Row | children/gap/align/justify/wrap；横向 flex | 同上，不重写成标准 Stack |
| Box | row/col、gap/padding、background/radius/border、width/height/flex、align/justify/wrap | 仅安全颜色及简单 CSS 长度；无任意 CSS 表达式/URL |
| Grid | 固定 columns 或 minChildWidth 响应式、gap | 1–24列；极端尺寸被约束 |
| Card | outline/soft、padding/gap | 自有面板外观 |
| Carousel | itemWidth/gap、水平滚动和 snap、键盘聚焦 | 无自动轮播；浏览器滚动表现待验收 |
| Divider | 水平分隔 | 自有颜色 |
| Spacer | flex 填空 | 用于 flex 父容器 |
| Title | sm/md/lg/xl/2xl/3xl、color | 自有字号/heading映射 |
| Text | 内联 Markdown/混合Node、xs/sm/md/lg、color/weight/italic/lineThrough/tabularNums/maxLines；有效HTTP(S)内嵌链接经真实手势交Host | 无目标为普通内容，非法/非HTTP(S)链接显式禁用；保留链接React子内容；图片不加载；不支持内联任意 HTML |
| Caption | text/color | 纯文本副说明 |
| Label | text | 可见标签；字段实际关联由各控件实现 |
| Badge | neutral/info/success/warning/danger/purple/pink、Lucide icon | 未知 icon 显式提示 |
| Icon | name、sm/md/lg/xl、color；实际本地 Lucide 图标 | 仅安装版本存在的名称 |
| Markdown | CommonMark/GFM、表格、任务列表、代码块；有效HTTP(S)内嵌链接经真实手势交Host | raw HTML 跳过；图片显式不可用；无目标为普通内容，非法/非HTTP(S)链接显式禁用；带语言的代码块使用下述本地高亮，代码块复制未扩展；公开Markdown契约未声明美元数学语法，需要公式时用独立Math组件 |
| Quote | text/cite | cite 为纯文本，不导航 |
| Code | code/language、横向滚动，本地高亮与真实点击复制原文 | 未指定/未知语言和超高亮预算仍保留原文；复制受Host能力、事件预算与完整文本校验约束，收到真实成功才提示Copied |
| Math | tex/inline；本地 KaTeX 到原生 MathML | trust=false；不打包字体；原生MathML效果待各WebView验收；有解析/展开预算 |
| Link | text/url；真实点击后交Host打开绝对HTTP(S)地址；Markdown/Text内嵌链接复用同一HostLink | 无userinfo凭据或控制字符；不自动导航、不在iframe内打开；Sent to browser只表示系统接受，不保证页面加载 |
| List | bullet/number/none、timeline；字符串或ListItem | 自有标记/连接线 |
| ListItem | title/description/marker/children | title/description 内联 Markdown仍受上述限制 |
| Button | solid/outline/soft/ghost，primary/danger/success，sm/md/lg，icon/disabled/block | 动作边界见下表；无动作发送标签给assistant |
| IconButton | outline/ghost/solid、icon/label/action | 同上；未知图标提示 |
| Input | text/number，label/placeholder，双向value | 数值编辑存number，清空存空字符串；原生input外观 |
| TextArea | value/label/placeholder/rows | 原生textarea；rows限制1–50 |
| Select | 字符串或value/label选项、label/placeholder | 不自动改写未知当前值，显示未知选项 |
| Checkbox | checked/label/lineThrough | true时显示删除线（如要求） |
| Slider | value/min/max/step/label/unit/marks | 范围或绑定无效时显式提示并禁用，不将越界值悄悄写回 |
| SegmentedControl | value/options/block | 选择状态写真实绑定；可换行 |
| RadioGroup | value/options/label；list/pills/cards及description | 原生单选语义，自有外观 |
| DatePicker | YYYY-MM-DD绑定、label/min/max | 使用浏览器原生日期控件；没有额外自有日历 |
| Form | children，稳定表单作用域 | assistant附当前状态；无浏览器表单提交或业务服务器 |
| Pressable | children/action/selected/padding/background/radius | 可点击容器；动作使用与Button相同的整计划检查；嵌套控件交互不应触发外层动作 |
| Timer | seconds绑定、label、倒计时/开始暂停/重置 | 浏览器计时，非实时系统；按墙上时钟补偿挂起，系统时钟调整会影响倒计时；宿主关闭时不提供后台提醒 |
| Popover | 字符串或Node trigger/content、点击/键盘、portal定位、Escape/外部焦点关闭 | 无远程内容；嵌套/焦点行为未运行验收 |
| Flashcard | front/back/hint；翻面、键盘、前后面可访问性 | reduced-motion采用静态翻面 |
| Celebration | when从false→true触发本地confetti | reduced-motion使用静态反馈；无音频 |
| Image | src/alt/aspectRatio/radius/fit；PNG/JPEG/GIF/WebP base64 | 外部URL明确不可用；无SVG data URL、加载失败无替代图片 |
| SearchImage | query/aspectRatio/radius/fit；真实父层搜索、重试/切换、来源文本 | q须2–200字符；API最多4图；来源未提供作者/许可时明确显示缺失，不声称所有搜索图片可再分发 |
| Chart | bar/line/area/pie/donut/scatter，x/series，number/currency/percent，height | 下方图表细项，不含原版未定义的Sparkline |
| Table | columns、rows；真实表格、Node单元格 | 无排序/分页API；超额显式错误 |
| Map | markers/route/selected/height；自动fit、拖动、按键/按钮、Ctrl/⌘+wheel、双指缩放/平移，label绑定选择、真实road route | 自有栅格，不是MapLibre；普通wheel滚动页面；无旋转/倾斜，仅WebMercator纬度范围 |
| MapMarker | label/lat/lng/category/icon | icon按公开emoji语义展示；Map外显式提示；不伪造点坐标 |
| Svg | children/viewBox/width/height | 只允许下面的绘图节点；无脚本、URL paint、href、foreignObject或任意样式 |
| Circle | cx/cy/r/fill/stroke/strokeWidth/opacity | 校验有限几何和安全paint |
| Ellipse | cx/cy/rx/ry/fill/stroke/strokeWidth/opacity | 同上 |
| Rect | x/y/width/height/fill/stroke/strokeWidth/opacity/rx | rx保持最后位置参数 |
| Line | x1/y1/x2/y2/stroke/strokeWidth/opacity | 原生SVG线 |
| Path | d/fill/stroke/strokeWidth/opacity | 标准路径指令及数值校验；非法路径显式提示 |
| Polyline | points/stroke/strokeWidth/fill/opacity | 保持stroke优先位置顺序 |
| Polygon | points/fill/stroke/strokeWidth/opacity | 原生SVG多边形 |
| G | children/transform/opacity | 标准安全transform；不是Group别名 |
| SvgText | text/x/y/size/fill/anchor/weight | 文本转义，无HTML |
| HtmlView | html/height/title；独立opaque srcdoc内的本地HTML/CSS/inline JS、DOM事件、canvas、动画循环，自动/指定高度 | 原公开script CDN未支持；无媒体/远程资源/原生动作/持久状态，详细隔离与生命周期见下节；native运行未验收 |
| YouTube | video/title；解析单一视频 ID，真实 Load YouTube video 点击后创建独立 Host 原生内嵌媒体视图；按 rect/clip/viewport 定位与隐藏 | media 能力默认关闭；临时独立 profile，无 helper iframe/外部浏览器替代、用户 cookie、native bindings、autoplay 或 URL 参数透传；原生视图回执不保证视频可用/播放，见下节 |

### Code / Markdown 本地语法高亮

使用官方 lowlight 3.3.0（MIT）和 highlight.js 11.11.1（BSD-3-Clause）。固定语言名/别名映射，不调用自动检测、不下载语法。语法输出仅转为React text/span/className，逐字检查文本与输入一致；不使用innerHTML，不执行HTML、JS或其他代码。语言别名是语法着色映射，不是编译器/方言正确性验证。

| 语法 | 接受的别名 |
|---|---|
| arduino | ino |
| bash | sh, zsh |
| c | h |
| cpp | cc, c++, h++, hpp, hh, hxx, cxx |
| csharp | cs, c# |
| css | 无 |
| diff | patch |
| go | golang |
| graphql | gql |
| ini | toml |
| java | jsp |
| javascript | js, jsx, mjs, cjs |
| json | jsonc |
| kotlin | kt, kts |
| less | 无 |
| lua | pluto |
| makefile | mk, mak, make |
| markdown | md, mkdown, mkd |
| objectivec | mm, objc, obj-c, obj-c++, objective-c++ |
| perl | pl, pm |
| php | 无 |
| php-template | 无 |
| plaintext | text, txt（明确纯文本） |
| python | py, gyp, ipython |
| python-repl | pycon |
| r | 无 |
| ruby | rb, gemspec, podspec, thor, irb |
| rust | rs |
| scss | 无 |
| shell | console, shellsession |
| sql | 无 |
| swift | 无 |
| typescript | ts, tsx, mts, cts |
| vbnet | vb |
| wasm | 无 |
| xml | html, xhtml, rss, atom, xjb, xsd, xsl, plist, wsf, svg |
| yaml | yml |

语言名去首尾空白、大小写不敏感；未知名称不猜测。不带语言标记的代码显示纯文本并说明。Markdown内联code保持纯文本样式；fenced/indented code按Markdown解析结果保留空白和末尾换行。高亮前预算：20,000个UTF-16 code units、1,000行、单行4,000 code units；高亮后最多12,000个AST节点/48层。超过预算、异常或输出文本不一致时显示全部原文及原因，不截断代码。预算用于降低主线程成本，不承诺硬性时间上限；iframe仍无worker。实际高亮、流式更新与WebView显示未运行验收。

### Markdown/Text 内嵌链接

- 内嵌链接和独立Link组件使用同一`HostLink`。仅无userinfo凭据、无控制字符且hostname非空的绝对HTTP(S)地址可形成Host动作；仍须满足`hostActions`能力、非streaming状态、真实click/键盘事件和既有payload/事件预算检查。
- 粗体、斜体、内联代码等React子内容保留原样，不转换成纯字符串。动作身份由URL payload决定；React节点不序列化为动作身份或Host消息。
- 缺少目标地址时显示普通内容；非法地址、相对地址或非HTTP(S)地址明确显示禁用提示。链接不添加导航`href`，不直接打开窗口或导航iframe；Host回执成功只表示系统接受请求。
- 远程图片仍不加载，raw HTML规则不变。Markdown代码块复制维持原状；独立Code组件继续通过既有Host能力复制原文。此接线未修改Go桥协议，也不代表实际WebView交互已验收。

### 图表细项

- bar: grouped或正负分离stacked；line/area: 普通或累计stacked，缺失值为gap；stacked行缺任一值时整行gap，不能默认为零。
- pie/donut: 每个series分别画有标签的真实图；零总量显示空状态；负值或缺失值显式报错。scatter要求数值x。
- stacked pie/donut/scatter 的组合语义未由公开契约定义，本实现将其作为参数边界显式拒绝，不将它们计作已证明的原版能力缺口，也不静默忽略该参数。每个图都可展开原始数据；SVG title与原始表提供可访问内容。
- 自有数值格式化与图例/轴样式。六种图表的真实数据标记均有hover/focus驱动的图下数据提示，包含系列、行、格式化值和精确原值；stacked额外显示正/负累计范围，pie/donut显示实际占比。方向键按图中标记顺序移动，Home/End跳首尾，Escape关闭详情。数据变更时按当前行/系列重新读取数值，不保留旧tooltip文本。没有额外图表动画或拖动缩放；未声明这些为已实现。
- 分配预算：Chart最多5,000行/32系列，Table最多50,000格；超限显示错误。预算是本实现边界，不是原版契约保证。

### 地图/图片桥接口

`requestResource<T>(path, query: Record<string,string>, signal?)` 返回已验证父窗口的响应体。

- `/api/map-tile`: z/x/y/style字符串；z=0..19、style=light/dark。回包 `{dataUrl, attribution, source, z,x,y,style}`。只用固定ArcGIS街道/深色灰底源，显示实际返回的来源署名；外部署名链接导航未实现。
- `/api/street-route`: coords为分号分隔lng,lat；回包 `{points:[[lat,lng],...]}`。父层只接受2–25个地点。失败保留错误，不以直线冒充道路。
- `/api/image-search`: q字符串；回包 images与imageSources等长数组。仅接受实际bitmap data URL，保留来源；不替换失败图片。
- `/api/wiki` 仅在桥允许列表中；原公开55组件没有Wiki组件，因此本包没有虚构同名组件或自动请求它。
- 请求取消和requestId去重；同时4个活跃请求以配合宿主队列，排队数量无额外产品上限。15秒活跃超时；32MiB地图缓存会淘汰并按需重载，不截断marker数量。

### HtmlView 内容和生命周期

- 原公开位置契约是 `HtmlView(html: string, height?: number, title?: string)`；用途包含需要循环的游戏、canvas和动画。这里运行真实inline JavaScript，不把HTML改成静态文字，也不把script CDN假装成本地功能。
- `html` 可以含页面/片段、style和inline script；不在React主文档插入HTML。opaque iframe仅有 `sandbox="allow-scripts"`，没有same-origin、form提交、弹窗、下载、顶层导航、权限或原生对象。其内部DOM事件、输入编辑、canvas绘制和动画循环属于本地页面行为；不暴露 `$binding`、assistant或资源接口。
- 未指定height时，组件只接收当前iframe真实contentWindow、origin=null的私有高度消息；消息只能调整本组件高度/状态，不能进入OpenUI宿主协议。自动高度按body布局/溢出测量，超出现有2,000px展示上限时保留内部滚动并提示；指定height允许1–2,000px，超额明确错误。依赖viewport高度的CSS布局宜指定height；不承诺任意CSS布局都能获得唯一自适应高度。
- `html`改变即销毁旧iframe并创建新opaque文档；旧窗口消息不再匹配。仅height/title或无关OpenUI状态变化不重启同一HTML。streaming期间不运行部分脚本并销毁旧活动页面，结束后再创建完整文档；组件移除销毁子页面与监听器。局部JS状态不回传、不跨文档保存。
- 宿主能力位`htmlView`默认false，只能由可信helper在真实frame准入就绪后启用。false时本组件明确显示不可用，不创建iframe；不影响其他OpenUI组件。Windows缺少所需第二层frame API、未接线平台或旧helper均不得冒充支持；能力更新沿用render消息，不重置用户状态。
- 自有CSP前缀先于输入HTML生效，后续meta只能叠加限制。无远程script/CDN、fetch/WebSocket、worker、frame URL加载、media、object、base或form action。保留外层CSP和固定native请求拦截，不放宽它们来加载图片/字体。data图像/字体受所有继承策略交集约束，未宣称跨引擎可用。`webrtc 'block'`仅作兼容引擎的额外限制；现有OS沙箱无网络能力才是独立网络边界。
- 仅OpenUI模式在native层准入唯一viewer的直接第二层srcdoc；普通HTML预览策略不变，第三层/其他URL继续拒绝。HtmlView自身不能通过伪造postMessage得到父组件身份。具体native接线见[HTMLVIEW-ISOLATION.md](HTMLVIEW-ISOLATION.md)。生产打包不会运行输入HTML或编译helper。
- 任意inline JS仍能耗尽CPU/内存或卡住helper；此隔离不是脚本时间预算或资源配额。CDN和HtmlView内的外网媒体仍未支持；独立YouTube组件使用下一节的Host媒体视图，不向HtmlView授予该能力，不以组件注册记为完整原版能力齐平。

### YouTube 独立原生媒体视图

- 独立编写的公开位置契约为 `YouTube(video: string, title?: string)`。`video` 接受精确11位 `[A-Za-z0-9_-]` ID，或受限的绝对HTTP(S) YouTube watch/embed/shorts/live、youtu.be短链；youtube-nocookie.com仅接受embed。输入最多2,048个UTF-16 code units，拒绝空白、控制字符、反斜杠、凭据、非默认端口和未列主机；watch须恰有一个有效v参数。仅提取ID，query/fragment中的时间、列表、autoplay等选项均不转发；不验证视频是否存在。精确主机/路径和拒绝边界见[YOUTUBE-MEDIA.md](YOUTUBE-MEDIA.md)。
- `media`默认false，只能由可信Host按owned runtime和当前所有权启用，render中的当前`revision`也由Host提供。首次真实Load YouTube video按钮激活才发送`userGesture:true`的upsert；自动render、timer、合成事件和HtmlView不能取得创建能力。播放器是Host侧单独管理的原生内嵌浏览器seat，保持原OpenUI helper/CSP/OS无网络边界，不是helper iframe或外部浏览器链接。
- 每个媒体seat使用全新临时profile/cookie store、`Untrusted`和`DisableBindings`；不继承SessionOrigin、Cookie、Preload、native bindings，不复制用户浏览器cookie。网站可设置自己的临时cookie；seat生命周期销毁时清理profile。不请求autoplay，视频错误、地区/年龄/登录限制及网络行为交由原生播放器/网站处理。
- typed媒体请求使用递增`sequence`及`media-<sequence>` ID，携带组件身份、Host revision、upsert/remove和经过校验的视频/几何字段。旧revision拒绝，remove销毁并退休该组件身份，旧upsert不能复活它；失效/过时回执不更新新组件。`native-view-created`、`hidden`、`closed`只确认原生视图放置/生命周期，不代表已验证视频存在、可用、加载或播放。
- 几何同步包含组件rect、祖先裁剪与viewport交集；离屏或被overlay遮挡时隐藏，native实际位置跟随Host缩放。标题最多512个UTF-8字节；viewport宽高1–4,096，rect宽高0–4,096，所有坐标有限且clip须位于viewport及有效rect内。没有新增seat数量产品上限。隐藏不承诺暂停播放/停止网络；完整销毁和限制见专文。
- 没有实际媒体/网络请求、浏览器执行或native运行验收，生产打包不验证播放器。业务Query/Mutation provider、script CDN和任意媒体URL仍未支持。

### 地图手势细项

- Ctrl/⌘+wheel 在鼠标位置锚定缩放；普通wheel保留页面滚动。使用原生non-passive监听，只在有效地图且非streaming时拦截已声明的地图缩放手势；提示文字可见。浏览器把触控板pinch作为Ctrl+wheel发送时沿用同一路径。
- 触屏单指平移、双指中心锚定pinch与平移；视图可为小数zoom，固定公共tile请求仍是合法整数z/x/y。zoom保持0–19，世界坐标水平环绕、纬度保持WebMercator边界，边界处会约束中心。
- 单双指切换从当前真实视图重新定锚，pointer capture丢失/取消、window blur、隐藏和卸载清理手势。正常marker轻点保持原label选择；移动/双指手势抑制收尾误点击。
- 放大可临时缩放已加载的较粗栅格；缩小时实际raster级别不高于当前view zoom的整数部分，避免枚举巨量微小tile。手势结束后再请求最终视图所需真实瓦片，不截断marker/可见tile数量、不伪造或替换tile。
- 坐标、路由、固定来源、来源署名和selected绑定没有新接口。zoom/pan是当前viewer的本地视图状态；没有新增跨窗口同步或后台定位。实际触屏、触控板、坐标锚定和WebView行为仍未运行验收。

## 动作与状态语义

| 契约 | 源级实现 | 未实现/待验收 |
|---|---|---|
| @Set | SDK本地计算并写绑定；同值写入也记录为dirty | 多步骤不是数据库事务；无跨窗口同步 |
| @Reset | SDK恢复当前声明默认值；pristine写清dirty | 流式重声明、同值写、重置后重开仅静态审查，未跑回归 |
| @ToAssistant / 无动作按钮 | humanFriendlyMessage + formState/context交宿主 | 保留原生确认/原会话发送；不能直接绕过确认 |
| 动作序列 | 先检查全计划/原生payload/真实手势；原生URL/文本固定为点击时快照，Set的valueAST仍由SDK按步骤读取当前store；clipboard/open_url等待Host实际回执后继续 | 未知动作整计划拒绝；执行中失败不继续后续步骤，但已完成的本地写入或系统操作不回滚；生命周期取消边界见下文；不是数据库事务 |
| open_url | 真实点击或键盘激活后的typed HTTP(S) Host请求 | 无自动调用、其他scheme或嵌入凭据；浏览器派发成功不等于网页加载成功 |
| @CopyToClipboard | 真实手势后复制指定完整文本；Host成功才显示成功 | 无浏览器clipboard fallback；NUL/不成对Unicode代理项或超事件预算明确拒绝，不截断 |
| 其他custom action | 拒绝 | 无业务动作provider |
| Query / Mutation | 声明在渲染前拒绝 | 无真实业务工具provider，默认数据不充当请求结果 |
| 初次恢复 | `$binding`及form键一次性原子恢复 | 只接受JSON可保留对象；非有限数/循环/超过128层等错误 |
| 流式默认值 | pristine键可跟随；dirty键保留本地编辑 | 不重放每次宿主render附带的旧state |
| 保存 | 仅dirty键快照；可分离JSON；宿主不接受时以后写入可重试 | 超48KiB保留本地编辑但明确无法持久化；重启丢失该未保存变更 |
| 错误/清错 | parser、SDK、边界异常可见；旧SDK错误不因新文本就清空 | resolved不等待资源成功，也不代表组件内部所有变体成功 |

### 剪贴板和外链动作边界

- `hostActions`缺省关闭，只能由真正内嵌的自有runtime启用；显式legacy viewer、缺少Host回执能力或不属于当前会话/实例的请求不能执行。详见[NATIVE-ACTIONS.md](NATIVE-ACTIONS.md)。
- 实际可信click或Enter/Space键盘事件同步建立一次许可，绑定本次计划已检查的payload；支持UserActivation的引擎另要求active。没有UserActivation的旧引擎仍要求当前真实`Event.isTrusted`处理栈，不用定时授权窗、不接受合成事件、timer、跨源HtmlView或自动render。
- sequence从1递增，requestId固定`native-action-<sequence>`；资源请求禁止该前缀。shell精确viewer来源/instance检查、Host当前owner及一次payload claim相互独立。没有新增通用native接口或放宽iframe/OS权限。
- 超过既有64KiB完整事件预算则不发送，原文不截断；文本NUL会在Windows文本剪贴板截断，因此跨平台统一拒绝；不成对Unicode代理项无法原样通过JSON/UTF8传输，也明确拒绝。
- 混合计划按原顺序分段执行，原生步骤等待真实回执。点击时已检查的计划及原生URL/文本保持固定；Set保留valueAST，由SDK在该步骤执行时读取当前store，因此后续Set可读取此前Set/Reset的写入，原生payload不会随之重新求值。普通响应式候选动作、标签或表单身份变化，包括计划自身Set/Reset引起的变化，不会取消正在执行的计划。
- 实际源程序/streaming修订、Host能力变化或组件卸载会取消尚未执行的步骤并撤销未完成许可；不会撤销已派发的系统操作。执行失败停止后续步骤，已完成的本地写入和系统操作不回滚。
- 同一控件执行期间保持pending并禁用，普通响应式更新不会重新开放重复提交。完成提示仅在当前控件候选仍与点击时调用匹配时显示，避免将旧payload的结果当成新内容的成功。等待Host回执15秒后只提示结果未确认，动作可能已经完成，不假称执行失败或成功；没有撤销已派发系统操作的承诺。
- Code复制、Link和已有CopyToClipboard/OpenUrl动作使用同一Host能力；Markdown/Text内嵌HTTP(S)链接已接入同一HostLink，具体边界见上节。Markdown代码块复制未扩展，HtmlView仍没有任何该能力。独立YouTube采用另行限制的`media`能力，不能作为任意媒体/CDN或通用native动作入口。

## 59个应用函数

全部为原创实现，保留公开函数名和参数次序；没有复制应用函数体。生成 schema 列出每个精确签名。

- 数组：Append, Prepend, Concat, Remove, RemoveAt, Update, Toggle, Includes, Find, IndexOf, Take, Slice, Unique, Reverse, Range, Flatten, Chunk, Pick, Shuffle
- 数值：Random, Pow, Sqrt, Clamp, Percent, Lerp, Allocate, Payment, Convert, Duration, MixColor
- 文本/格式：Default, Join, Plural, Pad, Upper, Lower, Trim, Capitalize, Split, Replace, Format
- 日期：Now, Today, FormatDate, AddDays, AddMonths, DaysBetween, MonthsBetween, StartOf
- 数据：GroupBy, Pivot, Top, Between, Bucket, Cumulative, MovingAvg, Change, Histogram, Stats

另外保留SDK本身的内置表达式函数，清单由构建生成的guide提供。不是任意JavaScript，也未提供未登记函数alias。

明确实现选择/边界：

- 数组不原地修改；对象字段为直接自有字段，拒绝prototype类字段。对象比较使用JS相等性，不声称结构深比较。Range末端排除；Flatten一层；Shuffle/Pick/Random为Math.random，非安全随机或可重播seed。
- Allocate按最大余数分配、限定可安全表示的总额精度；Payment为固定利率等额付款公式，无外部利率/金融服务。Convert只支持schema说明的同维单位，不猜单位。
- Format/FormatDate用当前WebView locale；percent按百分数单位输入。日期无时区服务，日期字符串按本地日历，StartOf周起点为周一，AddMonths按月末收敛。Now为调用时刻ISO文本，Today为本地日期；不会单独启动自动刷新时钟。
- GroupBy/Pivot按首次出现顺序；缺少数值不悄悄当零。Bucket按日期分桶，sum/count补空桶，avg/min/max不造零观测；不支持无限日期展开。
- Change(previous=0)返回null；其他情况使用绝对previous为分母。Stats使用总体标准差；空列表统计值为0。Histogram端点包含最大值；常量范围采用宽度1。这些未由公开描述完全规定的边界为本实现选择，尚不能宣称原版边界完全等价。
- 资源预算和异常会显示SDK错误，不生成伪结果。Range/Histogram至多10,000项，Pad至多16,384字符；functions-extra的数组输入至多10,000项，Bucket至多10,000期间，Pivot至多10,000输出单元格。Allocate分配余数循环不超过权重数，并拒绝精度溢出。

## 协议边界与验收缺口

沿用宿主预算：程序256KiB，状态48KiB，事件64KiB，assistant文本16KiB，错误4KiB，resolved程序32KiB；高度80–2000px。超过32KiB的合法程序可渲染，但不能发送超额resolved清错消息。API回包最多25MiB，HTML最多12MiB。Math公式20,000字符、最大展开1,000等安全预算也属于本实现边界。

必须在允许验证后补：真实流式增量与不完整错误恢复、错误去重/清错、脏键和Reset重新打开、混合Action的点击时payload与Set/Reset顺序求值、生命周期取消及pending/旧回执提示、表单动作原生确认、尺寸回传、异步资源失败/取消和署名、地图wheel/pinch锚点及单双指切换、图表负数与缺失值、hover/focus/键盘数据提示及更新后取值、SVG变换、popover焦点、Timer后台行为、YouTube媒体seat的真实手势/裁剪/缩放/遮挡/修订/销毁和临时profile隔离、真实视频与网络失败行为、Windows/macOS WebView的ES2022/MathML/CSS支持。当前只有源级核对和生产资产生成，不能宣称CH12全量验收或原版全功能一致。

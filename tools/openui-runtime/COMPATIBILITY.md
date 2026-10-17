# 源级兼容清单（2026-10-17）

此表描述独立实现代码的实际行为，不是测试通过清单。完整位置参数和枚举以同一 registry 生成的 `library-spec.json` 为准。UI 样式、具体默认间距和浏览器原生控件外观为自有设计，不宣称原版像素一致。Node 子槽经 SDK `renderNode` 递归渲染；没有将未知组件转换成普通文本或同名但语义不同的标准库组件。

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
| Text | 内联 Markdown/混合Node、xs/sm/md/lg、color/weight/italic/lineThrough/tabularNums/maxLines | 链接显式禁用；图片不加载；不支持内联任意 HTML |
| Caption | text/color | 纯文本副说明 |
| Label | text | 可见标签；字段实际关联由各控件实现 |
| Badge | neutral/info/success/warning/danger/purple/pink、Lucide icon | 未知 icon 显式提示 |
| Icon | name、sm/md/lg/xl、color；实际本地 Lucide 图标 | 仅安装版本存在的名称 |
| Markdown | CommonMark/GFM、表格、任务列表、代码块 | raw HTML 跳过；图片显式不可用；链接禁用；代码无语法高亮，Markdown数学扩展未接入 |
| Quote | text/cite | cite 为纯文本，不导航 |
| Code | code/language、横向滚动 | 语言标签可见；无语法高亮，复制明确不可用 |
| Math | tex/inline；本地 KaTeX 到原生 MathML | trust=false；不打包字体；原生MathML效果待各WebView验收；有解析/展开预算 |
| Link | text/url 被识别并呈现禁用链接提示 | 原版 http(s) 新窗口导航尚未实现，不能计为可点击链接 |
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
| Map | markers/route/selected/height；自动fit、拖动、按键/按钮缩放，label绑定选择、真实road route | 自有栅格，不是MapLibre；无wheel/pinch/旋转/倾斜；只支持WebMercator纬度范围 |
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
| HtmlView | 未注册 | 嵌套HTML/frame能力未实现；报缺失组件 |
| YouTube | 未注册 | 媒体/frame能力未实现；报缺失组件 |

### 图表细项

- bar: grouped或正负分离stacked；line/area: 普通或累计stacked，缺失值为gap；stacked行缺任一值时整行gap，不能默认为零。
- pie/donut: 每个series分别画有标签的真实图；零总量显示空状态；负值或缺失值显式报错。scatter要求数值x。
- stacked pie/donut/scatter 不支持且显式错误，不能将该参数静默忽略。每个图都可展开原始数据；SVG title与原始表提供可访问内容。
- 自有数值格式化与图例/轴样式。没有额外图表动画、悬浮tooltip或拖动缩放；未声明这些为已实现。
- 分配预算：Chart最多5,000行/32系列，Table最多50,000格；超限显示错误。预算是本实现边界，不是原版契约保证。

### 地图/图片桥接口

`requestResource<T>(path, query: Record<string,string>, signal?)` 返回已验证父窗口的响应体。

- `/api/map-tile`: z/x/y/style字符串；z=0..19、style=light/dark。回包 `{dataUrl, attribution, source, z,x,y,style}`。只用固定ArcGIS街道/深色灰底源，显示实际返回的来源署名；外部署名链接导航未实现。
- `/api/street-route`: coords为分号分隔lng,lat；回包 `{points:[[lat,lng],...]}`。父层只接受2–25个地点。失败保留错误，不以直线冒充道路。
- `/api/image-search`: q字符串；回包 images与imageSources等长数组。仅接受实际bitmap data URL，保留来源；不替换失败图片。
- `/api/wiki` 仅在桥允许列表中；原公开55组件没有Wiki组件，因此本包没有虚构同名组件或自动请求它。
- 请求取消和requestId去重；同时4个活跃请求以配合宿主队列，排队数量无额外产品上限。15秒活跃超时；32MiB地图缓存会淘汰并按需重载，不截断marker数量。

## 动作与状态语义

| 契约 | 源级实现 | 未实现/待验收 |
|---|---|---|
| @Set | SDK本地计算并写绑定；同值写入也记录为dirty | 多步骤不是数据库事务；无跨窗口同步 |
| @Reset | SDK恢复当前声明默认值；pristine写清dirty | 流式重声明、同值写、重置后重开仅静态审查，未跑回归 |
| @ToAssistant / 无动作按钮 | humanFriendlyMessage + formState/context交宿主 | 保留原生确认/原会话发送；不能直接绕过确认 |
| 动作序列 | 先检查全计划只含set/reset/continue_conversation，再执行SDK计划 | 含被禁动作则整计划不执行，不先偷偷改状态 |
| open_url | 拒绝，显示不可用 | 未接外部浏览器桥 |
| @CopyToClipboard | 注册签名以识别并明确拒绝 | 无复制权限/原生复制桥；名称存在不是能力完成 |
| 其他custom action | 拒绝 | 无业务动作provider |
| Query / Mutation | 声明在渲染前拒绝 | 无真实业务工具provider，默认数据不充当请求结果 |
| 初次恢复 | `$binding`及form键一次性原子恢复 | 只接受JSON可保留对象；非有限数/循环/超过128层等错误 |
| 流式默认值 | pristine键可跟随；dirty键保留本地编辑 | 不重放每次宿主render附带的旧state |
| 保存 | 仅dirty键快照；可分离JSON；宿主不接受时以后写入可重试 | 超48KiB保留本地编辑但明确无法持久化；重启丢失该未保存变更 |
| 错误/清错 | parser、SDK、边界异常可见；旧SDK错误不因新文本就清空 | resolved不等待资源成功，也不代表组件内部所有变体成功 |

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

必须在允许验证后补：真实流式增量与不完整错误恢复、错误去重/清错、脏键和Reset重新打开、表单动作原生确认、尺寸回传、异步资源失败/取消和署名、图表负数与缺失值、SVG变换、popover焦点、Timer后台行为、Windows/macOS WebView的ES2022/MathML/CSS支持。当前只有源级核对和生产资产生成，不能宣称CH12全量验收或原版全功能一致。

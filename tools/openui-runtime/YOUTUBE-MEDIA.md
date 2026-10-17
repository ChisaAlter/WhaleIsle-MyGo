# YouTube 独立原生媒体视图

此文记录独立编写的公开 `YouTube(video: string, title?: string)` 组件及其固定 Host 媒体协议。它没有复制原版应用实现，不将官方 SDK 的许可扩展到原版应用。当前 registry 登记 55 个公开契约组件；登记数量不是原版全功能或视觉等价声明。

这里只描述源码边界。未执行真实媒体/网络请求、项目测试、typecheck、浏览器 viewer、原生 helper 或 native 构建。生产打包仅生成资产，不能证明视频可用、能够播放或 Windows/macOS 原生行为已经验收。

## 输入契约

位置参数为 `video: string`、可选 `title: string`。标题是播放器的展示/可访问性说明，不是查询语句，也不证明它与远端视频标题相同；最多512个UTF-8字节。

`src/youtube-id.ts` 只提取一个视频 ID，不联网、不请求元数据、不确认视频存在或可用。输入须非空且最多2,048个JavaScript UTF-16 code units，不会先做trim。裸ID必须精确匹配11位 `[A-Za-z0-9_-]`。

URL必须显式使用 `http://` 或 `https://`。下表中的 `<id>` 均是上述11位ID；协议和主机名大小写不敏感，路径按表中大小写匹配。

| 主机 | 接受的路径 |
|---|---|
| `youtube.com`、`www.youtube.com`、`m.youtube.com` | `/watch`，query中必须恰有一个有效`v`；或`/embed/<id>`、`/shorts/<id>`、`/live/<id>`，后三种可带一个结尾`/` |
| `youtube-nocookie.com`、`www.youtube-nocookie.com` | `/embed/<id>`，可带一个结尾`/` |
| `youtu.be`、`www.youtu.be` | `/<id>`，可带一个结尾`/` |

- 拒绝字面空白、C0/C1控制字符、DEL、反斜杠，以及百分号编码的C0/C1控制字符和DEL。拒绝userinfo凭据、非法主机/端口和表外主机；只允许无端口或该协议的精确默认端口，HTTP为80、HTTPS为443。
- 路径在URL规范化前检查，额外路径段、点路径、编码路径、相对地址、无协议地址和非HTTP(S)地址不被接受。`/watch/`与`/watch`不同；watch中的重复`v`即使值相同也拒绝，解码后的唯一`v`仍须匹配ID。
- `/embed/videoseries`和`/embed/live_stream`是列表/频道入口，明确拒绝。这里只支持固定单一视频目标，没有播放列表、频道或搜索入口。
- 除watch的唯一`v`用于提取ID外，输入query和fragment都被丢弃；不透传`autoplay`、起播时间、列表或其他选项。Host使用校验后的ID构造固定播放器目标，不能借URL输入访问任意站点。

## 点击后才创建

组件初始显示本地Load YouTube video入口，不因render或解析成功自动创建联网播放器。`media`能力默认false；未启用、streaming期间、无有效Host revision或输入/几何无效时不能加载。

首次创建要求可见且未遮挡的Load YouTube video按钮真实点击，包括键盘激活按钮产生的可信click。自有viewer检查当前文档、可信事件、主按钮和同一事件未被使用；引擎提供UserActivation时还要求其active。没有自动播放请求、定时授权窗或合成事件许可。媒体网站之后的播放、登录提示、地区/年龄限制和错误由原生播放器/网站决定；创建视图不等于自动播放或绕过限制。

组件只在OpenUI helper内保留本地占位及控制UI。联网内容由Host另行创建原生内嵌浏览器视图（media seat）并放置在该区域。它不是无网络helper中的YouTube iframe，也不是用外部浏览器链接代替嵌入播放。既有helper的CSP、frame准入、OS无网络约束和资源桥边界不为媒体而放宽。

## 临时浏览器隔离

- 媒体seat使用全新临时profile和独立cookie store，配置`Untrusted`与`DisableBindings`。
- 不传入SessionOrigin、Cookie、Preload或native bindings；不从用户现有浏览器、会话或OpenUI helper复制cookie。媒体页不能通过绑定调用OpenUI状态、assistant、资源或系统动作。
- “临时”不意味着没有cookie。YouTube及其加载的网站资源可在此临时store中设置自己的cookie和网站状态；这些不是用户既有登录态，不能声称用户已经登录。
- seat生命周期结束时销毁原生视图及其临时profile/cookie store。仅隐藏是可见性变化，不承诺暂停视频、停止音频/网络请求或立即删除网站状态。
- 这是固定YouTube组件的专用能力，没有提供任意媒体URL、通用浏览器、script CDN或业务Query/Mutation provider。HtmlView没有`media`能力，不能创建或控制这些seat。

## 最小wire与归属

继承现有`channel=whale-openui`、`version=1`与64hex `instanceId`。媒体能力和当前正整数`revision`由可信Host随render提供；仅owned runtime、当前会话/实例所有权满足要求时启用，不能凭viewer自报名称获得能力。

请求类型为`media`，包含安全正整数递增`sequence`、固定`requestId=media-<sequence>`、布尔`userGesture`和typed `media`载荷：

- `operation`：`upsert`或`remove`
- `componentId`：本次组件/seat身份
- `revision`：创建该身份时的Host修订
- `videoId`与`title`：固定单视频ID及受限标题
- `rect`、`clip`：`{x, y, width, height}`
- `viewport`：`{width, height}`，以及布尔`visible`

首个upsert要求`userGesture:true`。该字段本身不能证明用户手势：精确viewer来源、实例、owned runtime、当前owner、Host revision和单调sequence仍须校验；后续已授权seat的几何更新不需要每次重新点击。没有为任意native调用开放通道。

旧revision不能创建或更新seat。remove销毁并退休该组件ID；晚到的旧upsert不能使它复活，显式重试使用新的组件身份和新的真实手势。视频ID或标题变化、程序修订、streaming、能力撤销、组件卸载和宿主生命周期结束会清理对应seat；同一已失效请求的回执不会应用到新组件或新修订。媒体请求不新增seat数量产品上限，现有事件预算及字段合法性检查仍然适用。

remove/组件卸载遇到既有Host队列已满或关闭回执失败时，只自动重试关闭请求，直到收到`closed`确认或该source/媒体能力退出。创建与播放绝不自动重试，隐藏也不承诺暂停播放。

## 放置、裁剪与遮挡

viewer报告组件完整rect，并将viewport与可裁剪祖先求交得到clip。滚动、布局和尺寸变化更新几何；离屏、文档隐藏或与已识别overlay重叠时请求隐藏。遮挡检测包含自有Popover/Celebration、声明的dialog/overlay，以及fixed/sticky层；相交即保守隐藏整个seat，不尝试将原生视图插入HTML的任意z-index层级。Host再按其实际native缩放转换坐标并放置/裁剪，不能直接把CSS像素当成所有平台的native像素。

容器隐藏、inert/aria-hidden、半透明、旋转/镜像/透视等非矩形变换，以及clip-path/mask等无法用矩形协议表示的裁剪会使seat保持隐藏并显示原因。轴对齐的正向缩放和平移可表达。没有宣称原生seat能够还原任意CSS效果或与页面像素一致。

几何字段必须有限。viewport宽高各为1–4,096，rect宽高各为0–4,096；rect的x/y绝对值不超过1,000,000。clip的x/y/宽/高均非负，不得超出viewport；非空clip还必须位于对应rect内，边界比较允许0.5的舍入容差。`visible:true`要求clip宽高均大于0。测量超出范围时viewer请求隐藏并说明原因，非法wire几何被拒绝，不通过裁剪欺骗跨窗口放置。上述为安全/可表示范围，不是对任意CSS布局或native渲染的已验收保证。

## 回执含义与待验收事项

Host回执经当前父窗口/通道/实例校验，以原`requestId`匹配。已退休、取消或超时请求的旧回执被忽略。已知状态只有：

- `native-view-created`：原生播放器视图已创建/放置
- `hidden`：原生视图隐藏
- `closed`：原生视图关闭

这些状态不报告YouTube视频元数据、加载完成、播放进度、音频状态或视频可用性。15秒未收到匹配回执只表示Host结果未确认，不能宣称播放器未创建或视频已失败。OpenUI的`resolved`也不等待或证明视频播放。

在允许验证后仍需检查：Windows/macOS真实手势及重复点击、旧revision/乱序回执/remove后迟到upsert、滚动和祖先裁剪、native缩放、overlay遮挡、离屏恢复、streaming/程序替换/组件卸载/窗口关闭，以及临时profile/cookie隔离和清理。实际视频错误、网络失败、地区/年龄/登录限制和网站播放器表现尚未验证；不得将源级实现或生产打包报告为已验证播放。

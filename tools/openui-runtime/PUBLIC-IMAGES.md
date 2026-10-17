# 公共图片父层资源接线

此文描述独立Image及Markdown/Text图片的源级范围。只复用现有owned Host资源桥；不把公共网络权限授予OpenUI viewer/helper，不扩展HtmlView资源能力。没有复制未许可应用组件。生产资产生成不等于实际获取、浏览器解码或安全验收。

## 输入与显示

- `Image(src, alt, aspectRatio?, radius?, fit?)`保留原有位置参数及样式行为。原有本地PNG/JPEG/GIF/WebP base64 data输入路径不变，不因为远程格式扩展就放行其他本地data MIME或SVG。
- 绝对公共HTTP(S)图片经`PublicImage`请求Host。远程URL从不直接赋给viewer的`img.src`；只有成功回包中通过校验的位图data URL，或带真实Host验证标记的严格静态SVG data URL才进入img。非法地址、缺少资源能力、获取失败、无效回包和解码失败均显示错误及可用的alt，不用库存图、占位假图或其他图片冒充结果。
- 独立Image保留alt/aspectRatio/fit/radius，提供显式重试。重试重新发起当前图片请求，不把失败状态当作成功。
- Markdown/Text的图片复用同一`PublicImage`，以span/img组成内联内容，保留alt及真实错误/来源；不插入figure/div/details或重试按钮，避免图片链接形成嵌套按钮或破坏段落结构。链接仍由既有`HostLink`处理，须真实手势和Host动作能力，图片获取本身不赋予导航许可。
- 来源显示Host返回的实际最终URL，不把请求输入伪称为最终来源。来源URL不包含fragment，也不代表作者、图片许可或再分发授权；没有这些元数据时不编造它们。

## 现有消息形状

调用仍是`requestResource('/api/image', {src}, signal)`，没有新增通用网络、native binding或JavaScript执行接口。

| 方向 | 内容 |
|---|---|
| viewer → 已验证父层 | `type: "api-request"`、`requestId: "owned-N"`、`path: "/api/image"`、`query: {src: "绝对HTTP(S)图片URL"}` |
| 父层 → 当前viewer | `type: "api-response"`、匹配的requestId、`status: 200`、`body: {dataUrl, source, validatedSvg?: true}` |
| 失败回包 | 同一requestId、失败status及真实错误消息；前端明确显示错误 |

这些字段仍受既有父窗口、`whale-openui`通道、version、instance和宿主所有权检查。`source`取自实际完成获取的最终重定向URL并移除fragment；`dataUrl`携带经过Host格式/预算校验的图片。只有imagepreview既有`validateSVG`成功且返回`data:image/svg+xml;base64`时，Host才附`validatedSvg: true`；位图省略此字段。前端仅在真实Host回包中同时见到该标记和SVG data MIME时接受SVG，直接用户提供的data SVG仍拒绝。HTML、未验证SVG和任意响应体不能通过此端点变成可执行viewer内容。

## 公共地址与格式边界

- 仅绝对HTTP(S)源地址可进入Host获取流程；带userinfo凭据、非HTTP(S)或私有/本地地址不允许。Host校验解析出的地址及实际连接目标，每次重定向重新校验公共地址，不依赖前端字符串检查来防止访问内网。
- 不携带用户Cookie、Authorization或原生凭据，不提供任意method/header/body透传。该端点只取受限的公共图片，不是访问内网、账户内容或其他资源的通用代理。
- 远程位图格式范围是PNG、JPEG、GIF、WebP、BMP、ICO（data MIME为`image/x-icon`）及静态AVIF；由Host校验格式及尺寸/帧预算后返回。
- 远程SVG复用imagepreview既有`validateSVG`，不扩大validator子集。仅通过严格静态验证并附`validatedSvg: true`的真实Host回包可展示；不把SVG伪装成位图，也不允许直接用户data SVG或未标记SVG。此范围不包含任意视频、动画AVIF、HTML、脚本或其他媒体格式。
- SVG维持1MiB、32M像素、单边32,768、最多10,000节点/40,000 tokens/64层预算。准入范围仅既有静态vector/text/gradient/local clip-mask和无环local reference；script、动画、filter、嵌套image、foreign content、external resource及CSS escapes均拒绝。
- Host准入不承诺所有WebView都能解码该格式。浏览器报告解码失败时保留真实错误，不切换其他网络地址、不生成替代图，也不把格式准入称作播放/显示成功。

## 预算与排队

| 层级 | 边界 |
|---|---|
| Host原始图片响应 | 最多16MiB |
| Host尺寸校验 | 32M像素预算 |
| GIF/ICO | 既有至多256帧/条目及累计像素预算继续生效 |
| SVG静态子集 | 最多1MiB、32M像素、单边32,768、10,000节点、40,000 tokens、64层；既有validator不扩展 |
| Host总获取时间 | DNS、重定向及响应体读取共享8秒预算，不是每步各8秒 |
| 资源API回包 | 最多25MiB，包含编码后的data URL及回包内容 |
| 前端资源传输 | 同时4个活跃请求，其他排队；活跃等待15秒超时 |

这些是单资源、传输和解码预算，不新增程序图片总数或排队数量的任意产品上限。超限明确失败，不截断程序中图片清单或用较小的假图掩盖失败。SearchImage的既有搜索结果范围、Map瓦片/路线协议和其他资源限制不因此被放宽。

## 取消与过时结果

- 每次图片请求有自己的AbortController及live/身份检查。src、源程序、viewer实例变化或组件卸载使旧结果失效；旧成功、失败和解码回调不能覆盖当前图片。显式重试也建立新的请求身份。
- 前端abort取消尚未派发的排队任务，或结束已经派发请求的前端等待；资源桥移除匹配的pending记录并忽略迟到回包。它不是Host网络取消确认，不承诺已派发的网络操作当时就停止。Host获取仍受自身预算约束。
- 同时4个活跃请求是前端资源传输调度边界；取消一个已派发请求的前端等待，不应被描述为已证明对应Host网络连接立即释放。
- `resolved`仍只表示程序解析/SDK检查和React提交调度完成，不等待图片获取或解码，也不能作为图片成功回执。

## 未扩展的能力与验收缺口

viewer/helper仍无直接网络、worker、远程脚本、媒体或任意原生能力。CSP、native资源拦截与OS无网络沙箱继续共同提供隔离；单独把组件放到普通浏览器不等于相同安全边界。HtmlView没有此资源桥，其内部图片、CSS URL和其他远程资源不因本次接线获得访问权限。独立YouTube媒体入口也不成为图片代理。

本次不运行项目测试、typecheck、原生构建、viewer/helper浏览器或实际图片获取。后续在允许验证后需覆盖：公共地址和重定向/DNS拒绝、最终来源去fragment、所有支持格式、严格静态SVG准入与直接data SVG/未标记或不安全SVG/损坏文件拒绝、字节/像素/帧/累计像素/时间/回包预算、超过四图的排队、取消与迟到回包、src/程序/实例切换及卸载、独立Image重试、Markdown/Text链接内图片结构、WebView原生解码失败，以及HtmlView仍无资源桥。生产打包成功不能替代这些运行和跨平台验收。

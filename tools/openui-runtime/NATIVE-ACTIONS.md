# Owned Host剪贴板与HTTP(S)动作

本模块为自有源码接线，生产打包不执行剪贴板、URL、用户HTML或native构建；实际用户交互和Windows/macOS系统接口尚未运行验收。

## 最小wire

- 继承`channel=whale-openui`、`version=1`、64hex `instanceId`。
- Render新增`hostActions:boolean`，默认false；shell使用`r.HostActions && IsOwnedViewer(viewerHTML)`，Host启动还要求没有显式legacy路径。不能凭viewer自报runtime名称启用。
- Request：`type=clipboard`附`text:string`，或`type=open-url`附`url:string`；另有`requestId=native-action-<sequence>`、JS安全正整数`sequence`、`userGesture:true`。
- `userGesture`本身不是证明：仅实际owned源码、精确viewer contentWindow和当前实例的消息可以进入Host；前端只在真实事件处理栈建立绑定payload的内部许可。HtmlView跨源消息不能伪造source身份。
- Host复用既有nonce/FIFO `APIResponse(instanceId, requestId, status, body:{ok,error?})`。shell按actionPending中的原始动作类型映射为`action-response`，含`requestId, action, ok, error?`；仅status为2xx且body.ok=true才成功。资源响应分支不变，资源requestId不可使用native-action前缀。

## 端到端语义

Host校验单调序列、请求ID、typed payload、会话能力和一次claim；在UI线程执行之前再次核对当前owner/实例/helper生命周期。实际`Clipboard.Write(transfer.TextData)`或已有HTTP(S) `OpenExternal`返回错误即失败。前端只根据匹配的真实回执显示Copied/Sent to browser；后者不表示远端网页已加载。

不执行`navigator.clipboard`、`execCommand`、`window.open`或iframe导航。URL限定绝对http(s)、非空hostname、无userinfo/控制字符。Code复制使用原始code字符串，和高亮DOM或高亮预算无关；空文本可复制。NUL和不成对Unicode代理项因无法跨平台完整保留而明确拒绝；完整消息仍受既有64KiB事件上限约束。

混合Action计划先整体校验；原生payload在手势内绑定，不用全局时间窗。SDK负责原有Set/Reset/assistant本地语义；原生步骤等待Host确认后才继续。失败停止后续步骤，但不会撤销此前的本地写入或已完成系统操作。纯本地动作不要求新增原生手势许可。控件动作参数、标签或表单身份在异步步骤之间改变时，剩余步骤取消；这也包含本地Set导致响应式动作参数改变的情况，因此不宣称该边界的混合计划完整等价。

同一真实事件只建立一次许可，内容/程序/streaming/能力变化或组件移除后旧结果不能伪装新控件成功。15秒未收到结果只表示无法确认，不能证明系统没有执行。不存在自动重试复制/外链动作。

## Markdown/Text内嵌链接

Markdown和Text的内嵌链接复用独立Link组件的`HostLink`和同一真实click/键盘手势许可。目标必须为绝对HTTP(S)地址、非空hostname，且无userinfo凭据或控制字符；有效目标也必须满足既有Host能力、streaming、payload和事件预算限制，收到真实Host回执才显示结果。没有目标地址时保留普通内容；非法地址或非HTTP(S)地址显示禁用提示。

链接标签保留原有React子内容，包括粗体、斜体和内联代码。动作身份绑定URL payload，不把React节点序列化为动作身份或Host消息。没有新增导航`href`、iframe导航、远程图片加载或Go桥协议；Host仍只接收既有typed `open-url`请求。

## 剩余范围

Markdown代码块复制维持原状，没有新增复制入口；独立Code组件保留已有复制原文能力。HtmlView无本模块能力，其他scheme、任意native方法、媒体/YouTube/CDN均未增加。Linux现有OpenUI桥未接入该能力。此模块不改变OS设置、token/entitlements或helper自身的网络/剪贴板权限。

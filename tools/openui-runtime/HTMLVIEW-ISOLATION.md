# HtmlView 内容与宿主隔离接线

此模块为自有实现；仅使用原公开参数契约，没有复制未许可应用组件。生成资产不是运行或安全验收结论。

## 文档与消息边界

```
trusted host shell (main frame)
  -> one opaque OpenUI viewer (existing native frame identity)
     -> zero or more opaque HtmlView srcdoc documents
```

HtmlView `sandbox`只有`allow-scripts`。每个文档在用户HTML之前声明更严格CSP；父CSP仍同时生效。组件只处理来自当前真实contentWindow且origin为null的高度、加载、脚本错误与有限策略类别通知。消息不转发宿主；内层文档没有资源、状态、clipboard、导航、assistant或原生调用接口。高度是非可信展示值，按既有2,000px边界校验，无法扩大宿主权限。

可信shell初始`render.htmlView=false`；只有已接线的helper可通过固定Eval调用`__whaleOpenUIHtmlViewSupport(boolean)`设置支持位。该setter沿用原render消息，不重置程序或SDK状态；子帧消息不能设置它。viewer只接受布尔值，缺省按false处理。组件在false时显示明确不可用原因而不创建嵌套iframe，因此缺少新native API的环境仍可继续使用其他OpenUI组件。

更换HTML/可信程序修订改变/进入streaming/移除组件都会销毁旧文档并取消前端资源准备与等待，旧回包不得创建或覆盖新文档。相同HTML的普通SDK状态rerender以及height/title调整保留当前文档；不读取或持久化子文档的JS状态。streaming期间不获取脚本、不创建iframe；前端abort不代表已派发Host网络操作已经停止。

## 静态classic脚本的准备边界

可信外层组件使用parse5 8.0.0（MIT）解析原始输入并读取source locations，不执行HTML。只为原始活动HTML命名空间的classic script外部src调用既有资源桥`/api/html-script`，仅接受无凭据的绝对公共HTTP(S)，拒绝`crossorigin="use-credentials"`；template内或脚本运行后新增的节点不触发请求，外部foreign namespace script明确拒绝。Host成功返回`{dataUrl, source, integrityVerified}`；非全空白integrity必须先由Host验证并标记true。外层定点替换原src属性为返回的base64 JavaScript data URL，并移除已由Host处理的integrity/crossorigin属性；其余markup次序和async/defer属性保持原样。所有准备成功后才创建opaque iframe，准备失败时任何用户HTML均未执行。

内层CSP的`script-src`只新增`data:`，不开放HTTP(S)脚本或`unsafe-eval`。此改动不授予内层资源桥、不转发其消息，也不改动外层CSP、native请求/导航拦截、OS无网络边界、worker/frame/media或原生权限。脚本的`currentScript.src`会是data URL；相对资源、动态远程加载、module/import、fetch及远程CSS依赖仍不支持。等待所有资源会改变async脚本的网络到达时序，不能宣称任意CDN库兼容。完整协议、预算与未验收项见[HTMLVIEW-SCRIPTS.md](HTMLVIEW-SCRIPTS.md)。

## Native最小接线

- macOS：`native/macos-preview/main.m`仅在OpenUI模式保留已准入的viewer WebFrame对象；第二层精确`about:srcdoc`仅在parent等于该对象，且viewer.parent仍等于mainFrame时准入。viewer与每个HtmlView实际frame仅准入一次初始srcdoc；内容更新必须换iframe。弱引用、指针身份集合保留仍存活的已准入HtmlView，以局部处理销毁的`NSURLErrorCancelled`和`WebKitErrorFrameLoadInterruptedByPolicyChange`，不吞掉shell/viewer或其他错误。原有普通HTML直接子帧规则不变。第三层、其他URL、新窗口、下载、文件选择、OS entitlement与签名检查均不放宽。
- Windows：需要实际frame身份/父子关系的专门准入，不能删除全局`frameStarted`检查后放行所有srcdoc。应随本模块的Windows源码接线原子发布。第二层直接子帧事件依赖`ICoreWebView2Frame7`（WebView2 1.0.3240.44+）；实际接口查询与订阅成功才启用。缺少接口或订阅失败时保持HtmlView能力false，不以此中断基础viewer。
- 本包不修改普通HTML助手行为、Host凭据、OS token/entitlements、原生绑定或网络权限。native编译/运行均未执行。
- 本模块的受限层级接线范围是macOS与Windows；Linux没有在本次实现身份/深度准入，不能以共享React源码存在就计为Linux功能完成。

## 为何保留frame-src none

HTML规范直接从srcdoc建立文档，其navigation request为null，并复制父策略容器；因此不为本模块放宽外层`frame-src`。它也不能替代native实际深度检查。该结论是标准与源码推导，特定WebView行为仍需运行验收。

- [HTML srcdoc navigation](https://html.spec.whatwg.org/multipage/browsing-the-web.html#create-navigation-params-from-a-srcdoc-resource)
- [HTML policy container inheritance](https://html.spec.whatwg.org/multipage/browsers.html#determine-navigation-params-policy-container)
- [Multiple CSP policies](https://www.w3.org/TR/CSP3/#multiple-policies)
- [CSP WebRTC directive](https://www.w3.org/TR/CSP3/#directive-webrtc)

opaque origin限制父DOM访问，不意味着凭据自动从任意网络请求中移除。`connect-src none`也不是所有引擎的完整WebRTC网络隔离。这里继续依赖已有OS无网络沙箱和native请求拒绝策略；不得把本组件单独放在有Host能力的普通浏览器页面并宣称具有同等保证。

## 支持及明确缺项

源级支持本地HTML/CSS/inline JS、DOM事件、输入编辑、canvas与动画循环，以及上述原始静态classic脚本预取。除上述静态脚本以外的远程资源、外部媒体/YouTube、远程CSS、原生动作和跨文档状态保存仍不可用；不能将静态脚本例外当成直接网络支持。自动高度是布局测量与内部滚动，不是任意viewport依赖CSS的通用解算器。任意可执行JS可能耗尽helper的CPU/内存；未实现执行时间上限。脚本预取成功不保证库可用，运行时错误/CSP拒绝仍显示真实状态；没有真实CDN下载、脚本执行或native运行验收。

# HtmlView 本地内容与宿主隔离接线

此模块为自有实现；仅使用原公开参数契约，没有复制未许可应用组件。生成资产不是运行或安全验收结论。

## 文档与消息边界

```
trusted host shell (main frame)
  -> one opaque OpenUI viewer (existing native frame identity)
     -> zero or more opaque HtmlView srcdoc documents
```

HtmlView `sandbox`只有`allow-scripts`。每个文档在用户HTML之前声明更严格CSP；父CSP仍同时生效。组件只处理来自当前真实contentWindow且origin为null的高度、加载、脚本错误与有限策略类别通知。消息不转发宿主；没有资源、状态、clipboard、导航、assistant或原生调用接口。高度是非可信展示值，按既有2,000px边界校验，无法扩大宿主权限。

可信shell初始`render.htmlView=false`；只有已接线的helper可通过固定Eval调用`__whaleOpenUIHtmlViewSupport(boolean)`设置支持位。该setter沿用原render消息，不重置程序或SDK状态；子帧消息不能设置它。viewer只接受布尔值，缺省按false处理。组件在false时显示明确不可用原因而不创建嵌套iframe，因此缺少新native API的环境仍可继续使用其他OpenUI组件。

更换HTML/进入streaming/移除组件都会销毁旧文档。相同HTML的普通rerender以及height/title调整保留当前文档；不读取或持久化子文档的JS状态。

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

真实支持本地HTML/CSS/inline JS、DOM事件、输入编辑、canvas与动画循环。原公开CDN例外没有实现；外部媒体/YouTube、远程资源、原生动作、跨文档状态保存均不可用。自动高度是布局测量与内部滚动，不是任意viewport依赖CSS的通用解算器。任意inline JS可能耗尽helper的CPU/内存；未实现执行时间上限。

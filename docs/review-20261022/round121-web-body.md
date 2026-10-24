# Round121 · Web retrieval result cards

本轮将原生聊天中的 `web_search` / `web_fetch` 首方结果接到已安装鲸屿的 WebBlock 展示规则；整体「布局、UI、功能、动效完全一致」目标仍未完成。本轮实现已构建为 [pass265](<C:/AI/WhaleIsle-MyGo/whaleisle-pass265.exe>)，并更新 [默认程序](<C:/AI/WhaleIsle-MyGo/whaleisle.exe>)。上一轮提交为 `69baaaf8`。原版进程20012、后端16008、LockApp12516及GUI3080保持未修改。

搜索结果现在显示完整 Markdown 答案、按真实顺序编号的来源、完整换行摘要及原始日期；来源区最高320DIP并独立滚动，答案留在来源区之外。抓取卡片显示最终元数据URL、HTTP状态和内联截断提示，标题链接独立指向调用参数中的原始URL；无额外复制、内层折叠或重复IN/OUT。缺失或不合法的结果元数据仍回退到该首方调用方拥有的OUT证据。文本模式保留原先的原始结果入口。实现见 [模型](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_web_model.go>)、[适配](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_web.go>)、[卡片](<C:/AI/WhaleIsle-MyGo/internal/ui/widgets/markdown/tool_webblock.go>)、[链接规则](<C:/AI/WhaleIsle-MyGo/internal/ui/widgets/markdown/tool_web_url.go>) 和 [本地化](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_web.go>)。

模型仅由保留的调用头、成功完成的根结果及持久化原始元数据派生，不改Host、会话或工具业务数据。精确名称匹配；查询数组不额外限制为4项；可选字符串显式null不被接受；抓取状态按JS有限整数语义保留负数、零、巨大值和IEEE舍入；元数据中的任意URL字符串仍可展示。链接使用新增的维护中WHATWG解析依赖，只有浏览器可解析的HTTP(S)地址变为可激活链接，回调保留原始拼写；www、IDNA、IPv6、凭据和非常规HTTP写法按浏览器解析而非手写net/url规则。点击及Enter激活，Space被消费而不激活、不折叠外层。Markdown答案中的URL激活也修正为同样的键盘行为，现有Markdown目标地址筛选策略保持其已有实现。详见 [模型样例与边界](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_web_model_test.go>)、[URL样例](<C:/AI/WhaleIsle-MyGo/internal/ui/widgets/markdown/tool_web_url_test.go>)、[组装验证](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_web_view_test.go>)。

已安装网站图标表的40个主机后缀/34个CC0 simple-icons路径提取为 [SVG数据](<C:/AI/WhaleIsle-MyGo/internal/ui/widgets/markdown/tool_web_glyph_data.go>)，最长主机后缀优先，未知主机使用已安装中等描边地球。标题栏搜索使用规则地球、抓取使用Browse图标，精确首方名称先于一般变体映射。来源链接按内容宽度布局，抓取链接填满卡片宽度；焦点轮廓按记录的2DIP宽/1DIP偏移绘制，悬停或键盘焦点使用3DIP偏移点状下划线。

长抓取URL使用新原生 [BreakAll API](<C:/AI/WhaleIsle-MyGo/third_party/mygo/ui/element.go#L1017>)，由生成的Element外观接入。DirectWrite CHARACTER、Pango WRAP_CHAR、CoreText cluster break及无字体回退均实现逐字素换行，NoWrap优先，缓存键及最小宽度包含新策略。测试发现DirectWrite在1DIP宽度会拆开旗帜区域指示符，已在 [共享布局](<C:/AI/WhaleIsle-MyGo/third_party/mygo/internal/text/layout.go#L804>) 只修复不合法的原生字素断点，继续使用原生字形排版、保留span与原始复制字节。测试及平台范围记录于 [框架检查](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-framework-checks.json>)；Linux/macOS仅交叉编译与vet，未在Windows上宣称运行验证。已有全框架分配预算/24DIP省略测试问题未在本轮解决。

| 实测维度 | 记录与原生结果 |
|---|---|
| 卡片 | padding12/14、radius16、首方caller margin4/0/4/4；620宽容器产生616宽卡片 |
| 来源 | 字体14/20、字重500；marker继承14/normal，2.5em=35；标题行20高，一行摘要及日期后63高；两来源+截断卡片145高 |
| 摘要/日期/提示 | 摘要13/19及日期13/20，各top2；来源间距10；截断13/20、top8；纯ASCII空白摘要/日期为空块且相邻margin折叠 |
| 空态 | 无答案无来源44高，截断72；空白答案保持存在且32；纯答案56 |
| 抓取 | URL13/19等宽字、字重500、图标1.1em及right5、gap6；元数据13/20、gap12；普通卡片69高 |
| 对照 | [独立核对](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-reconciled.json>) 共26/26几何项、20/20参考哈希一致，参考主JSON+源JSON共246212字节 |

已安装源码VM分别执行发射模块、打包模型、URL/标签、首方调用方与纯JSX卡片；浏览器使用真实已安装CSS及主题，保留一次独占Chrome DPR1验收。该验收明确为 **FAILED：8/10场景，46/48断言**，没有为取得通过而重跑。第一条失败是compact样式投影漏掉alignItems，原始完整S17记录能独立确认baseline及gap12，但不改原始失败结果。第二条组合了target清单与anchor属性，却未保存每个子结果/targetInfos，因此仍未解决、无法确认。完整 [原始验收](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-reference-2026-10-24T04-14-24-538Z-4fd967fc.json>)、[原始报告](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-reference-2026-10-24T04-14-24-538Z-4fd967fc-report.md>) 和 [失败分析](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-reference-2026-10-24T04-14-24-538Z-4fd967fc-analysis.md>) 保留不变。

参考浏览器89个发射/打包模型样例、21URL+5标题链接样例、6纯卡片分支、5ToolRow调用方和10静态预检均通过。105个来源明确编号、滚轮0→2820及实际5px滚动条保留；Tab/包含CR文本的Enter/Space和4指针+2Enter本地preventDefault载荷通过。ChromePID41384、端口49173及10个观察到的子进程均退出，临时profile删除。参考答案DOM为本地构造的段落/标题/粗体/链接，并未执行实际Markdown管线、React或Host；fetch标题图标在浏览器fixture中是地球近似，准确的Browse14来自已安装调用方源码VM。来源已知网站图标为源码及原生绘制证据，非浏览器已知主机渲染证明。截图：[参考浅色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-reference-2026-10-24T04-14-24-538Z-4fd967fc-light.png>)、[参考深色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-reference-2026-10-24T04-14-24-538Z-4fd967fc-dark.png>)、[三位数来源](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-reference-2026-10-24T04-14-24-538Z-4fd967fc-scroll.png>)。

最终受影响包回归全部通过：Chat68.874s、Frame12.112s、Markdown3.114s、i18n0.616s；chat/markdown分别运行MyGo生命周期分析均通过，受影响包go vet通过。记录于 [回归检查](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-affected-checks.json>)。早期实际失败均调查并修复：调用了不存在的WordBreak API、标题行高估1DIP、测试HasText将“Web result”误匹配卡片辅助名称、框架并发编辑期间的暂时签名未就绪，以及PowerShell内联Node引号/终端编码问题；最终代码使用公开BreakAll并通过上述检查，不将早期失败当作通过。

原生记录fixture初次PID10912及最终构建PID18500均通过注入MyGoSurface的点击、Enter、Space、UIA Link Invoke及主题稳定性检查，scale1.5；只记录回调，不打开真实浏览器、不发网页请求、不改OS剪贴板。两个fixture均已close并验证退出。见 [初次原生记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-native-state.json>)、[最终原生记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-native-release-state.json>) 和 [probe](<C:/AI/WhaleIsle-MyGo/scripts/tool-web-native-probe.ps1>)。软件渲染截图：[搜索浅色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-native-software-search-dark-false.png>)、[搜索深色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-native-software-search-dark-true.png>)、[抓取](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-native-software-fetch-light.png>)。这些不是物理输入或整机显示测试。

构建使用Go1.27.2及CGO_ENABLED=0，默认与pass265均86296576字节，SHA256为 `A62292738B575A741062405FFD207BD7F227C8758EDA9AD0559126D8A2878D48`；fixture51784704字节，SHA256为 `D0EA99944B40F6F0BCC915E60F5F61395C74B600952E5C78DF378BA6D972347A`。见 [构建记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-web-pass265-build.json>)。

仍待后续处理：图片结果body、详情类型/Inspect入口、lazy未完成参数、结果窗口与Inspector图结构/分页、其余工具重复参数策略、共享字体/token与tab基线、可选文本及多行链接片段的精确命中/装饰、虚拟滚动与memoization、真实剪贴板拒绝反馈、滚动条/tooltip/presence动效、Host主页/进程滚轮/物理输入、媒体、shell/tabs及材质控件、完整启动关闭。本轮web卡片仍沿用已有Markdown地址筛选及inline-link图标实现，多行链接下划线/命中片段与原版浏览器不作逐像素一致声明；原生滚动条与marker字形基线也未被26项盒尺寸对照证明。原版完整应用、真实网页导航、OS剪贴板及全框架测试均未作通过声明。

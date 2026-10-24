# Round122 · 图片工具结果与原图预览

本轮实现已安装鲸屿的 `read_image` 专用结果及通用工具图片结果，构建为 [pass266](<C:/AI/WhaleIsle-MyGo/whaleisle-pass266.exe>) 并更新 [默认程序](<C:/AI/WhaleIsle-MyGo/whaleisle.exe>)。整体「布局、UI、功能、动效完全一致」目标仍在进行。上一轮提交为 `1a0f0639`，参考Electron、后端及GUI3080未改动。

图片结果现在直接挂在工具行的外层展开区：移除配对调用中的第二层“图片集合”折叠、额外Copy以及重复参数/路径文件卡，显示缩略图、记录路径、每张图片尺寸/字节数，以及按原始顺序保留的额外文本。标题栏保留 `图片 (N)` / `Image (N)`；路径导航与缩略图打开图片互不折叠外层。首方读取图片的无卡片结果只显示OUT，不显示IN。未知工具的通用图片结果保留未知对象和数组的格式化JSON证据。其余已安装keyed registrant保持各自body优先级，避免图片抢走终端、文件diff或read等结果。实现见 [卡片](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_image_card.go>)、[聊天接入](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_lifecycle.go>)、[文案](<C:/AI/WhaleIsle-MyGo/internal/ui/i18n/catalog_tool_image.go>)。

[图片模型](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_image_model.go>) 由成功完成的原始结果派生，不读取记录路径作为本机图片。根调用需要保留的call head；PTC嵌套结果本身含调用名和参数，因此start已出保留窗口时仍可派生专用图片卡片。原版 `image-card-model`、`raw-tool-call` 在隔离Node VM执行159个样例，32个代表结果写入 [模型回归](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_image_model_test.go>)。测试保留JS trim、精确首方名、LF envelope、元数据无效时嵌套调用路径回退、附件数组原子校验、可选值null拒绝、正IEEE整数及大于native int的尺寸、nil/空字符串额外文本、未知块、原始/投影错误、重复附件及独立返回值。非数组畸形content在原版直接模型执行中抛TypeError，原生安全降级为原始输出。

附件ID按提供者不透明字符串处理：模型仅拒绝空字符串，原生session加载/缓存/重试不再TrimSpace，也不合并 `opaque`、` opaque ` 和纯空白ID。[精确ID回归](<C:/AI/WhaleIsle-MyGo/internal/store/session/attachment_opaque_id_test.go>) 经过本地RPC记录实际请求、缓存隔离及重试清理；本地端点仅测试数据，不是生产Host验证。既有session授权加载回调及内容缓存路径继续使用，记录路径从不成为附件读取或重试来源。

| 布局与状态 | 实现及证据 |
|---|---|
| body | margin4/0/4/4，无额外背景/描边；420宽行内416宽body |
| 元数据 | 原始path在前，每图片一行；等宽11/16，tertiary，top4；ASCII x、px和原版中英文逗号/字节格式，不额外千位分隔 |
| extra | 原始普通文本13/20、top4，保留前导空格、Tab、空行和Markdown字面符号，不送Markdown解析 |
| 单图 | 根据持久化尺寸预留singleFit；长边240，比例0.25–4，不超过自然尺寸；CSS最小44使32×16显示44×44；极高图顶对齐、极宽图左对齐 |
| 多图 | 64×64、gap10、自动换行，重复引用按顺序保留；206宽body的5张图为2/2/1，gallery高212 |
| loading | 保持预留尺寸；12/18加载文本；普通single/tile可聚焦且加载时不打开 |
| failure | 单图失败成为max-width240的12/18重试按钮，padding10/12、radius12；tile保持64×64、padding4、radius20；失败不会自动发重复请求 |
| frame | radius20；CSS.5px边框按device scale取整；浅/深色边框及danger填充来自已安装语义值；焦点2DIP、offset2 |

[原生布局预设](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_image_geometry_test.go>) 与 [独立核对](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-reconciled.json>) 通过17/17盒尺寸及位置，24/24参考源码哈希一致；单图frame240×135、元数据两行32、四行extra80，含margin的body263；多图body304。该对照证明盒尺寸，不证明字体光栅、CSS换行算法、图片裁切像素或全产品一致。

原图预览从固定880×620及底部文件名按钮改为窗口内自然比例：最大宽1600、四边40、避免放大，预览stage从标题栏下开始，遮罩也从该位置开始；36×36关闭按钮位于右20/标题栏下20。使用原生DialogBase约束Tab，Escape/背景点击/关闭按钮关闭并恢复原缩略图焦点。依据是 [精确已安装预览源码/CSS](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-preview-authority.json>)，不是较早只读TS中的全窗口居中规则。640×480、48高caption时图片框560×315、y106.5，关闭按钮x584/y68；见 [预览测试](<C:/AI/WhaleIsle-MyGo/internal/ui/chat/tool_image_lightbox_test.go>)。遮罩blur、浏览器presence退出保持和精确easing、缩放光栅、原生zoom-in指针，以及预览辅助名称从附件name传递仍有差异；预览图片alt目前为本地缓存basename。

参考Chrome验收保留 **FAILED：8/9场景，221/222断言**。静态预检175/175在唯一Chrome启动前通过；ToolRow/slot/locale/singleFit、gallery/loading/error/spacing、原始文本、局部输入与焦点检查通过。失败来自严格“目标清单只含file/blank”断言：当前Chrome即使禁用扩展仍创建Omnibox内部页面、Google Hangouts及扩展service-worker。完整目标清单已保存，没有附加/操纵这些目标，没有为通过而重跑验收。fixture网络检查没有HTTP/HTTPS/WS/WSS请求，但不宣称浏览器扩展全局无网络。见 [原始报告](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-reference-2026-10-24T04-52-44-248Z-778cf0b5-report.md>)、[原始指标](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-reference-2026-10-24T04-52-44-248Z-778cf0b5.json>)、[原始源码和golden](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-reference-2026-10-24T04-52-44-248Z-778cf0b5-source.json>)。source102063B+metrics82448B=184511B，保留21个可信注入事件；ChromePID38664及观察到的10个进程全部退出，55443端口释放，独占profile移除。早期静态准备稿从未启动Chrome且已清理。

浏览器执行精确installed emitted/bundle函数及CSS，但JSX为符号树+hook快照；披露、retry与lightbox是局部fixture处理器，未执行真实React effect、slot注册或Host授权加载。内联8×8PNG只是占位图，实际img238×238裁在240×135frame内，因此不认证真实附件裁切像素。图片预览stage的最终源级核对独立记录，未改浏览器验收数据。截图见 [参考页面](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-reference-2026-10-24T04-52-44-248Z-778cf0b5-initial.png>)、[原生浅色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-native-software-single-light.png>)、[原生深色](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-native-software-single-dark.png>)、[原生窄图库](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-native-software-tiles-narrow.png>)。

原生fixture只使用记录PNG及注入附件加载函数。第一次PID18272失败是probe把完整多行辅助名称误写为首行，产品未改；修正probe后PID27012通过。标题栏stage修正后PID9828也通过，最终精确中英文文案构建PID29124再次通过 [最终原生记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-native-final-state.json>)；每个fixture均关闭并验证退出。原生检查覆盖独立文件路径动作、图片点击/Enter/Space/UIA Invoke、modal Escape/焦点恢复、明确一次失败后一次重试、恢复图片打开以及深浅主题稳定，scale1.5。见 [失败记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-native-state.json>)、[通过记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-native-verified-state.json>)、[stage最终记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-native-release-state.json>)。没有生产Host、网页、OS剪贴板、物理输入或整机显示认证。

受影响包整轮回归通过：Chat55.160s、Frame12.317s、Session6.883s、i18n0.530s；chat MyGo生命周期分析和受影响go vet通过，见 [回归记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-affected-checks.json>)。最终stage改动追加针对性预览/图片测试2.733s、chat生命周期分析和vet；见 [最终检查](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-final-checks.json>)。最终精确locale字典改动只重跑针对性预览2.796s及构建；i18n的该正则没有匹配测试，因此该次i18n命令只编译，整包i18n0.530s通过记录来自上一完整回归。代码没有更改MyGo框架，无全框架测试通过声明。本轮遇到并修正的其他失败为新旧helper同名、测试开错固定disclosure key、单元RPC测试取错payload层、重复frame state key，以及布局预设未注入已加载图片而进入测试失败状态；没有通过修改断言掩盖未实现的产品行为。

默认与版本化产物均86354944字节，最终SHA256为 `FA4D7A18860FD9C28A9164DF53A28E5A427644B0AC07555442F024FFE7C9BB11`，精确最终值见 [构建记录](<C:/AI/WhaleIsle-MyGo/docs/review-20261022/evidence/tool-image-pass266-build.json>)。构建使用Go1.27.2、CGO_ENABLED=0；无二进制提交。

整体仍待后续处理：更广的typed详情/Inspect入口、lazy参数、Inspector图结构与分页、字体和精确基线/选择/文本换行、viewport与memoization、真实剪贴板拒绝、滚动条/tooltip/完整presence动效、Host主页/进程滚轮/物理输入、其余媒体、shell/tabs/材质、完整启动关闭。图片模型及主卡片已接入，但原图预览完整动效/blur、父级与历史scope的图片授权缓存生命周期、远程格式与动图显示、全部图片入口复用、原始复制逐字节和多行选区仍需要相应后续工作，不能据本轮判定全产品对齐完成。

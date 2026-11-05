# 鲸屿本体 ↔ MyGo 设置与弹出菜单 全量行级对比

日期：2026-02-07（v3，全量逐行核对）
方法：vendor 全部 20+ 个 locales.ts 完整读取（文案=行清单）+ 全部 settings section tsx 提取 + MyGo `internal/ui/settings` 92 个非测试文件的 providerCopy 全文提取 + 关键 pane 逐行核对。
标记：**[✗]** 已确认差异（两侧源码均见）；**[快照]** vendor 快照所见、已装版可能不同；**[实机]** 需运行截图核对；**[MyGo+]** MyGo 专有增量（保留，不计差异）。

## 0. 总结论

差异是**真"大量"**，四类叠加：

1. **缺行**（2 处确认）：内置功能页「机器人（Bots）·测试中」字段已备未渲染；界面设置缺「会话日志导出」标题栏行。
2. **布局形态**（外观页为主）：玻璃区块、字体区块、透明主题/侧栏遮罩开关行、滑杆行全部与本体不同构；权限菜单条目多一倍高。
3. **文案漂移**（~20 处确认）：rail、账号页、鲸鱼娘页、外观页、关于页都有标题/描述/选项名不一致。
4. **页内排序**：外观页块序不同（玻璃/字体位置互换、字号行位置不同）；通用页行序不同。

对齐面：技能、MCP、模型、Agent 预设、市场、远程、shell/subagent/agent-loop/web-search 各页的结构与核心文案与 vendor 逐字一致或同构；特效弹窗（思考炫光/输入特效）为深度移植。

---

## 1. Rail（侧栏）

| 本体（installed 分组） | MyGo | 判定 |
|---|---|---|
| 账户：账号与余额、用量与费用 | 账户：账号、用量与费用 | **[✗]** 「账号」vs「账号与余额」（vendor `nav: '账号与余额'`） |
| 通用：通用设置、外观、界面设置、对话 | 同 | ✓ |
| 智能体：模型、Agent 预设、技能、MCP、内置插件、市场 | 同 | ✓ |
| 功能：内置功能、鲸鱼娘、手机远程 | 内置功能、鲸鱼娘、手机远程 | ✓（remote nav 文案见 §10） |
| 其他：关于 | 其他：关于 | ✓ |
| 搜索框 / 打开配置文件 / 关闭 | 有 / 有 / 有 | ✓ |

实现级分区（不在 rail 显示）：agent-loop、web-search、subagent、shell、other 五页两侧均有对应物（MyGo sections.go 有同 id 分区；vendor 为 SettingsForm 卡页）。

---

## 2. 账号与余额

| 本体 | MyGo | 判定 |
|---|---|---|
| nav 「账号与余额」 | 「账号」 | **[✗]** |
| 未登录卡标题「当前未登录 DeepSeek 账号」 | 「登录 Whale Isle」 | **[✗]** |
| 未登录卡描述「登录 DeepSeek 账号获取专属 API Key」 | 「登录后即可使用账户模型并查看余额与用量。」 | **[✗]** |
| 「充值余额」 | 「余额」 | **[✗]** |
| 「赠金余额」 | 「奖励余额」 | **[✗]** |
| 「查询用量」 | 「用量」 | **[✗]** |
| 「更多账号信息」 | 「账户信息 ↗」 | **[✗]** |
| 「充值」topUp、「登录后查看」balanceSignedOut、「前往开放平台查看」balanceUnavailable | 需逐行核 account.go 钱包区 | [实机] |
| 头像/名字/联系方式 + 登出菜单 | 同构（account_signout.go 有确认弹窗） | ✓ 结构 |
| 意见反馈/联系我们入口 | account.go 有联系入口 | [实机] |

## 3. 用量与费用

| 本体 | MyGo | 判定 |
|---|---|---|
| 「会话累计费用」行（vendor 文案逐字） | 同文案逐字 | ✓ |
| 「官方峰谷时」行（desc 逐字） | 同文案逐字 | ✓ |
| 计费设置（sessionCost.noPriceHint 所指「设置→消耗统计→计费设置」） | MyGo 有 pricingView（usage.go 内嵌价目编辑） | [实机] 形态待核 |
| 统计面板本体（轮次/token/会话明细） | usage.go 内嵌统计面板：标题+导出菜单(JSON/每日CSV/模型CSV)+刷新+返回+统计区+账号区+余额区+用量网页链接/复制/充值 | **[✗/实机]** MyGo 面板是本地自绘版，与本体 StatsSection 布局/文案需逐项对（导出菜单项名「导出每日 CSV」vs 本体未知） |

## 4. 通用设置

本体行序（vendor general.item 快照；installed 0.3.13 已把部分行移走，见 §6）：语言 / 网页链接默认打开方式 / 性能与用量 / 工作步骤展示 / 繁忙时的发送行为 / 自定义指令 / 权限 / 显示代码工作视图 / 当前版本 / 关闭窗口时 / 启动时 / Harness 自动恢复 / Session Log 上传 / 快捷键 / 字号大小。

MyGo 行序：语言 / 显示代码工作视图 / 快捷键 / 关闭窗口时 / 启动时 / 性能与用量 / Session Log 上传 / Harness 版本 / Harness 自动恢复 / [MyGo+]本地运行与诊断 disclosure / （未登录时）账号登录行。

| 项 | 判定 |
|---|---|
| 行集本身与 0.3.13 移页规则一致（链接/性能/步骤/发送行为/指令/权限/字号已移出） | ✓ |
| 行序与本体 General 页序不同（即使移页后剩余行相对顺序也不同） | **[✗/实机]** |
| 「Harness 版本：」为纯文本行 vs 本体「当前版本 {version}」行 | **[✗]** 形态 |
| 快捷键行（shortcuts.go，含搜索/编辑/重置全套） | ✓（本体 shortcuts 行打开同一套快捷键编辑器）[实机] |
| Harness 自动恢复（enable/maxAttempts/baseDelay 三控件） | ✓（harness_restart.go 同构） |
| [MyGo+] 「本地运行与诊断」disclosure | [MyGo+] |

## 5. 外观 ⭐ 差异最大页

### 5.1 页内块序

| 本体（AppearanceSection.tsx 实证） | MyGo（appearance.go + wallpaper_background.go 实证） | 判定 |
|---|---|---|
| ①色制 ②主题 ③背景图 ④背景特效 ⑤指针特效 ⑥按钮悬停光泽 ⑦玻璃透明度 ⑧字体 ⑨appearance.item 槽（输入特效等） | ①颜色模式 ②主题库 ③[背景图+背景特效+指针特效+悬停光泽+玻璃块合一] ④字体 ⑤字号大小 ⑥思考炫光 ⑦输入特效 | **[✗]** 玻璃块被并入③内、字体提前、多一个「字号大小」独立行（本体 fontSize 在通用页/快照） |

### 5.2 玻璃透明度区块逐控件

| 本体 | MyGo | 判定 |
|---|---|---|
| 标题「玻璃透明度」+ `80%` 值同行 | 标题「玻璃与透明度」为区块头；滑杆行内嵌值 `80`（无 %、无度） | **[✗]** |
| 整段说明「控制界面表层的实心程度…」 | 无说明 | **[✗]** |
| 整宽滑杆 | bodyPreferenceRow 半宽 StepSlider | **[✗]** |
| ghost「重置」 | 整宽「恢复默认玻璃透明度」按钮 | **[✗]** |
| 毛玻璃/像素化滑杆在「背景图」行内 | 在玻璃区块内（且仅设壁纸时出现） | **[✗]** 位置+条件 |
| 终端透明度 `75%` 行内值 + 三段动态说明（needsBackdrop/deepHint/hint） | 紧凑滑杆、无 %、无任何说明 | **[✗]** |
| 透明主题：**开关居左**+文字；hint 三段动态（needsWallpaper/blurHint/hint） | 文字居左**开关居右**；hint 三段（近似但文案不同：「有壁纸时使表层完全透明…」） | **[✗]** 形态+文案 |
| 隐藏侧栏遮罩：开关居左 + 说明「…只保留中间的分割线」 | 开关居右、无说明 | **[✗]** |
| 玻璃滑杆在「透明主题+有壁纸」时禁用 | MyGo 同样禁用 ✓ | ✓ 行为 |

### 5.3 字体区块逐控件

| 本体 | MyGo | 判定 |
|---|---|---|
| 「字体」标题 + ghost「重置」 | themeTypographyPane（布局待逐控件核） | [实机] |
| 界面字体 Input+hint；字号滑杆+`14px`；等宽字体 Input+hint；代码字号滑杆+`12px` | 四字段存在但布局/单位呈现不同（MyGo 字号另有独立「字号大小」行） | **[✗]** |
| ghost「高级」按钮 → 展开输入框字体+终端字体 | 折叠面板或直接平铺 | **[✗]** |
| 各字段 hint 全文（type.interfaceHint/codeHint/composerHint/terminalHint） | 部分缺 | [实机] |

### 5.4 其余行

| 项 | 判定 |
|---|---|
| 色制磁贴（浅/深/跟随系统） | ✓ 同构（MyGo 标题文案「颜色模式」vs 本体「色制」**[✗]** 文案小差；desc「选择浅色、深色或跟随系统。」vs「选择现在使用浅色半、深色半，还是跟随系统。」**[✗]**） |
| 主题库（创建/导入/导出/复制/删除/编辑/高级 token/当前模式） | themeLibraryPane 存在 [实机] |
| 背景图（选择图片/清除/浏览图库/图源/收藏/裁切/玻璃联动提示） | wallpaperImagePreference 存在；毛玻璃/像素化位置不同（见 5.2） |
| 背景特效（流动渐变开关+特效设置弹窗：方案/光斑形态/配色/速度/数量） | backgroundEffectPreference 存在 [实机] 弹窗字段集待核 |
| 指针特效（弹窗：特效类型/方案/预览/颜色/速度/大小） | pointerEffectsPane 存在 [实机] |
| 按钮悬停光泽（desc「鼠标悬停在按钮上时叠加一层金属扫光…」） | MyGo desc 为「减少动态效果时显示静态光泽。」**[✗]** 文案语义不同 |
| 字号大小行（fontSize.title，px 步进器） | MyGo 有独立行（bodyFontSizePreference）；本体快照在通用页、0.3.13 或已移外观 | [快照] |
| 输入特效行 + 弹窗 | ✓ 深度移植（effect/cursor/colors/speed/blink/预设/导入导出全有） |
| 思考炫光行 + 弹窗 | ✓ 深度移植（mode/easing/palette/呼吸/色相循环/夜间/预设/导入导出全有）；本体在 interface.item、MyGo 在外观（0.3.13 移页 ✓） |

## 6. 界面设置

MyGo 行序（interface.go）：标题栏 Git 操作 / 终端抽屉开关 / 右侧栏按钮 / 拖动调整输入框大小 / 会话统计 / 对话与轨迹页签 / 显示底部归档管理区 / [MyGo+]本地偏好 disclosure。

| 项 | 判定 |
|---|---|
| 六行标题+desc 与 vendor 逐字一致（ui-git.titlebarGit、ui-titlebar.terminalToggle/surfacesToggle、ui-conversation.resize/statsLine/viewTabs） | ✓ |
| **缺行：会话日志导出**（session-log-export `settings.titlebarAction`——vendor interface.item 注册；MyGo 只有会话菜单「下载会话日志」动作） | **[✗]** |
| 已按 0.3.13 移走的行（sessionCost/peakValley→用量、beam→外观、dshbot/remoteWorkspace→内置功能、composerBeam→外观） | ✓（MyGo 注释与移动方向一致） |
| [MyGo+] 显示底部归档管理区、本地偏好 disclosure、viewTabs 类型异常 fallback 提示 | [MyGo+] |
| 行序与本体 interface.item 快照序不同 | [快照] |

## 7. 对话

MyGo 行序：权限 → 繁忙时的发送行为 → 工作步骤展示 → 工作步骤收起时机 → 网页链接默认打开方式 → 下一句提示 → 视觉回复 → 自定义指令。

| 项 | 判定 |
|---|---|
| 行集与 0.3.13 移页清单一致 | ✓ |
| 权限行：本体=PreferenceRow+Select（仅可查看/工作区内修改/完全权限）+FullAccess 确认弹窗（title/desc/acknowledge/enable） | MyGo 同构且有确认弹窗；**确认弹窗文案不同**：本体「启用完全权限后，新会话将减少确认步骤…仅建议在你信任后续任务时使用」/「我已了解风险，并愿意继续」vs MyGo「新任务将可以访问工作区之外的文件并执行命令。」/「我了解这些权限的影响」**[✗]** |
| busyEnter/transcriptView/linkOpening 标题+desc 逐字一致 | ✓ |
| 自定义指令（标题+scope 说明+textarea+计数器） | MyGo 有；desc「给智能体定几条规则，后续对所有任务都生效」逐字一致 ✓；本体行另有 placeholder「输入规则，例如：回复始终使用中文」[实机] |
| collapseTiming/nextPrompt/visualReplies 三行 vendor 快照无 → 0.3.13 新行，MyGo 已按新行实现 | ✓（文案为 MyGo 自拟，[实机]核） |
| 行序（权限在首位）vs 本体待核 | [实机] |

## 8. 内置功能

| 本体（installed） | MyGo | 判定 |
|---|---|---|
| 机器人（Bots）·测试中（dshbot.title/badge/description/restartHint 文案全在 locales） | **缺行**——`DshbotEnabled`/`OnSetDshbot` 已声明在 access 但 featuresPane 只渲 3 行 | **[✗]** |
| 鲸鱼娘助理（pet.whaleAssistant 文案复用） | ✓ 同文案 | ✓ |
| Project（含帮助气泡） | ✓（MyGo 帮助文案自拟） | ✓/[实机] |
| 远程工作区（SSH）（title/desc/restartHint 逐字一致） | ✓ | ✓ |
| 行序（Bots 相对位置） | [实机] |

## 9. 鲸鱼娘

结构 ✓（nav 标题+intro「你的常驻助理与桌面伙伴…」+双 tab+分组），行位置：本体 desktop tab = 外观组（在桌面显示鲸鱼娘→体型与不透明度→性格）→行为组（活跃频率→自主行为→拖拽方式→省电模式→点击音效）→联动组（快捷对话→看看模型）→恢复默认；MyGo 同序同组。

| 本体 | MyGo | 判定 |
|---|---|---|
| 「软萌」（natural） | 「自然」 | **[✗]** |
| 「元气」（genki） | 「元气」 | ✓ |
| 「傲娇」「毒舌」 | 同 | ✓ |
| 「活泼」（active） | 「活跃」 | **[✗]** |
| 「自由拖拽」 | 「自由拖动」 | **[✗]** |
| 「按住 Shift 拖拽」 | 「Shift 拖动」 | **[✗]** |
| 「锁定位置」 | 同 | ✓ |
| 拖拽 desc「锁定后不能拖动她；Shift 模式需按住 Shift 拖。」 | 「自由拖动、按住 Shift，或锁定位置。」（仅复述选项，无语义） | **[✗]** |
| enabledDesc「关闭后隐藏桌面上的她，随时可以再开。」 | 「关闭后隐藏桌面形象，随时可以再开。」 | **[✗]** 小 |
| personalityDesc「影响她的台词与「聊聊」语气；助理人格跟随同一设置。」 | 「影响台词与「聊聊」语气，助理人格跟随同一设置。」 | **[✗]** 小 |
| chatDesc「右键面板的「聊聊」对话卡。」 | 「右键面板中的「聊聊」对话卡。」 | **[✗]** 小 |
| powerSaveDesc「无交互约 5 分钟后降低帧率。」 | 「无交互约五分钟后降低帧率。」 | **[✗]** 小 |
| 「恢复桌面形象与行为默认」 | 「恢复外观与行为默认」 | **[✗]** |
| pet.enabled 在外观组首行 | 同位置 ✓ | ✓ |
| 体型与不透明度（合并行，内嵌体型/不透明度双 Select） | 同构 ✓ | ✓ |
| 自主行为行内嵌自言自语/自主乱逛双 Switch | 同构 ✓ | ✓ |
| 看看模型（pet_look.go 存在） | ✓ [实机] |
| intro「你的常驻助理与桌面伙伴。桌面和 IM 共用她的同一条对话。」 | petHeader 同文案 | ✓ |
| [MyGo+] 本地皮肤与动画 disclosure（THA4/皮肤/导入） | [MyGo+] |

## 10. 关于

| 本体 | MyGo | 判定 |
|---|---|---|
| intro「本桌面应用与官方 Harness。」 | 同 | ✓ |
| App 卡「Whale Isle / Based on DeepSeek Harness」 | 同 | ✓ |
| 「当前版本 {version}」+「检测更新」「在线安装」按钮 | 版本 chip + 「下载更新/继续下载/安装并重启/打开更新中心」按阶段 | **[✗]** 文案集不同（MyGo 阶段词：检测更新/下载更新/继续下载/安装并重启/打开更新中心；本体：检测更新/在线安装+update.badge/立即更新/稍后/重试） |
| 「运行目录 {path}」+「打开运行目录」 | 「运行目录 {path}」+（打开动作） | ✓ [实机] |
| 「凭据存储：系统钥匙串加密（safeStorage）/明文回退…」 | 「凭据存储：系统钥匙串加密（DPAPI）/明文文件（不安全）/尚未创建…」+警告文案集 | **[✗]** Windows 实现差异（合理，DPAPI vs safeStorage；文案集不同） |
| 「GitHub 发行版」 | 「GitHub 发行版」链接（指向 MyGo 仓库） | ✓/合理差异 |
| 「官方 Harness」分区（about.harness） | 同文案链接 | ✓ |
| 更新状态文案（updateChecking/updateNone/updateCurrent/updateAvailable/updateDownloading/updateInstalling/updateOpenedPage/updateManualInstall/updateError） | MyGo 阶段文案集不同（「当前已是最新版本」vs「已是最新版本 {latest}」等） | **[✗]** 文案 |
| [MyGo+] 应用数据与诊断 disclosure（标识符/userData/vendor/Analytics/诊断） | [MyGo+] |

## 11. 模型

| 本体 | MyGo | 判定 |
|---|---|---|
| 标题「模型」+ intro「填入各提供商的 API 密钥即可使用其模型。」 | 逐字一致 | ✓ |
| 提供商卡（名称+「自定义」tag+「已登录」pill+编辑/删除） | 同构 | ✓ |
| 添加模型提供商（addCatalog「第三方模型提供商」/addCustom「自定义模型 API」+两条 hint） | add_provider_editor.go 存在 | [实机] |
| 编辑器字段：API 密钥（placeholder/keyStored/keyEnvLocked/keyBlank/keyBlankNew/keyIllegalCharacters）、API 地址、模型目录（inherited/customized/恢复默认模型）、模型行（ID/显示名/上下文窗口/最大输出/effort 勾选/输入类型/添加/删除）、模型选项、高级 hint、冲突提示、只读提示 | model_*.go 全套存在（model_cards/body/catalog/configuration/controls/create/directory/discovery/providers/base_url） | ✓ 结构/[实机] 文案逐行核 |
| 识图模型（visionModel/hint/off） | vision.go 存在 | ✓ |
| 获取可用模型弹窗（fetch*） | model_discovery.go 存在 | ✓ 结构 |
| DeepSeek 账号卡特例 | ✓（account_models.go） |
| onboarding/welcome 弹窗（welcomeTitle/Body/onboarding*） | onboarding.go 存在 | [实机] |
| effort 名称集（Off/Minimal/Low/Medium/High/Extra High/Max，中英同文） | effortNames 一致 | ✓ |

## 12. Agent 预设

| 本体 | MyGo | 判定 |
|---|---|---|
| sectionIntro「选择 Agent 的工具和工作方式…」 | agent_preset_copy.go 同文案 | ✓ |
| 内置/自定义 分组 | ✓ | ✓ |
| 预设名+desc（标准模式/PTC 模式/极简模式/创造模式，逐字） | agent_preset_copy.go 逐字 | ✓ |
| 卡片：设为新任务默认/查看配置/新任务默认徽标/加载失败徽标 | agent_presets.go + reader 同构（MyGo 按钮文案「选择默认 · name」vs 本体「设为新任务默认」**[✗]** 小差） | **[✗]** 小 |
| 模式指南弹窗（PresetGuideDialog，guide-locales 全套） | agent_preset_guide.go 存在 | [实机] |
| 创造模式入口（creatorDraft「让 Agent 帮我创建预设模式」/createPlugin「让 Agent 创建插件」+描述） | creator_task.go 存在 | [实机] |
| switchRefused/standardUnavailable 提示 | 有对应 | [实机] |

## 13. 技能

标题「技能」+ intro 逐字一致；浏览 GitHub 技能/添加技能/刷新 三按钮 ✓；scopeNotice（projectCatalogUnavailable/sessionCatalogUnavailable）、搜索框 placeholder 逐字一致、来源筛选（全部/个人/当前项目/捆绑/其他）、结果计数、清除筛选、分组折叠+组级模型调用开关、行内（打开目录/模型调用/用户调用/删除）、编辑器（保存位置/名称/描述/何时使用/分组/正文/调用方式）、删除确认、GitHub 技能库弹窗（搜索/范围/安装/冲突/确认）全部存在。

| 差异 | 判定 |
|---|---|
| Hub 弹窗标题：本体「GitHub 技能库」vs MyGo「搜索技能」 | **[✗]** |
| Hub intro/搜索 placeholder/安装流其余文案 | MyGo 部分自拟（「搜索公开仓库中的技能。」）**[✗]** |
| resultCount 文案「工作区与个人技能 {count} 项」 | [实机] |
| 编辑器校验错误文案集 | [实机] |

## 14. MCP

标题「MCP 服务器」+ intro 逐字一致；添加服务器/刷新 icon 按钮、搜索、启用状态筛选、已配置分组+计数、「组合内 MCP」组+note、行（来源 pill 已配置/组成配置、测试、开关、登录、编辑、删除、只读）、stdio/HTTP 徽标、健康/阶段状态词（未挂载/等待依赖/加载中/已连接/失败/卸载中/连接中/重连中/连接失败）、工具数、测试成功/失败、编辑器（表单/JSON 双模式、基本信息 ID+名称+类型+超时、连接配置 stdio 命令/参数/工作目录/环境变量 或 http URL/请求头、stdio 风险提示）、删除确认、工具清单弹窗、导入——结构全套存在。

| 差异 | 判定 |
|---|---|
| 「添加服务器」按钮文案：本体「添加服务器」vs MyGo「Add MCP server/添加服务器」（mcpCatalogText 混源） | [实机] |
| 无 listRPC 时 MyGo 显示 stub 提示（KeySettingsMCPStubHint/SchemaHint） | [MyGo+] |
| 编辑器逐字段校验文案 | [实机] |

## 15. 内置插件 / 市场

| 本体 | MyGo | 判定 |
|---|---|---|
| 内置插件：nav/title「内置插件」+ intro「查看内置部署的插件列表」+ tab 容器（快照内仅 plugin-inventory 的「插件列表」tab → 单 tab 不显示 tab 栏） | MyGo：标题+KeyPass4ABuiltinHint+pluginSettingsTabs+inventory+「管理插件包」「打开市场」按钮 | **[✗]** 多了按钮+hint、intro 文案待核 |
| 插件列表 tab（plugin-inventory：会话插件/全局插件 分组+preset 切换器+启用于/去预设分组查看/状态 tag 全套） | plugin_inventory.go 存在 | [实机] |
| 市场：nav「市场」、标题「插件市场」+ intro、4 tab（发现/已安装/收藏/操作记录）、排序/时间范围/分类/搜索/已安装筛选、更新检查、详情弹窗（截图/README/兼容性/依赖）、安装/更新/卸载确认（警告文案）、批量更新、allowBuilds 授权、进度/重启状态、导出脱敏日志 | MyGo 市场为 overlay/contributed 面（OnOpenMarket），内容深度未逐项核 | [实机] 大项 |

## 16. 手机远程

| 本体 | MyGo | 判定 |
|---|---|---|
| nav「远程」（vendor）/ rail 显示「手机远程」（installed 0.3.13） | 「手机远程」 | ✓ |
| 页：标题「远程」+ intro + tab 容器（网关/消息渠道） | MyGo 为 contributed/overlay 页（OnOpenRemote） | [实机] |
| 网关 tab 行集：中继主机（relayUrl+hint+placeholder+save+恢复默认）、（已退役）宿主令牌、连接方式（局域网/服务器+modeHint/中继缺失提示）、监听端口、监听范围（整个局域网/仅本机+hint+已失效）、传输加密（明文/TLS+lanTlsHint/lanPlaintextWarning）、刷新配对码（rotateToken+hint+确认） | 未逐项核 | [实机] 大项 |
| 配对面板（开启/关闭、二维码、复制链接、已配对设备 N、设备管理弹窗：重命名/解除配对/在线/最近访问/配对于） | 未逐项核 | [实机] |

## 17. 实现级分区（shell/subagent/agent-loop/web-search）

| 分区 | 本体行集 | MyGo | 判定 |
|---|---|---|---|
| Shell（终端） | 命令超时(毫秒)、单流输出上限(字节)、已覆盖/恢复默认/保存 | sections.go 有 shell 分区 | [实机] |
| Subagent（子智能体） | 运行限制：最大递归深度(+3 段 help)、并行数量上限(+help)；模型选择：允许 Agent 为子智能体选择模型、可选模型目录（含 unavailable/partial/empty/conflict/off 提示） | served_subagent_models.go 存在 | [实机] |
| Agent loop | 并行工具调用数 | sections.go 有 agent-loop | [实机] |
| Web search（网页搜索） | API Key(+apiKeySet/apiKeyUnset)、接口地址、单次请求最多搜索次数 | sections.go 有 web-search | [实机] |
| 表单框架词（已覆盖/恢复默认/本部署只读/未加载/保存/保存中/校验错） | 逐字 | bodyHostPreference 框架 | ✓ 大概率逐字 |

## 18. Composer 弹出菜单

### 18.1 权限菜单（当前会话）
| 本体（PermissionSelect.tsx 实证） | MyGo | 判定 |
|---|---|---|
| 条目 = 图标 + 标签（+`<sup>`EXP 角标，仅 Auto review） | 条目 = 图标 + 标签 + **描述段** + 勾选列 | **[✗]** 行高/布局 |
| 选项集：仅可查看/工作区内修改/完全权限/Auto review EXP | 同集 | ✓ |
| 描述只作 trigger tooltip | MyGo 描述显示在条目内 | **[✗]** |
| 触发器 = 图标 + 标签 + EXP + chevron；aria「访问模式，当前：{name}」 | 同构 | ✓ |
| FullAccess/AutoReview 确认弹窗（RiskConfirmation：title/desc/acknowledge/cancel/enable） | permission_risk.go 有确认 | ✓ 结构/[实机] 文案 |

### 18.2 + 菜单（添加/命令面板）
| 本体 | MyGo | 判定 |
|---|---|---|
| 面板分两段：「添加」（附件动作）+「指令」（slash 命令：goal/plan/feedback/compact/permission/export + 技能） | 平铺工具项：添加文件/粘贴图片/引用文件夹/录音或语音设置/转写语音文件/切换语音服务/跟随发送设置/忙碌时排队/忙碌时插话/技能 | **[✗]** 结构（无分组标题）+条目集 |
| 附件项（vendor section.add 内为文件类动作；具体项集 [实机]） | 3 项附件 | [实机] |
| 指令段含命令名+描述（description.goal/plan/…） | MyGo 无 slash 命令项（指令是否另在输入 `/` 面板？）[实机] | **[✗/实机]** |
| 忙碌模式三项（跟随发送设置/排队/插话）、语音三项（录音/转写/切换服务）、技能开关 | MyGo 专有 | [MyGo+] |
| 搜索框「搜索…」 | 有查询过滤 | ✓ |

### 18.3 模型选择
| 本体 | MyGo | 判定 |
|---|---|---|
| 两级 root→model/effort、分组（DeepSeek 账号/各供应商）、>4 项出搜索「搜索模型…」、menu.aria、empty.models/efforts、effort.providerDefault「Default」、触发 `name · effort` 或回退「请选择模型」 | 同构两级面板；各文案基本逐字 | ✓ 结构/[实机] 像素级 |
| provider 组加载失败行 warning.groupLoad | 有对应 | [实机] |

### 18.4 其余
| 菜单 | 判定 |
|---|---|
| Agent 预设选择器（seatHint「选择新任务使用的 Agent 预设」） | preset_picker 存在 [实机] |
| 计划模式（plan_mode_control） | 存在 [实机] |
| 语音（录音/波形/转写/服务切换） | [MyGo+] 本体无此功能面 |
| 排队/插话发送按钮副动作（input.send.queue/steer） | [实机] |

## 19. 差异汇总（按修的成本排序）

| # | 差异 | 类别 | 规模 |
|---|---|---|---|
| 1 | 内置功能缺「机器人（Bots）·测试中」行 | 缺行 | 小（字段已备） |
| 2 | 界面设置缺「会话日志导出」行 | 缺行 | 小 |
| 3 | rail「账号」→「账号与余额」 | 文案 | 1 行 |
| 4 | 账号页 6 处文案（未登录卡×2、充值余额、赠金余额、查询用量、更多账号信息） | 文案 | 小 |
| 5 | 鲸鱼娘页 12 处文案（软萌/活泼/自由拖拽/按住Shift拖拽/拖拽desc/enabledDesc/personalityDesc/chatDesc/powerSaveDesc/恢复默认） | 文案 | 小 |
| 6 | 外观玻璃区块整段重排（标题改名+%值+全宽滑杆+说明段+ghost重置+开关居左+侧栏遮罩说明+终端透明度说明+毛玻璃/像素化移回背景图行） | 布局+文案 | 中 |
| 7 | 外观字体区块（ghost重置/高级按钮/字段hint/px单位） | 布局 | 中 |
| 8 | 外观页块序（玻璃/字体归位、字号行位置）+「色制」标题文案 | 排序+文案 | 中 |
| 9 | 权限菜单条目去描述段、对齐行样式 | 布局 | 小 |
| 10 | + 菜单分组（添加/指令两段）与条目集对齐 | 布局+条目 | 中（先实机核本体+项集） |
| 11 | 权限确认弹窗文案（默认权限页） | 文案 | 小 |
| 12 | 关于页更新状态文案集对齐 | 文案 | 小 |
| 13 | 技能 Hub 弹窗标题/intro 对齐 | 文案 | 小 |
| 14 | 「悬停光泽」desc 语义对齐 | 文案 | 1 行 |
| 15 | 内置插件页按钮/hint 收敛 | 布局 | 小 |
| 16 | 通用页行序 + 「Harness 版本」行形态 | 排序+形态 | 中 |
| 17 | 市场/远程/用量三大页内容深度核对后逐项对齐 | 大项 | 大（需实机取证） |
| 18 | 模型/Agent 预设/编辑器内部字段级文案核对 | 逐项 | 中（多已逐字，剩细节） |

## 20. 遗留边界

- vendor 快照 < installed 0.3.13：凡标 [快照]/[实机] 的项以实机为准——尤其 ①对话页行序 ②内置功能行序与 Bots 位置 ③市场/远程/用量页 ④+ 菜单「添加」段项集 ⑤各编辑器内部字段文案。
- [MyGo+] 增量一律保留：命名空间浏览、本地偏好、本地皮肤、语音、诊断 disclosure、归档管理区。
- 已排除误报：特效弹窗（炫光/输入特效）为全量移植非差异；界面设置 6 行/用量 2 行/技能 MCP 模型页核心文案已逐字一致。

## 21. 修复状态（2026-02-07 实施）

三条并行切片 + 收尾，全部落在 dirty worktree；`go build ./internal/ui/...`、`go test ./internal/ui/{settings,composer,i18n}`、`go vet` 全绿，`cmd/whaleisle` 整包可编译。

| §19 # | 状态 | 说明 |
|---|---|---|
| 1 Bots 行 | ✅ 已修 | `features.go` 新增 `features-dshbot` 行（badge 测试中+desc+restartHint），接 `InterfaceAccess.DshbotEnabled/OnSetDshbot`（接线原已存在 `settings_body_wiring.go`），位于远程工作区行之前（vendor order 80<81） |
| 2 会话日志导出行 | ✅ 已修 | `interface.go` host 偏好列表首位（vendor order 10），ns=`session-log-download` key=`titlebarAction`，vendor 文案逐字 |
| 3 rail 账号与余额 | ✅ 已修 | i18n.go；en 实为 'Account'（非 'Account & balance'）未动 |
| 4 账号页文案 | ✅ 已修 | 未登录卡标题/描述、充值余额、赠金余额、查询用量、更多账号信息（保留 ↗ 内联符）；另顺手对齐 `balanceUnavailable`→「前往开放平台查看 ↗」、sign-in desc |
| 5 鲸鱼娘页文案 | ✅ 已修 | 12 处全改 vendor pet.* 逐字；`pet_reset.go` 确认弹窗标题同步 |
| 6 玻璃区块 | ✅ 已修 | `wallpaper_background.go` 重写为块状：标题+`80%`、说明段、全宽滑杆、ghost 重置、终端透明度子块（三态 hint）、透明主题/侧栏遮罩开关居左+动态 hint |
| 7 字体区块 | ✅ 已修 | `theme_font_preferences.go` 重写：ghost 重置（ResetAppearanceFonts 六字段）、字号 `px` 值、字段 hint 逐字、ghost「高级」展开输入框/终端字体 |
| 8 块序+标题 | ⚠️ 部分 | 毛玻璃/像素化已移回背景图行内（`wallpaper_modals.go`）；**标题回退为「颜色模式」**——round164 实机 DOM 证实安装版即「颜色模式」+「选择浅色、深色或跟随系统。」，vendor 快照的「色制」为旧版 |
| 9 权限菜单 | ✅ 已修 | `permission_picker.go` 条目去描述段（保留 tooltip）；勾选列保留——vendor `Menu` selection='check' 本就渲染 ✓ |
| 10 + 菜单分组 | ✅ 已修 | `suggestItem.section` 字段；`/file /goal /plan /feedback`+附件/语音 intake→「添加」，其余命令+技能+模式项→「指令」；`paintSuggestPopup` 渲染非交互分段标题；项序按组稳定排序（添加组先于指令组，标题各现一次） |
| 11 权限确认弹窗 | ✅ 已修 | `permission_defaults.go` desc+复选框逐字 |
| 12 关于页文案 | ✅ 已修 | `about.go` current/available/downloading/installing/error 对齐 vendor；paused/verifying/ready 无对应保留 |
| 13 技能 Hub | ✅ 已修 | 标题「GitHub 技能库」、hubIntro、空态 hint、确认弹窗 body/noOverwrite 两行逐字 |
| 14 悬停光泽 desc | ✅ 已修 | vendor metallicPaint.description 逐字 |
| 15 插件页 | ✅ 无需改 | KeyPass4ABuiltinHint 已逐字 |
| 16 通用页 | ✅ 已修 | 保留「Harness 版本：{version}」——安装版 `ui-settings-general` 字典 `general.currentVersion` 证实即此文案（先改「当前版本：」已回退）；行序经安装版槽位注册核实一致 |
| 17 市场/远程/用量大页 | ⬜ 未动 | 需实机取证 |
| 18 编辑器字段级 | ⬜ 未动 | 需实机取证 |

**顺手修的小漂移**（不在 §19）：`agent_presets.go`「选择默认」→「设为新任务默认」；`native_commands.go`/`composer.go` 内置 slash 命令面（目标/计划/反馈/压缩/权限/下载日志 + vendor desc，按 DefinitionId 匹配）。

**未验证面**：以上均为源码+测试层验证（`go build/test/vet` 绿）；实机截图比对未做。§19 #16 行序、#17、#18 维持 `[实机]` 待取证。

## 22. 实机取证（安装版 0.3.13，Whale Isle.exe PID 21700）

取证方式（不重启、不注入 CDP）：从运行中进程解析 `resources\vendor` 真实路径，解包 56 个包的 `locale/{zh,en}.json` 字典 + 扫描全部 `client.js` 的 `settings.*.item` 槽位注册（id+order）。字典与注册表即安装版权威行清单，优先于 vendor 源码快照。UI Automation 无障碍树仅得 11 个顶层元素，不足以取行级证据；未截图。

### 22.1 安装版槽位注册（行序权威表）

| 分区 | 行（order 排序） |
|---|---|
| general | language(0) → interface(6) → developer-tools(15) → shortcuts(16) → close-behavior(25) → auto-start-desktop(26) → performance-usage(30) → session-log-upload-toast(90) → current-version+harness-restart(100) |
| interface | conversation(7) → titlebar-git(20) → terminal-toggle(30) → surfaces-toggle(40) → composer-resize(60) → stats-line(70) → view-tabs(80) → show-archived-list(90) |
| conversation | permission(0) → composer-enter(10) → transcript-view(20) → link-opening(30) → features(30) → next-prompt(40) → visual-replies(50) → custom-instructions(60) |
| features | close-behavior(25) → whale-assistant(78) → project(79) → remote-workspace(81) |
| appearance | font-size×3(5) → composer-typing-fx(10) → composer-beam(20) |
| usage | session-cost(10) → official-peak-valley(20) |
| pet | whale(0) → about(90) |

MyGo 各页行序与上表逐行一致（general/conversation/interface/features/appearance/usage 全部核实）。

### 22.2 安装版字典核实结果

- **✅ 全部已修文案与安装版逐字一致**：颜色模式、账号与余额、账号页全部、鲸鱼娘全部、玻璃区块、字体区块、添加/指令分段、slash 命令面、权限确认弹窗、GitHub 技能库、设为新任务默认、显示代码工作视图、繁忙时的发送行为。
- **✅ 「Harness 版本：{version}」**：安装版 `general.currentVersion` 原文，先改的「当前版本：」已回退。
- **⚠️ Bots 行安装版不存在**：全部安装版 client.js/字典 grep `dshbot`/`机器人` 零命中——Bots 是 vendor 快照（更新于 0.3.13）的功能。MyGo 保留该行：MyGo 已有 `config.DshbotEnabled`+助理面板实装，属 MyGo+/新版对齐，非安装版缺陷。
- **⚠️ 会话日志导出行安装版不存在**：`session-log-download` 全安装版零命中。MyGo 该行为 Host 描述门控，无下发不渲染，保留无害（前瞻兼容）。
- **✅ 字号三行**：安装版 appearance.item order 5 注册 字号大小/代码字号/终端字号三行**恒显**；MyGo 原将代码/终端字号藏在展开区——已修复（`theme_font_preferences.go` 三行恒显，字体族行留展开区）。
- **安装版 pet 页**：双标签「聊天与能力 / 桌面形象与行为」、体型与不透明度合并行、自主行为组（自言自语/自主乱逛开关）、点击音效——MyGo 全部已有且文案逐字。

### 22.3 范围裁定与仍待取证

**范围裁定（用户决定，移出对齐范围）**：
- **市场页**：本体由插件实现（`ui-plugin-market`），MyGo 不自行实现——不对齐。
- **手机远程页**：隔离，不进行对齐开发。

**已澄清（扫描误报）**：原先「三条跨区链接行」是扫描器把 `settings.section` 注册的 id/order 误读为页面内行——安装版不存在页内跳转链接行，该 id/order 是分区在 rail 的注册序号。

**安装版 rail 分组权威表**（`NAV_GROUPS` 硬编码，组内序按表而非注册 order）：

| 组 | 条目 | MyGo 判定 |
|---|---|---|
| account | account → usage-stats | ✅ 一致 |
| general | general → appearance → interface → conversation → archived-sessions（无包注册不渲染） | ✅ 一致 |
| agent | models → agent-presets → skills → mcp → plugins → market | ✅ 逐位一致 |
| features | features → pet → remote → remote-workspace | ✅ 已修：`OnOpenRemoteWorkspace` + rail 入口（ServerOutline）→ `SettingsPage="workspace"` 嵌入式远程页（宿主管线原已支持） |
| other | about | ✅ 一致 |

未列入 NAV_GROUPS 的插件分区按注册 order 追加进 features 组——MyGo contributed sections 同样进功能组，机制一致。

**仍待取证 → 已全部关闭**：
- **+ 菜单项集**：✅ 安装版 `ui-commands` 字典与 MyGo 已实现的 six 命令面（目标/计划/反馈/压缩/权限/下载日志+desc）+ 添加/指令分段逐字一致；intake 项为 composer 层功能非命令注册表项。关闭。
- **模型/Agent 预设字段级**：✅ `ui-settings-models` 字典仅 nav/addCustomUnavailable，模型页全部字段走 Host describe 下发——两侧同源，无静态文案可漂。Agent 预设已验证项（设为新任务默认）逐字。关闭。
### 22.4 块审修复（ui-conversation 面，2026-11-05）

**已修文案**（安装版逐字）：
- 附件/图片：拖入说明→`图片限制：最多 %d 张，每张 %s`；待上传→`待发送附件`；上传中→`上传中…`；图片三限→`一条消息最多添加 %d 张图片`/`单张图片不能超过 %s`/`图片总大小超过 %s，请移除部分图片`（`humanBytes` 人性化单位）
- 会话费用：`会话累计费用`→`当前会话费用`+连接符`·`→`：`；`没有设置价格`→`没有设置当前模型价格`；`未知路由`→`未知模型`；子代理/还有 `%d`→`%d 个`；计费按钮→`在 设置 → 消耗统计 → 计费设置 中设置模型价格`
- 峰谷：`高峰/空闲`→`高峰时段/空闲时段`；切换行→`%s · 距离切换剩余时间：%s`
- 队列 dock：图标按钮 aria/tooltip→`编辑排队消息`/`插话发送`/`删除排队消息`；计数→`{n} 条排队消息`
- hero 占位符全角逗号→安装版半角；自定义指令 textarea 补 placeholder `输入规则，例如：回复始终使用中文`

**22.4b 块审修复（titlebar/approval/message-edit/message-feedback）**：
- 标题栏：`切换右侧面板`→`切换右侧栏`（surfaces.toggle）；终端禁用提示→`终端抽屉不可用`（terminal.unavailable）
- 审批卡：`等待你的确认`→`等待审批`（approval.waiting）
- 消息编辑：`正在编辑已发送的消息`→`正在下方输入框中重新编辑`（editor.editing）
- 反馈：👍/👎 aria+tooltip→`好的回答`/`有问题的回答`、选中态→`取消标记`（重写 FeedbackButtonLabel 为安装版语义，原 ✓ 拼接废弃）；dialog hint 逐字（以帮助/包括）；分类`资源使用与成本`→`资源使用与费用`；`反馈提交失败`→`反馈保存失败`

**剩余缺口（这批四包）**：approval detail.aria/escalation 行、message-edit 的 running/unsupported/stale/busy 状态串（MyGo 编辑流无对应状态面）、feedback error.load。

**功能缺口**（安装版有、MyGo 无对应面，未建）：queue 行内编辑器（InlineEditor 保存/取消）、queue.edit.unsupported 提示、会话层级/更多会话操作 header、会话已被占用 error、parentOffline/workspace placeholder 变体、beam/typingFx 完整配置面板（~80 keys）、ask.* 交互细化、sessionCost.usage 峰谷明细行/priceTitle 行、terminal.expandRest、read.window/search.matches 截断行、todo `{done}/{total}` 分数格式、hero.claudeHeadline、command.attachmentsUnsupported、nextPrompt Tab 采纳提示。

**远程工作区（SSH）页**：✅ 已按 `dsh-remote` 插件源码（`plugin-src/`）全量对齐——五卡结构（当前机器卡：状态点+连接态+user@host:port+指纹态+远程目录/本地镜像 mono+取消当前/忘记指纹；SSH 机器卡：内联按钮行 设为当前/测试连接/编辑/删除、表单内嵌、空态 gating；sshconfig 导入卡：有 entry 才渲染；端口转发卡：needsMachine 门控+L/R mono 行+状态 tag+内联新增行；审计卡：查看/收起切换）。表单改安装版平铺序：名称/主机/端口/用户名→认证方式→密码（编辑时留空保持不变）/私钥路径/私钥口令→主机指纹校验三段（TOFU/严格/不校验）→默认远程目录→useAgent/keyboardInteractive checkbox+存入系统钥匙串 switch→跳板机五字段→测试连接/取消/保存右对齐。`ConnectionStatus` 补 host/username/port 字段。

## 22.5 残留目录全量扫尾（31 包）

方法：将 remaining-catalogs.txt 中 31 个安装版 zh 目录逐包与 internal/ui 全树做字面+前缀近似比对。结果：

- **实差已修**（verbatim 对齐）：ui-git 4 处（分支列表加载失败。/没有匹配的分支/搜索分支…/我已了解风险，并愿意继续）、ui-user-terminal 关闭终端、ui-files unsaved.title 有未保存的修改、preview.floating.noCwd 无法确定文件所属会话的工作区、model-selection search.empty 没有匹配的模型、open-in-app path.openError 打开失败，请重试、files search.pending 正在搜索文件…、visual-replies settings.description 搜索索引补全。
- **NO-MATCH 主体 = MyGo 更丰富面**：安装版插件多为简表面（如 ui-user-terminal 仅 20 keys、ui-preview 54 keys），MyGo 原生生面（重命名终端/shell 选择/分屏/批注截图/浏览器分区/发布向导/大文件扫描/PR 生成核对）无安装版对应文案，保留 MyGo 措辞。
- **结构性 gap（安装版有、MyGo 无对应面，记录不实现）**：ui-workflow-run 面板、git 菜单 detached/behind 禁用态、ui-visual-replies 卡片渲染面（MyGo 仅有设置项+OpenUI 隔离窗）。
- **已验证逐字**：agent-preset 名称/描述、goal dock、model-selection（此前审计）、deepseek-model-label（badge）。
- **范围外（按裁定）**：市场页（ui-plugin-market 插件自有）、手机远程（隔离）。

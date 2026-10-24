# Round168 · 内置插件/技能/MCP/Agent 预设 tab 与深色主题

本轮把设置剩余四个 tab 用原生管线逐一点开（含英文界面），并切到深色主题采集颜色 token。修了 4 处文案漂移（含英文语序），几何沿用 round160 的行/卡片原语；深色 token 与字体缩放按 equal/blocked 记录。

## 1. 方法

- 起服务：staged CLI，临时 `DSH_HOME`，端口 3099，token 从 stdout 取；`scripts/native-web-shot.mjs 1440 920 1.5 --eval`。
- 中文一轮：继续 → 设置与账号 → 设置 → 依次点「内置插件/技能/MCP/Agent 预设」，记录 labels/controls 的 rect + computed style。
- 英文一轮：通用设置 → 语言 → 选 English，再用英文导航名（Built-in plugins/Skills/MCP/Agent presets/About）逐 tab 采集；导航语言确实随语言切换（General/Appearance/Interface/Models/Built-in plugins/Skills/MCP/Agent presets/About）。
- 深色一轮：外观 → 深色 卡，然后读对话框/导航/行/胶囊/输入框/徽标的颜色。
- 收尾：harness 已停，临时 DSH_HOME 与探针文件已删；原生应用只读。

## 2. 各 tab 原生数字（1440×920，CSS px）

### 内置插件（Built-in plugins）

| 项 | 原生值 |
|---|---|
| 标题/简介 | 内置插件 18px/24（比其他 tab 的 16px 大）；查看内置部署的插件列表 13px muted |
| 搜索框 | 564×36 r12，底 rgba(255,255,255,.8)，占位 Search plugins / 搜索插件 |
| 分组头 | 会话插件（按钮 76×22）+ 右侧「标准模式（默认）」166×36 r12；副标题 由 Agent 预设按会话组成 12px |
| 插件卡 | 两列，各 275 宽，高度随描述 110（2 行）/92（1 行），列距 12；卡内名称 14/22 + code 12px + 描述 12/18 muted |
| 停用徽标 | 已停用 / Disabled，11px，底 rgb(245,246,247)，字 rgb(97,102,107) |
| 英文文案 | "Inspect the plugins this deployment ships." / "Session plugins" / "Standard mode (default)" / "Composed per session by agent presets" / "Global plugins" + "Shared by the system and every session" |

### 技能（Skills）

| 项 | 原生值 |
|---|---|
| 标题/简介 | 技能 16px/24；查找并管理用户、当前项目与捆绑技能。14px muted / "Find and manage user, current-project, and bundled skills." |
| 过滤 | 「全部 / All」chip，右对齐 |
| 分组行 | 工作区与个人技能 0 项（12px muted）；英文语序 **"0 workspace and personal skills"**（数字在前） |
| 空态 | 还没有技能。/ "No skills yet." 14px muted，单行 |

### MCP（MCP servers）

| 项 | 原生值 |
|---|---|
| 标题 | MCP 服务器 / "MCP servers" 16px/24，右上两个 28×28 图标按钮 |
| 简介 | 管理本机 MCP 服务器。受管条目写入 $DSH_HOME/mcp-servers.yaml；HTTP 服务器可在本页登录后在对话里使用工具。组成配置条目只读。14px muted，2 行 |
| 过滤/搜索 | 全部 82×36 r18 胶囊 + 434×24 搜索框 |
| 空态 | 还没有 MCP 服务器。添加一个服务器以连接本地工具或远程服务。 / "No MCP servers yet. Add one to connect local tools or a remote service." 14px muted，2 行 |

### Agent 预设（Agent presets）

| 项 | 原生值 |
|---|---|
| 标题/简介 | Agent 预设 18px/24；选择 Agent 的工具和工作方式。日常任务用「标准模式」，扩展 DSH 的能力用「创造模式」。13px muted |
| 分组 | 内置 / Built-in 12px muted；下方 自定义 / Custom |
| 预设卡 | 两列（x549 起，宽约 547），名称 15px + 徽标：内置 11px muted、**新任务默认 / "New task default"** 11px 白字深底 |
| 卡描述 | 13px muted，3 行（标准模式/创造模式/PTC 模式/极简模式各有原生文案） |
| 英文名称 | Standard mode / PTC mode / Creator mode / Minimal mode |

## 3. 深色主题 token（外观 → 深色 后实测）

| 元素 | 浅色 | 深色 |
|---|---|---|
| 对话框前景 | rgb(15,17,21) | rgb(249,250,251) |
| 导航选中底 | rgb(228,237,253) | rgb(53,54,56) |
| 导航选中字 | 蓝 srgb(0.186,0.324,0.615) | 蓝 srgb(0.561,0.685,0.992) ≈ rgb(143,175,253) |
| 行描述 | rgb(129,133,140) | rgb(173,178,184) |
| 胶囊（值/按钮）底 | rgb(245,246,247) | rgb(53,54,56) |
| 胶囊字 | rgb(15,17,21) | rgb(249,250,251) |
| 输入框底 | rgba(255,255,255,.8) | rgba(35,35,36,.8) |
| 输入框描边 | rgba(0,0,0,.12) | rgba(255,255,255,.2) |
| 停用徽标 | 底 #F5F6F7 / 字 #61666B | 底 rgb(53,54,56) / 字 rgb(207,211,214) |
| 主题预览卡 | 预览保持白/深色（预览内容本身） | 同左 |

MyGo 侧浅色已对齐（`Field #f5f6f7`、`Muted #81858c`、行 hairline、胶囊底/字）；深色由主题调色板派生（选中底=SurfaceHover、次要字=Muted、强调=Accent），本轮记录原生 token 作为后续校准基线，未改调色板。

## 4. 对照结果

| 表面 | 状态 | 说明 |
|---|---|---|
| 内置插件简介 | **fixed** | en `Built-in plugins deployed with this app.` → **`Inspect the plugins this deployment ships.`**（zh 原本一致） |
| 技能分组计数 | **fixed** | en 由 `Workspace and personal skills 2 skills` → **`2 workspace and personal skills`**（原生数字在前）；新增 `skillsGroupCountLabel`，自定义分组名保持原样 |
| MCP 空态第二行 | **fixed** | en `Add one to reach local tools or a remote service.` → **`Add one to connect local tools or a remote service.`**（zh 一致） |
| 预设「新任务默认」徽标 | **fixed** | `任务默认 / Task default` → **`新任务默认 / New task default`** |
| 内置插件标题/卡片/停用徽标 | equal | 18/24 标题、275 宽卡片、110/92 行高、11px 停用徽标与 MyGo 一致；卡片描述由 Host 清单下发 |
| 技能/MCP 标题、过滤、空态结构 | equal | 16/24 标题、右对齐过滤 chip、单/双行 muted 空态与 MyGo 同构 |
| Agent 预设名称/描述 | equal | Standard/PTC/Creator/Minimal 中英与 MyGo `agent_preset_copy.go` 逐字相同 |
| MyGo 独有项 | 保留 | Account/Market/Remote/Usage/Pet 条件入口、Whale 宠物页、插件“已停用”之外的 MyGo 本地动作均保留（产品面差异） |
| 深色 token | 记录（未改） | 原生值见 §3；MyGo 由主题派生，未在本轮改调色板 |
| 字体缩放 | blocked | 通用设置的字号步进需要输入框事件模拟；本轮未验证内容字号缩放对行高/字号的影响 |
| 账户 tab | blocked（沿用 round164） | 未登录 DSH_HOME 没有账户导航项 |

## 5. 修复与测试

- [catalog_pass4a.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/catalog_pass4a.go>)：内置插件简介英文、MCP 空态第二行英文。
- [catalog_pass4b.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/catalog_pass4b.go>)：预设默认徽标中英文。
- [skills.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/skills.go>)：新增 `skillsGroupCountLabel`，英文分组行改为「数字在前」。
- 测试：新增 [settings_native_tabs_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/settings_native_tabs_test.go>)（中英四个文案 + 分组计数语序）；[skills_surface_test.go](<C:/AI/WhaleIsle-MyGo/internal/ui/settings/skills_surface_test.go>) 的旧计数断言同步为原生语序。

门禁：`go test ./internal/ui/settings ./internal/ui/i18n ./internal/ui/frame -count=1` → settings 13.067s、i18n 0.502s、frame 14.932s，全部 ok；`go vet ./internal/ui/settings ./internal/ui/i18n` → 退出 0。未提交、未构建 exe；未改 cmd/runtime/vendor/third_party；harness 已停、临时目录已清。

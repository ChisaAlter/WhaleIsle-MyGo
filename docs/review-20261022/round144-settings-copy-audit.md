# Round144 · 设置页文案审计（General/Interface/Appearance/Models/Account/About + Pet）

本轮完成 round142 留下的设置页文案差集：把 `internal/ui/settings` 里的 623 处 `providerCopy("中文", "English")` 与原生字典逐条配对，修掉 46 处已确认的漂移（错译、英文措辞/大小写、中文用词），并把无法判定归属的候选逐条列出。

## 1. 方法

- 原生侧：重新以 UTF-8 解析 harness 61 个 client locale 文件（上一轮此前误用 ANSI 读取，中文列是乱码，本轮已重抽），得到 3884 条词条、1837 个 zh/en 键对；设置页对应 `ui-settings-general`、`ui-settings-models`、`ui-settings-account`、`ui-settings-shell`、`ui-theme`、`ui-permission-presets`、`ui-model-selection`、`ui-user-questions` 等包。
- MyGo 侧：扫描 `internal/ui/settings/*.go`（非测试）的 `providerCopy("zh","en")` 字面量，共 623 处、按文件归属 7 个页面；`internal/ui/settings` 内没有 `resumeText`。
- 配对：先按 zh 文本（去空白、忽略大小写）找原生键对，再按 en 反向找；两侧都相同记 matched。命中多个词典条目时优先采用与页面语境一致的包（如 Appearance→ui-theme、General/Pet→ui-settings-general、Models→ui-model-selection）。
- 判定：只修「同一语义」的确证漂移；上下文不同（同一词在不同页）、MyGo 自有文案、或原生没有对应键的，保留并在 §4 列出。

## 2. 计数

| 项 | 数量 |
|---|---|
| providerCopy 字面量站点 | 623 |
| 原生键对（UTF-8 重抽） | 1837 |
| 修复后逐字命中原生键对 | 143（修复前 94） |
| 原生无对应文案（MyGo 自有，未评级） | 460 |
| 确证漂移并修复 | 46 |
| 剩余候选（上下文不同/无对应键/组合后缀） | 33 行（按相关包过滤后 22 行） |

按页面（字面量站点 / 本轮修复）：

| 页面 | checked | fixed |
|---|---|---|
| General（settings/body_preferences/desktop/onboarding/permission_defaults 等） | 102 | 13 |
| Interface | 8 | 0 |
| Appearance（appearance/theme_font_preferences/wallpaper_background/effect_preferences/pointer_effects） | 155 | 16 |
| Models（models/model_*/provider_*/served_subagent_models/vision 等） | 130 | 2 |
| Account（account*） | 14 | 0 |
| About | 12 | 1 |
| Pet（单独页，pet.go/pet_look.go 等） | 64 | 11 |
| 其余（快捷键页等共用） | 138 | 3 |

Interface 与 Account 页未发现确证漂移：其文案与原生逐字一致，或属于 MyGo 自有面板说明（如账户通知反馈）。Usage 页不在 `internal/ui/settings` 中（见 §4）。

## 3. 修复清单（before → after，原生键）

General / About：

| 文件 | before | after | 原生键 |
|---|---|---|---|
| about.go | This desktop app and **the** official Harness. | This desktop app and official Harness. | about.intro |
| desktop.go | Show **the code work** view | Show coding view | developerTools.title |
| desktop.go | **Sending** while busy | Send behavior while busy | settings.enter.title |
| desktop.go | On the title-bar close button: minimize to the tray, or quit and stop the local Harness service. | The title-bar close button can hide the app in the tray, or quit and stop the local Harness service. | closeBehavior.description |
| desktop.go ×2 | **Closing** the window | When closing the window | closeBehavior.title |
| desktop.go | Open **the** launcher / Open the desktop | Open launcher first / Start desktop directly | autoStartDesktop.* |
| body_preferences.go | 完全访问 | 完全权限 | preset.fullAccess |
| body_preferences.go | 只读 / Read only | 仅可查看 / Read Only | preset.readOnly |
| body_preferences.go | 简洁 | 压缩 | label.compact |
| body_preferences.go | 引导运行 | 插话发送 | settings.enter.steer |
| onboarding.go | 正在**加载**设置… | 正在读取设置… | harnessRestart.loading |
| permission_defaults.go | 启用完全访问权限？ / Enable full access? | 确认启用完全权限？ / Enable Full access? | confirm.title |
| permission_defaults.go | 启用完全访问 / Enable full access | 启用完全权限 / Enable Full access | confirm.enable |

Appearance：

| 文件 | before | after | 原生键 |
|---|---|---|---|
| appearance.go | 色彩模式 | 色制 | scheme.title |
| effect_preferences.go | Thinking glow **while** sending | Thinking glow when sending | settings.beam.title |
| effect_preferences.go | 打字特效 | 输入特效 | settings.typingFx.* |
| effect_preferences.go | 保存为预设 | 保存预设 | *.panel.savePreset |
| effect_preferences.go | Reset to defaults | Reset | *.panel.reset |
| theme_font_preferences.go | 文字 | 前景 | editor.foreground |
| theme_font_preferences.go | 只影响会话内容的字号 | 仅影响会话内容的字号 | fontSize.description |
| theme_font_preferences.go | Code font size | Code size | type.sizeCode |
| theme_font_preferences.go | 字体与字号 | 字体 | type.title |
| pointer_effects.go | Pointer effects | Cursor effects | cursorFx.title |
| wallpaper_background.go | 收藏夹 | 收藏 | wallpaper.favorites |
| wallpaper_background.go | Background effects | Background effect | effect.title |
| wallpaper_background.go | 光团 / 射线 | 光球 / 光束 | effect.variant.orbs / .rays |
| wallpaper_background.go | Button hover sheen | Button sheen | metallicPaint.title |
| wallpaper_background.go ×2 | 玻璃不透明度 | 玻璃透明度 | glass.title |
| wallpaper_background.go | 终端不透明度 | 终端透明度 | glass.terminal |

Models：

| 文件 | before | after | 原生键 |
|---|---|---|---|
| models.go | DeepSeek 账户 | DeepSeek 账号 | provider.account |
| served_subagent_models.go | 没有可用模型。 | 没有可用的模型。 | empty.models |

Pet：

| 文件 | before | after | 原生键 |
|---|---|---|---|
| pet.go | Show Whale on **the** desktop | Show Whale on desktop | pet.enabled |
| pet.go | Scale and opacity / Scale | Size & opacity / Size | pet.scaleOpacity / pet.scale |
| pet.go | Controls the interval between self-talk and wandering. | Drives self-talk and wander intervals. | pet.activityDesc |
| pet.go | Let her talk to herself and wander around. | She talks to herself and wanders on her own. | pet.autonomyDesc |
| pet.go | Autonomous behavior / Wandering | Autonomy / Wander | pet.autonomy / pet.wander |
| pet.go | Power saving / Integration | Power save / Integrations | pet.powerSave / pet.group.link |
| pet.go ×2 | Chat and abilities / Appearance and behavior | Chat & capabilities / Desktop appearance & behavior | pet.tab.assistant / .desktop |
| pet_look.go | 看一眼模型 | 看看模型 | pet.lookModel |

快捷键页：`shortcuts.go` 的「重新读取」→「重新加载」（action.reload；Reset 按钮的 Restore default 属不同语境，保留）。

## 4. 剩余与归属

- **Usage 页不在本轮目录**：`internal/ui/settings` 没有 usage 页文件，用量表面归 `internal/ui/chat/performance_usage.go` 与用量面板（round142 已审 `chat.usage.*`）。若要覆盖设置里的 Usage 页，需要指出宿主文件。
- **保留的候选**（上下文不同或无原生对应键，未改）：`appearance.go` 壁纸「无壁纸 / None」（原生无 wallpaper-none 键，仅通用 `detail.none`「无」）；`models.go` 页标题「模型 / Models」（原生 `menu.model` 是单数菜单项）；`shortcuts.go`「恢复默认 / Restore default」（快捷键页动作，原生 `*.panel.reset` 是特效面板）；`onboarding.go`「下一步 / Next」（原生 `action.next` 是问题导航「下一题」）；`settings.go`「网页搜索 / Web search」（原生 `tool.title.webSearch` 是工具标题 "Search"）；`skills.go` 的「项 / skills」、`skill_hub.go` 的「技能 / Skill」计数/标签组合；`pet_look.go`「关闭看看 / Off」与 `vision.go`「关闭识图 / Off」（路由关闭项，原生 `effort.off` 是推理档）；`theme_font_preferences.go` 的「字体 /  font」「界面 / Interface」「默认 / Default」是行内组合后缀。
- **Account**：14 处字面量全部与原生一致或为 MyGo 自有通知/反馈文案，未改。
- **`cmd/` 内联文案**仍按 round142 结论单独列出（不属本轮）。

## 5. 回归

- `go test ./internal/ui/settings ./internal/ui/i18n ./internal/ui/frame -count=1` → settings 11.457s、i18n 0.400s、frame 12.831s，全部 ok（`desktop_behaviour_test.go` 的两处旧文案断言已同步为 "Open launcher first" / "Start desktop directly"）。
- `go vet ./internal/ui/settings ./internal/ui/i18n` → 退出 0。
- 未构建 exe、未提交；未改 cmd/、chat、boot 与安装版。

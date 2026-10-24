# Round134 · 应用菜单对齐（G37，application menu）

本轮按原生模板逐项对齐 MyGo 的应用菜单：`文件/运行/视图/帮助` 四个顶级菜单（macOS 另加应用菜单）的行、顺序、分隔符、文案与快捷键提示。所有行都绑定到既有页面动作，没有为对齐发明任何原生没有的命令。

## 1. 原生对照

模板在 [menu.js:22-121](<C:/AI/WhaleIsle-MyGo/reference/electron/src/main/menu.js>)：

- `commandItem`（`:9-19`）：一旦接受过配置 revision，菜单 keycap 只来自快捷键注册表，未绑定的命令**不显示** keycap；首个快照之前用出厂默认 `fallbackAccelerator`，避免菜单空白。命令点击先走 `dispatchMenuCommand(id)`，注册表不接管时才回落到内置 click。
- 文件：打开工作区…（`workspace.add`，回退 `CmdOrCtrl+O`）、打开启动器、在资源管理器中打开工作区、分隔符、设置…（`settings.open`，回退 `CmdOrCtrl+,`）、插件市场…（固定 `CmdOrCtrl+Shift+M`）、MCP…、技能…、分隔符、退出（macOS 为关闭窗口）。
- 运行：重启 Harness（固定 `CmdOrCtrl+Shift+R`）、重新加载界面（固定 `CmdOrCtrl+F5`；注释说明 `Ctrl+R` 已属注册表 `page.refresh`，重载改到 F5，避免一个和弦两个主人）。
- 视图：实际大小 / 放大 / 缩小 / 分隔符 / 全屏 / 开发者工具，均为 Electron role。
- 帮助：Web UI 指南、Python SDK、分隔符、关于（打开设置 about）。
- macOS 应用菜单：产品名、关于、分隔符、退出。

## 2. 逐项差异（原生 vs 本轮之前 → 本轮之后）

| 行 | 原生 | 之前 | 现在 |
|---|---|---|---|
| 打开工作区… | `workspace.add`，keycap 随 revision | 同，注册表 keycap | 保持 |
| 打开启动器 | 无 keycap | 显示了注册表 chord（Mod+Shift+L） | **无 keycap**，仍走 `launcher.open` |
| 在资源管理器中打开工作区 | 原生行（读 config.workspace 后 `shell.openPath`） | **缺失**，该位置是原生没有的「在本地打开」 | **新增** `workspace.reveal`，读 `cfgStore` 的 workspace 调 `components.OpenInFileManager`，未配置则不动；「在本地打开」退出菜单（命令与快捷键仍在） |
| 分隔符（资源管理器后） | 有 | 有 | 保持 |
| 设置… | `settings.open`，keycap 随 revision | 同 | 保持 |
| 插件市场… | 固定 `CmdOrCtrl+Shift+M` | 有，keycap 来自注册表（默认同为 Mod+Shift+M） | 保持注册表 keycap |
| MCP… / 技能… | 无 keycap | 绑定时显示 keycap | **无 keycap**，仍走 `settings.mcp`/`settings.skills` |
| 分隔符 + 退出（macOS 关闭窗口） | 有 | 有 | 保持 |
| 重启 Harness | 固定 `CmdOrCtrl+Shift+R` | 有，keycap=注册表默认 Mod+Alt+F5 | 行与动作保持；keycap 仍随注册表（见 §4 差异） |
| 重新加载界面 | `CmdOrCtrl+F5` | 文案是「重新绘制界面」 | 文案改为「重新加载界面 / Reload interface」，keycap 随注册表（默认 Mod+F5） |
| 视图六行 | 六个 role 行 | 同六行 + 多出「快捷键…」 | **删掉多出的「快捷键…」行**（命令与快捷键仍在）；开发者工具保留 role 且抑制加速键 |
| 帮助四行 | Web UI 指南 / Python SDK / 分隔符 / 关于 | 同 | 保持 |
| macOS 应用菜单 | 产品名 + 关于 + 分隔符 + 退出 | 标签写死「MyGo鲸屿」 | 标签取产品名（i18n `app.name` = Whale Isle），关于走 About 页命令 |
| Edit 菜单 | 原生没有 | 有 `RoleEditMenu` | **删除**（原生模板无此菜单） |
| keycap 来源 | 只有 commandItem 与固定 accelerator 行显示 | 任何有绑定的行都显示 | 只有原生给加速键的行显示；其余行即使用户绑定了快捷键也不显示 |
| 首次快照之前 | 显示出厂默认 | 全部空 keycap（`pageShortcutBindings` 在 status≠ready 时清空） | 显示 `shortcuts.SharedDefaults()` 的出厂默认 |

## 3. 实现

[application_menu.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/application_menu.go>)：

- 新增 `applicationMenuSpec{Bindings, HasRevision, Fallbacks, Platform, Dispatch}` 与 `keycap(id)`，逐字移植 `commandItem` 的解析规则：`HasRevision` 时只用接受过的绑定（未绑定即无 keycap），否则用出厂默认。
- `syncApplicationMenu` 用 `prefs.Get().Status == "ready"` 判定「已接受 revision」，把该状态纳入菜单 signature，配置 revision 变化时重建菜单；`Fallbacks = shortcuts.Resolve(shortcuts.SharedDefaults())`（workspace.add = Mod+O，与原生回退一致）。
- `buildApplicationMenu` 按原生模板建树；提供 `item`（带 keycap）与 `plain`（原生无加速键的行）两种构造；macOS 才加应用菜单，文件末行在 macOS 是 `window.close`、其它平台是 `app.quit`。
- 动作表新增 `workspace.reveal`，其余行沿用既有动作；`workspace.openLocal`、`shortcuts.open` 的动作保留（命令与快捷键仍可用），只是不再占菜单位。

## 4. 边界与差异（记录，不猜）

- **加速键实现方式不同**：原生把加速键装进 Electron 菜单（菜单本身是键盘主人）；MyGo 的页面快捷键已有唯一主人，菜单只用 label 尾部显示 keycap（mygo `MenuItem.SuppressAccelerator` 文档即为此模式），所以菜单不会二次抢键。副作用：keycap 文本由我们绘制，不是平台绘制。
- **重启 Harness 的 keycap**：原生写死 `Ctrl+Shift+R`；MyGo 显示实际生效的 chord（默认 Mod+Alt+F5）。若按原样显示固定值，用户按下会什么都不发生（真正的处理在注册表），因此本轮选择「keycap 永远等于会触发的和弦」，并在本文件记录该差异。
- **视图六行的 keycap**：原生是 Electron role（0 / + / - / F11 / F12）；MyGo 走自己的命令（默认 0 / = / - / Shift+Mod+I），keycap 随注册表。注册表默认值不在本轮写入范围，未改。
- **macOS 关于**：原生 `role: about` 打开系统关于面板；MyGo 没有原生关于面板，用「关于」设置页命令代替。
- **菜单栏可见性**（G03）与本轮无关，仍未在实机断言；OS 菜单本身无法在 headless 下断言。
- 未提交、未构建 exe。

## 5. 回归

- [application_menu_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/application_menu_test.go>) 重写为原生模板断言：顶级菜单与每张子菜单的 id 顺序和分隔符、macOS 应用菜单与「关闭窗口」末行、zh/en 文案前缀、keycap 规则（有 revision / 未绑定 / 首次快照前）、每行动作的 dispatch 路由、开发者工具 role、没有 Edit / 快捷键… / 在本地打开 这些原生没有的行、每行都有动作（含 `workspace.reveal`）。
- [application_commands_acceptance_windows_test.go](<C:/AI/WhaleIsle-MyGo/cmd/whaleisle/application_commands_acceptance_windows_test.go>) 同步新签名与新菜单身份：不再要求 `edit`/`workspace.openLocal`/`shortcuts.open` 行；keycap 跟随 revision 的证据改到 `settings.open`（接受 Mod+Alt+F7 后显示），并断言原生无加速键的行（`settings.mcp`、`launcher.open`）不显示 keycap。

命令（`. .\scripts\env.ps1`）：

- `go test ./cmd/whaleisle -count=1` → ok，20.5s。
- 定点：`-run "ApplicationMenu"` 五条 + `TestDeferredApplicationMenu*` 两条全 PASS（3.2s）；`TestFunctionalOwnedAppCommands` 四个子测试全 PASS（3.9s：menu-and-navigation 3.36s、preferences-persistence-conflicts 0.19s、native-input-priority 0.21s、menu-role-binding-owner 0.11s）。
- `go vet ./cmd/whaleisle` → 退出 0。

注：期间有其它成员在改 `internal/ui/chrome`、`internal/ui/frame`、`internal/ui/widgets`、`internal/ui/chat`，出现过一次编译瞬断（文件被占用），重试后全绿；这些文件不在本轮改动内。
